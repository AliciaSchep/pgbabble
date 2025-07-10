package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pgbabble/pkg/db"
)

// CreateSchemaTools creates all schema inspection tools for the LLM
func CreateSchemaTools(conn *db.Connection) []*Tool {
	return []*Tool{
		createListTablesToolI(conn),
		createDescribeTableTool(conn),
		createGetRelationshipsTool(conn),
		createSearchColumnsTool(conn),
	}
}

// CreateExecutionTools creates SQL execution tools for the LLM
func CreateExecutionTools(conn *db.Connection, getUserApproval func(string) bool) []*Tool {
	return []*Tool{
		createExecuteSQLTool(conn, getUserApproval),
		createExplainQueryTool(conn, getUserApproval),
	}
}

// createListTablesTool creates a tool to list all tables and views
func createListTablesToolI(conn *db.Connection) *Tool {
	return &Tool{
		Name:        "list_tables",
		Description: "Lists all tables and views in the database with their types and schemas",
		InputSchema: ToolSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
			Required:   []string{},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			tables, err := conn.ListTables(ctx)
			if err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("Error listing tables: %v", err),
					IsError: true,
				}, err
			}

			if len(tables) == 0 {
				return &ToolResult{
					Content: "No tables found in the database.",
				}, nil
			}

			// Group by schema for better organization
			schemaMap := make(map[string][]db.TableInfo)
			for _, table := range tables {
				schemaMap[table.Schema] = append(schemaMap[table.Schema], table)
			}

			var result strings.Builder
			result.WriteString("Database Tables and Views:\n")
			result.WriteString("=========================\n\n")

			for schema, schemaTables := range schemaMap {
				result.WriteString(fmt.Sprintf("Schema: %s\n", schema))
				result.WriteString(strings.Repeat("-", len(schema)+8) + "\n")

				for _, table := range schemaTables {
					result.WriteString(fmt.Sprintf("- %s (%s)\n", table.Name, table.Type))
				}
				result.WriteString("\n")
			}

			return &ToolResult{
				Content: result.String(),
			}, nil
		},
	}
}

// createDescribeTableTool creates a tool to describe a specific table
func createDescribeTableTool(conn *db.Connection) *Tool {
	return &Tool{
		Name:        "describe_table",
		Description: "Gets detailed information about a specific table including columns, data types, constraints, and relationships",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the table to describe. Can include schema (e.g., 'public.users' or just 'users')",
				},
			},
			Required: []string{"table_name"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			tableName, ok := input["table_name"].(string)
			if !ok {
				return &ToolResult{
					Content: "Error: table_name must be a string",
					IsError: true,
				}, fmt.Errorf("invalid table_name parameter")
			}

			// Parse schema.table if provided
			schema := "public"
			if strings.Contains(tableName, ".") {
				parts := strings.Split(tableName, ".")
				if len(parts) == 2 {
					schema = parts[0]
					tableName = parts[1]
				}
			}

			table, err := conn.DescribeTable(ctx, schema, tableName)
			if err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("Error describing table %s.%s: %v", schema, tableName, err),
					IsError: true,
				}, err
			}

			var result strings.Builder
			result.WriteString(fmt.Sprintf("Table: %s.%s (%s)\n", table.Schema, table.Name, table.Type))
			if table.Description != "" {
				result.WriteString(fmt.Sprintf("Description: %s\n", table.Description))
			}
			result.WriteString("\n")

			if len(table.Columns) == 0 {
				result.WriteString("No columns found.\n")
				return &ToolResult{Content: result.String()}, nil
			}

			result.WriteString("Columns:\n")
			result.WriteString("========\n")
			result.WriteString(fmt.Sprintf("%-20s %-15s %-8s %-8s %s\n", "Name", "Type", "Nullable", "Key", "Default"))
			result.WriteString(strings.Repeat("-", 70) + "\n")

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

				result.WriteString(fmt.Sprintf("%-20s %-15s %-8s %-8s %s\n",
					col.Name, col.DataType, nullable, key, defaultVal))
			}

			// Add foreign keys if any
			foreignKeys, err := conn.GetForeignKeys(ctx, schema, tableName)
			if err == nil && len(foreignKeys) > 0 {
				result.WriteString("\nForeign Keys:\n")
				result.WriteString("=============\n")
				for _, fk := range foreignKeys {
					result.WriteString(fmt.Sprintf("%s -> %s.%s.%s\n",
						fk.ColumnName, fk.ForeignTableSchema, fk.ForeignTableName, fk.ForeignColumnName))
				}
			}

			return &ToolResult{
				Content: result.String(),
			}, nil
		},
	}
}

