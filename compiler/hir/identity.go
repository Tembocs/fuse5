package hir

import (
	"fmt"
	"strings"
)

// NodeID format: `module_path::item_name::local_path`.
//
//   - module_path is the dotted module path (empty for the crate root).
//   - item_name is the declared name of the enclosing item (function,
//     struct, etc.). For impl methods, it is `impl_target.method`.
//   - local_path is a slash-delimited segment list describing the
//     node's position within the item body; its exact shape depends
//     on the enclosing node type.
//
// The local_path is *structural*, not allocation-ordered. Changing
// unrelated functions does not shift other items' NodeIDs. Adding a
// statement *does* shift indices for later statements in the same
// block — this is the desired behavior, because the edit has a direct
// effect on the identities it shifts, and the fingerprint diff will
// isolate the cascade (W04-P05-T02).
//
// The identity.go helpers are the single source of truth for NodeID
// construction. Builders and the AST bridge route through them so the
// format stays consistent.

// IdBuilder composes a NodeID incrementally. A builder anchored at an
// item yields NodeIDs for every child node in that item without the
// caller having to thread the item's prefix manually.
type IdBuilder struct {
	module string
	item   string
	path   []string
}

// NewIdBuilder anchors a builder at (module, item). For an impl method
// the item string should be `TypeName.method_name` (the Bridge chooses
// this spelling).
func NewIdBuilder(module, item string) *IdBuilder {
	return &IdBuilder{module: module, item: item}
}

// Anchor returns the module/item pair currently in effect.
func (b *IdBuilder) Anchor() (string, string) { return b.module, b.item }

// Push descends into a named sub-region (e.g. `body`, `then`,
// `arm[3]`). It returns the current builder for chaining.
func (b *IdBuilder) Push(seg string) *IdBuilder {
	b.path = append(b.path, seg)
	return b
}

// Pop undoes the most recent Push. Panics when nothing has been
// pushed — the mismatch is always a builder bug.
func (b *IdBuilder) Pop() *IdBuilder {
	if len(b.path) == 0 {
		panic("hir.IdBuilder: Pop on empty path")
	}
	b.path = b.path[:len(b.path)-1]
	return b
}

// Checkpoint captures the current path depth. Callers use Checkpoint
// + Restore in bridge pass recursion to guarantee stack balance even
// if a sub-recursion panics (invariants.go's recover observes the
// restored state).
func (b *IdBuilder) Checkpoint() int { return len(b.path) }

// Restore trims path back to the given depth. Must be ≤ current depth.
func (b *IdBuilder) Restore(depth int) {
	if depth > len(b.path) {
		panic(fmt.Sprintf("hir.IdBuilder: Restore(%d) beyond len %d", depth, len(b.path)))
	}
	b.path = b.path[:depth]
}

// Here returns the NodeID naming the current position.
func (b *IdBuilder) Here() NodeID {
	return NodeID(formatNodeID(b.module, b.item, b.path))
}

// Child returns the NodeID for a single-step descent without
// permanently pushing. Useful for leaf nodes where a Push/Pop pair
// would be noise.
func (b *IdBuilder) Child(seg string) NodeID {
	path := make([]string, len(b.path)+1)
	copy(path, b.path)
	path[len(b.path)] = seg
	return NodeID(formatNodeID(b.module, b.item, path))
}

// formatNodeID is the single spelling of the NodeID format. Changing
// this format is a language-wide change because every fingerprint,
// cache key, and LSP diagnostic anchor depends on it.
func formatNodeID(module, item string, path []string) string {
	var sb strings.Builder
	sb.WriteString(module)
	sb.WriteString("::")
	sb.WriteString(item)
	if len(path) > 0 {
		sb.WriteString("::")
		sb.WriteString(strings.Join(path, "/"))
	}
	return sb.String()
}

// ItemID returns the NodeID naming a bare item (no local path). Used
// by the bridge when registering top-level items.
func ItemID(module, item string) NodeID {
	return NodeID(formatNodeID(module, item, nil))
}
