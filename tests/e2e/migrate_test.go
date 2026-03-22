//go:build e2e

package e2e

import (
	"testing"
)

func TestMigrate_TablesExist(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Verify that migration-created tables are accessible
	tables := []string{"posts", "orders", "products", "items"}

	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			_, _, err := c.From(table).Limit(1).List(ctx)
			if err != nil {
				t.Errorf("table %s not accessible: %v", table, err)
			}
		})
	}
}

func TestMigrate_ReservedTablesExist(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Reserved tables should exist from bootstrap
	// We can't query them directly via CRUD (they may be blocked),
	// but we can verify auth works which proves _users and _sessions exist
	user, err := c.Auth.Me(ctx)
	if err != nil {
		t.Fatalf("me() failed — _users/_sessions may not exist: %v", err)
	}
	if user.ID == "" {
		t.Error("expected user ID — proves _users table exists")
	}
}
