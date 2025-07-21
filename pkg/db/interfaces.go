package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Connection interface defines all database operations available to other packages
// This allows for dependency injection and easier testing with mocks
type Connection interface {
	// Schema operations
	ListTables(ctx context.Context) ([]TableInfo, error)
	DescribeTable(ctx context.Context, schema, tableName string) (*TableInfo, error)
	GetForeignKeys(ctx context.Context, schema, tableName string) ([]ForeignKeyInfo, error)
	SearchColumns(ctx context.Context, pattern string) ([]ColumnInfo, error)

	// Query operations
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	EnsureConnection(ctx context.Context)
	ForceReconnect(ctx context.Context)
}

// Ensure that the concrete Connection struct implements the interface
var _ Connection = (*ConnectionImpl)(nil)
