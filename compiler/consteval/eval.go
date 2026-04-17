package consteval

import (
	"fmt"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"github.com/Tembocs/fuse5/compiler/hir"
	"github.com/Tembocs/fuse5/compiler/lex"
	"github.com/Tembocs/fuse5/compiler/typetable"
)

// Diagnostic mirrors lex.Diagnostic for ergonomic merging with the
// rest of the compiler's diagnostic stream.
type Diagnostic = lex.Diagnostic

// Result is the output of Evaluate — one entry per evaluated
// constant, keyed by its defining symbol ID. Values are deterministic
// (Rule 7.1): callers iterating over ConstValues or StaticValues
// must walk SortedConstSymbols / SortedStaticSymbols rather than the
// map directly.
type Result struct {
	// ConstValues maps a const decl's symbol ID to its value.
	ConstValues map[int]Value

	// StaticValues maps a static decl's symbol ID to its value.
	// W14 evaluates non-extern statics at compile time because any
	// `static NAME: T = expr;` initializer must itself be a
	// constant expression (reference §22).
	StaticValues map[int]Value

	// ArrayLengths records the evaluated length of every Array
	// TypeId whose length was specified by a const-expression path.
	// W14 seeds this map only for paths already reduced to uint64
	// by the bridge; later waves can extend it as the grammar grows.
	ArrayLengths map[typetable.TypeId]uint64

	// DiscriminantValues maps (enum symbol ID, variant index) to
	// the evaluated discriminant value. Populated by EvaluateEnum
	// for enums that carry explicit `Variant = <expr>` forms.
	DiscriminantValues map[DiscriminantKey]int64
}

// DiscriminantKey identifies one enum discriminant for
// Result.DiscriminantValues.
type DiscriminantKey struct {
	Enum    int
	Variant int
}

