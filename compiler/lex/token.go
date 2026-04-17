package lex

import "fmt"

// TokenKind enumerates the lexical categories the scanner emits. It covers
// every construct listed in reference §1 (literals, identifiers, keywords)
// plus the operator and punctuation set that later waves consume.
type TokenKind int

const (
	TokInvalid TokenKind = iota
	TokEOF

	// Literals (reference §1.3–§1.6, §42.3).
	TokInt
	TokFloat
	TokString
	TokRawString
	TokCString
	TokChar
	TokTrue
	TokFalse

	// Identifier (reference §1.9).
	TokIdent

	// Keywords (reference §1.9 reserved-word list). Kept in the same order
	// as the reference to make table audits straightforward.
	TokKwFn
	TokKwPub
	TokKwStruct
	TokKwEnum
	TokKwTrait
	TokKwImpl
	TokKwFor
	TokKwIn
	TokKwWhile
	TokKwLoop
	TokKwIf
	TokKwElse
	TokKwMatch
	TokKwReturn
	TokKwLet
	TokKwVar
	TokKwMove
	TokKwRef
	TokKwMutref
	TokKwOwned
	TokKwUnsafe
	TokKwSpawn
	TokKwChan
	TokKwImport
	TokKwAs
	TokKwMod
	TokKwUse
	TokKwType
	TokKwConst
	TokKwStatic
	TokKwExtern
	TokKwBreak
	TokKwContinue
	TokKwWhere
	TokKwSelfType
	TokKwSelfVal
	TokKwNone
	TokKwSome

	// Punctuation (reference §1, §3, §4, §5, §6).
	TokLParen
	TokRParen
	TokLBracket
	TokRBracket
	TokLBrace
	TokRBrace
	TokComma
	TokSemi
	TokColon
	TokColonColon
	TokArrow
	TokFatArrow
	TokDot
	TokDotDot
	TokDotDotEq
	TokQuestion
	TokQuestionDot
	TokAt
	TokHash

	// Operators (reference §5).
	TokEq
	TokEqEq
	TokBangEq
	TokLt
	TokLe
	TokGt
	TokGe
	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokPercent
	TokBang
	TokAmp
	TokAmpAmp
	TokPipe
	TokPipePipe
	TokCaret
	TokTilde
	TokShl
	TokShr
	TokPlusEq
	TokMinusEq
	TokStarEq
	TokSlashEq
	TokPercentEq
	TokAmpEq
	TokPipeEq
	TokCaretEq
	TokShlEq
	TokShrEq

	tokKindCount
)

