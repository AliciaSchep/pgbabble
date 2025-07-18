package chat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AliciaSchep/pgbabble/pkg/agent"
	"github.com/AliciaSchep/pgbabble/pkg/db"
	"github.com/AliciaSchep/pgbabble/pkg/display"
	"github.com/AliciaSchep/pgbabble/pkg/errors"
	"github.com/chzyer/readline"
)

// Session represents an interactive chat session
type Session struct {
	conn       db.Connection
	mode       string
	model      string
	rl         *readline.Instance
	agent      *agent.Agent
	agentReady bool
}

// NewSession creates a new chat session
func NewSession(conn db.Connection, mode string, model string) *Session {
	return &Session{
		conn:       conn,
		mode:       mode,
		model:      model,
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
	defer func() {
		if err := rl.Close(); err != nil {
			errors.ConnectionWarning("failed to close readline: %v", err)
		}
	}()

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
				errors.UserError("%v", err)
			}
			continue
		}

		// Handle natural language queries
		if err := s.handleQuery(ctx, line); err != nil {
			errors.UserError("%v", err)
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
		fmt.Printf("Current mode: %s\n", s.mode)
		switch s.mode {
		case "default":
			fmt.Println("Default mode: EXPLAIN sharing allowed, table size info shared, query row counts shared, but no actual query result data")
		case "schema-only":
			fmt.Println("Schema-only mode: No EXPLAIN sharing, no table size info, no query result data - maximum privacy")
		case "share-results":
			fmt.Println("Share-results mode: Full data sharing including EXPLAIN results, table sizes, and actual query result data")
		}
		fmt.Println()
		fmt.Println("💡 To change modes, exit pgbabble and restart with the --mode flag:")
		fmt.Println("   pgbabble --mode schema-only <connection>")
		fmt.Println("   pgbabble --mode share-results <connection>")
		return nil

	case "/clear", "/c":
		if s.agentReady {
			s.agent.ClearConversation()
			fmt.Println("🧹 Conversation history cleared")
		} else {
			fmt.Println("ℹ️  No conversation to clear")
		}
		return nil

	case "/browse", "/b":
		return s.browseLastResults(ctx)

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
		errors.APIError("AI service", err)
		return nil
	}

	// Display the response
	fmt.Println("🤖 AI Response:")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(response)
	fmt.Println()

	return nil
}

// initializeAgent sets up the LLM agent with schema tools
func (s *Session) initializeAgent() {
	agentClient, err := agent.NewAgent("", s.mode, s.model)
	if err != nil {
		errors.UserInfo("LLM features not available: %v", err)
		fmt.Println("   Set ANTHROPIC_API_KEY environment variable to enable AI features")
		fmt.Println()
		return
	}

	// Add schema inspection tools
	schemaTools := agent.CreateSchemaTools(s.conn, s.mode)
	for _, tool := range schemaTools {
		toolDef := agent.ConvertToolToDefinition(tool)
		agentClient.AddTool(toolDef)
	}

	// Add SQL execution tools with user approval callback
	executionTools := agent.CreateExecutionTools(s.conn, s.getUserApproval, s.mode)
	for _, tool := range executionTools {
		toolDef := agent.ConvertToolToDefinition(tool)
		agentClient.AddTool(toolDef)
	}

	s.agent = agentClient
	s.agentReady = true
	fmt.Println("✅ AI assistant ready with database schema tools!")
	fmt.Println("💡 I can inspect your database structure and generate custom SQL queries")
	fmt.Println()
}

// getUserApproval prompts the user to approve a SQL query execution
func (s *Session) getUserApproval(queryInfo string) bool {
	fmt.Println("\n🔍 SQL Query Ready for Execution:")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(queryInfo)
	fmt.Println(strings.Repeat("=", 50))

	// Change the readline prompt temporarily for this question
	s.rl.SetPrompt("Execute this query? (y/yes/n/no): ")

	response, err := s.rl.Readline()
	if err != nil {
		errors.UserError("error reading input: %v", err)
		// Reset prompt back to normal
		s.rl.SetPrompt("pgbabble> ")
		return false
	}

	// Reset prompt back to normal
	s.rl.SetPrompt("pgbabble> ")

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// showHelp displays available commands
func (s *Session) showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  /help, /h          Show this help message")
	fmt.Println("  /quit, /exit, /q   Exit pgbabble")
	fmt.Println("  /schema, /s        Show database schema overview")
	fmt.Println("  /tables, /t        List all tables and views")
	fmt.Println("  /describe <table>  Describe a specific table")
	fmt.Println("  /mode, /m          Show current data exposure mode")
	fmt.Println("  /clear, /c         Clear conversation history")
	fmt.Println("  /browse, /b        Browse last query results in less pager")
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

// browseLastResults opens the last query results in less for browsing
func (s *Session) browseLastResults(ctx context.Context) error {
	if agent.LastQueryResult == nil {
		fmt.Println("❌ No query results available to browse")
		fmt.Println("💡 Run a query first, then use /browse to view all results")
		return nil
	}

	if len(agent.LastQueryResult.AllRows) == 0 {
		fmt.Println("❌ Last query returned no results to browse")
		return nil
	}

	// Check if less is available
	if !display.CheckLessAvailable() {
		fmt.Println("❌ 'less' command not found on this system")
		fmt.Println("💡 All available results are already shown above")
		return nil
	}

	// Generate the full table content
	title := fmt.Sprintf("Query Results (%d rows)", len(agent.LastQueryResult.AllRows))
	content := display.GenerateFullTableContent(
		agent.LastQueryResult.ColumnNames,
		agent.LastQueryResult.AllRows,
		title,
	)

	// Add query info to the top
	fullContent := fmt.Sprintf("Query: %s\n\n%s", agent.LastQueryResult.QueryText, content)

	// Open in less
	return display.PageWithContext(ctx, title, fullContent)
}
