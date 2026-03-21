package filterql

import (
	"strings"
	"testing"
)

func TestTranspile_ComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		filter   map[string]any
		expected string
		params   []any
	}{
		{
			name:     "$eq",
			filter:   map[string]any{"name": map[string]any{"$eq": "John"}},
			expected: "name = $1",
			params:   []any{"John"},
		},
		{
			name:     "$ne",
			filter:   map[string]any{"age": map[string]any{"$ne": 25}},
			expected: "age != $1",
			params:   []any{25},
		},
		{
			name:     "$gt",
			filter:   map[string]any{"score": map[string]any{"$gt": 90.5}},
			expected: "score > $1",
			params:   []any{90.5},
		},
		{
			name:     "$gte",
			filter:   map[string]any{"count": map[string]any{"$gte": 10}},
			expected: "count >= $1",
			params:   []any{10},
		},
		{
			name:     "$lt",
			filter:   map[string]any{"price": map[string]any{"$lt": 100}},
			expected: "price < $1",
			params:   []any{100},
		},
		{
			name:     "$lte",
			filter:   map[string]any{"quantity": map[string]any{"$lte": 5}},
			expected: "quantity <= $1",
			params:   []any{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, params, err := Transpile(tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if clause != tt.expected {
				t.Errorf("expected clause %q, got %q", tt.expected, clause)
			}
			if len(params) != len(tt.params) {
				t.Errorf("expected %d params, got %d", len(tt.params), len(params))
			}
			for i, p := range params {
				if i >= len(tt.params) {
					t.Errorf("param %d: unexpected param %v", i, p)
					continue
				}
				// Special handling for slice comparison
				if slice1, ok1 := p.([]any); ok1 {
					if slice2, ok2 := tt.params[i].([]any); ok2 {
						if len(slice1) != len(slice2) {
							t.Errorf("param %d: expected slice length %d, got %d", i, len(slice2), len(slice1))
							continue
						}
						for j, val := range slice1 {
							if val != slice2[j] {
								t.Errorf("param %d[%d]: expected %v, got %v", i, j, slice2[j], val)
							}
						}
						continue
					}
				}
				if p != tt.params[i] {
					t.Errorf("param %d: expected %v, got %v", i, tt.params[i], p)
				}
			}
		})
	}
}

