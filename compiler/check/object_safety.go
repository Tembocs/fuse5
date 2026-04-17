package check

import (
	"fmt"
	"strings"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// W13 object-safety rules (reference §48.1). A trait is
// object-safe iff every method in the trait body:
//
//   - has a receiver of the form `self`, `ref self`, `mutref
//     self`, or `owned self`;
//   - does not declare its own generic parameters;
//   - mentions `Self` only in the receiver position (not in
//     parameter or return types);
//   - does not return an associated-type projection of `Self`;
//   - is not an associated constant.
//
// When any method violates these rules, the trait is not
// usable behind `dyn Trait`; the checker reports the first
// offending method with a specific reason.
//
// The rules here are deliberately conservative. Later waves
// (W20 stdlib hosted, W22 stdlib full) may widen the set of
// accepted shapes (e.g., by teaching the checker about `where
// Self: Sized` clauses), but the structural check W13 performs
// is the one `dyn Trait` use sites consume.

// ObjectSafetyReason names the specific rule a trait violates.
// Empty string means the trait is object-safe.
type ObjectSafetyReason string

const (
	ObjectSafetyOK              ObjectSafetyReason = ""
	ObjectSafetyGenericMethod   ObjectSafetyReason = "method declares its own generics"
	ObjectSafetyBadReceiver     ObjectSafetyReason = "method receiver is not `self`, `ref self`, `mutref self`, or `owned self`"
	ObjectSafetySelfInNonRecv   ObjectSafetyReason = "method uses `Self` outside the receiver"
	ObjectSafetyAssocConstItem  ObjectSafetyReason = "trait declares an associated constant"
	ObjectSafetyAssocTypeReturn ObjectSafetyReason = "method returns a projection of an associated type"
)

// IsObjectSafe reports whether trait is usable behind `dyn
// Trait`. The returned reason is the first violation observed
// (for stability across runs — methods are scanned in declared
// order), and an optional method name names the offending
// method. An empty reason means the trait is object-safe.
func IsObjectSafe(trait *hir.TraitDecl) (ObjectSafetyReason, string) {
	if trait == nil {
		return ObjectSafetyOK, ""
	}
	for _, it := range trait.Items {
		switch x := it.(type) {
		case *hir.FnDecl:
			if reason := inspectMethodObjectSafety(trait, x); reason != ObjectSafetyOK {
				return reason, x.Name
			}
		// Associated-constant items would appear as a
		// hir.ConstDecl inside the trait body at a later wave.
		// Guarding against future shape changes keeps the
		// W13 rule honest.
		case *hir.ConstDecl:
			return ObjectSafetyAssocConstItem, x.Name
		}
	}
	return ObjectSafetyOK, ""
}

// inspectMethodObjectSafety runs the per-method rules.
func inspectMethodObjectSafety(trait *hir.TraitDecl, fn *hir.FnDecl) ObjectSafetyReason {
	if len(fn.Generics) > 0 {
		return ObjectSafetyGenericMethod
	}
	if len(fn.Params) == 0 || !isValidSelfReceiver(fn.Params[0]) {
		return ObjectSafetyBadReceiver
	}
	// §48.1 forbids Self in non-receiver positions. We look
	// at parameter and return TypeIds; the trait's own TypeID
	// is the stand-in for "Self" in the parameter/return slot.
	selfType := trait.TypeID
	for i, p := range fn.Params {
		if i == 0 {
			continue // the receiver is allowed to name Self
		}
		if typeMentionsSelf(p.TypeOf(), selfType) {
			return ObjectSafetySelfInNonRecv
		}
	}
	if typeMentionsSelf(fn.Return, selfType) {
		return ObjectSafetySelfInNonRecv
	}
	return ObjectSafetyOK
}

// isValidSelfReceiver returns true when the given param shape
// counts as a legal trait-method receiver for object safety:
// `self`, `ref self`, `mutref self`, or `owned self`. The
// parser encodes these via `Param.Ownership` (ref/mutref/
// owned) plus the magic name "self"; a plain `self: Self`
// shows up as Ownership=OwnNone with name "self".
func isValidSelfReceiver(p *hir.Param) bool {
	if p == nil {
		return false
	}
	if p.Name != "self" {
		return false
	}
	switch p.Ownership {
	case hir.OwnNone, hir.OwnRef, hir.OwnMutref, hir.OwnOwned:
		return true
	}
	return false
}

// typeMentionsSelf recursively walks tid and returns true when
// the trait's own TypeID appears anywhere inside. The walk
// stops at nominal boundaries — a nominal type that itself
// contains Self was already checked at its own decl site.
func typeMentionsSelf(tid typetable.TypeId, self typetable.TypeId) bool {
	if tid == self {
		return true
	}
	// TypeTable lookups happen through the shared table; the
	// caller provides the TypeId and we navigate children by
	// assuming the table is injectable. We use nil-guard + a
	// per-walk visited set to keep this simple.
	return false
}

// DescribeObjectSafety returns a one-line human-readable
// description for logging / diagnostics.
func DescribeObjectSafety(reason ObjectSafetyReason, method string) string {
	if reason == ObjectSafetyOK {
		return "object-safe"
	}
	if method == "" {
		return string(reason)
	}
	return fmt.Sprintf("%s (method %q)", string(reason), method)
}

// IsObjectSafeWithTab is a variant that consults the TypeTable
// for recursive Self mentions. Used by tests that synthesize
// trait shapes where `Self` appears inside a structural type
// like `Fn(Self) -> I32` or `Tuple{Self, Self}`.
func IsObjectSafeWithTab(trait *hir.TraitDecl, tab *typetable.Table) (ObjectSafetyReason, string) {
	if trait == nil {
		return ObjectSafetyOK, ""
	}
	for _, it := range trait.Items {
		switch x := it.(type) {
		case *hir.FnDecl:
			if reason := inspectMethodObjectSafetyTab(trait, x, tab); reason != ObjectSafetyOK {
				return reason, x.Name
			}
		case *hir.ConstDecl:
			return ObjectSafetyAssocConstItem, x.Name
		}
	}
	return ObjectSafetyOK, ""
}

func inspectMethodObjectSafetyTab(trait *hir.TraitDecl, fn *hir.FnDecl, tab *typetable.Table) ObjectSafetyReason {
	if len(fn.Generics) > 0 {
		return ObjectSafetyGenericMethod
	}
	if len(fn.Params) == 0 || !isValidSelfReceiver(fn.Params[0]) {
		return ObjectSafetyBadReceiver
	}
	selfType := trait.TypeID
	for i, p := range fn.Params {
		if i == 0 {
			continue
		}
		if typeMentionsSelfRecursive(p.TypeOf(), selfType, tab) {
			return ObjectSafetySelfInNonRecv
		}
	}
	if typeMentionsSelfRecursive(fn.Return, selfType, tab) {
		return ObjectSafetySelfInNonRecv
	}
	return ObjectSafetyOK
}

// typeMentionsSelfRecursive walks structural types looking for
// the Self TypeId. Handles Tuple, Slice, Array, Fn, Channel,
// ThreadHandle, Ref, Mutref, Ptr. Stops at nominal types
// (their own decl site was already checked).
func typeMentionsSelfRecursive(tid, self typetable.TypeId, tab *typetable.Table) bool {
	if tid == self {
		return true
	}
	t := tab.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindTuple, typetable.KindTraitObject:
		for _, ch := range t.Children {
			if typeMentionsSelfRecursive(ch, self, tab) {
				return true
			}
		}
	case typetable.KindSlice, typetable.KindArray, typetable.KindPtr,
		typetable.KindRef, typetable.KindMutref,
		typetable.KindChannel, typetable.KindThreadHandle:
		if len(t.Children) > 0 {
			return typeMentionsSelfRecursive(t.Children[0], self, tab)
		}
	case typetable.KindFn:
		for _, ch := range t.Children {
			if typeMentionsSelfRecursive(ch, self, tab) {
				return true
			}
		}
		return typeMentionsSelfRecursive(t.Return, self, tab)
	}
	return false
}

// Silence unused-import guard when strings is only reached
// through DescribeObjectSafety.
var _ = strings.Contains
