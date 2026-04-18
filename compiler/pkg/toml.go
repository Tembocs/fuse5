package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

// Minimal TOML subset parser for fuse.toml. The supported grammar:
//
//   top-level := ( section | comment | blank )*
//   section := `[` name `]` entry*
//   name := IDENT ( `.` IDENT )*
//   entry := key `=` value
//   value := string | integer | boolean | array | inline-table
//   string := `"` chars `"`
//   array  := `[` value ( `,` value )* `]`
//   inline-table := `{` (key `=` value) ( `,` key `=` value )* `}`
//
// Unsupported: date-time values, nested tables beyond a single
// dot level, triple-quoted strings, literal strings. Those are
// not yet needed by fuse.toml and adding them is a follow-up.

type tomlDoc struct {
	sections []*tomlSection
}

type tomlSection struct {
	name    string
	entries []tomlKV
}

type tomlKV struct {
	key   string
	value *tomlValue
}

type tomlValue struct {
	kind    tomlKind
	str     string
	integer int64
	boolean bool
	array   []*tomlValue
	table   []tomlKV
}

type tomlKind int

const (
	tomlKindString tomlKind = iota + 1
	tomlKindInteger
	tomlKindBool
	tomlKindArray
	tomlKindInlineTable
)

type tomlParser struct {
	src []byte
	pos int
}

// parseToml is the entry point. Returns a parsed document or a
// parse error pointing at the first offending byte.
func parseToml(src []byte) (*tomlDoc, error) {
	p := &tomlParser{src: src}
	doc := &tomlDoc{}
	// Implicit top-level section: any key/value before the
	// first [section] belongs to an unnamed root. At fuse.toml
	// this would be a user error; reject it.
	var current *tomlSection
	for !p.eof() {
		p.skipSpaceAndComments()
		if p.eof() {
			break
		}
		if p.peek() == '[' {
			name, err := p.parseSectionHeader()
			if err != nil {
				return nil, err
			}
			current = &tomlSection{name: name}
			doc.sections = append(doc.sections, current)
			continue
		}
		if current == nil {
			return nil, p.errorf("key outside of any section — add a [section] header before line %d", p.line())
		}
		kv, err := p.parseEntry()
		if err != nil {
			return nil, err
		}
		current.entries = append(current.entries, kv)
	}
	return doc, nil
}

func (p *tomlParser) skipSpaceAndComments() {
	for !p.eof() {
		c := p.peek()
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
			continue
		}
		if c == '#' {
			for !p.eof() && p.peek() != '\n' {
				p.pos++
			}
			continue
		}
		break
	}
}

func (p *tomlParser) parseSectionHeader() (string, error) {
	// consume `[`
	p.pos++
	start := p.pos
	for !p.eof() && p.peek() != ']' && p.peek() != '\n' {
		p.pos++
	}
	if p.eof() || p.peek() != ']' {
		return "", p.errorf("unterminated section header")
	}
	name := strings.TrimSpace(string(p.src[start:p.pos]))
	p.pos++ // consume `]`
	if name == "" {
		return "", p.errorf("empty section name")
	}
	return name, nil
}

func (p *tomlParser) parseEntry() (tomlKV, error) {
	key, err := p.parseKey()
	if err != nil {
		return tomlKV{}, err
	}
	p.skipInlineWhitespace()
	if p.eof() || p.peek() != '=' {
		return tomlKV{}, p.errorf("expected `=` after key %q", key)
	}
	p.pos++ // consume `=`
	p.skipInlineWhitespace()
	val, err := p.parseValue()
	if err != nil {
		return tomlKV{}, err
	}
	return tomlKV{key: key, value: val}, nil
}

func (p *tomlParser) skipInlineWhitespace() {
	for !p.eof() && (p.peek() == ' ' || p.peek() == '\t') {
		p.pos++
	}
}

func (p *tomlParser) parseKey() (string, error) {
	start := p.pos
	for !p.eof() {
		c := p.peek()
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' {
			p.pos++
			continue
		}
		break
	}
	if p.pos == start {
		return "", p.errorf("expected identifier key")
	}
	return string(p.src[start:p.pos]), nil
}

