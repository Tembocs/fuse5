package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Server is the LSP 3.17 implementation for Fuse. Each Server
// instance owns:
//   - a Transport for stdio framing
//   - a DocStore for open documents
//   - a dispatch table keyed on method name
//
// Spec compliance at W19: only the methods declared in
// InitializeResult.Capabilities are advertised and wired. Any
// other method returns `ErrMethodNotFound`.
type Server struct {
	t        *Transport
	docs     *DocStore
	shutdown bool   // set after a `shutdown` request; `exit` notification sets done
	done     bool
	version  string
	mu       sync.Mutex
}

// New constructs a Server bound to the given transport. Version
// identifies the server for the initialize response.
func New(r io.Reader, w io.Writer, version string) *Server {
	return &Server{
		t:       NewTransport(r, w),
		docs:    NewDocStore(),
		version: version,
	}
}

// DocStore exposes the server's document store — used by tests
// and by the scripted-session harness to seed documents.
func (s *Server) DocStore() *DocStore { return s.docs }

// Run drives the read-dispatch-write loop until the client sends
// `exit` or the transport closes. The loop is single-threaded:
// a malformed message produces a JSON-RPC error response but
// does not terminate the session.
func (s *Server) Run() error {
	for {
		if s.done {
			return nil
		}
		body, err := s.t.ReadMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := s.Handle(body); err != nil {
			// Log-level error; the loop continues.
			// W19 has no logger surface yet — the error will
			// reappear as a JSON-RPC response when the handler
			// path reaches it. A truly fatal transport error
			// already returned above.
			_ = err
		}
	}
}

// Handle dispatches one raw message. Exported so tests can drive
// the server in-process without touching the transport layer.
func (s *Server) Handle(body []byte) error {
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return s.writeError(nil, ErrParse, fmt.Sprintf("parse error: %v", err))
	}
	if req.Method == "" {
		return s.writeError(req.ID, ErrInvalidRequest, "missing method")
	}
	return s.dispatch(req)
}

// dispatch routes a parsed request to its handler.
func (s *Server) dispatch(req Request) error {
	isRequest := len(req.ID) > 0 && string(req.ID) != "null"
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return nil // notification; acknowledged implicitly
	case "shutdown":
		s.shutdown = true
		return s.writeResult(req.ID, nil)
	case "exit":
		s.done = true
		return nil
	case "textDocument/didOpen":
		return s.handleDidOpen(req)
	case "textDocument/didChange":
		return s.handleDidChange(req)
	case "textDocument/didClose":
		return s.handleDidClose(req)
	case "textDocument/didSave":
		return nil // No server action required at W19.
	case "textDocument/hover":
		return s.handleHover(req)
	case "textDocument/definition":
		return s.handleDefinition(req)
	case "textDocument/completion":
		return s.handleCompletion(req)
	case "textDocument/documentSymbol":
		return s.handleDocumentSymbol(req)
	case "workspace/symbol":
		return s.handleWorkspaceSymbol(req)
	case "textDocument/semanticTokens/full":
		return s.handleSemanticTokens(req)
	case "textDocument/codeAction":
		return s.handleCodeAction(req)
	}
	if isRequest {
		return s.writeError(req.ID, ErrMethodNotFound, fmt.Sprintf("method not implemented: %s", req.Method))
	}
	return nil
}

// writeResult sends a successful JSON-RPC response. A nil result
// is allowed for requests that return void (shutdown).
func (s *Server) writeResult(id json.RawMessage, result any) error {
	return s.t.WriteResponse(Response{JSONRPC: "2.0", ID: id, Result: result})
}

// writeError sends a JSON-RPC error response.
func (s *Server) writeError(id json.RawMessage, code int, message string) error {
	return s.t.WriteResponse(Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &ResponseError{Code: code, Message: message},
	})
}

// Publish sends a textDocument/publishDiagnostics notification.
func (s *Server) Publish(uri string, diags []Diagnostic) error {
	return s.t.WriteNotification(Notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  PublishDiagnosticsParams{URI: uri, Diagnostics: diags},
	})
}

// handleInitialize returns the capability vector.
func (s *Server) handleInitialize(req Request) error {
	// We read (but don't currently use) the client capabilities.
	var params InitializeParams
	_ = json.Unmarshal(req.Params, &params)
	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:        1, // Full-text sync at W19
			HoverProvider:           true,
			DefinitionProvider:      true,
			CompletionProvider:      CompletionOptions{TriggerCharacters: []string{".", "::"}, ResolveProvider: false},
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			SemanticTokensProvider: SemanticTokensOptions{
				Legend: SemanticTokensLegend{
					TokenTypes:     semanticTokenTypes(),
					TokenModifiers: []string{"declaration", "readonly"},
				},
				Full: true,
			},
			CodeActionProvider: true,
		},
	}
	result.ServerInfo.Name = "fuse-lsp"
	result.ServerInfo.Version = s.version
	return s.writeResult(req.ID, result)
}

// handleDidOpen registers the document, then publishes diagnostics.
func (s *Server) handleDidOpen(req Request) error {
	var p DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	s.docs.Open(p.TextDocument.URI, p.TextDocument.LanguageID, p.TextDocument.Text, p.TextDocument.Version)
	return s.Publish(p.TextDocument.URI, s.computeDiagnostics(p.TextDocument.URI))
}

// handleDidChange applies the content changes then re-publishes.
func (s *Server) handleDidChange(req Request) error {
	var p DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	if s.docs.Apply(p.TextDocument, p.ContentChanges) == nil {
		return nil
	}
	return s.Publish(p.TextDocument.URI, s.computeDiagnostics(p.TextDocument.URI))
}

// handleDidClose removes the document and publishes an empty
// diagnostic list so the client clears any stale markers.
func (s *Server) handleDidClose(req Request) error {
	var p DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	s.docs.Close(p.TextDocument.URI)
	return s.Publish(p.TextDocument.URI, nil)
}

// requestParams is a helper for pulling typed params out of a
// request. Panics at decode errors are translated to InvalidParams.
func requestParams[T any](req Request, out *T) error {
	if len(req.Params) == 0 {
		return fmt.Errorf("missing params for %s", req.Method)
	}
	return json.Unmarshal(req.Params, out)
}

// uriPath strips the `file://` prefix from a URI so compiler
// spans can point at a file path.
func uriPath(uri string) string {
	const p = "file://"
	if strings.HasPrefix(uri, p) {
		return strings.TrimPrefix(uri, p)
	}
	return uri
}
