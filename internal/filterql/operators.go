package filterql

import (
	"fmt"
)

// addParam increments the counter and returns the parameter placeholder
func addParam(counter *int, params []any, value any) (string, []any) {
	*counter++
	params = append(params, value)
	return fmt.Sprintf("$%d", *counter), params
}

// Comparison operators

func handleEq(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s = %s", field, placeholder), params, nil
}

func handleNe(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s != %s", field, placeholder), params, nil
}

func handleGt(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s > %s", field, placeholder), params, nil
}

func handleGte(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s >= %s", field, placeholder), params, nil
}

func handleLt(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s < %s", field, placeholder), params, nil
}

func handleLte(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s <= %s", field, placeholder), params, nil
}

// Set & Range operators

func handleIn(field string, value any, counter *int) (string, []any, error) {
	valueSlice, ok := value.([]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: $in operator requires array value", ErrInvalidFilter)
	}

	placeholder, params := addParam(counter, nil, valueSlice)
	return fmt.Sprintf("%s = ANY(%s)", field, placeholder), params, nil
}

func handleNin(field string, value any, counter *int) (string, []any, error) {
	valueSlice, ok := value.([]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: $nin operator requires array value", ErrInvalidFilter)
	}

	placeholder, params := addParam(counter, nil, valueSlice)
	return fmt.Sprintf("%s != ALL(%s)", field, placeholder), params, nil
}

func handleBetween(field string, value any, counter *int) (string, []any, error) {
	valueSlice, ok := value.([]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: $between operator requires array with exactly 2 values", ErrInvalidFilter)
	}

	if len(valueSlice) != 2 {
		return "", nil, fmt.Errorf("%w: $between operator requires array with exactly 2 values", ErrInvalidFilter)
	}

	var params []any
	placeholder1, params1 := addParam(counter, nil, valueSlice[0])
	placeholder2, params2 := addParam(counter, params1, valueSlice[1])
	params = params2

	return fmt.Sprintf("%s BETWEEN %s AND %s", field, placeholder1, placeholder2), params, nil
}

// Null operators

func handleNull(field string, value any, counter *int) (string, []any, error) {
	isTrue, ok := value.(bool)
	if !ok || !isTrue {
		return "", nil, fmt.Errorf("%w: $null operator requires true", ErrInvalidFilter)
	}
	return fmt.Sprintf("%s IS NULL", field), nil, nil
}

func handleNotNull(field string, value any, counter *int) (string, []any, error) {
	isTrue, ok := value.(bool)
	if !ok || !isTrue {
		return "", nil, fmt.Errorf("%w: $notNull operator requires true", ErrInvalidFilter)
	}
	return fmt.Sprintf("%s IS NOT NULL", field), nil, nil
}

// Text operators

func handleLike(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s LIKE %s", field, placeholder), params, nil
}

func handleIlike(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s ILIKE %s", field, placeholder), params, nil
}

func handleContains(field string, value any, counter *int) (string, []any, error) {
	placeholder, params := addParam(counter, nil, value)
	return fmt.Sprintf("%s ILIKE '%%' || %s || '%%'", field, placeholder), params, nil
}

// Logical operators (recursive)

func handleAnd(value any, counter *int) (string, []any, error) {
	filters, ok := value.([]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: $and operator requires array of filter objects", ErrInvalidFilter)
	}

	if len(filters) == 0 {
		return "", nil, fmt.Errorf("%w: $and operator requires at least one filter", ErrInvalidFilter)
	}

	var clauses []string
	var allParams []any

	for _, filter := range filters {
		filterMap, ok := filter.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("%w: $and operator items must be filter objects", ErrInvalidFilter)
		}

		clause, params, err := transpile(filterMap, counter)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		allParams = append(allParams, params...)
	}

	if len(clauses) == 1 {
		return clauses[0], allParams, nil
	}

	// Join multiple clauses with AND
	joined := clauses[0]
	for i := 1; i < len(clauses); i++ {
		joined = fmt.Sprintf("%s AND %s", joined, clauses[i])
	}

	return fmt.Sprintf("(%s)", joined), allParams, nil
}

func handleOr(value any, counter *int) (string, []any, error) {
	filters, ok := value.([]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: $or operator requires array of filter objects", ErrInvalidFilter)
	}

	if len(filters) == 0 {
		return "", nil, fmt.Errorf("%w: $or operator requires at least one filter", ErrInvalidFilter)
	}

	var clauses []string
	var allParams []any

	for _, filter := range filters {
		filterMap, ok := filter.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("%w: $or operator items must be filter objects", ErrInvalidFilter)
		}

		clause, params, err := transpile(filterMap, counter)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		allParams = append(allParams, params...)
	}

	if len(clauses) == 1 {
		return clauses[0], allParams, nil
	}

	// Join multiple clauses with OR
	joined := clauses[0]
	for i := 1; i < len(clauses); i++ {
		joined = fmt.Sprintf("%s OR %s", joined, clauses[i])
	}

	return fmt.Sprintf("(%s)", joined), allParams, nil
}

func handleNot(value any, counter *int) (string, []any, error) {
	filter, ok := value.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("%w: $not operator requires a single filter object", ErrInvalidFilter)
	}

	clause, params, err := transpile(filter, counter)
	if err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("NOT %s", clause), params, nil
}
