package celql

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types/ref"
)

// transpiler walks the AST and converts it to SQL
type transpiler struct {
	auth    AuthContext
	params  []any
	counter int
}

// newTranspiler creates a new transpiler instance
func newTranspiler(auth AuthContext) *transpiler {
	return &transpiler{
		auth:    auth,
		params:  make([]any, 0),
		counter: 0,
	}
}

// transpile converts a CEL AST to SQL
func (t *transpiler) transpile(ast *cel.Ast) (TranspileResult, error) {
	nativeAST := ast.NativeRep()
	sql, err := t.walkNode(nativeAST.Expr())
	if err != nil {
		return TranspileResult{}, err
	}

	// Wrap the expression in parentheses unless it's a NOT expression or already wrapped
	if sql != "" && !strings.HasPrefix(sql, "(") && !strings.HasPrefix(sql, "NOT ") {
		sql = fmt.Sprintf("(%s)", sql)
	}

	return TranspileResult{
		SQL:    sql,
		Params: t.params,
	}, nil
}

// walkNode recursively transpiles AST nodes
func (t *transpiler) walkNode(node ast.Expr) (string, error) {
	switch node.Kind() {
	case ast.CallKind:
		return t.transpileCall(node.AsCall())
	case ast.SelectKind:
		return t.transpileSelect(node.AsSelect())
	case ast.IdentKind:
		return t.transpileIdent(node.AsIdent())
	case ast.ListKind:
		return t.transpileList(node.AsList())
	case ast.StructKind, ast.MapKind:
		return "", fmt.Errorf("struct and map literals are not supported")
	case ast.ComprehensionKind:
		return "", fmt.Errorf("comprehensions are not supported")
	case ast.LiteralKind:
		return t.transpileLiteral(node.AsLiteral())
	default:
		// Check if this is a unary or binary operator
		if call := node.AsCall(); call != nil {
			switch call.FunctionName() {
			case operators.LogicalNot:
				return t.transpileUnary(node)
			case operators.Equals, operators.NotEquals, operators.Less, operators.LessEquals,
				operators.Greater, operators.GreaterEquals, operators.In,
				operators.LogicalAnd, operators.LogicalOr:
				return t.transpileBinary(node)
			}
		}
		return "", fmt.Errorf("unsupported expression type")
	}
}

// transpileCall handles function calls
func (t *transpiler) transpileCall(call ast.CallExpr) (string, error) {
	fnName := call.FunctionName()

	// Handle binary operators
	if len(fnName) > 2 && fnName[0] == '_' && fnName[len(fnName)-1] == '_' ||
		len(fnName) > 1 && fnName[0] == '!' && fnName[1] == '_' {
		return t.transpileBinaryOperator(call)
	}

	// Handle special function names from macros
	if fnName == "@in" {
		return t.transpileInOperator(call)
	}

	switch fnName {
	case "has":
		// has(auth.meta.field) -> $N IS NOT NULL
		if len(call.Args()) != 1 {
			return "", fmt.Errorf("has() requires exactly one argument")
		}
		arg := call.Args()[0]
		if selectExpr := arg.AsSelect(); selectExpr != nil {
			field, err := t.transpileSelect(selectExpr)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s IS NOT NULL", field), nil
		}
		return "", fmt.Errorf("has() requires a field selector")

	case "now":
		// now() -> NOW()
		if len(call.Args()) != 0 {
			return "", fmt.Errorf("now() takes no arguments")
		}
		return "NOW()", nil

	case "today":
		// today() -> CURRENT_DATE
		if len(call.Args()) != 0 {
			return "", fmt.Errorf("today() takes no arguments")
		}
		return "CURRENT_DATE()", nil

	default:
		return "", fmt.Errorf("function %q is not supported", fnName)
	}
}

