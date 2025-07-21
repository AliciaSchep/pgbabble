package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AliciaSchep/pgbabble/pkg/db"
	"github.com/AliciaSchep/pgbabble/pkg/display"
	pkgerrors "github.com/AliciaSchep/pgbabble/pkg/errors"
)

// QueryTimeout is the default timeout for SQL query execution
var QueryTimeout = 60 * time.Second

// LastQueryResult stores the most recent query result for browsing
var LastQueryResult *QueryResultWithData

// QueryResultWithData extends QueryResultData with additional metadata
type QueryResultWithData struct {
	QueryResultData
	AllRows   [][]interface{}
	QueryText string
}

// CreateSchemaTools creates all schema inspection tools for the LLM
func CreateSchemaTools(conn db.Connection, mode string) []*Tool {
	return []*Tool{
		createListTablesTool(conn, mode),
		createDescribeTableTool(conn),
		createGetRelationshipsTool(conn),
		createSearchColumnsTool(conn),
	}
}

// CreateExecutionTools creates SQL execution tools for the LLM
func CreateExecutionTools(conn db.Connection, getUserApproval func(string) bool, mode string) []*Tool {
	return []*Tool{
		createExecuteSQLTool(conn, getUserApproval, mode),
		createExplainQueryTool(conn, getUserApproval, mode),
	}
}

