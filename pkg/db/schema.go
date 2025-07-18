package db

import (
	"context"
	"fmt"
	"strings"
)

// TableInfo represents information about a database table
type TableInfo struct {
	Schema        string
	Name          string
	Type          string // table, view, materialized view
	Description   string
	EstimatedRows int64 // Estimated row count from pg_class.reltuples
	Columns       []ColumnInfo
}

// ColumnInfo represents information about a table column
type ColumnInfo struct {
	Name         string
	DataType     string
	IsNullable   bool
	Default      string
	IsPrimaryKey bool
	Description  string
}

// ForeignKeyInfo represents foreign key relationships
type ForeignKeyInfo struct {
	TableSchema        string
	TableName          string
	ColumnName         string
	ForeignTableSchema string
	ForeignTableName   string
	ForeignColumnName  string
	ConstraintName     string
}

// IndexInfo represents database index information
type IndexInfo struct {
	Name       string
	TableName  string
	Columns    []string
	IsUnique   bool
	IsPrimary  bool
	Definition string
}

// ListTables returns all tables and views in the database
func (c *ConnectionImpl) ListTables(ctx context.Context) ([]TableInfo, error) {
	query := `
		SELECT 
			n.nspname as schema_name,
			c.relname as table_name,
			CASE c.relkind 
				WHEN 'r' THEN 'table'
				WHEN 'v' THEN 'view'
				WHEN 'm' THEN 'materialized view'
				ELSE 'other'
			END as table_type,
			COALESCE(c.reltuples, 0)::bigint as estimated_rows
		FROM pg_class c
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE c.relkind IN ('r', 'v', 'm')  -- tables, views, materialized views
		  AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name, table_name
	`

	rows, err := c.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		err := rows.Scan(&table.Schema, &table.Name, &table.Type, &table.EstimatedRows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table info: %w", err)
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during table listing iteration: %w", err)
	}
	return tables, nil
}

// DescribeTable returns detailed information about a specific table
func (c *ConnectionImpl) DescribeTable(ctx context.Context, schema, tableName string) (*TableInfo, error) {
	// Default schema
	if schema == "" {
		schema = "public"
	}

	// Get basic table info
	tableQuery := `
		SELECT 
			t.table_schema,
			t.table_name,
			t.table_type,
			COALESCE(obj_description(c.oid), '') as description
		FROM information_schema.tables t
		LEFT JOIN pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		WHERE t.table_schema = $1 AND t.table_name = $2
	`

	var table TableInfo
	var tableType string
	err := c.QueryRow(ctx, tableQuery, schema, tableName).Scan(
		&table.Schema, &table.Name, &tableType, &table.Description)
	if err != nil {
		return nil, fmt.Errorf("table %s.%s not found: %w", schema, tableName, err)
	}

	// Convert table type
	switch strings.ToLower(tableType) {
	case "base table":
		table.Type = "table"
	case "view":
		table.Type = "view"
	default:
		table.Type = strings.ToLower(tableType)
	}

	// Get column information
	columns, err := c.getTableColumns(ctx, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table columns: %w", err)
	}
	table.Columns = columns

	return &table, nil
}

// getTableColumns retrieves column information for a table
func (c *ConnectionImpl) getTableColumns(ctx context.Context, schema, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			COALESCE(c.column_default, '') as column_default,
			COALESCE(col_description(pgc.oid, c.ordinal_position), '') as description
		FROM information_schema.columns c
		LEFT JOIN pg_class pgc ON pgc.relname = c.table_name
		LEFT JOIN pg_namespace pgn ON pgn.oid = pgc.relnamespace AND pgn.nspname = c.table_schema
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := c.Query(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query table columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable, &col.Default, &col.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}
		columns = append(columns, col)
	}

	// Get primary key information
	pkColumns, err := c.getPrimaryKeyColumns(ctx, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys: %w", err)
	}

	// Mark primary key columns
	for i := range columns {
		for _, pkCol := range pkColumns {
			if columns[i].Name == pkCol {
				columns[i].IsPrimaryKey = true
				break
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during column iteration: %w", err)
	}
	return columns, nil
}

// getPrimaryKeyColumns returns the primary key column names for a table
func (c *ConnectionImpl) getPrimaryKeyColumns(ctx context.Context, schema, tableName string) ([]string, error) {
	query := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE i.indisprimary 
		AND n.nspname = $1 
		AND c.relname = $2
		ORDER BY a.attnum
	`

	rows, err := c.Query(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query primary keys: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, fmt.Errorf("failed to scan primary key column: %w", err)
		}
		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during primary key iteration: %w", err)
	}
	return columns, nil
}

// GetForeignKeys returns foreign key relationships for a table
func (c *ConnectionImpl) GetForeignKeys(ctx context.Context, schema, tableName string) ([]ForeignKeyInfo, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT 
			tc.table_schema,
			tc.table_name,
			kcu.column_name,
			ccu.table_schema AS foreign_table_schema,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu 
			ON ccu.constraint_name = tc.constraint_name 
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = $1 AND tc.table_name = $2
	`

	rows, err := c.Query(ctx, query, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer rows.Close()

	var foreignKeys []ForeignKeyInfo
	for rows.Next() {
		var fk ForeignKeyInfo
		err := rows.Scan(
			&fk.TableSchema, &fk.TableName, &fk.ColumnName,
			&fk.ForeignTableSchema, &fk.ForeignTableName, &fk.ForeignColumnName,
			&fk.ConstraintName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan foreign key info: %w", err)
		}
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, rows.Err()
}

// SearchColumns searches for columns matching a pattern across all tables
func (c *ConnectionImpl) SearchColumns(ctx context.Context, pattern string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			c.table_schema || '.' || c.table_name as table_name,
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			COALESCE(c.column_default, '') as column_default
		FROM information_schema.columns c
		WHERE c.table_schema NOT IN ('information_schema', 'pg_catalog')
		AND c.column_name ILIKE $1
		ORDER BY c.table_schema, c.table_name, c.ordinal_position
	`

	rows, err := c.Query(ctx, query, "%"+pattern+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var tableName string
		err := rows.Scan(&tableName, &col.Name, &col.DataType, &col.IsNullable, &col.Default)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}
		col.Description = fmt.Sprintf("Found in table: %s", tableName)
		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during column search iteration: %w", err)
	}
	return columns, nil
}
