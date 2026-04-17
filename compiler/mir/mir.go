package mir

import (
	"fmt"
	"sort"
)

// Reg is an opaque register identifier within a Function. Registers
// are local to their Function and have no meaning across functions.
type Reg int

// NoReg is the reserved zero value. A Reg of 0 means "no register" —
// useful for a Return whose value has not yet been assigned.
const NoReg Reg = 0

// BlockId identifies a Block within a Function. Like Reg, zero is
// reserved so a zero BlockId is unambiguously invalid.
type BlockId int

// NoBlock is the reserved zero BlockId.
const NoBlock BlockId = 0

// Op enumerates the instruction opcodes the W05 MIR supports. Adding
// a new Op is a wave change; the backend must be updated in lockstep
// or the codegen will emit the "unsupported opcode" diagnostic.
type Op int

const (
	// OpInvalid is the zero value. No valid instruction carries it.
	OpInvalid Op = iota
	// OpConstInt sets Dst to the integer constant IntValue.
	OpConstInt
	// OpAdd / OpSub / OpMul / OpDiv / OpMod compute Dst from Lhs
	// and Rhs using signed 64-bit arithmetic.
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	// OpParam reads the function parameter at ParamIndex into Dst.
	// Introduced at W06 to support fn parameters; the W05 spine did
	// not yet need this opcode.
	OpParam
	// OpCall invokes the function named CallName, passing the
	// registers in CallArgs, and stores the return value in Dst.
	// Introduced at W06 to support multi-function programs.
	OpCall
	// OpDrop invokes the destructor for the value held in Lhs.
	// CallName is the mangled `<TypeName>_drop` symbol. Introduced
	// at W09; emission is keyed by the liveness pass's DropIntent
	// list so codegen knows which locals need destruction.
	OpDrop
)

// String returns a stable human-readable name used by diagnostics,
// test output, and the C11 emitter's debug comments.
func (o Op) String() string {
	switch o {
	case OpInvalid:
		return "invalid"
	case OpConstInt:
		return "const_int"
	case OpAdd:
		return "add"
	case OpSub:
		return "sub"
	case OpMul:
		return "mul"
	case OpDiv:
		return "div"
	case OpMod:
		return "mod"
	case OpParam:
		return "param"
	case OpCall:
		return "call"
	case OpDrop:
		return "drop"
	}
	return "unknown"
}

// Inst is one non-terminator instruction inside a Block. Fields that
// are unused for a given Op are zero.
type Inst struct {
	Op         Op
	Dst        Reg
	Lhs        Reg
	Rhs        Reg
	IntValue   int64
	ParamIndex int    // OpParam: position in the containing fn's param list
	CallName   string // OpCall: C-level name of the callee
	CallArgs   []Reg  // OpCall: argument registers in order
}

// Terminator enumerates how a Block may end. Each Block has exactly
// one terminator; a Block with TermInvalid fails Validate.
type Terminator int

const (
	TermInvalid Terminator = iota
	// TermReturn terminates with a return value read from the
	// associated Block.ReturnReg register.
	TermReturn
	// TermJump is an unconditional branch to JumpTarget. Introduced
	// at W10 to support match-arm merge blocks.
	TermJump
	// TermIfEq compares Block.BranchReg with Block.BranchConst; if
	// equal, control flows to TrueTarget; otherwise FalseTarget.
	// Introduced at W10 to support cascading match dispatch.
	TermIfEq
)

// String renders a terminator for diagnostics.
func (t Terminator) String() string {
	switch t {
	case TermInvalid:
		return "invalid"
	case TermReturn:
		return "return"
	case TermJump:
		return "jump"
	case TermIfEq:
		return "if_eq"
	}
	return "unknown"
}

// Block is a basic block: a straight-line sequence of Inst followed
// by exactly one terminator.
//
// Terminator-specific fields:
//
//   - TermReturn: ReturnReg is the register whose value leaves the fn.
//   - TermJump: JumpTarget is the BlockId to transfer to.
//   - TermIfEq: BranchReg compared against BranchConst; on eq jumps
//     to TrueTarget, else to FalseTarget.
type Block struct {
	ID           BlockId
	Insts        []Inst
	Term         Terminator
	ReturnReg    Reg
	JumpTarget   BlockId
	BranchReg    Reg
	BranchConst  int64
	TrueTarget   BlockId
	FalseTarget  BlockId
}

// Function is one MIR function. Registers and block IDs are allocated
// by the Builder methods and remain stable for the lifetime of the
// Function (so tests can assert on specific register numbers).
type Function struct {
	Name      string
	Module    string
	NumRegs   int // total allocated registers (including NoReg slot 0)
	NumParams int // parameter count; first NumParams registers are params
	Blocks    []*Block
}

