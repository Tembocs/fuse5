package ast

// AllItemNodes is one zero-value instance of every concrete Item kind. Tests
// use it to assert that every grammar item production has a corresponding
// node type. Adding a new item-level AST node requires appending to this
// list; TestAstNodeCompleteness enforces the invariant.
var AllItemNodes = []Item{
	&FnDecl{},
	&StructDecl{},
	&EnumDecl{},
	&TraitDecl{},
	&ImplDecl{},
	&ConstDecl{},
	&StaticDecl{},
	&TypeDecl{},
	&ExternDecl{},
	&UnionDecl{},
	&TraitTypeItem{},
	&TraitConstItem{},
	&ImplTypeItem{},
}

// AllExprNodes is one zero-value instance of every concrete Expr kind.
var AllExprNodes = []Expr{
	&LiteralExpr{},
	&PathExpr{},
	&BinaryExpr{},
	&AssignExpr{},
	&UnaryExpr{},
	&CastExpr{},
	&CallExpr{},
	&FieldExpr{},
	&OptFieldExpr{},
	&TryExpr{},
	&IndexExpr{},
	&IndexRangeExpr{},
	&BlockExpr{},
	&IfExpr{},
	&MatchExpr{},
	&LoopExpr{},
	&WhileExpr{},
	&ForExpr{},
	&TupleExpr{},
	&StructLitExpr{},
	&ClosureExpr{},
	&SpawnExpr{},
	&UnsafeExpr{},
	&ParenExpr{},
}

// AllStmtNodes is one zero-value instance of every concrete Stmt kind.
var AllStmtNodes = []Stmt{
	&LetStmt{},
	&VarStmt{},
	&ReturnStmt{},
	&BreakStmt{},
	&ContinueStmt{},
	&ExprStmt{},
	&ItemStmt{},
}

// AllPatNodes is one zero-value instance of every concrete Pat kind.
var AllPatNodes = []Pat{
	&LiteralPat{},
	&WildcardPat{},
	&BindPat{},
	&CtorPat{},
	&TuplePat{},
	&OrPat{},
	&RangePat{},
	&AtPat{},
}

// AllTypeNodes is one zero-value instance of every concrete Type kind.
var AllTypeNodes = []Type{
	&PathType{},
	&TupleType{},
	&ArrayType{},
	&SliceType{},
	&PtrType{},
	&FnType{},
	&DynType{},
	&ImplType{},
	&UnitType{},
}
