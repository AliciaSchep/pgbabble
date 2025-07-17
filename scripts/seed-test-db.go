package main

import (
	"context"
	"fmt"
	"log"
	"time"

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

	// Connect to the database with retries
	ctx := context.Background()
	var conn *db.ConnectionImpl
	var err error
	
	for i := 0; i < 5; i++ {
		conn, err = db.Connect(ctx, cfg)
		if err == nil {
			break
		}
		fmt.Printf("⏳ Connection attempt %d failed, retrying in 2 seconds...\n", i+1)
		time.Sleep(2 * time.Second)
	}
	
	if err != nil {
		log.Fatalf("❌ Failed to connect to test database after 5 attempts: %v", err)
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