func TestTranspile_SetAndRangeOperators(t *testing.T) {
	tests := []struct {
		name     string
		filter   map[string]any
		expected string
		params   []any
	}{
		{
			name:     "$in",
			filter:   map[string]any{"status": map[string]any{"$in": []any{"active", "pending"}}},
			expected: "status = ANY($1)",
			params:   []any{[]any{"active", "pending"}},
		},
		{
			name:     "$nin",
			filter:   map[string]any{"type": map[string]any{"$nin": []any{"deleted", "archived"}}},
			expected: "type != ALL($1)",
			params:   []any{[]any{"deleted", "archived"}},
		},
		{
			name:     "$between",
			filter:   map[string]any{"created_at": map[string]any{"$between": []any{"2023-01-01", "2023-12-31"}}},
			expected: "created_at BETWEEN $1 AND $2",
			params:   []any{"2023-01-01", "2023-12-31"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, params, err := Transpile(tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if clause != tt.expected {
				t.Errorf("expected clause %q, got %q", tt.expected, clause)
			}
			if len(params) != len(tt.params) {
				t.Errorf("expected %d params, got %d", len(tt.params), len(params))
			}
			for i, p := range params {
				if i >= len(tt.params) {
					t.Errorf("param %d: unexpected param %v", i, p)
					continue
				}
				// Special handling for slice comparison
				if slice1, ok1 := p.([]any); ok1 {
					if slice2, ok2 := tt.params[i].([]any); ok2 {
						if len(slice1) != len(slice2) {
							t.Errorf("param %d: expected slice length %d, got %d", i, len(slice2), len(slice1))
							continue
						}
						for j, val := range slice1 {
							if val != slice2[j] {
								t.Errorf("param %d[%d]: expected %v, got %v", i, j, slice2[j], val)
							}
						}
						continue
					}
				}
				if p != tt.params[i] {
					t.Errorf("param %d: expected %v, got %v", i, tt.params[i], p)
				}
			}
		})
	}
}

func TestTranspile_NullOperators(t *testing.T) {
	tests := []struct {
		name     string
		filter   map[string]any
		expected string
		params   []any
	}{
		{
			name:     "$null",
			filter:   map[string]any{"deleted_at": map[string]any{"$null": true}},
			expected: "deleted_at IS NULL",
			params:   nil,
		},
		{
			name:     "$notNull",
			filter:   map[string]any{"name": map[string]any{"$notNull": true}},
			expected: "name IS NOT NULL",
			params:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, params, err := Transpile(tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if clause != tt.expected {
				t.Errorf("expected clause %q, got %q", tt.expected, clause)
			}
			if len(params) != len(tt.params) {
				t.Errorf("expected %d params, got %d", len(tt.params), len(params))
			}
		})
	}
}

func TestTranspile_TextOperators(t *testing.T) {
	tests := []struct {
		name     string
		filter   map[string]any
		expected string
		params   []any
	}{
		{
			name:     "$like",
			filter:   map[string]any{"title": map[string]any{"$like": "test%"}},
			expected: "title LIKE $1",
			params:   []any{"test%"},
		},
		{
			name:     "$ilike",
			filter:   map[string]any{"description": map[string]any{"$ilike": "%search%"}},
			expected: "description ILIKE $1",
			params:   []any{"%search%"},
		},
		{
			name:     "$contains",
			filter:   map[string]any{"content": map[string]any{"$contains": "keyword"}},
			expected: "content ILIKE '%' || $1 || '%'",
			params:   []any{"keyword"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, params, err := Transpile(tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if clause != tt.expected {
				t.Errorf("expected clause %q, got %q", tt.expected, clause)
			}
			if len(params) != len(tt.params) {
				t.Errorf("expected %d params, got %d", len(tt.params), len(params))
			}
			for i, p := range params {
				if i >= len(tt.params) {
					t.Errorf("param %d: unexpected param %v", i, p)
					continue
				}
				// Special handling for slice comparison
				if slice1, ok1 := p.([]any); ok1 {
					if slice2, ok2 := tt.params[i].([]any); ok2 {
						if len(slice1) != len(slice2) {
							t.Errorf("param %d: expected slice length %d, got %d", i, len(slice2), len(slice1))
							continue
						}
						for j, val := range slice1 {
							if val != slice2[j] {
								t.Errorf("param %d[%d]: expected %v, got %v", i, j, slice2[j], val)
							}
						}
						continue
					}
				}
				if p != tt.params[i] {
					t.Errorf("param %d: expected %v, got %v", i, tt.params[i], p)
				}
			}
		})
	}
}

func TestTranspile_LogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		filter   map[string]any
		expected string
		params   []any
	}{
		{
			name: "$and single",
			filter: map[string]any{
				"$and": []any{
					map[string]any{"name": map[string]any{"$eq": "John"}},
				},
			},
			expected: "name = $1",
			params:   []any{"John"},
		},
		{
			name: "$and multiple",
			filter: map[string]any{
				"$and": []any{
					map[string]any{"name": map[string]any{"$eq": "John"}},
					map[string]any{"age": map[string]any{"$gt": 25}},
				},
			},
			expected: "(name = $1 AND age > $2)",
			params:   []any{"John", 25},
		},
		{
			name: "$or single",
			filter: map[string]any{
				"$or": []any{
					map[string]any{"status": map[string]any{"$eq": "active"}},
				},
			},
			expected: "status = $1",
			params:   []any{"active"},
		},
		{
			name: "$or multiple",
			filter: map[string]any{
				"$or": []any{
					map[string]any{"status": map[string]any{"$eq": "active"}},
					map[string]any{"status": map[string]any{"$eq": "pending"}},
				},
			},
			expected: "(status = $1 OR status = $2)",
			params:   []any{"active", "pending"},
		},
		{
			name: "$not",
			filter: map[string]any{
				"$not": map[string]any{"deleted": map[string]any{"$null": true}},
			},
			expected: "NOT deleted IS NULL",
			params:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, params, err := Transpile(tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if clause != tt.expected {
				t.Errorf("expected clause %q, got %q", tt.expected, clause)
			}
			if len(params) != len(tt.params) {
				t.Errorf("expected %d params, got %d", len(tt.params), len(params))
			}
			for i, p := range params {
				if i >= len(tt.params) {
					t.Errorf("param %d: unexpected param %v", i, p)
					continue
				}
				// Special handling for slice comparison
				if slice1, ok1 := p.([]any); ok1 {
					if slice2, ok2 := tt.params[i].([]any); ok2 {
						if len(slice1) != len(slice2) {
							t.Errorf("param %d: expected slice length %d, got %d", i, len(slice2), len(slice1))
							continue
						}
						for j, val := range slice1 {
							if val != slice2[j] {
								t.Errorf("param %d[%d]: expected %v, got %v", i, j, slice2[j], val)
							}
						}
						continue
					}
				}
				if p != tt.params[i] {
					t.Errorf("param %d: expected %v, got %v", i, tt.params[i], p)
				}
			}
		})
	}
}

