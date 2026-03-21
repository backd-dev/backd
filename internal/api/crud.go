package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/backd-dev/backd/internal/auth"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/filterql"
	"github.com/go-chi/chi/v5"
)

// identifierRegex validates SQL identifiers (table/column names)
var identifierRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// sanitizeIdentifier validates that a string is safe to use as a SQL identifier
func sanitizeIdentifier(name string) (string, error) {
	if !identifierRegex.MatchString(name) {
		return "", fmt.Errorf("invalid identifier: %q", name)
	}
	return name, nil
}

// RegisterCRUDRoutes registers all CRUD routes for collections
func RegisterCRUDRoutes(r chi.Router, deps *Deps) {
	// CRUD routes for all collections
	// Pattern: /api/v1/{app}/{collection}
	r.Route("/{collection}", func(r chi.Router) {
		// List collection items - GET /{collection}
		r.Get("/", makeCRUDHandler(deps, "LIST").Handle(deps))

		// Create item - POST /{collection}
		r.Post("/", makeCRUDHandler(deps, "CREATE").Handle(deps))

		// Get specific item - GET /{collection}/{id}
		r.Get("/{id}", makeCRUDHandler(deps, "GET").Handle(deps))

		// Update item (replace) - PUT /{collection}/{id}
		r.Put("/{id}", makeCRUDHandler(deps, "UPDATE").Handle(deps))

		// Update item (partial) - PATCH /{collection}/{id}
		r.Patch("/{id}", makeCRUDHandler(deps, "PATCH").Handle(deps))

		// Delete item - DELETE /{collection}/{id}
		r.Delete("/{id}", makeCRUDHandler(deps, "DELETE").Handle(deps))
	})
}

// CRUDOperation represents the type of CRUD operation
type CRUDOperation string

const (
	OP_LIST   CRUDOperation = "LIST"
	OP_CREATE CRUDOperation = "CREATE"
	OP_GET    CRUDOperation = "GET"
	OP_UPDATE CRUDOperation = "UPDATE"
	OP_PATCH  CRUDOperation = "PATCH"
	OP_DELETE CRUDOperation = "DELETE"
)

// makeCRUDHandler creates a handler for the specified CRUD operation
// This implements the 8-step CRUD pipeline
func makeCRUDHandler(deps *Deps, operation CRUDOperation) Handler {
	return func(r *http.Request, rc *RequestContext) (any, error) {
		// Step 1: ParseQueryParams → where, select, order, limit, offset
		queryParams, err := ParseQueryParams(r)
		if err != nil {
			return nil, err
		}

		// Step 2: EvaluatePolicy → sqlClause, params, defaults
		// This would integrate with auth package's EvaluatePolicy method
		// For now, placeholder implementation
		policyResult, err := deps.Auth.EvaluatePolicy(r.Context(), rc.AppName, chi.URLParam(r, "collection"), string(operation), &auth.RequestContext{
			UID:           rc.UserID,
			Meta:          rc.Meta,
			MetaApp:       rc.MetaApp,
			Authenticated: rc.Authenticated,
			KeyType:       rc.KeyType,
		})
		if err != nil {
			return nil, ErrForbidden("Access denied")
		}

		// Step 3: ApplyDefaults → overwrite payload with auth.uid/now()/etc
		var payload map[string]any
		if operation == OP_CREATE || operation == OP_UPDATE || operation == OP_PATCH {
			// Parse request body for operations that need payload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				return nil, ErrBadRequest("INVALID_JSON", "Invalid JSON body")
			}
		}

		// Apply defaults from policy result
		if len(policyResult.Defaults) > 0 && (operation == OP_CREATE || operation == OP_PATCH) {
			// Create auth context for defaults application
			authContext := &auth.RequestContext{
				UID:           rc.UserID,
				Meta:          rc.Meta,
				MetaApp:       rc.MetaApp,
				Authenticated: rc.Authenticated,
				KeyType:       rc.KeyType,
			}

			// Apply defaults and merge with payload
			defaults := deps.Auth.ApplyDefaults(policyResult.Defaults, authContext)
			for key, value := range defaults {
				if _, exists := payload[key]; !exists {
					payload[key] = value
				}
			}
		}

		// Step 4: StripColumns → remove keys not in policy.columns
		if operation == OP_CREATE || operation == OP_UPDATE || operation == OP_PATCH {
			payload = stripColumns(payload, policyResult.Columns)
		}

		// Step 5: ExecuteQuery → []map[string]any rows
		var result any
		switch operation {
		case OP_LIST:
			result, err = executeListQuery(deps, r, rc, queryParams, policyResult)
		case OP_CREATE:
			result, err = executeCreateQuery(deps, r, rc, payload, policyResult)
		case OP_GET:
			result, err = executeGetQuery(deps, r, rc, queryParams, policyResult)
		case OP_UPDATE:
			result, err = executeUpdateQuery(deps, r, rc, payload, policyResult)
		case OP_PATCH:
			result, err = executePatchQuery(deps, r, rc, payload, policyResult)
		case OP_DELETE:
			result, err = executeDeleteQuery(deps, r, rc, queryParams, policyResult)
		default:
			return nil, ErrInternal("Unsupported operation")
		}

		if err != nil {
			return nil, err
		}

		// Step 6: FilterResponseColumns → apply SELECT policy.columns
		if operation == OP_LIST || operation == OP_GET {
			result = filterResponseColumns(result, policyResult.Columns)
		}

		// Step 7: ResolveFiles → __file UUID → FileDescriptor
		if operation == OP_LIST || operation == OP_GET || operation == OP_CREATE {
			result = resolveFiles(deps, r, rc, result)
		}

		// Step 8: WriteResponse → writeList or writeSuccess
		// This is handled by the Handler wrapper
		return result, nil
	}
}