// createListTablesTool creates a tool to list all tables and views
func createListTablesTool(conn db.Connection, mode string) *Tool {
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
					if mode == "default" || mode == "share-results" {
						// Include estimated table size information for default and share-results modes
						if table.EstimatedRows <= 0 {
							result.WriteString(fmt.Sprintf("- %s (%s) - empty or no stats\n", table.Name, table.Type))
						} else {
							result.WriteString(fmt.Sprintf("- %s (%s) - ~%d rows (estimated)\n", table.Name, table.Type, table.EstimatedRows))
						}
					} else {
						// Schema-only mode: no size information
						result.WriteString(fmt.Sprintf("- %s (%s)\n", table.Name, table.Type))
					}
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
func createDescribeTableTool(conn db.Connection) *Tool {
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
func createGetRelationshipsTool(conn db.Connection) *Tool {
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
func createSearchColumnsTool(conn db.Connection) *Tool {
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
func createExecuteSQLTool(conn db.Connection, getUserApproval func(string) bool, mode string) *Tool {
	return &Tool{
		Name:        "execute_sql",
		Description: "Execute a SQL query after getting user approval. Use this when you have generated a SQL query that answers the user's question. IMPORTANT: If the user rejects the query, do NOT immediately offer another SQL query. Instead, ask the user what they want changed or modified about the query approach.",
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
					Content: "User rejected the query execution. Do NOT immediately offer another SQL query. Instead, ask the user what they want changed, modified, or what approach they prefer. Find out what was wrong with the query or what they wanted differently.",
					IsError: false,
				}, nil
			}

			// Execute the approved query
			result, err := executeApprovedSQL(ctx, conn, sqlQuery, mode)
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
func executeApprovedSQL(ctx context.Context, conn db.Connection, sqlQuery string, mode string) (string, error) {
	// Validate that query is safe to execute
	if err := validateSafeQuery(sqlQuery); err != nil {
		return "", err
	}

	// Only SELECT and WITH queries are allowed
	return executeSelectQuery(ctx, conn, sqlQuery, mode)
}

// executeSelectQuery executes a SELECT query and displays results to user
func executeSelectQuery(ctx context.Context, conn db.Connection, sqlQuery string, mode string) (string, error) {
	// Add configurable query timeout while preserving cancellation from parent context
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// Record start time for execution timing
	startTime := time.Now()

	// Start progress indicator
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime)
				fmt.Printf("\rQuery running... %v elapsed", elapsed.Round(time.Second))
				_ = os.Stdout.Sync()
			case <-queryCtx.Done():
				// Context cancelled - stop progress indicator
				return
			case <-done:
				return
			}
		}
	}()

	fmt.Print("Executing query...")
	_ = os.Stdout.Sync()

	// Ensure we have a healthy connection (use parent context, not query context)
	conn.EnsureConnection(ctx)

	rows, err := conn.Query(queryCtx, sqlQuery)

	// Stop progress indicator
	close(done)
	fmt.Print("\r") // Clear the progress line

	if err != nil {
		fmt.Print("\r") // Clear progress line

		// Check for context cancellation and provide appropriate message
		if errors.Is(err, context.Canceled) || queryCtx.Err() == context.Canceled {
			fmt.Println("⏹️  Query cancelled by user")

			// With connection pools, cancelled connections are automatically handled
			// No need for manual reconnection

			return "Query was cancelled by the user. The database connection remains active and ready for new queries. Please ask the user what they would like to do next.", nil
		}
		if queryCtx.Err() == context.DeadlineExceeded {
			pkgerrors.UserError("Query timed out after %v", QueryTimeout)
			return "", fmt.Errorf("query timed out after %v - please check if your query is optimized or try adding LIMIT clause", QueryTimeout)
		}

		// Show concise error for technical users (without LLM instructions)
		userErrorMsg := formatUserError(err)
		pkgerrors.UserError("Query failed: %s", userErrorMsg)

		// Return LLM-friendly error message with tool instructions
		llmErrorMsg := formatDatabaseError(err)
		return "", fmt.Errorf("%s", llmErrorMsg)
	}
	defer rows.Close()

	// Get column descriptions
	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columnNames[i] = string(fd.Name)
	}

	// Collect all rows first for both display and LLM data
	allRows := make([][]interface{}, 0)
	rowCount := 0

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			userErrorMsg := formatUserError(err)
			pkgerrors.UserError("Error processing query results: %s", userErrorMsg)
			llmErrorMsg := formatDatabaseError(err)
			return "", fmt.Errorf("%s", llmErrorMsg)
		}

		// Make a copy of values to avoid reference issues
		rowCopy := make([]interface{}, len(values))
		copy(rowCopy, values)
		allRows = append(allRows, rowCopy)

		rowCount++

		// Limit to prevent overwhelming memory usage
		if rowCount >= 10000 {
			fmt.Printf("⚠️  Query returned more than 10,000 rows, limiting display...\n")
			break
		}
	}

	// Display results to user with intelligent formatting
	fmt.Println("\n📊 Query Results:")
	fmt.Println(strings.Repeat("=", 50))

	if len(allRows) > 0 {
		// Create table formatter and analyze data
		formatter := display.NewTableFormatter(columnNames)
		sampleSize := min(len(allRows), 10)
		formatter.AnalyzeData(allRows[:sampleSize])
		widths := formatter.CalculateColumnWidths()

		// Print header
		fmt.Print(formatter.FormatHeader(widths))

		// Print rows (limit display to 25 for initial view)
		displayLimit := min(len(allRows), 25)
		for i := 0; i < displayLimit; i++ {
			fmt.Print(formatter.FormatRow(allRows[i], widths))
		}

		if len(allRows) > displayLimit {
			fmt.Printf("... (showing first %d of %d rows, use /browse to view all)\n", displayLimit, len(allRows))
		}
	}

	// Prepare collected data for LLM if in share-results mode
	var collectedData *QueryResultData
	if mode == "share-results" {
		llmRowLimit := min(len(allRows), 50)
		collectedData = &QueryResultData{
			ColumnNames: columnNames,
			Rows:        allRows[:llmRowLimit],
			TotalRows:   rowCount,
			Truncated:   len(allRows) > 50,
		}
	}

	if err := rows.Err(); err != nil {
		userErrorMsg := formatUserError(err)
		pkgerrors.UserError("Error during query result iteration: %s", userErrorMsg)
		llmErrorMsg := formatDatabaseError(err)
		return "", fmt.Errorf("%s", llmErrorMsg)
	}

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Store results for browse functionality
	LastQueryResult = &QueryResultWithData{
		QueryResultData: QueryResultData{
			ColumnNames: columnNames,
			Rows:        allRows,
			TotalRows:   rowCount,
			Truncated:   false,
		},
		AllRows:   allRows,
		QueryText: sqlQuery,
	}

	fmt.Printf("\n✅ Query executed successfully (%d rows in %v)\n\n", rowCount, executionTime)

	// Format result for LLM based on mode using collected data
	return formatQueryResult(mode, rowCount, executionTime, collectedData), nil
}

// QueryResultData represents the actual data from a query for LLM sharing
type QueryResultData struct {
	ColumnNames []string
	Rows        [][]interface{}
	TotalRows   int
	Truncated   bool
}

