package db

import (
	"context"
	"fmt"
	"time"

	"github.com/AliciaSchep/pgbabble/pkg/config"
	"github.com/AliciaSchep/pgbabble/pkg/errors"
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
		return nil, fmt.Errorf("failed to connect to PostgreSQL %s@%s:%d/%s: %w", 
			cfg.User, cfg.Host, cfg.Port, cfg.Database, err)
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		if closeErr := conn.Close(ctx); closeErr != nil {
			errors.ConnectionWarning("failed to close connection during setup: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to ping PostgreSQL %s@%s:%d/%s: %w", 
			cfg.User, cfg.Host, cfg.Port, cfg.Database, err)
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
			errors.ConnectionWarning("failed to close connection: %v", err)
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
	retryCount := 0
	maxRetries := 3
	
	for err != nil && retryCount < maxRetries {
		// Check if context was cancelled
		if ctx.Err() != nil {
			fmt.Printf(" connection retry cancelled: %v\n", ctx.Err())
			return
		}
		
		fmt.Print("Connection to PostgreSQL was lost. Waiting 5s...")
		if c.conn != nil {
			if err := c.conn.Close(ctx); err != nil {
				errors.ConnectionWarning("failed to close stale connection: %v", err)
			}
		}
		
		// Wait with context cancellation support
		select {
		case <-time.After(5 * time.Second):
			// Continue with reconnection
		case <-ctx.Done():
			fmt.Printf(" connection retry cancelled: %v\n", ctx.Err())
			return
		}
		
		fmt.Print(" reconnecting...")
		c.reconnectWithRetry(ctx)
		if c.conn != nil {
			err = c.conn.Ping(ctx)
		}
		retryCount++
	}
	
	if err != nil && retryCount >= maxRetries {
		fmt.Printf(" failed to reconnect after %d attempts: %v\n", maxRetries, err)
	}
}

// reconnectWithRetry attempts to reconnect to the database
func (c *ConnectionImpl) reconnectWithRetry(ctx context.Context) {
	// Check if context was cancelled before attempting connection
	if ctx.Err() != nil {
		fmt.Printf(" reconnection cancelled: %v\n", ctx.Err())
		return
	}
	
	// Create new connection
	conn, err := pgx.Connect(ctx, c.config.ConnectionString())
	if err != nil {
		fmt.Printf(" reconnection failed: %v\n", err)
		return
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		fmt.Printf(" ping failed: %v\n", err)
		if closeErr := conn.Close(ctx); closeErr != nil {
			errors.ConnectionWarning("failed to close connection after ping failure: %v", closeErr)
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
		return nil, fmt.Errorf("failed to get PostgreSQL version from %s@%s:%d/%s: %w", 
			c.config.User, c.config.Host, c.config.Port, c.config.Database, err)
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