// filterResponseColumns removes keys from response data that are not in the allowed SELECT columns
func filterResponseColumns(result any, allowedColumns []string) any {
	if result == nil || len(allowedColumns) == 0 || allowedColumns[0] == "*" {
		return result // No filtering needed
	}

	// Create a set of allowed columns for efficient lookup
	allowed := make(map[string]bool)
	for _, col := range allowedColumns {
		allowed[col] = true
	}

	switch v := result.(type) {
	case map[string]any:
		// Check if this is a paginated response with data array
		if data, exists := v["data"]; exists {
			if dataSlice, ok := data.([]map[string]any); ok {
				filtered := make([]map[string]any, len(dataSlice))
				for i, record := range dataSlice {
					filtered[i] = filterSingleRecord(record, allowed)
				}
				v["data"] = filtered
				return v
			}
		}
		// Otherwise treat as single record
		return filterSingleRecord(v, allowed)
	case []map[string]any:
		// Array of records
		filtered := make([]map[string]any, len(v))
		for i, record := range v {
			filtered[i] = filterSingleRecord(record, allowed)
		}
		return filtered
	default:
		return result
	}
}

// resolveFiles converts __file UUIDs to FileDescriptor objects
func resolveFiles(deps *Deps, r *http.Request, rc *RequestContext, result any) any {
	if deps.Storage == nil {
		return result // No storage configured, return as-is
	}

	switch v := result.(type) {
	case map[string]any:
		// Check if this is a paginated response with data array
		if data, exists := v["data"]; exists {
			if dataSlice, ok := data.([]map[string]any); ok {
				// Use storage.ResolveFiles for batch resolution
				resolved, err := deps.Storage.ResolveFiles(r.Context(), rc.AppName, dataSlice)
				if err != nil {
					// If resolution fails, return original data
					return result
				}
				v["data"] = resolved
				return v
			}
		}
		// Otherwise treat as single record
		single := []map[string]any{v}
		resolved, err := deps.Storage.ResolveFiles(r.Context(), rc.AppName, single)
		if err == nil && len(resolved) > 0 {
			return resolved[0]
		}
		return v
	case []map[string]any:
		// Array of records
		resolved, err := deps.Storage.ResolveFiles(r.Context(), rc.AppName, v)
		if err != nil {
			// If resolution fails, return original data
			return result
		}
		return resolved
	default:
		return result
	}
}

// filterSingleRecord filters a single record based on allowed columns
func filterSingleRecord(record map[string]any, allowed map[string]bool) map[string]any {
	if record == nil {
		return record
	}

	filtered := make(map[string]any)
	for key, value := range record {
		if allowed[key] {
			filtered[key] = value
		}
	}
	return filtered
}
func stripColumns(payload map[string]any, allowedColumns []string) map[string]any {
	if payload == nil || len(allowedColumns) == 0 {
		return payload
	}

	// Check if wildcard is present
	for _, col := range allowedColumns {
		if col == "*" {
			return payload // Return original payload if wildcard is present
		}
	}

	// Create a set of allowed columns for efficient lookup
	allowed := make(map[string]bool)
	for _, col := range allowedColumns {
		allowed[col] = true
	}

	// Filter payload
	filtered := make(map[string]any)
	for key, value := range payload {
		if allowed[key] {
			filtered[key] = value
		}
	}

	return filtered
}

// Placeholder query execution functions
// These would be fully implemented with actual database queries

