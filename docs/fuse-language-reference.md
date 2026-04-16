# Fuse Language Reference

> Status: normative for Fuse.
>
> This document is the specification of the Fuse language. It is written for
> two audiences at once:
>
> - users, who need to understand how to write correct Fuse programs
> - implementers, who must be able to build a correct compiler from this
>   document alone
>
> Every section that defines a language feature includes an implementation
> contract where the feature has behavior that is not deducible from surface
> syntax alone. If an implementation contract is ambiguous, the document is
> defective.
>
> Every feature section carries an implementation status tag:
>
> - `SPECIFIED — Wxx`: specified here, implementation scheduled for the named
>   wave. Not yet implemented.
> - `DONE — Wxx`: implemented, proof program exists in `tests/e2e/`, CI passes.
> - `STUB — emits: "..."`: partially wired; the compiler emits the named
>   diagnostic when this feature is used. Entry exists in STUBS.md.
>
> Every feature is scheduled to a concrete wave. "TBD" is not a status
> (Rule 2.5).
>
> Every code example that demonstrates a complete program includes its expected
> output. These examples are the source of truth for the e2e test suite and
> must fail if the feature is stubbed.
>
> Fuse v1 is frozen. Features not described in this document do not exist.

## Splitting recommendation

This document is kept as a single file for review. For final published form,
splitting into three sub-documents is recommended:

**1. `reference-core.md` — §1–7**
Lexical basics, all types, variables, operators, control flow, pattern
matching, and functions. The part a new reader needs to be productive. Has no
dependency on ownership or generics knowledge.

**2. `reference-types.md` — §8–16**
Methods, structs, enums, traits, generics, ownership and borrowing, closures,
and error handling. The type system and ownership model reward focused reading
and are frequently cross-referenced when writing non-trivial programs.

**3. `reference-systems.md` — §17–27 and appendices**
Concurrency, modules, constants, type aliases, iterators, extern/FFI, unsafe,
primitive methods, numeric conversions, and decorators. The systems-level
surface that most readers only need occasionally and advanced users reach for
deliberately.

Each sub-document would be self-contained with its own short introduction. The
complete example in §27 and the appendices would move to `reference-systems.md`
as a natural landing point after the full language is introduced.

---

---

## 1. Lexical Basics

> Implementation status: SPECIFIED — W01

### 1.1 Source encoding

Fuse source files are UTF-8. A byte-order mark is not permitted. Line endings
may be LF or CRLF; the lexer normalizes them.

### 1.2 Comments

Line comments run to end of line. Block comments may nest.

```fuse
// this is a line comment

/* this is a block comment */

/* block comments
   /* may nest */
   without issue */
```

### 1.3 Integer literals

Decimal, hexadecimal, octal, and binary forms are all supported. An optional
suffix pins the type; without a suffix the type is inferred from context.

```fuse
let a = 42;            // inferred
let b = 42i32;         // I32
let c = 0xff_u8;       // U8, value 255
let d = 0o77_usize;    // USize, value 63
let e = 0b1010_i64;    // I64, value 10
```

### 1.4 Float literals

A float literal contains a decimal point, an exponent, or both. Suffixes `f32`
and `f64` pin the type.

```fuse
let x = 1.0;           // F64 (default)
let y = 3.14f32;       // F32
let z = 6.02e23;       // F64
let w = 1.5e-4f64;     // F64
```

### 1.5 String literals

String literals are UTF-8. Standard escape sequences apply.

```fuse
let s = "hello, world\n";
let t = "tab\there";
let u = "quote: \"yes\"";
let v = "backslash: \\";
let w = "unicode: \u{1F600}";
```

### 1.6 Raw string literals

Raw strings contain no escape sequences. The number of `#` characters on the
opener and closer must match. The sequence `r#abc` is not a raw string.

```fuse
let a = r"no \n escapes here";
let b = r#"can contain " freely"#;
let c = r##"can contain "# freely"##;
```

### 1.7 Boolean literals

```fuse
let yes = true;
let no  = false;
```

### 1.8 Unit literal

Unit is both a type and its own sole value.

```fuse
let u: () = ();
```

### 1.9 Identifiers and keywords

Identifiers start with a Unicode letter or `_` and continue with Unicode
letters, digits, or `_`. The following words are reserved and may not be used
as identifiers:

```
fn  pub  struct  enum  trait  impl  for  in  while  loop
if  else  match  return  let  var  move  ref  mutref  owned
unsafe  spawn  chan  import  as  mod  use  type  const  static
extern  break  continue  where  Self  self  true  false  None  Some
```

### 1.10 Implementation contracts

#### Raw string recognition

A raw string is recognized only when the lexer matches the full prefix pattern
`r#*"` and the corresponding closing `"#*` sequence exists. The lexer must not
enter raw-string mode on `r` followed by `#` alone.

`r#abc` must tokenize as:

```text
IDENT("r")
#
IDENT("abc")
```

#### `?.` longest-match rule

The lexer emits `?.` as one token. It is not `?` followed by `.`. The parser
must interpret `expr?.field` as optional chaining, not as postfix `?` applied
to `expr` followed by field access on the result.

#### BOM rejection

A UTF-8 byte-order mark at the start of a source file is a lexical error.
The lexer must emit a diagnostic and refuse to tokenize the file. Silent BOM
stripping is forbidden because it hides encoding drift across tools.

#### Literal normalization

Literal text must be normalized at the HIR-to-MIR boundary.

- Integer suffixes are stripped before MIR constant emission.
- String literal payloads are stored without their surrounding quote characters.

The generated C must never contain raw Fuse literal spellings such as
`INT32_C(64usize)` or doubled string quotes such as `""NaN""`.

---

## 2. Primitive Types

> Implementation status: SPECIFIED — W04 (TypeTable), W06 (type checking)

### 2.1 Integer types

Signed and unsigned integers in widths 8, 16, 32, 64, and 128 bits, plus
pointer-sized variants.

```fuse
let a: I8   = -127;
let b: I16  = 32_000;
let c: I32  = 1_000_000;
let d: I64  = 9_000_000_000;
let e: I128 = 170_141_183_460_469_231_731;
let f: ISize = -1;          // platform word size, signed

let g: U8   = 255;
let h: U16  = 65_535;
let i: U32  = 4_294_967_295;
let j: U64  = 18_446_744_073_709_551_615;
let k: U128 = 340_282_366_920_938_463_463;
let l: USize = 0;           // platform word size, unsigned
```

### 2.2 Platform-sized aliases

`Int` is `ISize` and `Float` is `F64`. These are language-level aliases, not
separate types.

```fuse
let n: Int   = -1;    // same as ISize
let f: Float = 0.0;   // same as F64
```

### 2.3 Floating-point types

```fuse
let a: F32 = 1.0f32;
let b: F64 = 1.0;      // F64 is the default float type
```

### 2.4 Boolean

```fuse
let x: Bool = true;
let y: Bool = false;
```

### 2.5 Character

A `Char` is a Unicode scalar value.

```fuse
let c: Char = 'A';
let d: Char = '✓';
```

### 2.6 Unit

Unit `()` is a zero-size type with exactly one value. It is the implicit return
type of functions that do not return a meaningful value, and the type of
side-effecting expressions used as statements.

```fuse
fn log(msg: ref String) -> () {
    // ...
}

let result: () = log(ref s);
```

### 2.7 Never

`Never` (written `!` in some positions) is the type of expressions that do not
return. It is a subtype of every type.

```fuse
fn abort(msg: ref String) -> Never {
    // calls the runtime panic surface and does not return
    fuse_rt_panic(msg);
}

fn example(x: I32) -> I32 {
    if x < 0 {
        abort(ref "negative");  // type Never, valid in I32 position
    }
    return x;
}
```

### 2.8 Implementation contracts

#### Nominal type identity

Two nominal types are the same type if and only if they share:

- the same declared name, and
- the same defining symbol or defining module identity

Name-only equality is invalid in a multi-module compiler. Two types called
`Expr` from different modules are distinct types.

#### Primitive method registration

The checker must register the primitive method surface (see §24) before any
function body checking begins. Primitive method lookup must not depend on
user-declared impls.

#### `Never` subtyping

`Never` (`!`) is a subtype of every type. A diverging expression may appear in
any position that expects a value of any type; the compiler must accept the
expression without inventing a concrete fallback value. See §6 for the
divergence control-flow contract.

---

## 3. Compound Types

> Implementation status: SPECIFIED — W04 (TypeTable), W06 (type checking)

### 3.1 Tuples

A tuple groups a fixed number of values of possibly different types. Elements
are accessed by decimal index.

```fuse
let pair: (I32, Bool) = (42, true);
let x = pair.0;   // 42
let y = pair.1;   // true

// destructuring in let
let (a, b) = pair;
```

### 3.2 Arrays

An array is a fixed-length sequence of elements of a single type. The length is
part of the type.

```fuse
let xs: [I32; 3] = [1, 2, 3];
let first = xs[0];
let len = 3;     // known statically
```

### 3.3 Slices

A slice is a dynamically sized view over a contiguous sequence. Slices are
always behind a borrow.

```fuse
fn sum(values: ref [I32]) -> I32 {
    var total = 0;
    for v in values {
        total = total + v;
    }
    return total;
}

let arr = [1, 2, 3, 4];
let result = sum(ref arr);    // arr coerces to [I32]
```

### 3.4 Raw pointers

`Ptr[T]` is a raw pointer. It does not participate in ownership analysis and
may only be used in `unsafe` contexts.

```fuse
unsafe {
    let p: Ptr[I32] = allocate_i32();
    *p = 42;
    let v: I32 = *p;
}
```

### 3.5 Implementation contracts

#### Tuple numeric fields

When the receiver is a tuple type, a decimal field name such as `0` or `1` is
an index into the tuple, not a struct field name. This is the only legal tuple
field access form.

#### Raw pointers are not borrows

`Ptr[T]` does not participate in ownership analysis and is not implicitly
dereferenced. The backend must track `Ptr[T]` values in a separate pointer
category from borrow pointers (see backend representation contracts).

---

## 4. Variables and Bindings

> Implementation status: SPECIFIED — W05 (minimal `let`), W06 (full type checking)

### 4.1 Immutable binding (`let`)

`let` introduces an immutable binding. The bound value cannot be reassigned.

```fuse
let x = 10;
let y: I32 = 20;
// x = 5;  -- compile error: x is not mutable
```

### 4.2 Mutable binding (`var`)

`var` introduces a mutable binding. The binding can be reassigned. The type
must remain the same.

```fuse
var count = 0;
count = count + 1;
count += 1;
```

### 4.3 Type inference

The type is inferred from the initializing expression when not explicitly
annotated.

```fuse
let a = 42;             // I32 inferred
let b = 3.14;           // F64 inferred
let c = true;           // Bool inferred
let d = "hello";        // String inferred
```

### 4.4 Shadowing

A new binding with the same name shadows the previous one. Both may have
different types.

```fuse
let x = 5;
let x = x + 1;       // new binding, value 6
let x = "now a string";  // different type, shadows again
```

### 4.5 Explicit move

`move` transfers ownership out of a binding explicitly when the context requires
it.

```fuse
let s = String.from("hello");
let t = move s;       // s is moved into t; s is no longer accessible
```

---

## 5. Operators

> Implementation status: SPECIFIED — W06 (type checking), W15 (lowering)

### 5.1 Arithmetic

```fuse
let a = 10 + 3;   // 13
let b = 10 - 3;   // 7
let c = 10 * 3;   // 30
let d = 10 / 3;   // 3  (integer division)
let e = 10 % 3;   // 1
```

### 5.2 Comparison

Comparisons return `Bool`. Equality uses semantic trait-driven dispatch for
non-scalar types, not raw pointer equality.

```fuse
let eq  = 1 == 1;    // true
let ne  = 1 != 2;    // true
let lt  = 1 < 2;     // true
let le  = 1 <= 1;    // true
let gt  = 2 > 1;     // true
let ge  = 2 >= 2;    // true
```

### 5.3 Logical

Short-circuit evaluation applies to `&&` and `||`.

```fuse
let a = true && false;   // false
let b = true || false;   // true
let c = !true;           // false
```

### 5.4 Bitwise

```fuse
let a = 0b1100 & 0b1010;   // 0b1000
let b = 0b1100 | 0b1010;   // 0b1110
let c = 0b1100 ^ 0b1010;   // 0b0110
let d = !0b1100_u8;         // bitwise NOT
let e = 1 << 3;             // 8
let f = 16 >> 2;            // 4
```

### 5.5 Compound assignment

```fuse
var x = 10;
x += 5;    // 15
x -= 3;    // 12
x *= 2;    // 24
x /= 4;    // 6
x %= 4;    // 2
x &= 3;    // 2
x |= 1;    // 3
x ^= 1;    // 2
x <<= 1;   // 4
x >>= 1;   // 2
```

### 5.6 Optional chaining (`?.`)

`?.` is a single token. On a non-None/non-error value it continues the chain;
on absence or error it short-circuits and returns the absent/error value from
the enclosing scope.

```fuse
let name = user?.profile?.displayName;
```

### 5.7 Error propagation (`?`)

`?` on a `Result[T, E]` extracts `T` on success or returns `Err(e)` from the
enclosing function immediately. `?` on `Option[T]` extracts `T` or returns
`None`. See §14 for full semantics.

```fuse
fn read_config() -> Result[Config, IoError] {
    let path = find_config_path()?;    // returns Err early if absent
    let text = read_file(ref path)?;   // returns Err early on IO failure
    return parse_config(ref text);
}
```

### 5.8 Implementation contracts

#### Numeric widening

Binary operators between two numeric types in the same family are permitted
when the wider type can represent all values of the narrower. For example,
`I32 == ISize` is legal. Bitwise operators require matching signedness and
width.

#### Equality lowering behavioral contract

The lowerer must not emit raw backend equality for every type. For non-scalar
types, equality must lower through the type's equality semantics rather than
a plain C `==` or `!=`. A lowering that emits `==` for a struct type is a bug.

#### Optional chaining lowering

`expr?.field` lowers to a conditional read that short-circuits on absence or
error according to the operand type. `?.` must not lower as `?` applied to
`expr` followed by field access on the result.

---

## 6. Control Flow

> Implementation status: SPECIFIED — W15 (lowering)

### 6.1 `if` / `else if` / `else`

`if` is an expression. Both branches must have the same type when the result is
used.

```fuse
if x > 0 {
    log("positive");
} else if x < 0 {
    log("negative");
} else {
    log("zero");
}

// if as expression
let label = if x > 0 { "pos" } else { "non-pos" };
```

### 6.2 `while`

Loops while the condition is true.

```fuse
var i = 0;
while i < 10 {
    i += 1;
}
```

### 6.3 `loop`

An unconditional loop. Use `break` to exit, optionally with a value.

```fuse
var n = 0;
loop {
    n += 1;
    if n == 5 { break; }
}

// break with a value
let result = loop {
    n += 1;
    if n >= 10 { break n; }
};
```

### 6.4 `for` / `in`

Iterates over any value that implements the iterator protocol.

```fuse
for i in 0..10 {       // 0, 1, …, 9
    log(i);
}

for i in 0..=10 {      // 0, 1, …, 10 (inclusive)
    log(i);
}

let items = [1, 2, 3];
for item in items {
    log(item);
}
```

### 6.5 `break` and `continue`

`break` exits the innermost loop. `continue` skips to the next iteration.

```fuse
for i in 0..20 {
    if i % 2 == 0 { continue; }   // skip even
    if i > 9      { break;    }   // stop after 9
    log(i);
}
```

### 6.6 `return`

Returns a value from the enclosing function. `return` with no value returns
unit.

```fuse
fn find(haystack: ref [I32], needle: I32) -> Option[I32] {
    for item in haystack {
        if item == needle { return Some(item); }
    }
    return None;
}
```

### 6.7 Implementation contracts

#### Sealed control-flow blocks

Lowering of `return`, `break`, and `continue` must seal the current basic
block. Later control-flow construction must not treat the sealed block as a
reachable fallthrough predecessor.

#### Divergence is structural

After a diverging call, there is no continuing control-flow path. MIR must
model that structurally. The code generator must not synthesize fake
temporaries or fallback values that assume execution continues.

---

## 7. Pattern Matching

> Implementation status: SPECIFIED — W10 (match lowering), W08 (match on
> generic enum specializations)

`match` is an expression. Arms are tested in source order. The compiler
enforces exhaustiveness.

### 7.1 Literal patterns

```fuse
let msg = match code {
    0 { "ok" }
    1 { "not found" }
    2 { "permission denied" }
    _ { "unknown" }
};
```

### 7.2 Wildcard (`_`)

Matches any value without binding it.

```fuse
match pair {
    (0, _) { "first is zero" }
    (_, 0) { "second is zero" }
    _      { "neither" }
}
```

### 7.3 Binding patterns

Binds the matched value to a name.