func (p *tomlParser) parseValue() (*tomlValue, error) {
	if p.eof() {
		return nil, p.errorf("expected value, got EOF")
	}
	c := p.peek()
	switch {
	case c == '"':
		s, err := p.parseString()
		if err != nil {
			return nil, err
		}
		return &tomlValue{kind: tomlKindString, str: s}, nil
	case c == '[':
		return p.parseArray()
	case c == '{':
		return p.parseInlineTable()
	case c == 't' || c == 'f':
		return p.parseBool()
	case c == '-' || (c >= '0' && c <= '9'):
		return p.parseInteger()
	}
	return nil, p.errorf("unexpected character %q in value", c)
}

func (p *tomlParser) parseString() (string, error) {
	p.pos++ // consume opening `"`
	var sb strings.Builder
	for !p.eof() {
		c := p.peek()
		if c == '"' {
			p.pos++
			return sb.String(), nil
		}
		if c == '\\' {
			p.pos++
			if p.eof() {
				return "", p.errorf("unterminated string escape")
			}
			esc := p.peek()
			p.pos++
			switch esc {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			default:
				return "", p.errorf("unknown escape \\%q", esc)
			}
			continue
		}
		sb.WriteByte(c)
		p.pos++
	}
	return "", p.errorf("unterminated string")
}

func (p *tomlParser) parseArray() (*tomlValue, error) {
	p.pos++ // consume `[`
	out := &tomlValue{kind: tomlKindArray}
	for {
		p.skipSpaceAndComments()
		if p.eof() {
			return nil, p.errorf("unterminated array")
		}
		if p.peek() == ']' {
			p.pos++
			return out, nil
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		out.array = append(out.array, v)
		p.skipSpaceAndComments()
		if p.eof() {
			return nil, p.errorf("unterminated array")
		}
		if p.peek() == ',' {
			p.pos++
			continue
		}
		if p.peek() == ']' {
			p.pos++
			return out, nil
		}
		return nil, p.errorf("expected `,` or `]` in array")
	}
}

func (p *tomlParser) parseInlineTable() (*tomlValue, error) {
	p.pos++ // consume `{`
	out := &tomlValue{kind: tomlKindInlineTable}
	for {
		p.skipInlineWhitespace()
		if p.eof() {
			return nil, p.errorf("unterminated inline table")
		}
		if p.peek() == '}' {
			p.pos++
			return out, nil
		}
		kv, err := p.parseEntry()
		if err != nil {
			return nil, err
		}
		out.table = append(out.table, kv)
		p.skipInlineWhitespace()
		if p.eof() {
			return nil, p.errorf("unterminated inline table")
		}
		if p.peek() == ',' {
			p.pos++
			continue
		}
		if p.peek() == '}' {
			p.pos++
			return out, nil
		}
		return nil, p.errorf("expected `,` or `}` in inline table")
	}
}

func (p *tomlParser) parseBool() (*tomlValue, error) {
	if strings.HasPrefix(string(p.src[p.pos:]), "true") {
		p.pos += 4
		return &tomlValue{kind: tomlKindBool, boolean: true}, nil
	}
	if strings.HasPrefix(string(p.src[p.pos:]), "false") {
		p.pos += 5
		return &tomlValue{kind: tomlKindBool, boolean: false}, nil
	}
	return nil, p.errorf("expected boolean literal")
}

func (p *tomlParser) parseInteger() (*tomlValue, error) {
	start := p.pos
	if p.peek() == '-' {
		p.pos++
	}
	for !p.eof() {
		c := p.peek()
		if c < '0' || c > '9' {
			break
		}
		p.pos++
	}
	raw := string(p.src[start:p.pos])
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, p.errorf("invalid integer %q: %v", raw, err)
	}
	return &tomlValue{kind: tomlKindInteger, integer: n}, nil
}

func (p *tomlParser) eof() bool   { return p.pos >= len(p.src) }
func (p *tomlParser) peek() byte  { return p.src[p.pos] }
func (p *tomlParser) line() int   { return strings.Count(string(p.src[:p.pos]), "\n") + 1 }
func (p *tomlParser) errorf(format string, args ...any) error {
	return fmt.Errorf("fuse.toml:%d: %s", p.line(), fmt.Sprintf(format, args...))
}