// transpileSelect handles field/attribute access
func (t *transpiler) transpileSelect(sel ast.SelectExpr) (string, error) {
	// Handle presence tests (has() function calls)
	if sel.IsTestOnly() {
		// has(auth.meta.field) becomes field IS NOT NULL
		// We need to get the actual value and check if it's not null
		fieldName := sel.FieldName()
		operand := sel.Operand()

		// Check if this is auth.meta.field access
		if operand.Kind() == ast.SelectKind {
			operandSel := operand.AsSelect()
			if operandIdent := operandSel.Operand().AsIdent(); operandIdent == "auth" {
				if operandField := operandSel.FieldName(); operandField == "meta" || operandField == "metaApp" {
					// This is auth.meta.field or auth.metaApp.field
					var value any
					if operandField == "meta" && t.auth.Meta != nil {
						value = t.auth.Meta[fieldName]
					} else if operandField == "metaApp" && t.auth.MetaApp != nil {
						value = t.auth.MetaApp[fieldName]
					}
					param := t.addParam(value)
					return fmt.Sprintf("%s IS NOT NULL", param), nil
				}
			}
		}

		// For other fields, just return the field name with IS NOT NULL
		operandStr, err := t.walkNode(operand)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s IS NOT NULL", operandStr, fieldName), nil
	}

	operand, err := t.walkNode(sel.Operand())
	if err != nil {
		return "", err
	}

	fieldName := sel.FieldName()

	// Handle auth.field access
	if operand == "auth" {
		switch fieldName {
		case "uid":
			param := t.addParam(t.auth.UID)
			return param, nil
		case "authenticated":
			param := t.addParam(t.auth.Authenticated)
			return param, nil
		case "meta", "metaApp":
			// These will be handled as nested access
			return fmt.Sprintf("%s.%s", operand, fieldName), nil
		default:
			return "", fmt.Errorf("auth field %q is not supported", fieldName)
		}
	}

	// Handle row.field access
	if operand == "row" {
		return fmt.Sprintf("%s", fieldName), nil
	}

	// Handle nested auth.meta.field access
	if strings.HasPrefix(operand, "auth.") {
		var value any

		// Handle auth.meta.field pattern
		if operand == "auth.meta" && fieldName != "" {
			if t.auth.Meta != nil {
				value = t.auth.Meta[fieldName]
			}
		} else if operand == "auth.metaApp" && fieldName != "" {
			if t.auth.MetaApp != nil {
				value = t.auth.MetaApp[fieldName]
			}
		} else {
			// Handle nested pattern like auth.meta.field where field is the final field
			parts := strings.Split(strings.TrimPrefix(operand, "auth."), ".")
			if len(parts) >= 2 && parts[0] == "meta" {
				if t.auth.Meta != nil {
					key := fieldName
					if key == "" {
						key = parts[1]
					}
					value = t.auth.Meta[key]
				}
			} else if len(parts) >= 2 && parts[0] == "metaApp" {
				if t.auth.MetaApp != nil {
					key := fieldName
					if key == "" {
						key = parts[1]
					}
					value = t.auth.MetaApp[key]
				}
			}
		}

		param := t.addParam(value)
		return param, nil
	}

	return "", fmt.Errorf("field access not supported: %s.%s", operand, fieldName)
}

// transpileIdent handles identifiers
func (t *transpiler) transpileIdent(ident string) (string, error) {
	switch ident {
	case "true":
		return "TRUE", nil
	case "false":
		return "FALSE", nil
	case "row", "auth":
		return ident, nil
	default:
		return "", fmt.Errorf("identifier %q is not supported", ident)
	}
}

// transpileList handles list literals
func (t *transpiler) transpileList(list ast.ListExpr) (string, error) {
	var elements []string
	for _, elem := range list.Elements() {
		lit := elem.AsLiteral()
		if lit == nil {
			return "", fmt.Errorf("list elements must be literals")
		}

		// Add literal as parameter
		value := lit.Value()
		param := t.addParam(value)
		elements = append(elements, param)
	}

	return fmt.Sprintf("ARRAY[%s]", strings.Join(elements, ", ")), nil
}

