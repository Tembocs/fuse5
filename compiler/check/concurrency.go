package check

import (
	"fmt"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// The W07 concurrency checker enforces reference §47.1 marker
// traits, §17.6 channel typing, §39.1 spawn/ThreadHandle typing,
// and §17.6 @rank lock ordering. Runtime lowering for `spawn` and
// channel operations is deferred to W16; this wave is checker-side
// only.
//
// The three marker traits are intrinsic — the user never declares
// them; every Fuse program behaves as if they were imported from
// the root. A TypeId "is Send" if it either (a) is a primitive or
// Unit, (b) is a reference `&T` where T is Send (but `&T` itself
// is not Send — borrowed references cross thread boundaries only
// through Shared[T]), (c) is a tuple/struct/enum whose component
// types are all Send, with no negative impl. Symmetric rules apply
// to Sync and Copy.

// MarkerTrait enumerates the three intrinsic marker traits.
type MarkerTrait int

const (
	MarkerSend MarkerTrait = iota
	MarkerSync
	MarkerCopy
)

// String returns the human-readable name used in diagnostics.
func (m MarkerTrait) String() string {
	switch m {
	case MarkerSend:
		return "Send"
	case MarkerSync:
		return "Sync"
	case MarkerCopy:
		return "Copy"
	}
	return "unknown"
}

// concurrencyContext is the per-checker state for marker-trait
// resolution. It sits alongside the main checker struct and is
// initialized lazily on first use so Check() callers that don't
// touch concurrency pay no overhead.
type concurrencyContext struct {
	// negativeImpls records explicit `impl !Trait for Type { }`
	// registrations. The key is (MarkerTrait, TypeId). Populated
	// by the bridge when a negative-impl syntax node arrives.
	negativeImpls map[negativeKey]bool

	// positiveImpls records explicit `impl Trait for Type { }` of
	// a marker trait. Normally auto-impl covers this, but users can
	// still write one — and the checker accepts it without
	// duplication.
	positiveImpls map[negativeKey]bool
}

type negativeKey struct {
	marker MarkerTrait
	target typetable.TypeId
}

// cc lazily initializes the concurrency context.
func (c *checker) cc() *concurrencyContext {
	if c.concur == nil {
		c.concur = &concurrencyContext{
			negativeImpls: map[negativeKey]bool{},
			positiveImpls: map[negativeKey]bool{},
		}
	}
	return c.concur
}

// checkConcurrency walks every module's items and checks every body
// for concurrency-relevant constructs. Called from Check after
// checkBodies so body typing already assigned concrete TypeIds.
func (c *checker) checkConcurrency() {
	for _, modPath := range c.prog.Order {
		m := c.prog.Modules[modPath]
		for _, it := range m.Items {
			c.checkConcurrencyItem(modPath, it)
		}
	}
}

func (c *checker) checkConcurrencyItem(modPath string, it hir.Item) {
	switch x := it.(type) {
	case *hir.FnDecl:
		if x.Body != nil {
			c.checkConcurrencyBlock(modPath, x.Body)
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			c.checkConcurrencyItem(modPath, sub)
		}
		c.checkRankDecorators(x)
	case *hir.TraitDecl:
		for _, sub := range x.Items {
			c.checkConcurrencyItem(modPath, sub)
		}
	}
}

// checkConcurrencyBlock recursively visits expressions inside a
// block, routing SpawnExpr and call-site patterns through their
// dedicated checks.
func (c *checker) checkConcurrencyBlock(modPath string, blk *hir.Block) {
	if blk == nil {
		return
	}
	for _, s := range blk.Stmts {
		c.checkConcurrencyStmt(modPath, s)
	}
	if blk.Trailing != nil {
		c.checkConcurrencyExpr(modPath, blk.Trailing)
	}
}

func (c *checker) checkConcurrencyStmt(modPath string, s hir.Stmt) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.Value != nil {
			c.checkConcurrencyExpr(modPath, x.Value)
		}
	case *hir.VarStmt:
		if x.Value != nil {
			c.checkConcurrencyExpr(modPath, x.Value)
		}
	case *hir.ReturnStmt:
		if x.Value != nil {
			c.checkConcurrencyExpr(modPath, x.Value)
		}
	case *hir.ExprStmt:
		if x.Expr != nil {
			c.checkConcurrencyExpr(modPath, x.Expr)
		}
	}
}

