package chat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"pgbabble/pkg/agent"
	"pgbabble/pkg/db"
)

// Session represents an interactive chat session
type Session struct {
	conn       *db.Connection
	mode       string
	rl         *readline.Instance
	agent      *agent.Agent
	agentReady bool
}

// NewSession creates a new chat session
func NewSession(conn *db.Connection, mode string) *Session {
	return &Session{
		conn:       conn,
		mode:       mode,
		agentReady: false,
	}
}

// Start begins the interactive chat session
func (s *Session) Start(ctx context.Context) error {
	// Configure readline
	rl, err := readline.NewEx(&readline.Config{
		Prompt:      "pgbabble> ",
		HistoryFile: os.ExpandEnv("$HOME/.pgbabble_history"),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize readline: %w", err)
	}
	defer rl.Close()
	
	s.rl = rl
	
	// Initialize agent if API key is available
	s.initializeAgent()
	
	// Main chat loop
	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					break
				} else {
					continue
				}
			}
			break
		}
		
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Handle commands
		if strings.HasPrefix(line, "/") {
			if err := s.handleCommand(ctx, line); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			continue
		}
		
		// Handle natural language queries
		if err := s.handleQuery(ctx, line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
	
	fmt.Println("Goodbye!")
	return nil
}

// handleCommand processes slash commands
func (s *Session) handleCommand(ctx context.Context, cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}
	
	switch parts[0] {
	case "/quit", "/exit", "/q":
		os.Exit(0)
		
	case "/help", "/h":
		s.showHelp()
		
	case "/schema", "/s":
		return s.showSchema(ctx)
		
	case "/tables", "/t":
		return s.listTables(ctx)
		
	case "/describe", "/d":
		if len(parts) < 2 {
			return fmt.Errorf("usage: /describe <table_name>")
		}
		return s.describeTable(ctx, parts[1])
		
	case "/mode", "/m":
		if len(parts) < 2 {
			fmt.Printf("Current mode: %s\n", s.mode)
			fmt.Println("Available modes: default, summary_data, full_data")
			return nil
		}
		return s.setMode(parts[1])
		
	default:
		return fmt.Errorf("unknown command: %s (type /help for available commands)", parts[0])
	}
	
	return nil
}

// handleQuery processes natural language queries using the LLM agent
func (s *Session) handleQuery(ctx context.Context, query string) error {
	if !s.agentReady {
		fmt.Println("❌ LLM agent not available")
		fmt.Println("💡 To enable AI features, set your ANTHROPIC_API_KEY environment variable")
		fmt.Println("   You can get an API key from https://console.anthropic.com/")
		fmt.Println()
		fmt.Println("🔍 In the meantime, you can explore the database using:")
		fmt.Println("   /schema   - View database schema")
		fmt.Println("   /tables   - List all tables")
		fmt.Println("   /describe <table> - Describe a specific table")
		return nil
	}

	fmt.Printf("🤔 Processing: %s\n", query)
	fmt.Println()

	// Send query to LLM agent
	response, err := s.agent.SendMessage(ctx, query)
	if err != nil {
		fmt.Printf("❌ Error getting response from AI: %v\n", err)
		return nil
	}

	// Display the response
	fmt.Println("🤖 AI Response:")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(response)
	fmt.Println()

	// If the response contains SQL, offer to execute it
	if s.containsSQL(response) {
		return s.handleSQLResponse(ctx, response)
	}

	return nil
}

// initializeAgent sets up the LLM agent with schema tools
func (s *Session) initializeAgent() {
	agentClient, err := agent.NewAgent("")
	if err != nil {
		fmt.Printf("ℹ️  LLM features not available: %v\n", err)
		fmt.Println("   Set ANTHROPIC_API_KEY environment variable to enable AI features")
		fmt.Println()
		return
	}

	// Add schema inspection tools
	tools := agent.CreateSchemaTools(s.conn)
	for _, tool := range tools {
		toolDef := agent.ConvertToolToDefinition(tool)
		agentClient.AddTool(toolDef)
	}

	s.agent = agentClient
	s.agentReady = true
	fmt.Println("✅ AI assistant ready with database schema tools!")
	fmt.Println("💡 I can inspect your database structure and generate custom SQL queries")
	fmt.Println()
}

