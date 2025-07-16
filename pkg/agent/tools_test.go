package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/AliciaSchep/pgbabble/pkg/db"
	"github.com/jackc/pgx/v5"
)

func TestMarshalToolsToJSON(t *testing.T) {
	// Create test tools
	tools := []*Tool{
		{
			Name:        "test_tool_1",
			Description: "First test tool",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"param1": map[string]interface{}{
						"type": "string",
					},
				},
				Required: []string{"param1"},
			},
			Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
				return &ToolResult{Content: "test"}, nil
			},
		},
		{
			Name:        "test_tool_2",
			Description: "Second test tool",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
				Required:   []string{},
			},
			Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
				return &ToolResult{Content: "test2"}, nil
			},
		},
	}

	// Test JSON marshaling
	jsonStr, err := MarshalToolsToJSON(tools)
	if err != nil {
		t.Errorf("unexpected error marshaling tools: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Verify JSON contains expected tool names
	if !strings.Contains(jsonStr, "test_tool_1") {
		t.Error("expected JSON to contain 'test_tool_1'")
	}
	if !strings.Contains(jsonStr, "test_tool_2") {
		t.Error("expected JSON to contain 'test_tool_2'")
	}

	// Verify it's valid JSON by unmarshaling
	var result []interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Errorf("failed to unmarshal generated JSON: %v", err)
	}
}

func TestToolResult_Basic(t *testing.T) {
	// Test successful result
	result := &ToolResult{
		Content: "success message",
		IsError: false,
	}

	if result.Content != "success message" {
		t.Errorf("expected content 'success message', got '%s'", result.Content)
	}
	if result.IsError {
		t.Error("expected IsError to be false")
	}

	// Test error result
	errorResult := &ToolResult{
		Content: "error message",
		IsError: true,
	}

	if !errorResult.IsError {
		t.Error("expected IsError to be true")
	}
}

func TestToolSchema_Basic(t *testing.T) {
	schema := ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"param1": map[string]interface{}{
				"type":        "string",
				"description": "Test parameter",
			},
			"param2": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
		},
		Required: []string{"param1"},
	}

	if schema.Type != "object" {
		t.Errorf("expected type 'object', got '%s'", schema.Type)
	}

	if len(schema.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(schema.Properties))
	}

	if len(schema.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(schema.Required))
	}

	if schema.Required[0] != "param1" {
		t.Errorf("expected required field 'param1', got '%s'", schema.Required[0])
	}
}

func TestTool_BasicExecution(t *testing.T) {
	tool := &Tool{
		Name:        "echo_tool",
		Description: "Echoes input back",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"message"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			message, ok := input["message"].(string)
			if !ok {
				return &ToolResult{
					Content: "Invalid message type",
					IsError: true,
				}, nil
			}
			return &ToolResult{
				Content: "Echo: " + message,
				IsError: false,
			}, nil
		},
	}

	ctx := context.Background()

	// Test successful execution
	input := map[string]interface{}{
		"message": "hello world",
	}
	result, err := tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected successful execution")
	}
	if !strings.Contains(result.Content, "hello world") {
		t.Errorf("expected result to contain 'hello world', got '%s'", result.Content)
	}

	// Test with invalid input
	invalidInput := map[string]interface{}{
		"message": 123, // Invalid type
	}
	result, err = tool.Handler(ctx, invalidInput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result with invalid input")
	}
}

// MockConnection implements db.Connection interface for testing
type MockConnection struct {
	tables     []db.TableInfo
	foreignKeys []db.ForeignKeyInfo
	columns    []db.ColumnInfo
	queryError error
	queryResult map[string]interface{}
}

func (m *MockConnection) ListTables(ctx context.Context) ([]db.TableInfo, error) {
	return m.tables, nil
}

func (m *MockConnection) DescribeTable(ctx context.Context, schema, tableName string) (*db.TableInfo, error) {
	for _, table := range m.tables {
		if table.Schema == schema && table.Name == tableName {
			return &table, nil
		}
	}
	return nil, fmt.Errorf("table %s.%s not found", schema, tableName)
}

func (m *MockConnection) GetForeignKeys(ctx context.Context, schema, tableName string) ([]db.ForeignKeyInfo, error) {
	var result []db.ForeignKeyInfo
	for _, fk := range m.foreignKeys {
		if fk.TableSchema == schema && fk.TableName == tableName {
			result = append(result, fk)
		}
	}
	return result, nil
}

func (m *MockConnection) SearchColumns(ctx context.Context, pattern string) ([]db.ColumnInfo, error) {
	var result []db.ColumnInfo
	for _, col := range m.columns {
		if strings.Contains(strings.ToLower(col.Name), strings.ToLower(pattern)) {
			result = append(result, col)
		}
	}
	return result, nil
}

