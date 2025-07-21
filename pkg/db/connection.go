package db

import (
	"context"
	"fmt"
	"time"

	"github.com/AliciaSchep/pgbabble/pkg/config"
	"github.com/AliciaSchep/pgbabble/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectionImpl wraps a PostgreSQL connection pool and implements the Connection interface
type ConnectionImpl struct {
	pool   *pgxpool.Pool
	config *config.DBConfig
}

// Connect establishes a connection pool to PostgreSQL using the provided config
func Connect(ctx context.Context, cfg *config.DBConfig) (*ConnectionImpl, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database configuration cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Create connection pool with default settings
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure pool settings for our use case
	poolConfig.MaxConns = 5        // Small pool since we're a CLI tool
	poolConfig.MinConns = 1        // Keep at least one connection alive
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool to PostgreSQL %s: %w",
			cfg.MaskedURI(), err)
	}

	// Test the connection pool
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL %s: connection test failed",
			cfg.MaskedURI())
	}

	return &ConnectionImpl{
		pool:   pool,
		config: cfg,
	}, nil
}

// Close closes the database connection pool
func (c *ConnectionImpl) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

// Query executes a query and returns the rows
func (c *ConnectionImpl) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return c.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (c *ConnectionImpl) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return c.pool.QueryRow(ctx, sql, args...)
}

// EnsureConnection ensures we have a healthy database connection pool
func (c *ConnectionImpl) EnsureConnection(ctx context.Context) {
	// With connection pools, this is much simpler - just ping to verify
	if c.pool != nil {
		if err := c.pool.Ping(ctx); err != nil {
			errors.ConnectionWarning("connection pool ping failed: %v", err)
		}
	}
}

// ForceReconnect is not needed with connection pools - they handle reconnection automatically
// This method is kept for interface compatibility but is essentially a no-op
func (c *ConnectionImpl) ForceReconnect(ctx context.Context) {
	// Connection pools automatically handle connection failures and reconnection
	// We just ping to ensure the pool is healthy
	c.EnsureConnection(ctx)
}

// Exec executes a query without returning any rows
func (c *ConnectionImpl) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := c.pool.Exec(ctx, sql, args...)
	return err
}

// GetDatabaseInfo returns basic information about the connected database
func (c *ConnectionImpl) GetDatabaseInfo(ctx context.Context) (*DatabaseInfo, error) {
	info := &DatabaseInfo{
		Host:     c.config.Host,
		Port:     c.config.Port,
		Database: c.config.Database,
		User:     c.config.User,
	}

	// Get PostgreSQL version
	var version string
	err := c.pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return nil, fmt.Errorf("failed to get PostgreSQL version from %s: %w",
			c.config.MaskedURI(), err)
	}
	info.Version = version

	return info, nil
}

// DatabaseInfo holds information about the connected database
type DatabaseInfo struct {
	Host     string
	Port     int
	Database string
	User     string
	Version  string
}
