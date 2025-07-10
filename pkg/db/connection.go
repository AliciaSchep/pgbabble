package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"pgbabble/pkg/config"
)

// Connection wraps a PostgreSQL connection pool
type Connection struct {
	pool   *pgxpool.Pool
	config *config.DBConfig
}

// Connect establishes a connection to PostgreSQL using the provided config
func Connect(ctx context.Context, cfg *config.DBConfig) (*Connection, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Create connection pool config
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set pool settings
	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{
		pool:   pool,
		config: cfg,
	}, nil
}

// Close closes the database connection pool
func (c *Connection) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

// Pool returns the underlying connection pool for direct use
func (c *Connection) Pool() *pgxpool.Pool {
	return c.pool
}

// Query executes a query and returns the rows
func (c *Connection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return c.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (c *Connection) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return c.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a query without returning any rows
func (c *Connection) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := c.pool.Exec(ctx, sql, args...)
	return err
}

// GetDatabaseInfo returns basic information about the connected database
func (c *Connection) GetDatabaseInfo(ctx context.Context) (*DatabaseInfo, error) {
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
		return nil, fmt.Errorf("failed to get PostgreSQL version: %w", err)
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