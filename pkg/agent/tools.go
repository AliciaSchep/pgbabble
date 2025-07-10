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