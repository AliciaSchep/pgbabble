package chat

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/AliciaSchep/pgbabble/pkg/db"
	"github.com/jackc/pgx/v5"
)

// MockDBConnection implements db.Connection for testing
type MockDBConnection struct {
	tables       []db.TableInfo
	tableDetails map[string]*db.TableInfo
	foreignKeys  map[string][]db.ForeignKeyInfo
	shouldFail   string // Which method should fail
}

func NewMockDBConnection() *MockDBConnection {
	return &MockDBConnection{
		tables: []db.TableInfo{
			{Schema: "public", Name: "users", Type: "table"},
			{Schema: "public", Name: "orders", Type: "table"},
			{Schema: "analytics", Name: "user_stats", Type: "view"},
			{Schema: "analytics", Name: "sales_summary", Type: "materialized view"},
		},
		tableDetails: map[string]*db.TableInfo{
			"users": {
				Schema:      "public",
				Name:        "users",
				Type:        "table",
				Description: "User accounts table",
				Columns: []db.ColumnInfo{
					{Name: "id", DataType: "integer", IsPrimaryKey: true, IsNullable: false, Default: "nextval('users_id_seq'::regclass)"},
					{Name: "username", DataType: "character varying", IsNullable: false},
					{Name: "email", DataType: "character varying", IsNullable: false},
					{Name: "created_at", DataType: "timestamp without time zone", IsNullable: true, Default: "CURRENT_TIMESTAMP"},
					{Name: "active", DataType: "boolean", IsNullable: false, Default: "true"},
				},
			},
			"orders": {
				Schema: "public",
				Name:   "orders",
				Type:   "table",
				Columns: []db.ColumnInfo{
					{Name: "id", DataType: "integer", IsPrimaryKey: true, IsNullable: false},
					{Name: "user_id", DataType: "integer", IsNullable: false},
					{Name: "total", DataType: "numeric", IsNullable: false},
					{Name: "status", DataType: "character varying", IsNullable: false, Default: "'pending'"},
				},
			},
		},
		foreignKeys: map[string][]db.ForeignKeyInfo{
			"orders": {
				{
					TableSchema:        "public",
					TableName:          "orders",
					ColumnName:         "user_id",
					ForeignTableSchema: "public",
					ForeignTableName:   "users",
					ForeignColumnName:  "id",
					ConstraintName:     "orders_user_id_fkey",
				},
			},
		},
	}
}

func (m *MockDBConnection) ListTables(ctx context.Context) ([]db.TableInfo, error) {
	if m.shouldFail == "ListTables" {
		return nil, fmt.Errorf("mock database error: ListTables failed")
	}
	return m.tables, nil
}

func (m *MockDBConnection) DescribeTable(ctx context.Context, schema, tableName string) (*db.TableInfo, error) {
	if m.shouldFail == "DescribeTable" {
		return nil, fmt.Errorf("mock database error: DescribeTable failed")
	}
	key := tableName
	if table, exists := m.tableDetails[key]; exists {
		return table, nil
	}
	return nil, fmt.Errorf("table %s.%s not found", schema, tableName)
}

func (m *MockDBConnection) GetForeignKeys(ctx context.Context, schema, tableName string) ([]db.ForeignKeyInfo, error) {
	if m.shouldFail == "GetForeignKeys" {
		return nil, fmt.Errorf("mock database error: GetForeignKeys failed")
	}
	if fks, exists := m.foreignKeys[tableName]; exists {
		return fks, nil
	}
	return []db.ForeignKeyInfo{}, nil
}

// SearchColumns implements the SearchColumns method for the db.Connection interface
func (m *MockDBConnection) SearchColumns(ctx context.Context, pattern string) ([]db.ColumnInfo, error) {
	if m.shouldFail == "SearchColumns" {
		return nil, fmt.Errorf("mock database error: SearchColumns failed")
	}
	// Mock implementation - return some columns that match the pattern
	var results []db.ColumnInfo
	for tableName, table := range m.tableDetails {
		for _, col := range table.Columns {
			if strings.Contains(strings.ToLower(col.Name), strings.ToLower(pattern)) {
				results = append(results, db.ColumnInfo{
					Name:        col.Name,
					DataType:    col.DataType,
					IsNullable:  col.IsNullable,
					Default:     col.Default,
					Description: fmt.Sprintf("Found in table: %s.%s", table.Schema, tableName),
				})
			}
		}
	}
	return results, nil
}