```fuse
match value {
    0       { "zero" }
    n       { "got value: {n}" }
};
```

### 7.4 Enum constructor patterns

Destructures enum variants and binds their payload fields.

```fuse
enum Shape {
    Circle(F64),
    Rect(F64, F64),
    Point,
}

let area = match shape {
    Circle(r)    { 3.14159 * r * r }
    Rect(w, h)   { w * h }
    Point        { 0.0 }
};
```

### 7.5 Struct-variant patterns

```fuse
enum Event {
    KeyPress { key: Char, shift: Bool },
    Click    { x: I32, y: I32 },
}

match event {
    KeyPress { key: 'q', shift: _ } { quit(); }
    KeyPress { key: k, shift: s }   { handle_key(k, s); }
    Click { x, y }                  { handle_click(x, y); }
}
```

### 7.6 Tuple patterns

```fuse
let point = (3, 4);
match point {
    (0, 0) { "origin" }
    (x, 0) { "on x-axis" }
    (0, y) { "on y-axis" }
    (x, y) { "general" }
}
```

### 7.7 Nested patterns

```fuse
match result {
    Ok(Some(v)) { use(v); }
    Ok(None)    { default(); }
    Err(e)      { fail(e); }
}
```

### 7.8 Guards

A guard is an additional boolean condition on an arm.

```fuse
match n {
    x if x < 0  { "negative" }
    x if x == 0 { "zero" }
    x            { "positive" }
}
```

### 7.9 Implementation contracts

#### Structured patterns at HIR

`MatchArm` must carry structured `Pattern` nodes (`LiteralPat`, `BindPat`,
`ConstructorPat`, `WildcardPat`, `OrPat`, `RangePat`, `AtBindPat`). Storing
patterns as a text description at HIR is forbidden because it makes real
dispatch impossible at MIR.

#### Match behavioral contract

Given a `match` expression with N arms over an enum with discriminants:

- the generated code must evaluate the discriminant tag exactly once
- each arm must be tested in source order
- only the matching arm's body executes
- an arm containing a binding pattern must extract the payload value before
  entering the arm's body
- a match with N arms produces at least N-1 conditional branches in MIR;
  wildcard arms produce unconditional fallthrough jumps

A lowering that unconditionally jumps to the first arm is not a correct
implementation of `match`.

#### Exhaustiveness checking

The checker must verify that a `match` on a finite domain (enum, bool) covers
every case or contains a wildcard arm. Non-exhaustive `match` on such a domain
is a compile error. Unreachable arms (shadowed by an earlier wildcard or
equivalent) must produce a diagnostic.

#### Match proof program

```fuse
enum Color { Red, Green, Blue }

fn main() -> I32 {
    let c = Color.Green;
    match c {
        Red   { return 1; }
        Green { return 2; }
        Blue  { return 3; }
    }
}
```

Expected: exits with code 2.
E2E fixture: `tests/e2e/match_enum_dispatch.fuse`

---

## 8. Functions

> Implementation status: SPECIFIED — W06 (type checking), W15 (lowering)

### 8.1 Basic declaration

```fuse
fn add(a: I32, b: I32) -> I32 {
    return a + b;
}
```

### 8.2 Public functions

`pub` makes a function visible outside its module.

```fuse
pub fn greet(name: ref String) -> String {
    return String.from("hello, ") + name;
}
```

### 8.3 Implicit return

The last expression in a block, if it has no semicolon, is the return value.

```fuse
fn square(x: I32) -> I32 {
    x * x
}
```

### 8.4 Ownership-annotated parameters

Parameters may be borrowed (`ref`), mutably borrowed (`mutref`), or ownership-
transferred (`owned`). A plain parameter is passed by value.

```fuse
fn print_len(s: ref String) -> USize {
    return s.len();
}

fn append(s: mutref String, suffix: ref String) {
    s.push(ref suffix);
}

fn consume(s: owned String) -> USize {
    return s.len();
}

fn copy_int(x: I32) -> I32 {
    return x;           // I32 is a value type; ownership is implicit
}
```

### 8.5 Multiple return values via tuples

```fuse
fn min_max(values: ref [I32]) -> (I32, I32) {
    var lo = values[0];
    var hi = values[0];
    for v in values {
        if v < lo { lo = v; }
        if v > hi { hi = v; }
    }
    return (lo, hi);
}

let (lo, hi) = min_max(ref arr);
```

### 8.6 No-return functions

A function that never returns has return type `Never`.

```fuse
pub fn panic(msg: ref String) -> Never {
    fuse_rt_panic(ref msg);
}
```

### 8.7 Implementation contracts

#### Function type registration pre-pass

Every function declaration node, including impl methods and extern
declarations, must receive its function type before any function body is
checked.

The checker therefore requires two passes:

- Pass 1: register all function types
- Pass 2: check all function bodies

If impl methods only receive their type during body checking, their metadata
remains `Unknown` during lowering and code generation, which corrupts the
backend pipeline.

#### No function-type gaps after checking

After successful checking, no function declaration in checked HIR may retain
an unknown function type. This is a hard invariant, not a best-effort rule.

---

## 9. Methods and Impl Blocks

> Implementation status: SPECIFIED — W06 (type checking), W15 (lowering)

### 9.1 Inherent methods

Methods are defined inside `impl TypeName` blocks.

```fuse
struct Counter {
    value: I32,
}

impl Counter {
    // associated function (no receiver): acts like a constructor
    pub fn new() -> Counter {
        return Counter { value: 0 };
    }

    // immutable borrow of self
    pub fn get(ref self) -> I32 {
        return self.value;
    }

    // mutable borrow of self
    pub fn increment(mutref self) {
        self.value += 1;
    }

    // consuming self
    pub fn reset(owned self) -> Counter {
        return Counter { value: 0 };
    }
}
```

### 9.2 Calling methods

```fuse
var c = Counter.new();
c.increment();         // implicit mutref because c is a var binding
let v = c.get();
let c2 = c.reset();    // consumes c
```

### 9.3 Implicit `mutref` on mutable receivers

When a method receiver is `mutref self` and the call target is an existing
`var` binding, the `mutref` annotation is not required at the call site.

```fuse
var items = List[I32].new();
items.push(1);    // push takes mutref self; no explicit mutref needed
items.push(2);
```

### 9.4 Implementation contracts

#### Field access versus method-call disambiguation

The surface syntax `obj.name` is ambiguous between a data field read and a
method reference. The lowerer must disambiguate using position.

- If `obj.name` is the direct callee of a call expression, treat it as a
  method reference and emit a method call with `obj` as the first argument.
- Otherwise, treat it as a field access and emit the appropriate
  field-address or field-read logic.

Lowering `self.len()` as a field access is a backend bug.

#### Implicit `mutref` on mutable receivers

When a method receiver is declared as `mutref self`, the call site does not
need to spell `mutref` explicitly if the receiver is an existing mutable
binding (`var` or a `mutref` parameter).

---

## 10. Structs

> Implementation status: SPECIFIED — W06 (type checking), W15 (lowering)

### 10.1 Plain struct

Fields are private by default. `pub` fields are accessible outside the module.

```fuse
struct Point {
    pub x: F64,
    pub y: F64,
}
```

### 10.2 Struct literals

```fuse
let p = Point { x: 1.0, y: 2.0 };
let q = Point { x: 0.0, y: 0.0 };
```

### 10.3 Field access

```fuse
let px = p.x;
let py = p.y;
```

### 10.4 Mutable field update

```fuse
var origin = Point { x: 0.0, y: 0.0 };
origin.x = 3.0;
```

### 10.5 Nested structs

```fuse
struct Line {
    start: Point,
    end:   Point,
}

let line = Line {
    start: Point { x: 0.0, y: 0.0 },
    end:   Point { x: 1.0, y: 1.0 },
};

let x0 = line.start.x;
```

### 10.6 `@value` structs

`@value` opts the struct into auto-derived behavior for core traits (equality,
hashing, copy-by-value, formatting). All fields must themselves implement those
traits.

```fuse
@value struct Color {
    r: U8,
    g: U8,
    b: U8,
}

let red  = Color { r: 255, g: 0, b: 0 };
let red2 = Color { r: 255, g: 0, b: 0 };
let same = red == red2;    // true; derived equality
```

### 10.7 Implementation contracts

#### Struct literal disambiguation

`IDENT {` is not automatically a struct literal. It is a struct literal only
if the brace body syntactically looks like a field list, either empty or
beginning with `IDENT :`. Otherwise the identifier remains an expression and
the `{` opens the surrounding block.

#### `@value` auto-derivation

The checker must register auto-derived trait implementations for every
`@value` struct before body checking begins. If any field type does not
implement the required trait (equality, hashing, copy, formatting), the
`@value` annotation is a compile error.

---

## 11. Enums

> Implementation status: SPECIFIED — W06 (type checking), W15 (lowering),
> W08 (generic enum layouts)

### 11.1 Unit variants

```fuse
enum Direction { North, South, East, West }

let d = Direction.North;
```

### 11.2 Tuple-like variants

```fuse
enum Maybe {
    Just(I32),
    Nothing,
}

let v = Maybe.Just(42);
let n = Maybe.Nothing;
```

### 11.3 Struct-like variants

```fuse
enum Message {
    Quit,
    Move { x: I32, y: I32 },
    Write(String),
    ChangeColor(U8, U8, U8),
}

let m = Message.Move { x: 10, y: 20 };
```

### 11.4 Qualified variant access

Variants are hoisted into the enclosing module namespace. The qualified form
`EnumName.Variant` is always valid and required when disambiguation is needed.

```fuse
let d: Direction = North;              // bare form, unambiguous here
let d: Direction = Direction.North;    // qualified form, always valid
```

### 11.5 Matching enum variants

```fuse
match message {
    Message.Quit              { return; }
    Message.Move { x, y }    { move_to(x, y); }
    Message.Write(text)       { write(ref text); }
    Message.ChangeColor(r, g, b) { set_color(r, g, b); }
}
```

### 11.6 Implementation contracts

#### Enum variant hoisting

Enum variants are hoisted into the enclosing module namespace. The resolver
must detect conflicts between variants introduced by different enums in the
same module. Qualified access (`EnumName.Variant`) remains valid even when no
conflict exists.

#### Qualified enum variant resolution

The resolver must support `EnumName.Variant` in expression and pattern
position. Enum variants are hoisted to module scope, so qualified access is
required to make non-trivial code unambiguous.

#### Unit erasure is total

The unit type `()` has no runtime representation. If it appears in fields,
variant payloads, parameters, arguments, pattern bindings, or function
pointer typedefs, it is erased at every one of those sites. Partial erasure
is not allowed.

#### Enum layout behavioral contract

For a non-generic enum, the generated C struct must contain:

- an integer `_tag` field whose value uniquely identifies each variant
- payload fields for each variant that carries data

For a generic enum `E[T]`, each concrete specialization `E[I32]`, `E[Bool]`,
etc. must produce a distinct C struct with payload fields typed to the
concrete type argument. Unspecialized generic enum definitions must not be
emitted.

A wave that claims enum construction and destructuring work must include a
proof program that constructs a variant, matches on it, extracts the payload,
and returns a value that proves the correct arm executed.

---

## 12. Traits

> Implementation status: SPECIFIED — W06 (trait resolution)

### 12.1 Trait declaration

A trait declares method signatures. Implementations must provide all of them.

```fuse
pub trait Drawable {
    fn draw(ref self);
    fn bounding_box(ref self) -> (F64, F64, F64, F64);
}
```

### 12.2 Default method bodies

A trait may supply default method implementations that implementers can override.

```fuse
pub trait Greet {
    fn name(ref self) -> String;

    fn greet(ref self) -> String {
        return "Hello, " + self.name();
    }
}
```

### 12.3 Supertraits

A trait may require that implementers also implement another trait.

```fuse
pub trait Hashable: Equatable {
    fn hash(ref self) -> U64;
}
// any type implementing Hashable must also implement Equatable
```

### 12.4 Implementing a trait

```fuse
struct Circle {
    radius: F64,
    center: Point,
}

impl Drawable for Circle {
    fn draw(ref self) {
        // ...
    }

    fn bounding_box(ref self) -> (F64, F64, F64, F64) {
        let r = self.radius;
        return (
            self.center.x - r,
            self.center.y - r,
            self.center.x + r,
            self.center.y + r,
        );
    }
}
```

### 12.5 Trait bounds on parameters

```fuse
fn print_all[T: Drawable](items: ref [T]) {
    for item in items {
        item.draw();
    }
}
```

### 12.6 `Self` type

Inside a trait, `Self` refers to the concrete implementing type.

```fuse
pub trait Clone {
    fn clone(ref self) -> Self;
}

impl Clone for Point {
    fn clone(ref self) -> Self {
        return Point { x: self.x, y: self.y };
    }
}
```

### 12.7 Implementation contracts

#### Bound-chain method lookup

When a receiver is a type parameter with trait bounds, method lookup must
search:

1. the directly declared trait bounds
2. the supertraits of those bounds recursively

This is required so that a type parameter bounded by `Hashable` can resolve
methods declared on `Equatable` when `Hashable: Equatable`.

#### Trait parameter ABI casts

When a trait type appears in a parameter position, the backend may need to
cast the concrete pointer type to the trait-representation pointer type at
the call site. Omitting this cast is a backend ABI bug.

#### Coherence and orphan rules

An `impl` of a trait `T` for a type `U` is only permitted in a module that
owns either `T` or `U`. Two overlapping impls for the same `(T, U)` pair are
a compile error. This rule prevents incoherent method resolution across
crates and modules.

---

## 13. Generics

> Implementation status: SPECIFIED — W08

### 13.1 Generic functions

```fuse
fn identity[T](x: T) -> T {
    return x;
}

let a = identity[I32](42);
let b = identity[Bool](true);
```

### 13.2 Type inference at call sites

When the type argument can be inferred from the value argument or expected
return type, the explicit annotation may be omitted.

```fuse
let a = identity(42);      // T inferred as I32
let b = identity(true);    // T inferred as Bool
```

### 13.3 Generic structs

```fuse
struct Pair[A, B] {
    first:  A,
    second: B,
}

let p = Pair[I32, String] { first: 1, second: String.from("one") };
let n = p.first;
```

### 13.4 Generic enums

```fuse
enum Option[T] {
    Some(T),
    None,
}

enum Result[T, E] {
    Ok(T),
    Err(E),
}

let a: Option[I32] = Some(42);
let b: Option[I32] = None;
```

### 13.5 Generic traits

```fuse
pub trait Into[T] {
    fn into(self) -> T;
}

pub trait From[T] {
    fn from(value: T) -> Self;
}
```

### 13.6 Generic impl blocks

```fuse
impl[T] Option[T] {
    pub fn is_some(ref self) -> Bool {
        match self {
            Some(_) { return true; }
            None    { return false; }
        }
    }

    pub fn unwrap(owned self) -> T {
        match self {
            Some(v) { return v; }
            None    { panic("unwrap on None"); }
        }
    }
}
```

### 13.7 Trait bounds on generic parameters

```fuse
fn largest[T: Comparable](a: T, b: T) -> T {
    if a > b { return a; }
    return b;
}
```

### 13.8 Multiple type parameters and `where` clauses

`where` moves bounds out of the parameter list for readability.

```fuse
fn zip[A, B](left: ref [A], right: ref [B]) -> [(A, B)]
where
    A: Clone,
    B: Clone,
{
    // ...
}
```

### 13.9 Monomorphization

The compiler produces a concrete function for each distinct set of type
arguments. Generic originals are never emitted to the backend.

```fuse
// source: one generic function
fn wrap[T](x: T) -> Option[T] { return Some(x); }

// binary: two concrete specializations are generated
//   wrap__I32(x: I32) -> Option_I32
//   wrap__Bool(x: Bool) -> Option_Bool
let a = wrap[I32](1);
let b = wrap[Bool](true);
```

### 13.10 Implementation contracts

#### Valid specialization requires complete substitution

A specialization is valid if and only if every required type parameter has
been substituted with a concrete type. Required parameters include:

- the generic parameters declared by the function
- the generic parameters declared by its enclosing impl target, if any

Partial specialization must be rejected.

#### Recursive concreteness

A type is concrete only if it contains no unresolved type parameters anywhere
in its structure. Concreteness is recursive. `Option[T]` with `T` unresolved
is not concrete; `Option[I32]` is.

#### Specialization names include all type arguments

Mangled specialization names must distinguish all concrete type arguments.
`Option[ExprId]` and `Option[StmtId]` must not collide.

#### Only concrete generic instantiations are emitted

The backend emits only concrete instantiations. Unresolved generic types
must be filtered out before type emission.

#### Unresolved types are hard errors before codegen

If any MIR type reaching codegen is unresolved or unknown, the compiler must
emit a diagnostic and abort code generation. Substituting `Unknown` or `int`
is not allowed.

#### Contextual inference from expected type

When explicit type arguments are absent, generic arguments are inferred from:

1. value argument types
2. the expected result type from the surrounding expression context

`List.new()` in a field typed `List[Expr]` must infer `Expr` from context.

#### Explicit type arguments on zero-argument generic calls

Calls such as `size_of[T]()` and `marker[Int]()` carry type information even
when they have no value arguments. The monomorphizer must receive explicit
callee type arguments directly; a value-argument-only inference path is
incomplete.

#### Generic compilation behavioral contract

Given `fn identity[T](x: T) -> T { return x; }` and a call
`identity[I32](42)`:

- the generated C must contain a function named `Fuse_identity__I32` (or a
  deterministic equivalent) that takes `int32_t` and returns `int32_t`
- `main` must call `Fuse_identity__I32(42)`, not `identity` or any generic
  form
- calling `identity[Bool](true)` must produce a distinct function
  `Fuse_identity__Bool`
- the two specializations must not share generated code

#### Generic proof programs

```fuse
// tests/e2e/identity_generic.fuse
fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity[I32](42); }
// expected: exit code 42
```

```fuse
// tests/e2e/multiple_instantiations.fuse
fn first[T](a: T, b: T) -> T { return a; }
fn main() -> I32 {
    let x = first[I32](10, 20);
    let y = first[I32](3, 7);
    return x + y;
}
// expected: exit code 13
```

---

## 14. Ownership and Borrowing

> Implementation status: SPECIFIED — W09

### 14.1 Value semantics

By default, values are passed and returned by copy for small scalar types, or
by move for heap-allocated types.

```fuse
let x = 42;
let y = x;      // copy for scalar types
let s = String.from("hello");
let t = s;      // move: s is no longer accessible
```

### 14.2 Shared borrows (`ref`)

`ref` borrows a value immutably. Multiple shared borrows may coexist.

```fuse
fn length(s: ref String) -> USize {
    return s.len();
}

let s = String.from("world");
let n = length(ref s);    // s is still accessible after the call
```

### 14.3 Mutable borrows (`mutref`)

`mutref` borrows a value mutably. Only one mutable borrow may exist at a time.

```fuse
fn clear(s: mutref String) {
    s.clear_in_place();
}

var s = String.from("hello");
clear(mutref s);
```

### 14.4 Ownership transfer (`owned`)

`owned` transfers ownership of a heap value into the callee. The caller may not
use the value afterwards.

```fuse
fn take_string(s: owned String) -> USize {
    return s.len();   // s is dropped at end of function
}

let s = String.from("hi");
let n = take_string(owned s);
// s is no longer accessible here
```

### 14.5 Explicit move

`move` makes an ownership transfer explicit at the call site when the compiler
cannot infer it.

```fuse
let s = String.from("data");
store(move s);   // explicit move into store()
```

### 14.6 Deterministic destruction

Owned values are destroyed at the point of their last use, on every control
flow path. The compiler inserts drop calls; no garbage collector runs.

```fuse
fn example() {
    let handle = FileHandle.open("log.txt");
    process(ref handle);
    // handle is dropped here automatically — file is closed
}
```

### 14.7 The `Drop` trait

Types that need custom cleanup implement `Drop`. The compiler calls `drop`
at the point of destruction.

```fuse
pub trait Drop {
    fn drop(owned self);
}

impl Drop for FileHandle {
    fn drop(owned self) {
        fuse_rt_file_close(self.fd);
    }
}
```

### 14.8 Implementation contracts

#### Borrow lowering rule

`ref x` and `mutref x` must lower to `InstrBorrow` with a precise borrow
kind. They must not lower to a generic unary operator representation. The
backend must be able to distinguish borrow formation from other unary
expressions.

#### Escape rules

A borrowed value may not outlive its owner. Escaping borrows are rejected
unless the language construct explicitly transfers ownership or performs an
allowed move. A borrow may not be stored in a struct field (see §54).

#### Single liveness computation

Liveness is computed once per function and consumed by later passes. It
must not be recomputed opportunistically downstream.

#### Drop codegen behavioral contract

An owned local `x` of a type with a `Drop` implementation must, at the
point of its last use on every control flow path, cause the emission of:

```c
TypeName_drop(&_lN);
```

in the generated C. A comment `/* drop _lN */` is not a drop. A wave that
claims deterministic destruction is implemented must include a proof program
whose generated C contains this call, verified by inspection.

#### Drop metadata flow

The checker must record which types implement `Drop` in the type table or a
side table accessible to codegen. Without this flow, the backend cannot emit
correct destructor calls.

---

## 15. Closures

> Implementation status: SPECIFIED — W12 (lowering)

### 15.1 Closure expressions

A closure is an anonymous function that may capture variables from its
enclosing scope. Closures use `fn` syntax.

```fuse
let add = fn(x: I32, y: I32) -> I32 { x + y };
let result = add(3, 4);    // 7
```

### 15.2 Capture

Closures implicitly capture variables from the enclosing scope. The capture
mode follows the same ownership rules as explicit borrows and moves.

```fuse
let threshold = 10;
let is_big = fn(n: I32) -> Bool { n > threshold };   // captures threshold by ref

let base = String.from("prefix");
let prepend = fn(s: ref String) -> String {
    return base + s;    // captures base
};
```

### 15.3 Closures as arguments

Functions accept closures through trait-bounded parameters.

```fuse
fn filter[T](items: ref [T], pred: fn(ref T) -> Bool) -> [T] {
    var out: [T] = [];
    for item in items {
        if pred(ref item) { out.push(item); }
    }
    return out;
}

let evens = filter(ref numbers, fn(n: ref I32) -> Bool { *n % 2 == 0 });
```

### 15.4 Closures with `spawn`

Closures used with `spawn` must capture by value (or by `owned` transfer) so
they are self-contained.

```fuse
let msg = String.from("hello from thread");
spawn fn() {
    log(ref msg);
};
```

### 15.5 Implementation contracts

#### Capture analysis

Closure bodies must be scanned for references to outer variables before
lowering begins. Captured variables are collected and become fields of a
generated environment struct. The capture mode of each variable follows the
same ownership rules as explicit borrows and moves.

#### Closure lifting to MIR

Each closure must produce:

- a lifted MIR function taking the environment as an additional parameter
- a concrete environment struct type holding the captured variables
- a closure expression at the original site that initializes the environment
  and pairs it with the lifted function pointer

A lowering that returns `constUnit()` for closure expressions is a silent
miscompilation. See L011.

#### Closure callable integration

Closures implement the appropriate callable trait from §55
(`Fn` / `FnMut` / `FnOnce`) depending on the capture mode of the closure
body. A closure that calls a `FnOnce`-only operation through a captured
value cannot itself implement `Fn`.

---

## 16. Error Handling

> Implementation status: SPECIFIED — W11 (? lowering), W08 (? with generic
> Result)

### 16.1 `Option[T]`

`Option[T]` represents an optional value: either `Some(v)` or `None`.

```fuse
fn find_user(id: U64) -> Option[User] {
    if id == 0 { return None; }
    return Some(load_user(id));
}

match find_user(42) {
    Some(u) { greet(ref u); }
    None    { log("not found"); }
}
```

### 16.2 `Result[T, E]`

`Result[T, E]` represents an operation that may fail: either `Ok(v)` or
`Err(e)`.

```fuse
fn parse_int(s: ref String) -> Result[I32, ParseError] {
    // ...
}

match parse_int(ref input) {
    Ok(n)  { use(n); }
    Err(e) { report(ref e); }
}
```

### 16.3 `?` — error propagation

`?` on a `Result[T, E]` extracts `T` on success, or immediately returns
`Err(e)` from the enclosing function on failure. `?` on `Option[T]` extracts
`T` or immediately returns `None`.

The generated code always contains a branch: it is not a pass-through.

```fuse
fn load_config(path: ref String) -> Result[Config, IoError] {
    let text  = read_file(ref path)?;       // returns Err on failure
    let config = parse_config(ref text)?;   // returns Err on failure
    return Ok(config);
}
```

### 16.4 Chaining with `?.`

`?.` combines optional chaining and error propagation for nested optional
access.

```fuse
let city = user?.address?.city;   // None if any step is None
```

### 16.5 Combinators

`Option` and `Result` expose methods for common transformation patterns.

```fuse
let n: I32 = find_user(id)
    .map(fn(u: ref User) -> I32 { u.score })
    .unwrap_or(0);

let s: String = parse_int(ref input)
    .map(fn(n: I32) -> String { n.to_string() })
    .unwrap_or(String.from("invalid"));
```

### 16.6 Implementation contracts

#### `?` operator behavioral contract

Given a function `f` returning `Result[U, E]` and an expression `expr?`
where `expr: Result[T, E]`:

- if `expr` evaluates to `Ok(v)`, the `?` expression evaluates to `v: T`
  and execution continues normally
- if `expr` evaluates to `Err(e)`, the enclosing function immediately
  returns `Err(e): Result[U, E]` and does not execute subsequent statements

The generated code must contain a branch: a discriminant read on the
result's `_tag` field, a success path that extracts `_f0` as type `T`, and
an early-return path on the error path.

For `Option[T]`, the analogous rule applies: `Some(v)` continues with `v`;
`None` causes the enclosing function to return `None`.

A lowering that returns `Unknown`, passes through `expr` unchanged, or does
not emit a branch is not a correct implementation of `?`.

#### Error propagation proof program

```fuse
fn maybe_fail(fail: Bool) -> Result[I32, Bool] {
    if fail { return Err(true); }
    return Ok(42);
}

fn run(fail: Bool) -> Result[I32, Bool] {
    let v = maybe_fail(fail)?;
    return Ok(v + 1);
}

fn main() -> I32 {
    match run(false) {
        Ok(v)  { return v; }
        Err(_) { return 0; }
    }
}
```

Expected: exits with code 43.
E2E fixture: `tests/e2e/error_propagation.fuse`

---

## 17. Concurrency

> Implementation status: SPECIFIED — W07 (semantics), W16 (runtime ABI),
> W21 (hosted stdlib surface)

### 17.1 `spawn`

`spawn` creates a new concurrent thread of execution. It takes a closure and
returns a `ThreadHandle[T]` where `T` is the closure's return type. The
caller may discard the handle for fire-and-forget behavior or call
`handle.join()` to wait for the thread and retrieve its result. See §39 for
full handle semantics. Fuse does not use `async`/`await`.

```fuse
// handle discarded: fire-and-forget
spawn fn() {
    do_work();
};

// handle retained: join for result
let handle: ThreadHandle[I32] = spawn fn() -> I32 { return compute(); };
let value = handle.join().unwrap();
```

### 17.2 Channels (`Chan[T]`)

`Chan[T]` is the primary message-passing primitive between threads.

```fuse
import full.chan.Chan;

let ch: Chan[I32] = Chan.new[I32]();

// sender thread
spawn fn() {
    ch.send(42);
};

// receiver
let value = ch.recv();
```

### 17.3 Channel operations

```fuse
ch.send(value);          // blocks until receiver is ready
let v = ch.recv();       // blocks until a value arrives
ch.close();              // signals no more sends; recv returns None after drain
```

### 17.4 Shared mutable state (`Shared[T]`)

`Shared[T]` wraps a value with synchronization. Access requires acquiring the
lock.

```fuse
import full.sync.Shared;

let counter: Shared[I32] = Shared.new(0);

spawn fn() {
    let guard = counter.lock();
    *guard += 1;
};    // guard released when dropped
```

### 17.5 Lock ranking (`@rank`)

`@rank(N)` declares the acquisition order of synchronization primitives at
compile time. Acquiring a lock of rank N while holding a lock of rank ≥ N is a
compile error, preventing deadlock cycles.

```fuse
@rank(1) let db_lock:    Shared[Db]    = Shared.new(db);
@rank(2) let cache_lock: Shared[Cache] = Shared.new(cache);

// valid: acquire rank 1 first, then rank 2
let db    = db_lock.lock();
let cache = cache_lock.lock();

// compile error: rank 2 held before rank 1
// let cache = cache_lock.lock();
// let db    = db_lock.lock();
```

### 17.6 Implementation contracts

#### Spawn behavioral contract

`spawn expr` where `expr` is a closure or function value must lower to a
runtime call of the form `fuse_rt_thread_spawn(fn_ptr, env_ptr)` which
returns an opaque thread handle raw value. The handle is wrapped into a
`ThreadHandle[T]` in Fuse-level code. A lowering that calls `expr`
synchronously is not correct. A wave that claims spawn is implemented must
include a proof program that spawns a thread and observes an effect the
spawned thread produces.

When the surface-level result of `spawn` is discarded, the compiler may
still allocate a handle for lifetime correctness but need not emit a join.
When the result is bound, `join()` must lower to `fuse_rt_thread_join` on
that handle.

#### Channel type behavioral contract

`Chan[T]` must be represented as a distinct type kind in the type table
(`KindChannel` with element type `T`). Send and receive operations must be
type-checked against the element type. A send of `Bool` to a `Chan[I32]` is
a type error.

#### Channel operation lowering

`ch.send(v)`, `ch.recv()`, and `ch.close()` must lower to
`fuse_rt_chan_send`, `fuse_rt_chan_recv`, and `fuse_rt_chan_close` calls,
respectively. The send path must encode the element type in its codegen so
that the runtime boundary is monomorphic.

#### Lock ranking enforcement

`@rank(N)` constraints are enforced at compile time by the checker. A code
path that acquires a lock of rank `N` while statically holding a lock of
rank `≥ N` is a compile error. Enforcement must be structural, not
heuristic.

#### Send and Sync gating

`spawn` requires the closure's captured environment to be `Send` (§47).
`Shared[T]` requires `T: Send + Sync`. `Chan[T]` requires `T: Send`. The
checker must enforce these bounds before lowering; violations must not
reach codegen.

---

## 18. Modules and Imports

> Implementation status: SPECIFIED — W03

### 18.1 Module structure

Each source file is a module. The module path mirrors the directory path
relative to the source root.

```
src/
  util/
    math.fuse    →  module path: util.math
  main.fuse      →  module path: main
```

### 18.2 Importing a module

```fuse
import util.math;

let x = util.math.sqrt(2.0);
```

### 18.3 Importing a specific item

When the last segment is an item name inside a module, the item is imported
directly.

```fuse
import util.math.sqrt;
import core.result.Result;
import core.option.Option;

let x = sqrt(2.0);
let r: Result[I32, String] = Ok(42);
```

### 18.4 Import aliases

```fuse
import util.math as m;

let x = m.sqrt(2.0);
```

### 18.5 Visibility

`pub` makes a function, struct, enum, trait, or field visible outside its
module. Items without `pub` are private to the module.

```fuse
pub fn exported() { }
fn internal() { }

pub struct PublicStruct {
    pub visible_field:  I32,
    hidden_field:       I32,    // not accessible outside the module
}
```

### 18.6 Qualified enum variant access

Variants are hoisted to module scope. When two enums in the same module declare
variants with the same name, the qualified form disambiguates.

```fuse
import shapes.Shape;

let s = Shape.Circle(1.0);           // qualified
let s = Circle(1.0);                 // bare — only valid if unambiguous
```

### 18.7 Implementation contracts

#### Module-first import resolution

Resolve an import by first treating the full dotted path as a module path.
If no module exists at that path, retry by treating the final segment as an
item name inside the preceding module path.

#### Stdlib modules are checked like user modules

Stdlib modules must be type-checked in the same pass as user modules. Any
pass that skips stdlib bodies while still lowering or codegening them
violates the frontend-backend contract.

#### Import cycle detection

Cyclic imports must produce a diagnostic naming the cycle path. The
compiler must not hang or panic on a cyclic import graph.

---

## 19. Constants and Statics

> Implementation status: SPECIFIED — W06 (declaration and checking), W14
> (const expression evaluation)

### 19.1 `const`

Constants are evaluated at compile time. The type must be explicit. `const` may
appear at module scope or inside functions.

```fuse
const MAX_RETRIES: I32 = 5;
const PI: F64 = 3.141592653589793;
const GREETING: String = "hello";
```

### 19.2 `static`

Statics live for the entire program lifetime. Unlike `const`, a `static` has a
fixed address.

```fuse
static GLOBAL_COUNTER: I32 = 0;
```

---

## 20. Type Aliases

> Implementation status: SPECIFIED — W06

`type` introduces a transparent alias. The alias and the original type are
interchangeable.

```fuse
type Meters  = F64;
type Seconds = F64;
type NodeId  = U64;

fn distance(a: Meters, b: Meters) -> Meters {
    return (a - b).abs();
}

let d: Meters = distance(1.0, 4.0);
```

---

## 21. Iterators

> Implementation status: SPECIFIED — W19 (core stdlib; depends on associated
> types, §30)

### 21.1 `for` / `in` desugars to the iterator protocol

Any type that implements the `Iterator` trait can be used in a `for` loop.

```fuse
pub trait Iterator {
    type Item;
    fn next(mutref self) -> Option[Self.Item];
}
```

### 21.2 Implementing `Iterator`

