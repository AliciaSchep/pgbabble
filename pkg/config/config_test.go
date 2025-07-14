package config

import (
	"os"
	"testing"
)

func TestNewDBConfigFromURI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expected    *DBConfig
		expectError bool
	}{
		{
			name: "valid postgresql URI",
			uri:  "postgresql://user:pass@localhost:5432/mydb",
			expected: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				User:     "user",
				Password: "pass",
				SSLMode:  "prefer",
			},
			expectError: false,
		},
		{
			name: "postgres URI (alternative scheme)",
			uri:  "postgres://user:pass@localhost:5432/mydb",
			expected: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				User:     "user",
				Password: "pass",
				SSLMode:  "prefer",
			},
			expectError: false,
		},
		{
			name: "URI without port (should default to 5432)",
			uri:  "postgresql://user:pass@localhost/mydb",
			expected: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				User:     "user",
				Password: "pass",
				SSLMode:  "prefer",
			},
			expectError: false,
		},
		{
			name: "URI without password",
			uri:  "postgresql://user@localhost:5432/mydb",
			expected: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				User:     "user",
				Password: "",
				SSLMode:  "prefer",
			},
			expectError: false,
		},
		{
			name: "URI with SSL mode",
			uri:  "postgresql://user:pass@localhost:5432/mydb?sslmode=require",
			expected: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				User:     "user",
				Password: "pass",
				SSLMode:  "require",
			},
			expectError: false,
		},
		{
			name:        "invalid scheme",
			uri:         "mysql://user:pass@localhost:3306/mydb",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid port",
			uri:         "postgresql://user:pass@localhost:invalid/mydb",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "malformed URI",
			uri:         "not-a-uri",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewDBConfigFromURI(tt.uri)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config.Host != tt.expected.Host {
				t.Errorf("Host: expected %s, got %s", tt.expected.Host, config.Host)
			}
			if config.Port != tt.expected.Port {
				t.Errorf("Port: expected %d, got %d", tt.expected.Port, config.Port)
			}
			if config.Database != tt.expected.Database {
				t.Errorf("Database: expected %s, got %s", tt.expected.Database, config.Database)
			}
			if config.User != tt.expected.User {
				t.Errorf("User: expected %s, got %s", tt.expected.User, config.User)
			}
			if config.Password != tt.expected.Password {
				t.Errorf("Password: expected %s, got %s", tt.expected.Password, config.Password)
			}
			if config.SSLMode != tt.expected.SSLMode {
				t.Errorf("SSLMode: expected %s, got %s", tt.expected.SSLMode, config.SSLMode)
			}
		})
	}
}