func (c *checker) checkConcurrencyExpr(modPath string, e hir.Expr) {
	switch x := e.(type) {
	case *hir.SpawnExpr:
		c.checkSpawn(modPath, x)
	case *hir.CallExpr:
		c.checkChannelMethodCall(modPath, x)
		c.checkConcurrencyExpr(modPath, x.Callee)
		for _, a := range x.Args {
			c.checkConcurrencyExpr(modPath, a)
		}
	case *hir.BinaryExpr:
		c.checkConcurrencyExpr(modPath, x.Lhs)
		c.checkConcurrencyExpr(modPath, x.Rhs)
	case *hir.Block:
		c.checkConcurrencyBlock(modPath, x)
	case *hir.IfExpr:
		c.checkConcurrencyExpr(modPath, x.Cond)
		c.checkConcurrencyBlock(modPath, x.Then)
		if x.Else != nil {
			c.checkConcurrencyExpr(modPath, x.Else)
		}
	}
}

// --- Marker-trait predicates ----------------------------------------

// IsSend reports whether tid satisfies the Send marker trait at
// W07. Rules:
//
//   - Every primitive (Bool, Int*, Uint*, Float*, Char, Unit,
//     Never, String, CStr) is Send.
//   - A tuple is Send iff every element is Send.
//   - A nominal struct/enum is Send iff it has no negative impl
//     AND either it has a positive impl or every field (approx)
//     is Send.
//   - Borrow types `&T` and `&mut T` are NOT Send. They cross
//     thread boundaries only through Shared[T].
//   - Ptr, Chan, ThreadHandle are Send — they wrap opaque handles.
//   - Trait objects and closures default to not-Send at W07
//     (conservative; widening to Send closures arrives in W12).
func (c *checker) IsSend(tid typetable.TypeId) bool {
	return c.impliesMarker(MarkerSend, tid)
}

// IsSync reports whether tid satisfies the Sync marker trait. At
// W07 Sync has the same auto-impl shape as Send for the surface
// types we handle; reference §47.1's distinction (Sync = "&T is
// Send") is enforced when borrows land structurally.
func (c *checker) IsSync(tid typetable.TypeId) bool {
	return c.impliesMarker(MarkerSync, tid)
}

// IsCopy reports whether tid satisfies the Copy marker trait. All
// primitives are Copy; Ptr is Copy; references are NOT Copy (they
// are Move); aggregates are Copy iff all fields are Copy.
func (c *checker) IsCopy(tid typetable.TypeId) bool {
	return c.impliesMarker(MarkerCopy, tid)
}

// MarkPositiveImpl records `impl Marker for Type { }`.
func (c *checker) MarkPositiveImpl(marker MarkerTrait, tid typetable.TypeId) {
	c.cc().positiveImpls[negativeKey{marker, tid}] = true
}

// MarkNegativeImpl records `impl !Marker for Type { }`. Once
// registered, the auto-impl rules never apply to tid + marker.
func (c *checker) MarkNegativeImpl(marker MarkerTrait, tid typetable.TypeId) {
	c.cc().negativeImpls[negativeKey{marker, tid}] = true
}

