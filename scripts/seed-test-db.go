package main

import (
	"context"
	"fmt"
	"log"

	"github.com/AliciaSchep/pgbabble/internal/testutil"
	"github.com/AliciaSchep/pgbabble/pkg/db"
)

func main() {
	fmt.Println("🌱 Seeding test database...")

	// Get database configuration from environment variables
	cfg := testutil.GetRealDatabaseConfig()
	if cfg == nil {
		log.Fatal("❌ No test database configuration found. Please set PGBABBLE_TEST_* environment variables.")
	}

	fmt.Printf("📡 Connecting to test database: %s@%s:%d/%s\n", cfg.User, cfg.Host, cfg.Port, cfg.Database)

	// Connect to the database
	ctx := context.Background()
	conn, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to test database: %v", err)
	}
	defer conn.Close()

	// Setup test schema
	fmt.Println("🔧 Setting up test schema and seed data...")
	if err := testutil.SetupTestSchema(ctx, func(ctx context.Context, sql string) error {
		return conn.Exec(ctx, sql)
	}); err != nil {
		log.Fatalf("❌ Failed to setup test schema: %v", err)
	}

	fmt.Println("✅ Test database seeded successfully!")
	fmt.Println("🧪 Ready for test execution")
}