package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// SetGlobalMeta sets global metadata for a user using surgical JSONB updates
func (a *authImpl) SetGlobalMeta(ctx context.Context, appName, userID, key string, value any) error {
	if userID == "" || key == "" {
		return fmt.Errorf("userID and key cannot be empty")
	}

	authDB := a.resolveAuthDB(appName)

	// Build JSON path for global metadata: {_,key}
	path := fmt.Sprintf("{_,%s}", strings.ReplaceAll(key, `"`, `\"`))

	var query string
	var args []any

	if value == nil {
		// Delete the key using #- operator
		query = `
			UPDATE _users 
			SET metadata = metadata #- $1, updated_at = NOW()
			WHERE id = $2`
		args = []any{path, userID}
	} else {
		// Set/update the key using jsonb_set
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}

		query = `
			UPDATE _users 
			SET metadata = jsonb_set(metadata, $1, $2::jsonb), updated_at = NOW()
			WHERE id = $3`
		args = []any{path, string(valueJSON), userID}
	}

	err := a.db.Exec(ctx, authDB, query, args...)
	if err != nil {
		slog.Error("failed to set global metadata", "app", appName, "user_id", userID, "key", key, "error", err)
		return fmt.Errorf("auth.SetGlobalMeta: %w", err)
	}

	slog.Debug("global metadata set", "app", appName, "user_id", userID, "key", key)
	return nil
}

// SetAppMeta sets app-specific metadata for a user using surgical JSONB updates
func (a *authImpl) SetAppMeta(ctx context.Context, appName, userID, key string, value any) error {
	if userID == "" || key == "" {
		return fmt.Errorf("userID and key cannot be empty")
	}

	authDB := a.resolveAuthDB(appName)

	// Build JSON path for app-specific metadata: {appName,key}
	path := fmt.Sprintf("{%s,%s}", appName, strings.ReplaceAll(key, `"`, `\"`))

	var query string
	var args []any

	if value == nil {
		// Delete the key using #- operator
		query = `
			UPDATE _users 
			SET metadata = metadata #- $1, updated_at = NOW()
			WHERE id = $2`
		args = []any{path, userID}
	} else {
		// Set/update the key using jsonb_set
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}

		query = `
			UPDATE _users 
			SET metadata = jsonb_set(metadata, $1, $2::jsonb), updated_at = NOW()
			WHERE id = $3`
		args = []any{path, string(valueJSON), userID}
	}

	err := a.db.Exec(ctx, authDB, query, args...)
	if err != nil {
		slog.Error("failed to set app metadata", "app", appName, "user_id", userID, "key", key, "error", err)
		return fmt.Errorf("auth.SetAppMeta: %w", err)
	}

	slog.Debug("app metadata set", "app", appName, "user_id", userID, "key", key)
	return nil
}

// ApplyDefaults resolves and applies default values for row operations
func (a *authImpl) ApplyDefaults(defaults map[string]string, rc *RequestContext) map[string]any {
	if defaults == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)

	for key, defaultValue := range defaults {
		switch {
		case strings.HasPrefix(defaultValue, "auth.uid"):
			if rc.Authenticated && rc.UID != "" {
				result[key] = rc.UID
			}
		case strings.HasPrefix(defaultValue, "auth.meta."):
			metaKey := strings.TrimPrefix(defaultValue, "auth.meta.")
			if rc.Meta != nil {
				if value, exists := rc.Meta[metaKey]; exists {
					result[key] = value
				}
			}
		case defaultValue == "now()":
			// This would be handled at the SQL level with NOW()
			result[key] = "NOW()"
		case defaultValue == "today()":
			// This would be handled at the SQL level with CURRENT_DATE
			result[key] = "CURRENT_DATE"
		default:
			// Pass through literal values
			result[key] = defaultValue
		}
	}

	slog.Debug("defaults applied", "defaults_count", len(result))
	return result
}