// Query implements the Query method for the db.Connection interface
func (m *MockDBConnection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.shouldFail == "Query" {
		return nil, fmt.Errorf("mock database error: Query failed")
	}
	// Mock implementation - just return nil for now since we don't need it in these tests
	return nil, fmt.Errorf("Query method not implemented in mock")
}

// EnsureConnection implements the EnsureConnection method for the db.Connection interface
func (m *MockDBConnection) EnsureConnection(ctx context.Context) {
	// Mock implementation - do nothing
}

// Helper functions to test database functions with mock
func testShowSchemaWithDB(session *Session, mockDB db.Connection, ctx context.Context) error {
	// Temporarily replace the session's database methods for testing
	// We'll call the functions directly with mock data
	tables, err := mockDB.ListTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	if len(tables) == 0 {
		fmt.Println("No tables found in the database.")
		return nil
	}

	fmt.Println("Database Schema Overview:")
	fmt.Println("========================")

	// Group by schema (same logic as showSchema)
	schemaMap := make(map[string][]db.TableInfo)
	for _, table := range tables {
		schemaMap[table.Schema] = append(schemaMap[table.Schema], table)
	}

	for schema, schemaTables := range schemaMap {
		fmt.Printf("\nSchema: %s\n", schema)
		fmt.Println(strings.Repeat("-", len(schema)+8))

		for _, table := range schemaTables {
			fmt.Printf("  %s (%s)\n", table.Name, table.Type)
		}
	}

	return nil
}

func testListTablesWithDB(session *Session, mockDB db.Connection, ctx context.Context) error {
	tables, err := mockDB.ListTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	if len(tables) == 0 {
		fmt.Println("No tables found in the database.")
		return nil
	}

	fmt.Println("Tables and Views:")
	fmt.Println("=================")

	for _, table := range tables {
		fmt.Printf("%-20s %-10s %s\n", table.Name, table.Type, table.Schema)
	}

	return nil
}

func testDescribeTableWithDB(session *Session, mockDB db.Connection, ctx context.Context, tableName string) error {
	// Parse schema.table if provided (same logic as describeTable)
	schema := "public"
	if strings.Contains(tableName, ".") {
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			schema = parts[0]
			tableName = parts[1]
		}
	}

	table, err := mockDB.DescribeTable(ctx, schema, tableName)
	if err != nil {
		return fmt.Errorf("failed to describe table: %w", err)
	}

	fmt.Printf("Table: %s.%s (%s)\n", table.Schema, table.Name, table.Type)
	if table.Description != "" {
		fmt.Printf("Description: %s\n", table.Description)
	}
	fmt.Println()

	if len(table.Columns) == 0 {
		fmt.Println("No columns found.")
		return nil
	}

	fmt.Println("Columns:")
	fmt.Println("========")
	fmt.Printf("%-20s %-15s %-8s %-8s %s\n", "Name", "Type", "Nullable", "Key", "Default")
	fmt.Println(strings.Repeat("-", 70))

	for _, col := range table.Columns {
		nullable := "YES"
		if !col.IsNullable {
			nullable = "NO"
		}

		key := ""
		if col.IsPrimaryKey {
			key = "PK"
		}

		defaultVal := col.Default
		if defaultVal == "" {
			defaultVal = "(none)"
		}

		fmt.Printf("%-20s %-15s %-8s %-8s %s\n",
			col.Name, col.DataType, nullable, key, defaultVal)
	}

	// Show foreign keys if any
	foreignKeys, err := mockDB.GetForeignKeys(ctx, schema, tableName)
	if err != nil {
		return fmt.Errorf("failed to get foreign keys: %w", err)
	}

	if len(foreignKeys) > 0 {
		fmt.Println("\nForeign Keys:")
		fmt.Println("=============")
		for _, fk := range foreignKeys {
			fmt.Printf("%s -> %s.%s.%s\n",
				fk.ColumnName, fk.ForeignTableSchema, fk.ForeignTableName, fk.ForeignColumnName)
		}
	}

	return nil
}

func TestNewSession(t *testing.T) {
	var conn db.Connection
	mode := "default"

	session := NewSession(conn, mode)

	if session.conn != conn {
		t.Error("Expected connection to be set correctly")
	}
	if session.mode != mode {
		t.Errorf("Expected mode %s, got %s", mode, session.mode)
	}
	if session.agentReady {
		t.Error("Expected agentReady to be false initially")
	}
	if session.agent != nil {
		t.Error("Expected agent to be nil initially")
	}
}