```fuse
struct Range {
    current: I32,
    end:     I32,
}

impl Iterator for Range {
    type Item = I32;

    fn next(mutref self) -> Option[I32] {
        if self.current >= self.end { return None; }
        let v = self.current;
        self.current += 1;
        return Some(v);
    }
}

for n in Range { current: 0, end: 5 } {
    log(n);    // 0 1 2 3 4
}
```

### 21.3 Common iterator methods

Standard iterator adapters are methods on the `Iterator` trait.

```fuse
let doubled: [I32] = (0..5)
    .map(fn(n: I32) -> I32 { n * 2 })
    .collect();

let big: [I32] = items
    .filter(fn(n: ref I32) -> Bool { *n > 10 })
    .collect();

let total: I32 = (1..=100).fold(0, fn(acc: I32, n: I32) -> I32 { acc + n });
```

### 21.4 Implementation contracts

#### `for..in` desugaring

`for x in expr { body }` desugars to:

```fuse
var __iter = expr.into_iter();
loop {
    match __iter.next() {
        Some(x) { body }
        None    { break; }
    }
}
```

The desugaring is applied in lowering, not parsing. AST preserves the
`for..in` shape.

---

## 22. Extern and FFI

> Implementation status: SPECIFIED — W06 (declarations), W17 (codegen)

### 22.1 Extern declarations

`extern` declares a function or symbol implemented outside Fuse. No body is
provided.

```fuse
extern fn strlen(s: Ptr[U8]) -> USize;
extern fn malloc(size: USize) -> Ptr[U8];
extern fn free(ptr: Ptr[U8]);
```

### 22.2 Calling extern functions

All extern call sites are `unsafe`. A safe wrapper is the idiomatic way to
expose them.

```fuse
pub fn alloc(size: USize) -> Ptr[U8] {
    unsafe {
        return malloc(size);
    }
}
```

### 22.3 Runtime bridge naming convention

The Fuse runtime bridge uses the prefix `fuse_rt_{module}_{operation}`.

```fuse
extern fn fuse_rt_io_write(fd: I32, buf: Ptr[U8], len: USize) -> ISize;
extern fn fuse_rt_thread_spawn(func: Ptr[()], arg: Ptr[()]) -> I64;
extern fn fuse_rt_chan_send(ch: Ptr[()], value: Ptr[()]);
extern fn fuse_rt_chan_recv(ch: Ptr[()]) -> Ptr[()];
```

### 22.4 Extern statics

```fuse
extern static ERRNO: I32;
```

### 22.5 Implementation contracts

#### Runtime bridge naming

Runtime bridge functions use a stable naming convention rooted in the
runtime surface, typically `fuse_rt_{module}_{operation}`.

#### Unsafe call-site rule

Every FFI call site is `unsafe` unless the language surface explicitly
provides a safe wrapper. A safe wrapper must document and enforce its
safety invariants.

---

## 23. Unsafe

> Implementation status: SPECIFIED — W17 (backend), W19 (stdlib bridge files)

### 23.1 Unsafe blocks

Operations that require `unsafe` must appear inside an `unsafe` block. The
block makes the unsafety visible to reviewers.

```fuse
let raw: Ptr[I32] = unsafe { malloc(4) as Ptr[I32] };
```

### 23.2 What requires `unsafe`

- dereferencing a `Ptr[T]`
- raw pointer arithmetic
- calling an `extern` function
- unchecked indexing (when the language surface exposes it)
- any operation explicitly documented as unsafe

### 23.3 Dereferencing raw pointers

```fuse
let p: Ptr[I32] = get_pointer();
let v: I32 = unsafe { *p };
unsafe { *p = 99; }
```

### 23.4 Pointer arithmetic

```fuse
let base: Ptr[I32] = unsafe { allocate_array(10) };
let third: Ptr[I32] = unsafe { base.offset(2) };
let val: I32 = unsafe { *third };
```

### 23.5 Unsafe must not leak through safe APIs

A function with a safe signature must not perform unsafe operations whose
safety invariants the caller cannot reason about. Unsafe behavior must remain
visible at the use site or be formally justified inside a safe wrapper.

```fuse
// WRONG: hides unsafety behind a safe signature without documentation
fn get_item(p: Ptr[I32]) -> I32 {
    return *p;    // compile error: dereference requires unsafe block
}

// CORRECT: safe wrapper that documents its invariants
/// Safety: p must be a valid, non-null, aligned pointer to an I32.
pub fn read_ptr(p: Ptr[I32]) -> I32 {
    unsafe { return *p; }
}
```

### 23.6 Implementation contracts

#### Unsafe remains visible

Unsafe operations must remain visible at their use site. A safe surface
that silently performs unsafe behavior without documented invariants is
forbidden.

#### Unsafe bridge files are enumerated

New unsafe bridge files require explicit documentation updates in the
repository layout document. The core bridge surface is restricted to files
such as `stdlib/core/rt_bridge/alloc.fuse`, `stdlib/core/rt_bridge/panic.fuse`,
and `stdlib/core/rt_bridge/intrinsics.fuse`.

---

## 24. Primitive Methods

> Implementation status: SPECIFIED — W06

Primitive types expose methods intrinsically. These do not require explicit
`impl` blocks in user code.

```fuse
// integers
let a: I32 = -5;
let b = a.abs();              // 5
let c = a.min(0);             // -5
let d = a.max(0);             // 0
let e = a.toFloat();          // -5.0 as F64
let f = (42_i64).toInt();     // I64 to I64 (identity)

// floats
let x: F64 = -3.7;
let y = x.abs();              // 3.7
let z = x.floor();            // -4.0
let w = x.ceil();             // -3.0
let q = x.sqrt();             // NaN for negative inputs
let nan = (0.0 / 0.0).isNan();    // true
let inf = (1.0 / 0.0).isInfinite(); // true

// chars
let c: Char = 'A';
let n = c.toInt();            // 65 as U32
let is_letter = c.isLetter(); // true
let is_digit  = c.isDigit();  // false

// bools
let t = true;
let f = t.not();              // false
```

---

## 25. Numeric Conversions and Widening

> Implementation status: SPECIFIED — W06

Fuse does not perform implicit numeric conversions. Widening for operators in
the same numeric family (e.g. `I32 == ISize`) is permitted where the wider
type can represent all values of the narrower.

```fuse
let a: I32   = 10;
let b: ISize = 20;
let c = a == b;      // legal: same signed family, ISize is wider

// explicit conversion when needed across families
let d: U32  = 10;
let e: I32  = d.toInt() as I32;

let f: F64  = 3.14;
let g: I32  = f.toInt() as I32;   // truncates toward zero
```

---

## 26. Decorators

> Implementation status: `@value` SPECIFIED — W06; `@rank` SPECIFIED — W07;
> `@repr`, `@align` SPECIFIED — W06 (check), W17 (codegen); `@inline`, `@cold`
> SPECIFIED — W17; `@cfg` SPECIFIED — W03 (see §37, §41, §50)

Decorators appear on type declarations and synchronization primitives to
express compile-time properties.

### 26.1 `@value`

Auto-derives equality, hashing, copy, and formatting for a struct whose fields
all implement those traits.

```fuse
@value struct Rgb { r: U8, g: U8, b: U8 }

let a = Rgb { r: 255, g: 0, b: 0 };
let b = Rgb { r: 255, g: 0, b: 0 };
let same = a == b;   // true
```

### 26.2 `@rank`

Declares the lock-acquisition rank of a `Shared[T]`. See §17.5.

```fuse
@rank(1) let accounts: Shared[AccountMap] = Shared.new(AccountMap.new());
@rank(2) let audit:    Shared[AuditLog]   = Shared.new(AuditLog.new());
```

---

## 27. Complete Example

A small program that exercises ownership, generics, error handling, pattern
matching, and iteration together.

```fuse
import core.result.Result;
import core.option.Option;
import core.string.String;

@value struct ParseError {
    message: String,
}

fn parse_positive(s: ref String) -> Result[U32, ParseError] {
    let n = s.parse_u32()?;
    if n == 0 {
        return Err(ParseError { message: String.from("must be positive") });
    }
    return Ok(n);
}

fn sum_strings(inputs: ref [String]) -> Result[U32, ParseError] {
    var total: U32 = 0;
    for s in inputs {
        let n = parse_positive(ref s)?;
        total += n;
    }
    return Ok(total);
}

pub fn main() -> I32 {
    let inputs = [
        String.from("10"),
        String.from("20"),
        String.from("30"),
    ];

    match sum_strings(ref inputs) {
        Ok(total) {
            // prints 60
            log(total);
            return 0;
        }
        Err(e) {
            log(ref e.message);
            return 1;
        }
    }
}
```

---

## 28. Type Casting

> Implementation status: SPECIFIED — W06 (type checking), W15 (lowering)

> **Importance:** everyday need. Without explicit casts, numeric FFI and
> pointer work require unsafe gymnastics for every type boundary crossing.

`as` converts between numeric types and between pointer types. It is always
explicit and never implicit. Narrowing casts truncate; widening casts
sign-extend or zero-extend according to the source type.

```fuse
// numeric casts
let a: I32  = 1000;
let b: I16  = a as I16;        // truncates if out of range
let c: I64  = a as I64;        // widening, sign-extends
let d: U32  = a as U32;        // reinterprets bit pattern
let e: F64  = a as F64;        // integer to float
let f: I32  = 3.9f64 as I32;   // float to integer, truncates toward zero

// pointer casts (require unsafe when dereferencing)
let p: Ptr[U8]  = buf.as_ptr();
let q: Ptr[I32] = p as Ptr[I32];   // reinterpret pointer type

// usize <-> pointer (for pointer arithmetic)
let addr: USize = p as USize;
let p2: Ptr[U8] = addr as Ptr[U8];
```

### 28.1 Implementation contracts

#### Cast semantics

- Numeric widening casts sign-extend for signed source types and zero-extend
  for unsigned source types.
- Numeric narrowing casts truncate the bit pattern.
- Float-to-integer casts truncate toward zero; NaN and out-of-range values
  produce a platform-defined but deterministic result recorded in CI goldens.
- Pointer-to-pointer casts reinterpret bits and do not participate in
  ownership analysis.
- Pointer-to-integer casts produce `USize` / `ISize` representations of the
  address value. Integer-to-pointer casts are permitted only in `unsafe`
  contexts.
- There is no `as` cast between unrelated nominal types.

---

## 29. Function Pointers

> Implementation status: SPECIFIED — W06 (types), W12 (Fn auto-impl),
> W15 (lowering)

> **Importance:** high. Required for callbacks, dispatch tables, plugin
> interfaces, and passing behavior across FFI boundaries. Without function
> pointer types, closures and trait objects are the only abstraction — and
> neither crosses an FFI boundary.

A function pointer type is written `fn(ParamTypes) -> ReturnType`. It holds
the address of a function with no captured state.

```fuse
// function pointer type
let f: fn(I32, I32) -> I32 = add;

fn add(a: I32, b: I32) -> I32 { return a + b; }
fn mul(a: I32, b: I32) -> I32 { return a * b; }

// call through a function pointer
let result = f(3, 4);     // 7

// function pointer in a struct (dispatch table / vtable pattern)
struct MathOps {
    combine:  fn(I32, I32) -> I32,
    identity: fn() -> I32,
}

let add_ops = MathOps {
    combine:  add,
    identity: fn() -> I32 { return 0; },
};

let r = (add_ops.combine)(10, 20);    // 30

// array of function pointers (jump table)
let handlers: [fn(I32) -> I32; 3] = [negate, double, square];

fn dispatch(op: USize, x: I32) -> I32 {
    return handlers[op](x);
}

// passing a function pointer across FFI
extern fn qsort(
    base:    Ptr[()],
    n:       USize,
    size:    USize,
    compare: fn(Ptr[()], Ptr[()]) -> I32,
);

fn compare_i32(a: Ptr[()], b: Ptr[()]) -> I32 {
    unsafe {
        let x = *(a as Ptr[I32]);
        let y = *(b as Ptr[I32]);
        return x - y;
    }
}

unsafe {
    qsort(arr.as_ptr() as Ptr[()], arr.len(), size_of[I32](), compare_i32);
}
```

### 29.1 Implementation contracts

#### Function pointer ABI

A function pointer type `fn(A, B) -> R` has a stable C ABI representation
equivalent to the C function pointer type `R (*)(A, B)` (after Fuse-to-C
type mapping). Function pointers are not closures and do not carry captured
state. They implement `Fn` / `FnMut` / `FnOnce` automatically (§55).

Function pointer types across the FFI boundary must preserve the exact C
calling convention of the target platform.

---

## 30. Associated Types

> Implementation status: SPECIFIED — W06

> **Importance:** high. Without associated types, generic interfaces become
> verbose — every method must repeat type parameters. Associated types let a
> trait pin a related type once, keeping implementations readable and call
> sites clean. Iterators, allocators, and indexing all depend on them.

An associated type is declared with `type` inside a trait and defined in each
`impl` block. It is referenced as `Self.TypeName` or `T.TypeName` at use sites.

```fuse
// declaring an associated type
pub trait Container {
    type Item;
    fn get(ref self, index: USize) -> Option[Self.Item];
    fn len(ref self) -> USize;
}

// implementing: the impl pins the associated type
impl Container for Vec[I32] {
    type Item = I32;

    fn get(ref self, index: USize) -> Option[I32] {
        if index >= self.len() { return None; }
        return Some(self.data[index]);
    }

    fn len(ref self) -> USize { return self.count; }
}

// using the associated type in a generic bound
fn print_all[C: Container](c: ref C) {
    var i: USize = 0;
    while i < c.len() {
        match c.get(i) {
            Some(v) { log(v); }
            None    { }
        }
        i += 1;
    }
}

// constraining an associated type
fn sum_container[C](c: ref C) -> I32
where
    C: Container,
    C.Item = I32,
{
    var total = 0;
    var i: USize = 0;
    while i < c.len() {
        match c.get(i) {
            Some(v) { total += v; }
            None    { }
        }
        i += 1;
    }
    return total;
}
```

### 30.1 Implementation contracts

#### Associated type projection

`Self.TypeName` and `T.TypeName` (for a type parameter `T` with a trait
bound that declares the associated type) are projections that the checker
must resolve to the concrete associated type declared by the applicable
impl. Unresolved associated type projections are hard errors before codegen.

#### Associated type constraints in where clauses

`where T.Item = I32` constrains the associated type; the checker must
reject impls whose associated type does not match the constraint.

---

## 31. Additional Pattern Forms

> Implementation status: SPECIFIED — W10

> **Importance:** medium-high. Or-patterns and range patterns eliminate
> boilerplate repetition in match arms. Their absence forces multiple identical
> arms or guard expressions for what should be a single readable arm.

### 31.1 Or-patterns

Multiple patterns separated by `|` share a single arm body.

```fuse
match direction {
    North | South { return "vertical"; }
    East  | West  { return "horizontal"; }
}

match byte {
    0x09 | 0x0A | 0x0D | 0x20 { return true; }    // is whitespace
    _                          { return false; }
}
```

### 31.2 Range patterns

A range pattern matches any value within the range. Both inclusive (`..=`) and
half-open forms are supported in patterns.

```fuse
match score {
    90..=100 { "A" }
    80..=89  { "B" }
    70..=79  { "C" }
    60..=69  { "D" }
    _        { "F" }
}

match byte {
    0x41..=0x5A { "uppercase ASCII letter" }
    0x61..=0x7A { "lowercase ASCII letter" }
    0x30..=0x39 { "ASCII digit" }
    _           { "other" }
}
```

### 31.3 Binding with `@`

`name @ pattern` binds the matched value to `name` while also testing the
inner pattern.

```fuse
match value {
    n @ 1..=9  { log("single digit: {n}"); }
    n @ 10..=99 { log("double digit: {n}"); }
    n           { log("three or more digits: {n}"); }
}
```

### 31.4 Implementation contracts

#### Or-pattern lowering

`P1 | P2` matches when either inner pattern matches. Both inner patterns
must bind the same set of names with the same types. The lowerer produces
a disjunction of conditional branches that merge into the arm body.

#### Range pattern lowering

`lo..=hi` and `lo..hi` patterns lower to a pair of integer comparisons on
the discriminant value. The endpoints must be constant expressions of the
scrutinee's type.

#### `@` binding

`name @ pattern` binds the matched value to `name` while also testing the
inner pattern. The binding must be visible in both guard expressions and
the arm body.

---

## 32. Slice Range Indexing

> Implementation status: SPECIFIED — W15 (lowering)

> **Importance:** high. Sub-slicing is the core operation for parsing,
> streaming, and buffer management. Without range indexing, working with
> byte buffers and string data requires manual pointer arithmetic.

Ranges produce a slice view of the original without copying.

