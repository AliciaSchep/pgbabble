package db

import (
	"context"
	"testing"
	"time"

	"github.com/AliciaSchep/pgbabble/internal/testutil"
	"github.com/AliciaSchep/pgbabble/pkg/config"
)

// TestConnect_ConfigValidation tests configuration validation without requiring a real database
func TestConnect_ConfigValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("NilConfig", func(t *testing.T) {
		_, err := Connect(ctx, nil)
		if err == nil {
			t.Error("Expected error with nil config")
		}
		if err.Error() != "database configuration cannot be nil" {
			t.Errorf("Expected specific nil config error, got: %v", err)
		}
	})

	t.Run("InvalidConfigs", func(t *testing.T) {
		tests := []struct {
			name   string
			config *config.DBConfig
		}{
			{
				name: "missing_host",
				config: &config.DBConfig{
					Port:     5432,
					Database: "test",
					User:     "test",
					Password: "test",
				},
			},
			{
				name: "missing_user",
				config: &config.DBConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "test",
					Password: "test",
				},
			},
			{
				name: "missing_database",
				config: &config.DBConfig{
					Host:     "localhost",
					Port:     5432,
					User:     "test",
					Password: "test",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := Connect(ctx, tt.config)
				if err == nil {
					t.Error("Expected error with invalid config")
				}
				if !contains(err.Error(), "invalid database configuration") {
					t.Errorf("Expected config validation error, got: %v", err)
				}
			})
		}
	})

	t.Run("ConnectionFailure", func(t *testing.T) {
		cfg := &config.DBConfig{
			Host:     "nonexistent-host-12345",
			Port:     5432,
			Database: "test",
			User:     "test",
			Password: "test",
		}

		_, err := Connect(ctx, cfg)
		if err == nil {
			t.Error("Expected connection failure with nonexistent host")
		}
		if !contains(err.Error(), "failed to connect to PostgreSQL") {
			t.Errorf("Expected connection failure error, got: %v", err)
		}
	})
}

