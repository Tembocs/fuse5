package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/lex"
)

// -- Transport --------------------------------------------------

// TestLspTransport exercises the JSON-RPC framing: the transport
// reads Content-Length-framed messages and writes them back with
// the correct envelope.
func TestLspTransport(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","method":"ping"}`
		in := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
		tr := NewTransport(strings.NewReader(in), &bytes.Buffer{})
		got, err := tr.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		if string(got) != body {
			t.Errorf("body = %q, want %q", got, body)
		}
	})
	t.Run("write-frames-correctly", func(t *testing.T) {
		var out bytes.Buffer
		tr := NewTransport(strings.NewReader(""), &out)
		body := []byte(`{"jsonrpc":"2.0"}`)
		if err := tr.WriteMessage(body); err != nil {
			t.Fatalf("WriteMessage: %v", err)
		}
		want := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
		if out.String() != want {
			t.Errorf("frame = %q, want %q", out.String(), want)
		}
	})
	t.Run("malformed-returns-error", func(t *testing.T) {
		// Missing Content-Length header.
		tr := NewTransport(strings.NewReader("Bogus: 1\r\n\r\n{}"), &bytes.Buffer{})
		if _, err := tr.ReadMessage(); err == nil {
			t.Fatal("malformed frame should error")
		}
	})
	t.Run("eof-propagates", func(t *testing.T) {
		tr := NewTransport(strings.NewReader(""), &bytes.Buffer{})
		if _, err := tr.ReadMessage(); err != io.EOF {
			t.Errorf("EOF read = %v, want io.EOF", err)
		}
	})
}

// -- Initialize -------------------------------------------------

// TestLspInitialize verifies the server's capability advertisement
// matches what the W19 scope implements — nothing more, nothing
// less.
func TestLspInitialize(t *testing.T) {
	s, client := newTestServer(t)
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":%s}`,
		`{"processId":1,"rootUri":"file:///ws","clientInfo":{"name":"test"}}`)
	if err := s.Handle([]byte(req)); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	resp := client.firstResponse(t)
	if resp.Error != nil {
		t.Fatalf("initialize error: %+v", resp.Error)
	}
	resultBytes, _ := json.Marshal(resp.Result)
	var result InitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	caps := result.Capabilities
	if !caps.HoverProvider || !caps.DefinitionProvider ||
		!caps.DocumentSymbolProvider || !caps.WorkspaceSymbolProvider ||
		!caps.CodeActionProvider || !caps.SemanticTokensProvider.Full {
		t.Errorf("missing capability: %+v", caps)
	}
	if caps.TextDocumentSync != 1 {
		t.Errorf("textDocumentSync = %d, want 1 (full)", caps.TextDocumentSync)
	}
	if result.ServerInfo.Name != "fuse-lsp" {
		t.Errorf("serverInfo.name = %q, want fuse-lsp", result.ServerInfo.Name)
	}
}

// -- Doc sync ---------------------------------------------------

// TestLspDocSync verifies didOpen / didChange / didClose apply to
// the document store.
func TestLspDocSync(t *testing.T) {
	s, _ := newTestServer(t)
	uri := "file:///ws/a.fuse"

	// didOpen.
	must(t, s.Handle(didOpenMsg(uri, "fn main() -> I32 { return 0; }", 1)))
	got := s.DocStore().Get(uri)
	if got == nil {
		t.Fatalf("didOpen did not record the document")
	}
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}

	// didChange — full-text replacement.
	must(t, s.Handle(didChangeMsg(uri, 2, nil, "fn main() -> I32 { return 42; }")))
	if s.DocStore().Get(uri).Version != 2 {
		t.Errorf("didChange did not bump version")
	}
	if !strings.Contains(s.DocStore().Get(uri).Text, "42") {
		t.Errorf("didChange did not update text: %q", s.DocStore().Get(uri).Text)
	}

	// didClose.
	must(t, s.Handle(didCloseMsg(uri)))
	if s.DocStore().Get(uri) != nil {
		t.Errorf("didClose did not remove the document")
	}
}

// -- Diagnostics stream ----------------------------------------

// TestLspDiagnosticsStream verifies that didOpen on a malformed
// source publishes a diagnostic via the notification channel.
func TestLspDiagnosticsStream(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/bad.fuse"

	// A source the lexer / parser rejects — missing closing brace
	// is a shape the parser reliably complains about.
	src := `fn main() -> I32 { return 0; `
	must(t, s.Handle(didOpenMsg(uri, src, 1)))

	notes := client.notifications("textDocument/publishDiagnostics")
	if len(notes) == 0 {
		t.Fatalf("no publishDiagnostics notification fired")
	}
	// The last notification is the most recent one.
	var params PublishDiagnosticsParams
	raw, _ := json.Marshal(notes[len(notes)-1].Params)
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("decode params: %v", err)
	}
	if params.URI != uri {
		t.Errorf("params.uri = %q, want %q", params.URI, uri)
	}
	if len(params.Diagnostics) == 0 {
		t.Errorf("expected ≥1 diagnostic for malformed source")
	}
}

