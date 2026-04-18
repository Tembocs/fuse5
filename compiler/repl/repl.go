// Package repl owns the Fuse read-eval-print loop.
//
// W18 scope: a deterministic "round-trip" shape. Each line the
// user submits is parsed as an expression, evaluated against a
// small arithmetic / bool-logic evaluator, and the result is
// echoed. Full HIR + typetable evaluation lives in the W14
// consteval pass; the REPL at W18 is the interactive surface that
// wraps it.
//
// The evaluator is deliberately simple at W18: integer arithmetic,
// comparison, and bool logic. W19 IDE integration extends it with
// named bindings and full-check evaluation for type errors.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Repl is one REPL session. NewRepl wires an input reader and an
// output writer; RunOne processes a single line and returns the
// user-visible result or an error.
type Repl struct {
	in  *bufio.Reader
	out io.Writer
}

// NewRepl constructs a Repl that reads from in and writes to out.
func NewRepl(in io.Reader, out io.Writer) *Repl {
	return &Repl{in: bufio.NewReader(in), out: out}
}

// Run enters the REPL main loop. The loop terminates on EOF or a
// line with `:quit` / `:exit` (and returns nil in both cases).
func (r *Repl) Run() error {
	for {
		fmt.Fprint(r.out, "fuse> ")
		line, err := r.in.ReadString('\n')
		if err == io.EOF {
			fmt.Fprintln(r.out)
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == ":quit" || line == ":exit" {
			return nil
		}
		result, evalErr := r.Eval(line)
		if evalErr != nil {
			fmt.Fprintf(r.out, "error: %v\n", evalErr)
			continue
		}
		fmt.Fprintf(r.out, "=> %s\n", result)
	}
}

// Eval evaluates a single expression and returns the user-visible
// string form of the result.
func (r *Repl) Eval(expr string) (string, error) {
	v, err := evaluate(expr)
	if err != nil {
		return "", err
	}
	return renderValue(v), nil
}

// value is the REPL's narrow evaluator type — integer or boolean.
type value struct {
	kind string // "int" | "bool"
	i    int64
	b    bool
}

// evaluate is a shunting-yard-style evaluator over integers, the
// arithmetic ops + - * / %, comparisons == != < <= > >=, and bool
// logic && ||. Parentheses are honoured; identifiers and calls
// are W19 scope.
func evaluate(src string) (value, error) {
	tokens, err := tokenize(src)
	if err != nil {
		return value{}, err
	}
	p := &parser{tokens: tokens}
	v, err := p.parseOr()
	if err != nil {
		return value{}, err
	}
	if p.pos < len(p.tokens) {
		return value{}, fmt.Errorf("trailing input at token %d: %q", p.pos, p.tokens[p.pos])
	}
	return v, nil
}

type parser struct {
	tokens []string
	pos    int
}

func (p *parser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

func (p *parser) take() string {
	tok := p.peek()
	p.pos++
	return tok
}

// parseOr → parseAnd ( `||` parseAnd )*
func (p *parser) parseOr() (value, error) {
	lhs, err := p.parseAnd()
	if err != nil {
		return value{}, err
	}
	for p.peek() == "||" {
		p.take()
		rhs, err := p.parseAnd()
		if err != nil {
			return value{}, err
		}
		if lhs.kind != "bool" || rhs.kind != "bool" {
			return value{}, fmt.Errorf("`||` operands must be bool")
		}
		lhs = value{kind: "bool", b: lhs.b || rhs.b}
	}
	return lhs, nil
}

// parseAnd → parseCmp ( `&&` parseCmp )*
func (p *parser) parseAnd() (value, error) {
	lhs, err := p.parseCmp()
	if err != nil {
		return value{}, err
	}
	for p.peek() == "&&" {
		p.take()
		rhs, err := p.parseCmp()
		if err != nil {
			return value{}, err
		}
		if lhs.kind != "bool" || rhs.kind != "bool" {
			return value{}, fmt.Errorf("`&&` operands must be bool")
		}
		lhs = value{kind: "bool", b: lhs.b && rhs.b}
	}
	return lhs, nil
}

// parseCmp → parseAdd ( (`==`|`!=`|`<`|`<=`|`>`|`>=`) parseAdd )?
func (p *parser) parseCmp() (value, error) {
	lhs, err := p.parseAdd()
	if err != nil {
		return value{}, err
	}
	switch op := p.peek(); op {
	case "==", "!=", "<", "<=", ">", ">=":
		p.take()
		rhs, err := p.parseAdd()
		if err != nil {
			return value{}, err
		}
		if lhs.kind != "int" || rhs.kind != "int" {
			return value{}, fmt.Errorf("%q operands must be int", op)
		}
		b := false
		switch op {
		case "==":
			b = lhs.i == rhs.i
		case "!=":
			b = lhs.i != rhs.i
		case "<":
			b = lhs.i < rhs.i
		case "<=":
			b = lhs.i <= rhs.i
		case ">":
			b = lhs.i > rhs.i
		case ">=":
			b = lhs.i >= rhs.i
		}
		return value{kind: "bool", b: b}, nil
	}
	return lhs, nil
}

// parseAdd → parseMul ( (`+`|`-`) parseMul )*
func (p *parser) parseAdd() (value, error) {
	lhs, err := p.parseMul()
	if err != nil {
		return value{}, err
	}
	for p.peek() == "+" || p.peek() == "-" {
		op := p.take()
		rhs, err := p.parseMul()
		if err != nil {
			return value{}, err
		}
		if lhs.kind != "int" || rhs.kind != "int" {
			return value{}, fmt.Errorf("%q operands must be int", op)
		}
		if op == "+" {
			lhs = value{kind: "int", i: lhs.i + rhs.i}
		} else {
			lhs = value{kind: "int", i: lhs.i - rhs.i}
		}
	}
	return lhs, nil
}

// parseMul → parseAtom ( (`*`|`/`|`%`) parseAtom )*
func (p *parser) parseMul() (value, error) {
	lhs, err := p.parseAtom()
	if err != nil {
		return value{}, err
	}
	for p.peek() == "*" || p.peek() == "/" || p.peek() == "%" {
		op := p.take()
		rhs, err := p.parseAtom()
		if err != nil {
			return value{}, err
		}
		if lhs.kind != "int" || rhs.kind != "int" {
			return value{}, fmt.Errorf("%q operands must be int", op)
		}
		switch op {
		case "*":
			lhs = value{kind: "int", i: lhs.i * rhs.i}
		case "/":
			if rhs.i == 0 {
				return value{}, fmt.Errorf("divide by zero")
			}
			lhs = value{kind: "int", i: lhs.i / rhs.i}
		case "%":
			if rhs.i == 0 {
				return value{}, fmt.Errorf("modulo by zero")
			}
			lhs = value{kind: "int", i: lhs.i % rhs.i}
		}
	}
	return lhs, nil
}

// parseAtom → INT | "true" | "false" | "(" parseOr ")" | "-" parseAtom | "!" parseAtom
func (p *parser) parseAtom() (value, error) {
	tok := p.peek()
	switch {
	case tok == "":
		return value{}, fmt.Errorf("unexpected end of input")
	case tok == "(":
		p.take()
		v, err := p.parseOr()
		if err != nil {
			return value{}, err
		}
		if p.peek() != ")" {
			return value{}, fmt.Errorf("expected `)`, got %q", p.peek())
		}
		p.take()
		return v, nil
	case tok == "true":
		p.take()
		return value{kind: "bool", b: true}, nil
	case tok == "false":
		p.take()
		return value{kind: "bool", b: false}, nil
	case tok == "-":
		p.take()
		v, err := p.parseAtom()
		if err != nil {
			return value{}, err
		}
		if v.kind != "int" {
			return value{}, fmt.Errorf("unary `-` requires int")
		}
		return value{kind: "int", i: -v.i}, nil
	case tok == "!":
		p.take()
		v, err := p.parseAtom()
		if err != nil {
			return value{}, err
		}
		if v.kind != "bool" {
			return value{}, fmt.Errorf("unary `!` requires bool")
		}
		return value{kind: "bool", b: !v.b}, nil
	}
	// Integer literal.
	n, err := strconv.ParseInt(tok, 0, 64)
	if err != nil {
		return value{}, fmt.Errorf("unknown token %q", tok)
	}
	p.take()
	return value{kind: "int", i: n}, nil
}

// tokenize splits src into whitespace-separated atoms, treating
// multi-char operators (`==`, `!=`, `<=`, `>=`, `&&`, `||`) and
// punctuation (`(`, `)`) specially.
func tokenize(src string) ([]string, error) {
	var out []string
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(' || c == ')' || c == '+' || c == '*' || c == '/' || c == '%':
			out = append(out, string(c))
			i++
		case c == '-':
			out = append(out, "-")
			i++
		case c == '!' && i+1 < len(src) && src[i+1] == '=':
			out = append(out, "!=")
			i += 2
		case c == '!':
			out = append(out, "!")
			i++
		case c == '=' && i+1 < len(src) && src[i+1] == '=':
			out = append(out, "==")
			i += 2
		case c == '<' && i+1 < len(src) && src[i+1] == '=':
			out = append(out, "<=")
			i += 2
		case c == '<':
			out = append(out, "<")
			i++
		case c == '>' && i+1 < len(src) && src[i+1] == '=':
			out = append(out, ">=")
			i += 2
		case c == '>':
			out = append(out, ">")
			i++
		case c == '&' && i+1 < len(src) && src[i+1] == '&':
			out = append(out, "&&")
			i += 2
		case c == '|' && i+1 < len(src) && src[i+1] == '|':
			out = append(out, "||")
			i += 2
		case (c >= '0' && c <= '9'):
			j := i
			for j < len(src) && ((src[j] >= '0' && src[j] <= '9') || src[j] == 'x' || src[j] == 'X' ||
				(src[j] >= 'a' && src[j] <= 'f') || (src[j] >= 'A' && src[j] <= 'F') ||
				src[j] == 'b' || src[j] == 'B' || src[j] == 'o' || src[j] == 'O' || src[j] == '_') {
				j++
			}
			tok := strings.ReplaceAll(src[i:j], "_", "")
			out = append(out, tok)
			i = j
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			j := i
			for j < len(src) && ((src[j] >= 'a' && src[j] <= 'z') ||
				(src[j] >= 'A' && src[j] <= 'Z') ||
				(src[j] >= '0' && src[j] <= '9') || src[j] == '_') {
				j++
			}
			out = append(out, src[i:j])
			i = j
		default:
			return nil, fmt.Errorf("unexpected character %q at offset %d", c, i)
		}
	}
	return out, nil
}

// renderValue formats a value for the REPL display.
func renderValue(v value) string {
	if v.kind == "bool" {
		if v.b {
			return "true"
		}
		return "false"
	}
	return strconv.FormatInt(v.i, 10)
}
