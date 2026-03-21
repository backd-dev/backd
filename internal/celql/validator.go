package celql

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
)

// ValidationError represents an error with an unsupported CEL construct
type ValidationError struct {
	Expression string
	Message    string
	Hint       string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("ERROR: untranslatable expression: %q\n  └─► %s\n  └─► hint: %s", e.Expression, e.Message, e.Hint)
}

// validator walks the AST to detect unsupported constructs
type validator struct {
	expression string
}

// newValidator creates a new validator instance
func newValidator() *validator {
	return &validator{}
}

// validate checks the AST for unsupported constructs
func (v *validator) validate(ast *cel.Ast) error {
	v.expression = "cel expression" // TODO: Get from source info
	nativeAST := ast.NativeRep()
	return v.walkNode(nativeAST.Expr())
}

// walkNode recursively validates AST nodes
func (v *validator) walkNode(node ast.Expr) error {
	switch node.Kind() {
	case ast.CallKind:
		return v.validateCall(node.AsCall())
	case ast.SelectKind:
		return v.validateSelect(node.AsSelect())
	case ast.IdentKind:
		return v.validateIdent(node.AsIdent())
	case ast.ListKind:
		return v.validateList(node.AsList())
	case ast.StructKind:
		return v.validateStruct(node.AsStruct())
	case ast.MapKind:
		return v.validateMap(node.AsMap())
	case ast.ComprehensionKind:
		return v.errorf(node, "comprehensions are not supported", "use simple boolean expressions instead")
	case ast.LiteralKind:
		return nil // Literals are always valid
	default:
		// Check if this is a unary or binary operator (they are represented as CallExpr)
		if call := node.AsCall(); call != nil {
			switch call.FunctionName() {
			case operators.LogicalNot:
				return v.validateUnary(node)
			case operators.Equals, operators.NotEquals, operators.Less, operators.LessEquals,
				operators.Greater, operators.GreaterEquals, operators.In,
				operators.LogicalAnd, operators.LogicalOr:
				return v.validateBinary(node)
			default:
				return v.errorf(node, "unsupported expression type", "use only supported operators and functions")
			}
		}
		return v.errorf(node, "unsupported expression type", "use only supported operators and functions")
	}
}

// validateCall validates function calls
func (v *validator) validateCall(call ast.CallExpr) error {
	// Check for allowed functions
	fnName := call.FunctionName()

	// Binary operators like _==_, _!=_, etc. are handled in walkNode
	if fnName == "" || len(fnName) > 2 && (fnName[0] == '_' && fnName[len(fnName)-1] == '_') {
		return nil
	}

	// Allow only specific functions
	switch fnName {
	case "has", "now", "today":
		// Validate arguments
		for i, arg := range call.Args() {
			if err := v.walkNode(arg); err != nil {
				return err
			}
			// has() must be called on auth.meta or auth.metaApp
			if fnName == "has" && i == 0 {
				if selectExpr := arg.AsSelect(); selectExpr != nil {
					if !v.isAuthAccess(selectExpr) {
						return v.errorf(arg, "has() can only be used on auth fields",
							"use has(auth.meta.field) or has(auth.metaApp.field)")
					}
				} else {
					return v.errorf(arg, "has() requires an auth field selector",
						"use has(auth.meta.field) or has(auth.metaApp.field)")
				}
			}
		}
		return nil
	default:
		return v.errorf(call.Target(), "function %q is not supported", "use only has(), now(), or today()")
	}
}

// validateSelect validates field/attribute access
func (v *validator) validateSelect(sel ast.SelectExpr) error {
	// Validate the operand
	if err := v.walkNode(sel.Operand()); err != nil {
		return err
	}

	// Check for unsupported method calls
	if sel.IsTestOnly() {
		return v.errorf(sel.Operand(), "presence tests are not supported", "use has() function instead")
	}

	// Only allow dot notation on auth and row
	operand := sel.Operand()
	if ident := operand.AsIdent(); ident != "" {
		switch ident {
		case "row", "auth":
			return nil // Allowed
		default:
			return v.errorf(sel.Operand(), "field access on %q is not supported",
				"use only row.field or auth.field")
		}
	}

	// Allow nested auth.meta.field access
	if selectExpr := operand.AsSelect(); selectExpr != nil {
		if v.isAuthAccess(selectExpr) {
			return nil
		}
	}

	return v.errorf(sel.Operand(), "nested field access is not supported", "use only row.field or auth.field")
}

// validateIdent validates identifier references
func (v *validator) validateIdent(ident string) error {
	switch ident {
	case "row", "auth":
		return nil
	case "true", "false":
		return nil
	default:
		return v.errorf(nil, "identifier %q is not supported",
			"use only row, auth, true, or false")
	}
}

// validateList validates list literals
func (v *validator) validateList(list ast.ListExpr) error {
	for _, elem := range list.Elements() {
		if err := v.walkNode(elem); err != nil {
			return err
		}
	}
	return nil
}

// validateStruct validates struct literals
func (v *validator) validateStruct(structExpr ast.StructExpr) error {
	return v.errorf(nil, "struct literals are not supported", "use only simple boolean expressions")
}

// validateMap validates map literals
func (v *validator) validateMap(mapExpr ast.MapExpr) error {
	return v.errorf(nil, "map literals are not supported", "use only simple boolean expressions")
}

// validateUnary validates unary operators
func (v *validator) validateUnary(node ast.Expr) error {
	call := node.AsCall()
	if err := v.walkNode(call.Args()[0]); err != nil {
		return err
	}

	switch call.FunctionName() {
	case operators.LogicalNot:
		return nil // Logical NOT is allowed
	default:
		return v.errorf(node, "unary operator %q is not supported", "use only ! for logical NOT")
	}
}

// validateBinary validates binary operators
func (v *validator) validateBinary(node ast.Expr) error {
	call := node.AsCall()
	args := call.Args()

	// Validate both operands
	if err := v.walkNode(args[0]); err != nil {
		return err
	}
	if err := v.walkNode(args[1]); err != nil {
		return err
	}

	switch call.FunctionName() {
	case operators.Equals, operators.NotEquals, operators.Less, operators.LessEquals,
		operators.Greater, operators.GreaterEquals, operators.In:
		return nil // Comparison operators are allowed
	case operators.LogicalAnd, operators.LogicalOr:
		return nil // Logical operators are allowed
	default:
		return v.errorf(node, "binary operator %q is not supported",
			"use only ==, !=, <, <=, >, >=, &&, ||, or in")
	}
}

// isAuthAccess checks if a select expression is accessing auth fields
func (v *validator) isAuthAccess(sel ast.SelectExpr) bool {
	// Check for direct auth.field
	if ident := sel.Operand().AsIdent(); ident == "auth" {
		return true
	}

	// Check for nested auth.meta.field
	if selectExpr := sel.Operand().AsSelect(); selectExpr != nil {
		if ident := selectExpr.Operand().AsIdent(); ident == "auth" {
			if fieldIdent := selectExpr.FieldName(); fieldIdent != "" {
				return fieldIdent == "meta" || fieldIdent == "metaApp"
			}
		}
	}

	return false
}

// errorf creates a ValidationError with context
func (v *validator) errorf(node ast.Expr, message, hint string) error {
	return ValidationError{
		Expression: v.expression,
		Message:    message,
		Hint:       hint,
	}
}