// TestLspDiagnosticsSemantic pins the 2026-04-18 audit fix
// promoting computeDiagnostics from parse-only to
// parse+resolve+bridge+check. A source that parses cleanly but
// fails type-checking (return of `Unit` where `I32` is declared)
// must now produce a diagnostic whose Message carries the
// checker's hint inline as ` hint: <text>`.
func TestLspDiagnosticsSemantic(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/semantic.fuse"

	// Parses cleanly; the checker complains because the return
	// expression is Bool but the fn declares I32.
	src := `fn main() -> I32 { let x: Bool = true; return x; }`
	must(t, s.Handle(didOpenMsg(uri, src, 1)))

	notes := client.notifications("textDocument/publishDiagnostics")
	if len(notes) == 0 {
		t.Fatalf("no publishDiagnostics notification fired")
	}
	var params PublishDiagnosticsParams
	raw, _ := json.Marshal(notes[len(notes)-1].Params)
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatalf("decode params: %v", err)
	}
	if len(params.Diagnostics) == 0 {
		t.Fatalf("expected ≥1 check diagnostic for return-type mismatch")
	}
	sawHint := false
	for _, d := range params.Diagnostics {
		if strings.Contains(d.Message, " hint: ") {
			sawHint = true
			break
		}
	}
	if !sawHint {
		t.Errorf("no diagnostic carried an inline `hint:` payload; messages=%+v",
			params.Diagnostics)
	}
}

// -- Quick fixes -----------------------------------------------

// TestLspQuickFixes verifies that a diagnostic with a trailing
// `hint:` produces a CodeAction offering the suggestion.
func TestLspQuickFixes(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/q.fuse"
	must(t, s.Handle(didOpenMsg(uri, "fn main() -> I32 { return 0; }", 1)))

	// Send a textDocument/codeAction request with a synthetic
	// diagnostic that carries `hint:` content.
	req := fmt.Sprintf(`{
		"jsonrpc":"2.0","id":11,"method":"textDocument/codeAction","params":{
			"textDocument":{"uri":%q},
			"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},
			"context":{"diagnostics":[{
				"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":5}},
				"severity":1,"message":"expected return hint: return value"
			}]}
		}
	}`, uri)
	must(t, s.Handle([]byte(req)))
	resp := client.lastResponse()
	if resp.Error != nil {
		t.Fatalf("codeAction error: %+v", resp.Error)
	}
	raw, _ := json.Marshal(resp.Result)
	var actions []CodeAction
	if err := json.Unmarshal(raw, &actions); err != nil {
		t.Fatalf("decode actions: %v", err)
	}
	if len(actions) == 0 {
		t.Fatalf("no quick-fix offered for diagnostic with hint")
	}
	if !strings.Contains(actions[0].Title, "return value") {
		t.Errorf("action title missing suggestion text: %q", actions[0].Title)
	}
}

// -- Hover ------------------------------------------------------