// transpileLiteral handles literal values
func (t *transpiler) transpileLiteral(lit ref.Val) (string, error) {
	value := lit.Value()

	switch v := value.(type) {
	case string:
		param := t.addParam(v)
		return param, nil
	case int64, int, float64:
		param := t.addParam(v)
		return param, nil
	case bool:
		if v {
			return "TRUE", nil
		}
		return "FALSE", nil
	case nil:
		return "NULL", nil
	case time.Time:
		param := t.addParam(v)
		return param, nil
	default:
		param := t.addParam(v)
		return param, nil
	}
}

// transpileUnary handles unary operators
func (t *transpiler) transpileUnary(node ast.Expr) (string, error) {
	call := node.AsCall()
	if len(call.Args()) != 1 {
		return "", fmt.Errorf("unary operator requires exactly one operand")
	}

	operand, err := t.walkNode(call.Args()[0])
	if err != nil {
		return "", err
	}

	switch call.FunctionName() {
	case operators.LogicalNot:
		return fmt.Sprintf("NOT (%s)", operand), nil
	default:
		return "", fmt.Errorf("unsupported unary operator: %s", call.FunctionName())
	}
}

// transpileBinary handles binary operators
func (t *transpiler) transpileBinary(node ast.Expr) (string, error) {
	call := node.AsCall()
	if len(call.Args()) != 2 {
		return "", fmt.Errorf("binary operator requires exactly two operands")
	}

	left, err := t.walkNode(call.Args()[0])
	if err != nil {
		return "", err
	}

	right, err := t.walkNode(call.Args()[1])
	if err != nil {
		return "", err
	}

	switch call.FunctionName() {
	case operators.Equals:
		return fmt.Sprintf("%s = %s", left, right), nil
	case operators.NotEquals:
		return fmt.Sprintf("%s != %s", left, right), nil
	case operators.Less:
		return fmt.Sprintf("%s < %s", left, right), nil
	case operators.LessEquals:
		return fmt.Sprintf("%s <= %s", left, right), nil
	case operators.Greater:
		return fmt.Sprintf("%s > %s", left, right), nil
	case operators.GreaterEquals:
		return fmt.Sprintf("%s >= %s", left, right), nil
	case operators.In:
		// left in right -> left = ANY(right)
		return fmt.Sprintf("%s = ANY(%s)", left, right), nil
	case operators.LogicalAnd:
		return fmt.Sprintf("(%s AND %s)", left, right), nil
	case operators.LogicalOr:
		return fmt.Sprintf("(%s OR %s)", left, right), nil
	default:
		return "", fmt.Errorf("unsupported binary operator: %s", call.FunctionName())
	}
}

// addParam adds a parameter and returns the placeholder
func (t *transpiler) addParam(value any) string {
	t.counter++
	t.params = append(t.params, value)
	return fmt.Sprintf("$%d", t.counter)
}

