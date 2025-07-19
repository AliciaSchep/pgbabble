package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AliciaSchep/pgbabble/pkg/agent"
	"github.com/AliciaSchep/pgbabble/pkg/chat"
	"github.com/AliciaSchep/pgbabble/pkg/config"
	"github.com/AliciaSchep/pgbabble/pkg/db"
	"github.com/spf13/cobra"
)

func getVersionString() string {
	if commit != "unknown" && buildDate != "unknown" {
		return fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate)
	}
	return version
}

var (
	// Version information (set at build time)
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"

	// Database connection flags
	host     string
	port     int
	user     string
	password string
	database string

	// Application flags
	mode  string
	model string
)

var rootCmd = &cobra.Command{
	Use:   "pgbabble [postgresql://uri] or [flags]",
	Short: "Interactive PostgreSQL CLI powered by LLM",
	Long: `PGBabble is a CLI tool for interacting with PostgreSQL databases using natural language.
It converts your questions into SQL queries and executes them safely.

Examples:
  pgbabble "postgresql://user:pass@localhost/mydb"
  pgbabble --host localhost --user myuser --dbname mydb
  pgbabble --model claude-sonnet-4-0 "postgresql://user:pass@localhost/mydb"
  pgbabble --model claude-3-5-haiku-latest --host localhost --dbname mydb
  export PGHOST=localhost PGUSER=myuser PGDATABASE=mydb && pgbabble`,
	Args:    cobra.MaximumNArgs(1),
	Version: getVersionString(),
	RunE:    runPGBabble,
}

func init() {
	// Database connection flags
	rootCmd.Flags().StringVar(&host, "host", "", "Database host (default: localhost, or PGHOST)")
	rootCmd.Flags().IntVar(&port, "port", 0, "Database port (default: 5432, or PGPORT)")
	rootCmd.Flags().StringVar(&user, "user", "", "Database user (default: current user, or PGUSER)")
	rootCmd.Flags().StringVar(&password, "password", "", "Database password (or PGPASSWORD)")
	rootCmd.Flags().StringVar(&database, "dbname", "", "Database name (required, or PGDATABASE)")

	// Application flags
	rootCmd.Flags().StringVar(&mode, "mode", "default", "Data exposure mode: default, schema-only, share-results")
	rootCmd.Flags().StringVar(&model, "model", agent.DefaultModel, "Claude model to use (e.g., claude-sonnet-4-0, claude-3-5-haiku-latest)")
}

func runPGBabble(cmd *cobra.Command, args []string) error {
	// Create cancellable context that responds to interrupt signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var dbConfig *config.DBConfig
	var err error

	// Determine if we have a URI argument or should use flags
	if len(args) == 1 {
		// Parse URI
		dbConfig, err = config.NewDBConfigFromURI(args[0])
		if err != nil {
			return fmt.Errorf("failed to parse database URI: %w", err)
		}
	} else {
		// Use flags and environment variables
		dbConfig = config.NewDBConfigFromFlags(host, user, password, database, port)
	}

	// Validate configuration
	if err := dbConfig.Validate(); err != nil {
		return fmt.Errorf("invalid database configuration: %w", err)
	}

	// Validate mode
	if mode != "default" && mode != "schema-only" && mode != "share-results" {
		return fmt.Errorf("invalid mode: %s (must be: default, schema-only, share-results)", mode)
	}

	// Connect to database
	fmt.Printf("Connecting to PostgreSQL database: %s@%s:%d/%s\n",
		dbConfig.User, dbConfig.Host, dbConfig.Port, dbConfig.Database)

	conn, err := db.Connect(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	// Get database info
	dbInfo, err := conn.GetDatabaseInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database info: %w", err)
	}

	fmt.Printf("Connected successfully!\n")
	fmt.Printf("Database: %s\n", dbInfo.Database)
	fmt.Printf("User: %s\n", dbInfo.User)
	fmt.Printf("Version: %s\n", dbInfo.Version)
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("Model: %s\n", model)
	fmt.Println("Type /help for commands, /quit to exit")
	fmt.Println()

	// Start interactive chat with cancellable context
	chatSession := chat.NewSession(conn, mode, model)
	return chatSession.Start(ctx)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