func (m *MockConnection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryError != nil {
		return nil, m.queryError
	}
	// Return nil as we're not testing actual query execution here
	return nil, nil
}

func (m *MockConnection) EnsureConnection(ctx context.Context) {
	// No-op for mock
}

func TestCreateSchemaTools(t *testing.T) {
	mockDB := &MockConnection{
		tables: []db.TableInfo{
			{
				Schema: "public",
				Name:   "users",
				Columns: []db.ColumnInfo{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
					{Name: "name", DataType: "varchar", IsNullable: false},
					{Name: "email", DataType: "varchar", IsNullable: true},
				},
			},
			{
				Schema: "public",
				Name:   "orders",
				Columns: []db.ColumnInfo{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
					{Name: "user_id", DataType: "integer", IsNullable: false},
					{Name: "total", DataType: "decimal", IsNullable: false},
				},
			},
		},
	}

	tools := CreateSchemaTools(mockDB, "default")
	if len(tools) == 0 {
		t.Fatal("expected schema tools to be created")
	}

	// Verify we get the expected tools
	expectedTools := []string{"list_tables", "describe_table", "get_relationships", "search_columns"}
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("expected tool '%s' not found in schema tools", expected)
		}
	}
}

func TestCreateExecutionTools(t *testing.T) {
	mockDB := &MockConnection{}

	getUserApproval := func(query string) bool { return true }
	tools := CreateExecutionTools(mockDB, getUserApproval, "default")
	if len(tools) == 0 {
		t.Fatal("expected execution tools to be created")
	}

	// Verify we get the expected tools
	expectedTools := []string{"execute_sql", "explain_query"}
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("expected tool '%s' not found in execution tools", expected)
		}
	}
}

func TestListTablesTool(t *testing.T) {
	mockDB := &MockConnection{
		tables: []db.TableInfo{
			{Schema: "public", Name: "users", EstimatedRows: 100},
			{Schema: "public", Name: "orders", EstimatedRows: 50},
			{Schema: "inventory", Name: "products", EstimatedRows: 200},
		},
	}

	tool := createListTablesTool(mockDB, "default")
	if tool.Name != "list_tables" {
		t.Errorf("expected tool name 'list_tables', got '%s'", tool.Name)
	}

	// Test tool execution
	ctx := context.Background()
	result, err := tool.Handler(ctx, map[string]interface{}{})
	if err != nil {
		t.Errorf("unexpected error executing list_tables tool: %v", err)
	}

	if result.IsError {
		t.Errorf("expected successful result, got error: %s", result.Content)
	}

	// Verify result contains table information
	content := result.Content
	if !strings.Contains(content, "users") || !strings.Contains(content, "orders") || !strings.Contains(content, "products") {
		t.Errorf("expected result to contain table names, got: %s", content)
	}
}

func TestDescribeTableTool(t *testing.T) {
	mockDB := &MockConnection{
		tables: []db.TableInfo{
			{
				Schema: "public",
				Name:   "users",
				Columns: []db.ColumnInfo{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
					{Name: "name", DataType: "varchar", IsNullable: false},
					{Name: "email", DataType: "varchar", IsNullable: true},
				},
			},
		},
	}

	tool := createDescribeTableTool(mockDB)
	if tool.Name != "describe_table" {
		t.Errorf("expected tool name 'describe_table', got '%s'", tool.Name)
	}

	// Test successful execution
	ctx := context.Background()
	input := map[string]interface{}{
		"table_name": "public.users",
	}
	result, err := tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error executing describe_table tool: %v", err)
	}

	if result.IsError {
		t.Errorf("expected successful result, got error: %s", result.Content)
	}

	// Verify result contains column information
	content := result.Content
	if !strings.Contains(content, "id") || !strings.Contains(content, "name") || !strings.Contains(content, "email") {
		t.Errorf("expected result to contain column names, got: %s", content)
	}

	// Test missing required parameters
	result, err = tool.Handler(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing parameters")
	}
	if result == nil || !result.IsError {
		t.Error("expected error result for missing parameters")
	}

	// Test with missing table
	input = map[string]interface{}{
		"table_name": "public.nonexistent",
	}
	result, err = tool.Handler(ctx, input)
	if err == nil {
		t.Error("expected error for nonexistent table")
	}
	if result == nil || !result.IsError {
		t.Error("expected error result for nonexistent table")
	}
}

