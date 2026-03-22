package backd

import (
	"context"
	"encoding/json"
	"fmt"
)

// QueryBuilder builds and executes CRUD queries against a collection.
type QueryBuilder struct {
	http    *httpClient
	apiURL  string
	table   string
	columns []string
	where   WhereFilter
	orders  []string
	lim     int
	off     int
}

func newQueryBuilder(h *httpClient, apiURL, table string) *QueryBuilder {
	return &QueryBuilder{
		http:   h,
		apiURL: apiURL,
		table:  table,
		lim:    50,
	}
}

// Select sets the columns to return.
func (q *QueryBuilder) Select(columns ...string) *QueryBuilder {
	q.columns = columns
	return q
}

// Where sets the filter for the query.
func (q *QueryBuilder) Where(filter WhereFilter) *QueryBuilder {
	q.where = filter
	return q
}

// Order adds a sort clause (e.g. "created_at", "desc").
func (q *QueryBuilder) Order(column string, direction ...string) *QueryBuilder {
	dir := "asc"
	if len(direction) > 0 {
		dir = direction[0]
	}
	q.orders = append(q.orders, column+":"+dir)
	return q
}

// Limit sets the maximum number of rows to return.
func (q *QueryBuilder) Limit(n int) *QueryBuilder {
	q.lim = n
	return q
}

// Offset sets the number of rows to skip.
func (q *QueryBuilder) Offset(n int) *QueryBuilder {
	q.off = n
	return q
}

func (q *QueryBuilder) buildParams() map[string]string {
	params := map[string]string{
		"limit":  fmt.Sprintf("%d", q.lim),
		"offset": fmt.Sprintf("%d", q.off),
	}
	if len(q.columns) > 0 {
		s := ""
		for i, c := range q.columns {
			if i > 0 {
				s += ","
			}
			s += c
		}
		params["select"] = s
	}
	if q.where != nil {
		data, _ := json.Marshal(q.where)
		params["where"] = string(data)
	}
	if len(q.orders) > 0 {
		s := ""
		for i, o := range q.orders {
			if i > 0 {
				s += ","
			}
			s += o
		}
		params["order"] = s
	}
	return params
}

// List executes the query and returns a paginated result.
func (q *QueryBuilder) List(ctx context.Context) ([]map[string]any, int, error) {
	return requestWithMeta[[]map[string]any](ctx, q.http, requestOptions{
		method:  "GET",
		baseURL: q.apiURL,
		path:    "/" + q.table,
		params:  q.buildParams(),
	})
}

// Get retrieves a single row by ID.
func (q *QueryBuilder) Get(ctx context.Context, id string) (map[string]any, error) {
	return request[map[string]any](ctx, q.http, requestOptions{
		method:  "GET",
		baseURL: q.apiURL,
		path:    "/" + q.table + "/" + id,
	})
}

// Single retrieves the first row matching the filter.
func (q *QueryBuilder) Single(ctx context.Context) (map[string]any, error) {
	params := q.buildParams()
	params["limit"] = "1"
	data, _, err := requestWithMeta[[]map[string]any](ctx, q.http, requestOptions{
		method:  "GET",
		baseURL: q.apiURL,
		path:    "/" + q.table,
		params:  params,
	})
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, &QueryError{
			BackdError: BackdError{Code: "NOT_FOUND", Detail: "no matching record found", Status: 404},
			Table:      q.table,
		}
	}
	return data[0], nil
}

// Insert creates a new row.
func (q *QueryBuilder) Insert(ctx context.Context, row map[string]any) (map[string]any, error) {
	return request[map[string]any](ctx, q.http, requestOptions{
		method:  "POST",
		baseURL: q.apiURL,
		path:    "/" + q.table,
		body:    row,
	})
}

// InsertMany creates multiple rows sequentially.
func (q *QueryBuilder) InsertMany(ctx context.Context, rows []map[string]any) ([]map[string]any, error) {
	results := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result, err := q.Insert(ctx, row)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

// Update replaces a row entirely by ID.
func (q *QueryBuilder) Update(ctx context.Context, id string, row map[string]any) (map[string]any, error) {
	return request[map[string]any](ctx, q.http, requestOptions{
		method:  "PUT",
		baseURL: q.apiURL,
		path:    "/" + q.table + "/" + id,
		body:    row,
	})
}

// Patch partially updates a row by ID.
func (q *QueryBuilder) Patch(ctx context.Context, id string, partial map[string]any) (map[string]any, error) {
	return request[map[string]any](ctx, q.http, requestOptions{
		method:  "PATCH",
		baseURL: q.apiURL,
		path:    "/" + q.table + "/" + id,
		body:    partial,
	})
}

// Delete removes a row by ID.
func (q *QueryBuilder) Delete(ctx context.Context, id string) error {
	_, err := request[any](ctx, q.http, requestOptions{
		method:  "DELETE",
		baseURL: q.apiURL,
		path:    "/" + q.table + "/" + id,
	})
	return err
}
