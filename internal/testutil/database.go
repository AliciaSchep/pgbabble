package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/AliciaSchep/pgbabble/pkg/config"
)

// TestDatabase represents a test database instance
type TestDatabase struct {
	Config *config.DBConfig
}

// NewTestDatabase creates a new PostgreSQL test database
// Uses environment variables or starts a Docker container
func NewTestDatabase(ctx context.Context, t *testing.T) (*TestDatabase, error) {
	// Try environment variables first (for CI or local development)
	if envConfig := tryEnvironmentConfig(); envConfig != nil {
		t.Logf("Using environment-configured test database: %s",
			envConfig.MaskedURI())

		testDB := &TestDatabase{
			Config: envConfig,
		}

		// Setup schema and seed data
		if err := testDB.setupSchema(ctx); err != nil {
			return nil, fmt.Errorf("failed to setup test schema: %w", err)
		}

		return testDB, nil
	}

	// If no environment config, we'll rely on the caller to set up database
	// (This is where Docker setup would happen in the Makefile)
	return nil, fmt.Errorf("no test database configuration found - please set PGBABBLE_TEST_* environment variables or run 'make test-db-start'")
}

// tryEnvironmentConfig attempts to create a config from environment variables
func tryEnvironmentConfig() *config.DBConfig {
	host := os.Getenv("PGBABBLE_TEST_HOST")
	if host == "" {
		host = os.Getenv("PGHOST")
	}
	if host == "" {
		return nil
	}

	user := os.Getenv("PGBABBLE_TEST_USER")
	if user == "" {
		user = os.Getenv("PGUSER")
	}

	database := os.Getenv("PGBABBLE_TEST_DATABASE")
	if database == "" {
		database = os.Getenv("PGDATABASE")
	}

	password := os.Getenv("PGBABBLE_TEST_PASSWORD")
	if password == "" {
		password = os.Getenv("PGPASSWORD")
	}

	port := 5432
	if portStr := os.Getenv("PGBABBLE_TEST_PORT"); portStr != "" {
		if p, err := parsePort(portStr); err == nil {
			port = p
		}
	} else if portStr := os.Getenv("PGPORT"); portStr != "" {
		if p, err := parsePort(portStr); err == nil {
			port = p
		}
	}

	return &config.DBConfig{
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
	}
}

// parsePort safely parses a port string
func parsePort(portStr string) (int, error) {
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		return 0, err
	}
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port: %d", port)
	}
	return port, nil
}

// setupSchema creates the test schema and seeds initial data
func (td *TestDatabase) setupSchema(ctx context.Context) error {
	// We'll use the db.Connect method from the main package
	// Import would create a cycle, so we'll pass the connection setup to tests
	return nil // Schema setup will be done in the test
}

// GetConfig returns the database configuration
func (td *TestDatabase) GetConfig() *config.DBConfig {
	return td.Config
}

// CleanupAfterTest provides a cleanup function for use with t.Cleanup()
func (td *TestDatabase) CleanupAfterTest(ctx context.Context, t *testing.T) {
	t.Cleanup(func() {
		// Any cleanup if we managed our own container would go here
		// For now, we rely on external setup/teardown
	})
}

// GetRealDatabaseConfig attempts to get database config from environment variables
// This is a helper that tests can use directly without creating a TestDatabase
func GetRealDatabaseConfig() *config.DBConfig {
	host := os.Getenv("PGBABBLE_TEST_HOST")
	if host == "" {
		host = os.Getenv("PGHOST")
	}
	if host == "" {
		host = "localhost" // Try localhost as default
	}

	user := os.Getenv("PGBABBLE_TEST_USER")
	if user == "" {
		user = os.Getenv("PGUSER")
	}
	if user == "" {
		user = os.Getenv("USER") // Try current user as default
	}

	database := os.Getenv("PGBABBLE_TEST_DATABASE")
	if database == "" {
		database = os.Getenv("PGDATABASE")
	}
	if database == "" {
		database = "postgres" // Try default postgres database
	}

	password := os.Getenv("PGBABBLE_TEST_PASSWORD")
	if password == "" {
		password = os.Getenv("PGPASSWORD")
	}

	port := 5432
	if portStr := os.Getenv("PGBABBLE_TEST_PORT"); portStr != "" {
		if p, err := parsePort(portStr); err == nil {
			port = p
		}
	} else if portStr := os.Getenv("PGPORT"); portStr != "" {
		if p, err := parsePort(portStr); err == nil {
			port = p
		}
	}

	return &config.DBConfig{
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
	}
}

// SetupTestSchema creates test tables in the given database connection
// This avoids import cycles by having tests call this with their connection
func SetupTestSchema(ctx context.Context, execFunc func(context.Context, string) error) error {
	schema := `
	-- Drop tables if they exist (for clean setup)
	DROP TABLE IF EXISTS test_order_items CASCADE;
	DROP TABLE IF EXISTS test_orders CASCADE;
	DROP TABLE IF EXISTS test_products CASCADE;
	DROP TABLE IF EXISTS test_users CASCADE;

	-- Create users table
	CREATE TABLE test_users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create products table
	CREATE TABLE test_products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		category VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create orders table
	CREATE TABLE test_orders (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES test_users(id),
		total_amount DECIMAL(10,2) NOT NULL,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create order_items table
	CREATE TABLE test_order_items (
		id SERIAL PRIMARY KEY,
		order_id INTEGER REFERENCES test_orders(id),
		product_id INTEGER REFERENCES test_products(id),
		quantity INTEGER NOT NULL,
		price DECIMAL(10,2) NOT NULL
	);

	-- Insert seed data
	INSERT INTO test_users (username, email) VALUES
		('alice', 'alice@example.com'),
		('bob', 'bob@example.com'),
		('charlie', 'charlie@example.com');

	INSERT INTO test_products (name, price, category) VALUES
		('Laptop', 999.99, 'Electronics'),
		('Mouse', 29.99, 'Electronics'),
		('Book', 19.99, 'Books'),
		('Coffee Mug', 12.99, 'Kitchen'),
		('Desk Chair', 199.99, 'Furniture');

	INSERT INTO test_orders (user_id, total_amount, status) VALUES
		(1, 1029.98, 'completed'),
		(2, 19.99, 'pending'),
		(3, 212.98, 'shipped');

	INSERT INTO test_order_items (order_id, product_id, quantity, price) VALUES
		(1, 1, 1, 999.99),
		(1, 2, 1, 29.99),
		(2, 3, 1, 19.99),
		(3, 4, 1, 12.99),
		(3, 5, 1, 199.99);
	`

	return execFunc(ctx, schema)
}

// CleanupTestSchema removes test tables
func CleanupTestSchema(ctx context.Context, execFunc func(context.Context, string) error) error {
	cleanup := `
	DROP TABLE IF EXISTS test_order_items CASCADE;
	DROP TABLE IF EXISTS test_orders CASCADE;
	DROP TABLE IF EXISTS test_products CASCADE;
	DROP TABLE IF EXISTS test_users CASCADE;
	`
	return execFunc(ctx, cleanup)
}
