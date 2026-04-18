package lsp

import (
	"strings"
	"sync"
	"unicode/utf16"
	"unicode/utf8"
)

// DocStore is the in-memory model of the workspace's open
// documents. The server applies didOpen / didChange / didClose
// notifications to the store and re-derives diagnostics from it
// whenever the text changes.
//
// Concurrency: every public method is safe under a single
// goroutine doing reads + writes. The server is strictly
// single-threaded for state mutations, so the mutex protects
// against race-detector warnings but doesn't shape the caller's
// contract.
type DocStore struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// Document is one open source file.
type Document struct {
	URI      string
	Version  int
	Text     string
	Language string
}

// NewDocStore returns a fresh, empty store.
func NewDocStore() *DocStore {
	return &DocStore{docs: map[string]*Document{}}
}

// Open records a new document. An existing URI is replaced —
// didOpen on an already-open document is permitted per the LSP
// spec and is observationally equivalent to close+open.
func (s *DocStore) Open(uri, languageID, text string, version int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = &Document{URI: uri, Version: version, Text: text, Language: languageID}
}

// Close removes a document. Closing an unknown URI is a no-op —
// clients sometimes close documents they never formally opened.
func (s *DocStore) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

// Apply mutates the document referenced by ident with the given
// content changes. A change whose Range is nil is a full-text
// replacement; otherwise the range is an incremental edit.
//
// Returns the updated Document, or nil when the URI is unknown.
func (s *DocStore) Apply(ident VersionedTextDocumentIdentifier, changes []TextDocumentContentChangeEvent) *Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[ident.URI]
	if !ok {
		return nil
	}
	for _, ch := range changes {
		if ch.Range == nil {
			doc.Text = ch.Text
			continue
		}
		doc.Text = applyRangeEdit(doc.Text, *ch.Range, ch.Text)
	}
	doc.Version = ident.Version
	return doc
}

// Get returns the Document for uri, or nil when not open.
func (s *DocStore) Get(uri string) *Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.docs[uri]
}

// URIs returns every open document URI sorted lexicographically
// for determinism. Used by workspace/symbol.
func (s *DocStore) URIs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.docs))
	for uri := range s.docs {
		out = append(out, uri)
	}
	// Lexicographic order for deterministic iteration.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// applyRangeEdit replaces the bytes in text within r with newText
// and returns the result. Line/column are 0-based as in LSP.
func applyRangeEdit(text string, r Range, newText string) string {
	startOffset := positionToByteOffset(text, r.Start)
	endOffset := positionToByteOffset(text, r.End)
	if startOffset < 0 || endOffset < 0 || startOffset > len(text) || endOffset > len(text) || startOffset > endOffset {
		return text + newText // degrade gracefully; server would reject the edit upstream
	}
	var sb strings.Builder
	sb.Grow(len(text) - (endOffset - startOffset) + len(newText))
	sb.WriteString(text[:startOffset])
	sb.WriteString(newText)
	sb.WriteString(text[endOffset:])
	return sb.String()
}

// positionToByteOffset converts a 0-based (line, character)
// LSP position to a byte offset in text. Returns -1 for positions
// past the end of the document.
//
// LSP 3.17 §3.17/PositionEncodingKind defaults character to a
// UTF-16 code-unit count. Fuse sources are stored as UTF-8; we
// decode runes and advance Character by utf16.RuneLen(r) per
// rune. ASCII text is observationally unchanged — one byte per
// rune, one UTF-16 code unit per rune — so existing test
// corpora remain valid.
func positionToByteOffset(text string, p Position) int {
	if p.Line < 0 || p.Character < 0 {
		return -1
	}
	line := 0
	lineStart := 0
	// Walk to the target line.
	for i := 0; i < len(text); i++ {
		if line == p.Line {
			break
		}
		if text[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	if line != p.Line {
		// p.Line past end of document.
		if p.Line == line+1 && p.Character == 0 && len(text) > 0 && text[len(text)-1] == '\n' {
			return len(text)
		}
		return -1
	}
	// Walk UTF-16 code units along the target line.
	col := 0
	off := lineStart
	for off < len(text) && text[off] != '\n' {
		if col == p.Character {
			return off
		}
		r, size := utf8.DecodeRuneInString(text[off:])
		col += utf16.RuneLen(r)
		off += size
	}
	if col == p.Character {
		return off
	}
	return -1
}

// byteOffsetToPosition is the inverse mapping used to convert
// compiler spans back to LSP positions. Emits Character as a
// UTF-16 code-unit count (LSP 3.17 §3.17/PositionEncodingKind).
func byteOffsetToPosition(text string, offset int) Position {
	if offset < 0 {
		return Position{}
	}
	if offset > len(text) {
		offset = len(text)
	}
	line := 0
	lineStart := 0
	for i := 0; i < offset; i++ {
		if text[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	col := 0
	for i := lineStart; i < offset; {
		r, size := utf8.DecodeRuneInString(text[i:])
		col += utf16.RuneLen(r)
		i += size
	}
	return Position{Line: line, Character: col}
}
