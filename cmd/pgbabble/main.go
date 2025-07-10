package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"pgbabble/pkg/config"
	"pgbabble/pkg/db"
	"pgbabble/pkg/chat"
)

var (
	// Database connection flags
	host     string
	port     int
	user     string
	password string
	database string
	
	// Application flags
	mode string
)

var rootCmd = &cobra.Command{
	Use:   "pgbabble [postgresql://uri] or [flags]",
	Short: "Interactive PostgreSQL CLI powered by LLM",
	Long: `PGBabble is a CLI tool for interacting with PostgreSQL databases using natural language.
It converts your questions into SQL queries and executes them safely.

Examples:
  pgbabble "postgresql://user:pass@localhost/mydb"
  pgbabble --host localhost --user myuser --dbname mydb
  export PGHOST=localhost PGUSER=myuser PGDATABASE=mydb && pgbabble`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPGBabble,
}

func init() {
	// Database connection flags
	rootCmd.Flags().StringVar(&host, "host", "", "Database host (default: localhost, or PGHOST)")
	rootCmd.Flags().IntVar(&port, "port", 0, "Database port (default: 5432, or PGPORT)")
	rootCmd.Flags().StringVar(&user, "user", "", "Database user (default: current user, or PGUSER)")
	rootCmd.Flags().StringVar(&password, "password", "", "Database password (or PGPASSWORD)")
	rootCmd.Flags().StringVar(&database, "dbname", "", "Database name (required, or PGDATABASE)")
	
	// Application flags
	rootCmd.Flags().StringVar(&mode, "mode", "default", "Data exposure mode: default, summary_data, full_data")
}

func runPGBabble(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
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
	if mode != "default" && mode != "summary_data" && mode != "full_data" {
		return fmt.Errorf("invalid mode: %s (must be: default, summary_data, full_data)", mode)
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
	fmt.Println("Type /help for commands, /quit to exit")
	fmt.Println()
	
	// Start interactive chat
	chatSession := chat.NewSession(conn, mode)
	return chatSession.Start(ctx)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}