// transpileBinaryOperator handles binary operators like _==_, _!=_, etc.
func (t *transpiler) transpileBinaryOperator(call ast.CallExpr) (string, error) {
	// Handle unary operators like !_
	if call.FunctionName() == "!_" || call.FunctionName() == "-_" {
		if len(call.Args()) != 1 {
			return "", fmt.Errorf("unary operator requires exactly one operand")
		}
		left, err := t.walkNode(call.Args()[0])
		if err != nil {
			return "", err
		}
		if call.FunctionName() == "!_" {
			return fmt.Sprintf("NOT (%s)", left), nil
		}
		return fmt.Sprintf("-%s", left), nil
	}

	if len(call.Args()) != 2 {
		return "", fmt.Errorf("binary operator requires exactly two operands")
	}

	left, err := t.walkNode(call.Args()[0])
	if err != nil {
		return "", err
	}

	// Check for null comparisons before walking the right side
	if rightLiteral := call.Args()[1].AsLiteral(); rightLiteral != nil {
		if val := rightLiteral.Value(); fmt.Sprintf("%v", val) == "NULL_VALUE" {
			switch call.FunctionName() {
			case operators.Equals:
				return fmt.Sprintf("(%s IS NULL)", left), nil
			case operators.NotEquals:
				return fmt.Sprintf("(%s IS NOT NULL)", left), nil
			}
		}
	}

	right, err := t.walkNode(call.Args()[1])
	if err != nil {
		return "", err
	}

	switch call.FunctionName() {
	case operators.Equals:
		// Handle null comparisons specially
		if rightLiteral := call.Args()[1].AsLiteral(); rightLiteral != nil {
			if val := rightLiteral.Value(); fmt.Sprintf("%v", val) == "NULL_VALUE" {
				return fmt.Sprintf("%s IS NULL", left), nil
			}
		}
		return fmt.Sprintf("%s = %s", left, right), nil
	case operators.NotEquals:
		// Handle null comparisons specially
		if rightLiteral := call.Args()[1].AsLiteral(); rightLiteral != nil {
			if val := rightLiteral.Value(); fmt.Sprintf("%v", val) == "NULL_VALUE" {
				return fmt.Sprintf("%s IS NOT NULL", left), nil
			}
		}
		return fmt.Sprintf("%s != %s", left, right), nil
	case operators.Less:
		return fmt.Sprintf("%s < %s", left, right), nil
	case operators.LessEquals:
		return fmt.Sprintf("%s <= %s", left, right), nil
	case operators.Greater:
		return fmt.Sprintf("%s > %s", left, right), nil
	case operators.GreaterEquals:
		return fmt.Sprintf("%s >= %s", left, right), nil
	case operators.In:
		// Check if right side is a list literal
		if listExpr := call.Args()[1].AsList(); listExpr != nil {
			var elements []string
			for _, elem := range listExpr.Elements() {
				lit := elem.AsLiteral()
				if lit == nil {
					return "", fmt.Errorf("list elements must be literals")
				}
				value := lit.Value()
				param := t.addParam(value)
				elements = append(elements, param)
			}
			return fmt.Sprintf("%s = ANY(ARRAY[%s])", left, strings.Join(elements, ", ")), nil
		}
		return fmt.Sprintf("%s = ANY(%s)", left, right), nil
	case operators.LogicalAnd:
		return fmt.Sprintf("(%s AND %s)", left, right), nil
	case operators.LogicalOr:
		// Check if we need to wrap individual conditions
		leftNeedsWrap := !strings.HasPrefix(left, "(")
		rightNeedsWrap := !strings.HasPrefix(right, "(")

		if leftNeedsWrap {
			left = fmt.Sprintf("(%s)", left)
		}
		if rightNeedsWrap {
			right = fmt.Sprintf("(%s)", right)
		}

		// Always wrap the OR expression
		return fmt.Sprintf("(%s OR %s)", left, right), nil
	case operators.LogicalNot:
		// For logical not, only one operand
		if len(call.Args()) != 1 {
			return "", fmt.Errorf("logical not requires exactly one operand")
		}
		left, err := t.walkNode(call.Args()[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("NOT (%s)", left), nil
	default:
		return "", fmt.Errorf("unsupported binary operator: %s", call.FunctionName())
	}
}

// transpileInOperator handles the @in operator (from "in" keyword)
func (t *transpiler) transpileInOperator(call ast.CallExpr) (string, error) {
	if len(call.Args()) != 2 {
		return "", fmt.Errorf("in operator requires exactly two operands")
	}

	left, err := t.walkNode(call.Args()[0])
	if err != nil {
		return "", err
	}

	right, err := t.walkNode(call.Args()[1])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s = ANY(%s))", left, right), nil
}