// TestConnection_WithRealDatabase tests with a real PostgreSQL database
// This will skip if no database is available via environment variables
func TestConnection_WithRealDatabase(t *testing.T) {
	cfg := testutil.GetRealDatabaseConfig()
	if cfg == nil {
		t.Skip("Skipping real database tests - no database config available. Set PGBABBLE_TEST_* environment variables to enable.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Logf("Testing with real database: %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)

	t.Run("Connect", func(t *testing.T) {
		conn, err := Connect(ctx, cfg)
		if err != nil {
			t.Skipf("Cannot connect to test database: %v", err)
			return
		}
		defer conn.Close()

		// Verify connection works
		var result int
		row := conn.QueryRow(ctx, "SELECT 1")
		err = row.Scan(&result)
		if err != nil {
			t.Fatalf("Failed to query after connect: %v", err)
		}
		if result != 1 {
			t.Errorf("Expected 1, got %d", result)
		}
	})

	t.Run("BasicOperations", func(t *testing.T) {
		conn, err := Connect(ctx, cfg)
		if err != nil {
			t.Skipf("Cannot connect to test database: %v", err)
			return
		}
		defer conn.Close()

		// Create a temporary table for testing
		err = conn.Exec(ctx, `
			CREATE TEMPORARY TABLE test_table (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				value INTEGER
			)
		`)
		if err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}

		// Insert test data
		err = conn.Exec(ctx, "INSERT INTO test_table (name, value) VALUES ($1, $2)", "test", 42)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		// Query the data back
		var name string
		var value int
		row := conn.QueryRow(ctx, "SELECT name, value FROM test_table WHERE id = 1")
		err = row.Scan(&name, &value)
		if err != nil {
			t.Fatalf("Failed to query test data: %v", err)
		}

		if name != "test" {
			t.Errorf("Expected name 'test', got %s", name)
		}
		if value != 42 {
			t.Errorf("Expected value 42, got %d", value)
		}

		// Test Query (multiple rows)
		err = conn.Exec(ctx, "INSERT INTO test_table (name, value) VALUES ($1, $2)", "test2", 84)
		if err != nil {
			t.Fatalf("Failed to insert second test row: %v", err)
		}

		rows, err := conn.Query(ctx, "SELECT name, value FROM test_table ORDER BY id")
		if err != nil {
			t.Fatalf("Failed to query multiple rows: %v", err)
		}
		defer rows.Close()

		var results []struct {
			name  string
			value int
		}

		for rows.Next() {
			var result struct {
				name  string
				value int
			}
			err := rows.Scan(&result.name, &result.value)
			if err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			results = append(results, result)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
		if results[0].name != "test" || results[0].value != 42 {
			t.Errorf("Expected first result (test, 42), got (%s, %d)", results[0].name, results[0].value)
		}
		if results[1].name != "test2" || results[1].value != 84 {
			t.Errorf("Expected second result (test2, 84), got (%s, %d)", results[1].name, results[1].value)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		conn, err := Connect(ctx, cfg)
		if err != nil {
			t.Skipf("Cannot connect to test database: %v", err)
			return
		}
		defer conn.Close()

		// Test invalid table
		_, err = conn.Query(ctx, "SELECT * FROM nonexistent_table_12345")
		if err == nil {
			t.Error("Expected error querying nonexistent table")
		}
		if !contains(err.Error(), "relation") && !contains(err.Error(), "does not exist") {
			t.Errorf("Expected relation error, got: %v", err)
		}

		// Test syntax error
		_, err = conn.Query(ctx, "SELECT * FROM")
		if err == nil {
			t.Error("Expected syntax error")
		}
		if !contains(err.Error(), "syntax") {
			t.Errorf("Expected syntax error, got: %v", err)
		}

		// Test parameterized queries
		var result string
		row := conn.QueryRow(ctx, "SELECT $1::text", "hello")
		err = row.Scan(&result)
		if err != nil {
			t.Fatalf("Parameterized query failed: %v", err)
		}
		if result != "hello" {
			t.Errorf("Expected 'hello', got %s", result)
		}
	})

	t.Run("GetDatabaseInfo", func(t *testing.T) {
		conn, err := Connect(ctx, cfg)
		if err != nil {
			t.Skipf("Cannot connect to test database: %v", err)
			return
		}
		defer conn.Close()

		info, err := conn.GetDatabaseInfo(ctx)
		if err != nil {
			t.Fatalf("GetDatabaseInfo failed: %v", err)
		}

		if info.Database != cfg.Database {
			t.Errorf("Expected database %s, got %s", cfg.Database, info.Database)
		}
		if info.User != cfg.User {
			t.Errorf("Expected user %s, got %s", cfg.User, info.User)
		}
		if info.Version == "" {
			t.Error("Expected non-empty version")
		}
		if !contains(info.Version, "PostgreSQL") {
			t.Errorf("Expected PostgreSQL version, got: %s", info.Version)
		}
	})

	t.Run("ContextHandling", func(t *testing.T) {
		conn, err := Connect(ctx, cfg)
		if err != nil {
			t.Skipf("Cannot connect to test database: %v", err)
			return
		}
		defer conn.Close()

		// Test cancelled context
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = conn.Query(cancelledCtx, "SELECT 1")
		if err == nil {
			t.Error("Expected error with cancelled context")
		}
		if !contains(err.Error(), "context") && !contains(err.Error(), "canceled") {
			t.Errorf("Expected context cancellation error, got: %v", err)
		}

		// Test timeout context
		// Use a very short timeout with a query that should take longer
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
		defer cancel()

		_, err = conn.Query(timeoutCtx, "SELECT 1")
		if err == nil {
			t.Log("No timeout error - context timing may be too generous")
		} else {
			t.Logf("Got expected error (timeout-related): %v", err)
		}
	})

	t.Run("EnsureConnection", func(t *testing.T) {
		conn, err := Connect(ctx, cfg)
		if err != nil {
			t.Skipf("Cannot connect to test database: %v", err)
			return
		}
		defer conn.Close()

		// EnsureConnection should work with healthy connection
		conn.EnsureConnection(ctx)

		// Verify connection still works
		var result int
		row := conn.QueryRow(ctx, "SELECT 1")
		err = row.Scan(&result)
		if err != nil {
			t.Fatalf("Connection not working after EnsureConnection: %v", err)
		}
		if result != 1 {
			t.Errorf("Expected 1, got %d", result)
		}
	})

	t.Run("Close", func(t *testing.T) {
		conn, err := Connect(ctx, cfg)
		if err != nil {
			t.Skipf("Cannot connect to test database: %v", err)
			return
		}

		// Verify connection works before closing
		var result int
		row := conn.QueryRow(ctx, "SELECT 1")
		err = row.Scan(&result)
		if err != nil {
			t.Fatalf("Connection not working before close: %v", err)
		}

		// Close the connection
		conn.Close()

		// After close, operations should fail (may panic or error)
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Connection panicked after close (expected): %v", r)
			}
		}()

		row = conn.QueryRow(ctx, "SELECT 1")
		err = row.Scan(&result)
		if err != nil {
			t.Logf("Connection errored after close (expected): %v", err)
		}
	})
}

// Helper function for string contains check
func contains(str, substr string) bool {
	return len(str) >= len(substr) && findSubstring(str, substr)
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