// TestLspHover checks the hover resolver returns type / doc info
// for items and locals.
func TestLspHover(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/h.fuse"
	src := "/// Returns 42 always.\npub fn answer() -> I32 {\n    let x: I32 = 42;\n    return x;\n}\n"
	must(t, s.Handle(didOpenMsg(uri, src, 1)))

	// Hover on `answer` (line 1, col 7).
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"method":"textDocument/hover","params":{"textDocument":{"uri":%q},"position":{"line":1,"character":7}}}`, uri)
	must(t, s.Handle([]byte(req)))
	resp := client.lastResponse()
	raw, _ := json.Marshal(resp.Result)
	var hv Hover
	if err := json.Unmarshal(raw, &hv); err != nil {
		t.Fatalf("decode hover: %v", err)
	}
	if !strings.Contains(hv.Contents.Value, "answer") {
		t.Errorf("hover missing identifier: %q", hv.Contents.Value)
	}
	if !strings.Contains(hv.Contents.Value, "Returns 42 always") {
		t.Errorf("hover missing doc comment: %q", hv.Contents.Value)
	}
	if !strings.Contains(hv.Contents.Value, "fn") {
		t.Errorf("hover missing kind: %q", hv.Contents.Value)
	}

	// Hover on `x` — should resolve as a local with type I32.
	req = fmt.Sprintf(`{"jsonrpc":"2.0","id":4,"method":"textDocument/hover","params":{"textDocument":{"uri":%q},"position":{"line":2,"character":9}}}`, uri)
	must(t, s.Handle([]byte(req)))
	resp = client.lastResponse()
	raw, _ = json.Marshal(resp.Result)
	hv = Hover{}
	_ = json.Unmarshal(raw, &hv)
	if !strings.Contains(hv.Contents.Value, "I32") {
		t.Errorf("hover on local x missing I32: %q", hv.Contents.Value)
	}
}

// -- Goto definition -------------------------------------------

func TestLspGotoDefinition(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/g.fuse"
	src := "fn helper() -> I32 {\n    return 7;\n}\n\nfn main() -> I32 {\n    return helper();\n}\n"
	must(t, s.Handle(didOpenMsg(uri, src, 1)))

	// Goto on `helper` inside main (line 5, col 11).
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":5,"method":"textDocument/definition","params":{"textDocument":{"uri":%q},"position":{"line":5,"character":11}}}`, uri)
	must(t, s.Handle([]byte(req)))
	resp := client.lastResponse()
	raw, _ := json.Marshal(resp.Result)
	var locs []Location
	if err := json.Unmarshal(raw, &locs); err != nil {
		t.Fatalf("decode locs: %v", err)
	}
	if len(locs) == 0 {
		t.Fatalf("no definition found for helper")
	}
	// Helper is declared on line 0.
	if locs[0].Range.Start.Line != 0 {
		t.Errorf("definition line = %d, want 0", locs[0].Range.Start.Line)
	}
	if locs[0].URI != uri {
		t.Errorf("definition uri = %q, want %q", locs[0].URI, uri)
	}
}

// -- Document symbols ------------------------------------------

func TestLspDocumentSymbols(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/s.fuse"
	src := "fn a() {}\n\nstruct Pair { x: I32, y: I32 }\n\nenum Dir { North, South }\n\npub const MAX: I32 = 100;\n"
	must(t, s.Handle(didOpenMsg(uri, src, 1)))

	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":6,"method":"textDocument/documentSymbol","params":{"textDocument":{"uri":%q}}}`, uri)
	must(t, s.Handle([]byte(req)))
	resp := client.lastResponse()
	raw, _ := json.Marshal(resp.Result)
	var syms []DocumentSymbol
	if err := json.Unmarshal(raw, &syms); err != nil {
		t.Fatalf("decode syms: %v", err)
	}
	want := map[string]int{
		"a":    SymbolFunction,
		"Pair": SymbolStruct,
		"Dir":  SymbolEnum,
		"MAX":  SymbolConstant,
	}
	got := map[string]int{}
	for _, s := range syms {
		got[s.Name] = s.Kind
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Errorf("symbol %s kind = %d, want %d", name, got[name], kind)
		}
	}

	// Workspace symbol with a query: only `Pair` and items
	// containing 'p' substring.
	req = `{"jsonrpc":"2.0","id":7,"method":"workspace/symbol","params":{"query":"Pai"}}`
	must(t, s.Handle([]byte(req)))
	resp = client.lastResponse()
	raw, _ = json.Marshal(resp.Result)
	var wsSyms []SymbolInformation
	_ = json.Unmarshal(raw, &wsSyms)
	if len(wsSyms) != 1 || wsSyms[0].Name != "Pair" {
		t.Errorf("workspace/symbol filter failed: %+v", wsSyms)
	}
}

// -- Completion -------------------------------------------------

func TestLspCompletion(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/c.fuse"
	src := "fn helper() -> I32 { return 0; }\n\nfn main() -> I32 {\n    \n}\n"
	must(t, s.Handle(didOpenMsg(uri, src, 1)))

	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":8,"method":"textDocument/completion","params":{"textDocument":{"uri":%q},"position":{"line":3,"character":4}}}`, uri)
	must(t, s.Handle([]byte(req)))
	resp := client.lastResponse()
	raw, _ := json.Marshal(resp.Result)
	var list CompletionList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("decode completion: %v", err)
	}
	hasReturn := false
	hasHelper := false
	hasLet := false
	for _, it := range list.Items {
		switch it.Label {
		case "return":
			hasReturn = true
		case "helper":
			hasHelper = true
		case "let":
			hasLet = true
		}
	}
	if !hasReturn || !hasLet {
		t.Errorf("keyword completions missing: %+v", list.Items[:min(5, len(list.Items))])
	}
	if !hasHelper {
		t.Errorf("declared-item completion missing for helper")
	}
}