func TestNewDBConfigFromFlags(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"PGHOST":     os.Getenv("PGHOST"),
		"PGPORT":     os.Getenv("PGPORT"),
		"PGUSER":     os.Getenv("PGUSER"),
		"PGPASSWORD": os.Getenv("PGPASSWORD"),
		"PGDATABASE": os.Getenv("PGDATABASE"),
		"PGSSLMODE":  os.Getenv("PGSSLMODE"),
		"USER":       os.Getenv("USER"),
	}
	defer func() {
		// Restore original environment
		for key, value := range originalEnv {
			if value == "" {
				if err := os.Unsetenv(key); err != nil {
					t.Logf("failed to unset env var %s: %v", key, err)
				}
			} else {
				if err := os.Setenv(key, value); err != nil {
					t.Logf("failed to set env var %s: %v", key, err)
				}
			}
		}
	}()

	tests := []struct {
		name     string
		host     string
		user     string
		password string
		database string
		port     int
		envVars  map[string]string
		expected *DBConfig
	}{
		{
			name:     "flags only",
			host:     "localhost",
			user:     "testuser",
			password: "testpass",
			database: "testdb",
			port:     5432,
			envVars:  map[string]string{},
			expected: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
				SSLMode:  "prefer",
			},
		},
		{
			name:     "environment variables fallback",
			host:     "",
			user:     "",
			password: "",
			database: "",
			port:     0,
			envVars: map[string]string{
				"PGHOST":     "envhost",
				"PGPORT":     "5433",
				"PGUSER":     "envuser",
				"PGPASSWORD": "envpass",
				"PGDATABASE": "envdb",
				"PGSSLMODE":  "require",
			},
			expected: &DBConfig{
				Host:     "envhost",
				Port:     5433,
				Database: "envdb",
				User:     "envuser",
				Password: "envpass",
				SSLMode:  "require",
			},
		},
		{
			name:     "flags override environment",
			host:     "flaghost",
			user:     "flaguser",
			password: "",
			database: "flagdb",
			port:     5434,
			envVars: map[string]string{
				"PGHOST":     "envhost",
				"PGUSER":     "envuser",
				"PGPASSWORD": "envpass",
				"PGDATABASE": "envdb",
			},
			expected: &DBConfig{
				Host:     "flaghost",
				Port:     5434,
				Database: "flagdb",
				User:     "flaguser",
				Password: "envpass", // from env since flag was empty
				SSLMode:  "prefer",
			},
		},
		{
			name:     "default values",
			host:     "",
			user:     "",
			password: "",
			database: "",
			port:     0,
			envVars: map[string]string{
				"USER": "currentuser",
			},
			expected: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "",
				User:     "currentuser",
				Password: "",
				SSLMode:  "prefer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			envVarsToUnset := []string{"PGHOST", "PGPORT", "PGUSER", "PGPASSWORD", "PGDATABASE", "PGSSLMODE", "USER"}
			for _, envVar := range envVarsToUnset {
				if err := os.Unsetenv(envVar); err != nil {
					t.Logf("failed to unset %s: %v", envVar, err)
				}
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("failed to set env var %s: %v", key, err)
				}
			}

			config := NewDBConfigFromFlags(tt.host, tt.user, tt.password, tt.database, tt.port)

			if config.Host != tt.expected.Host {
				t.Errorf("Host: expected %s, got %s", tt.expected.Host, config.Host)
			}
			if config.Port != tt.expected.Port {
				t.Errorf("Port: expected %d, got %d", tt.expected.Port, config.Port)
			}
			if config.Database != tt.expected.Database {
				t.Errorf("Database: expected %s, got %s", tt.expected.Database, config.Database)
			}
			if config.User != tt.expected.User {
				t.Errorf("User: expected %s, got %s", tt.expected.User, config.User)
			}
			if config.Password != tt.expected.Password {
				t.Errorf("Password: expected %s, got %s", tt.expected.Password, config.Password)
			}
			if config.SSLMode != tt.expected.SSLMode {
				t.Errorf("SSLMode: expected %s, got %s", tt.expected.SSLMode, config.SSLMode)
			}
		})
	}
}

func TestConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   *DBConfig
		expected string
	}{
		{
			name: "all fields",
			config: &DBConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "mydb",
				User:     "user",
				Password: "pass",
				SSLMode:  "require",
			},
			expected: "host=localhost port=5432 user=user password=pass dbname=mydb sslmode=require",
		},
		{
			name: "minimal config",
			config: &DBConfig{
				Host:     "localhost",
				Database: "mydb",
				User:     "user",
			},
			expected: "host=localhost user=user dbname=mydb",
		},
		{
			name: "with port only",
			config: &DBConfig{
				Host: "localhost",
				Port: 5433,
				User: "user",
			},
			expected: "host=localhost port=5433 user=user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ConnectionString()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *DBConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &DBConfig{
				Host:     "localhost",
				User:     "user",
				Database: "mydb",
			},
			expectError: false,
		},
		{
			name: "missing host",
			config: &DBConfig{
				User:     "user",
				Database: "mydb",
			},
			expectError: true,
			errorMsg:    "database host is required",
		},
		{
			name: "missing user",
			config: &DBConfig{
				Host:     "localhost",
				Database: "mydb",
			},
			expectError: true,
			errorMsg:    "database user is required",
		},
		{
			name: "missing database",
			config: &DBConfig{
				Host: "localhost",
				User: "user",
			},
			expectError: true,
			errorMsg:    "database name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %s, got %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
