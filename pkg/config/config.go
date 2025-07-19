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
	// Validate basic format
	if uri == "" {
		return nil, fmt.Errorf("URI cannot be empty")
	}

	if len(uri) > 2048 {
		return nil, fmt.Errorf("URI too long (max 2048 characters)")
	}

	if !strings.HasPrefix(uri, "postgresql://") && !strings.HasPrefix(uri, "postgres://") {
		return nil, fmt.Errorf("invalid PostgreSQL URI: must start with postgresql:// or postgres://")
	}

	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL URI: %w", err)
	}

	// Validate required components
	if parsedURL.Hostname() == "" {
		return nil, fmt.Errorf("invalid PostgreSQL URI: hostname is required")
	}

	database := strings.TrimPrefix(parsedURL.Path, "/")
	if database == "" {
		return nil, fmt.Errorf("invalid PostgreSQL URI: database name is required")
	}

	// Validate database name doesn't contain suspicious characters
	if strings.ContainsAny(database, ";&|<>\"'`") {
		return nil, fmt.Errorf("invalid database name: contains unsafe characters")
	}

	config := &DBConfig{
		Host:     parsedURL.Hostname(),
		Database: database,
		SSLMode:  "prefer", // default
	}

	// Parse port
	if parsedURL.Port() != "" {
		port, err := strconv.Atoi(parsedURL.Port())
		if err != nil {
			return nil, fmt.Errorf("invalid port in URI: %w", err)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid port in URI: must be between 1 and 65535")
		}
		config.Port = port
	} else {
		config.Port = 5432 // default PostgreSQL port
	}

	// Parse user info
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		if username == "" {
			return nil, fmt.Errorf("invalid PostgreSQL URI: username cannot be empty")
		}
		// Validate username doesn't contain suspicious characters
		if strings.ContainsAny(username, ";&|<>\"'`") {
			return nil, fmt.Errorf("invalid username: contains unsafe characters")
		}
		config.User = username
		if password, ok := parsedURL.User.Password(); ok {
			config.Password = password
		}
	}

	// Parse query parameters
	for key, values := range parsedURL.Query() {
		if len(values) > 0 {
			switch key {
			case "sslmode":
				sslMode := values[0]
				// Validate SSL mode is one of the supported values
				validSSLModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
				isValid := false
				for _, validMode := range validSSLModes {
					if sslMode == validMode {
						isValid = true
						break
					}
				}
				if !isValid {
					return nil, fmt.Errorf("invalid sslmode '%s': must be one of %v", sslMode, validSSLModes)
				}
				config.SSLMode = sslMode
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

// MaskedURI returns a PostgreSQL URI with password masked for logging
func (c *DBConfig) MaskedURI() string {
	var userInfo string
	if c.User != "" {
		if c.Password != "" {
			userInfo = c.User + ":***@"
		} else {
			userInfo = c.User + "@"
		}
	}

	host := c.Host
	if c.Port != 5432 && c.Port != 0 {
		host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}

	uri := fmt.Sprintf("postgresql://%s%s/%s", userInfo, host, c.Database)

	// Add SSL mode if not default
	if c.SSLMode != "" && c.SSLMode != "prefer" {
		uri += "?sslmode=" + c.SSLMode
	}

	return uri
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
