// Package lsp implements the Wave 19 Language Server Protocol
// server for Fuse. The server speaks LSP 3.17 over stdio, reuses
// the Stage-1 compiler pipeline (lex + parse + resolve + check
// + W18 diagnostics) and the W18 pass cache for tight-loop
// incremental re-evaluation.
//
// The protocol-level type layer is a focused subset: only the
// methods and capabilities the W19 exit criteria name. LSP is a
// large spec; adding a new capability means declaring its types
// here, wiring a handler, and then advertising it in the
// initialize response.
package lsp

import "encoding/json"

// Request is one JSON-RPC request. A request with a non-null ID
// expects a response; a request with a null/missing ID is a
// notification (no response). The LSP spec uses both shapes on
// the same transport.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // nil/absent => notification
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is one JSON-RPC response. Result and Error are mutually
// exclusive — either the method produced a result, or an error.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError is the JSON-RPC error envelope (spec §5.1.4).
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Notification is a JSON-RPC notification (no ID). Used when the
// server publishes diagnostics without waiting on a client reply.
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Well-known JSON-RPC error codes the LSP spec pins.
const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

// Position is a 0-based (line, character) pair, matching LSP's
// textDocument position convention. Character counts UTF-16 code
// units per the spec; at W19 we treat it as byte-count since the
// Fuse source is ASCII-heavy and the test surface does not
// exercise multi-byte characters. The full UTF-16 mapping lands
// when stdlib/full strings are typed in W22.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is an inclusive-start, exclusive-end byte range.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location is a file URI plus a Range — the navigation payload.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a document by URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier is TextDocumentIdentifier plus a
// monotonic version stamp the client increments on every edit.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentItem is the full payload of a document — what
// didOpen sends.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentContentChangeEvent describes one change from didChange.
// When Range is nil the change is a full-text replacement.
type TextDocumentContentChangeEvent struct {
	Range *Range `json:"range,omitempty"`
	Text  string `json:"text"`
}

// TextDocumentPositionParams is the common shape for hover / goto /
// completion.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// InitializeParams is what the client sends on the initial handshake.
// W19 only reads the client info; full ClientCapabilities parsing
// is future work.
type InitializeParams struct {
	ProcessID int    `json:"processId,omitempty"`
	RootURI   string `json:"rootUri,omitempty"`
	ClientInfo struct {
		Name    string `json:"name"`
		Version string `json:"version,omitempty"`
	} `json:"clientInfo,omitempty"`
}

// InitializeResult advertises the server's capabilities. W19
// advertises only the methods this wave implements.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// ServerCapabilities is the capability vector. Each field here must
// correspond to a wired handler.
type ServerCapabilities struct {
	TextDocumentSync      int                    `json:"textDocumentSync"` // 1 = full-sync, 2 = incremental
	HoverProvider         bool                   `json:"hoverProvider"`
	DefinitionProvider    bool                   `json:"definitionProvider"`
	CompletionProvider    CompletionOptions      `json:"completionProvider"`
	DocumentSymbolProvider bool                  `json:"documentSymbolProvider"`
	WorkspaceSymbolProvider bool                 `json:"workspaceSymbolProvider"`
	SemanticTokensProvider SemanticTokensOptions `json:"semanticTokensProvider"`
	CodeActionProvider    bool                   `json:"codeActionProvider"`
}

// CompletionOptions carries trigger-character config.
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider"`
}

// SemanticTokensOptions advertises the token legend.
type SemanticTokensOptions struct {
	Legend SemanticTokensLegend `json:"legend"`
	Full   bool                 `json:"full"`
}

// SemanticTokensLegend declares the token types and modifiers the
// server emits. The client maps these to theme colours.
type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

// PublishDiagnosticsParams is what the server sends to publish
// diagnostics for a file.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Diagnostic is the LSP-side diagnostic shape. Mirrors
// diagnostics.Rendered but with LSP's Range convention.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"` // 1=Error 2=Warning 3=Information 4=Hint
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// DiagnosticSeverity constants.
const (
	SeverityError       = 1
	SeverityWarning     = 2
	SeverityInformation = 3
	SeverityHint        = 4
)

// HoverParams is the hover-request params shape.
type HoverParams struct{ TextDocumentPositionParams }

// Hover is the hover-response shape.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent is a markdown-formatted string.
type MarkupContent struct {
	Kind  string `json:"kind"`  // "plaintext" or "markdown"
	Value string `json:"value"`
}

// CompletionParams is the completion-request shape.
type CompletionParams struct{ TextDocumentPositionParams }

// CompletionItem is one completion entry.
type CompletionItem struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind,omitempty"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
}

// CompletionList is the completion-response shape.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// DocumentSymbol is one entry in the documentSymbol tree.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// WorkspaceSymbolParams asks for matching symbols across the workspace.
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// SymbolInformation is a flat workspace-symbol entry.
type SymbolInformation struct {
	Name     string   `json:"name"`
	Kind     int      `json:"kind"`
	Location Location `json:"location"`
}

// SymbolKind constants matching the LSP spec.
const (
	SymbolFile        = 1
	SymbolModule      = 2
	SymbolNamespace   = 3
	SymbolClass       = 5
	SymbolMethod      = 6
	SymbolProperty    = 7
	SymbolField       = 8
	SymbolConstructor = 9
	SymbolEnum        = 10
	SymbolInterface   = 11
	SymbolFunction    = 12
	SymbolVariable    = 13
	SymbolConstant    = 14
	SymbolString      = 15
	SymbolStruct      = 23
)

// CompletionItemKind constants matching the LSP spec.
const (
	CompletionText         = 1
	CompletionMethod       = 2
	CompletionFunction     = 3
	CompletionConstructor  = 4
	CompletionField        = 5
	CompletionVariable     = 6
	CompletionClass        = 7
	CompletionInterface    = 8
	CompletionKeyword      = 14
	CompletionSnippet      = 15
	CompletionStruct       = 22
	CompletionTypeParameter = 25
)

// SemanticTokens is the semanticTokens/full response shape. Data
// is a packed [deltaLine, deltaStart, length, tokenType,
// tokenModifierBitmask] × N array; see LSP spec §3.17/textDocument/semanticTokens.
type SemanticTokens struct {
	Data []int `json:"data"`
}

// CodeActionParams is the code-action request shape.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext carries the diagnostics relevant to the action.
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// CodeAction is a quick-fix offering.
type CodeAction struct {
	Title       string       `json:"title"`
	Kind        string       `json:"kind,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
}

// WorkspaceEdit is the edit payload of a quick fix.
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

// TextEdit describes one text replacement.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// DidOpenTextDocumentParams is the didOpen notification shape.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams is the didChange notification shape.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier   `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams is the didClose notification shape.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams is the didSave notification shape.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

// DocumentSymbolParams is the documentSymbol request shape.
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SemanticTokensParams is the semanticTokens/full request shape.
type SemanticTokensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}