func TestTranspile_MultipleFields(t *testing.T) {
	filter := map[string]any{
		"name":   map[string]any{"$eq": "John"},
		"age":    map[string]any{"$gt": 25},
		"status": map[string]any{"$eq": "active"},
	}

	clause, params, err := Transpile(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Map iteration order is non-deterministic, so check structural properties
	if len(params) != 3 {
		t.Errorf("expected 3 params, got %d", len(params))
	}

	// Clause should be wrapped in parens and contain AND
	if clause[0] != '(' || clause[len(clause)-1] != ')' {
		t.Errorf("expected clause wrapped in parens, got %q", clause)
	}

	// All three field comparisons must appear somewhere in the clause
	for _, part := range []string{"name = ", "age > ", "status = "} {
		if !strings.Contains(clause, part) {
			t.Errorf("expected clause to contain %q, got %q", part, clause)
		}
	}
}

func TestTranspile_ParameterCounterContinuity(t *testing.T) {
	filter := map[string]any{
		"$and": []any{
			map[string]any{"name": map[string]any{"$eq": "John"}},
			map[string]any{"age": map[string]any{"$gt": 25}},
			map[string]any{
				"$or": []any{
					map[string]any{"status": map[string]any{"$eq": "active"}},
					map[string]any{"status": map[string]any{"$eq": "pending"}},
				},
			},
		},
	}

	_, params, err := Transpile(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have parameters $1, $2, $3, $4 in order
	expectedParams := []any{"John", 25, "active", "pending"}
	if len(params) != len(expectedParams) {
		t.Errorf("expected %d params, got %d", len(expectedParams), len(params))
	}
	for i, p := range params {
		if p != expectedParams[i] {
			t.Errorf("param %d: expected %v, got %v", i, expectedParams[i], p)
		}
	}
}

func TestTranspile_EmptyFilter(t *testing.T) {
	clause, params, err := Transpile(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "" {
		t.Errorf("expected empty clause, got %q", clause)
	}
	if params != nil {
		t.Errorf("expected nil params, got %v", params)
	}
}

func TestTranspile_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		filter      map[string]any
		expectedErr error
	}{
		{
			name:        "invalid operator",
			filter:      map[string]any{"name": map[string]any{"$invalid": "test"}},
			expectedErr: ErrInvalidOperator,
		},
		{
			name:        "field without operator object",
			filter:      map[string]any{"name": "test"},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$in with non-array",
			filter: map[string]any{
				"status": map[string]any{"$in": "active"},
			},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$between with wrong array size",
			filter: map[string]any{
				"date": map[string]any{"$between": []any{"2023-01-01"}},
			},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$null with false",
			filter: map[string]any{
				"deleted": map[string]any{"$null": false},
			},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$and with non-array",
			filter: map[string]any{
				"$and": "invalid",
			},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$and with empty array",
			filter: map[string]any{
				"$and": []any{},
			},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$and with non-object items",
			filter: map[string]any{
				"$and": []any{"invalid"},
			},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$or with non-array",
			filter: map[string]any{
				"$or": "invalid",
			},
			expectedErr: ErrInvalidFilter,
		},
		{
			name: "$not with non-object",
			filter: map[string]any{
				"$not": "invalid",
			},
			expectedErr: ErrInvalidFilter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Transpile(tt.filter)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}
			if !containsError(err, tt.expectedErr) {
				t.Errorf("expected error containing %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func containsError(err, expected error) bool {
	errMsg := err.Error()
	expMsg := expected.Error()

	// Check for exact match
	if errMsg == expMsg {
		return true
	}

	// Check for substring match for specific error types
	if expected == ErrInvalidOperator && errMsg == "invalid operator: $invalid" {
		return true
	}
	if expected == ErrInvalidFilter &&
		(errMsg == "invalid filter structure: field \"name\" must have operator object" ||
			errMsg == "invalid filter structure: $in operator requires array value" ||
			errMsg == "invalid filter structure: $between operator requires array with exactly 2 values" ||
			errMsg == "invalid filter structure: $null operator requires true" ||
			errMsg == "invalid filter structure: $and operator requires array of filter objects" ||
			errMsg == "invalid filter structure: $and operator requires at least one filter" ||
			errMsg == "invalid filter structure: $and operator items must be filter objects" ||
			errMsg == "invalid filter structure: $or operator requires array of filter objects" ||
			errMsg == "invalid filter structure: $not operator requires a single filter object") {
		return true
	}

	return false
}
