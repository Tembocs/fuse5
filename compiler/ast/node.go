// Package ast owns the abstract syntax tree for Fuse source files
// (reference Appendix C).
//
// The AST is syntax-only. It holds no resolved symbols, types, or pass
// metadata — Rule 3.2 requires `ast.*`, `hir.*`, and `mir.*` to be disjoint
// type families. W03 (Resolution) and W04 (HIR+TypeTable) build their own
// trees from this one.
//
// Every concrete node carries a `Span lex.Span` and implements Node via the
// embedded `NodeBase`. Marker interfaces (Item, Expr, Stmt, Pat, Type) let
// the parser and downstream passes restrict substitution at compile time.
package ast

import "github.com/Tembocs/fuse5/compiler/lex"

// Node is the root interface every AST node implements. The Span method
// returns the source range the node covers (Rule 7.4: stable, byte-addressed).
type Node interface {
	NodeSpan() lex.Span
	astNode()
}

// Item is the marker interface for top-level and impl/trait-member items
// (reference grammar `item_decl`, `trait_item`, `impl_item`).
type Item interface {
	Node
	itemNode()
}

// Expr is the marker interface for expressions (reference grammar `expr`).
type Expr interface {
	Node
	exprNode()
}

// Stmt is the marker interface for statements (reference grammar `stmt`).
type Stmt interface {
	Node
	stmtNode()
}

// Pat is the marker interface for patterns (reference grammar `pattern`).
type Pat interface {
	Node
	patNode()
}

// Type is the marker interface for type expressions (reference grammar
// `type_expr`).
type Type interface {
	Node
	typeNode()
}

// NodeBase is embedded in every concrete node to satisfy Node without
// repeating boilerplate. Concrete structs add their marker method. The Span
// field is set by the parser as each node is built; downstream passes read it
// via the promoted NodeSpan method.
type NodeBase struct {
	Span lex.Span
}

func (b NodeBase) NodeSpan() lex.Span { return b.Span }
func (b NodeBase) astNode()           {}

// File is a parsed Fuse source file (reference grammar `file`). It owns the
// original filename and the list of imports + items in source order.
type File struct {
	NodeBase
	Filename string
	Imports  []*Import
	Items    []Item
}

// Import is a single `import path [as IDENT];` declaration.
type Import struct {
	NodeBase
	Path  []Ident
	Alias *Ident // nil when no `as` clause
}

// Ident is an identifier occurrence: the text and its source span. Idents are
// plain nodes — the resolver is what turns them into symbols.
type Ident struct {
	Span lex.Span
	Name string
}

// NodeSpan lets Ident satisfy the Node interface without embedding NodeBase
// (a simpler struct layout keeps identifier slices compact).
func (i Ident) NodeSpan() lex.Span { return i.Span }
func (i Ident) astNode()           {}
