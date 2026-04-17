package resolve

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/ast"
	"github.com/Tembocs/fuse5/compiler/lex"
)

// indexModule registers every top-level item in items into the module's
// scope, returning diagnostics for any duplicate definition (Rule 6.9).
// Enum variants are hoisted into the module scope per reference §18.6;
// conflicts between enums and variants or between variants from
// different enums are reported here.
//
// The scope must already be attached to the Module. Callers pass items
// after @cfg filtering so removed items never participate in the index
// (reference §50.1).
func (r *resolver) indexModule(m *Module, items []ast.Item) []lex.Diagnostic {
	var diags []lex.Diagnostic
	for _, it := range items {
		switch x := it.(type) {
		case *ast.FnDecl:
			diags = append(diags, r.indexOne(m, x.Name, x.Vis, symFnKind(x), x)...)
		case *ast.StructDecl:
			diags = append(diags, r.indexOne(m, x.Name, x.Vis, SymStruct, x)...)
		case *ast.EnumDecl:
			enumID, ds := r.indexOneID(m, x.Name, x.Vis, SymEnum, x)
			diags = append(diags, ds...)
			if enumID != NoSymbol {
				diags = append(diags, r.indexEnumVariants(m, x, enumID)...)
			}
		case *ast.TraitDecl:
			diags = append(diags, r.indexOne(m, x.Name, x.Vis, SymTrait, x)...)
		case *ast.ConstDecl:
			diags = append(diags, r.indexOne(m, x.Name, x.Vis, SymConst, x)...)
		case *ast.StaticDecl:
			diags = append(diags, r.indexOne(m, x.Name, x.Vis, SymStatic, x)...)
		case *ast.TypeDecl:
			diags = append(diags, r.indexOne(m, x.Name, x.Vis, SymTypeAlias, x)...)
		case *ast.UnionDecl:
			diags = append(diags, r.indexOne(m, x.Name, x.Vis, SymUnion, x)...)
		case *ast.ExternDecl:
			diags = append(diags, r.indexExtern(m, x)...)
		case *ast.ImplDecl:
			// Impls don't introduce names in the module scope; their
			// items are indexed into a per-impl scope when method
			// resolution lands in W06.
		}
	}
	return diags
}

// indexOne registers one named item and returns any duplicate
// diagnostic. Convenience wrapper around indexOneID for callers that
// don't need the assigned ID.
func (r *resolver) indexOne(m *Module, name ast.Ident, vis ast.Visibility, kind SymKind, node ast.Node) []lex.Diagnostic {
	_, diags := r.indexOneID(m, name, vis, kind, node)
	return diags
}

// indexOneID registers one named item, returns the assigned SymbolID
// (NoSymbol if the insertion failed due to a duplicate name), and any
// diagnostic produced.
func (r *resolver) indexOneID(m *Module, name ast.Ident, vis ast.Visibility, kind SymKind, node ast.Node) (SymbolID, []lex.Diagnostic) {
	if name.Name == "" {
		return NoSymbol, nil
	}
	if prior := m.Scope.LookupLocal(name.Name); prior != NoSymbol {
		return NoSymbol, []lex.Diagnostic{dupNameDiag(name, prior, r.symbols, m.Path)}
	}
	id := r.symbols.Add(Symbol{
		Kind:   kind,
		Name:   name.Name,
		Module: m.Path,
		Vis:    vis,
		Span:   name.Span,
		Node:   node,
	})
	m.Scope.Insert(name.Name, id)
	return id, nil
}

// indexEnumVariants hoists each variant of e into the module's scope as
// its own symbol (reference §11.6, §18.6). Variants inherit the enum's
// visibility (reference §53.1 — a narrower visibility may not shadow a
// wider one, and variants cannot be marked independently). Conflicts
// between variants in the same module are reported.
func (r *resolver) indexEnumVariants(m *Module, e *ast.EnumDecl, enumID SymbolID) []lex.Diagnostic {
	var diags []lex.Diagnostic
	for _, v := range e.Variants {
		if prior := m.Scope.LookupLocal(v.Name.Name); prior != NoSymbol {
			diags = append(diags, dupNameDiag(v.Name, prior, r.symbols, m.Path))
			continue
		}
		id := r.symbols.Add(Symbol{
			Kind:   SymEnumVariant,
			Name:   v.Name.Name,
			Module: m.Path,
			Vis:    e.Vis,
			Span:   v.Name.Span,
			Parent: enumID,
			Node:   v,
		})
		m.Scope.Insert(v.Name.Name, id)
	}
	return diags
}

// indexExtern handles `extern fn` / `extern static` declarations.
func (r *resolver) indexExtern(m *Module, ed *ast.ExternDecl) []lex.Diagnostic {
	switch inner := ed.Item.(type) {
	case *ast.FnDecl:
		return r.indexOne(m, inner.Name, inner.Vis, SymExternFn, inner)
	case *ast.StaticDecl:
		return r.indexOne(m, inner.Name, inner.Vis, SymExternStatic, inner)
	}
	return nil
}

// symFnKind distinguishes plain fn items from extern fn items based on
// the FnDecl.IsExtern flag. Called by indexModule for bare FnDecls; the
// ExternDecl wrapper routes through indexExtern instead.
func symFnKind(fn *ast.FnDecl) SymKind {
	if fn.IsExtern {
		return SymExternFn
	}
	return SymFn
}

// dupNameDiag constructs the standard duplicate-item diagnostic for a
// name that already exists in the module scope. The hint names the
// prior definition's span so the reader can find both occurrences
// (Rule 6.17).
func dupNameDiag(name ast.Ident, prior SymbolID, t *SymbolTable, modulePath string) lex.Diagnostic {
	priorSpan := lex.Span{}
	kind := "item"
	if p := t.Get(prior); p != nil {
		priorSpan = p.Span
		kind = p.Kind.String()
	}
	msg := fmt.Sprintf("duplicate item %q in module %q", name.Name, modulePath)
	hint := fmt.Sprintf("prior %s definition at %s", kind, priorSpan.String())
	return lex.Diagnostic{Span: name.Span, Message: msg, Hint: hint}
}
