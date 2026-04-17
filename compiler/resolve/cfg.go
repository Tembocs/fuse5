package resolve

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// BuildConfig holds the evaluation environment for `@cfg` predicates
// (reference §50.1). Vars is the key/value map used by
// `@cfg(name = "value")` forms; Features is the boolean-flag map used
// by `@cfg(feature = "name")` forms.
//
// Both maps may be nil; a nil map behaves as an empty map.
type BuildConfig struct {
	Vars     map[string]string
	Features map[string]bool
}

// lookupVar returns (value, true) when name is present in Vars.
func (c BuildConfig) lookupVar(name string) (string, bool) {
	if c.Vars == nil {
		return "", false
	}
	v, ok := c.Vars[name]
	return v, ok
}

// hasFeature reports whether a feature flag is enabled.
func (c BuildConfig) hasFeature(name string) bool {
	if c.Features == nil {
		return false
	}
	return c.Features[name]
}

// evalCfgDecorator is the top-level @cfg evaluator. It accepts an
// `@cfg(...)` decorator and returns (keep, diagnostics). When the
// decorator is syntactically malformed, diagnostics describe the problem
// and keep is false so the item is removed from the build (Rule 6.9 —
// never produce silent wrong output).
func evalCfgDecorator(d *ast.Decorator, cfg BuildConfig, filename string) (bool, []lex.Diagnostic) {
	// `@cfg()` with no arguments is treated as unconditionally true.
	// Empty conjunctions are already the identity of `all()`.
	if len(d.Args) == 0 {
		return true, nil
	}
	var diags []lex.Diagnostic
	keep := true
	for _, arg := range d.Args {
		ok, d2 := evalCfgArg(arg, cfg, filename)
		diags = append(diags, d2...)
		if !ok {
			keep = false
		}
	}
	return keep, diags
}

// evalCfgArg evaluates one top-level argument to `@cfg`. The argument
// may be a named form (`os = "linux"` or `feature = "x"`) or a nested
// predicate expression (`not(...)`, `all(...)`, `any(...)`).
func evalCfgArg(arg *ast.DecoratorArg, cfg BuildConfig, filename string) (bool, []lex.Diagnostic) {
	if arg.Name != nil {
		return evalCfgKeyValue(arg.Name.Name, arg.Value, cfg, filename, arg.NodeSpan())
	}
	return evalCfgPred(arg.Value, cfg, filename)
}

// evalCfgPred evaluates a predicate expression. Supported forms:
//
//   - CallExpr with callee `not`/`all`/`any`
//   - AssignExpr `name = "value"` (nested key/value form)
//   - PathExpr for a bare key is not supported (the reference requires
//     explicit key = value form).
func evalCfgPred(e ast.Expr, cfg BuildConfig, filename string) (bool, []lex.Diagnostic) {
	switch x := e.(type) {
	case *ast.CallExpr:
		name, ok := callName(x)
		if !ok {
			return false, []lex.Diagnostic{cfgError(x.NodeSpan(), filename,
				"malformed @cfg predicate",
				"use `name = \"value\"`, `not(...)`, `all(...)`, or `any(...)`")}
		}
		switch name {
		case "not":
			if len(x.Args) != 1 {
				return false, []lex.Diagnostic{cfgError(x.NodeSpan(), filename,
					"`not(...)` takes exactly one argument",
					"pass a single predicate like `not(os = \"linux\")`")}
			}
			v, d := evalCfgPred(x.Args[0], cfg, filename)
			return !v, d
		case "all":
			var ds []lex.Diagnostic
			for _, a := range x.Args {
				v, d := evalCfgPred(a, cfg, filename)
				ds = append(ds, d...)
				if !v {
					return false, ds
				}
			}
			return true, ds
		case "any":
			var ds []lex.Diagnostic
			hit := false
			for _, a := range x.Args {
				v, d := evalCfgPred(a, cfg, filename)
				ds = append(ds, d...)
				if v {
					hit = true
				}
			}
			return hit, ds
		default:
			return false, []lex.Diagnostic{cfgError(x.NodeSpan(), filename,
				fmt.Sprintf("unknown @cfg combinator %q", name),
				"use `not`, `all`, or `any`")}
		}
	case *ast.AssignExpr:
		if x.Op != ast.AssignEq {
			return false, []lex.Diagnostic{cfgError(x.NodeSpan(), filename,
				"@cfg key/value uses `=`, not compound assignment",
				"write `name = \"value\"`")}
		}
		key, ok := pathToIdent(x.Lhs)
		if !ok {
			return false, []lex.Diagnostic{cfgError(x.Lhs.NodeSpan(), filename,
				"@cfg key must be a bare identifier",
				"for example `os = \"linux\"`")}
		}
		return evalCfgKeyValue(key, x.Rhs, cfg, filename, x.NodeSpan())
	default:
		return false, []lex.Diagnostic{cfgError(e.NodeSpan(), filename,
			"malformed @cfg predicate",
			"use `name = \"value\"`, `not(...)`, `all(...)`, or `any(...)`")}
	}
}

