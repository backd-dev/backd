package db

import (
	"context"
	"fmt"
)

// Exec executes a query that doesn't return rows
func (db *dbImpl) Exec(ctx context.Context, appName, query string, args ...any) error {
	pool, err := db.Pool(appName)
	if err != nil {
		return fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	_, err = pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

// Query executes a query that returns multiple rows
func (db *dbImpl) Query(ctx context.Context, appName, query string, args ...any) ([]map[string]any, error) {
	pool, err := db.Pool(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rows: %w", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Scan all rows
	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create map for this row
		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// QueryOne executes a query that returns a single row
func (db *dbImpl) QueryOne(ctx context.Context, appName, query string, args ...any) (map[string]any, error) {
	results, err := db.Query(ctx, appName, query, args...)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no rows returned")
	}

	if len(results) > 1 {
		return nil, fmt.Errorf("expected 1 row, got %d", len(results))
	}

	return results[0], nil
}