// createGetRelationshipsTool creates a tool to get foreign key relationships for a table
func createGetRelationshipsTool(conn *db.Connection) *Tool {
	return &Tool{
		Name:        "get_relationships",
		Description: "Gets foreign key relationships for a specific table, showing how it connects to other tables",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the table to get relationships for. Can include schema (e.g., 'public.orders' or just 'orders')",
				},
			},
			Required: []string{"table_name"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			tableName, ok := input["table_name"].(string)
			if !ok {
				return &ToolResult{
					Content: "Error: table_name must be a string",
					IsError: true,
				}, fmt.Errorf("invalid table_name parameter")
			}

			// Parse schema.table if provided
			schema := "public"
			if strings.Contains(tableName, ".") {
				parts := strings.Split(tableName, ".")
				if len(parts) == 2 {
					schema = parts[0]
					tableName = parts[1]
				}
			}

			foreignKeys, err := conn.GetForeignKeys(ctx, schema, tableName)
			if err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("Error getting relationships for %s.%s: %v", schema, tableName, err),
					IsError: true,
				}, err
			}

			var result strings.Builder
			result.WriteString(fmt.Sprintf("Foreign Key Relationships for %s.%s:\n", schema, tableName))
			result.WriteString(strings.Repeat("=", 50) + "\n\n")

			if len(foreignKeys) == 0 {
				result.WriteString("No foreign key relationships found.\n")
			} else {
				result.WriteString("Outgoing References (this table -> other tables):\n")
				result.WriteString("------------------------------------------------\n")
				for _, fk := range foreignKeys {
					result.WriteString(fmt.Sprintf("- %s.%s -> %s.%s.%s\n",
						fk.TableName, fk.ColumnName,
						fk.ForeignTableSchema, fk.ForeignTableName, fk.ForeignColumnName))
				}
			}

			return &ToolResult{
				Content: result.String(),
			}, nil
		},
	}
}

// createSearchColumnsTool creates a tool to search for columns matching a pattern
func createSearchColumnsTool(conn *db.Connection) *Tool {
	return &Tool{
		Name:        "search_columns",
		Description: "Searches for columns matching a pattern across all tables in the database",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Pattern to search for in column names (case-insensitive, supports partial matches)",
				},
			},
			Required: []string{"pattern"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			pattern, ok := input["pattern"].(string)
			if !ok {
				return &ToolResult{
					Content: "Error: pattern must be a string",
					IsError: true,
				}, fmt.Errorf("invalid pattern parameter")
			}

			columns, err := conn.SearchColumns(ctx, pattern)
			if err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("Error searching columns: %v", err),
					IsError: true,
				}, err
			}

			var result strings.Builder
			result.WriteString(fmt.Sprintf("Columns matching pattern '%s':\n", pattern))
			result.WriteString(strings.Repeat("=", 30+len(pattern)) + "\n\n")

			if len(columns) == 0 {
				result.WriteString("No columns found matching the pattern.\n")
			} else {
				result.WriteString(fmt.Sprintf("%-30s %-20s %-15s %s\n", "Table", "Column", "Data Type", "Nullable"))
				result.WriteString(strings.Repeat("-", 80) + "\n")

				for _, col := range columns {
					nullable := "YES"
					if !col.IsNullable {
						nullable = "NO"
					}

					// Extract table name from description
					tableName := "unknown"
					if strings.HasPrefix(col.Description, "Found in table: ") {
						tableName = strings.TrimPrefix(col.Description, "Found in table: ")
					}

					result.WriteString(fmt.Sprintf("%-30s %-20s %-15s %s\n",
						tableName, col.Name, col.DataType, nullable))
				}
			}

			return &ToolResult{
				Content: result.String(),
			}, nil
		},
	}
}