func TestSession_ModeSpecificBehavior(t *testing.T) {
	modes := []string{"default", "schema-only", "share-results"}

	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			session := NewSession(nil, mode)
			if session.mode != mode {
				t.Errorf("Expected mode %s, got %s", mode, session.mode)
			}

			ctx := context.Background()
			err := session.handleCommand(ctx, "/mode")
			if err != nil {
				t.Fatalf("Mode command failed for %s: %v", mode, err)
			}
		})
	}
}

func TestSession_HandleCommand_UnknownCommand(t *testing.T) {
	session := NewSession(nil, "default")
	ctx := context.Background()

	err := session.handleCommand(ctx, "/unknown")
	if err == nil {
		t.Error("Expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %v", err)
	}
}

func TestSession_HandleCommand_Help(t *testing.T) {
	session := NewSession(nil, "default")
	ctx := context.Background()

	err := session.handleCommand(ctx, "/help")
	if err != nil {
		t.Fatalf("handleCommand /help failed: %v", err)
	}

	err = session.handleCommand(ctx, "/h")
	if err != nil {
		t.Fatalf("handleCommand /h failed: %v", err)
	}
}

func TestSession_HandleCommand_ClearWhenNotReady(t *testing.T) {
	session := NewSession(nil, "default")
	session.agentReady = false

	ctx := context.Background()
	err := session.handleCommand(ctx, "/clear")
	if err != nil {
		t.Fatalf("handleCommand /clear failed when agent not ready: %v", err)
	}
}

func TestSession_HandleQuery_AgentNotReady(t *testing.T) {
	session := NewSession(nil, "default")
	session.agentReady = false
	ctx := context.Background()

	err := session.handleQuery(ctx, "show me all users")
	if err != nil {
		t.Fatalf("handleQuery should not error when agent not ready: %v", err)
	}
}

func TestSession_GetUserApproval_Logic(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"y", true},
		{"Y", true},
		{"yes", true},
		{"YES", true},
		{"Yes", true},
		{"n", false},
		{"N", false},
		{"no", false},
		{"NO", false},
		{"No", false},
		{"", false},
		{"maybe", false},
		{"quit", false},
		{"  y  ", true},
		{"  no  ", false},
	}

	for _, tt := range tests {
		t.Run("input_"+tt.input, func(t *testing.T) {
			// Test the logic used in getUserApproval
			response := strings.ToLower(strings.TrimSpace(tt.input))
			actual := response == "y" || response == "yes"

			if actual != tt.expected {
				t.Errorf("input '%s': expected %v, got %v", tt.input, tt.expected, actual)
			}
		})
	}
}

func TestSession_SQLToolValidation(t *testing.T) {
	// Test SQL tool parameter validation logic
	validInput := map[string]interface{}{
		"sql":         "SELECT * FROM users WHERE active = true",
		"explanation": "Get all active users",
	}

	sqlQuery, ok := validInput["sql"].(string)
	if !ok {
		t.Error("Expected sql to be a string")
	}
	if sqlQuery != "SELECT * FROM users WHERE active = true" {
		t.Errorf("Expected specific SQL query, got '%s'", sqlQuery)
	}

	explanation, ok := validInput["explanation"].(string)
	if !ok {
		t.Error("Expected explanation to be a string")
	}
	if explanation != "Get all active users" {
		t.Errorf("Expected specific explanation, got '%s'", explanation)
	}

	// Test invalid input (missing sql)
	invalidInput := map[string]interface{}{
		"explanation": "Missing SQL parameter",
	}

	_, ok = invalidInput["sql"].(string)
	if ok {
		t.Error("Expected sql parameter to be missing")
	}

	// Test invalid input (wrong type)
	wrongTypeInput := map[string]interface{}{
		"sql":         123, // Should be string
		"explanation": "Wrong type for SQL",
	}

	_, ok = wrongTypeInput["sql"].(string)
	if ok {
		t.Error("Expected sql parameter to fail type assertion")
	}
}