// Module is a collection of MIR functions that share a compilation
// unit. At W05 there is exactly one function per program (the `main`
// fn); later waves add multiple functions and inter-function
// references.
type Module struct {
	Functions []*Function
}

// Builder allocates registers and blocks while lowering. It is
// scoped to a single Function; separate Functions need separate
// Builders so register numbering stays per-function.
type Builder struct {
	fn      *Function
	current *Block
}

// NewFunction constructs an empty Function with its first Block
// pre-allocated. The first Block is the entry block.
func NewFunction(module, name string) (*Function, *Builder) {
	fn := &Function{Name: name, Module: module, NumRegs: 1} // reserve NoReg slot
	b := &Builder{fn: fn}
	b.BeginBlock()
	return fn, b
}

// BeginBlock allocates a new Block and makes it the current block.
// The newly allocated block has a fresh BlockId and is appended to
// the function's Blocks slice in allocation order.
func (b *Builder) BeginBlock() BlockId {
	id := BlockId(len(b.fn.Blocks) + 1)
	blk := &Block{ID: id}
	b.fn.Blocks = append(b.fn.Blocks, blk)
	b.current = blk
	return id
}

// NewReg allocates a fresh register in the function.
func (b *Builder) NewReg() Reg {
	r := Reg(b.fn.NumRegs)
	b.fn.NumRegs++
	return r
}

// ConstInt emits a const-int instruction and returns the destination
// register.
func (b *Builder) ConstInt(value int64) Reg {
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpConstInt, Dst: dst, IntValue: value,
	})
	return dst
}

// Binary emits a binary arithmetic instruction and returns the
// destination register. Op must be OpAdd/Sub/Mul/Div/Mod; any other
// Op panics (a pipeline bug, not a user error).
func (b *Builder) Binary(op Op, lhs, rhs Reg) Reg {
	switch op {
	case OpAdd, OpSub, OpMul, OpDiv, OpMod:
	default:
		panic(fmt.Sprintf("mir.Builder.Binary: %s is not a binary op", op))
	}
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: op, Dst: dst, Lhs: lhs, Rhs: rhs,
	})
	return dst
}

// Return terminates the current block with a TermReturn whose value
// comes from reg. After Return, the current block must not receive
// any more instructions; Builder panics on further emits.
func (b *Builder) Return(reg Reg) {
	b.current.Term = TermReturn
	b.current.ReturnReg = reg
	b.current = nil
}

// Param emits an OpParam reading the function parameter at index.
// Must be called once per parameter at the start of the function
// body; each call advances the param index automatically via the
// Function.NumParams counter. Returns the destination register.
func (b *Builder) Param(index int) Reg {
	dst := b.NewReg()
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpParam, Dst: dst, ParamIndex: index,
	})
	if index+1 > b.fn.NumParams {
		b.fn.NumParams = index + 1
	}
	return dst
}

// Call emits an OpCall invoking the named fn with the given argument
// registers and returns the destination register that receives the
// result.
func (b *Builder) Call(callName string, args []Reg) Reg {
	dst := b.NewReg()
	cloned := make([]Reg, len(args))
	copy(cloned, args)
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpCall, Dst: dst, CallName: callName, CallArgs: cloned,
	})
	return dst
}

// Drop emits an OpDrop that calls the destructor `dropName` on the
// value in `target`. Introduced at W09 to support types with
// `impl Drop`. The destructor is expected to take a pointer to the
// dropped value, matching codegen's `TypeName_drop(&_lN)` form.
func (b *Builder) Drop(dropName string, target Reg) {
	b.current.Insts = append(b.current.Insts, Inst{
		Op: OpDrop, Lhs: target, CallName: dropName,
	})
}

// Jump terminates the current block with an unconditional branch
// to target. After Jump the current block is sealed.
func (b *Builder) Jump(target BlockId) {
	b.current.Term = TermJump
	b.current.JumpTarget = target
	b.current = nil
}

// IfEq terminates the current block with a conditional branch:
// if reg == constant, control flows to trueTarget, else
// falseTarget.
func (b *Builder) IfEq(reg Reg, constant int64, trueTarget, falseTarget BlockId) {
	b.current.Term = TermIfEq
	b.current.BranchReg = reg
	b.current.BranchConst = constant
	b.current.TrueTarget = trueTarget
	b.current.FalseTarget = falseTarget
	b.current = nil
}

// UseBlock switches the builder to continue emitting into the
// named block. Required after BeginBlock allocates a future block
// that a prior TermJump / TermIfEq targets.
func (b *Builder) UseBlock(id BlockId) {
	for _, blk := range b.fn.Blocks {
		if blk.ID == id {
			b.current = blk
			return
		}
	}
}

