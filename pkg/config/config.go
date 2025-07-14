package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// DBConfig holds database connection configuration
type DBConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// NewDBConfigFromURI parses a PostgreSQL URI and returns a DBConfig
func NewDBConfigFromURI(uri string) (*DBConfig, error) {
	if !strings.HasPrefix(uri, "postgresql://") && !strings.HasPrefix(uri, "postgres://") {
		return nil, fmt.Errorf("invalid PostgreSQL URI: must start with postgresql:// or postgres://")
	}

	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL URI: %w", err)
	}

	config := &DBConfig{
		Host:     parsedURL.Hostname(),
		Database: strings.TrimPrefix(parsedURL.Path, "/"),
		SSLMode:  "prefer", // default
	}

	// Parse port
	if parsedURL.Port() != "" {
		port, err := strconv.Atoi(parsedURL.Port())
		if err != nil {
			return nil, fmt.Errorf("invalid port in URI: %w", err)
		}
		config.Port = port
	} else {
		config.Port = 5432 // default PostgreSQL port
	}

	// Parse user info
	if parsedURL.User != nil {
		config.User = parsedURL.User.Username()
		if password, ok := parsedURL.User.Password(); ok {
			config.Password = password
		}
	}

	// Parse query parameters
	for key, values := range parsedURL.Query() {
		if len(values) > 0 {
			switch key {
			case "sslmode":
				config.SSLMode = values[0]
			}
		}
	}

	return config, nil
}

// NewDBConfigFromFlags creates a DBConfig from individual CLI flags and environment variables
func NewDBConfigFromFlags(host, user, password, database string, port int) *DBConfig {
	config := &DBConfig{
		Host:     getStringWithFallback(host, "PGHOST", "localhost"),
		Port:     getIntWithFallback(port, "PGPORT", 5432),
		Database: getStringWithFallback(database, "PGDATABASE", ""),
		User:     getStringWithFallback(user, "PGUSER", ""),
		Password: getStringWithFallback(password, "PGPASSWORD", ""),
		SSLMode:  getStringWithFallback("", "PGSSLMODE", "prefer"),
	}

	// If no user specified, try to get from environment or default to current user
	if config.User == "" {
		if currentUser := os.Getenv("USER"); currentUser != "" {
			config.User = currentUser
		}
	}

	return config
}

// ConnectionString returns a PostgreSQL connection string
func (c *DBConfig) ConnectionString() string {
	var parts []string

	if c.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", c.Host))
	}
	if c.Port != 0 {
		parts = append(parts, fmt.Sprintf("port=%d", c.Port))
	}
	if c.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", c.User))
	}
	if c.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", c.Password))
	}
	if c.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", c.Database))
	}
	if c.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", c.SSLMode))
	}

	return strings.Join(parts, " ")
}

// Validate checks if the configuration has required fields
func (c *DBConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

// getStringWithFallback returns the flag value, or env var, or default
func getStringWithFallback(flag, envVar, defaultValue string) string {
	if flag != "" {
		return flag
	}
	if envValue := os.Getenv(envVar); envValue != "" {
		return envValue
	}
	return defaultValue
}

// getIntWithFallback returns the flag value, or env var, or default
func getIntWithFallback(flag int, envVar string, defaultValue int) int {
	if flag != 0 {
		return flag
	}
	if envValue := os.Getenv(envVar); envValue != "" {
		if parsed, err := strconv.Atoi(envValue); err == nil {
			return parsed
		}
	}
	return defaultValue
}