// MarshalToolsToJSON converts tools to JSON for debugging/logging
func MarshalToolsToJSON(tools []*Tool) (string, error) {
	// Create a simplified version for JSON marshaling
	type ToolForJSON struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		InputSchema ToolSchema `json:"input_schema"`
	}

	jsonTools := make([]ToolForJSON, len(tools))
	for i, tool := range tools {
		jsonTools[i] = ToolForJSON{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	data, err := json.MarshalIndent(jsonTools, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// createExecuteSQLTool creates a tool for executing SQL queries with user approval
func createExecuteSQLTool(conn *db.Connection, getUserApproval func(string) bool) *Tool {
	return &Tool{
		Name:        "execute_sql",
		Description: "Execute a SQL query after getting user approval. Use this when you have generated a SQL query that answers the user's question.",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sql": map[string]interface{}{
					"type":        "string",
					"description": "The SQL query to execute",
				},
				"explanation": map[string]interface{}{
					"type":        "string", 
					"description": "Brief explanation of what this query does",
				},
			},
			Required: []string{"sql", "explanation"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			sqlQuery, ok := input["sql"].(string)
			if !ok {
				return &ToolResult{
					Content: "Error: sql parameter must be a string",
					IsError: true,
				}, fmt.Errorf("invalid sql parameter")
			}

			explanation, ok := input["explanation"].(string)
			if !ok {
				explanation = "SQL query execution"
			}

			// Present SQL to user for approval
			approved := getUserApproval(fmt.Sprintf("%s\n\nSQL Query:\n%s", explanation, sqlQuery))
			
			if !approved {
				return &ToolResult{
					Content: "User rejected the query execution",
					IsError: false,
				}, nil
			}

			// Execute the approved query
			result, err := executeApprovedSQL(ctx, conn, sqlQuery)
			if err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("Query execution failed: %s", err.Error()),
					IsError: true,
				}, nil
			}

			return &ToolResult{
				Content: result,
				IsError: false,
			}, nil
		},
	}
}

// executeApprovedSQL executes SQL and returns execution metadata (not actual data)
func executeApprovedSQL(ctx context.Context, conn *db.Connection, sqlQuery string) (string, error) {
	// Determine query type for appropriate execution method
	trimmedSQL := strings.TrimSpace(strings.ToUpper(sqlQuery))
	
	if strings.HasPrefix(trimmedSQL, "SELECT") || strings.HasPrefix(trimmedSQL, "WITH") {
		// Handle SELECT queries
		return executeSelectQuery(ctx, conn, sqlQuery)
	} else {
		// Handle INSERT/UPDATE/DELETE queries
		return executeModifyQuery(ctx, conn, sqlQuery)
	}
}

// executeSelectQuery executes a SELECT query and displays results to user
func executeSelectQuery(ctx context.Context, conn *db.Connection, sqlQuery string) (string, error) {
	rows, err := conn.Query(ctx, sqlQuery)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column descriptions
	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columnNames[i] = string(fd.Name)
	}

	// Display results to user (simple table format)
	fmt.Println("\n📊 Query Results:")
	fmt.Println(strings.Repeat("=", 50))
	
	// Print header
	fmt.Printf("| ")
	for _, name := range columnNames {
		fmt.Printf("%-15s | ", truncateString(name, 15))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", len(columnNames)*18))

	// Print rows and count them
	rowCount := 0
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return "", fmt.Errorf("failed to read row: %w", err)
		}

		fmt.Printf("| ")
		for _, value := range values {
			fmt.Printf("%-15s | ", truncateString(formatValue(value), 15))
		}
		fmt.Println()
		rowCount++

		// Limit display to prevent overwhelming output
		if rowCount >= 100 {
			fmt.Printf("... (showing first 100 rows)\n")
			break
		}
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error reading results: %w", err)
	}

	fmt.Printf("\n✅ Query executed successfully (%d rows)\n\n", rowCount)
	
	// Return metadata for LLM (no actual data)
	return fmt.Sprintf("Query executed successfully, %d rows returned", rowCount), nil
}