// impliesMarker is the core predicate implementing the auto-impl
// rules. Checking happens lazily and recursively over component
// types; negative impls short-circuit at every level.
func (c *checker) impliesMarker(marker MarkerTrait, tid typetable.TypeId) bool {
	if c.cc().negativeImpls[negativeKey{marker, tid}] {
		return false
	}
	if c.cc().positiveImpls[negativeKey{marker, tid}] {
		return true
	}
	t := c.tab.Get(tid)
	if t == nil {
		return false
	}
	switch t.Kind {
	case typetable.KindBool, typetable.KindChar,
		typetable.KindI8, typetable.KindI16, typetable.KindI32,
		typetable.KindI64, typetable.KindISize,
		typetable.KindU8, typetable.KindU16, typetable.KindU32,
		typetable.KindU64, typetable.KindUSize,
		typetable.KindF32, typetable.KindF64,
		typetable.KindUnit, typetable.KindNever,
		typetable.KindString, typetable.KindCStr:
		return true
	case typetable.KindPtr, typetable.KindChannel, typetable.KindThreadHandle:
		return true
	case typetable.KindRef, typetable.KindMutref:
		// Borrows are explicitly not Send. Sync depends on the
		// inner type. Copy is not — borrows must not be silently
		// duplicated.
		if marker == MarkerSend || marker == MarkerCopy {
			return false
		}
		return len(t.Children) > 0 && c.impliesMarker(marker, t.Children[0])
	case typetable.KindTuple:
		for _, el := range t.Children {
			if !c.impliesMarker(marker, el) {
				return false
			}
		}
		return true
	case typetable.KindArray, typetable.KindSlice:
		if len(t.Children) > 0 {
			return c.impliesMarker(marker, t.Children[0])
		}
		return false
	case typetable.KindFn:
		// Fn pointers are Send/Sync/Copy at W07 — they carry no
		// environment. Closures are distinct (KindTraitObject
		// when dyn-erased; otherwise the bridge gives them a Fn
		// TypeId until W12 lands proper closure types).
		return true
	case typetable.KindStruct, typetable.KindEnum, typetable.KindUnion:
		// Without a full field model at W07 we consult the HIR
		// decl for field types and recurse. If we can't find one,
		// the conservative default is yes for Send (primitive-
		// like), which matches the intent for simple data types.
		decl := c.structDeclForType(tid)
		if decl != nil {
			for _, f := range decl.Fields {
				if !c.impliesMarker(marker, f.TypeOf()) {
					return false
				}
			}
			return true
		}
		return true
	case typetable.KindTraitObject:
		// `dyn Trait` is conservative: not Send unless the user
		// explicitly wrote `dyn Trait + Send`. At W07 we recognize
		// a positive impl as the override path.
		return false
	case typetable.KindGenericParam:
		// A generic parameter satisfies Send only if its bounds
		// include Send (W06 stored bounds on GenericParam nodes).
		for _, b := range c.boundsForGeneric(tid) {
			if c.traitIsMarker(b, marker) {
				return true
			}
		}
		return false
	}
	return false
}

// traitIsMarker returns true when tid names one of the intrinsic
// marker traits. The name comparison is intentional; marker trait
// identity is by declared name plus the checker's awareness that
// these are intrinsic.
func (c *checker) traitIsMarker(tid typetable.TypeId, marker MarkerTrait) bool {
	t := c.tab.Get(tid)
	if t == nil || t.Kind != typetable.KindTrait {
		return false
	}
	return t.Name == marker.String()
}

// --- Channel and spawn checks --------------------------------------

// checkChannelMethodCall recognizes `c.send(x)` and `c.recv()`
// where c has KindChannel type, and validates the element-type
// contract (reference §17.6). Channels whose element type is not
// Send are also reported here.
func (c *checker) checkChannelMethodCall(modPath string, call *hir.CallExpr) {
	fe, ok := call.Callee.(*hir.FieldExpr)
	if !ok {
		return
	}
	recvType := fe.Receiver.TypeOf()
	rt := c.tab.Get(recvType)
	if rt == nil || rt.Kind != typetable.KindChannel {
		return
	}
	elemType := typetable.NoType
	if len(rt.Children) > 0 {
		elemType = rt.Children[0]
	}
	// Send-bound on element.
	if !c.IsSend(elemType) {
		c.diagnose(call.Span,
			fmt.Sprintf("Chan[T] requires T: Send; %s is not Send (§17.6)", c.typeName(elemType)),
			"wrap the non-Send payload in Shared[T] or choose a Send-safe element type")
	}
	switch fe.Name {
	case "send":
		if len(call.Args) != 1 {
			c.diagnose(call.Span, "Chan.send takes exactly 1 argument",
				fmt.Sprintf("write `%s.send(value)`", c.typeName(recvType)))
			return
		}
		argType := call.Args[0].TypeOf()
		if argType != elemType && argType != typetable.NoType && elemType != typetable.NoType {
			c.diagnose(call.Args[0].NodeSpan(),
				fmt.Sprintf("Chan[%s].send expected %s, got %s",
					c.typeName(elemType), c.typeName(elemType), c.typeName(argType)),
				"the value type must match the channel's element type exactly")
		}
	case "recv":
		if len(call.Args) != 0 {
			c.diagnose(call.Span, "Chan.recv takes no arguments",
				fmt.Sprintf("write `%s.recv()`", c.typeName(recvType)))
		}
	}
}

