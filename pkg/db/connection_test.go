package db

import (
	"context"
	"testing"
	"time"

	"pgbabble/pkg/config"
)

func TestDBConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.DBConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &config.DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "test",
				User:     "testuser",
				Password: "testpass",
			},
			expectError: false,
		},
		{
			name: "missing host",
			config: &config.DBConfig{
				Port:     5432,
				Database: "test",
				User:     "testuser",
			},
			expectError: true,
		},
		{
			name: "missing user",
			config: &config.DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "test",
			},
			expectError: true,
		},
		{
			name: "missing database",
			config: &config.DBConfig{
				Host: "localhost",
				Port: 5432,
				User: "testuser",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestConnect_InvalidConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with invalid config
	invalidConfig := &config.DBConfig{
		Host: "", // missing required field
	}

	_, err := Connect(ctx, invalidConfig)
	if err == nil {
		t.Error("expected error with invalid config but got none")
	}
}

// Note: These tests require a real PostgreSQL instance to run
// In a real project, you might use testcontainers or docker-compose for integration tests
func TestConnect_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires a PostgreSQL instance running locally
	// Skip if not available (this would be better handled with testcontainers)
	cfg := &config.DBConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "postgres", // default database that should exist
		User:     "postgres",
		Password: "password", // adjust as needed for your test setup
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := Connect(ctx, cfg)
	if err != nil {
		t.Skipf("skipping integration test - cannot connect to PostgreSQL: %v", err)
		return
	}
	defer conn.Close()

	// Test basic operations
	info, err := conn.GetDatabaseInfo(ctx)
	if err != nil {
		t.Errorf("failed to get database info: %v", err)
		return
	}

	if info.Database != cfg.Database {
		t.Errorf("expected database %s, got %s", cfg.Database, info.Database)
	}

	if info.User != cfg.User {
		t.Errorf("expected user %s, got %s", cfg.User, info.User)
	}

	if info.Version == "" {
		t.Error("expected non-empty version string")
	}
}

func TestConnection_Close(t *testing.T) {
	// Test that Close() doesn't panic with nil pool
	conn := &Connection{}
	conn.Close() // should not panic
}

func TestConnectionString_Generation(t *testing.T) {
	cfg := &config.DBConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "require",
	}

	connStr := cfg.ConnectionString()
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=require"

	if connStr != expected {
		t.Errorf("expected connection string %s, got %s", expected, connStr)
	}
}