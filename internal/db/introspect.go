package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Tables returns information about all non-system tables in the app database
func (db *dbImpl) Tables(ctx context.Context, appName string) ([]TableInfo, error) {
	pool, err := db.Pool(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	// Query for all tables that don't start with underscore
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name NOT LIKE '\_%'
		ORDER BY table_name
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}

		// Get column information for this table
		columns, err := db.Columns(ctx, appName, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}

		tables = append(tables, TableInfo{
			Name:    tableName,
			Columns: columns,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	slog.Debug("introspected tables", "app", appName, "count", len(tables))
	return tables, nil
}

// Columns returns information about all columns in a table
func (db *dbImpl) Columns(ctx context.Context, appName, table string) ([]ColumnInfo, error) {
	pool, err := db.Pool(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool for app %s: %w", appName, err)
	}

	// Query for column information
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable
		FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := pool.Query(ctx, query, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for table %s: %w", table, err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var columnName, dataType, isNullable string
		if err := rows.Scan(&columnName, &dataType, &isNullable); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		column := ColumnInfo{
			Name:     columnName,
			Type:     dataType,
			Nullable: isNullable == "YES",
			IsFile:   strings.HasSuffix(columnName, "__file"),
		}
		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	slog.Debug("introspected columns", "app", appName, "table", table, "count", len(columns))
	return columns, nil
}