// formatQueryResult formats the query execution result based on the mode
func formatQueryResult(mode string, rowCount int, executionTime time.Duration, data *QueryResultData) string {
	nextStep := "Next step: Summarise these results for the user and ask or propose next steps. Do not run execute_sql tool immediately without asking the user what they want to do next."
	switch mode {
	case "share-results":
		var result strings.Builder
		result.WriteString(fmt.Sprintf("Query executed successfully, %d rows returned in %v\n\n", rowCount, executionTime))

		result.WriteString("Query Results:\n")
		result.WriteString(strings.Repeat("=", 50) + "\n")

		// Add column headers
		result.WriteString("| ")
		for _, name := range data.ColumnNames {
			result.WriteString(fmt.Sprintf("%-15s | ", truncateString(name, 15)))
		}
		result.WriteString("\n")
		result.WriteString(strings.Repeat("-", len(data.ColumnNames)*18) + "\n")

		// Add data rows
		for _, row := range data.Rows {
			result.WriteString("| ")
			for _, value := range row {
				result.WriteString(fmt.Sprintf("%-15s | ", truncateString(formatValue(value), 15)))
			}
			result.WriteString("\n")
		}

		if data.Truncated {
			result.WriteString("... (showing first 50 rows for analysis)\n")
		}

		result.WriteString(nextStep)

		return result.String()

	case "schema-only":
		return fmt.Sprintf("Query executed successfully and results were displayed to the user. %s", nextStep)

	default: // "default" mode
		return fmt.Sprintf("Query executed successfully, %d rows returned in %v. Results were displayed to the user. %s", rowCount, executionTime, nextStep)
	}
}

// validateSafeQuery ensures the query is safe to execute (SELECT/WITH only)
func validateSafeQuery(sqlQuery string) error {
	// Clean and normalize the query
	cleaned := strings.TrimSpace(strings.ToUpper(sqlQuery))

	// Remove comments
	lines := strings.Split(cleaned, "\n")
	var cleanedLines []string
	for _, line := range lines {
		// Remove line comments
		if commentPos := strings.Index(line, "--"); commentPos != -1 {
			line = line[:commentPos]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}
	cleaned = strings.Join(cleanedLines, " ")

	// Remove block comments (/* ... */)
	for {
		start := strings.Index(cleaned, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(cleaned[start:], "*/")
		if end == -1 {
			return fmt.Errorf("unclosed block comment in query")
		}
		cleaned = cleaned[:start] + " " + cleaned[start+end+2:]
	}

	cleaned = strings.TrimSpace(cleaned)

	// Check for allowed query types
	allowedPrefixes := []string{"SELECT", "WITH"}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleaned, prefix) {
			// Additional validation for potentially dangerous functions/keywords
			if err := validateQueryContent(cleaned); err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("only SELECT and WITH queries are allowed for security reasons")
}

// validateQueryContent checks for dangerous functions or patterns within allowed queries
func validateQueryContent(query string) error {
	// List of dangerous functions/patterns to block
	dangerousPatterns := []string{
		"PG_SLEEP",
		"PG_TERMINATE_BACKEND",
		"PG_CANCEL_BACKEND",
		"COPY",
		"\\COPY",
		"DBLINK",
		"DBLINK_EXEC",
		"PERFORM",
		"DO $$",
		"DO $",
		"CREATE",
		"DROP",
		"ALTER",
		"TRUNCATE",
		"DELETE",
		"INSERT",
		"UPDATE",
		"GRANT",
		"REVOKE",
		"SET ROLE",
		"SET SESSION",
		"RESET",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(query, pattern) {
			return fmt.Errorf("query contains potentially dangerous operation: %s", pattern)
		}
	}

	return nil
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

// formatDatabaseError provides LLM-friendly error messages with tool instructions
func formatDatabaseError(err error) string {
	errStr := err.Error()

	// Handle common PostgreSQL error patterns
	if strings.Contains(errStr, "relation") && strings.Contains(errStr, "does not exist") {
		return fmt.Sprintf("Table or view not found: %v. Use the list_tables tool to see available tables.", err)
	}

	if strings.Contains(errStr, "column") && strings.Contains(errStr, "does not exist") {
		return fmt.Sprintf("Column not found: %v. Use the describe_table tool to see available columns.", err)
	}

	if strings.Contains(errStr, "syntax error") {
		return fmt.Sprintf("SQL syntax error: %v. Please check your query syntax.", err)
	}

	if strings.Contains(errStr, "permission denied") {
		return fmt.Sprintf("Permission denied: %v. You may not have access to this table or operation.", err)
	}

	if strings.Contains(errStr, "connection") && (strings.Contains(errStr, "refused") || strings.Contains(errStr, "closed")) {
		return fmt.Sprintf("Database connection issue: %v. The connection may have been lost. Attempting to reconnect...", err)
	}

	if strings.Contains(errStr, "dial") || strings.Contains(errStr, "network") || strings.Contains(errStr, "timeout") {
		return fmt.Sprintf("Network connectivity issue: %v. The database server may be unreachable.", err)
	}

	// Return original error if no specific pattern matches
	return fmt.Sprintf("Database error: %v", err)
}

// formatUserError provides user-friendly error messages without LLM tool instructions
func formatUserError(err error) string {
	errStr := err.Error()

	// Handle common PostgreSQL error patterns
	if strings.Contains(errStr, "relation") && strings.Contains(errStr, "does not exist") {
		return fmt.Sprintf("Table or view not found\n%v", err)
	}

	if strings.Contains(errStr, "column") && strings.Contains(errStr, "does not exist") {
		return fmt.Sprintf("Column not found\n%v", err)
	}

	if strings.Contains(errStr, "syntax error") {
		return fmt.Sprintf("SQL syntax error\n%v", err)
	}

	if strings.Contains(errStr, "permission denied") {
		return fmt.Sprintf("Permission denied\n%v", err)
	}

	if strings.Contains(errStr, "connection") && (strings.Contains(errStr, "refused") || strings.Contains(errStr, "closed")) {
		return fmt.Sprintf("Database connection issue\n%v", err)
	}

	if strings.Contains(errStr, "dial") || strings.Contains(errStr, "network") || strings.Contains(errStr, "timeout") {
		return fmt.Sprintf("Network connectivity issue\n%v", err)
	}

	// Return original error if no specific pattern matches
	return fmt.Sprintf("Database error\n%v", err)
}

// createExplainQueryTool creates a tool for analyzing query execution plans
func createExplainQueryTool(conn db.Connection, getUserApproval func(string) bool, mode string) *Tool {
	return &Tool{
		Name:        "explain_query",
		Description: "Analyze a SQL query's execution plan using EXPLAIN (without actually executing the query). This helps understand query performance and optimization opportunities. IMPORTANT: If the user declines analysis, ask what they want changed or if they prefer a different approach.",
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
					Content: "User declined query analysis. Ask the user what they want changed about the query or if they prefer a different approach to analyze their data.",
					IsError: false,
				}, nil
			}

			// Validate that query is safe to analyze
			if err := validateSafeQuery(sqlQuery); err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("Query validation failed: %s", err.Error()),
					IsError: true,
				}, nil
			}

			// Always execute EXPLAIN to show results to user
			explainSQL := "EXPLAIN " + sqlQuery
			result, err := executeExplainQuery(ctx, conn, explainSQL, sqlQuery)
			if err != nil {
				return &ToolResult{
					Content: fmt.Sprintf("EXPLAIN query failed: %s", err.Error()),
					IsError: true,
				}, nil
			}

			// Conditionally share with LLM based on mode
			if mode == "schema-only" {
				return &ToolResult{
					Content: "EXPLAIN analysis was displayed to the user. Query structure appears well-formed, but execution plan details are not shared in schema-only mode for privacy.",
					IsError: false,
				}, nil
			}

			// For default and share-results modes, share full EXPLAIN with LLM
			return &ToolResult{
				Content: result,
				IsError: false,
			}, nil
		},
	}
}