// executeModifyQuery executes INSERT/UPDATE/DELETE and returns affected rows
func executeModifyQuery(ctx context.Context, conn *db.Connection, sqlQuery string) (string, error) {
	result, err := conn.Pool().Exec(ctx, sqlQuery)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	rowsAffected := result.RowsAffected()
	fmt.Printf("\n✅ Query executed successfully (%d rows affected)\n\n", rowsAffected)
	
	return fmt.Sprintf("Query executed successfully, %d rows affected", rowsAffected), nil
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatValue(value interface{}) string {
	if value == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", value)
}

// createExplainQueryTool creates a tool for analyzing query execution plans
func createExplainQueryTool(conn *db.Connection, getUserApproval func(string) bool) *Tool {
	return &Tool{
		Name:        "explain_query",
		Description: "Analyze a SQL query's execution plan using EXPLAIN (without actually executing the query). This helps understand query performance and optimization opportunities.",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sql": map[string]interface{}{
					"type":        "string",
					"description": "The SQL query to analyze (SELECT, INSERT, UPDATE, DELETE, etc.)",
				},
				"explanation": map[string]interface{}{
					"type":        "string",
					"description": "Brief explanation of why you want to analyze this query",
				},
			},
			Required: []string{"sql", "explanation"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			sqlQuery, ok := input["sql"].(string)
			if !ok {
				return &ToolResult{
					Content: "Error: sql parameter must be a string",
					IsError: true,
				}, fmt.Errorf("invalid sql parameter")
			}

			explanation, ok := input["explanation"].(string)
			if !ok {
				explanation = "Query analysis"
			}

			// Present query to user for approval
			approved := getUserApproval(fmt.Sprintf("%s\n\nQuery to analyze:\n%s\n\nNote: This will run EXPLAIN (not ANALYZE) - no data will be modified", explanation, sqlQuery))

			if !approved {
				return &ToolResult{
					Content: "User declined query analysis",
					IsError: false,
				}, nil
			}

			// Execute EXPLAIN on the query
			explainSQL := "EXPLAIN " + sqlQuery
			result, err := executeExplainQuery(ctx, conn, explainSQL, sqlQuery)
			if err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("EXPLAIN query failed: %s", err.Error()),
					IsError: true,
				}, nil
			}

			return &ToolResult{
				Content: result,
				IsError: false,
			}, nil
		},
	}
}

// executeExplainQuery executes EXPLAIN query and returns formatted results for LLM
func executeExplainQuery(ctx context.Context, conn *db.Connection, explainSQL, originalSQL string) (string, error) {
	rows, err := conn.Query(ctx, explainSQL)
	if err != nil {
		return "", fmt.Errorf("EXPLAIN query failed: %w", err)
	}
	defer rows.Close()

	var result strings.Builder
	result.WriteString("Query Execution Plan Analysis:\n")
	result.WriteString("============================\n\n")
	result.WriteString(fmt.Sprintf("Original Query:\n%s\n\n", originalSQL))
	result.WriteString("Execution Plan:\n")
	result.WriteString("---------------\n")

	// Read all plan rows
	planLines := []string{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return "", fmt.Errorf("failed to read EXPLAIN result: %w", err)
		}

		// EXPLAIN returns a single column with the plan text
		if len(values) > 0 {
			planLine := formatValue(values[0])
			planLines = append(planLines, planLine)
		}
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error reading EXPLAIN results: %w", err)
	}

	// Format the plan for LLM consumption
	for _, line := range planLines {
		result.WriteString(line + "\n")
	}

	result.WriteString("\nThis execution plan shows:\n")
	result.WriteString("- The operations PostgreSQL will perform\n")
	result.WriteString("- The order of operations (read from innermost to outermost)\n")
	result.WriteString("- Cost estimates for each operation\n")
	result.WriteString("- Join methods and access patterns\n")
	result.WriteString("\nUse this information to suggest optimizations like:\n")
	result.WriteString("- Adding indexes on frequently filtered columns\n")
	result.WriteString("- Rewriting queries for better performance\n")
	result.WriteString("- Identifying expensive operations\n")

	// Display to user
	fmt.Println("\n📊 Query Execution Plan:")
	fmt.Println(strings.Repeat("=", 50))
	for _, line := range planLines {
		fmt.Println(line)
	}
	fmt.Printf("\n✅ EXPLAIN completed (%d plan lines)\n\n", len(planLines))

	return result.String(), nil
}