// kindNames maps every TokenKind to the stable name used in golden output,
// diagnostics, and test assertions. Any addition to the TokenKind enum must
// add a row here; TestTokenKindCoverage enforces this.
var kindNames = map[TokenKind]string{
	TokInvalid: "INVALID",
	TokEOF:     "EOF",

	TokInt:       "INT",
	TokFloat:     "FLOAT",
	TokString:    "STRING",
	TokRawString: "RAWSTRING",
	TokCString:   "CSTRING",
	TokChar:      "CHAR",
	TokTrue:      "TRUE",
	TokFalse:     "FALSE",

	TokIdent: "IDENT",

	TokKwFn:       "KW_FN",
	TokKwPub:      "KW_PUB",
	TokKwStruct:   "KW_STRUCT",
	TokKwEnum:     "KW_ENUM",
	TokKwTrait:    "KW_TRAIT",
	TokKwImpl:     "KW_IMPL",
	TokKwFor:      "KW_FOR",
	TokKwIn:       "KW_IN",
	TokKwWhile:    "KW_WHILE",
	TokKwLoop:     "KW_LOOP",
	TokKwIf:       "KW_IF",
	TokKwElse:     "KW_ELSE",
	TokKwMatch:    "KW_MATCH",
	TokKwReturn:   "KW_RETURN",
	TokKwLet:      "KW_LET",
	TokKwVar:      "KW_VAR",
	TokKwMove:     "KW_MOVE",
	TokKwRef:      "KW_REF",
	TokKwMutref:   "KW_MUTREF",
	TokKwOwned:    "KW_OWNED",
	TokKwUnsafe:   "KW_UNSAFE",
	TokKwSpawn:    "KW_SPAWN",
	TokKwChan:     "KW_CHAN",
	TokKwImport:   "KW_IMPORT",
	TokKwAs:       "KW_AS",
	TokKwMod:      "KW_MOD",
	TokKwUse:      "KW_USE",
	TokKwType:     "KW_TYPE",
	TokKwConst:    "KW_CONST",
	TokKwStatic:   "KW_STATIC",
	TokKwExtern:   "KW_EXTERN",
	TokKwBreak:    "KW_BREAK",
	TokKwContinue: "KW_CONTINUE",
	TokKwWhere:    "KW_WHERE",
	TokKwSelfType: "KW_SELFTYPE",
	TokKwSelfVal:  "KW_SELFVAL",
	TokKwNone:     "KW_NONE",
	TokKwSome:     "KW_SOME",

	TokLParen:      "LPAREN",
	TokRParen:      "RPAREN",
	TokLBracket:    "LBRACKET",
	TokRBracket:    "RBRACKET",
	TokLBrace:      "LBRACE",
	TokRBrace:      "RBRACE",
	TokComma:       "COMMA",
	TokSemi:        "SEMI",
	TokColon:       "COLON",
	TokColonColon:  "COLONCOLON",
	TokArrow:       "ARROW",
	TokFatArrow:    "FATARROW",
	TokDot:         "DOT",
	TokDotDot:      "DOTDOT",
	TokDotDotEq:    "DOTDOTEQ",
	TokQuestion:    "QUESTION",
	TokQuestionDot: "QUESTIONDOT",
	TokAt:          "AT",
	TokHash:        "HASH",

	TokEq:        "EQ",
	TokEqEq:      "EQEQ",
	TokBangEq:    "BANGEQ",
	TokLt:        "LT",
	TokLe:        "LE",
	TokGt:        "GT",
	TokGe:        "GE",
	TokPlus:      "PLUS",
	TokMinus:     "MINUS",
	TokStar:      "STAR",
	TokSlash:     "SLASH",
	TokPercent:   "PERCENT",
	TokBang:      "BANG",
	TokAmp:       "AMP",
	TokAmpAmp:    "AMPAMP",
	TokPipe:      "PIPE",
	TokPipePipe:  "PIPEPIPE",
	TokCaret:     "CARET",
	TokTilde:     "TILDE",
	TokShl:       "SHL",
	TokShr:       "SHR",
	TokPlusEq:    "PLUSEQ",
	TokMinusEq:   "MINUSEQ",
	TokStarEq:    "STAREQ",
	TokSlashEq:   "SLASHEQ",
	TokPercentEq: "PERCENTEQ",
	TokAmpEq:     "AMPEQ",
	TokPipeEq:    "PIPEEQ",
	TokCaretEq:   "CARETEQ",
	TokShlEq:     "SHLEQ",
	TokShrEq:     "SHREQ",
}

// String returns the stable textual name of the token kind.
func (k TokenKind) String() string {
	if name, ok := kindNames[k]; ok {
		return name
	}
	return fmt.Sprintf("TokenKind(%d)", int(k))
}

// keywords maps every reserved word (reference §1.9) to its TokenKind. Order
// must match the reference-listed order; TestKeywords enforces this by
// iterating keywordList in order.
var keywords = map[string]TokenKind{
	"fn": TokKwFn, "pub": TokKwPub, "struct": TokKwStruct, "enum": TokKwEnum,
	"trait": TokKwTrait, "impl": TokKwImpl, "for": TokKwFor, "in": TokKwIn,
	"while": TokKwWhile, "loop": TokKwLoop, "if": TokKwIf, "else": TokKwElse,
	"match": TokKwMatch, "return": TokKwReturn, "let": TokKwLet, "var": TokKwVar,
	"move": TokKwMove, "ref": TokKwRef, "mutref": TokKwMutref, "owned": TokKwOwned,
	"unsafe": TokKwUnsafe, "spawn": TokKwSpawn, "chan": TokKwChan, "import": TokKwImport,
	"as": TokKwAs, "mod": TokKwMod, "use": TokKwUse, "type": TokKwType,
	"const": TokKwConst, "static": TokKwStatic, "extern": TokKwExtern,
	"break": TokKwBreak, "continue": TokKwContinue, "where": TokKwWhere,
	"Self": TokKwSelfType, "self": TokKwSelfVal,
	"true": TokTrue, "false": TokFalse,
	"None": TokKwNone, "Some": TokKwSome,
}

// keywordList is the reference-ordered list of reserved words from §1.9. It
// exists so tests and diagnostics can iterate keywords deterministically
// without depending on Go map iteration order (Rule 3.6, Rule 7.3).
var keywordList = []string{
	"fn", "pub", "struct", "enum", "trait", "impl", "for", "in", "while", "loop",
	"if", "else", "match", "return", "let", "var", "move", "ref", "mutref", "owned",
	"unsafe", "spawn", "chan", "import", "as", "mod", "use", "type", "const", "static",
	"extern", "break", "continue", "where", "Self", "self", "true", "false", "None", "Some",
}

// IsKeyword reports whether s is a reserved word (reference §1.9) and returns
// the corresponding token kind.
func IsKeyword(s string) (TokenKind, bool) {
	k, ok := keywords[s]
	return k, ok
}