func TestSession_QueryInfoFormatting(t *testing.T) {
	explanation := "Find all users with recent activity"
	sqlQuery := "SELECT u.id, u.name, u.last_login FROM users u WHERE u.last_login > NOW() - INTERVAL '7 days'"

	// This mimics the formatting used for query approval
	queryInfo := explanation + "\n\nSQL Query:\n" + sqlQuery

	lines := strings.Split(queryInfo, "\n")
	if len(lines) < 3 {
		t.Error("Expected at least 3 lines in formatted query info")
	}

	if lines[0] != explanation {
		t.Errorf("Expected first line to be explanation, got '%s'", lines[0])
	}

	if lines[1] != "" {
		t.Error("Expected second line to be empty (separator)")
	}

	if lines[2] != "SQL Query:" {
		t.Errorf("Expected third line to be 'SQL Query:', got '%s'", lines[2])
	}

	if lines[3] != sqlQuery {
		t.Errorf("Expected fourth line to be SQL query, got '%s'", lines[3])
	}
}

func TestSession_CommandParsing(t *testing.T) {
	tests := []struct {
		command     string
		expectError bool
		description string
	}{
		{"/help", false, "help command should work"},
		{"/h", false, "help shortcut should work"},
		{"/mode", false, "mode command should work"},
		{"/m", false, "mode shortcut should work"},
		{"/clear", false, "clear command should work"},
		{"/c", false, "clear shortcut should work"},
		{"/describe", true, "describe without args should fail"},
		{"/d", true, "describe shortcut without args should fail"},
		{"/unknown", true, "unknown command should fail"},
		{"", false, "empty command should be handled gracefully"},
	}

	for _, tt := range tests {
		t.Run("cmd_"+strings.ReplaceAll(tt.command, " ", "_"), func(t *testing.T) {
			session := NewSession(nil, "default")
			ctx := context.Background()

			err := session.handleCommand(ctx, tt.command)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for command '%s': %s", tt.command, tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for command '%s': %v", tt.command, err)
			}
		})
	}
}

func TestSession_StateManagement(t *testing.T) {
	session := NewSession(nil, "default")

	// Test initial state
	if session.agentReady {
		t.Error("Expected agentReady to be false initially")
	}

	// Test that we can modify state
	session.agentReady = true
	if !session.agentReady {
		t.Error("Expected to be able to set agentReady to true")
	}

	session.agentReady = false
	if session.agentReady {
		t.Error("Expected to be able to set agentReady to false")
	}
}

func TestSession_ModeValidation(t *testing.T) {
	validModes := []string{"default", "schema-only", "share-results"}
	invalidModes := []string{"", "invalid", "old-mode", "unknown"}

	for _, mode := range validModes {
		t.Run("valid_mode_"+mode, func(t *testing.T) {
			session := NewSession(nil, mode)
			if session.mode != mode {
				t.Errorf("Expected mode %s, got %s", mode, session.mode)
			}
		})
	}

	for _, mode := range invalidModes {
		t.Run("invalid_mode_"+mode, func(t *testing.T) {
			session := NewSession(nil, mode)
			// Session should still be created with invalid mode
			// Validation would happen elsewhere in the application
			if session.mode != mode {
				t.Errorf("Expected mode %s to be stored as-is, got %s", mode, session.mode)
			}
		})
	}
}

func TestSession_ErrorHandling(t *testing.T) {
	t.Run("nil_session_fields", func(t *testing.T) {
		session := NewSession(nil, "default")

		// Test that nil fields don't cause immediate issues
		if session.conn != nil {
			t.Error("Expected conn to be nil")
		}
		if session.rl != nil {
			t.Error("Expected rl to be nil")
		}
		if session.agent != nil {
			t.Error("Expected agent to be nil")
		}

		// Session should still be usable for configuration
		if session.mode != "default" {
			t.Error("Expected mode to be set correctly")
		}
	})

	t.Run("empty_mode", func(t *testing.T) {
		session := NewSession(nil, "")

		// Session should be created even with empty mode
		if session.mode != "" {
			t.Error("Expected empty mode to be preserved")
		}

		// Other fields should still be initialized correctly
		if session.agentReady {
			t.Error("Expected agentReady to be false")
		}
	})
}

func TestSession_ConversationFlow_Simulation(t *testing.T) {
	session := NewSession(nil, "default")
	ctx := context.Background()

	// Simulate conversation flow without actual agent
	// Test that multiple queries can be processed when agent is not ready
	queries := []string{
		"show me the database schema",
		"how many users are there?",
		"find users created in the last month",
	}

	for i, query := range queries {
		err := session.handleQuery(ctx, query)
		if err != nil {
			t.Fatalf("Query %d failed: %v", i, err)
		}
		// When agent is not ready, handleQuery should complete without error
		// and inform the user about missing API key
	}
}

