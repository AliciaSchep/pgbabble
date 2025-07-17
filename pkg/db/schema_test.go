package db

import (
	"context"
	"testing"

	"github.com/AliciaSchep/pgbabble/internal/testutil"
)

func TestListTables_WithRealDatabase(t *testing.T) {
	cfg := testutil.GetRealDatabaseConfig()
	if cfg == nil {
		t.Skip("Skipping real database tests - no database config available.")
		return
	}

	conn, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	if err := testutil.SetupTestSchema(ctx, func(ctx context.Context, sql string) error {
		return conn.Exec(ctx, sql)
	}); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}
	defer func() {
		if err := testutil.CleanupTestSchema(ctx, func(ctx context.Context, sql string) error {
			return conn.Exec(ctx, sql)
		}); err != nil {
			t.Logf("Warning: Failed to cleanup test schema: %v", err)
		}
	}()

	tables, err := conn.ListTables(ctx)
	if err != nil {
		t.Fatalf("ListTables failed: %v", err)
	}

	testTableNames := map[string]bool{
		"test_users":       false,
		"test_products":    false,
		"test_orders":      false,
		"test_order_items": false,
	}

	for _, table := range tables {
		if _, exists := testTableNames[table.Name]; exists {
			testTableNames[table.Name] = true
			if table.Schema != "public" {
				t.Errorf("Expected table %s to be in 'public' schema, got %s", table.Name, table.Schema)
			}
			if table.Type != "table" {
				t.Errorf("Expected table %s to have type 'table', got %s", table.Name, table.Type)
			}
		}
	}

	for tableName, found := range testTableNames {
		if !found {
			t.Errorf("Expected to find test table %s", tableName)
		}
	}
}

func TestDescribeTable_WithRealDatabase(t *testing.T) {
	cfg := testutil.GetRealDatabaseConfig()
	if cfg == nil {
		t.Skip("Skipping real database tests - no database config available.")
		return
	}

	conn, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	if err := testutil.SetupTestSchema(ctx, func(ctx context.Context, sql string) error {
		return conn.Exec(ctx, sql)
	}); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}
	defer func() {
		if err := testutil.CleanupTestSchema(ctx, func(ctx context.Context, sql string) error {
			return conn.Exec(ctx, sql)
		}); err != nil {
			t.Logf("Warning: Failed to cleanup test schema: %v", err)
		}
	}()

	t.Run("test_users_table", func(t *testing.T) {
		table, err := conn.DescribeTable(ctx, "public", "test_users")
		if err != nil {
			t.Fatalf("DescribeTable failed: %v", err)
		}

		if table.Name != "test_users" {
			t.Errorf("Expected table name 'test_users', got %s", table.Name)
		}
		if table.Schema != "public" {
			t.Errorf("Expected schema 'public', got %s", table.Schema)
		}

		expectedColumns := map[string]struct {
			dataType     string
			isNullable   bool
			isPrimaryKey bool
		}{
			"id":         {"integer", false, true},
			"username":   {"character varying", false, false},
			"email":      {"character varying", false, false},
			"created_at": {"timestamp without time zone", true, false},
		}

		if len(table.Columns) != len(expectedColumns) {
			t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(table.Columns))
		}

		for _, col := range table.Columns {
			expected, exists := expectedColumns[col.Name]
			if !exists {
				t.Errorf("Unexpected column: %s", col.Name)
				continue
			}

			if col.DataType != expected.dataType {
				t.Errorf("Column %s: expected data type %s, got %s", col.Name, expected.dataType, col.DataType)
			}
			if col.IsNullable != expected.isNullable {
				t.Errorf("Column %s: expected nullable %v, got %v", col.Name, expected.isNullable, col.IsNullable)
			}
			if col.IsPrimaryKey != expected.isPrimaryKey {
				t.Errorf("Column %s: expected primary key %v, got %v", col.Name, expected.isPrimaryKey, col.IsPrimaryKey)
			}
		}
	})

	t.Run("nonexistent_table", func(t *testing.T) {
		_, err := conn.DescribeTable(ctx, "public", "nonexistent_table")
		if err == nil {
			t.Error("Expected error when describing nonexistent table")
		}
	})
}

