package celql

import (
	"fmt"
	"testing"

	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
)

func TestDebugAST(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ast, err := c.Parse("row.user_id == auth.uid")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	native := ast.NativeRep()
	debugNode(native.Expr(), 0)
}

func debugNode(node ast.Expr, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	
	fmt.Printf("%sKind: %v\n", indent, node.Kind())
	
	switch node.Kind() {
	case ast.CallKind:
		call := node.AsCall()
		fmt.Printf("%sFunctionName: %q\n", indent, call.FunctionName())
		fmt.Printf("%sArgs: %d\n", indent, len(call.Args()))
		for i, arg := range call.Args() {
			fmt.Printf("%sArg %d:\n", indent, i)
			debugNode(arg, depth+1)
		}
	case ast.SelectKind:
		sel := node.AsSelect()
		fmt.Printf("%sField: %q\n", indent, sel.FieldName())
		fmt.Printf("%sOperand:\n", indent)
		debugNode(sel.Operand(), depth+1)
	case ast.IdentKind:
		ident := node.AsIdent()
		fmt.Printf("%sIdent: %q\n", indent, ident)
	case ast.LiteralKind:
		lit := node.AsLiteral()
		fmt.Printf("%sValue: %v\n", indent, lit.Value())
	}
	
	// Check if this looks like a binary operator
	if call := node.AsCall(); call != nil {
		switch call.FunctionName() {
		case operators.Equals, operators.NotEquals, operators.Less, operators.LessEquals,
			 operators.Greater, operators.GreaterEquals, operators.In,
			 operators.LogicalAnd, operators.LogicalOr, operators.LogicalNot:
			fmt.Printf("%s-> Binary operator: %s\n", indent, call.FunctionName())
		}
	}
}