// -- Semantic tokens -------------------------------------------

func TestLspSemanticTokens(t *testing.T) {
	s, client := newTestServer(t)
	uri := "file:///ws/st.fuse"
	src := "fn main() -> I32 {\n    let x: I32 = add(1, 2);\n    return x;\n}\n"
	must(t, s.Handle(didOpenMsg(uri, src, 1)))

	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":9,"method":"textDocument/semanticTokens/full","params":{"textDocument":{"uri":%q}}}`, uri)
	must(t, s.Handle([]byte(req)))
	resp := client.lastResponse()
	raw, _ := json.Marshal(resp.Result)
	var tok SemanticTokens
	if err := json.Unmarshal(raw, &tok); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}
	// Stream must be a multiple of 5 (deltaLine,deltaStart,len,type,mods).
	if len(tok.Data)%5 != 0 {
		t.Fatalf("packed data not a multiple of 5: %v", tok.Data)
	}
	if len(tok.Data) == 0 {
		t.Fatalf("no tokens produced for non-trivial source")
	}
	// Look for at least one tokKeyword (fn / let / return) and
	// one tokType (I32, a CamelCase-ish identifier).
	sawKeyword, sawType := false, false
	for i := 0; i < len(tok.Data); i += 5 {
		switch tok.Data[i+3] {
		case tokKeyword:
			sawKeyword = true
		case tokType:
			sawType = true
		}
	}
	if !sawKeyword {
		t.Errorf("no keyword token in stream")
	}
	if !sawType {
		t.Errorf("no type token in stream")
	}
}

// -- Test harness ----------------------------------------------

// testClient captures everything the server writes. Responses and
// notifications are split so tests can probe each channel
// independently.
type testClient struct {
	out bytes.Buffer
	t   *testing.T
}

func (c *testClient) Write(p []byte) (int, error) { return c.out.Write(p) }

// newTestServer constructs an in-process server wired to a
// testClient. The client captures every framed message.
func newTestServer(t *testing.T) (*Server, *testClient) {
	t.Helper()
	client := &testClient{t: t}
	s := New(strings.NewReader(""), client, "test")
	return s, client
}

// framed returns the list of LSP-framed messages the server has
// written so far. Each message is the raw JSON body.
func (c *testClient) framed() [][]byte {
	raw := c.out.Bytes()
	var out [][]byte
	for len(raw) > 0 {
		idx := bytes.Index(raw, []byte("\r\n\r\n"))
		if idx < 0 {
			break
		}
		header := string(raw[:idx])
		body := raw[idx+4:]
		var length int
		for _, line := range strings.Split(header, "\r\n") {
			if strings.HasPrefix(line, "Content-Length:") {
				fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")), "%d", &length)
			}
		}
		if length <= 0 || length > len(body) {
			break
		}
		out = append(out, append([]byte(nil), body[:length]...))
		raw = body[length:]
	}
	return out
}

// parsed returns every framed message decoded as a Response (ID
// set) or stored as a raw notification envelope.
func (c *testClient) messages() []map[string]any {
	var out []map[string]any
	for _, b := range c.framed() {
		var m map[string]any
		if err := json.Unmarshal(b, &m); err == nil {
			out = append(out, m)
		}
	}
	return out
}

// firstResponse finds the first framed message that has an "id"
// key — that is, a JSON-RPC response.
func (c *testClient) firstResponse(t *testing.T) Response {
	t.Helper()
	for _, b := range c.framed() {
		var r Response
		_ = json.Unmarshal(b, &r)
		if len(r.ID) > 0 {
			return r
		}
	}
	t.Fatalf("no response among: %v", c.messages())
	return Response{}
}

// lastResponse returns the most recent response. Tests that
// issue multiple requests call this after each to inspect the
// corresponding response.
func (c *testClient) lastResponse() Response {
	frames := c.framed()
	for i := len(frames) - 1; i >= 0; i-- {
		var r Response
		if err := json.Unmarshal(frames[i], &r); err == nil && len(r.ID) > 0 {
			return r
		}
	}
	return Response{}
}

// notifications returns every framed notification whose method
// equals the given name.
func (c *testClient) notifications(method string) []Notification {
	var out []Notification
	for _, b := range c.framed() {
		var n Notification
		if err := json.Unmarshal(b, &n); err == nil && n.Method == method {
			out = append(out, n)
		}
	}
	return out
}

