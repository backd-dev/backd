package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/backd-dev/backd/internal/celql"
	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
)

// LoadPolicies loads and caches RLS policies for an app
func (a *authImpl) LoadPolicies(ctx context.Context, appName string, cfg *config.AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Clear existing cache for this app
	a.clearAppCache(appName)

	// Delete existing policies from database
	deleteQuery := `DELETE FROM _policies WHERE app_name = $1`
	err := a.db.Exec(ctx, appName, deleteQuery, appName)
	if err != nil {
		slog.Error("failed to delete existing policies", "app", appName, "error", err)
		return fmt.Errorf("auth.LoadPolicies: %w", err)
	}

	// Insert new policies and cache them
	for tableName, tablePolicies := range cfg.Policies {
		for operation, policyEntry := range tablePolicies {
			// Parse and validate CEL expression
			ast, err := a.celql.Parse(policyEntry.Expression)
			if err != nil {
				slog.Error("failed to parse policy expression", "app", appName, "table", tableName, "operation", operation, "expression", policyEntry.Expression, "error", err)
				return fmt.Errorf("auth.LoadPolicies: parse failed for %s.%s: %w", tableName, operation, err)
			}

			err = a.celql.Validate(ast)
			if err != nil {
				slog.Error("failed to validate policy expression", "app", appName, "table", tableName, "operation", operation, "expression", policyEntry.Expression, "error", err)
				return fmt.Errorf("auth.LoadPolicies: validation failed for %s.%s: %w", tableName, operation, err)
			}

			// Cache the parsed AST
			key := policyKey{appName: appName, table: tableName, operation: operation}
			a.cache.programs[key] = ast
			a.cache.policies[key] = &policyEntry

			// Insert into database
			insertQuery := `
				INSERT INTO _policies (id, app_name, table_name, operation, expression, check_expr, columns, defaults, soft_delete, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`

			policyID := db.NewXID()
			columnsJSON, _ := json.Marshal(policyEntry.Columns)
			defaultsJSON, _ := json.Marshal(policyEntry.Defaults)

			err = a.db.Exec(ctx, appName, insertQuery,
				policyID, appName, tableName, operation,
				policyEntry.Expression, policyEntry.Check,
				string(columnsJSON), string(defaultsJSON), policyEntry.Soft)
			if err != nil {
				slog.Error("failed to insert policy", "app", appName, "table", tableName, "operation", operation, "error", err)
				return fmt.Errorf("auth.LoadPolicies: %w", err)
			}
		}
	}

	slog.Info("policies loaded", "app", appName, "policy_count", len(a.cache.programs))
	return nil
}

// EvaluatePolicy evaluates a cached RLS policy
func (a *authImpl) EvaluatePolicy(ctx context.Context, appName, table, operation string, rc *RequestContext) (PolicyResult, error) {
	// For secret key requests, bypass RLS entirely
	if rc.KeyType == string(KeyTypeSecret) {
		return PolicyResult{
			SQLClause: "TRUE",
			Params:    []any{},
			Defaults:  make(map[string]string),
			SoftCol:   "",
		}, nil
	}

	key := policyKey{appName: appName, table: table, operation: operation}

	// Check cache
	ast, ok := a.cache.programs[key]
	if !ok {
		slog.Info("policy not found", "app", appName, "table", table, "operation", operation)
		return PolicyResult{}, fmt.Errorf("auth.EvaluatePolicy: no policy found")
	}

	policy := a.cache.policies[key]

	// Build auth context for CEL evaluation
	authContext := celql.AuthContext{
		UID:           rc.UID,
		Meta:          rc.Meta,
		MetaApp:       rc.MetaApp,
		Authenticated: rc.Authenticated,
		KeyType:       rc.KeyType,
	}

	// Transpile CEL to SQL
	result, err := a.celql.Transpile(ast, authContext)
	if err != nil {
		slog.Error("failed to transpile policy", "app", appName, "table", table, "operation", operation, "error", err)
		return PolicyResult{}, fmt.Errorf("auth.EvaluatePolicy: %w", err)
	}

	// Build policy result
	policyResult := PolicyResult{
		SQLClause: result.SQL,
		Params:    result.Params,
		Defaults:  policy.Defaults,
		SoftCol:   policy.Soft,
	}

	slog.Debug("policy evaluated", "app", appName, "table", table, "operation", operation, "sql_clause", result.SQL)
	return policyResult, nil
}

// clearAppCache removes all cached policies for an app
func (a *authImpl) clearAppCache(appName string) {
	for key := range a.cache.programs {
		if key.appName == appName {
			delete(a.cache.programs, key)
			delete(a.cache.policies, key)
		}
	}
}

// parsePolicyKey parses a string key into policyKey struct
func parsePolicyKey(keyStr string) policyKey {
	parts := strings.Split(keyStr, ":")
	if len(parts) != 3 {
		return policyKey{}
	}
	return policyKey{
		appName:   parts[0],
		table:     parts[1],
		operation: parts[2],
	}
}