// evalCfgKeyValue implements the `name = "value"` form. Special-cases
// `feature = "x"` to consult BuildConfig.Features.
func evalCfgKeyValue(key string, value ast.Expr, cfg BuildConfig, filename string, span lex.Span) (bool, []lex.Diagnostic) {
	_ = span // accepted for symmetry with other evaluators; unused here
	strVal, ok := stringLit(value)
	if !ok {
		return false, []lex.Diagnostic{cfgError(value.NodeSpan(), filename,
			"@cfg value must be a string literal",
			fmt.Sprintf("write `%s = \"...\"`", key))}
	}
	if key == "feature" {
		return cfg.hasFeature(strVal), nil
	}
	got, present := cfg.lookupVar(key)
	if !present {
		return false, nil
	}
	return got == strVal, nil
}

// callName extracts the callee identifier of a CallExpr when the callee
// is a single-segment PathExpr. Returns ("", false) otherwise.
func callName(c *ast.CallExpr) (string, bool) {
	return pathToIdent(c.Callee)
}

// pathToIdent returns the identifier text when e is a PathExpr with a
// single segment and no type arguments. Otherwise ("", false).
func pathToIdent(e ast.Expr) (string, bool) {
	p, ok := e.(*ast.PathExpr)
	if !ok {
		return "", false
	}
	if len(p.Segments) != 1 || len(p.TypeArgs) != 0 {
		return "", false
	}
	return p.Segments[0].Name, true
}

// stringLit unwraps a literal expression to its text when it is a
// string or raw-string literal. The literal text is the *unquoted*
// source spelling (reference §1.10 — content escaping is the checker's
// responsibility in W06); for @cfg matching we compare the quoted form
// with surrounding quotes stripped.
func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.LiteralExpr)
	if !ok {
		return "", false
	}
	switch lit.Kind {
	case ast.LitString, ast.LitRawString, ast.LitCString:
		return unquoteLiteral(lit.Text), true
	}
	return "", false
}

// unquoteLiteral strips surrounding quote marks from a string-literal
// source text. It is intentionally minimal — escape handling is the
// checker's job in W06. The @cfg evaluator needs only the lexical
// payload for plain ASCII comparisons like `os == "linux"`.
func unquoteLiteral(text string) string {
	if len(text) >= 2 {
		first, last := text[0], text[len(text)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return text[1 : len(text)-1]
		}
	}
	// Raw strings are written `r"..."` or `r#"..."#`.
	if len(text) >= 3 && text[0] == 'r' {
		rest := text[1:]
		if len(rest) >= 2 && rest[0] == '"' && rest[len(rest)-1] == '"' {
			return rest[1 : len(rest)-1]
		}
		// `r#"..."#`
		hashes := 0
		for hashes < len(rest) && rest[hashes] == '#' {
			hashes++
		}
		if hashes > 0 && hashes*2 < len(rest) && rest[hashes] == '"' && rest[len(rest)-hashes-1] == '"' {
			return rest[hashes+1 : len(rest)-hashes-1]
		}
	}
	// C strings are written `c"..."`.
	if len(text) >= 3 && text[0] == 'c' && text[1] == '"' && text[len(text)-1] == '"' {
		return text[2 : len(text)-1]
	}
	return text
}

// cfgError constructs a uniform @cfg diagnostic. Every @cfg diagnostic
// carries a primary span and a one-line message (Rule 6.17).
func cfgError(span lex.Span, filename, msg, hint string) lex.Diagnostic {
	if span.File == "" {
		span.File = filename
	}
	return lex.Diagnostic{Span: span, Message: msg, Hint: hint}
}

// cfgDecorators returns the subset of decorators named `cfg`. Order
// from the AST is preserved.
func cfgDecorators(decs []*ast.Decorator) []*ast.Decorator {
	var out []*ast.Decorator
	for _, d := range decs {
		if d.Name.Name == "cfg" {
			out = append(out, d)
		}
	}
	return out
}

// itemDecorators pulls the Decorators slice off of each item type that
// carries one. Items without decorators (ImplDecl, TraitDecl, TypeDecl)
// return nil; those items are therefore never `@cfg`-gated, matching
// the reference which only attaches `@cfg` to items that accept
// decorators in the grammar.
func itemDecorators(it ast.Item) []*ast.Decorator {
	switch x := it.(type) {
	case *ast.FnDecl:
		return x.Decorators
	case *ast.StructDecl:
		return x.Decorators
	case *ast.EnumDecl:
		return x.Decorators
	case *ast.ConstDecl:
		return x.Decorators
	case *ast.StaticDecl:
		return x.Decorators
	case *ast.UnionDecl:
		return x.Decorators
	case *ast.ExternDecl:
		return itemDecorators(x.Item)
	}
	return nil
}