// executeExplainQuery executes EXPLAIN query and returns formatted results for LLM
func executeExplainQuery(ctx context.Context, conn db.Connection, explainSQL, originalSQL string) (string, error) {
	// Add timeout for EXPLAIN queries while preserving cancellation from parent context
	queryCtx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// Record start time
	startTime := time.Now()

	// Ensure we have a healthy connection (use parent context, not query context)
	conn.EnsureConnection(ctx)

	rows, err := conn.Query(queryCtx, explainSQL)
	if err != nil {
		// Check for context cancellation and provide appropriate message
		if errors.Is(err, context.Canceled) || queryCtx.Err() == context.Canceled {
			fmt.Println("⏹️  EXPLAIN query cancelled by user")

			// With connection pools, cancelled connections are automatically handled
			// No need for manual reconnection

			return "EXPLAIN query was cancelled by the user. The database connection remains active and ready for new queries.", nil
		}
		if queryCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("EXPLAIN query timed out after %v", QueryTimeout)
		}
		return "", fmt.Errorf("%s", formatDatabaseError(err))
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
			return "", fmt.Errorf("%s", formatDatabaseError(err))
		}

		// EXPLAIN returns a single column with the plan text
		if len(values) > 0 {
			planLine := formatValue(values[0])
			planLines = append(planLines, planLine)
		}
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("%s", formatDatabaseError(err))
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

	// Calculate execution time
	executionTime := time.Since(startTime)

	// Display to user
	fmt.Println("\n📊 Query Execution Plan:")
	fmt.Println(strings.Repeat("=", 50))
	for _, line := range planLines {
		fmt.Println(line)
	}
	fmt.Printf("\n✅ EXPLAIN completed (%d plan lines in %v)\n\n", len(planLines), executionTime)

	return result.String(), nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