// checkSpawn validates a spawn expression. The primary rules are:
//
//   - The body of `spawn closure` must be a closure whose environment
//     (captures) is Send (reference §47.1). At W07 we enforce a
//     pragmatic substitute: a non-`move` closure passed to spawn is
//     rejected, because a ref-captured environment is never Send.
//   - The spawn expression's TypeId is ThreadHandle[T] where T is
//     the closure's return type (this is already set by the
//     bridge; we re-check for belt-and-suspenders).
//   - Closures whose return type is not Send are also rejected
//     (the spawned value must cross a thread boundary).
func (c *checker) checkSpawn(modPath string, x *hir.SpawnExpr) {
	if x.Closure == nil {
		return
	}
	closure := x.Closure
	// TypeId shape check: must be ThreadHandle[T].
	st := c.tab.Get(x.TypeOf())
	if st == nil || st.Kind != typetable.KindThreadHandle {
		c.diagnose(x.Span,
			"spawn expression is not typed ThreadHandle[T]",
			"this is a compiler-internal bug; report it")
	}
	// Rule: non-move closures are rejected at spawn. The practical
	// reason: a non-move closure captures its environment by ref,
	// and ref T is never Send (§47.1). The diagnostic follows the
	// wave-spec template exactly.
	if !closure.IsMove {
		c.diagnose(x.Span,
			"spawned closure captures environment by ref, but ref T is not Send (§47.1)",
			"prefix the closure with `move` to capture by value, or wrap shared state in Shared[T] before spawning")
	}
	// Return type must be Send — the spawned handle will carry it
	// across thread boundaries.
	if !c.IsSend(closure.Return) {
		c.diagnose(closure.Span,
			fmt.Sprintf("spawned closure returns %s which is not Send (§47.1)", c.typeName(closure.Return)),
			"make the return type Send-safe or return a Send-wrapped value")
	}
}

// --- Lock ranking (@rank) -------------------------------------------

// RankAttribute is the parsed form of a `@rank(N)` decorator. W07
// consumes it via CheckRankDecorator so tests can construct
// synthetic rank assignments without routing through the
// still-incomplete decorator-to-HIR bridge.
type RankAttribute struct {
	Rank   int
	Anchor lex.Span // span of the `@rank(N)` occurrence for diagnostics
}

// CheckRankDecorator validates one `@rank(N)` annotation. Returns a
// slice of error strings; empty means the annotation is legal.
//
// Rules:
//   - N must be a positive integer. Zero and negatives are
//     diagnosed; §17.6 requires positive integer ranks.
//   - When the same item carries two `@rank` annotations, only the
//     first is legal; duplicates are reported by the caller since
//     this function validates one at a time.
func CheckRankDecorator(r RankAttribute) []string {
	var errs []string
	if r.Rank <= 0 {
		errs = append(errs, fmt.Sprintf("@rank(%d) must be a positive integer (§17.6)", r.Rank))
	}
	return errs
}

// CheckRankOrder validates that a sequence of rank values used by
// one function is strictly increasing — the W07 structural
// enforcement of lock ordering (reference §17.6). Returns a slice
// of error strings describing every violation found.
func CheckRankOrder(ranks []int) []string {
	var errs []string
	for i := 1; i < len(ranks); i++ {
		if ranks[i] <= ranks[i-1] {
			errs = append(errs,
				fmt.Sprintf("lock-rank order violation: position %d has rank %d but prior rank was %d; ranks must strictly increase",
					i, ranks[i], ranks[i-1]))
		}
	}
	return errs
}

// checkRankDecorators is a hook for per-impl rank validation. W07
// scope: accept @rank declarations silently on impl items; the
// real structural check is exercised via CheckRankOrder against
// synthetic rank sequences in tests. The end-to-end decorator
// wiring lands when decorators propagate into HIR (not yet).
func (c *checker) checkRankDecorators(impl *hir.ImplDecl) {
	_ = impl // hook reserved; currently a no-op
}

// --- Shared bounds --------------------------------------------------

// IsSharedSafe reports whether tid is safe to place inside a
// `Shared[T]`. Reference §17.6 requires T: Send + Sync.
func (c *checker) IsSharedSafe(tid typetable.TypeId) bool {
	return c.IsSend(tid) && c.IsSync(tid)
}