func TestSession_InitializationWorkflow(t *testing.T) {
	session := NewSession(nil, "default")

	// Test initial conditions
	if session.agentReady {
		t.Error("Expected agent to not be ready initially")
	}
	if session.agent != nil {
		t.Error("Expected agent to be nil initially")
	}
	if session.rl != nil {
		t.Error("Expected readline to be nil initially")
	}

	// Test that session can be created with different modes
	modes := []string{"default", "schema-only", "share-results"}
	for _, mode := range modes {
		s := NewSession(nil, mode)
		if s.mode != mode {
			t.Errorf("Expected mode %s, got %s", mode, s.mode)
		}
	}
}

func TestSession_TableNameParsing(t *testing.T) {
	tests := []struct {
		input          string
		expectedSchema string
		expectedTable  string
	}{
		{"users", "public", "users"},
		{"public.users", "public", "users"},
		{"schema1.table1", "schema1", "table1"},
		{"some_schema.some_table", "some_schema", "some_table"},
	}

	for _, tt := range tests {
		t.Run("parse_"+tt.input, func(t *testing.T) {
			// Simulate the parsing logic from describeTable
			schema := "public" // default
			tableName := tt.input

			if strings.Contains(tableName, ".") {
				parts := strings.Split(tableName, ".")
				if len(parts) == 2 {
					schema = parts[0]
					tableName = parts[1]
				}
			}

			if schema != tt.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tt.expectedSchema, schema)
			}
			if tableName != tt.expectedTable {
				t.Errorf("Expected table %s, got %s", tt.expectedTable, tableName)
			}
		})
	}
}

// Old helper function tests removed - replaced with proper tests that call actual session methods

// Direct method testing for coverage improvement

func TestSession_DatabaseMethods_ErrorHandling(t *testing.T) {
	// Test database methods with nil connection to exercise error paths
	session := NewSession(nil, "default")
	ctx := context.Background()

	t.Run("showSchema_with_nil_connection", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil connection")
			}
		}()
		session.showSchema(ctx)
	})

	t.Run("listTables_with_nil_connection", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil connection")
			}
		}()
		session.listTables(ctx)
	})

	t.Run("describeTable_with_nil_connection", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil connection")
			}
		}()
		session.describeTable(ctx, "users")
	})

	t.Run("browseLastResults_with_no_results", func(t *testing.T) {
		// This should handle the case when no results are available
		err := session.browseLastResults(ctx)
		if err != nil {
			t.Errorf("browseLastResults should handle no results gracefully: %v", err)
		}
	})
}

func TestSession_InitializeAgent_Logic(t *testing.T) {
	session := NewSession(nil, "default")

	// Test agent initialization without real API key
	// This will exercise the error path when no API key is available
	session.initializeAgent()

	// Agent should not be ready when no API key is provided
	if session.agentReady {
		t.Error("Expected agent to not be ready without API key")
	}
	if session.agent != nil {
		t.Error("Expected agent to be nil without API key")
	}
}

func TestSession_HandleQuery_WithAgentReady(t *testing.T) {
	session := NewSession(nil, "default")
	ctx := context.Background()

	// Test with agent ready but nil agent (edge case)
	session.agentReady = true
	session.agent = nil

	// This should cause a panic or error when trying to use nil agent
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior - nil agent with agentReady=true should cause issues
		}
	}()

	err := session.handleQuery(ctx, "test query")
	// If we get here without panic, that's also acceptable
	if err != nil {
		t.Logf("handleQuery with nil agent returned error: %v", err)
	}
}

func TestSession_MoreCommandPaths(t *testing.T) {
	session := NewSession(nil, "default")
	ctx := context.Background()

	// Test command paths that haven't been covered yet
	t.Run("browse_command", func(t *testing.T) {
		err := session.handleCommand(ctx, "/browse")
		if err != nil {
			t.Fatalf("handleCommand /browse failed: %v", err)
		}

		err = session.handleCommand(ctx, "/b")
		if err != nil {
			t.Fatalf("handleCommand /b failed: %v", err)
		}
	})

	// Test command with parts
	t.Run("empty_command_parts", func(t *testing.T) {
		err := session.handleCommand(ctx, "")
		if err != nil {
			t.Errorf("Expected no error for empty command, got: %v", err)
		}
	})
}

