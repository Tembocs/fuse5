package parse

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// updateGoldens rewrites the committed `.ast` goldens. Rule 6.2 forbids
// silent golden updates; this flag makes updates explicit.
var updateGoldens = flag.Bool("update", false, "rewrite parse goldens under testdata/")

// TestGolden parses every testdata/*.fuse and compares the rendered AST
// against testdata/*.ast. Running with `-count=3` (as in the wave plan's
// Verify) proves the same parse produces the same text every time
// (Rule 6.2, Rule 7.1).
func TestGolden(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join("testdata", "*.fuse"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no testdata/*.fuse fixtures found")
	}
	for _, in := range entries {
		in := in
		name := strings.TrimSuffix(filepath.Base(in), ".fuse")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatalf("read %s: %v", in, err)
			}
			f, diags := Parse(filepath.Base(in), src)
			got := renderAST(f, diags)

			goldenPath := strings.TrimSuffix(in, ".fuse") + ".ast"
			if *updateGoldens {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v (run with -update to create)", goldenPath, err)
			}
			// Normalize CRLF -> LF so a Windows checkout without
			// .gitattributes does not spuriously fail (same defense as the
			// lex goldens).
			wantNorm := bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
			if string(wantNorm) != got {
				t.Errorf("golden mismatch for %s\n--- want ---\n%s--- got ---\n%s", name, wantNorm, got)
			}
		})
	}
}

// renderAST produces a stable indented printout of a File. The format omits
// byte offsets and filenames so it is insensitive to LF vs CRLF on disk and
// to the testdata path.
func renderAST(f *ast.File, diags []lex.Diagnostic) string {
	var b strings.Builder
	w := &astWriter{b: &b, depth: 0}
	w.writeFile(f)
	for _, d := range diags {
		fmt.Fprintf(&b, "DIAG %d:%d: %s\n",
			d.Span.Start.Line, d.Span.Start.Column, d.Message)
	}
	return b.String()
}

type astWriter struct {
	b     *strings.Builder
	depth int
}

func (w *astWriter) indent() {
	for i := 0; i < w.depth; i++ {
		w.b.WriteString("  ")
	}
}

func (w *astWriter) line(format string, args ...any) {
	w.indent()
	fmt.Fprintf(w.b, format, args...)
	w.b.WriteByte('\n')
}

func (w *astWriter) child(f func()) {
	w.depth++
	f()
	w.depth--
}

func (w *astWriter) writeFile(f *ast.File) {
	w.line("File")
	w.child(func() {
		for _, imp := range f.Imports {
			w.line("Import %s", joinSegs(imp.Path))
		}
		for _, it := range f.Items {
			w.writeItem(it)
		}
	})
}

func (w *astWriter) writeItem(it ast.Item) {
	switch v := it.(type) {
	case *ast.FnDecl:
		flags := ""
		if v.IsConst {
			flags += "const "
		}
		if v.IsExtern {
			flags += "extern "
		}
		w.line("FnDecl %s%s vis=%s", flags, v.Name.Name, visStr(v.Vis))
		w.child(func() {
			for _, d := range v.Decorators {
				w.writeDecorator(d)
			}
			for _, gp := range v.Generics {
				w.line("Generic %s", gp.Name.Name)
			}
			for _, pr := range v.Params {
				w.writeParam(pr)
			}
			if v.Return != nil {
				w.line("Return:")
				w.child(func() { w.writeType(v.Return) })
			}
			if v.Body != nil {
				w.writeBlock(v.Body)
			}
		})
	case *ast.StructDecl:
		w.line("StructDecl %s vis=%s", v.Name.Name, visStr(v.Vis))
		w.child(func() {
			for _, d := range v.Decorators {
				w.writeDecorator(d)
			}
			for _, f := range v.Fields {
				w.line("Field %s", f.Name.Name)
				w.child(func() { w.writeType(f.Type) })
			}
			for i, t := range v.Tuple {
				w.line("TupleField %d", i)
				w.child(func() { w.writeType(t) })
			}
		})
	case *ast.EnumDecl:
		w.line("EnumDecl %s", v.Name.Name)
		w.child(func() {
			for _, d := range v.Decorators {
				w.writeDecorator(d)
			}
			for _, va := range v.Variants {
				w.line("Variant %s", va.Name.Name)
				w.child(func() {
					for _, t := range va.Tuple {
						w.writeType(t)
					}
					for _, f := range va.Fields {
						w.line("Field %s", f.Name.Name)
						w.child(func() { w.writeType(f.Type) })
					}
				})
			}
		})
	case *ast.UnionDecl:
		w.line("UnionDecl %s", v.Name.Name)
		w.child(func() {
			for _, d := range v.Decorators {
				w.writeDecorator(d)
			}
			for _, f := range v.Fields {
				w.line("Field %s", f.Name.Name)
				w.child(func() { w.writeType(f.Type) })
			}
		})
	case *ast.ConstDecl:
		w.line("ConstDecl %s", v.Name.Name)
		w.child(func() {
			w.line("Type:")
			w.child(func() { w.writeType(v.Type) })
			w.line("Value:")
			w.child(func() { w.writeExpr(v.Value) })
		})
	case *ast.StaticDecl:
		w.line("StaticDecl %s extern=%v", v.Name.Name, v.IsExtern)
		w.child(func() {
			for _, d := range v.Decorators {
				w.writeDecorator(d)
			}
			w.line("Type:")
			w.child(func() { w.writeType(v.Type) })
			if v.Value != nil {
				w.line("Value:")
				w.child(func() { w.writeExpr(v.Value) })
			}
		})
	case *ast.TypeDecl:
		w.line("TypeDecl %s", v.Name.Name)
		w.child(func() { w.writeType(v.Target) })
	case *ast.TraitDecl:
		w.line("TraitDecl %s", v.Name.Name)
	case *ast.ImplDecl:
		w.line("ImplDecl")
	case *ast.ExternDecl:
		w.line("ExternDecl")
		w.child(func() { w.writeItem(v.Item) })
	default:
		w.line("%T", it)
	}
}

func (w *astWriter) writeParam(p *ast.Param) {
	w.line("Param %s own=%s", p.Name.Name, ownStr(p.Ownership))
	if p.Type != nil {
		w.child(func() { w.writeType(p.Type) })
	}
}

func (w *astWriter) writeDecorator(d *ast.Decorator) {
	w.line("Decorator @%s", d.Name.Name)
	for _, a := range d.Args {
		if a.Name != nil {
			w.child(func() { w.line("Arg %s =", a.Name.Name); w.child(func() { w.writeExpr(a.Value) }) })
		} else {
			w.child(func() { w.writeExpr(a.Value) })
		}
	}
}

func (w *astWriter) writeType(t ast.Type) {
	switch v := t.(type) {
	case *ast.PathType:
		w.line("PathType %s", joinSegs(v.Segments))
		for _, a := range v.Args {
			w.child(func() { w.writeType(a) })
		}
	case *ast.TupleType:
		w.line("TupleType")
		for _, e := range v.Elements {
			w.child(func() { w.writeType(e) })
		}
	case *ast.ArrayType:
		w.line("ArrayType")
		w.child(func() {
			w.writeType(v.Element)
			w.writeExpr(v.Length)
		})
	case *ast.SliceType:
		w.line("SliceType")
		w.child(func() { w.writeType(v.Element) })
	case *ast.PtrType:
		w.line("PtrType")
		w.child(func() { w.writeType(v.Pointee) })
	case *ast.FnType:
		w.line("FnType")
		for _, p := range v.Params {
			w.child(func() { w.writeType(p) })
		}
		if v.Return != nil {
			w.child(func() { w.writeType(v.Return) })
		}
	case *ast.DynType:
		w.line("DynType")
		for _, tr := range v.Traits {
			w.child(func() { w.writeType(tr) })
		}
	case *ast.ImplType:
		w.line("ImplType")
		w.child(func() { w.writeType(v.Trait) })
	case *ast.UnitType:
		w.line("UnitType")
	default:
		w.line("%T", t)
	}
}

func (w *astWriter) writeBlock(b *ast.BlockExpr) {
	w.line("Block")
	w.child(func() {
		for _, s := range b.Stmts {
			w.writeStmt(s)
		}
		if b.Trailing != nil {
			w.line("Trailing:")
			w.child(func() { w.writeExpr(b.Trailing) })
		}
	})
}

func (w *astWriter) writeStmt(s ast.Stmt) {
	switch v := s.(type) {
	case *ast.LetStmt:
		w.line("Let")
		w.child(func() {
			w.writePat(v.Pattern)
			if v.Type != nil {
				w.writeType(v.Type)
			}
			if v.Value != nil {
				w.writeExpr(v.Value)
			}
		})
	case *ast.VarStmt:
		w.line("Var %s", v.Name.Name)
		w.child(func() {
			if v.Type != nil {
				w.writeType(v.Type)
			}
			w.writeExpr(v.Value)
		})
	case *ast.ReturnStmt:
		w.line("Return")
		if v.Value != nil {
			w.child(func() { w.writeExpr(v.Value) })
		}
	case *ast.BreakStmt:
		w.line("Break")
	case *ast.ContinueStmt:
		w.line("Continue")
	case *ast.ExprStmt:
		w.line("ExprStmt")
		w.child(func() { w.writeExpr(v.Expr) })
	case *ast.ItemStmt:
		w.line("ItemStmt")
		w.child(func() { w.writeItem(v.Item) })
	default:
		w.line("%T", s)
	}
}

func (w *astWriter) writeExpr(e ast.Expr) {
	switch v := e.(type) {
	case *ast.LiteralExpr:
		w.line("Lit %s %s", litKindStr(v.Kind), v.Text)
	case *ast.PathExpr:
		w.line("Path %s", joinSegs(v.Segments))
	case *ast.BinaryExpr:
		w.line("Binary %s", binOpSym(v.Op))
		w.child(func() { w.writeExpr(v.Lhs); w.writeExpr(v.Rhs) })
	case *ast.AssignExpr:
		w.line("Assign %s", assignOpSym(v.Op))
		w.child(func() { w.writeExpr(v.Lhs); w.writeExpr(v.Rhs) })
	case *ast.UnaryExpr:
		w.line("Unary %s", unaryOpSym(v.Op))
		w.child(func() { w.writeExpr(v.Operand) })
	case *ast.CastExpr:
		w.line("Cast")
		w.child(func() { w.writeExpr(v.Expr); w.writeType(v.Type) })
	case *ast.CallExpr:
		w.line("Call")
		w.child(func() {
			w.writeExpr(v.Callee)
			for _, a := range v.Args {
				w.writeExpr(a)
			}
		})
	case *ast.FieldExpr:
		w.line("Field .%s", v.Name.Name)
		w.child(func() { w.writeExpr(v.Receiver) })
	case *ast.OptFieldExpr:
		w.line("OptField ?.%s", v.Name.Name)
		w.child(func() { w.writeExpr(v.Receiver) })
	case *ast.TryExpr:
		w.line("Try ?")
		w.child(func() { w.writeExpr(v.Receiver) })
	case *ast.IndexExpr:
		w.line("Index")
		w.child(func() { w.writeExpr(v.Receiver); w.writeExpr(v.Index) })
	case *ast.IndexRangeExpr:
		w.line("IndexRange inclusive=%v", v.Inclusive)
		w.child(func() {
			w.writeExpr(v.Receiver)
			if v.Low != nil {
				w.writeExpr(v.Low)
			} else {
				w.line("(open)")
			}
			if v.High != nil {
				w.writeExpr(v.High)
			} else {
				w.line("(open)")
			}
		})
	case *ast.BlockExpr:
		w.writeBlock(v)
	case *ast.IfExpr:
		w.line("If")
		w.child(func() {
			w.writeExpr(v.Cond)
			w.writeBlock(v.Then)
			if v.Else != nil {
				w.writeExpr(v.Else)
			}
		})
	case *ast.MatchExpr:
		w.line("Match")
		w.child(func() {
			w.writeExpr(v.Scrutinee)
			for _, arm := range v.Arms {
				w.line("Arm")
				w.child(func() {
					w.writePat(arm.Pattern)
					if arm.Guard != nil {
						w.writeExpr(arm.Guard)
					}
					w.writeBlock(arm.Body)
				})
			}
		})
	case *ast.LoopExpr:
		w.line("Loop")
		w.child(func() { w.writeBlock(v.Body) })
	case *ast.WhileExpr:
		w.line("While")
		w.child(func() { w.writeExpr(v.Cond); w.writeBlock(v.Body) })
	case *ast.ForExpr:
		w.line("For")
		w.child(func() { w.writePat(v.Pattern); w.writeExpr(v.Iter); w.writeBlock(v.Body) })
	case *ast.TupleExpr:
		w.line("Tuple")
		for _, el := range v.Elements {
			w.child(func() { w.writeExpr(el) })
		}
	case *ast.StructLitExpr:
		w.line("StructLit %s", joinSegs(v.Path.Segments))
		w.child(func() {
			for _, f := range v.Fields {
				if f.Shorthand {
					w.line("FieldShort %s", f.Name.Name)
				} else {
					w.line("Field %s", f.Name.Name)
					w.child(func() { w.writeExpr(f.Value) })
				}
			}
			if v.Base != nil {
				w.line("Base:")
				w.child(func() { w.writeExpr(v.Base) })
			}
		})
	case *ast.ClosureExpr:
		w.line("Closure move=%v", v.IsMove)
	case *ast.SpawnExpr:
		w.line("Spawn")
	case *ast.UnsafeExpr:
		w.line("Unsafe")
		w.child(func() { w.writeBlock(v.Body) })
	case *ast.ParenExpr:
		w.line("Paren")
		w.child(func() { w.writeExpr(v.Inner) })
	default:
		w.line("%T", e)
	}
}

func (w *astWriter) writePat(p ast.Pat) {
	switch v := p.(type) {
	case *ast.LiteralPat:
		w.line("LitPat %s", v.Value.Text)
	case *ast.WildcardPat:
		w.line("Wildcard")
	case *ast.BindPat:
		w.line("Bind %s", v.Name.Name)
	case *ast.CtorPat:
		w.line("Ctor %s rest=%v", joinSegs(v.Path), v.HasRest)
		w.child(func() {
			for _, pp := range v.Tuple {
				w.writePat(pp)
			}
			for _, fp := range v.Struct {
				w.line("Field %s", fp.Name.Name)
				if fp.Pattern != nil {
					w.child(func() { w.writePat(fp.Pattern) })
				}
			}
		})
	case *ast.TuplePat:
		w.line("TuplePat")
		w.child(func() {
			for _, pp := range v.Elements {
				w.writePat(pp)
			}
		})
	case *ast.OrPat:
		w.line("OrPat")
		w.child(func() {
			for _, pp := range v.Alts {
				w.writePat(pp)
			}
		})
	case *ast.RangePat:
		w.line("RangePat inclusive=%v", v.Inclusive)
		w.child(func() { w.writeExpr(v.Lo); w.writeExpr(v.Hi) })
	case *ast.AtPat:
		w.line("At %s", v.Name.Name)
		w.child(func() { w.writePat(v.Pattern) })
	default:
		w.line("%T", p)
	}
}

func joinSegs(segs []ast.Ident) string {
	names := make([]string, len(segs))
	for i, s := range segs {
		names[i] = s.Name
	}
	return strings.Join(names, ".")
}

func visStr(v ast.Visibility) string {
	switch v {
	case ast.VisPrivate:
		return "private"
	case ast.VisPub:
		return "pub"
	case ast.VisPubMod:
		return "pub(mod)"
	case ast.VisPubPkg:
		return "pub(pkg)"
	}
	return "?"
}

func ownStr(o ast.Ownership) string {
	switch o {
	case ast.OwnNone:
		return "none"
	case ast.OwnRef:
		return "ref"
	case ast.OwnMutref:
		return "mutref"
	case ast.OwnOwned:
		return "owned"
	}
	return "?"
}

func litKindStr(k ast.LiteralKind) string {
	switch k {
	case ast.LitInt:
		return "INT"
	case ast.LitFloat:
		return "FLOAT"
	case ast.LitString:
		return "STRING"
	case ast.LitRawString:
		return "RAWSTRING"
	case ast.LitCString:
		return "CSTRING"
	case ast.LitChar:
		return "CHAR"
	case ast.LitBool:
		return "BOOL"
	case ast.LitNone:
		return "NONE"
	}
	return "?"
}
