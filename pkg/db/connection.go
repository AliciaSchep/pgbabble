package db

import (
	"context"
	"fmt"
	"time"

	"github.com/AliciaSchep/pgbabble/pkg/config"
	"github.com/jackc/pgx/v5"
)

// ConnectionImpl wraps a PostgreSQL connection and implements the Connection interface
type ConnectionImpl struct {
	conn   *pgx.Conn
	config *config.DBConfig
}

// Connect establishes a connection to PostgreSQL using the provided config
func Connect(ctx context.Context, cfg *config.DBConfig) (*ConnectionImpl, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database configuration cannot be nil")
	}
	
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Create single connection
	conn, err := pgx.Connect(ctx, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		if closeErr := conn.Close(ctx); closeErr != nil {
			fmt.Printf("Warning: failed to close connection: %v\n", closeErr)
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &ConnectionImpl{
		conn:   conn,
		config: cfg,
	}, nil
}

// Close closes the database connection
func (c *ConnectionImpl) Close() {
	if c.conn != nil {
		if err := c.conn.Close(context.Background()); err != nil {
			fmt.Printf("Warning: failed to close connection: %v\n", err)
		}
	}
}

// Query executes a query and returns the rows
func (c *ConnectionImpl) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return c.conn.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns at most one row
func (c *ConnectionImpl) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return c.conn.QueryRow(ctx, sql, args...)
}

// EnsureConnection ensures we have a healthy database connection, reconnecting if necessary
func (c *ConnectionImpl) EnsureConnection(ctx context.Context) {
	if c.conn == nil {
		// Initial connection during startup
		c.reconnectWithRetry(ctx)
		return
	}

	// Check if connection is alive
	err := c.conn.Ping(ctx)
	for err != nil {
		fmt.Print("Connection to PostgreSQL was lost. Waiting 5s...")
		if c.conn != nil {
			if err := c.conn.Close(ctx); err != nil {
				fmt.Printf("Warning: failed to close connection: %v\n", err)
			}
		}
		time.Sleep(5 * time.Second)
		fmt.Print(" reconnecting...")
		c.reconnectWithRetry(ctx)
		if c.conn != nil {
			err = c.conn.Ping(ctx)
		}
	}
}

// reconnectWithRetry attempts to reconnect to the database
func (c *ConnectionImpl) reconnectWithRetry(ctx context.Context) {
	// Create new connection
	conn, err := pgx.Connect(ctx, c.config.ConnectionString())
	if err != nil {
		fmt.Printf(" failed to connect: %v\n", err)
		return
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		fmt.Printf(" failed to ping: %v\n", err)
		if closeErr := conn.Close(ctx); closeErr != nil {
			fmt.Printf("Warning: failed to close connection: %v\n", closeErr)
		}
		return
	}

	c.conn = conn
	fmt.Print(" connected!\n")
}

// Exec executes a query without returning any rows
func (c *ConnectionImpl) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := c.conn.Exec(ctx, sql, args...)
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
	err := c.conn.QueryRow(ctx, "SELECT version()").Scan(&version)
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