// didOpenMsg / didChangeMsg / didCloseMsg are raw JSON messages
// suitable for s.Handle.
func didOpenMsg(uri, text string, version int) []byte {
	p := DidOpenTextDocumentParams{TextDocument: TextDocumentItem{
		URI: uri, LanguageID: "fuse", Version: version, Text: text,
	}}
	body, _ := json.Marshal(p)
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":%s}`, body))
}

func didChangeMsg(uri string, version int, r *Range, text string) []byte {
	p := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{URI: uri, Version: version},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Range: r, Text: text},
		},
	}
	body, _ := json.Marshal(p)
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didChange","params":%s}`, body))
}

func didCloseMsg(uri string) []byte {
	p := DidCloseTextDocumentParams{TextDocument: TextDocumentIdentifier{URI: uri}}
	body, _ := json.Marshal(p)
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didClose","params":%s}`, body))
}

// TestLspUtf16Positions pins the W24 UTF-16 position contract:
// Position.Character is counted in UTF-16 code units, not bytes.
// The corpus mixes ASCII, a multi-byte BMP rune (⚙ U+2699, 3 UTF-8
// bytes → 1 UTF-16 unit), and a supplementary-plane rune
// (🚀 U+1F680, 4 UTF-8 bytes → 2 UTF-16 units) so regressions in
// either direction fail fast.
func TestLspUtf16Positions(t *testing.T) {
	// Line 0: "let a = 1;"               (ASCII only)
	// Line 1: "// ⚙ three-byte BMP"     ('⚙' is 3 UTF-8 bytes, 1 UTF-16 unit)
	// Line 2: "// 🚀 four-byte astral"  ('🚀' is 4 UTF-8 bytes, 2 UTF-16 units)
	text := "let a = 1;\n// \u2699 three-byte BMP\n// \U0001F680 four-byte astral\n"

	t.Run("byteOffsetToPosition-uses-utf16-units", func(t *testing.T) {
		// The '⚙' starts at line 1 after the "// " prefix (3 bytes),
		// so byte offset = len("let a = 1;\n") + 3 = 11 + 3 = 14.
		// UTF-16 column for byte just AFTER '⚙' should be 3 (// ) + 1 (⚙) = 4.
		gearByte := strings.Index(text, "\u2699")
		got := byteOffsetToPosition(text, gearByte+len("\u2699"))
		want := Position{Line: 1, Character: 4}
		if got != want {
			t.Errorf("post-gear position = %+v, want %+v", got, want)
		}

		// The '🚀' is supplementary-plane → 2 UTF-16 code units.
		rocketByte := strings.Index(text, "\U0001F680")
		got = byteOffsetToPosition(text, rocketByte+len("\U0001F680"))
		want = Position{Line: 2, Character: 5}
		if got != want {
			t.Errorf("post-rocket position = %+v, want %+v", got, want)
		}
	})

	t.Run("positionToByteOffset-roundtrips", func(t *testing.T) {
		// Round-trip every meaningful offset against byteOffsetToPosition.
		// This pins the inverse.
		for _, offset := range []int{0, 5, 11, 14, 17, 30, len(text) - 1, len(text)} {
			pos := byteOffsetToPosition(text, offset)
			if got := positionToByteOffset(text, pos); got != offset {
				// Offsets that land inside a multi-byte sequence can't
				// round-trip — byteOffsetToPosition snaps to the next
				// rune boundary. Accept an equivalent offset that
				// points at the same logical position.
				if byteOffsetToPosition(text, got) != pos {
					t.Errorf("positionToByteOffset(%+v) = %d, want %d", pos, got, offset)
				}
			}
		}
	})

	t.Run("diagnostic-translation-transcodes", func(t *testing.T) {
		// A diagnostic whose byte span falls after the '🚀' should
		// surface the LSP character as UTF-16 units, not UTF-8 bytes.
		rocketByte := strings.Index(text, "\U0001F680")
		afterRocketByte := rocketByte + len("\U0001F680")
		diags := []lex.Diagnostic{{
			Span: lex.Span{
				File:  "utf16.fuse",
				Start: lex.Position{Offset: afterRocketByte, Line: 3, Column: 8},
				End:   lex.Position{Offset: afterRocketByte + 1, Line: 3, Column: 9},
			},
			Message: "probe",
		}}
		out := translateDiagnostics(text, diags)
		if len(out) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(out))
		}
		// Byte column at afterRocketByte on line 2: "// " (3 bytes)
		//   + "🚀" (4 bytes) = 7. UTF-16 column: 3 + 2 = 5.
		if got := out[0].Range.Start.Character; got != 5 {
			t.Errorf("LSP start.character = %d, want 5 (UTF-16 units, not 7 bytes)", got)
		}
	})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("server handle failed: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
