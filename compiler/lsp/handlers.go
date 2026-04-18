package lsp

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Tembocs/fuse5/compiler/doc"
)

// identifierPattern matches the canonical Fuse identifier shape
// `[A-Za-z_][A-Za-z_0-9]*`. Used by hover / goto / completion to
// locate the token under the cursor without invoking the real
// lexer.
var identifierPattern = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*`)

// handleHover returns a markdown blob naming the identifier under
// the cursor and the type-or-doc information the server can
// resolve for it at W19.
func (s *Server) handleHover(req Request) error {
	var p HoverParams
	if err := requestParams(req, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	doc := s.docs.Get(p.TextDocument.URI)
	if doc == nil {
		return s.writeResult(req.ID, nil)
	}
	word, wordRange, ok := identifierAt(doc.Text, p.Position)
	if !ok {
		return s.writeResult(req.ID, nil)
	}
	info := resolveHoverInfo(doc.Text, word)
	return s.writeResult(req.ID, Hover{
		Contents: MarkupContent{Kind: "markdown", Value: info},
		Range:    &wordRange,
	})
}

// handleDefinition returns the location of the declaration for
// the identifier under the cursor. W19 resolves: top-level items
// (fn / struct / enum / trait / const / static) declared in the
// same document; local bindings introduced by `let`; and
// parameter names.
func (s *Server) handleDefinition(req Request) error {
	var p TextDocumentPositionParams
	if err := requestParams(req, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	docu := s.docs.Get(p.TextDocument.URI)
	if docu == nil {
		return s.writeResult(req.ID, nil)
	}
	word, _, ok := identifierAt(docu.Text, p.Position)
	if !ok {
		return s.writeResult(req.ID, nil)
	}
	loc, ok := findDefinition(p.TextDocument.URI, docu.Text, word)
	if !ok {
		return s.writeResult(req.ID, nil)
	}
	return s.writeResult(req.ID, []Location{loc})
}

// handleCompletion returns identifiers in scope at the cursor
// plus the Fuse keyword set. Identifiers are sorted so the
// response is deterministic.
func (s *Server) handleCompletion(req Request) error {
	var p CompletionParams
	if err := requestParams(req, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	docu := s.docs.Get(p.TextDocument.URI)
	if docu == nil {
		return s.writeResult(req.ID, CompletionList{Items: nil})
	}
	items := completionsFor(docu.Text)
	return s.writeResult(req.ID, CompletionList{IsIncomplete: false, Items: items})
}

// handleDocumentSymbol returns a hierarchical symbol list for the
// requested document.
func (s *Server) handleDocumentSymbol(req Request) error {
	var p DocumentSymbolParams
	if err := requestParams(req, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	docu := s.docs.Get(p.TextDocument.URI)
	if docu == nil {
		return s.writeResult(req.ID, []DocumentSymbol{})
	}
	return s.writeResult(req.ID, extractDocumentSymbols(docu.Text))
}

// handleWorkspaceSymbol filters documentSymbol across every open
// document by the request's query substring.
func (s *Server) handleWorkspaceSymbol(req Request) error {
	var p WorkspaceSymbolParams
	if err := requestParams(req, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	var out []SymbolInformation
	query := strings.ToLower(p.Query)
	for _, uri := range s.docs.URIs() {
		docu := s.docs.Get(uri)
		if docu == nil {
			continue
		}
		for _, sym := range extractDocumentSymbols(docu.Text) {
			if query == "" || strings.Contains(strings.ToLower(sym.Name), query) {
				out = append(out, SymbolInformation{
					Name:     sym.Name,
					Kind:     sym.Kind,
					Location: Location{URI: uri, Range: sym.Range},
				})
			}
		}
	}
	return s.writeResult(req.ID, out)
}

// handleSemanticTokens emits a packed token stream for syntax
// highlighting. W19 produces tokens for identifiers that the
// regex-based classifier can resolve — keywords, types (heuristic:
// CamelCase identifiers), functions (identifier followed by `(`),
// and plain variables (everything else).
func (s *Server) handleSemanticTokens(req Request) error {
	var p SemanticTokensParams
	if err := requestParams(req, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	docu := s.docs.Get(p.TextDocument.URI)
	if docu == nil {
		return s.writeResult(req.ID, SemanticTokens{Data: []int{}})
	}
	return s.writeResult(req.ID, SemanticTokens{Data: computeSemanticTokens(docu.Text)})
}

// handleCodeAction offers quick-fixes for diagnostics that have
// a structured suggestion. W19 surfaces any Diagnostic whose
// message includes ` hint:` as a CodeAction with the suggestion
// as the replacement text at the diagnostic's range.
func (s *Server) handleCodeAction(req Request) error {
	var p CodeActionParams
	if err := requestParams(req, &p); err != nil {
		return s.writeError(req.ID, ErrInvalidParams, err.Error())
	}
	var actions []CodeAction
	for _, d := range p.Context.Diagnostics {
		if idx := strings.Index(d.Message, "hint:"); idx >= 0 {
			suggestion := strings.TrimSpace(d.Message[idx+len("hint:"):])
			if suggestion == "" {
				continue
			}
			actions = append(actions, CodeAction{
				Title:       "apply suggestion: " + suggestion,
				Kind:        "quickfix",
				Diagnostics: []Diagnostic{d},
				Edit: &WorkspaceEdit{
					Changes: map[string][]TextEdit{
						p.TextDocument.URI: {{Range: d.Range, NewText: suggestion}},
					},
				},
			})
		}
	}
	return s.writeResult(req.ID, actions)
}

// identifierAt returns the identifier under (or immediately to the
// left of) the given position. The range is returned so callers
// can replace-in-place for quick fixes.
func identifierAt(text string, p Position) (string, Range, bool) {
	offset := positionToByteOffset(text, p)
	if offset < 0 {
		return "", Range{}, false
	}
	// Expand left while we see identifier characters.
	start := offset
	for start > 0 && isIdentByte(text[start-1]) {
		start--
	}
	end := offset
	for end < len(text) && isIdentByte(text[end]) {
		end++
	}
	if start == end {
		return "", Range{}, false
	}
	word := text[start:end]
	r := Range{
		Start: byteOffsetToPosition(text, start),
		End:   byteOffsetToPosition(text, end),
	}
	return word, r, true
}

func isIdentByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// resolveHoverInfo is the source for the markdown blob shown on
// hover. W19 uses a deliberately-light heuristic: report the
// identifier's kind (fn / struct / enum / const / static /
// trait / let) as detected by line-based scanning, plus any
// `///` doc comment preceding the declaration.
func resolveHoverInfo(text, word string) string {
	for _, it := range doc.Extract([]byte(text)) {
		if it.Name == word {
			var sb strings.Builder
			sb.WriteString("**")
			sb.WriteString(it.Kind)
			sb.WriteString("** `")
			sb.WriteString(word)
			sb.WriteString("`")
			if it.Doc != "" {
				sb.WriteString("\n\n")
				sb.WriteString(it.Doc)
			}
			return sb.String()
		}
	}
	// Fallback: look for a `let <word>` binding.
	letRe := regexp.MustCompile(`(?m)let\s+` + regexp.QuoteMeta(word) + `\b\s*:\s*([A-Za-z_][A-Za-z_0-9\[\] ,<>]*)`)
	if m := letRe.FindStringSubmatch(text); m != nil {
		return "**local** `" + word + "`: `" + strings.TrimSpace(m[1]) + "`"
	}
	// Fallback: parameter with a type annotation.
	paramRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\s*:\s*([A-Za-z_][A-Za-z_0-9\[\] ,<>]*)`)
	if m := paramRe.FindStringSubmatch(text); m != nil {
		return "**parameter** `" + word + "`: `" + strings.TrimSpace(m[1]) + "`"
	}
	return "`" + word + "`"
}

// findDefinition locates the declaration of `name` in text and
// returns a Location pointing at the declaration line.
func findDefinition(uri, text, name string) (Location, bool) {
	items := doc.Extract([]byte(text))
	for _, it := range items {
		if it.Name == name {
			line := it.Line - 1 // LSP is 0-indexed
			return Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: line, Character: 0},
					End:   Position{Line: line, Character: 0},
				},
			}, true
		}
	}
	// `let <name>` binding.
	letRe := regexp.MustCompile(`(?m)let\s+` + regexp.QuoteMeta(name) + `\b`)
	if idx := letRe.FindStringIndex(text); idx != nil {
		pos := byteOffsetToPosition(text, idx[0])
		return Location{URI: uri, Range: Range{Start: pos, End: pos}}, true
	}
	return Location{}, false
}

// completionsFor returns identifiers in scope plus the keyword
// set. Deduped and sorted deterministically.
func completionsFor(text string) []CompletionItem {
	seen := map[string]int{}
	items := []CompletionItem{}

	// Keywords are always offered.
	for _, kw := range fuseKeywords {
		if _, ok := seen[kw]; !ok {
			seen[kw] = len(items)
			items = append(items, CompletionItem{Label: kw, Kind: CompletionKeyword, Detail: "keyword"})
		}
	}
	// Items declared in the document.
	for _, it := range doc.Extract([]byte(text)) {
		if _, ok := seen[it.Name]; ok {
			continue
		}
		seen[it.Name] = len(items)
		items = append(items, CompletionItem{
			Label:  it.Name,
			Kind:   symbolKindToCompletion(it.Kind),
			Detail: it.Kind,
		})
	}
	// Identifiers that appear in the source (shallow).
	for _, match := range identifierPattern.FindAllString(text, -1) {
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = len(items)
		items = append(items, CompletionItem{Label: match, Kind: CompletionVariable})
	}
	// Deterministic order by label.
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

// extractDocumentSymbols builds a flat-but-structured symbol list
// from the document. Nested items (methods inside impl blocks) are
// approximated via the impl pattern but kept flat at W19 — the
// hierarchical shape lands when the resolver-backed path reaches
// LSP.
func extractDocumentSymbols(text string) []DocumentSymbol {
	items := doc.Extract([]byte(text))
	out := make([]DocumentSymbol, 0, len(items))
	for _, it := range items {
		line := it.Line - 1
		rng := Range{
			Start: Position{Line: line, Character: 0},
			End:   Position{Line: line, Character: 0},
		}
		out = append(out, DocumentSymbol{
			Name:           it.Name,
			Kind:           classifySymbolKind(it.Kind),
			Range:          rng,
			SelectionRange: rng,
		})
	}
	return out
}

// symbolKindToCompletion maps a doc.Item kind to the matching
// CompletionItem kind.
func symbolKindToCompletion(kind string) int {
	switch kind {
	case "fn":
		return CompletionFunction
	case "struct":
		return CompletionStruct
	case "enum":
		return CompletionClass
	case "trait":
		return CompletionInterface
	case "impl":
		return CompletionClass
	case "const", "static":
		return CompletionVariable
	}
	return CompletionText
}

// classifySymbolKind maps a doc.Item kind to the matching LSP
// SymbolKind.
func classifySymbolKind(kind string) int {
	switch kind {
	case "fn":
		return SymbolFunction
	case "struct":
		return SymbolStruct
	case "enum":
		return SymbolEnum
	case "trait":
		return SymbolInterface
	case "impl":
		return SymbolClass
	case "const", "static":
		return SymbolConstant
	}
	return SymbolVariable
}

// fuseKeywords is the completion-time set of Fuse keywords.
// Reference §1.4 pins the list; Fuse's keyword set is smaller than
// Rust's by design.
var fuseKeywords = []string{
	"fn", "let", "const", "static", "if", "else", "while", "for",
	"in", "match", "return", "break", "continue", "struct", "enum",
	"trait", "impl", "pub", "mod", "use", "as", "true", "false",
	"spawn", "unsafe", "move", "ref", "mut", "self", "Self",
}

// keywordSet is the O(1) lookup form of fuseKeywords.
var keywordSet = map[string]bool{}

func init() {
	for _, kw := range fuseKeywords {
		keywordSet[kw] = true
	}
}
