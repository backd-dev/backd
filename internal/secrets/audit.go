package secrets

import (
	"context"

	"github.com/backd-dev/backd/internal/db"
)

// LogAccess records secret access in the _secret_audit table
// This is separated so it can be stubbed in unit tests without touching DB
func LogAccess(ctx context.Context, database db.DB, appName, secretName, action string) error {
	query := `
		INSERT INTO _secret_audit (id, secret_name, action, app_name) 
		VALUES ($1, $2, $3, $4)
	`

	// Generate a new ID for the audit record
	id := db.NewXID()

	return database.Exec(ctx, appName, query, id, secretName, action, appName)
}
