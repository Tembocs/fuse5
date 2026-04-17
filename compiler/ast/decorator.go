package ast

// Decorator is `@NAME[(args...)]`. Positional args are expressions; named
// args use `name = expr` — both are stored on DecoratorArg.
type Decorator struct {
	NodeBase
	Name Ident
	Args []*DecoratorArg
}

// DecoratorArg is one argument inside a decorator's argument list. The `Name`
// field is filled for the `name = expr` form; otherwise only `Value` is set.
type DecoratorArg struct {
	NodeBase
	Name  *Ident // nil for positional
	Value Expr
}