```fuse
let arr = [1, 2, 3, 4, 5];

let all   = arr[..];       // entire array as a slice
let first = arr[..3];      // [1, 2, 3]  — indices 0, 1, 2
let last  = arr[2..];      // [3, 4, 5]  — index 2 to end
let mid   = arr[1..4];     // [2, 3, 4]  — indices 1, 2, 3

// slices of slices
fn parse_header(buf: ref [U8]) -> ref [U8] {
    let end = find_newline(ref buf);
    return buf[..end];
}

// mutable sub-slice
fn zero_fill(buf: mutref [U8], from: USize, to: USize) {
    let region: mutref [U8] = buf[from..to];
    for b in region {
        b = 0;
    }
}
```

### 32.1 Implementation contracts

#### Slice formation

`arr[a..b]` lowers to a slice descriptor `{ ptr: base + a, len: b - a }`
that borrows from `arr`. The slice is a borrow; it does not own or copy
the underlying storage. Out-of-bounds ranges are a runtime panic in debug
builds and a bounds-checked error in release builds.

---

## 33. Overflow-Aware Arithmetic

> Implementation status: SPECIFIED — W15 (lowering), W17 (codegen default
> policy), W19 (stdlib methods)

> **Importance:** high for systems code. In debug builds, integer overflow
> is a programming error and should panic. In release builds, the behavior
> must be explicit. Wrapping arithmetic is required for hash functions,
> checksums, ring buffers, and any algorithm that intentionally uses modular
> arithmetic. Saturating arithmetic is required for audio, graphics, and
> signal processing. Checked arithmetic is required for safe parsing and
> protocol implementations. Without these, correct systems code requires
> manual bit masking.

```fuse
// checked: returns Option; None on overflow
let a: Option[I32] = 2_000_000_000_i32.checked_add(2_000_000_000);  // None

// wrapping: always succeeds, wraps on overflow (modular arithmetic)
let b: I32 = 2_000_000_000_i32.wrapping_add(2_000_000_000);  // -294967296

// saturating: clamps to min/max on overflow
let c: I32 = 2_000_000_000_i32.saturating_add(2_000_000_000);  // 2_147_483_647

// these exist for all four arithmetic operations
let d = x.wrapping_sub(y);
let e = x.wrapping_mul(y);
let f = x.wrapping_shl(n);    // wrapping shift

let g = x.saturating_sub(y);
let h = x.saturating_mul(y);

let i: Option[I32] = x.checked_sub(y);
let j: Option[I32] = x.checked_mul(y);
let k: Option[I32] = x.checked_div(y);    // also catches divide-by-zero
```

### 33.1 Implementation contracts

#### Default overflow behavior

Plain `+`, `-`, `*`, `/`, `%`, `<<`, `>>` on integer types panic on overflow
in debug builds. In release builds, the default is implementation-defined
per-target-profile but must be deterministic and documented. Code that
requires specific overflow behavior must use the explicit `checked_*`,
`wrapping_*`, or `saturating_*` methods.

#### Divide-by-zero

Integer divide-by-zero and modulo-by-zero panic in all build profiles.
Float divide-by-zero produces `∞` or `NaN` per IEEE 754. Neither form
produces undefined behavior.

---

## 34. Memory Intrinsics

> Implementation status: SPECIFIED — W14 (const-eval context), W17 (codegen
> emission), W19 (stdlib surface)

> **Importance:** critical for a systems language. `size_of` and `align_of`
> are required for every custom allocator, every FFI struct, every arena, and
> every unsafe collection. Their absence makes it impossible to correctly
> allocate or lay out memory without hardcoding sizes that will silently break
> on different targets or after struct changes.

```fuse
// size in bytes of a type
let s1 = size_of[I32]();      // 4
let s2 = size_of[F64]();      // 8
let s3 = size_of[Point]();    // depends on fields and padding

// alignment requirement of a type
let a1 = align_of[I32]();     // 4
let a2 = align_of[F64]();     // 8
let a3 = align_of[Point]();   // alignment of the most-aligned field

// size of a value (not just the type)
let p = Point { x: 1.0, y: 2.0 };
let s = size_of_val(ref p);   // same as size_of[Point]()

// practical: a typed bump allocator
fn alloc_typed[T](arena: mutref Arena) -> Ptr[T] {
    let size  = size_of[T]();
    let align = align_of[T]();
    return arena.alloc_raw(size, align) as Ptr[T];
}

// practical: copying raw bytes
unsafe {
    let src: Ptr[T] = get_source();
    let dst: Ptr[T] = alloc_typed[T](ref arena);
    copy_nonoverlapping(
        src as Ptr[U8],
        dst as Ptr[U8],
        size_of[T](),
    );
}
```

### 34.1 Implementation contracts

#### `size_of[T]` and `align_of[T]`

`size_of[T]()` and `align_of[T]()` return `USize` values computed at
compile time for every concrete type. They must be callable in `const`
contexts. Their arguments are explicit type arguments, not value arguments
(see §13, explicit type arguments on zero-argument generic calls).

#### `size_of_val`

`size_of_val(ref v)` returns the run-time size of the referent. For
statically sized types, it equals `size_of[T]()`. For slices and other
dynamically sized referents, it returns the actual storage size.

---

## 35. Null Pointers

> Implementation status: SPECIFIED — W17 (lowering), W19 (stdlib surface)

> **Importance:** high. Null pointers appear constantly in C APIs, OS
> interfaces, and optional return values from FFI. Without a way to express,
> test, and construct null, every FFI boundary requires workarounds.

```fuse
// constructing a null pointer
let null_p: Ptr[I32] = Ptr.null();

// testing for null
if p.is_null() {
    return Err(NullPointerError {});
}

// the canonical nullable pointer pattern: wrap in Option
fn find_symbol(name: ref String) -> Option[Ptr[()]] {
    let p: Ptr[()] = unsafe { dlsym(handle, name.as_ptr()) };
    if p.is_null() { return None; }
    return Some(p);
}

// passing null to C APIs
unsafe {
    let result = ffi_open(path.as_ptr(), Ptr.null());
}

// comparing pointers
let same = p == q;       // pointer equality
let is_null = p == Ptr.null[I32]();
```

### 35.1 Implementation contracts

#### `Ptr.null[T]()` lowering

`Ptr.null[T]()` lowers to the integer value `0` cast to `Ptr[T]`. It is
valid in `const` contexts and as a compile-time initializer. Dereferencing
a null pointer is undefined behavior (trapped in debug builds).

---

## 36. Variadic Extern Functions

> Implementation status: SPECIFIED — W06 (check), W17 (codegen)

> **Importance:** medium-high. Many foundational C APIs — `printf`, `scanf`,
> `open`, `ioctl`, `fcntl` — are variadic. Without variadic extern support,
> these cannot be declared correctly and calls to them require unsafe casts
> through a void pointer, losing all type safety at the boundary.

`...` in an extern parameter list marks a variadic function. The call site
passes additional arguments normally. Fuse does not support declaring variadic
Fuse functions, only calling variadic C functions.

```fuse
extern fn printf(fmt: Ptr[U8], ...) -> I32;
extern fn sprintf(buf: Ptr[U8], fmt: Ptr[U8], ...) -> I32;
extern fn open(path: Ptr[U8], flags: I32, ...) -> I32;
extern fn ioctl(fd: I32, request: U64, ...) -> I32;

// calling variadic extern functions
unsafe {
    printf(c"hello %s, you are %d years old\n", name.as_ptr(), age);
    let fd = open(c"/dev/null", O_RDONLY);
}
```

### 36.1 Implementation contracts

#### Variadic call ABI

Variadic calls follow the host platform's C calling convention for variadic
functions (typically meaning floats are promoted to doubles and short
integers are promoted to ints at the call site). Fuse does not declare
variadic Fuse functions; only calling variadic C functions is supported.

---

## 37. Struct Layout Control

> Implementation status: SPECIFIED — W06 (check), W17 (codegen)

> **Importance:** non-negotiable for FFI. Without layout control, structs
> passed to or returned from C functions have undefined field ordering and
> padding, making the FFI ABI wrong in ways that are silent and
> machine-dependent. SIMD and DMA work requires specific alignment guarantees
> that the compiler will not provide without explicit instruction.

### 37.1 `@repr(C)` — C-compatible layout

Fields are laid out in source order with C-compatible padding rules. Required
for any struct shared with C code.

```fuse
@repr(C)
struct IpHeader {
    version_ihl:  U8,
    tos:          U8,
    total_length: U16,
    id:           U16,
    frag_offset:  U16,
    ttl:          U8,
    protocol:     U8,
    checksum:     U16,
    src_addr:     U32,
    dst_addr:     U32,
}

// safe to pass to C networking code
extern fn send_ip_packet(hdr: Ptr[IpHeader], payload: Ptr[U8], len: USize) -> I32;
```

### 37.2 `@repr(packed)` — no padding

All fields are packed with no alignment padding between them. Required for
network protocol headers, file format structs, and hardware register maps.

```fuse
@repr(packed)
struct EthernetFrame {
    dst_mac:   [U8; 6],
    src_mac:   [U8; 6],
    ethertype: U16,
}
// size_of[EthernetFrame]() == 14, regardless of alignment requirements
```

### 37.3 `@align(N)` — explicit alignment

Forces the struct to be aligned to at least `N` bytes. Required for SIMD
types (16 or 32 byte alignment), DMA buffers (page alignment), and
cache-line-aligned data structures.

```fuse
@align(16)
struct SimdVector {
    data: [F32; 4],
}

@align(64)    // one cache line
struct CacheLinePadded {
    counter: I64,
    _pad:    [U8; 56],
}
```

### 37.4 `@repr(u8)` / `@repr(i32)` — explicit enum discriminant type

Controls the underlying integer type of an enum's discriminant. Required when
a Fuse enum is used in an FFI context that expects a specific integer type.

```fuse
@repr(U8)
enum Status {
    Ok    = 0,
    Error = 1,
    Retry = 2,
}

@repr(I32)
enum OsError {
    NotFound    = -2,
    Permission  = -1,
    Success     =  0,
}
```

### 37.5 Implementation contracts

#### `@repr(C)` layout

Fields are laid out in declaration order with C ABI padding rules for the
target platform. Fuse-specific reordering optimizations are disabled for
these types.

#### `@repr(packed)` layout

No padding is inserted between fields. Dereferencing a misaligned field
may require an unaligned read; the compiler must emit unaligned access
instructions on targets where natural alignment is required.

#### `@align(N)` validation

`N` must be a power of two at least as large as the type's natural
alignment. The compiler must reject values smaller than the natural
alignment.

#### `@repr(Ixx)` / `@repr(Uxx)` discriminant

For enums, this annotation pins the discriminant's integer type. Variants
with explicit values must fit within the chosen integer range; overflow is
a compile error.

---

## 38. Newtype Pattern

> Implementation status: SPECIFIED — W06 (single-field struct form)

> **Importance:** medium-high. Newtypes prevent units-of-measure bugs and
> accidental API misuse at zero runtime cost. Passing a raw `I32` as a user
> ID when a function expects a port number is a type error, not a runtime
> crash, when distinct newtypes wrap the same primitive.

A newtype is a struct with one field. The compiler treats it as a distinct type
from the wrapped value.

```fuse
struct UserId(U64);
struct SessionId(U64);
struct Milliseconds(I64);
struct Bytes(USize);

// these are now distinct types — cannot be mixed accidentally
fn get_user(id: UserId) -> Option[User] { ... }
fn get_session(id: SessionId) -> Option[Session] { ... }

let uid = UserId(42);
let sid = SessionId(42);

get_user(uid);    // ok
get_user(sid);    // compile error: expected UserId, got SessionId

// impl methods on newtypes
impl Milliseconds {
    pub fn to_seconds(ref self) -> F64 {
        return self.0 as F64 / 1000.0;
    }
}

impl Bytes {
    pub fn kilobytes(ref self) -> F64 {
        return self.0 as F64 / 1024.0;
    }
}

let t = Milliseconds(3500);
let s = t.to_seconds();    // 3.5
```

---

## 39. Thread Handles and Joining

> Implementation status: SPECIFIED — W07 (type surface), W16 (runtime)

> **Importance:** high. `spawn` without a join handle is fire-and-forget —
> useful for background tasks but insufficient when the calling thread must
> wait for a result, propagate errors from the spawned thread, or ensure
> resources are cleaned up before proceeding.

`spawn` returns a `ThreadHandle[T]` where `T` is the return type of the
closure. Calling `join()` blocks until the thread finishes and returns the
result.

```fuse
import full.thread.ThreadHandle;

// spawn returns a handle
let handle: ThreadHandle[I32] = spawn fn() -> I32 {
    return expensive_computation();
};

// do other work while the thread runs
let local_result = do_local_work();

// wait for the thread and get its result
let thread_result: Result[I32, ThreadError] = handle.join();

match thread_result {
    Ok(v)  { use(v + local_result); }
    Err(e) { log("thread panicked"); }
}

// parallel map pattern
fn par_map[T, U](items: ref [T], f: fn(ref T) -> U) -> [U] {
    let handles: [ThreadHandle[U]] = items
        .map(fn(item: ref T) -> ThreadHandle[U] {
            spawn fn() -> U { return f(ref item); }
        })
        .collect();

    return handles
        .map(fn(h: ThreadHandle[U]) -> U { h.join().unwrap() })
        .collect();
}
```

### 39.1 Implementation contracts

#### `ThreadHandle[T]` layout

`ThreadHandle[T]` is an owned handle wrapping an opaque runtime thread
identifier (typically `I64` from `fuse_rt_thread_spawn`). Dropping a
`ThreadHandle[T]` without calling `join()` detaches the thread; the runtime
continues to execute it. `T` is the closure's return type.

#### `join()` lowering

`handle.join()` lowers to `fuse_rt_thread_join(handle.raw())`, which blocks
until the thread completes and returns a `Result[T, ThreadError]`. `Err`
cases include thread panic and runtime-level failures.

#### Send bound on captured environment

The closure passed to `spawn` must capture only `Send` values (§47). The
checker must reject a `spawn` whose closure environment contains a
non-`Send` type.

---

## 40. Compiler Intrinsics

> Implementation status: SPECIFIED — W17

> **Importance:** high for systems code. Intrinsics expose CPU and compiler
> capabilities that have no equivalent in ordinary source code: signaling
> unreachable paths (enables dead-code elimination), hinting branch
> probability (for hot path optimization), memory barriers (for lock-free
> data structures and device driver memory ordering), and prefetching.

```fuse
// unreachable: tells the optimizer this path cannot be taken
// undefined behavior if it actually is reached — use only when provably dead
fn get_nonzero(x: I32) -> I32 {
    if x == 0 { unreachable(); }
    return x;
}

// likely / unlikely: branch prediction hints
fn process(items: ref [Item]) {
    for item in items {
        if likely(item.is_valid()) {
            fast_path(ref item);
        } else {
            slow_path(ref item);
        }
    }
}

fn open_file(path: ref String) -> Result[File, IoError] {
    let fd = unsafe { fuse_rt_io_open(path.as_ptr()) };
    if unlikely(fd < 0) {
        return Err(IoError.from_errno());
    }
    return Ok(File { fd });
}

// memory barriers: required for lock-free programming and device drivers
fn publish_value(slot: mutref I32, value: I32) {
    *slot = value;
    fence(Ordering.Release);    // all writes before this are visible to readers
}

fn read_value(slot: ref I32) -> I32 {
    fence(Ordering.Acquire);    // all reads after this see writes before the Release
    return *slot;
}

// prefetch: hint the cache subsystem to load memory before it's needed
fn process_large_array(data: ref [F64]) {
    for i in 0..data.len() {
        if i + 16 < data.len() {
            prefetch(ref data[i + 16]);    // prefetch 16 elements ahead
        }
        process(data[i]);
    }
}

// assume: tells the optimizer a condition is always true
fn safe_divide(a: I32, b: I32) -> I32 {
    assume(b != 0);    // optimizer may rely on this; UB if false
    return a / b;
}
```

### 40.1 Implementation contracts

#### Intrinsic surface

The compiler recognizes intrinsic functions by name: `unreachable`,
`likely`, `unlikely`, `fence`, `prefetch`, `assume`. They are resolved to
target-specific emission rather than ordinary function calls. Each
intrinsic has a stable signature documented here; user code must not
declare functions with the same name at module scope.

#### Undefined behavior on violated assumptions

Reaching `unreachable()` or violating an `assume(cond)` where `cond` is
false is undefined behavior. Optimizers may rely on these paths being
impossible. Debug builds must trap on these violations; release builds
may elide the check.

---

## 41. Inline Annotations

> Implementation status: SPECIFIED — W17

> **Importance:** medium. Inlining decisions are normally left to the
> optimizer, but systems code has legitimate reasons to override them: hot
> inner loops should be inlined to eliminate call overhead; large functions
> called in many places should not be inlined to avoid code bloat; and some
> functions must never be inlined for security or stack-size reasons. Without
> annotations, the programmer has no recourse when the optimizer makes the
> wrong choice for a performance-critical path.

