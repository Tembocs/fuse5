package lsp

// semanticTokenTypes is the legend the server advertises. Index in
// this slice = token-type integer in the packed data stream.
func semanticTokenTypes() []string {
	return []string{
		"keyword",
		"type",
		"function",
		"variable",
		"parameter",
		"number",
		"string",
		"comment",
		"operator",
	}
}

// Indices into semanticTokenTypes() for callers of the packed
// encoder.
const (
	tokKeyword  = 0
	tokType     = 1
	tokFunction = 2
	tokVariable = 3
	tokParameter = 4
	tokNumber    = 5
	tokString    = 6
	tokComment   = 7
	tokOperator  = 8
)

// computeSemanticTokens scans text and returns the packed
// [deltaLine, deltaStart, length, tokenType, tokenModifiers] × N
// array that LSP clients consume for semantic highlighting.
//
// W19 uses a simple line-based tokeniser: no full lex pass, just
// an identifier regex + keyword lookup + "CamelCase => type"
// heuristic + "identifier followed by `(` => function" detection.
// The regex-based classifier matches the W19 scope; W22 stdlib
// replaces it with a real lex pass.
func computeSemanticTokens(text string) []int {
	var out []int
	lastLine, lastCol := 0, 0
	lines := splitLines(text)
	for lineIdx, line := range lines {
		for _, m := range identifierPattern.FindAllStringIndex(line, -1) {
			start := m[0]
			length := m[1] - m[0]
			word := line[start:m[1]]
			tokType := classifySemanticToken(word, line, m[1])
			deltaLine := lineIdx - lastLine
			deltaStart := start
			if deltaLine == 0 {
				deltaStart -= lastCol
			}
			out = append(out, deltaLine, deltaStart, length, tokType, 0)
			lastLine = lineIdx
			lastCol = start
		}
	}
	return out
}

// classifySemanticToken picks the token-type integer for an
// identifier: keyword → tokKeyword, CamelCase → tokType,
// identifier followed by `(` → tokFunction, else tokVariable.
func classifySemanticToken(word, line string, endOffset int) int {
	if keywordSet[word] {
		return tokKeyword
	}
	if endOffset < len(line) && line[endOffset] == '(' {
		return tokFunction
	}
	if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
		return tokType
	}
	return tokVariable
}

// splitLines splits text on newlines without stripping the
// terminator, so column offsets remain consistent across
// platforms.
func splitLines(text string) []string {
	var out []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			out = append(out, text[start:i])
			start = i + 1
		}
	}
	if start <= len(text) {
		out = append(out, text[start:])
	}
	return out
}
