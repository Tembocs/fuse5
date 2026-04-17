package hir

import (
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// NodeID is the stable identity of an HIR node. Values are derived
// from (module path, item path, local path) rather than allocation
// order, so an unrelated source edit does not shift identities for
// unrelated functions (W04-P05-T02). See identity.go for the format.
type NodeID string

// EmptyNodeID is the zero value. It is legal on synthetic nodes that
// do not participate in incremental caching; a non-synthetic node
// with EmptyNodeID is a builder bug.
const EmptyNodeID NodeID = ""

// Node is the root interface every HIR node implements.
type Node interface {
	NodeSpan() lex.Span
	NodeHirID() NodeID
	hirNode()
}

// Typed narrows the interface to nodes that carry a TypeId. Every
// expression and pattern implements Typed; items (FnDecl, StructDecl)
// do not because their "type" is the nominal TypeId that the bridge
// registers separately.
type Typed interface {
	Node
	TypeOf() typetable.TypeId
}

// Base is embedded in every concrete HIR node and provides the Node
// interface boilerplate. Concrete types add marker methods
// (exprNode(), patNode(), etc.) to narrow the set of acceptable
// substitutions at compile time.
type Base struct {
	ID   NodeID
	Span lex.Span
}

// NodeSpan satisfies Node.
func (b *Base) NodeSpan() lex.Span { return b.Span }

// NodeHirID satisfies Node.
func (b *Base) NodeHirID() NodeID { return b.ID }

// hirNode is the sealing marker for the Node interface.
func (b *Base) hirNode() {}

// TypedBase is embedded in every HIR node that carries a TypeId. It
// adds the Type field on top of Base. Using a narrower embed avoids
// carrying a zero TypeId on items that have nominal identity through
// other channels (e.g. FnDecl records its nominal TypeId on the
// Program.ItemTypes map).
type TypedBase struct {
	Base
	Type typetable.TypeId
}

// TypeOf satisfies Typed.
func (b *TypedBase) TypeOf() typetable.TypeId { return b.Type }

// Item marker — declarations at module or impl scope.
type Item interface {
	Node
	itemNode()
}

// Expr marker — everything that produces a value.
type Expr interface {
	Typed
	exprNode()
}

// Stmt marker — everything that appears in statement position.
type Stmt interface {
	Node
	stmtNode()
}

// Pat marker — pattern nodes. Always structured; reference §7.9 and
// L007 forbid text-form patterns at HIR.
type Pat interface {
	Typed
	patNode()
}
