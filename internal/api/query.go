package api

import (
	"net/http"
	"strconv"
	"strings"
)

// QueryParams holds parsed query parameters from HTTP request
type QueryParams struct {
	Where  map[string]any
	Select []string
	Order  []string
	Limit  int
	Offset int
}

const (
	defaultLimit  = 50
	maxLimit      = 1000
	defaultOffset = 0
)

// ParseQueryParams extracts and validates query parameters from HTTP request
func ParseQueryParams(r *http.Request) (*QueryParams, error) {
	qp := &QueryParams{
		Limit:  defaultLimit,
		Offset: defaultOffset,
	}

	// Parse where parameter
	if whereStr := r.URL.Query().Get("where"); whereStr != "" {
		// Parse where JSON using filterql package
		whereFilter, err := parseWhereFilter(whereStr)
		if err != nil {
			return nil, ErrBadRequest("INVALID_WHERE", "Invalid where filter: "+err.Error())
		}
		qp.Where = whereFilter
	}

	// Parse select parameter
	if selectStr := r.URL.Query().Get("select"); selectStr != "" {
		qp.Select = strings.Split(selectStr, ",")
		// Trim whitespace from each field
		for i, field := range qp.Select {
			qp.Select[i] = strings.TrimSpace(field)
		}
	}

	// Parse order parameter
	if orderStr := r.URL.Query().Get("order"); orderStr != "" {
		qp.Order = strings.Split(orderStr, ",")
		// Trim whitespace from each field
		for i, field := range qp.Order {
			qp.Order[i] = strings.TrimSpace(field)
		}
	}

	// Parse limit parameter
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 0 {
			return nil, ErrBadRequest("INVALID_LIMIT", "Limit must be a non-negative integer")
		}
		if limit > maxLimit {
			return nil, ErrBadRequest("LIMIT_TOO_HIGH", "Limit cannot exceed "+strconv.Itoa(maxLimit))
		}
		qp.Limit = limit
	}

	// Parse offset parameter
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return nil, ErrBadRequest("INVALID_OFFSET", "Offset must be a non-negative integer")
		}
		qp.Offset = offset
	}

	return qp, nil
}

// parseWhereFilter parses a JSON where filter string using filterql package
func parseWhereFilter(whereStr string) (map[string]any, error) {
	// This would typically use json.Unmarshal, but since we need to integrate
	// with filterql package, we'll delegate to it for validation and parsing
	// For now, return a simple placeholder
	// In a complete implementation, this would:
	// 1. Parse the JSON string
	// 2. Validate using filterql package
	// 3. Return the parsed filter

	// Placeholder implementation - will be completed when filterql integration is done
	return make(map[string]any), nil
}