// SortedConstSymbols returns the const symbol IDs in ascending order.
func (r *Result) SortedConstSymbols() []int {
	out := make([]int, 0, len(r.ConstValues))
	for k := range r.ConstValues {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// SortedStaticSymbols returns the static symbol IDs in ascending
// order.
func (r *Result) SortedStaticSymbols() []int {
	out := make([]int, 0, len(r.StaticValues))
	for k := range r.StaticValues {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// maxRecursionDepth caps const-fn recursion. Prevents runaway
// evaluation when the input program has a bug; real const-fn usage
// (factorial, table generation, size computations) fits comfortably
// inside this bound.
const maxRecursionDepth = 256

// maxLoopIterations bounds any `loop` / `while` / `for` iteration
// count during evaluation. A const fn that iterates beyond this is
// rejected as non-terminating rather than looping forever.
const maxLoopIterations = 1 << 20

// Evaluator is the per-evaluation state. One Evaluator per Evaluate
// call; the struct is exported so tests can drive it directly without
// going through the full Evaluate wrapper.
type Evaluator struct {
	Prog *hir.Program
	Tab  *typetable.Table

	constDecls  map[int]*hir.ConstDecl
	staticDecls map[int]*hir.StaticDecl
	constFns    map[int]*hir.FnDecl
	fnBySym     map[int]*hir.FnDecl

	// constValuesCache memoizes ConstDecl evaluations so a const
	// that is read repeatedly is evaluated exactly once. This is
	// both a performance shortcut and a correctness aid: a const
	// body with side-effectful-looking-but-pure shape (e.g. an
	// iterative counter) cannot produce different values in
	// different call sites.
	constValuesCache map[int]Value

	depth int
	diags []Diagnostic
}

// NewEvaluator wires a fresh Evaluator to a checked Program.
func NewEvaluator(prog *hir.Program) *Evaluator {
	ev := &Evaluator{
		Prog:             prog,
		Tab:              prog.Types,
		constDecls:       map[int]*hir.ConstDecl{},
		staticDecls:      map[int]*hir.StaticDecl{},
		constFns:         map[int]*hir.FnDecl{},
		fnBySym:          map[int]*hir.FnDecl{},
		constValuesCache: map[int]Value{},
	}
	ev.indexItems()
	return ev
}

// Evaluate runs the top-level W14 contract over prog: it evaluates
// every ConstDecl and non-extern StaticDecl and returns the result
// plus any diagnostics accumulated during evaluation.
//
// Deterministic iteration: prog.Order drives module iteration, and
// within each module items are iterated in their declared order.
func Evaluate(prog *hir.Program) (*Result, []Diagnostic) {
	ev := NewEvaluator(prog)
	res := &Result{
		ConstValues:        map[int]Value{},
		StaticValues:       map[int]Value{},
		ArrayLengths:       map[typetable.TypeId]uint64{},
		DiscriminantValues: map[DiscriminantKey]int64{},
	}
	for _, modPath := range prog.Order {
		mod := prog.Modules[modPath]
		for _, it := range mod.Items {
			switch x := it.(type) {
			case *hir.ConstDecl:
				if x.SymID == 0 {
					continue
				}
				if v, ok := ev.evalConstDecl(x); ok {
					res.ConstValues[x.SymID] = v
				}
			case *hir.StaticDecl:
				if x.IsExtern || x.Value == nil || x.SymID == 0 {
					continue
				}
				if v, ok := ev.evalStaticDecl(x); ok {
					res.StaticValues[x.SymID] = v
				}
			case *hir.EnumDecl:
				ev.evalEnumDiscriminants(x, res)
			}
		}
	}
	return res, ev.diags
}

// indexItems walks every module once and indexes ConstDecls,
// StaticDecls, and FnDecls (including const FnDecls) by their
// resolve.SymbolID. Called from NewEvaluator; downstream lookups
// are O(1).
func (ev *Evaluator) indexItems() {
	for _, modPath := range ev.Prog.Order {
		mod := ev.Prog.Modules[modPath]
		for _, it := range mod.Items {
			ev.indexItem(it)
		}
	}
}

func (ev *Evaluator) indexItem(it hir.Item) {
	switch x := it.(type) {
	case *hir.ConstDecl:
		if x.SymID != 0 {
			ev.constDecls[x.SymID] = x
		}
	case *hir.StaticDecl:
		if x.SymID != 0 {
			ev.staticDecls[x.SymID] = x
		}
	case *hir.FnDecl:
		if x.SymID != 0 {
			ev.fnBySym[x.SymID] = x
			if x.IsConst {
				ev.constFns[x.SymID] = x
			}
		}
	case *hir.ImplDecl:
		for _, sub := range x.Items {
			ev.indexItem(sub)
		}
	}
}

// evalConstDecl evaluates c.Value and caches the result.
func (ev *Evaluator) evalConstDecl(c *hir.ConstDecl) (Value, bool) {
	if c.SymID != 0 {
		if v, ok := ev.constValuesCache[c.SymID]; ok {
			return v, true
		}
	}
	if c.Value == nil {
		ev.emit(c.NodeSpan(),
			fmt.Sprintf("const %q has no initializer", c.Name),
			"add an initializer: `const "+c.Name+": T = <expr>;`")
		return Value{}, false
	}
	env := &scope{}
	v, flow := ev.evalExpr(c.Value, env)
	if flow != flowNormal {
		ev.emit(c.Value.NodeSpan(),
			"const initializer cannot return early",
			"remove the early return and let the final expression be the value")
		return Value{}, false
	}
	if v.Kind == VKInvalid {
		return Value{}, false
	}
	if c.SymID != 0 {
		ev.constValuesCache[c.SymID] = v
	}
	return v, true
}

// evalStaticDecl evaluates s.Value. A static's initializer must be a
// constant expression by §22; the same evaluator handles it.
func (ev *Evaluator) evalStaticDecl(s *hir.StaticDecl) (Value, bool) {
	if s.Value == nil {
		return Value{}, false
	}
	env := &scope{}
	v, flow := ev.evalExpr(s.Value, env)
	if flow != flowNormal {
		ev.emit(s.Value.NodeSpan(),
			"static initializer cannot return early",
			"remove the early return and let the final expression be the value")
		return Value{}, false
	}
	if v.Kind == VKInvalid {
		return Value{}, false
	}
	return v, true
}

// evalEnumDiscriminants walks an enum's variants and fills in the
// discriminant for each one. A variant without an explicit expression
// carries its declared index. The W14 const evaluator does not yet
// read an explicit `Variant = <expr>` form because the AST does not
// carry it; this function is in place so the §46.1 contract lands
// once W15 adds the HIR field. Today it records variant indices as
// deterministic discriminants (Rule 7.1).
func (ev *Evaluator) evalEnumDiscriminants(e *hir.EnumDecl, res *Result) {
	if e.SymID == 0 {
		return
	}
	for i := range e.Variants {
		res.DiscriminantValues[DiscriminantKey{Enum: e.SymID, Variant: i}] = int64(i)
	}
}

// flow signals the control-flow outcome of an evaluated expression
// or statement. `flowNormal` means the value in hand is the result;
// `flowReturn` means the enclosing fn (or block) must stop and
// propagate its return value up; `flowBreak` / `flowContinue` come
// from loops.
type flow int

const (
	flowNormal flow = iota
	flowReturn
	flowBreak
	flowContinue
	flowError
)

// scope is a lexical binding frame. Parent resolution is explicit
// (L013 defense): looking up an unbound name returns (Value{}, false)
// and the caller must emit a diagnostic rather than default to zero.
type scope struct {
	parent   *scope
	bindings map[string]Value
}

func newChildScope(parent *scope) *scope {
	return &scope{parent: parent, bindings: map[string]Value{}}
}

func (s *scope) bind(name string, v Value) {
	if s.bindings == nil {
		s.bindings = map[string]Value{}
	}
	s.bindings[name] = v
}

func (s *scope) lookup(name string) (Value, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if v, ok := cur.bindings[name]; ok {
			return v, true
		}
	}
	return Value{}, false
}

// evalExpr is the dispatch root. Every HIR expression kind either
// has a handler here or produces a diagnostic explaining why it is
// not a constant expression (Rule 6.9 — silent stubs forbidden).
func (ev *Evaluator) evalExpr(e hir.Expr, env *scope) (Value, flow) {
	if ev.depth > maxRecursionDepth {
		ev.emit(e.NodeSpan(),
			fmt.Sprintf("const-evaluation exceeded recursion depth of %d", maxRecursionDepth),
			"reduce recursion or refactor the algorithm to iterate")
		return Value{}, flowError
	}
	switch x := e.(type) {
	case *hir.LiteralExpr:
		return ev.evalLiteral(x)
	case *hir.PathExpr:
		return ev.evalPath(x, env)
	case *hir.UnaryExpr:
		return ev.evalUnary(x, env)
	case *hir.BinaryExpr:
		return ev.evalBinary(x, env)
	case *hir.IfExpr:
		return ev.evalIf(x, env)
	case *hir.Block:
		return ev.evalBlock(x, env)
	case *hir.CallExpr:
		return ev.evalCall(x, env)
	case *hir.CastExpr:
		return ev.evalCast(x, env)
	case *hir.TupleExpr:
		return ev.evalTuple(x, env)
	case *hir.StructLitExpr:
		return ev.evalStructLit(x, env)
	case *hir.FieldExpr:
		return ev.evalField(x, env)
	case *hir.IndexExpr:
		return ev.evalIndex(x, env)
	case *hir.MatchExpr:
		return ev.evalMatch(x, env)
	case *hir.LoopExpr:
		return ev.evalLoop(x, env)
	case *hir.WhileExpr:
		return ev.evalWhile(x, env)
	}
	ev.emit(e.NodeSpan(),
		fmt.Sprintf("expression form %T is not evaluable in a const context", e),
		"limit const bodies to arithmetic, comparison, `if`, `match`, and calls to other const fns")
	return Value{}, flowError
}

// evalLiteral handles LitInt, LitBool, and LitChar. LitString /
// LitFloat / LitNone are not constant-evaluable at W14 (reference
// §46.1 limits the surface to integer and boolean operations); the
// evaluator emits a diagnostic for them rather than silently
// defaulting.
func (ev *Evaluator) evalLiteral(l *hir.LiteralExpr) (Value, flow) {
	switch l.Kind {
	case hir.LitInt:
		u, err := parseIntegerLiteral(l.Text)
		if err != nil {
			ev.emit(l.NodeSpan(),
				fmt.Sprintf("invalid integer literal %q: %v", l.Text, err),
				"use a value that fits in the declared integer type")
			return Value{}, flowError
		}
		bits := integerBits(ev.Tab, l.Type)
		return IntValue(l.Type, maskToWidth(u, bits)), flowNormal
	case hir.LitBool:
		return BoolValue(ev.Tab.Bool(), l.Bool), flowNormal
	case hir.LitChar:
		cp, err := parseCharLiteral(l.Text)
		if err != nil {
			ev.emit(l.NodeSpan(),
				fmt.Sprintf("invalid char literal %q: %v", l.Text, err),
				"use a single Unicode scalar value")
			return Value{}, flowError
		}
		return CharValue(ev.Tab.Char(), cp), flowNormal
	}
	ev.emit(l.NodeSpan(),
		fmt.Sprintf("literal kind %d is not evaluable in a const context", l.Kind),
		"restrict const initializers to integer, boolean, and char literals")
	return Value{}, flowError
}

// parseIntegerLiteral accepts the same surface Go's strconv does for
// numeric base prefixes — 0x / 0o / 0b — and tolerates underscore
// separators used in the language reference. A trailing Fuse type
// suffix (one of i8/i16/i32/i64/u8/u16/u32/u64/isize/usize/f32/f64)
// is stripped before parsing; the checker already validated that
// the suffix matches the declared type.
func parseIntegerLiteral(text string) (uint64, error) {
	// Strip underscores; the lexer preserves them for diagnostics
	// but they are not part of the numeric value (reference §1.5).
	clean := strings.ReplaceAll(text, "_", "")
	clean = stripIntSuffix(clean)
	if strings.HasPrefix(clean, "-") {
		s, err := strconv.ParseInt(clean, 0, 64)
		if err != nil {
			return 0, err
		}
		return uint64(s), nil
	}
	return strconv.ParseUint(clean, 0, 64)
}

// stripIntSuffix removes a trailing Fuse numeric type suffix from the
// literal text. Suffix list mirrors the language reference (§1.5).
func stripIntSuffix(s string) string {
	for _, suf := range []string{"isize", "usize", "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64"} {
		if strings.HasSuffix(s, suf) {
			return s[:len(s)-len(suf)]
		}
	}
	return s
}

// parseCharLiteral extracts a Unicode code point from the source
// spelling. Source includes the single quotes; the W14 subset
// supports plain ASCII, `\n`, `\t`, `\r`, `\\`, `\'`, `\"`, and `\0`.
func parseCharLiteral(text string) (uint64, error) {
	if len(text) < 3 || text[0] != '\'' || text[len(text)-1] != '\'' {
		return 0, fmt.Errorf("malformed char literal")
	}
	body := text[1 : len(text)-1]
	if len(body) == 1 {
		return uint64(body[0]), nil
	}
	if len(body) >= 2 && body[0] == '\\' {
		switch body[1] {
		case 'n':
			return uint64('\n'), nil
		case 't':
			return uint64('\t'), nil
		case 'r':
			return uint64('\r'), nil
		case '\\':
			return uint64('\\'), nil
		case '\'':
			return uint64('\''), nil
		case '"':
			return uint64('"'), nil
		case '0':
			return 0, nil
		}
	}
	return 0, fmt.Errorf("unsupported escape sequence")
}

// evalPath resolves a path reference.
//
// Resolution order:
//  1. Single-segment name that names a bound local — local values
//     win (the HIR should never shadow a const with a local because
//     the resolver hoists top-level names, but this ordering keeps
//     a recursive const fn's parameter from being shadowed by a
//     sibling const).
//  2. Resolved symbol points at a ConstDecl — recursively evaluate.
//  3. Resolved symbol points at a static decl — recursively
//     evaluate (reference §22 says statics are const at init time).
//  4. Resolved symbol points at an enum variant — yield an Int value
//     equal to the variant's declared index.
//  5. Otherwise emit a diagnostic.
func (ev *Evaluator) evalPath(p *hir.PathExpr, env *scope) (Value, flow) {
	if len(p.Segments) == 1 {
		if v, ok := env.lookup(p.Segments[0]); ok {
			return v, flowNormal
		}
	}
	if p.Symbol != 0 {
		if c, ok := ev.constDecls[p.Symbol]; ok {
			ev.depth++
			defer func() { ev.depth-- }()
			v, ok := ev.evalConstDecl(c)
			if !ok {
				return Value{}, flowError
			}
			return v, flowNormal
		}
		if s, ok := ev.staticDecls[p.Symbol]; ok {
			ev.depth++
			defer func() { ev.depth-- }()
			v, ok := ev.evalStaticDecl(s)
			if !ok {
				return Value{}, flowError
			}
			return v, flowNormal
		}
	}
	// Two-segment paths that name an enum variant lower to an Int
	// discriminant. The checker has already resolved the type; we
	// accept the segments verbatim and consult the program's
	// EnumDecls for the variant index.
	if len(p.Segments) == 2 {
		if idx, ok := ev.lookupEnumVariantIndex(p.Segments[0], p.Segments[1]); ok {
			return IntValue(p.Type, uint64(idx)), flowNormal
		}
	}
	ev.emit(p.NodeSpan(),
		fmt.Sprintf("path %q does not name a constant value", strings.Join(p.Segments, ".")),
		"the const context accepts locals, other const declarations, static initializers, and enum variant names")
	return Value{}, flowError
}

// lookupEnumVariantIndex finds a variant by (enumName, variantName).
// Returns the variant's declared index in its enum's variant list.
func (ev *Evaluator) lookupEnumVariantIndex(enumName, variantName string) (int, bool) {
	for _, modPath := range ev.Prog.Order {
		mod := ev.Prog.Modules[modPath]
		for _, it := range mod.Items {
			e, ok := it.(*hir.EnumDecl)
			if !ok || e.Name != enumName {
				continue
			}
			for i, v := range e.Variants {
				if v.Name == variantName {
					return i, true
				}
			}
		}
	}
	return 0, false
}

// evalUnary computes prefix-unary operations.
func (ev *Evaluator) evalUnary(u *hir.UnaryExpr, env *scope) (Value, flow) {
	operand, fl := ev.evalExpr(u.Operand, env)
	if fl != flowNormal {
		return operand, fl
	}
	switch u.Op {
	case hir.UnNeg:
		if operand.Kind != VKInt {
			ev.emit(u.NodeSpan(),
				"unary `-` requires an integer operand in a const context",
				"apply `-` to an integer value")
			return Value{}, flowError
		}
		bits := integerBits(ev.Tab, operand.Type)
		return IntValue(operand.Type, maskToWidth(-operand.Int, bits)), flowNormal
	case hir.UnNot:
		if operand.Kind == VKBool {
			return BoolValue(operand.Type, !operand.Bool), flowNormal
		}
		if operand.Kind == VKInt {
			bits := integerBits(ev.Tab, operand.Type)
			return IntValue(operand.Type, maskToWidth(^operand.Int, bits)), flowNormal
		}
		ev.emit(u.NodeSpan(),
			"unary `!` requires a bool or integer operand in a const context",
			"apply `!` to a bool or integer value")
		return Value{}, flowError
	}
	ev.emit(u.NodeSpan(),
		"this unary operator is not evaluable in a const context",
		"the const evaluator supports `-` and `!`")
	return Value{}, flowError
}

// evalBinary implements arithmetic, comparison, bitwise, and
// short-circuiting logical operators on constant operands. Integer
// semantics follow the language reference:
//
//   - signed division and modulo use truncated (Go) semantics
//   - shift-left on signed types masks overflow away to match the
//     backend's eventual wrapping contract
//   - short-circuit && / || do not evaluate RHS when LHS determines
//     the result — important because evaluation is observable
//     through recursion depth bounds.
func (ev *Evaluator) evalBinary(b *hir.BinaryExpr, env *scope) (Value, flow) {
	// Short-circuit operators: evaluate RHS only when needed.
	if b.Op == hir.BinLogAnd || b.Op == hir.BinLogOr {
		lhs, fl := ev.evalExpr(b.Lhs, env)
		if fl != flowNormal {
			return lhs, fl
		}
		if lhs.Kind != VKBool {
			ev.emit(b.NodeSpan(),
				"`&&` / `||` require boolean operands",
				"use `&`, `|` for bitwise combination on integers")
			return Value{}, flowError
		}
		if b.Op == hir.BinLogAnd && !lhs.Bool {
			return BoolValue(ev.Tab.Bool(), false), flowNormal
		}
		if b.Op == hir.BinLogOr && lhs.Bool {
			return BoolValue(ev.Tab.Bool(), true), flowNormal
		}
		rhs, fl := ev.evalExpr(b.Rhs, env)
		if fl != flowNormal {
			return rhs, fl
		}
		if rhs.Kind != VKBool {
			ev.emit(b.NodeSpan(),
				"`&&` / `||` require boolean operands",
				"use `&`, `|` for bitwise combination on integers")
			return Value{}, flowError
		}
		return BoolValue(ev.Tab.Bool(), rhs.Bool), flowNormal
	}

	lhs, fl := ev.evalExpr(b.Lhs, env)
	if fl != flowNormal {
		return lhs, fl
	}
	rhs, fl := ev.evalExpr(b.Rhs, env)
	if fl != flowNormal {
		return rhs, fl
	}

	// Comparison — produce a Bool.
	if isComparisonOp(b.Op) {
		return ev.evalCompare(b, lhs, rhs)
	}

	// Arithmetic / bitwise — operands must be integers of the same
	// type. The checker enforces equal types on arithmetic operands
	// at W06; the evaluator double-checks and diagnoses if not.
	if lhs.Kind != VKInt || rhs.Kind != VKInt {
		ev.emit(b.NodeSpan(),
			fmt.Sprintf("operator %s requires integer operands in a const context", opSymbol(b.Op)),
			"apply arithmetic and bitwise operators to integer values")
		return Value{}, flowError
	}
	if lhs.Type != rhs.Type {
		ev.emit(b.NodeSpan(),
			"arithmetic operands differ in type in a const context",
			"cast one operand so both sides share a single integer type")
		return Value{}, flowError
	}
	bits := integerBits(ev.Tab, lhs.Type)
	signed := isSigned(ev.Tab, lhs.Type)

	switch b.Op {
	case hir.BinAdd:
		return IntValue(lhs.Type, maskToWidth(lhs.Int+rhs.Int, bits)), flowNormal
	case hir.BinSub:
		return IntValue(lhs.Type, maskToWidth(lhs.Int-rhs.Int, bits)), flowNormal
	case hir.BinMul:
		return IntValue(lhs.Type, maskToWidth(lhs.Int*rhs.Int, bits)), flowNormal
	case hir.BinDiv:
		if rhs.Int == 0 {
			ev.emit(b.NodeSpan(),
				"integer division by zero during const evaluation",
				"guard the divisor with `if d != 0 { a / d } else { 0 }` or use a non-zero literal")
			return Value{}, flowError
		}
		if signed {
			l := lhs.SignedInt(ev.Tab)
			r := rhs.SignedInt(ev.Tab)
			return IntValue(lhs.Type, maskToWidth(uint64(l/r), bits)), flowNormal
		}
		return IntValue(lhs.Type, maskToWidth(lhs.Int/rhs.Int, bits)), flowNormal
	case hir.BinMod:
		if rhs.Int == 0 {
			ev.emit(b.NodeSpan(),
				"integer modulo by zero during const evaluation",
				"guard the divisor or use a non-zero literal")
			return Value{}, flowError
		}
		if signed {
			l := lhs.SignedInt(ev.Tab)
			r := rhs.SignedInt(ev.Tab)
			return IntValue(lhs.Type, maskToWidth(uint64(l%r), bits)), flowNormal
		}
		return IntValue(lhs.Type, maskToWidth(lhs.Int%rhs.Int, bits)), flowNormal
	case hir.BinAnd:
		return IntValue(lhs.Type, maskToWidth(lhs.Int&rhs.Int, bits)), flowNormal
	case hir.BinOr:
		return IntValue(lhs.Type, maskToWidth(lhs.Int|rhs.Int, bits)), flowNormal
	case hir.BinXor:
		return IntValue(lhs.Type, maskToWidth(lhs.Int^rhs.Int, bits)), flowNormal
	case hir.BinShl:
		shift := uint(rhs.Int)
		if shift >= uint(bits) {
			return IntValue(lhs.Type, 0), flowNormal
		}
		return IntValue(lhs.Type, maskToWidth(lhs.Int<<shift, bits)), flowNormal
	case hir.BinShr:
		shift := uint(rhs.Int)
		if shift >= uint(bits) {
			if signed {
				// arithmetic shift of negative value saturates to -1
				l := lhs.SignedInt(ev.Tab)
				if l < 0 {
					return IntValue(lhs.Type, maskToWidth(^uint64(0), bits)), flowNormal
				}
			}
			return IntValue(lhs.Type, 0), flowNormal
		}
		if signed {
			l := lhs.SignedInt(ev.Tab)
			return IntValue(lhs.Type, maskToWidth(uint64(l>>shift), bits)), flowNormal
		}
		return IntValue(lhs.Type, maskToWidth(lhs.Int>>shift, bits)), flowNormal
	}
	ev.emit(b.NodeSpan(),
		fmt.Sprintf("binary operator %s is not evaluable in a const context", opSymbol(b.Op)),
		"the const evaluator supports arithmetic, comparison, bitwise, and short-circuiting logical operators")
	return Value{}, flowError
}

// evalCompare implements the six comparison operators over integer
// or boolean values. Integer comparison respects signedness of the
// operand type.
func (ev *Evaluator) evalCompare(b *hir.BinaryExpr, lhs, rhs Value) (Value, flow) {
	if lhs.Kind != rhs.Kind {
		ev.emit(b.NodeSpan(),
			"comparison operands must share a kind in a const context",
			"compare integers with integers and bools with bools")
		return Value{}, flowError
	}
	var out bool
	switch lhs.Kind {
	case VKInt:
		if isSigned(ev.Tab, lhs.Type) {
			li := lhs.SignedInt(ev.Tab)
			ri := rhs.SignedInt(ev.Tab)
			out = compareI64(li, ri, b.Op)
		} else {
			out = compareU64(lhs.Int, rhs.Int, b.Op)
		}
	case VKBool:
		var li, ri int64
		if lhs.Bool {
			li = 1
		}
		if rhs.Bool {
			ri = 1
		}
		out = compareI64(li, ri, b.Op)
	case VKChar:
		out = compareU64(lhs.Int, rhs.Int, b.Op)
	default:
		ev.emit(b.NodeSpan(),
			"comparison operands must be integers, booleans, or chars in a const context",
			"lift the computation into an integer form")
		return Value{}, flowError
	}
	return BoolValue(ev.Tab.Bool(), out), flowNormal
}

func compareI64(l, r int64, op hir.BinaryOp) bool {
	switch op {
	case hir.BinEq:
		return l == r
	case hir.BinNe:
		return l != r
	case hir.BinLt:
		return l < r
	case hir.BinLe:
		return l <= r
	case hir.BinGt:
		return l > r
	case hir.BinGe:
		return l >= r
	}
	return false
}

func compareU64(l, r uint64, op hir.BinaryOp) bool {
	switch op {
	case hir.BinEq:
		return l == r
	case hir.BinNe:
		return l != r
	case hir.BinLt:
		return l < r
	case hir.BinLe:
		return l <= r
	case hir.BinGt:
		return l > r
	case hir.BinGe:
		return l >= r
	}
	return false
}

// evalIf routes to then or else based on the cond value.
func (ev *Evaluator) evalIf(x *hir.IfExpr, env *scope) (Value, flow) {
	cond, fl := ev.evalExpr(x.Cond, env)
	if fl != flowNormal {
		return cond, fl
	}
	if cond.Kind != VKBool {
		ev.emit(x.Cond.NodeSpan(),
			"`if` condition in a const context must be a bool",
			"wrap the condition in a boolean expression such as `x != 0`")
		return Value{}, flowError
	}
	if cond.Bool {
		return ev.evalBlock(x.Then, env)
	}
	if x.Else == nil {
		return UnitValue(ev.Tab.Unit()), flowNormal
	}
	switch e := x.Else.(type) {
	case *hir.Block:
		return ev.evalBlock(e, env)
	case *hir.IfExpr:
		return ev.evalIf(e, env)
	}
	return ev.evalExpr(x.Else, env)
}

// evalBlock evaluates a block; the block's value is the trailing
// expression (if any) or Unit. ReturnStmts break out and propagate.
func (ev *Evaluator) evalBlock(blk *hir.Block, env *scope) (Value, flow) {
	child := newChildScope(env)
	for _, s := range blk.Stmts {
		v, fl := ev.evalStmt(s, child)
		if fl != flowNormal {
			return v, fl
		}
	}
	if blk.Trailing != nil {
		return ev.evalExpr(blk.Trailing, child)
	}
	return UnitValue(ev.Tab.Unit()), flowNormal
}

// evalStmt handles the subset of statements that are evaluable at
// compile time. Assignment is allowed only to `var` bindings the
// evaluator created; writes through pointers or field-assignment
// through `mutref` are non-const (reference §46.1).
func (ev *Evaluator) evalStmt(s hir.Stmt, env *scope) (Value, flow) {
	switch x := s.(type) {
	case *hir.LetStmt:
		if x.Value == nil {
			return UnitValue(ev.Tab.Unit()), flowNormal
		}
		v, fl := ev.evalExpr(x.Value, env)
		if fl != flowNormal {
			return v, fl
		}
		if ok := ev.bindPattern(x.Pattern, v, env); !ok {
			return Value{}, flowError
		}
		return UnitValue(ev.Tab.Unit()), flowNormal
	case *hir.VarStmt:
		v, fl := ev.evalExpr(x.Value, env)
		if fl != flowNormal {
			return v, fl
		}
		env.bind(x.Name, v)
		return UnitValue(ev.Tab.Unit()), flowNormal
	case *hir.ReturnStmt:
		if x.Value == nil {
			return UnitValue(ev.Tab.Unit()), flowReturn
		}
		v, fl := ev.evalExpr(x.Value, env)
		if fl == flowError {
			return v, fl
		}
		return v, flowReturn
	case *hir.BreakStmt:
		if x.Value != nil {
			v, fl := ev.evalExpr(x.Value, env)
			if fl == flowError {
				return v, fl
			}
			return v, flowBreak
		}
		return UnitValue(ev.Tab.Unit()), flowBreak
	case *hir.ContinueStmt:
		return UnitValue(ev.Tab.Unit()), flowContinue
	case *hir.ExprStmt:
		return ev.evalExpr(x.Expr, env)
	}
	ev.emit(s.NodeSpan(),
		fmt.Sprintf("statement form %T is not evaluable in a const context", s),
		"limit const bodies to let/var bindings, `return`, and expression statements")
	return Value{}, flowError
}

// bindPattern binds a value to a pattern's names. W14 supports the
// subset of patterns the checker has proved total: BindPat for plain
// bindings, WildcardPat to discard, ConstructorPat with positional
// sub-patterns for tuple destructuring, and LiteralPat for
// refutable-on-match positions (the caller checks for mismatch).
func (ev *Evaluator) bindPattern(p hir.Pat, v Value, env *scope) bool {
	switch x := p.(type) {
	case *hir.BindPat:
		env.bind(x.Name, v)
		return true
	case *hir.WildcardPat:
		return true
	case *hir.ConstructorPat:
		// Tuple-shaped destructuring.
		if len(x.Tuple) > 0 {
			if v.Kind != VKTuple && v.Kind != VKStruct {
				ev.emit(p.NodeSpan(),
					"tuple destructuring in a const context requires a tuple or tuple-struct value",
					"supply a tuple literal or a tuple-struct constructor")
				return false
			}
			if v.Kind == VKTuple && len(x.Tuple) != len(v.Elems) {
				ev.emit(p.NodeSpan(),
					fmt.Sprintf("tuple arity mismatch: pattern has %d elements, value has %d",
						len(x.Tuple), len(v.Elems)),
					"align the pattern arity with the tuple's arity")
				return false
			}
			for i, sub := range x.Tuple {
				if !ev.bindPattern(sub, v.Elems[i], env) {
					return false
				}
			}
			return true
		}
		if len(x.Fields) > 0 {
			if v.Kind != VKStruct {
				ev.emit(p.NodeSpan(),
					"struct destructuring in a const context requires a struct value",
					"supply a struct literal")
				return false
			}
			for _, f := range x.Fields {
				sub, ok := v.Fields[f.Name]
				if !ok {
					ev.emit(p.NodeSpan(),
						fmt.Sprintf("field %q not present in the struct value", f.Name),
						"match only fields that the struct declares")
					return false
				}
				if !ev.bindPattern(f.Pattern, sub, env) {
					return false
				}
			}
			return true
		}
		// Unit-variant pattern: nothing to bind.
		return true
	case *hir.LiteralPat:
		// LiteralPat in a let/var position behaves as refutable;
		// the evaluator rejects it in W14 (W10 match handles
		// refutable patterns inside match arms).
		ev.emit(p.NodeSpan(),
			"literal pattern in a const let/var binding is refutable",
			"use `match` for refutable pattern matching")
		return false
	}
	ev.emit(p.NodeSpan(),
		fmt.Sprintf("pattern form %T is not evaluable in a const context", p),
		"limit const destructuring to plain names, wildcards, and tuple or struct constructors")
	return false
}

// evalCall handles direct calls to const fns and the size_of /
// align_of intrinsics.
func (ev *Evaluator) evalCall(c *hir.CallExpr, env *scope) (Value, flow) {
	// Intrinsic recognition: the callee path's final segment is
	// "size_of" or "align_of" and it carries exactly one TypeArg.
	if intr, ok := recognizeIntrinsic(c); ok {
		return ev.evalIntrinsic(c, intr)
	}
	callee, ok := c.Callee.(*hir.PathExpr)
	if !ok {
		ev.emit(c.NodeSpan(),
			"const call target must be a plain fn path",
			"remove indirect or method-dispatch callees from const contexts")
		return Value{}, flowError
	}
	fn, ok := ev.fnBySym[callee.Symbol]
	if !ok {
		ev.emit(c.NodeSpan(),
			fmt.Sprintf("cannot resolve fn %q in a const context", strings.Join(callee.Segments, ".")),
			"call a fn declared with `const fn`")
		return Value{}, flowError
	}
	if !fn.IsConst {
		ev.emit(c.NodeSpan(),
			fmt.Sprintf("cannot call non-const fn %q in a const context", fn.Name),
			"declare the callee with `const fn`, or inline the expression")
		return Value{}, flowError
	}
	if fn.IsExtern {
		ev.emit(c.NodeSpan(),
			fmt.Sprintf("cannot call FFI fn %q in a const context", fn.Name),
			"const fns cannot call extern declarations")
		return Value{}, flowError
	}
	if len(fn.Params) != len(c.Args) {
		ev.emit(c.NodeSpan(),
			fmt.Sprintf("const call arity mismatch: fn %q expects %d args, got %d",
				fn.Name, len(fn.Params), len(c.Args)),
			"match the fn's declared parameter count")
		return Value{}, flowError
	}
	// Evaluate arguments in call-site scope.
	argVals := make([]Value, len(c.Args))
	for i, a := range c.Args {
		v, fl := ev.evalExpr(a, env)
		if fl != flowNormal {
			return v, fl
		}
		argVals[i] = v
	}
	// Fresh scope with param bindings for the callee body.
	callScope := newChildScope(nil)
	for i, p := range fn.Params {
		callScope.bind(p.Name, argVals[i])
	}
	ev.depth++
	defer func() { ev.depth-- }()
	if fn.Body == nil {
		ev.emit(c.NodeSpan(),
			fmt.Sprintf("const fn %q has no body", fn.Name),
			"provide a body for the const fn")
		return Value{}, flowError
	}
	v, fl := ev.evalBlock(fn.Body, callScope)
	if fl == flowReturn {
		return v, flowNormal
	}
	if fl == flowError {
		return v, fl
	}
	return v, flowNormal
}

// evalCast narrows or widens an integer value to the cast's target
// type. Bool→Int and Int→Bool casts are rejected; the evaluator
// mirrors the checker's existing surface rather than implementing a
// broader cast table (reference §15 "Casts").
func (ev *Evaluator) evalCast(c *hir.CastExpr, env *scope) (Value, flow) {
	inner, fl := ev.evalExpr(c.Expr, env)
	if fl != flowNormal {
		return inner, fl
	}
	target := c.TypeOf()
	if !isIntegerType(ev.Tab, target) && typeKind(ev.Tab, target) != typetable.KindChar {
		ev.emit(c.NodeSpan(),
			"const cast target must be an integer or char type",
			"cast between integer widths or to `Char`")
		return Value{}, flowError
	}
	if inner.Kind == VKBool {
		if inner.Bool {
			return IntValue(target, 1), flowNormal
		}
		return IntValue(target, 0), flowNormal
	}
	if inner.Kind != VKInt && inner.Kind != VKChar {
		ev.emit(c.NodeSpan(),
			"const cast source must be an integer, bool, or char",
			"cast from an integer/bool/char value")
		return Value{}, flowError
	}
	targetBits := integerBits(ev.Tab, target)
	// Sign-extend when the source is signed and narrower than the
	// target; otherwise just mask to the target width.
	srcBits := integerBits(ev.Tab, inner.Type)
	v := inner.Int
	if isSigned(ev.Tab, inner.Type) && srcBits < targetBits {
		if v&(uint64(1)<<(srcBits-1)) != 0 {
			// extend sign
			mask := ^uint64(0) << srcBits
			v |= mask
		}
	}
	return IntValue(target, maskToWidth(v, targetBits)), flowNormal
}

// evalTuple constructs a tuple value from positional sub-expressions.
func (ev *Evaluator) evalTuple(t *hir.TupleExpr, env *scope) (Value, flow) {
	elems := make([]Value, len(t.Elements))
	for i, e := range t.Elements {
		v, fl := ev.evalExpr(e, env)
		if fl != flowNormal {
			return v, fl
		}
		elems[i] = v
	}
	return TupleValue(t.TypeOf(), elems), flowNormal
}

// evalStructLit builds a struct value; field order tracks source
// declaration order so String() and goldens are stable (Rule 7.1).
func (ev *Evaluator) evalStructLit(s *hir.StructLitExpr, env *scope) (Value, flow) {
	order := make([]string, 0, len(s.Fields))
	fields := map[string]Value{}
	for _, f := range s.Fields {
		v, fl := ev.evalExpr(f.Value, env)
		if fl != flowNormal {
			return v, fl
		}
		order = append(order, f.Name)
		fields[f.Name] = v
	}
	return StructValue(s.StructType, order, fields), flowNormal
}

// evalField reads a field from a struct or a tuple index.
func (ev *Evaluator) evalField(f *hir.FieldExpr, env *scope) (Value, flow) {
	recv, fl := ev.evalExpr(f.Receiver, env)
	if fl != flowNormal {
		return recv, fl
	}
	switch recv.Kind {
	case VKStruct:
		v, ok := recv.Fields[f.Name]
		if !ok {
			ev.emit(f.NodeSpan(),
				fmt.Sprintf("field %q not found on the struct value", f.Name),
				"use a field name declared by the struct")
			return Value{}, flowError
		}
		return v, flowNormal
	case VKTuple:
		idx, err := strconv.Atoi(f.Name)
		if err != nil || idx < 0 || idx >= len(recv.Elems) {
			ev.emit(f.NodeSpan(),
				fmt.Sprintf("tuple index %q is out of range (arity %d)", f.Name, len(recv.Elems)),
				"use a numeric index in `0..len(tuple)`")
			return Value{}, flowError
		}
		return recv.Elems[idx], flowNormal
	}
	ev.emit(f.NodeSpan(),
		"field access in a const context requires a struct or tuple value",
		"construct the receiver from a struct literal or tuple literal")
	return Value{}, flowError
}

// evalIndex reads an array element at a constant index.
func (ev *Evaluator) evalIndex(x *hir.IndexExpr, env *scope) (Value, flow) {
	recv, fl := ev.evalExpr(x.Receiver, env)
	if fl != flowNormal {
		return recv, fl
	}
	if recv.Kind != VKArray {
		ev.emit(x.NodeSpan(),
			"const indexing requires an array value",
			"build the receiver from an array literal or a const array initializer")
		return Value{}, flowError
	}
	idx, fl := ev.evalExpr(x.Index, env)
	if fl != flowNormal {
		return idx, fl
	}
	if idx.Kind != VKInt {
		ev.emit(x.NodeSpan(),
			"const index must be an integer",
			"supply an integer-typed const expression")
		return Value{}, flowError
	}
	i := int(idx.Int)
	if i < 0 || i >= len(recv.Elems) {
		ev.emit(x.NodeSpan(),
			fmt.Sprintf("const index %d out of bounds (len %d)", i, len(recv.Elems)),
			"check the index is in `0..len(array)` at the call site")
		return Value{}, flowError
	}
	return recv.Elems[i], flowNormal
}

// evalMatch dispatches to the first arm whose pattern matches and
// whose optional guard evaluates to true.
func (ev *Evaluator) evalMatch(m *hir.MatchExpr, env *scope) (Value, flow) {
	scrut, fl := ev.evalExpr(m.Scrutinee, env)
	if fl != flowNormal {
		return scrut, fl
	}
	for _, arm := range m.Arms {
		armScope := newChildScope(env)
		matched, ok := ev.matchPattern(arm.Pattern, scrut, armScope)
		if !ok {
			return Value{}, flowError
		}
		if !matched {
			continue
		}
		if arm.Guard != nil {
			g, gfl := ev.evalExpr(arm.Guard, armScope)
			if gfl != flowNormal {
				return g, gfl
			}
			if g.Kind != VKBool || !g.Bool {
				continue
			}
		}
		return ev.evalBlock(arm.Body, armScope)
	}
	ev.emit(m.NodeSpan(),
		"no match arm matched the scrutinee in a const context",
		"add a wildcard arm `_ => <value>` or cover every variant")
	return Value{}, flowError
}

// matchPattern is the evaluator's refutable pattern matcher. Returns
// (matched, ok): matched is whether the pattern matched; ok is
// whether evaluation succeeded. Bind- and Wildcard-patterns always
// match; literal and constructor patterns are refutable.
func (ev *Evaluator) matchPattern(p hir.Pat, scrut Value, env *scope) (bool, bool) {
	switch x := p.(type) {
	case *hir.WildcardPat:
		return true, true
	case *hir.BindPat:
		env.bind(x.Name, scrut)
		return true, true
	case *hir.LiteralPat:
		switch x.Kind {
		case hir.LitInt:
			if scrut.Kind != VKInt {
				return false, true
			}
			u, err := parseIntegerLiteral(x.Text)
			if err != nil {
				ev.emit(p.NodeSpan(),
					fmt.Sprintf("invalid integer pattern %q", x.Text),
					"use a valid integer literal")
				return false, false
			}
			return maskToWidth(u, integerBits(ev.Tab, scrut.Type)) == scrut.Int, true
		case hir.LitBool:
			if scrut.Kind != VKBool {
				return false, true
			}
			return x.Bool == scrut.Bool, true
		}
		return false, true
	case *hir.ConstructorPat:
		// Unit-variant pattern — matches if the scrutinee's Int
		// value (treated as a discriminant) equals the variant
		// index recorded in the enum.
		if len(x.Tuple) == 0 && len(x.Fields) == 0 {
			if scrut.Kind != VKInt {
				return false, true
			}
			if idx, ok := ev.lookupEnumVariantIndex(firstPathSeg(x.Path), x.VariantName); ok {
				return uint64(idx) == scrut.Int, true
			}
			return false, true
		}
		// Tuple-shaped variant pattern: scrutinee must be a tuple
		// Value produced elsewhere (the evaluator has no runtime
		// payload model yet).
		if len(x.Tuple) > 0 {
			if scrut.Kind != VKTuple {
				return false, true
			}
			if len(x.Tuple) != len(scrut.Elems) {
				return false, true
			}
			for i, sub := range x.Tuple {
				ok := bindOrRecurse(ev, sub, scrut.Elems[i], env)
				if !ok {
					return false, true
				}
			}
			return true, true
		}
		// Struct-shaped variant pattern.
		if scrut.Kind != VKStruct {
			return false, true
		}
		for _, f := range x.Fields {
			v, present := scrut.Fields[f.Name]
			if !present {
				return false, true
			}
			if !bindOrRecurse(ev, f.Pattern, v, env) {
				return false, true
			}
		}
		return true, true
	case *hir.OrPat:
		for _, alt := range x.Alts {
			matched, ok := ev.matchPattern(alt, scrut, env)
			if !ok {
				return false, false
			}
			if matched {
				return true, true
			}
		}
		return false, true
	case *hir.RangePat:
		lo, fl := ev.evalExpr(x.Lo, env)
		if fl != flowNormal {
			return false, false
		}
		hi, fl := ev.evalExpr(x.Hi, env)
		if fl != flowNormal {
			return false, false
		}
		if scrut.Kind != VKInt || lo.Kind != VKInt || hi.Kind != VKInt {
			return false, true
		}
		if scrut.Int < lo.Int {
			return false, true
		}
		if x.Inclusive {
			return scrut.Int <= hi.Int, true
		}
		return scrut.Int < hi.Int, true
	}
	ev.emit(p.NodeSpan(),
		fmt.Sprintf("pattern form %T is not matchable in a const context", p),
		"limit const match arms to literals, wildcards, binds, and constructor patterns")
	return false, false
}

func bindOrRecurse(ev *Evaluator, p hir.Pat, v Value, env *scope) bool {
	switch sub := p.(type) {
	case *hir.BindPat:
		env.bind(sub.Name, v)
		return true
	case *hir.WildcardPat:
		return true
	}
	m, ok := ev.matchPattern(p, v, env)
	return ok && m
}

func firstPathSeg(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[0]
}

// evalLoop evaluates an unconditional `loop { body }`. The loop must
// exit via `break` within maxLoopIterations; unreachable infinite
// loops produce a diagnostic.
func (ev *Evaluator) evalLoop(x *hir.LoopExpr, env *scope) (Value, flow) {
	for i := 0; i < maxLoopIterations; i++ {
		v, fl := ev.evalBlock(x.Body, env)
		switch fl {
		case flowNormal, flowContinue:
			continue
		case flowBreak:
			return v, flowNormal
		case flowReturn, flowError:
			return v, fl
		}
	}
	ev.emit(x.NodeSpan(),
		fmt.Sprintf("const `loop` did not `break` within %d iterations", maxLoopIterations),
		"add a `break` that triggers within the const-evaluation iteration bound")
	return Value{}, flowError
}

// evalWhile evaluates a `while cond { body }` loop.
func (ev *Evaluator) evalWhile(x *hir.WhileExpr, env *scope) (Value, flow) {
	for i := 0; i < maxLoopIterations; i++ {
		cond, fl := ev.evalExpr(x.Cond, env)
		if fl != flowNormal {
			return cond, fl
		}
		if cond.Kind != VKBool {
			ev.emit(x.Cond.NodeSpan(),
				"`while` condition in a const context must be a bool",
				"use a boolean expression such as `i < n`")
			return Value{}, flowError
		}
		if !cond.Bool {
			return UnitValue(ev.Tab.Unit()), flowNormal
		}
		v, bfl := ev.evalBlock(x.Body, env)
		switch bfl {
		case flowNormal, flowContinue:
			continue
		case flowBreak:
			return v, flowNormal
		case flowReturn, flowError:
			return v, bfl
		}
	}
	ev.emit(x.NodeSpan(),
		fmt.Sprintf("const `while` did not terminate within %d iterations", maxLoopIterations),
		"reduce the loop bound or rewrite iteratively")
	return Value{}, flowError
}

// emit appends a diagnostic with span, primary explanation, and
// suggestion (Rule 6.17).
func (ev *Evaluator) emit(span lex.Span, msg, suggestion string) {
	ev.diags = append(ev.diags, Diagnostic{
		Span:    span,
		Message: msg,
		Hint:    suggestion,
	})
}

// Diagnostics exposes the accumulated diagnostics so tests can
// inspect them directly.
func (ev *Evaluator) Diagnostics() []Diagnostic { return ev.diags }

// isComparisonOp reports whether op produces a Bool from two
// operands of the same kind.
func isComparisonOp(op hir.BinaryOp) bool {
	switch op {
	case hir.BinEq, hir.BinNe, hir.BinLt, hir.BinLe, hir.BinGt, hir.BinGe:
		return true
	}
	return false
}

// opSymbol returns a human-readable spelling of op for diagnostics.
func opSymbol(op hir.BinaryOp) string {
	switch op {
	case hir.BinAdd:
		return "`+`"
	case hir.BinSub:
		return "`-`"
	case hir.BinMul:
		return "`*`"
	case hir.BinDiv:
		return "`/`"
	case hir.BinMod:
		return "`%`"
	case hir.BinShl:
		return "`<<`"
	case hir.BinShr:
		return "`>>`"
	case hir.BinAnd:
		return "`&`"
	case hir.BinOr:
		return "`|`"
	case hir.BinXor:
		return "`^`"
	case hir.BinLogAnd:
		return "`&&`"
	case hir.BinLogOr:
		return "`||`"
	case hir.BinEq:
		return "`==`"
	case hir.BinNe:
		return "`!=`"
	case hir.BinLt:
		return "`<`"
	case hir.BinLe:
		return "`<=`"
	case hir.BinGt:
		return "`>`"
	case hir.BinGe:
		return "`>=`"
	}
	return "<?>"
}

// assertUintFits reports whether v fits in `bits` unsigned bits. Used
// by intrinsic evaluation where USize has a specific width.
func assertUintFits(v uint64, b int) bool {
	if b >= 64 {
		return true
	}
	return v>>b == 0
}

// ceilToPow2 rounds up to the next power of two. Used in layout
// computations for alignment rounding.
func ceilToPow2(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	return uint64(1) << (64 - bits.LeadingZeros64(n-1))
}
