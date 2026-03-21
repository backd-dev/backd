package api

import (
	"net/http"

	"github.com/backd-dev/backd/internal/auth"
	"github.com/go-chi/chi/v5"
)

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
		// This would apply defaults from policy result
		// Placeholder implementation

		// Step 4: StripColumns → remove keys not in policy.columns
		// This would filter request payload based on policy
		// Placeholder implementation

		// Step 5: ExecuteQuery → []map[string]any rows
		var result any
		switch operation {
		case OP_LIST:
			result, err = executeListQuery(deps, rc, queryParams, policyResult)
		case OP_CREATE:
			result, err = executeCreateQuery(deps, rc, queryParams, policyResult)
		case OP_GET:
			result, err = executeGetQuery(deps, rc, queryParams, policyResult)
		case OP_UPDATE:
			result, err = executeUpdateQuery(deps, rc, queryParams, policyResult)
		case OP_PATCH:
			result, err = executePatchQuery(deps, rc, queryParams, policyResult)
		case OP_DELETE:
			result, err = executeDeleteQuery(deps, rc, queryParams, policyResult)
		default:
			return nil, ErrInternal("Unsupported operation")
		}

		if err != nil {
			return nil, err
		}

		// Step 6: FilterResponseColumns → apply SELECT policy.columns
		// This would filter response based on policy
		// Placeholder implementation

		// Step 7: ResolveFiles → __file UUID → FileDescriptor
		// This would integrate with storage package to resolve file references
		// Placeholder implementation

		// Step 8: WriteResponse → writeList or writeSuccess
		// This is handled by the Handler wrapper
		return result, nil
	}
}

// Placeholder query execution functions
// These would be fully implemented with actual database queries

func executeListQuery(deps *Deps, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	// Placeholder: would execute SELECT query with policy clause
	return []map[string]any{}, nil
}

func executeCreateQuery(deps *Deps, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	// Placeholder: would execute INSERT query with policy defaults
	return map[string]any{"id": "new-id"}, nil
}

func executeGetQuery(deps *Deps, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	// Placeholder: would execute SELECT query for single item
	return map[string]any{"id": "item-id"}, nil
}

func executeUpdateQuery(deps *Deps, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	// Placeholder: would execute UPDATE query
	return map[string]any{"id": "item-id"}, nil
}

func executePatchQuery(deps *Deps, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	// Placeholder: would execute PATCH query
	return map[string]any{"id": "item-id"}, nil
}

func executeDeleteQuery(deps *Deps, rc *RequestContext, qp *QueryParams, policyResult auth.PolicyResult) (any, error) {
	// Placeholder: would execute DELETE query
	return nil, nil // No content for successful delete
}