```fuse
@inline
fn fast_min(a: I32, b: I32) -> I32 {
    if a < b { return a; }
    return b;
}

@inline(always)
fn hot_path_helper(x: I32) -> I32 {
    return x * x + x;
}

@inline(never)
fn large_cold_function(data: ref [U8]) -> Result[Report, Error] {
    // called rarely; should not be inlined at every call site
    // ...
}

// @cold marks a function as unlikely to be called (affects surrounding layout)
@cold
fn handle_fatal_error(e: ref Error) -> Never {
    log(ref e.message);
    abort();
}
```

### 41.1 Implementation contracts

#### Inline directives

- `@inline` is a hint; the compiler may inline or not.
- `@inline(always)` requires the body to be inlined at every call site
  unless doing so would violate soundness (recursion, ABI boundary).
- `@inline(never)` forbids inlining.
- `@cold` marks the function as rarely called; the compiler may place the
  body in a cold section of the binary.

Conflicting annotations (e.g. `@inline(always)` on a recursive function)
must produce a diagnostic rather than silently fall back.

---

## 42. Strings and Raw Bytes

> Implementation status: SPECIFIED — W19 (core stdlib)

> **Importance:** high. Every system that reads files, handles network data,
> or interfaces with C needs to move between text and raw byte representations.
> The distinction between an owned UTF-8 string, a string slice, and a raw
> byte buffer is not cosmetic — each has a different memory representation,
> ownership model, and set of valid operations.

### 42.1 `String` — owned UTF-8

An owned, heap-allocated, valid UTF-8 string.

```fuse
let s = String.from("hello");
let t = String.with_capacity(64);
let u = s + " world";         // concatenation, produces a new String
let len = s.len();            // length in bytes (not codepoints)
let chars = s.char_count();   // number of Unicode scalar values
```

### 42.2 `[U8]` — raw byte slice

No encoding guarantee. Use for binary data, network buffers, file contents.

```fuse
fn hash(data: ref [U8]) -> U64 {
    var h: U64 = 0xcbf29ce484222325;
    for byte in data {
        h ^= byte as U64;
        h = h.wrapping_mul(0x00000100000001b3);
    }
    return h;
}
```

### 42.3 C strings

`c"..."` is a null-terminated byte string literal for FFI. Its type is
`Ptr[U8]` and it is valid to pass directly to C functions.

```fuse
extern fn puts(s: Ptr[U8]) -> I32;

unsafe {
    puts(c"hello from Fuse\n");
}

// converting a Fuse String to a C-compatible pointer
let s = String.from("hello");
let p: Ptr[U8] = s.as_c_str();   // valid as long as s is alive
unsafe { puts(p); }
```

### 42.4 Conversions

```fuse
// String -> [U8]
let bytes: ref [U8] = s.as_bytes();

// [U8] -> String (may fail if not valid UTF-8)
let result: Result[String, Utf8Error] = String.from_utf8(ref bytes);

// String -> raw Ptr[U8] for FFI (null-terminated)
let ptr: Ptr[U8] = s.as_c_str();

// raw bytes -> String slice (unsafe: caller guarantees valid UTF-8)
let s: ref String = unsafe { String.from_utf8_unchecked(ref bytes) };
```

### 42.5 Implementation contracts

#### `c"..."` literal type

`c"..."` is a lexical form that produces a null-terminated byte literal
of type `Ptr[U8]` with `'static` storage duration. The literal is emitted
as a read-only byte array in the generated object file. Its contents are
the literal bytes followed by a single `0` byte; embedded NUL bytes in the
source are a compile error.

#### `String` UTF-8 invariant

A `String` value must contain valid UTF-8 at all times. Construction via
`String.from_utf8` returns `Result`; `String.from_utf8_unchecked` is
`unsafe` and may create a value that violates the invariant, with
undefined behavior for downstream operations that assume UTF-8.

---

## 43. Formatting and Output

> Implementation status: SPECIFIED — W19 (core stdlib)

> **Importance:** medium-high. Every program needs output. Without a
> formatting system, printing anything beyond raw bytes requires manual string
> construction through FFI — which is both tedious and error-prone.

### 43.1 The `Format` trait

Types implement `Format` to describe how they render to a formatter.

```fuse
pub trait Format {
    fn fmt(ref self, f: mutref Formatter);
}

impl Format for Point {
    fn fmt(ref self, f: mutref Formatter) {
        f.write("Point(");
        self.x.fmt(ref f);
        f.write(", ");
        self.y.fmt(ref f);
        f.write(")");
    }
}
```

### 43.2 Format strings

`format!(...)` produces a `String`. `print!(...)` writes to stdout.
`eprint!(...)` writes to stderr.

```fuse
let s = format!("x={}, y={}", point.x, point.y);
print!("result: {}\n", value);
eprint!("error: {}\n", e);

// padding and alignment
let s = format!("{:>10}", "right");    // right-aligned in 10 chars
let s = format!("{:<10}", "left");     // left-aligned
let s = format!("{:0>8}", 42);        // zero-padded to 8 chars: "00000042"

// numeric bases
let s = format!("{:x}", 255);         // "ff"
let s = format!("{:X}", 255);         // "FF"
let s = format!("{:b}", 42);          // "101010"
let s = format!("{:o}", 8);           // "10"
```

### 43.3 Debug formatting

`@value` structs and standard library types automatically implement a debug
format for development output.

```fuse
@value struct Color { r: U8, g: U8, b: U8 }

let c = Color { r: 255, g: 128, b: 0 };
print!("{:?}\n", c);    // Color { r: 255, g: 128, b: 0 }
```

### 43.4 Implementation contracts

#### `format!` / `print!` / `eprint!` expansion

These are compiler-recognized macros rather than ordinary functions. The
format string is parsed at compile time; each `{}` placeholder is matched
to a positional argument whose type must implement `Format`. A mismatch
between placeholders and arguments is a compile error.

---

## 44. Panics and Abort

> Implementation status: SPECIFIED — W16 (runtime) with diagnostic-emitting
> surface in W06

> **Importance:** high. A systems language must define what happens when an
> invariant is violated. Panics handle programmer errors (out-of-bounds
> access, unwrap on None); abort handles unrecoverable failures. Without clear
> semantics, error-handling discipline breaks down.

### 44.1 `panic`

`panic` signals an unrecoverable programmer error. The runtime prints a
message and terminates the process. Panics do not unwind — there is no
exception handling in Fuse.

```fuse
fn get(slice: ref [I32], index: USize) -> I32 {
    if index >= slice.len() {
        panic("index out of bounds: {index} >= {slice.len()}");
    }
    return slice[index];
}

// panic in option/result combinators
let v = opt.unwrap();           // panics with "unwrap on None" if None
let v = result.expect("msg");   // panics with "msg: <error>" if Err
```

### 44.2 `abort`

`abort` immediately terminates the process without any message or cleanup.
Use in situations where even panic output cannot be trusted (signal handlers,
allocator failure, stack overflow recovery).

```fuse
fn handle_oom() -> Never {
    // cannot allocate, cannot print, cannot do anything
    abort();
}
```

### 44.3 `assert` and `debug_assert`

`assert!(cond)` panics if the condition is false. `debug_assert!(cond)` is
compiled out in release builds.

```fuse
assert!(index < len, "index {index} out of range {len}");
assert!(ptr != Ptr.null(), "null pointer in invariant-checked path");

debug_assert!(self.is_valid(), "struct invariant violated");
```

### 44.4 Implementation contracts

#### Panic lowering

`panic(msg)` lowers to `fuse_rt_panic(msg.as_ptr(), msg.len())`. Panics do
not unwind; the runtime prints the message to stderr and terminates the
process with a non-zero exit code. The calling convention assumes the
runtime never returns.

#### Abort lowering

`abort()` lowers to `fuse_rt_abort()`, which is declared as a
`Never`-returning function. It must not invoke drop cleanup; it
immediately terminates the process.

#### `assert!` / `debug_assert!`

`assert!(cond, msg)` expands to an `if !cond { panic(msg); }` at the use
site. `debug_assert!` expands to the same in debug builds and expands to
nothing in release builds. Their arguments must be evaluable without side
effects observable after the `debug_assert!` is elided.

---

## 45. Struct Update Syntax

> Implementation status: SPECIFIED — W15 (lowering)

> **Importance:** low-medium. Useful for constructing modified copies of
> structs without naming every unchanged field — common in configuration
> objects and builder patterns.

`..other` at the end of a struct literal fills remaining fields from `other`.

```fuse
let default_config = Config {
    threads:    4,
    buffer_size: 4096,
    timeout_ms:  5000,
    verbose:     false,
};

let fast_config = Config {
    threads:    16,
    buffer_size: 65536,
    ..default_config    // timeout_ms and verbose come from default_config
};

let quiet_config = Config {
    verbose: false,
    ..default_config
};
```

### 45.1 Implementation contracts

#### Spread ordering

In `Config { field_a: v, ..base }`, explicit field assignments take
precedence over the spread source. Each field in the resulting struct is
initialized from either an explicit assignment or from the spread source,
but not both. Moves from `base` apply only to fields not otherwise
assigned; fields assigned explicitly do not consume `base`.

---

## 46. Compile-Time Functions (`const fn`)

> Implementation status: SPECIFIED — W14

> **Importance:** medium. Compile-time evaluation eliminates entire classes
> of magic constants, enables lookup-table generation, and allows type-level
> computations to be verified at build time rather than discovered at runtime.
> Important for embedded targets where ROM-resident tables must be generated
> correctly and for cryptographic implementations with compile-time key
> schedules.

`const fn` functions may be called in constant contexts (const/static
initializers, array lengths, enum discriminants). They may only use operations
that are evaluable at compile time.

```fuse
const fn factorial(n: U64) -> U64 {
    if n == 0 { return 1; }
    return n * factorial(n - 1);
}

const FACT_10: U64 = factorial(10);    // evaluated at compile time: 3628800

const fn kilobytes(n: USize) -> USize { return n * 1024; }
const fn megabytes(n: USize) -> USize { return kilobytes(n) * 1024; }

const BUFFER_SIZE: USize = megabytes(4);    // 4194304

// compile-time lookup table
const fn make_sin_table() -> [F32; 256] {
    // ... generates a table of sin values at compile time
}
const SIN_TABLE: [F32; 256] = make_sin_table();
```

### 46.1 Implementation contracts

#### Constant evaluation surface

A `const fn` body may use:

- literals, arithmetic, comparison, and bitwise operations on integer and
  boolean values
- pattern matching, `if`, `loop`, `while` with constant control flow
- calls to other `const fn` functions
- struct and tuple construction and destructuring
- array and slice indexing with constant indices

A `const fn` body may not use:

- FFI calls
- allocation or deallocation
- thread operations
- non-`const` function calls
- interior mutability

#### Constant evaluator

The compiler includes a const evaluator that runs over checked HIR. The
evaluator is deterministic: the same input always produces the same output
bit-for-bit.

---

## 47. Marker Traits

> Implementation status: SPECIFIED — W07

> **Importance:** high for safe concurrent and systems code. Marker traits
> carry no methods — they categorize types by a property the compiler enforces.
> `Send` and `Sync` equivalents prevent data races at the type level: you
> cannot send a non-thread-safe type to another thread, and the compiler
> rejects the attempt. Without marker traits, thread safety is a convention,
> not an invariant.

A marker trait has no methods. It is implemented for types that possess the
corresponding property.

```fuse
// Send: safe to transfer ownership to another thread
pub trait Send { }

// Sync: safe to share a reference across threads
pub trait Sync { }

// Copy: value can be duplicated by copying bytes (no Drop impl allowed)
pub trait Copy { }

// these are automatically implemented for types whose fields implement them
// primitive types are Send + Sync + Copy
// types containing Ptr[T] are not automatically Send or Sync

// spawn requires Send
fn spawn_with[T: Send](value: T, f: fn(T)) -> ThreadHandle[()]  {
    return spawn fn() { f(value); };
}

// Shared[T] requires T: Send + Sync
// Chan[T] requires T: Send

// opting out of automatic implementation
struct NonSendResource {
    handle: Ptr[()],    // raw pointer — not Send
}

// the compiler prevents this:
// let h = NonSendResource { handle: get_handle() };
// spawn fn() { use(h); };   // compile error: NonSendResource is not Send
```

### 47.1 Implementation contracts

#### Auto-implementation rules

- `Send` is auto-implemented for a type `T` if every field of `T` is
  `Send`. Primitive scalar types are `Send`. `Ptr[U]`, `Cell[U]`,
  `RefCell[U]` are not `Send`.
- `Sync` is auto-implemented for a type `T` if every field of `T` is
  `Sync`. Primitive scalar types are `Sync`. `Cell[U]`, `RefCell[U]`
  are not `Sync`. `Shared[U]` is `Sync` when `U: Send`.
- `Copy` is auto-implemented for a type `T` if every field of `T` is
  `Copy` and `T` does not implement `Drop`. A manual impl of `Drop` for
  a type that would otherwise be `Copy` removes the auto `Copy` impl.

#### Borrow types are not Send or Sync

Borrow types `ref T` and `mutref T` are never `Send` and never `Sync`,
regardless of whether `T` itself implements those traits. Borrows are
scope-local by §54 and cannot outlive their owner; a borrow that crossed a
thread boundary would require lifetime reasoning Fuse intentionally does not
provide. The checker must reject `spawn` captures, `Chan[T]` element types,
and `Shared[T]` inner types that contain a borrow at any nesting depth.

#### Opting out

A type may opt out of an auto-implemented marker by declaring a negative
impl (syntax: `impl !Send for T {}`). Negative impls are available only in
the module that owns the type definition.

#### Compiler enforcement

The checker must reject any program that constructs a `Chan[T]`,
`Shared[T]`, or `spawn` whose relevant type parameters do not satisfy
`Send` / `Sync`. Violations are hard errors before lowering.

---

## 48. Trait Objects and Dynamic Dispatch

> Implementation status: SPECIFIED — W13

> **Importance: non-negotiable for a real systems language.** Without trait
> objects, every collection must be homogeneous, every callback site must know
> the concrete type at compile time, and plugin or component architectures are
> impossible without manual vtable simulation in unsafe code. Monomorphization
> handles the common case well, but dynamic dispatch is required whenever the
> concrete type is determined at runtime: device drivers, GUI widget trees,
> game entity systems, network protocol handlers, and any extensible API.

A trait object is written `dyn TraitName`. It carries a pointer to the value
and a pointer to a vtable of the trait's methods. Ownership forms apply
normally: `ref dyn Trait`, `mutref dyn Trait`, `owned dyn Trait`.

```fuse
// declaring a trait
pub trait Draw {
    fn draw(ref self);
    fn bounding_box(ref self) -> (F64, F64, F64, F64);
}

// heterogeneous collection — impossible with generics alone
var widgets: [owned dyn Draw] = [];
widgets.push(owned Circle { radius: 1.0, center: Point.origin() });
widgets.push(owned Rect   { width: 2.0, height: 3.0 });
widgets.push(owned Text   { content: String.from("hello") });

for widget in widgets {
    widget.draw();
}

// trait object as function parameter — accepts any Draw implementer
fn render(surface: mutref Surface, item: ref dyn Draw) {
    let bb = item.bounding_box();
    surface.clip(bb);
    item.draw();
}

// returning a trait object from a function
fn make_shape(kind: ref String) -> owned dyn Draw {
    match kind.as_str() {
        "circle" { return owned Circle.default(); }
        "rect"   { return owned Rect.default(); }
        _        { panic("unknown shape: {kind}"); }
    }
}

// trait objects in structs (plugin / handler pattern)
struct EventBus {
    handlers: [owned dyn EventHandler],
}

impl EventBus {
    pub fn register(mutref self, h: owned dyn EventHandler) {
        self.handlers.push(owned h);
    }

    pub fn dispatch(ref self, event: ref Event) {
        for h in self.handlers {
            h.on_event(ref event);
        }
    }
}

// trait objects with multiple trait bounds
fn log_and_draw(item: ref (dyn Draw + dyn Format)) {
    print!("{}\n", item);
    item.draw();
}
```

### 48.1 Implementation contracts

#### Trait object representation

`dyn Trait` is represented as a two-word fat pointer: `{ data: Ptr[()],
vtable: Ptr[VtableOf<Trait>] }`. The vtable is a static, per-(trait, impl)
table containing function pointers for each method in declaration order,
preceded by the size and drop function pointer of the concrete type.

#### Ownership forms on trait objects

`ref dyn Trait`, `mutref dyn Trait`, and `owned dyn Trait` are all
supported. Ownership rules apply to the `data` pointer of the fat pointer
exactly as they would to the concrete value.

#### Object safety

A trait is object-safe and usable as `dyn Trait` only if:

- every method's receiver is `ref self`, `mutref self`, or `owned self`
  (not associated-function style)
- no method has generic type parameters
- no method mentions `Self` in a non-receiver position
- the trait does not require any associated constants