// containsSQL checks if the response contains SQL code
func (s *Session) containsSQL(content string) bool {
	content = strings.ToLower(content)
	sqlKeywords := []string{"select", "insert", "update", "delete", "create", "alter", "drop"}
	
	for _, keyword := range sqlKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// handleSQLResponse handles responses that contain SQL code
func (s *Session) handleSQLResponse(ctx context.Context, content string) error {
	// Extract SQL from the response (simple implementation)
	lines := strings.Split(content, "\n")
	var sqlLines []string
	var inSQLBlock bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Detect SQL code blocks
		if strings.HasPrefix(line, "```sql") || strings.HasPrefix(line, "```") {
			inSQLBlock = !inSQLBlock
			continue
		}
		
		if inSQLBlock {
			sqlLines = append(sqlLines, line)
		} else if line != "" && (strings.HasPrefix(strings.ToLower(line), "select") ||
			strings.HasPrefix(strings.ToLower(line), "with") ||
			strings.HasPrefix(strings.ToLower(line), "insert") ||
			strings.HasPrefix(strings.ToLower(line), "update") ||
			strings.HasPrefix(strings.ToLower(line), "delete")) {
			// Line looks like SQL
			sqlLines = append(sqlLines, line)
		}
	}
	
	if len(sqlLines) == 0 {
		return nil
	}
	
	sql := strings.Join(sqlLines, "\n")
	if sql == "" {
		return nil
	}
	
	fmt.Println("🔍 Generated SQL:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(sql)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()
	
	// Ask for approval
	fmt.Print("Execute this query? (y/n/e=edit): ")
	
	// Read user input
	response, err := s.rl.Readline()
	if err != nil {
		return err
	}
	
	response = strings.ToLower(strings.TrimSpace(response))
	
	switch response {
	case "y", "yes":
		return s.executeSQL(ctx, sql)
	case "e", "edit":
		fmt.Println("💡 SQL editing not implemented yet - you can copy/paste the SQL to run manually")
		return nil
	case "n", "no":
		fmt.Println("❌ Query not executed")
		return nil
	default:
		fmt.Println("❌ Invalid response. Query not executed")
		return nil
	}
}

// executeSQL executes the SQL query and displays results
func (s *Session) executeSQL(ctx context.Context, sql string) error {
	fmt.Println("⚡ Executing query...")
	
	// TODO: Implement actual query execution and result display
	// For now, just show a placeholder
	fmt.Println("🚧 Query execution coming in Phase 3!")
	fmt.Printf("📝 SQL to execute:\n%s\n", sql)
	
	return nil
}

// showHelp displays available commands
func (s *Session) showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  /help, /h          Show this help message")
	fmt.Println("  /quit, /exit, /q   Exit pgbabble")
	fmt.Println("  /schema, /s        Show database schema overview")
	fmt.Println("  /tables, /t        List all tables and views")
	fmt.Println("  /describe <table>  Describe a specific table")
	fmt.Println("  /mode [mode]       Show or set data exposure mode")
	fmt.Println()
	fmt.Println("Or just type a natural language question about your data!")
}

// showSchema displays a database schema overview
func (s *Session) showSchema(ctx context.Context) error {
	tables, err := s.conn.ListTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}
	
	if len(tables) == 0 {
		fmt.Println("No tables found in the database.")
		return nil
	}
	
	fmt.Println("Database Schema Overview:")
	fmt.Println("========================")
	
	// Group by schema
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

// listTables displays all tables and views
func (s *Session) listTables(ctx context.Context) error {
	tables, err := s.conn.ListTables(ctx)
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

// describeTable shows detailed information about a table
func (s *Session) describeTable(ctx context.Context, tableName string) error {
	// Parse schema.table if provided
	schema := "public"
	if strings.Contains(tableName, ".") {
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			schema = parts[0]
			tableName = parts[1]
		}
	}
	
	table, err := s.conn.DescribeTable(ctx, schema, tableName)
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
	foreignKeys, err := s.conn.GetForeignKeys(ctx, schema, tableName)
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

// setMode changes the data exposure mode
func (s *Session) setMode(newMode string) error {
	validModes := map[string]bool{
		"default":      true,
		"summary_data": true,
		"full_data":    true,
	}
	
	if !validModes[newMode] {
		return fmt.Errorf("invalid mode: %s (valid modes: default, summary_data, full_data)", newMode)
	}
	
	oldMode := s.mode
	s.mode = newMode
	fmt.Printf("Mode changed from '%s' to '%s'\n", oldMode, newMode)
	
	switch newMode {
	case "default":
		fmt.Println("Default mode: Only schema information is sent to LLM, no query result data")
	case "summary_data":
		fmt.Println("Summary data mode: Schema + summary statistics (row counts, cardinality) sent to LLM")
	case "full_data":
		fmt.Println("Full data mode: Schema + actual query result data sent to LLM")
	}
	
	return nil
}