func TestGetRelationshipsTool(t *testing.T) {
	mockDB := &MockConnection{
		foreignKeys: []db.ForeignKeyInfo{
			{
				TableSchema:        "public",
				TableName:          "orders",
				ColumnName:         "user_id",
				ForeignTableSchema: "public",
				ForeignTableName:   "users",
				ForeignColumnName:  "id",
			},
		},
	}

	tool := createGetRelationshipsTool(mockDB)
	if tool.Name != "get_relationships" {
		t.Errorf("expected tool name 'get_relationships', got '%s'", tool.Name)
	}

	// Test successful execution
	ctx := context.Background()
	input := map[string]interface{}{
		"table_name": "public.orders",
	}
	result, err := tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error executing get_relationships tool: %v", err)
	}

	if result.IsError {
		t.Errorf("expected successful result, got error: %s", result.Content)
	}

	// Verify result contains foreign key information
	content := result.Content
	if !strings.Contains(content, "user_id") || !strings.Contains(content, "users") {
		t.Errorf("expected result to contain foreign key info, got: %s", content)
	}
}

func TestSearchColumnsTool(t *testing.T) {
	mockDB := &MockConnection{
		columns: []db.ColumnInfo{
			{Name: "user_id", DataType: "integer"},
			{Name: "user_id", DataType: "integer"},
			{Name: "email", DataType: "varchar"},
		},
	}

	tool := createSearchColumnsTool(mockDB)
	if tool.Name != "search_columns" {
		t.Errorf("expected tool name 'search_columns', got '%s'", tool.Name)
	}

	// Test successful search
	ctx := context.Background()
	input := map[string]interface{}{
		"pattern": "user_id",
	}
	result, err := tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error executing search_columns tool: %v", err)
	}

	if result.IsError {
		t.Errorf("expected successful result, got error: %s", result.Content)
	}

	// Verify result contains matching columns
	content := result.Content
	if !strings.Contains(content, "user_id") {
		t.Errorf("expected result to contain 'user_id', got: %s", content)
	}

	// Test with no matches
	input = map[string]interface{}{
		"pattern": "nonexistent_column",
	}
	result, err = tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected successful result with no matches, got error: %s", result.Content)
	}
}

func TestCreateExecuteSQLTool(t *testing.T) {
	mockDB := &MockConnection{}

	getUserApproval := func(query string) bool { return true }
	tool := createExecuteSQLTool(mockDB, getUserApproval, "default")
	if tool.Name != "execute_sql" {
		t.Errorf("expected tool name 'execute_sql', got '%s'", tool.Name)
	}

	// Test missing required parameters
	ctx := context.Background()
	result, err := tool.Handler(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing parameters")
	}
	if result == nil || !result.IsError {
		t.Error("expected error result for missing parameters")
	}

	// Test invalid SQL (non-SELECT)
	input := map[string]interface{}{
		"sql":         "DELETE FROM users",
		"explanation": "Delete users",
	}
	result, err = tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for non-SELECT query")
	}

	// Test dangerous patterns
	input = map[string]interface{}{
		"sql":         "SELECT * FROM users; DROP TABLE users;",
		"explanation": "Query with dangerous pattern",
	}
	result, err = tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for dangerous SQL")
	}
}

func TestCreateExplainQueryTool(t *testing.T) {
	mockDB := &MockConnection{}

	getUserApproval := func(query string) bool { return true }
	tool := createExplainQueryTool(mockDB, getUserApproval, "default")
	if tool.Name != "explain_query" {
		t.Errorf("expected tool name 'explain_query', got '%s'", tool.Name)
	}

	// Test missing required parameters
	ctx := context.Background()
	result, err := tool.Handler(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing parameters")
	}
	if result == nil || !result.IsError {
		t.Error("expected error result for missing parameters")
	}

	// Test invalid SQL (non-SELECT)
	input := map[string]interface{}{
		"sql":         "DELETE FROM users",
		"explanation": "Delete users",
	}
	result, err = tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for non-SELECT query")
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a smaller", 5, 10, 5},
		{"b smaller", 10, 5, 5},
		{"equal", 5, 5, 5},
		{"negative numbers", -5, -10, -10},
		{"zero", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestExecuteApprovedSQL(t *testing.T) {
	mockDB := &MockConnection{}
	ctx := context.Background()

	// Test invalid query (non-SELECT) - this will fail validation
	result, err := executeApprovedSQL(ctx, mockDB, "DELETE FROM users", "default")
	if err == nil {
		t.Error("expected error for non-SELECT query")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got: %s", result)
	}

	// Test dangerous query pattern
	result, err = executeApprovedSQL(ctx, mockDB, "SELECT * FROM users; DROP TABLE users;", "default")
	if err == nil {
		t.Error("expected error for dangerous query")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got: %s", result)
	}
}

// Note: executeSelectQuery and executeExplainQuery are complex functions that require
// actual database query execution with pgx.Rows. These would need more sophisticated
// mocking to test thoroughly, but the validation logic is covered by other tests.