Non-object-safe traits must produce a compile error at any `dyn Trait`
use site.

#### Multi-trait dynamic dispatch

`dyn A + B` produces a fat pointer whose vtable is a combined vtable
containing entries for both traits. The order is deterministic: traits are
listed alphabetically for vtable layout. All named traits must be
object-safe.

---

## 49. Unions

> Implementation status: SPECIFIED — W06 (check), W17 (codegen)

> **Importance:** medium for most code; high for protocol parsing, hardware
> register maps, and bit-level type reinterpretation. Enums handle the
> common tagged-union case, but `union` is needed when the discriminant is
> stored externally, when memory layout must exactly match a hardware register,
> or when reinterpreting the raw bit pattern of a value (float ↔ integer).
> All union field access is `unsafe`.

```fuse
// reinterpreting a float's bits as an integer
union FloatBits {
    f: F32,
    i: U32,
}

fn fast_inv_sqrt(x: F32) -> F32 {
    let mut u = FloatBits { f: x };
    unsafe {
        u.i = 0x5f3759df - (u.i >> 1);
        u.f *= 1.5 - 0.5 * x * u.f * u.f;
    }
    return unsafe { u.f };
}

// hardware register with multiple interpretations
union StatusRegister {
    raw:    U32,
    fields: StatusFields,
}

@repr(C)
struct StatusFields {
    ready:     U8,
    error:     U8,
    count:     U16,
}

fn read_status(base: Ptr[U32]) -> StatusRegister {
    unsafe {
        return StatusRegister { raw: *base };
    }
}

// parsing a network packet header
union IpAddress {
    octets: [U8; 4],
    word:   U32,
}

fn is_loopback(addr: IpAddress) -> Bool {
    return unsafe { addr.octets[0] == 127 };
}
```

### 49.1 Implementation contracts

#### Union layout

A `union` occupies the size of its largest field with alignment equal to
the strictest alignment of any field. The compiler does not track which
field is active; all field reads and writes are `unsafe`.

#### Drop restriction

A union's fields may not implement `Drop`. The language has no mechanism
to track which field is active, so automatic drop is unsound. Use structs
with explicit discriminants for managed resources.

---

## 50. Conditional Compilation

> Implementation status: SPECIFIED — W03

> **Importance: non-negotiable for a real systems language.** Writing code
> that targets Linux, macOS, and Windows from a single source tree requires
> platform-conditional compilation. The same applies to CPU architecture
> (x86 vs ARM), OS version, and optional feature flags. Without this, a
> "cross-platform" codebase is actually three separate partially-maintained
> codebases, or everything lives behind runtime checks that cannot be
> eliminated by the optimizer.

`@cfg(...)` controls whether items and blocks are included in the build. The
predicate is evaluated by the compiler, not at runtime.

```fuse
// platform-specific implementations
@cfg(os = "linux")
fn get_page_size() -> USize {
    return unsafe { fuse_rt_linux_page_size() };
}

@cfg(os = "windows")
fn get_page_size() -> USize {
    return unsafe { GetSystemInfo_page_size() };
}

@cfg(os = "macos")
fn get_page_size() -> USize {
    return unsafe { fuse_rt_macos_page_size() };
}

// CPU architecture
@cfg(arch = "x86_64")
fn fast_popcount(x: U64) -> U32 {
    return unsafe { __builtin_popcountll(x) };
}

@cfg(not(arch = "x86_64"))
fn fast_popcount(x: U64) -> U32 {
    return portable_popcount(x);    // fallback
}

// feature flags (set at build time)
@cfg(feature = "simd")
fn dot_product(a: ref [F32], b: ref [F32]) -> F32 {
    return simd_dot(ref a, ref b);
}

@cfg(not(feature = "simd"))
fn dot_product(a: ref [F32], b: ref [F32]) -> F32 {
    return scalar_dot(ref a, ref b);
}

// compound predicates
@cfg(all(os = "linux", arch = "x86_64"))
fn platform_specific() { }

@cfg(any(os = "linux", os = "macos"))
fn unix_only() { }

// conditional blocks within a function body
fn setup_timer() {
    @cfg(os = "linux")  { setup_timerfd(); }
    @cfg(os = "macos")  { setup_kqueue_timer(); }
    @cfg(os = "windows") { setup_waitable_timer(); }
}
```

### 50.1 Implementation contracts

#### `@cfg` predicate evaluation

`@cfg` predicates are evaluated at the resolve stage, before type
checking. Items with a false predicate are removed from the HIR before
check runs; they do not participate in name resolution for other items
in the build.

Supported predicate forms:

- `@cfg(name = "value")` — matches when a configuration variable has the
  given value (e.g. `os = "linux"`).
- `@cfg(feature = "name")` — matches when the named feature is enabled
  in the build configuration.
- `@cfg(not(pred))` — logical negation.
- `@cfg(all(pred, pred, ...))` — logical conjunction.
- `@cfg(any(pred, pred, ...))` — logical disjunction.

#### Duplicate-item resolution across `@cfg`

When two items share a name but have mutually exclusive `@cfg` predicates,
exactly one item survives resolution for any given build. If multiple
items survive (predicates overlap or both evaluate true), the resolver
must emit a diagnostic.

---

## 51. Interior Mutability

> Implementation status: SPECIFIED — W19 (core stdlib)

> **Importance:** significant limitation without it. Interior mutability is
> needed for caches, lazy initialization, reference-counted values, and any
> pattern where a value is shared but occasionally needs to update internal
> state. Without it, the ownership model forces a choice: either keep the
> value exclusively mutable (can't share) or immutable (can't update). The
> most common real-world case is a shared cache where reads are constant but
> cache misses need to write.

`Cell[T]` allows mutation through a shared reference for `Copy` types.
`RefCell[T]` extends this to non-Copy types with runtime borrow checking.
Both are single-threaded; for multi-threaded use, `Shared[T]` (§17.4) is
the right tool.

```fuse
import core.cell.Cell;
import core.cell.RefCell;

// Cell: mutation through shared reference for Copy types
struct Counter {
    value: Cell[I32],
}

impl Counter {
    pub fn new() -> Counter {
        return Counter { value: Cell.new(0) };
    }

    // takes ref self, but can still mutate value through Cell
    pub fn increment(ref self) {
        self.value.set(self.value.get() + 1);
    }

    pub fn get(ref self) -> I32 {
        return self.value.get();
    }
}

// RefCell: runtime-checked mutation for non-Copy types
struct Cache {
    store: RefCell[HashMap[String, String]],
}

impl Cache {
    pub fn get_or_compute(ref self, key: ref String) -> String {
        {
            let store = self.store.borrow();     // shared borrow
            if let Some(v) = store.get(ref key) {
                return v.clone();
            }
        }    // shared borrow released here

        let value = compute(ref key);
        let mut store = self.store.borrow_mut();  // mutable borrow
        store.insert(key.clone(), value.clone());
        return value;
    }
}

// lazy initialization
struct LazyConfig {
    data: RefCell[Option[Config]],
}

impl LazyConfig {
    pub fn get(ref self) -> ref Config {
        if self.data.borrow().is_none() {
            *self.data.borrow_mut() = Some(load_config());
        }
        return self.data.borrow().as_ref().unwrap();
    }
}
```

### 51.1 Implementation contracts

#### `Cell[T]` and `RefCell[T]` marker impact

Neither `Cell[T]` nor `RefCell[T]` is `Sync`. Neither is `Send` unless `T`
is `Send`. The checker must enforce these negative impls so that
interior-mutable types cannot be shared across threads without explicit
synchronization (use `Shared[T]` for that).

#### `RefCell[T]` runtime borrow tracking

`RefCell[T]` maintains a runtime borrow count. `borrow()` increments a
shared-borrow counter; `borrow_mut()` takes the exclusive borrow only when
no borrows are outstanding. Violations (e.g. `borrow_mut()` while a shared
borrow exists) panic at runtime. The compiler does not attempt to prove
correctness at compile time.

---

## 52. Custom Allocators

> Implementation status: SPECIFIED — W20

> **Importance:** high for embedded, game development, and high-performance
> servers. The default allocator is unsuitable for many systems contexts:
> embedded systems have no heap; games need arena allocators that reset every
> frame; high-performance servers need thread-local slab allocators that avoid
> contention. Without a pluggable allocator interface, the language cannot
> serve these contexts.

The `Allocator` trait describes a memory source. Collections and smart
pointers accept an optional allocator parameter.

```fuse
pub trait Allocator {
    fn alloc(mutref self, size: USize, align: USize) -> Result[Ptr[U8], AllocError];
    fn dealloc(mutref self, ptr: Ptr[U8], size: USize, align: USize);
    fn realloc(
        mutref self,
        ptr:      Ptr[U8],
        old_size: USize,
        new_size: USize,
        align:    USize,
    ) -> Result[Ptr[U8], AllocError];
}

// bump allocator — O(1) alloc, free-all-at-once
struct BumpAllocator {
    base:   Ptr[U8],
    offset: USize,
    cap:    USize,
}

impl Allocator for BumpAllocator {
    fn alloc(mutref self, size: USize, align: USize) -> Result[Ptr[U8], AllocError] {
        let aligned = align_up(self.offset, align);
        if aligned + size > self.cap { return Err(AllocError.OutOfMemory); }
        let ptr = unsafe { self.base.offset(aligned) };
        self.offset = aligned + size;
        return Ok(ptr);
    }

    fn dealloc(mutref self, _ptr: Ptr[U8], _size: USize, _align: USize) {
        // bump allocator frees everything at once via reset()
    }

    fn realloc(mutref self, _: Ptr[U8], _: USize, _: USize, _: USize)
        -> Result[Ptr[U8], AllocError]
    {
        return Err(AllocError.NotSupported);
    }
}

impl BumpAllocator {
    pub fn reset(mutref self) { self.offset = 0; }
}

// using a custom allocator with a collection
let mut arena = BumpAllocator.from_slice(ref buffer);
let mut vec: Vec[I32, BumpAllocator] = Vec.new_in(ref arena);
vec.push(1);
vec.push(2);
arena.reset();    // reclaim all memory at once
```

### 52.1 Implementation contracts

#### Allocator-parameterized collections

Core collections (`Vec`, `HashMap`, `Box`, etc.) are generic over an
allocator parameter `A: Allocator`. Default parameter values resolve to
the global `SystemAllocator` when omitted. Every allocation-taking
collection method routes through the provided allocator; none may fall
back to the global allocator silently.

#### Global allocator

The global allocator is declared once per binary. The default provides a
thin wrapper over the runtime's `fuse_rt_alloc_*` surface. A program may
override it with a custom global allocator via a designated attribute
(syntax pending plan revision).

---

## 53. Visibility Granularity

> Implementation status: SPECIFIED — W03

> **Importance:** low-medium. Having only `pub` and private means large modules
> cannot express "visible to sibling modules but not external users." The
> workaround is to reorganize into more files or to expose things publicly that
> should be internal. Annoying but not a hard blocker.

Visibility modifiers control how widely an item is accessible.

```fuse
// private: visible only in this file (default)
fn internal_helper() { }

// pub(mod): visible within the same module and its children
pub(mod) fn module_internal() { }

// pub(pkg): visible within the same package
pub(pkg) fn package_internal() { }

// pub: visible everywhere
pub fn public_api() { }

// on struct fields
pub struct Connection {
    pub address:  String,       // externally visible
    pub(mod) fd:  I32,          // visible to module internals
    state:        ConnState,    // private to this file
}
```

### 53.1 Implementation contracts

#### Visibility levels

Fuse recognizes exactly four visibility levels, in increasing order:

- private (default, unannotated): visible only in the file where declared
- `pub(mod)`: visible in the declaring module and its descendants
- `pub(pkg)`: visible across the entire package boundary
- `pub`: visible everywhere an import reaches

The resolver must enforce these levels on every use site, including
method resolution and field access. A narrower visibility may never shadow
a wider one (a `pub(mod)` impl method is still `pub(mod)`).

---

## 54. Borrow Scope and Struct References

> Implementation status: SPECIFIED — W09

> **Importance:** medium. Fuse intentionally avoids lifetime annotations and a
> borrow checker. This keeps the language simpler but has one concrete
> consequence: a borrow cannot be stored in a struct field. Borrows are scoped
> to the block where they are created. If you need a struct that refers to
> data it does not own, the alternatives are raw pointers (unsafe) or
> restructuring ownership so the struct holds the data.

```fuse
// ALLOWED: borrow used within the same scope
fn process(data: ref [U8]) -> U64 {
    let slice: ref [U8] = data;    // borrow lives within this function
    return hash(ref slice);
}

// ALLOWED: borrow passed through a function (does not outlive the call)
fn with_data[T](data: ref [U8], f: fn(ref [U8]) -> T) -> T {
    return f(ref data);
}

// NOT ALLOWED: storing a borrow in a struct field
// struct View {
//     data: ref [U8],    // compile error: struct fields cannot be borrows
// }

// ALTERNATIVE 1: the struct owns its data
struct View {
    data: [U8],
}

// ALTERNATIVE 2: raw pointer (unsafe, explicit lifetime management)
struct RawView {
    data: Ptr[U8],
    len:  USize,
}

// ALTERNATIVE 3: pass the data alongside the struct
fn process_view(buf: ref [U8], view_offset: USize, view_len: USize) {
    let slice = buf[view_offset .. view_offset + view_len];
    // work with slice, which borrows buf for this scope only
}
```

### 54.1 Implementation contracts

#### No borrow in struct fields

Struct fields may not have a borrow type (`ref T`, `mutref T`). The
checker must reject struct declarations whose field types contain a
borrow at any nesting depth. This is a language-level restriction, not a
limitation of the implementation: Fuse intentionally avoids lifetime
annotations, and borrow-in-struct would require them.

#### Borrow scope

A borrow is alive from its point of formation to the end of the innermost
enclosing block. Borrows cannot be returned unless the returned borrow
points into a parameter that was itself borrowed; this case is handled
structurally by the ownership checker without explicit lifetime variables.

---

## 55. Callable Traits

> Implementation status: SPECIFIED — W12

> **Importance:** high. Without a callable trait hierarchy, you cannot write a
> generic function that accepts any closure or function pointer with a given
> signature. You are limited to concrete function pointer types (which lose
> capture) or monomorphized generics (which work but cannot be stored in a
> heterogeneous collection or returned as a trait object). Callable traits are
> the bridge between closures and generic abstractions.

`Fn`, `FnMut`, and `FnOnce` form a hierarchy. `FnOnce` is the most general;
`Fn` is the most restrictive.

```fuse
// Fn: callable by shared reference; may be called many times
pub trait Fn[Args, Ret]: FnMut[Args, Ret] {
    fn call(ref self, args: Args) -> Ret;
}

// FnMut: callable by mutable reference; may be called many times
pub trait FnMut[Args, Ret]: FnOnce[Args, Ret] {
    fn call_mut(mutref self, args: Args) -> Ret;
}

// FnOnce: callable by consuming self; may be called at most once
pub trait FnOnce[Args, Ret] {
    fn call_once(owned self, args: Args) -> Ret;
}

// using Fn in a generic bound
fn apply_twice[F: Fn[(I32), I32]](f: ref F, x: I32) -> I32 {
    return f.call(x) |> f.call();
}

let double = fn(x: I32) -> I32 { x * 2 };
let result = apply_twice(ref double, 3);    // 12

// using FnMut for a closure that captures mutable state
fn make_counter() -> impl FnMut[(), I32] {
    var count = 0;
    return fn() -> I32 {
        count += 1;
        return count;
    };
}

var counter = make_counter();
let a = counter.call_mut(());    // 1
let b = counter.call_mut(());    // 2

// storing callable trait objects in a vec
var callbacks: [owned dyn FnOnce[(), ()]] = [];
callbacks.push(owned fn() { cleanup_a(); });
callbacks.push(owned fn() { cleanup_b(); });

for cb in callbacks {
    cb.call_once(());    // each runs once, in order
}

// function pointers automatically implement Fn/FnMut/FnOnce
fn double(x: I32) -> I32 { x * 2 }
let f: fn(I32) -> I32 = double;
apply_twice(ref f, 5);    // 20
```

### 55.1 Implementation contracts

#### Hierarchy and auto-impl

`Fn: FnMut: FnOnce` forms a supertrait chain. The compiler auto-implements
`Fn` / `FnMut` / `FnOnce` for:

- every function pointer type `fn(A, B, ...) -> R` (implements all three)
- every closure expression, with the most permissive trait that matches
  the closure's capture pattern: pure captures → `Fn`, mutating captures
  → `FnMut`, consuming captures → `FnOnce`

#### Call desugaring

A call `f(args)` where `f: F` and `F: Fn[Args, R]` desugars to
`f.call(args)`. For `FnMut`, it desugars to `f.call_mut(args)`. For
`FnOnce`, it desugars to `f.call_once(args)`. The choice is driven by the
tightest available bound at the call site.