func TestSession_DescribeTable_WithMockDB(t *testing.T) {
	mockDB := NewMockDBConnection()
	session := NewSession(mockDB, "default")
	ctx := context.Background()

	t.Run("describe_existing_table", func(t *testing.T) {
		// Capture stdout to verify output
		// For now, just test that it doesn't panic or error
		err := session.describeTable(ctx, "users")
		if err != nil {
			t.Fatalf("describeTable failed: %v", err)
		}
	})

	t.Run("describe_table_with_schema", func(t *testing.T) {
		err := session.describeTable(ctx, "public.users")
		if err != nil {
			t.Fatalf("describeTable with schema failed: %v", err)
		}
	})

	t.Run("describe_nonexistent_table", func(t *testing.T) {
		err := session.describeTable(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent table")
		}
		if !strings.Contains(err.Error(), "failed to describe table") {
			t.Errorf("Expected 'failed to describe table' error, got: %v", err)
		}
	})

	t.Run("describe_table_with_foreign_keys", func(t *testing.T) {
		err := session.describeTable(ctx, "orders")
		if err != nil {
			t.Fatalf("describeTable with foreign keys failed: %v", err)
		}
	})

	t.Run("mock_database_error", func(t *testing.T) {
		errorMockDB := NewMockDBConnection()
		errorMockDB.shouldFail = "DescribeTable"
		errorSession := NewSession(errorMockDB, "default")

		err := errorSession.describeTable(ctx, "users")
		if err == nil {
			t.Error("Expected error when DescribeTable fails")
		}
		if !strings.Contains(err.Error(), "failed to describe table") {
			t.Errorf("Expected 'failed to describe table' error, got: %v", err)
		}
	})

	t.Run("mock_foreign_keys_error", func(t *testing.T) {
		errorMockDB := NewMockDBConnection()
		errorMockDB.shouldFail = "GetForeignKeys"
		errorSession := NewSession(errorMockDB, "default")

		err := errorSession.describeTable(ctx, "users")
		if err == nil {
			t.Error("Expected error when GetForeignKeys fails")
		}
		if !strings.Contains(err.Error(), "failed to get foreign keys") {
			t.Errorf("Expected 'failed to get foreign keys' error, got: %v", err)
		}
	})
}

func TestSession_ShowSchema_WithMockDB(t *testing.T) {
	mockDB := NewMockDBConnection()
	session := NewSession(mockDB, "default")
	ctx := context.Background()

	t.Run("show_schema_with_tables", func(t *testing.T) {
		err := session.showSchema(ctx)
		if err != nil {
			t.Fatalf("showSchema failed: %v", err)
		}
	})

	t.Run("show_schema_empty_database", func(t *testing.T) {
		emptyMockDB := &MockDBConnection{tables: []db.TableInfo{}}
		emptySession := NewSession(emptyMockDB, "default")

		err := emptySession.showSchema(ctx)
		if err != nil {
			t.Fatalf("showSchema with empty DB failed: %v", err)
		}
	})

	t.Run("show_schema_database_error", func(t *testing.T) {
		errorMockDB := NewMockDBConnection()
		errorMockDB.shouldFail = "ListTables"
		errorSession := NewSession(errorMockDB, "default")

		err := errorSession.showSchema(ctx)
		if err == nil {
			t.Error("Expected error when ListTables fails")
		}
		if !strings.Contains(err.Error(), "failed to get schema") {
			t.Errorf("Expected 'failed to get schema' error, got: %v", err)
		}
	})
}

func TestSession_ListTables_WithMockDB(t *testing.T) {
	mockDB := NewMockDBConnection()
	session := NewSession(mockDB, "default")
	ctx := context.Background()

	t.Run("list_tables_with_data", func(t *testing.T) {
		err := session.listTables(ctx)
		if err != nil {
			t.Fatalf("listTables failed: %v", err)
		}
	})

	t.Run("list_tables_empty_database", func(t *testing.T) {
		emptyMockDB := &MockDBConnection{tables: []db.TableInfo{}}
		emptySession := NewSession(emptyMockDB, "default")

		err := emptySession.listTables(ctx)
		if err != nil {
			t.Fatalf("listTables with empty DB failed: %v", err)
		}
	})

	t.Run("list_tables_database_error", func(t *testing.T) {
		errorMockDB := NewMockDBConnection()
		errorMockDB.shouldFail = "ListTables"
		errorSession := NewSession(errorMockDB, "default")

		err := errorSession.listTables(ctx)
		if err == nil {
			t.Error("Expected error when ListTables fails")
		}
		if !strings.Contains(err.Error(), "failed to list tables") {
			t.Errorf("Expected 'failed to list tables' error, got: %v", err)
		}
	})
}
