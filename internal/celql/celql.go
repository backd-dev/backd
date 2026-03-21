package celql

import (
	"fmt"

	"github.com/google/cel-go/cel"
)

// AuthContext provides authentication and authorization context for CEL expression evaluation
type AuthContext struct {
	UID           string         // User ID from session
	Meta          map[string]any // Global metadata (metadata._)
	MetaApp       map[string]any // App-specific metadata (metadata.<app_name>)
	Authenticated bool           // Whether the request is authenticated
	KeyType       string         // Type of API key used (if any)
}

// TranspileResult contains the transpiled SQL and its bound parameters
type TranspileResult struct {
	SQL    string // SQL WHERE clause with $N parameter placeholders
	Params []any  // Parameter values in order matching $N placeholders
}

// CELQL interface defines the public API for CEL expression processing
type CELQL interface {
	Parse(expression string) (*cel.Ast, error)
	Validate(ast *cel.Ast) error
	Transpile(ast *cel.Ast, auth AuthContext) (TranspileResult, error)
}

// celqlImpl implements the CELQL interface
type celqlImpl struct {
	env *cel.Env
}

// New creates a new CELQL instance
func New() (CELQL, error) {
	env, err := cel.NewEnv(
		cel.Variable("row", cel.DynType),
		cel.Variable("auth", cel.DynType),
		cel.Function("has", cel.Overload("has", []*cel.Type{cel.DynType}, cel.BoolType)),
		cel.Function("now", cel.Overload("now", []*cel.Type{}, cel.TimestampType)),
		cel.Function("today", cel.Overload("today", []*cel.Type{}, cel.StringType)),
	)
	if err != nil {
		return nil, fmt.Errorf("celql.New: %w", err)
	}

	return &celqlImpl{
		env: env,
	}, nil
}

// Parse parses a CEL expression string into an AST
func (c *celqlImpl) Parse(expression string) (*cel.Ast, error) {
	ast, iss := c.env.Parse(expression)
	if iss.Err() != nil {
		return nil, fmt.Errorf("celql.Parse: %w", iss.Err())
	}
	return ast, nil
}

// Validate checks if the AST contains only supported constructs
func (c *celqlImpl) Validate(ast *cel.Ast) error {
	validator := newValidator()
	if err := validator.validate(ast); err != nil {
		return fmt.Errorf("celql.Validate: %w", err)
	}
	return nil
}

// Transpile converts a validated CEL AST into SQL with bound parameters
func (c *celqlImpl) Transpile(ast *cel.Ast, auth AuthContext) (TranspileResult, error) {
	validator := newValidator()
	if err := validator.validate(ast); err != nil {
		return TranspileResult{}, fmt.Errorf("celql.Transpile: validation failed: %w", err)
	}

	transpiler := newTranspiler(auth)
	result, err := transpiler.transpile(ast)
	if err != nil {
		return TranspileResult{}, fmt.Errorf("celql.Transpile: %w", err)
	}

	return result, nil
}