// CurrentBlock returns the active block. A nil result indicates the
// builder has already terminated its current block; callers should
// call BeginBlock before emitting.
func (b *Builder) CurrentBlock() *Block { return b.current }

// Function returns the Function being built. Safe to call after
// Return; the Function is usable immediately.
func (b *Builder) Function() *Function { return b.fn }

// Validate enforces the W05 MIR invariants:
//
//   - Every Block has exactly one non-TermInvalid terminator.
//   - TermReturn blocks have a non-NoReg ReturnReg.
//   - Every register referenced by an instruction has been
//     previously defined by a ConstInt or Binary destination in the
//     same function (SSA-style "uses follow defs").
//   - Every Inst's Op is a recognized OpConstInt / OpAdd / OpSub /
//     OpMul / OpDiv / OpMod.
//
// Violations are compiler bugs, not user errors; Validate returns an
// error so the pipeline can fail loudly without leaving a half-baked
// MIR to downstream codegen.
func (f *Function) Validate() error {
	defined := map[Reg]bool{}
	for _, blk := range f.Blocks {
		for i, in := range blk.Insts {
			switch in.Op {
			case OpConstInt:
				if in.Dst == NoReg {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: const_int without destination", f.Name, blk.ID, i)
				}
				defined[in.Dst] = true
			case OpAdd, OpSub, OpMul, OpDiv, OpMod:
				if in.Dst == NoReg || in.Lhs == NoReg || in.Rhs == NoReg {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: %s missing register", f.Name, blk.ID, i, in.Op)
				}
				if !defined[in.Lhs] {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: use of undefined register %d", f.Name, blk.ID, i, in.Lhs)
				}
				if !defined[in.Rhs] {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: use of undefined register %d", f.Name, blk.ID, i, in.Rhs)
				}
				defined[in.Dst] = true
			case OpParam:
				if in.Dst == NoReg {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: param without destination", f.Name, blk.ID, i)
				}
				if in.ParamIndex < 0 {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: negative param index", f.Name, blk.ID, i)
				}
				defined[in.Dst] = true
			case OpCall:
				if in.Dst == NoReg {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: call without destination", f.Name, blk.ID, i)
				}
				if in.CallName == "" {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: call has empty target name", f.Name, blk.ID, i)
				}
				for j, a := range in.CallArgs {
					if !defined[a] {
						return fmt.Errorf("mir.Validate: %s/block %d inst %d: call arg %d uses undefined register %d", f.Name, blk.ID, i, j, a)
					}
				}
				defined[in.Dst] = true
			case OpDrop:
				if in.CallName == "" {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: drop has empty destructor name", f.Name, blk.ID, i)
				}
				if in.Lhs != NoReg && !defined[in.Lhs] {
					return fmt.Errorf("mir.Validate: %s/block %d inst %d: drop target register %d is undefined", f.Name, blk.ID, i, in.Lhs)
				}
			default:
				return fmt.Errorf("mir.Validate: %s/block %d inst %d: unknown op %d (W09 supports const_int/add/sub/mul/div/mod/param/call/drop)", f.Name, blk.ID, i, in.Op)
			}
		}
		switch blk.Term {
		case TermReturn:
			if blk.ReturnReg == NoReg {
				return fmt.Errorf("mir.Validate: %s/block %d: return without value register", f.Name, blk.ID)
			}
			if !defined[blk.ReturnReg] {
				return fmt.Errorf("mir.Validate: %s/block %d: return uses undefined register %d", f.Name, blk.ID, blk.ReturnReg)
			}
		case TermJump:
			if blk.JumpTarget == NoBlock {
				return fmt.Errorf("mir.Validate: %s/block %d: jump without target", f.Name, blk.ID)
			}
		case TermIfEq:
			if !defined[blk.BranchReg] {
				return fmt.Errorf("mir.Validate: %s/block %d: if_eq uses undefined branch register %d", f.Name, blk.ID, blk.BranchReg)
			}
			if blk.TrueTarget == NoBlock || blk.FalseTarget == NoBlock {
				return fmt.Errorf("mir.Validate: %s/block %d: if_eq missing branch targets", f.Name, blk.ID)
			}
		default:
			return fmt.Errorf("mir.Validate: %s/block %d: missing or invalid terminator (%s)", f.Name, blk.ID, blk.Term)
		}
	}
	return nil
}

// SortedFunctionNames returns the Module's function names in
// lexicographic order. Callers that want deterministic iteration
// use this instead of ranging Module.Functions directly (Rule 7.1).
func (m *Module) SortedFunctionNames() []string {
	out := make([]string, 0, len(m.Functions))
	for _, f := range m.Functions {
		out = append(out, f.Name)
	}
	sort.Strings(out)
	return out
}
