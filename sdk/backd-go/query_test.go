package backd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryBuilder_BuildParams(t *testing.T) {
	qb := &QueryBuilder{
		table:   "orders",
		columns: []string{"id", "status"},
		where:   WhereFilter{"status": map[string]any{"$eq": "pending"}},
		orders:  []string{"created_at:desc"},
		lim:     20,
		off:     10,
	}

	params := qb.buildParams()

	if params["select"] != "id,status" {
		t.Errorf("select = %q, want %q", params["select"], "id,status")
	}
	if params["limit"] != "20" {
		t.Errorf("limit = %q, want %q", params["limit"], "20")
	}
	if params["offset"] != "10" {
		t.Errorf("offset = %q, want %q", params["offset"], "10")
	}
	if params["order"] != "created_at:desc" {
		t.Errorf("order = %q, want %q", params["order"], "created_at:desc")
	}

	// where should be JSON-encoded
	var w map[string]any
	if err := json.Unmarshal([]byte(params["where"]), &w); err != nil {
		t.Fatalf("where is not valid JSON: %v", err)
	}
}

func TestQueryBuilder_DefaultParams(t *testing.T) {
	qb := &QueryBuilder{table: "orders", lim: 50}
	params := qb.buildParams()

	if params["limit"] != "50" {
		t.Errorf("default limit = %q, want 50", params["limit"])
	}
	if params["offset"] != "0" {
		t.Errorf("default offset = %q, want 0", params["offset"])
	}
	if _, ok := params["select"]; ok {
		t.Error("select should not be set when no columns specified")
	}
	if _, ok := params["where"]; ok {
		t.Error("where should not be set when no filter specified")
	}
}

func TestQueryBuilder_Chain(t *testing.T) {
	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, "http://api", "orders")

	result := qb.Select("id", "total").
		Where(WhereFilter{"status": map[string]any{"$eq": "pending"}}).
		Order("created_at", "desc").
		Limit(10).
		Offset(5)

	if result != qb {
		t.Error("chain methods should return the same QueryBuilder")
	}
	if len(qb.columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(qb.columns))
	}
	if qb.lim != 10 {
		t.Errorf("expected limit 10, got %d", qb.lim)
	}
	if qb.off != 5 {
		t.Errorf("expected offset 5, got %d", qb.off)
	}
}

func TestQueryBuilder_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/orders/abc123" {
			t.Errorf("expected path /orders/abc123, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "abc123", "total": 99.99}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")

	ctx := t.Context()
	row, err := qb.Get(ctx, "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row["id"] != "abc123" {
		t.Errorf("expected id abc123, got %v", row["id"])
	}
}

func TestQueryBuilder_Insert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/orders" {
			t.Errorf("expected path /orders, got %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		body["id"] = "new1"
		json.NewEncoder(w).Encode(map[string]any{"data": body})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")

	ctx := t.Context()
	row, err := qb.Insert(ctx, map[string]any{"total": 49.99})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row["id"] != "new1" {
		t.Errorf("expected id new1, got %v", row["id"])
	}
}

func TestQueryBuilder_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/orders/abc" {
			t.Errorf("expected path /orders/abc, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "abc"}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")
	ctx := t.Context()
	_, err := qb.Update(ctx, "abc", map[string]any{"status": "shipped"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryBuilder_Patch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "abc"}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")
	ctx := t.Context()
	_, err := qb.Patch(ctx, "abc", map[string]any{"status": "shipped"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryBuilder_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")
	ctx := t.Context()
	err := qb.Delete(ctx, "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryBuilder_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// Check where param is URL-encoded
		where := r.URL.Query().Get("where")
		if where == "" {
			t.Error("where param should be set")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":  []map[string]any{{"id": "1"}, {"id": "2"}},
			"count": 42,
		})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")
	ctx := t.Context()

	data, count, err := qb.Where(WhereFilter{"status": map[string]any{"$eq": "pending"}}).List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 rows, got %d", len(data))
	}
	if count != 42 {
		t.Errorf("expected count 42, got %d", count)
	}
}

func TestQueryBuilder_Single_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}, "count": 0})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")
	ctx := t.Context()

	_, err := qb.Single(ctx)
	if err == nil {
		t.Fatal("expected error for empty result")
	}
	qErr, ok := err.(*QueryError)
	if !ok {
		t.Fatalf("expected QueryError, got %T", err)
	}
	if qErr.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %s", qErr.Code)
	}
	if qErr.Table != "orders" {
		t.Errorf("expected table orders, got %s", qErr.Table)
	}
}

func TestQueryBuilder_InsertMany(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": callCount}})
	}))
	defer srv.Close()

	h := newHTTPClient("pk", "")
	qb := newQueryBuilder(h, srv.URL, "orders")
	ctx := t.Context()

	results, err := qb.InsertMany(ctx, []map[string]any{{"total": 10}, {"total": 20}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", callCount)
	}
}