---

## 56. Opaque Return Types (`impl Trait`)

> Implementation status: SPECIFIED — W06 (check), W08 (monomorphization),
> W15 (lowering)

> **Importance:** medium-high. Without opaque return types, returning a closure
> or a complex iterator chain from a function forces the caller to name the
> concrete type — which for iterator adapters is a deeply nested unreadable
> type — or box it on the heap. `impl Trait` in return position lets the
> function promise "I return something that implements this trait" without
> committing to a name. Essential for iterator-heavy and callback-heavy APIs.

`impl Trait` in return position means "some concrete type that implements
Trait, chosen by the function, not named at the call site."

```fuse
// returning a closure without naming its type
fn make_adder(n: I32) -> impl Fn[(I32), I32] {
    return fn(x: I32) -> I32 { x + n };
}

let add5 = make_adder(5);
let result = add5.call(3);    // 8

// returning an iterator chain without naming the adapter type
fn evens_up_to(limit: I32) -> impl Iterator[Item=I32] {
    return (0..limit).filter(fn(n: ref I32) -> Bool { *n % 2 == 0 });
}

for n in evens_up_to(10) {
    log(n);    // 0 2 4 6 8
}

// builder pattern returning impl Trait for chaining
fn parse_csv(input: ref String) -> impl Iterator[Item=Result[Row, ParseError]] {
    return input
        .lines()
        .skip(1)                             // skip header
        .map(fn(line: ref String) -> Result[Row, ParseError] {
            parse_row(ref line)
        });
}

// impl Trait in parameter position: equivalent to a generic bound
fn print_all(items: impl Iterator[Item=String]) {
    for item in items {
        print!("{}\n", item);
    }
}
// above is shorthand for: fn print_all[I: Iterator[Item=String]](items: I)
```

### 56.1 Implementation contracts

#### Return-position opaque types

`fn f() -> impl Trait` means the function has one concrete return type,
chosen by the function body, that implements `Trait`. Callers may not
name the type; they see it only through `Trait`'s surface. All return
paths in the function body must produce the same concrete type.

Opaque return types are resolved during checking; the concrete type is
recorded in the type table under a synthetic identity and is available
for monomorphization. No runtime representation is added (this is
`static` dispatch, not `dyn`).

#### Parameter-position `impl Trait`

`fn f(x: impl Trait)` is exactly equivalent to
`fn f[T: Trait](x: T)`. The two forms generate identical code.

---

## 57. Backend Representation Contracts

> Implementation status: SPECIFIED — W17

This section defines the backend contracts that are not deducible solely
from surface syntax but are required for correct compiler implementation.
Violating any of them produces generated code that does not reflect the
language semantics.

### 57.1 Contract 1: The two pointer categories

In the bootstrap C backend there are exactly two categories of
pointer-typed locals.

1. Borrow pointers (from `ref`, `mutref`, and implicit borrow formation).
2. `Ptr[T]` values (raw pointers).

Borrow pointers originate from borrow semantics and may need implicit
dereference to recover values. `Ptr[T]` values are first-class pointer
values and must never be implicitly dereferenced. The backend must track
these categories separately.

### 57.2 Contract 2: Unit erasure is total

Once `()` is erased in a concrete ABI position, it is erased at every
producer and consumer site. There is no partially materialized unit value
in generated code. Constructors, patterns, function pointers, and
aggregate layout must all agree.

### 57.3 Contract 3: Monomorphization completeness

Every emitted generic specialization must be complete and concrete. The
backend must not emit calls to unresolved generic symbols. Partial
specializations are a hard error before codegen.

### 57.4 Contract 4: Divergence is structural

A basic block ending in a diverging call has no successors. Code
generation must not synthesize reads of values that would exist only if
control flow continued. Join blocks, aggregate fallbacks, and destruction
paths all depend on accurate divergence handling.

### 57.5 Contract 5: Emission order and aggregate fallback

Composite type definitions must be emitted before they are used by
function signatures or locals in generated C. Aggregate fallback values
must be typed zero-initializers such as `(FuseType_Foo){0}`, not scalar
`0`.

### 57.6 Contract 6: Identifier sanitization and collision avoidance

All backend-emitted identifiers must be legal in the target backend
language and must be collision-resistant.

- C keywords must be escaped or renamed deterministically.
- Numeric field names used internally must be sanitized.
- Same-named symbols from different modules must not collide after
  mangling.

### 57.7 Contract 7: Closure representation

A closure lowers to a pair `{ fn_ptr: Ptr[()], env_ptr: Ptr[()] }` where
`fn_ptr` points at the lifted body function and `env_ptr` points at the
environment struct. Closure calls dispatch through `fn_ptr` passing
`env_ptr` as an implicit first argument. Function pointers (no captures)
use a null `env_ptr`.

### 57.8 Contract 8: Trait object representation

A trait object `dyn Trait` lowers to a fat pointer
`{ data: Ptr[()], vtable: Ptr[Vtable_Trait] }`. Vtables are emitted as
static read-only tables, one per (trait, concrete impl) pair, with a
deterministic layout: `[size, align, drop_fn, method_1, method_2, ...]`
in trait declaration order.

### 57.9 Contract 9: Channel and thread runtime ABI

- `Chan[T]` lowers to an opaque `Ptr[()]` returned by `fuse_rt_chan_new`.
- `ThreadHandle[T]` lowers to an opaque `I64` returned by
  `fuse_rt_thread_spawn`.
- The runtime surface is monomorphized: `Chan[I32]` uses the same
  `fuse_rt_chan_send` entry point, with caller-side serialization of the
  element type.

### 57.10 Contract 10: Alignment and padding

All struct layouts follow the target platform's default C ABI unless
overridden by `@repr(packed)` or `@align(N)` (§37). The compiler must not
reorder fields for its own purposes; layouts are stable and reproducible
across builds.

---

## Appendix A — Keyword Reference

| Keyword | Purpose |
|---|---|
| `fn` | function declaration |
| `pub` | public visibility |
| `struct` | struct type declaration |
| `enum` | enum type declaration |
| `trait` | trait declaration |
| `impl` | implementation block |
| `for` | loop over iterator or trait bound |
| `in` | separator in `for..in` |
| `while` | conditional loop |
| `loop` | unconditional loop |
| `if` | conditional expression |
| `else` | alternate branch |
| `match` | pattern matching expression |
| `return` | return from function |
| `let` | immutable binding |
| `var` | mutable binding |
| `move` | explicit ownership transfer |
| `ref` | shared borrow |
| `mutref` | mutable borrow |
| `owned` | consuming parameter |
| `unsafe` | unsafe block |
| `spawn` | create a concurrent thread |
| `chan` | channel type keyword |
| `import` | import a module or item |
| `as` | alias in import or cast |
| `mod` | reserved |
| `use` | reserved |
| `type` | type alias |
| `const` | compile-time constant |
| `static` | program-lifetime static |
| `extern` | foreign function or static |
| `break` | exit a loop |
| `continue` | skip to next iteration |
| `where` | trait bound clause |
| `Self` | the implementing type in a trait |
| `self` | method receiver |
| `true` | boolean literal |
| `false` | boolean literal |
| `None` | absent Option variant |
| `Some` | present Option variant |

## Appendix B — Operator Precedence

Higher number = tighter binding.

| Level | Operators |
|---|---|
| 1 (loosest) | `=` `+=` `-=` `*=` `/=` `%=` `&=` `\|=` `^=` `<<=` `>>=` |
| 2 | `\|\|` |
| 3 | `&&` |
| 4 | `==` `!=` `<` `>` `<=` `>=` |
| 5 | `\|` |
| 6 | `^` |
| 7 | `&` |
| 8 | `<<` `>>` |
| 9 | `+` `-` |
| 10 | `*` `/` `%` |
| 11 | unary `!` `-` |
| 12 | `?` `?.` postfix calls field access indexing |

## Appendix C — Grammar Reference (EBNF)

> Implementation status: SPECIFIED — W02 (parser)

The EBNF below defines the surface grammar at a reference level. Later
implementation documents may refine parser strategy, but they must not
change the language accepted by this grammar without updating this
section.

```ebnf
file            = { import_decl | item_decl } ;

import_decl     = "import" path [ "as" IDENT ] ";" ;
path            = IDENT { "." IDENT } ;

item_decl       = fn_decl
                | struct_decl
                | enum_decl
                | trait_decl
                | impl_decl
                | const_decl
                | static_decl
                | type_decl
                | extern_decl
                | union_decl ;

fn_decl         = [ "pub" | visibility ] [ "const" ] [ decorator ]
                  "fn" IDENT [ generic_params ]
                  "(" [ param_list ] ")"
                  [ "->" type_expr ]
                  [ where_clause ]
                  block_expr ;

visibility      = "pub" "(" ( "mod" | "pkg" ) ")" ;

param_list      = param { "," param } ;
param           = [ ownership ] IDENT ":" type_expr ;
ownership       = "ref" | "mutref" | "owned" ;

struct_decl     = [ "pub" | visibility ] [ decorator ] "struct" IDENT
                  [ generic_params ]
                  ( "{" [ field_list ] "}"
                  | "(" [ type_list ] ")" ";"
                  | ";" ) ;

enum_decl       = [ "pub" | visibility ] [ decorator ] "enum" IDENT
                  [ generic_params ]
                  "{" [ variant_list ] "}" ;

variant_list    = variant { "," variant } [ "," ] ;
variant         = IDENT [ "=" expr ]
                | IDENT "(" [ type_list ] ")"
                | IDENT "{" [ field_list ] "}" ;

trait_decl      = [ "pub" | visibility ] "trait" IDENT [ generic_params ]
                  [ ":" type_list ]
                  [ where_clause ]
                  "{" { trait_item } "}" ;

trait_item      = fn_decl
                | "type" IDENT [ ":" type_list ] ";"
                | "const" IDENT ":" type_expr [ "=" expr ] ";" ;

impl_decl       = "impl" [ generic_params ] type_expr
                  [ ":" type_expr ]
                  [ where_clause ]
                  "{" { impl_item } "}" ;

impl_item       = fn_decl
                | "type" IDENT "=" type_expr ";"
                | "const" IDENT ":" type_expr "=" expr ";" ;

const_decl      = [ "pub" | visibility ] "const" IDENT ":" type_expr "=" expr ";" ;
static_decl     = [ "pub" | visibility ] "static" IDENT ":" type_expr "=" expr ";" ;
type_decl       = [ "pub" | visibility ] "type" IDENT [ generic_params ] "=" type_expr ";" ;
extern_decl     = "extern" ( fn_extern | static_extern ) ;
fn_extern       = "fn" IDENT "(" [ param_list [ "," "..." ] | "..." ] ")"
                  [ "->" type_expr ] ";" ;
static_extern   = "static" IDENT ":" type_expr ";" ;
union_decl      = [ "pub" | visibility ] [ decorator ] "union" IDENT
                  [ generic_params ] "{" field_list "}" ;

field_list      = field { "," field } [ "," ] ;
field           = [ "pub" | visibility ] IDENT ":" type_expr ;

generic_params  = "[" generic_param { "," generic_param } "]" ;
generic_param   = IDENT [ ":" type_list ] ;
where_clause    = "where" where_pred { "," where_pred } ;
where_pred      = type_expr ":" type_list
                | type_expr "=" type_expr ;

type_expr       = path [ type_args ]
                | tuple_type
                | array_type
                | slice_type
                | ptr_type
                | fn_type
                | dyn_type
                | impl_type
                | unit_type ;

type_args       = "[" type_expr { "," type_expr } "]" ;
tuple_type      = "(" type_expr { "," type_expr } [ "," ] ")" ;
array_type      = "[" type_expr ";" expr "]" ;
slice_type      = "[" type_expr "]" ;
ptr_type        = "Ptr" "[" type_expr "]" ;
fn_type         = "fn" "(" [ type_list ] ")" [ "->" type_expr ] ;
dyn_type        = "dyn" type_expr { "+" type_expr } ;
impl_type       = "impl" type_expr ;
unit_type       = "(" ")" ;
type_list       = type_expr { "," type_expr } ;

decorator       = "@" IDENT [ "(" decorator_args ")" ] ;
decorator_args  = decorator_arg { "," decorator_arg } ;
decorator_arg   = IDENT [ "=" expr ] | expr ;

expr            = assignment_expr ;
assignment_expr = logic_or_expr [ assign_op assignment_expr ] ;
logic_or_expr   = logic_and_expr { "||" logic_and_expr } ;
logic_and_expr  = compare_expr { "&&" compare_expr } ;
compare_expr    = or_expr { compare_op or_expr } ;
or_expr         = xor_expr { "|" xor_expr } ;
xor_expr        = and_expr { "^" and_expr } ;
and_expr        = shift_expr { "&" shift_expr } ;
shift_expr      = additive_expr { shift_op additive_expr } ;
additive_expr   = multiplicative_expr { add_op multiplicative_expr } ;
multiplicative_expr
                = cast_expr { mul_op cast_expr } ;
cast_expr       = unary_expr { "as" type_expr } ;
unary_expr      = { unary_op } postfix_expr ;
unary_op        = "!" | "-" | "*" | "&" ;
postfix_expr    = primary_expr { postfix_op } ;
postfix_op      = "." IDENT
                | "?." IDENT
                | "?" 
                | "(" [ arg_list ] ")"
                | "[" expr "]"
                | "[" expr ".." [ expr ] "]"
                | "[" [ expr ] ".." expr "]"
                | "[" expr "..=" expr "]" ;

primary_expr    = literal
                | path [ type_args ]
                | block_expr
                | if_expr
                | match_expr
                | loop_expr
                | while_expr
                | for_expr
                | tuple_expr
                | struct_lit
                | closure_expr
                | spawn_expr
                | unsafe_block
                | "(" expr ")" ;

block_expr      = "{" { stmt } [ expr ] "}" ;
stmt            = let_stmt
                | var_stmt
                | return_stmt
                | break_stmt
                | continue_stmt
                | expr_stmt
                | item_decl ;

let_stmt        = "let" pattern [ ":" type_expr ] [ "=" expr ] ";" ;
var_stmt        = "var" IDENT [ ":" type_expr ] "=" expr ";" ;
return_stmt     = "return" [ expr ] ";" ;
break_stmt      = "break" [ expr ] ";" ;
continue_stmt   = "continue" ";" ;
expr_stmt       = expr ";" ;

if_expr         = "if" expr block_expr [ "else" ( if_expr | block_expr ) ] ;
match_expr      = "match" expr "{" { match_arm } "}" ;
match_arm       = pattern [ "if" expr ] block_expr ;
loop_expr       = "loop" block_expr ;
while_expr      = "while" expr block_expr ;
for_expr        = "for" pattern "in" expr block_expr ;

pattern         = literal_pat
                | wildcard_pat
                | bind_pat
                | ctor_pat
                | tuple_pat
                | struct_pat
                | or_pat
                | range_pat
                | at_pat ;

literal_pat     = literal ;
wildcard_pat    = "_" ;
bind_pat        = IDENT ;
ctor_pat        = path [ "(" [ pattern_list ] ")" | "{" [ field_pat_list ] "}" ] ;
tuple_pat       = "(" pattern_list ")" ;
struct_pat      = path "{" [ field_pat_list ] [ ".." ] "}" ;
or_pat          = pattern "|" pattern { "|" pattern } ;
range_pat       = expr ".." expr | expr "..=" expr ;
at_pat          = IDENT "@" pattern ;

closure_expr    = "fn" "(" [ param_list ] ")" [ "->" type_expr ] block_expr ;
spawn_expr      = "spawn" closure_expr ;
unsafe_block    = "unsafe" block_expr ;

compare_op      = "==" | "!=" | "<" | ">" | "<=" | ">=" ;
assign_op       = "=" | "+=" | "-=" | "*=" | "/=" | "%="
                | "&=" | "|=" | "^=" | "<<=" | ">>=" ;
shift_op        = "<<" | ">>" ;
add_op          = "+" | "-" ;
mul_op          = "*" | "/" | "%" ;

arg_list        = expr { "," expr } ;
pattern_list    = pattern { "," pattern } ;
field_pat_list  = field_pat { "," field_pat } ;
field_pat       = IDENT [ ":" pattern ] ;
tuple_expr      = "(" expr { "," expr } [ "," ] ")" ;
struct_lit      = path "{" [ struct_lit_field { "," struct_lit_field } ]
                  [ ".." expr ] "}" ;
struct_lit_field = IDENT ":" expr | IDENT ;

literal         = INTEGER | FLOAT | STRING | RAW_STRING | CSTRING | CHAR
                | "true" | "false" | "None" ;
```

The grammar above is the language surface. It is not an LL(k) or LR(k)
specification; ambiguities named in the feature sections above (struct
literal disambiguation in §10.7, `?.` longest-match in §1.10, field versus
method call in §9.4) are resolved by the parser per their implementation
contracts, not by grammar rewriting.