func executeListQuery(deps *Deps, r *http.Request, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	collection, err := sanitizeIdentifier(chi.URLParam(r, "collection"))
	if err != nil {
		return nil, ErrBadRequest("INVALID_COLLECTION", err.Error())
	}

	// Start building WHERE parts and params from the RLS policy
	whereParts := []string{policyResult.SQLClause}
	args := append([]any{}, policyResult.Params...)

	// Integrate filterql for user-supplied where clause
	if len(qp.Where) > 0 {
		filterClause, filterParams, err := filterql.Transpile(qp.Where)
		if err != nil {
			return nil, ErrBadRequest("INVALID_WHERE", err.Error())
		}
		if filterClause != "" {
			whereParts = append(whereParts, filterClause)
			args = append(args, filterParams...)
		}
	}

	fullWhere := strings.Join(whereParts, " AND ")
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s", collection, fullWhere)

	// Add ORDER BY clause (sanitize each field)
	if len(qp.Order) > 0 {
		var orderParts []string
		for _, o := range qp.Order {
			orderParts = append(orderParts, o) // TODO: validate order fields
		}
		query += " ORDER BY " + strings.Join(orderParts, ", ")
	}

	// Add LIMIT and OFFSET
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", qp.Limit, qp.Offset)

	// Execute query
	rows, err := deps.DB.Query(r.Context(), rc.AppName, query, args...)
	if err != nil {
		return nil, ErrInternal("Database query failed")
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", collection, fullWhere)
	countResult, err := deps.DB.QueryOne(r.Context(), rc.AppName, countQuery, args...)
	if err != nil {
		return nil, ErrInternal("Failed to get total count")
	}

	var count int64
	if countResult != nil {
		if countVal, ok := countResult["count"].(int64); ok {
			count = countVal
		}
	}

	// Return paginated response
	return map[string]any{
		"data":   rows,
		"count":  count,
		"limit":  qp.Limit,
		"offset": qp.Offset,
	}, nil
}

func executeCreateQuery(deps *Deps, r *http.Request, rc *RequestContext, payload map[string]any, policyResult auth.PolicyResult) (any, error) {
	collection, err := sanitizeIdentifier(chi.URLParam(r, "collection"))
	if err != nil {
		return nil, ErrBadRequest("INVALID_COLLECTION", err.Error())
	}
	id := db.NewXID()

	// Build INSERT query with policy clause
	// Get column names and values from payload
	var columns []string
	var values []any
	var placeholders []string

	for key, value := range payload {
		columns = append(columns, key)
		values = append(values, value)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)))
	}

	// Add ID and created_at
	columns = append(columns, "id", "created_at")
	values = append(values, id, "NOW()")
	placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)), "NOW()")

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", collection, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	// Execute query
	err = deps.DB.Exec(r.Context(), rc.AppName, query, values...)
	if err != nil {
		return nil, ErrInternal("Failed to create record")
	}

	// Return the created record
	return map[string]any{"id": id}, nil
}

func executeGetQuery(deps *Deps, r *http.Request, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	collection, err := sanitizeIdentifier(chi.URLParam(r, "collection"))
	if err != nil {
		return nil, ErrBadRequest("INVALID_COLLECTION", err.Error())
	}
	id := chi.URLParam(r, "id")

	// Build SELECT query with policy clause
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s AND id = $%d", collection, policyResult.SQLClause, len(policyResult.Params)+1)
	args := append(policyResult.Params, id)

	// Execute query
	result, err := deps.DB.QueryOne(r.Context(), rc.AppName, query, args...)
	if err != nil {
		return nil, ErrNotFound("Record not found")
	}

	return result, nil
}

func executeUpdateQuery(deps *Deps, r *http.Request, rc *RequestContext, payload map[string]any, policyResult auth.PolicyResult) (any, error) {
	collection, err := sanitizeIdentifier(chi.URLParam(r, "collection"))
	if err != nil {
		return nil, ErrBadRequest("INVALID_COLLECTION", err.Error())
	}
	id := chi.URLParam(r, "id")

	// Build UPDATE query with policy clause
	var setClauses []string
	var values []any

	for key, value := range payload {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, len(values)+1))
		values = append(values, value)
	}

	// Add updated_at
	setClauses = append(setClauses, "updated_at = NOW()")

	// Add WHERE clause parameters
	values = append(values, policyResult.Params...)
	values = append(values, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s AND id = $%d", collection, strings.Join(setClauses, ", "), policyResult.SQLClause, len(values))

	// Execute query
	err = deps.DB.Exec(r.Context(), rc.AppName, query, values...)
	if err != nil {
		return nil, ErrInternal("Failed to update record")
	}

	// Return the updated record
	return map[string]any{"id": id}, nil
}

func executePatchQuery(deps *Deps, r *http.Request, rc *RequestContext, payload map[string]any, policyResult auth.PolicyResult) (any, error) {
	// For PATCH, we use the same logic as UPDATE but only update provided fields
	return executeUpdateQuery(deps, r, rc, payload, policyResult)
}

func executeDeleteQuery(deps *Deps, r *http.Request, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	collection, err := sanitizeIdentifier(chi.URLParam(r, "collection"))
	if err != nil {
		return nil, ErrBadRequest("INVALID_COLLECTION", err.Error())
	}
	id := chi.URLParam(r, "id")

	// Build DELETE query with policy clause
	query := fmt.Sprintf("DELETE FROM %s WHERE %s AND id = $%d", collection, policyResult.SQLClause, len(policyResult.Params)+1)
	args := append(policyResult.Params, id)

	// Execute query
	err = deps.DB.Exec(r.Context(), rc.AppName, query, args...)
	if err != nil {
		return nil, ErrInternal("Failed to delete record")
	}

	// No content for successful delete
	return nil, nil
}
