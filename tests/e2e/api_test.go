//go:build e2e

package e2e

import (
	"testing"

	backd "github.com/backd-dev/backd/sdk/backd-go"
)

func TestCRUD_InsertAndGet(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	row, err := c.From("posts").Insert(ctx, map[string]any{
		"title":   "E2E Test Post",
		"content": "Hello from E2E",
		"author":  "e2e",
	})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	id, ok := row["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected id in insert response")
	}

	t.Cleanup(func() {
		_ = c.From("posts").Delete(t.Context(), id)
	})

	got, err := c.From("posts").Get(ctx, id)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got["title"] != "E2E Test Post" {
		t.Errorf("expected title 'E2E Test Post', got %v", got["title"])
	}
}

func TestCRUD_List(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Insert two posts
	for i := range 2 {
		row, err := c.From("posts").Insert(ctx, map[string]any{
			"title": "List Test " + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		id := row["id"].(string)
		t.Cleanup(func() {
			_ = c.From("posts").Delete(t.Context(), id)
		})
	}

	data, count, err := c.From("posts").Limit(100).List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(data) < 2 {
		t.Errorf("expected at least 2 rows, got %d", len(data))
	}
	if count < 2 {
		t.Errorf("expected count >= 2, got %d", count)
	}
}

func TestCRUD_Update(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	row, err := c.From("posts").Insert(ctx, map[string]any{"title": "Before Update"})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	id := row["id"].(string)
	t.Cleanup(func() { _ = c.From("posts").Delete(t.Context(), id) })

	updated, err := c.From("posts").Update(ctx, id, map[string]any{"title": "After Update"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated["title"] != "After Update" {
		t.Errorf("expected 'After Update', got %v", updated["title"])
	}
}

func TestCRUD_Patch(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	row, err := c.From("posts").Insert(ctx, map[string]any{"title": "Patch Me", "content": "original"})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	id := row["id"].(string)
	t.Cleanup(func() { _ = c.From("posts").Delete(t.Context(), id) })

	patched, err := c.From("posts").Patch(ctx, id, map[string]any{"content": "patched"})
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}
	if patched["content"] != "patched" {
		t.Errorf("expected 'patched', got %v", patched["content"])
	}
}

func TestCRUD_Delete(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	row, err := c.From("posts").Insert(ctx, map[string]any{"title": "Delete Me"})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	id := row["id"].(string)

	err = c.From("posts").Delete(ctx, id)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = c.From("posts").Get(ctx, id)
	if err == nil {
		t.Error("expected error getting deleted record")
	}
}

func TestCRUD_WhereFilter_Eq(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	row, err := c.From("posts").Insert(ctx, map[string]any{"title": "FilterMe", "author": "filter_test"})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	id := row["id"].(string)
	t.Cleanup(func() { _ = c.From("posts").Delete(t.Context(), id) })

	data, _, err := c.From("posts").
		Where(backd.WhereFilter{"author": map[string]any{"$eq": "filter_test"}}).
		List(ctx)
	if err != nil {
		t.Fatalf("list with where failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected at least 1 matching row")
	}
}

func TestCRUD_Pagination(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Insert 5 posts
	for i := range 5 {
		row, err := c.From("posts").Insert(ctx, map[string]any{
			"title": "Page Test " + string(rune('A'+i)),
		})
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
		id := row["id"].(string)
		t.Cleanup(func() { _ = c.From("posts").Delete(t.Context(), id) })
	}

	// Request with limit=2
	data, count, err := c.From("posts").Limit(2).Offset(0).List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(data) > 2 {
		t.Errorf("expected at most 2 rows, got %d", len(data))
	}
	// count should reflect total rows, not just the page
	if count < 5 {
		t.Errorf("expected count >= 5 (total rows), got %d", count)
	}
}

func TestCRUD_SelectColumns(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	row, err := c.From("posts").Insert(ctx, map[string]any{"title": "Select Test", "content": "secret"})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	id := row["id"].(string)
	t.Cleanup(func() { _ = c.From("posts").Delete(t.Context(), id) })

	data, _, err := c.From("posts").
		Select("id", "title").
		Where(backd.WhereFilter{"id": map[string]any{"$eq": id}}).
		List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected at least 1 row")
	}
	if data[0]["title"] == nil {
		t.Error("expected title in selected columns")
	}
}
