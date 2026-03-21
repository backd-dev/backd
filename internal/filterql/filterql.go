package filterql

import (
	"errors"
	"fmt"
)

// Errors
var (
	ErrInvalidOperator = errors.New("invalid operator")
	ErrInvalidFilter   = errors.New("invalid filter structure")
)

// Transpile converts a user-supplied where JSON filter into SQL WHERE clause
func Transpile(filter map[string]any) (clause string, params []any, err error) {
	if len(filter) == 0 {
		return "", nil, nil
	}

	counter := 0
	return transpile(filter, &counter)
}

// transpile handles the recursive conversion of filter to SQL
func transpile(filter map[string]any, counter *int) (string, []any, error) {
	var clauses []string
	var allParams []any

	for key, value := range filter {
		switch {
		case key == "$and":
			clause, params, err := handleAnd(value, counter)
			if err != nil {
				return "", nil, err
			}
			clauses = append(clauses, clause)
			allParams = append(allParams, params...)
		case key == "$or":
			clause, params, err := handleOr(value, counter)
			if err != nil {
				return "", nil, err
			}
			clauses = append(clauses, clause)
			allParams = append(allParams, params...)
		case key == "$not":
			clause, params, err := handleNot(value, counter)
			if err != nil {
				return "", nil, err
			}
			clauses = append(clauses, clause)
			allParams = append(allParams, params...)
		default:
			// Regular field name with operator object
			fieldOps, ok := value.(map[string]any)
			if !ok {
				return "", nil, fmt.Errorf("%w: field %q must have operator object", ErrInvalidFilter, key)
			}

			for op, opValue := range fieldOps {
				clause, params, err := handleFieldOperator(key, op, opValue, counter)
				if err != nil {
					return "", nil, err
				}
				clauses = append(clauses, clause)
				allParams = append(allParams, params...)
			}
		}
	}

	if len(clauses) == 0 {
		return "", nil, nil
	}

	if len(clauses) == 1 {
		return clauses[0], allParams, nil
	}

	// Join multiple field clauses with AND
	joined := clauses[0]
	for i := 1; i < len(clauses); i++ {
		joined = fmt.Sprintf("%s AND %s", joined, clauses[i])
	}

	return fmt.Sprintf("(%s)", joined), allParams, nil
}

// handleFieldOperator processes field-level operators
func handleFieldOperator(field, operator string, value any, counter *int) (string, []any, error) {
	switch operator {
	case "$eq":
		return handleEq(field, value, counter)
	case "$ne":
		return handleNe(field, value, counter)
	case "$gt":
		return handleGt(field, value, counter)
	case "$gte":
		return handleGte(field, value, counter)
	case "$lt":
		return handleLt(field, value, counter)
	case "$lte":
		return handleLte(field, value, counter)
	case "$in":
		return handleIn(field, value, counter)
	case "$nin":
		return handleNin(field, value, counter)
	case "$between":
		return handleBetween(field, value, counter)
	case "$null":
		return handleNull(field, value, counter)
	case "$notNull":
		return handleNotNull(field, value, counter)
	case "$like":
		return handleLike(field, value, counter)
	case "$ilike":
		return handleIlike(field, value, counter)
	case "$contains":
		return handleContains(field, value, counter)
	default:
		return "", nil, fmt.Errorf("%w: %s", ErrInvalidOperator, operator)
	}
}
