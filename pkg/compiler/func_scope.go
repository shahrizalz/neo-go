package compiler

import (
	"go/ast"
	"go/token"
)

// A funcScope represents the scope within the function context.
// It holds al the local variables along with the initialized struct positions.
type funcScope struct {
	// Identifier of the function.
	name string

	// Selector of the function if there is any. Only functions imported
	// from other packages should have a selector.
	selector *ast.Ident

	// The declaration of the function in the AST. Nil if this scope is not a function.
	decl *ast.FuncDecl

	// Program label of the scope
	label uint16

	// Range of opcodes corresponding to the function.
	rng DebugRange
	// Variables together with it's type in neo-vm.
	variables []string

	// Local variables
	locals map[string]int

	// voidCalls are basically functions that return their value
	// into nothing. The stack has their return value but there
	// is nothing that consumes it. We need to keep track of
	// these functions so we can cleanup (drop) the returned
	// value from the stack. We also need to add every voidCall
	// return value to the stack size.
	voidCalls map[*ast.CallExpr]bool

	// Local variable counter.
	i int
}

func newFuncScope(decl *ast.FuncDecl, label uint16) *funcScope {
	return &funcScope{
		name:      decl.Name.Name,
		decl:      decl,
		label:     label,
		locals:    map[string]int{},
		voidCalls: map[*ast.CallExpr]bool{},
		variables: []string{},
		i:         -1,
	}
}

// analyzeVoidCalls checks for functions that are not assigned
// and therefore we need to cleanup the return value from the stack.
func (c *funcScope) analyzeVoidCalls(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.AssignStmt:
		for i := 0; i < len(n.Rhs); i++ {
			switch n.Rhs[i].(type) {
			case *ast.CallExpr:
				return false
			}
		}
	case *ast.ReturnStmt:
		switch n.Results[0].(type) {
		case *ast.CallExpr:
			return false
		}
	case *ast.BinaryExpr:
		return false
	case *ast.CallExpr:
		c.voidCalls[n] = true
		return false
	}
	return true
}

func (c *funcScope) stackSize() int64 {
	size := 0
	ast.Inspect(c.decl, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				size += len(n.Rhs)
			}
		case *ast.ReturnStmt, *ast.IfStmt:
			size++
		// This handles the inline GenDecl like "var x = 2"
		case *ast.GenDecl:
			switch t := n.Specs[0].(type) {
			case *ast.ValueSpec:
				if len(t.Values) > 0 {
					size++
				}
			}
		}
		return true
	})

	numArgs := len(c.decl.Type.Params.List)
	// Also take care of struct methods recv: e.g. (t Token).Foo().
	if c.decl.Recv != nil {
		numArgs += len(c.decl.Recv.List)
	}
	return int64(size + numArgs + len(c.voidCalls))
}

// newLocal creates a new local variable into the scope of the function.
func (c *funcScope) newLocal(name string) int {
	c.i++
	c.locals[name] = c.i
	return c.i
}

// loadLocal loads the position of a local variable inside the scope of the function.
func (c *funcScope) loadLocal(name string) int {
	i, ok := c.locals[name]
	if !ok {
		// should emit a compiler warning.
		return c.newLocal(name)
	}
	return i
}