func TestGetForeignKeys_WithRealDatabase(t *testing.T) {
	cfg := testutil.GetRealDatabaseConfig()
	if cfg == nil {
		t.Skip("Skipping real database tests - no database config available.")
		return
	}

	conn, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	if err := testutil.SetupTestSchema(ctx, func(ctx context.Context, sql string) error {
		return conn.Exec(ctx, sql)
	}); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}
	defer func() {
		if err := testutil.CleanupTestSchema(ctx, func(ctx context.Context, sql string) error {
			return conn.Exec(ctx, sql)
		}); err != nil {
			t.Logf("Warning: Failed to cleanup test schema: %v", err)
		}
	}()

	t.Run("test_orders_foreign_keys", func(t *testing.T) {
		fks, err := conn.GetForeignKeys(ctx, "public", "test_orders")
		if err != nil {
			t.Fatalf("GetForeignKeys failed: %v", err)
		}

		if len(fks) != 1 {
			t.Errorf("Expected 1 foreign key for test_orders, got %d", len(fks))
			return
		}

		fk := fks[0]
		if fk.TableName != "test_orders" {
			t.Errorf("Expected table name 'test_orders', got %s", fk.TableName)
		}
		if fk.ColumnName != "user_id" {
			t.Errorf("Expected column name 'user_id', got %s", fk.ColumnName)
		}
		if fk.ForeignTableName != "test_users" {
			t.Errorf("Expected foreign table 'test_users', got %s", fk.ForeignTableName)
		}
		if fk.ForeignColumnName != "id" {
			t.Errorf("Expected foreign column 'id', got %s", fk.ForeignColumnName)
		}
	})

	t.Run("test_order_items_foreign_keys", func(t *testing.T) {
		fks, err := conn.GetForeignKeys(ctx, "public", "test_order_items")
		if err != nil {
			t.Fatalf("GetForeignKeys failed: %v", err)
		}

		if len(fks) != 2 {
			t.Errorf("Expected 2 foreign keys for test_order_items, got %d", len(fks))
			return
		}

		foreignKeyMap := make(map[string]ForeignKeyInfo)
		for _, fk := range fks {
			foreignKeyMap[fk.ColumnName] = fk
		}

		if fk, exists := foreignKeyMap["order_id"]; exists {
			if fk.ForeignTableName != "test_orders" {
				t.Errorf("Expected order_id to reference test_orders, got %s", fk.ForeignTableName)
			}
		} else {
			t.Error("Expected foreign key for order_id")
		}

		if fk, exists := foreignKeyMap["product_id"]; exists {
			if fk.ForeignTableName != "test_products" {
				t.Errorf("Expected product_id to reference test_products, got %s", fk.ForeignTableName)
			}
		} else {
			t.Error("Expected foreign key for product_id")
		}
	})

	t.Run("table_with_no_foreign_keys", func(t *testing.T) {
		fks, err := conn.GetForeignKeys(ctx, "public", "test_users")
		if err != nil {
			t.Fatalf("GetForeignKeys failed: %v", err)
		}

		if len(fks) != 0 {
			t.Errorf("Expected 0 foreign keys for test_users, got %d", len(fks))
		}
	})
}

func TestSearchColumns_WithRealDatabase(t *testing.T) {
	cfg := testutil.GetRealDatabaseConfig()
	if cfg == nil {
		t.Skip("Skipping real database tests - no database config available.")
		return
	}

	conn, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	if err := testutil.SetupTestSchema(ctx, func(ctx context.Context, sql string) error {
		return conn.Exec(ctx, sql)
	}); err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}
	defer func() {
		if err := testutil.CleanupTestSchema(ctx, func(ctx context.Context, sql string) error {
			return conn.Exec(ctx, sql)
		}); err != nil {
			t.Logf("Warning: Failed to cleanup test schema: %v", err)
		}
	}()

	t.Run("search_for_id_columns", func(t *testing.T) {
		columns, err := conn.SearchColumns(ctx, "id")
		if err != nil {
			t.Fatalf("SearchColumns failed: %v", err)
		}

		expectedColumns := map[string]bool{
			"id":         false,
			"user_id":    false,
			"order_id":   false,
			"product_id": false,
		}

		for _, col := range columns {
			if _, exists := expectedColumns[col.Name]; exists {
				expectedColumns[col.Name] = true
			}
		}

		for colName, found := range expectedColumns {
			if !found {
				t.Errorf("Expected to find column with name containing 'id': %s", colName)
			}
		}

		if len(columns) < 7 {
			t.Errorf("Expected at least 7 columns with 'id' in name, got %d", len(columns))
		}
	})

	t.Run("search_for_name_columns", func(t *testing.T) {
		columns, err := conn.SearchColumns(ctx, "name")
		if err != nil {
			t.Fatalf("SearchColumns failed: %v", err)
		}

		expectedColumns := map[string]bool{
			"username": false,
			"name":     false,
		}

		for _, col := range columns {
			if _, exists := expectedColumns[col.Name]; exists {
				expectedColumns[col.Name] = true
			}
		}

		for colName, found := range expectedColumns {
			if !found {
				t.Errorf("Expected to find column with name containing 'name': %s", colName)
			}
		}

		if len(columns) < 2 {
			t.Errorf("Expected at least 2 columns with 'name' in name, got %d", len(columns))
		}
	})

	t.Run("search_for_nonexistent_pattern", func(t *testing.T) {
		columns, err := conn.SearchColumns(ctx, "nonexistent_pattern_xyz")
		if err != nil {
			t.Fatalf("SearchColumns failed: %v", err)
		}

		if len(columns) != 0 {
			t.Errorf("Expected 0 columns for nonexistent pattern, got %d", len(columns))
		}
	})
}

func TestGetDatabaseInfo_WithRealDatabase(t *testing.T) {
	cfg := testutil.GetRealDatabaseConfig()
	if cfg == nil {
		t.Skip("Skipping real database tests - no database config available.")
		return
	}

	conn, err := Connect(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	info, err := conn.GetDatabaseInfo(ctx)
	if err != nil {
		t.Fatalf("GetDatabaseInfo failed: %v", err)
	}

	if info.Database != cfg.Database {
		t.Errorf("Expected database %s, got %s", cfg.Database, info.Database)
	}
	if info.User != cfg.User {
		t.Errorf("Expected user %s, got %s", cfg.User, info.User)
	}
	if info.Host != cfg.Host {
		t.Errorf("Expected host %s, got %s", cfg.Host, info.Host)
	}
	if info.Port != cfg.Port {
		t.Errorf("Expected port %d, got %d", cfg.Port, info.Port)
	}
	if info.Version == "" {
		t.Error("Expected non-empty version string")
	}
	t.Logf("Database info: %+v", info)
}
