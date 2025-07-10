package agent

import (
	"testing"
)

func TestValidateSafeQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		errorContains string
	}{
		// Valid queries
		{
			name:        "simple select",
			query:       "SELECT * FROM users",
			expectError: false,
		},
		{
			name:        "select with where",
			query:       "SELECT id, name FROM users WHERE active = true",
			expectError: false,
		},
		{
			name:        "select with joins",
			query:       "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id",
			expectError: false,
		},
		{
			name:        "CTE with WITH",
			query:       "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users",
			expectError: false,
		},
		{
			name:        "select with line comments",
			query:       "SELECT * FROM users -- this is a comment\nWHERE id = 1",
			expectError: false,
		},
		{
			name:        "select with block comments",
			query:       "SELECT * FROM users /* this is a comment */ WHERE id = 1",
			expectError: false,
		},
		{
			name:        "lowercase select",
			query:       "select * from users",
			expectError: false,
		},
		{
			name:        "mixed case select",
			query:       "Select * From users",
			expectError: false,
		},

		// Invalid queries - DDL
		{
			name:          "create table",
			query:         "CREATE TABLE test (id int)",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "drop table",
			query:         "DROP TABLE users",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "alter table",
			query:         "ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},

		// Invalid queries - DML
		{
			name:          "insert",
			query:         "INSERT INTO users (name) VALUES ('test')",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "update",
			query:         "UPDATE users SET name = 'test' WHERE id = 1",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "delete",
			query:         "DELETE FROM users WHERE id = 1",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "truncate",
			query:         "TRUNCATE TABLE users",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},

		// Invalid queries - dangerous functions in SELECT
		{
			name:          "pg_sleep in select",
			query:         "SELECT pg_sleep(10)",
			expectError:   true,
			errorContains: "potentially dangerous operation: PG_SLEEP",
		},
		{
			name:          "pg_terminate_backend",
			query:         "SELECT pg_terminate_backend(123)",
			expectError:   true,
			errorContains: "potentially dangerous operation: PG_TERMINATE_BACKEND",
		},
		{
			name:          "dblink_exec",
			query:         "SELECT dblink_exec('host=localhost', 'DROP TABLE test')",
			expectError:   true,
			errorContains: "potentially dangerous operation: DBLINK",
		},
		{
			name:          "copy in select",
			query:         "SELECT * FROM users; COPY users TO '/tmp/users.csv'",
			expectError:   true,
			errorContains: "potentially dangerous operation: COPY",
		},

		// Invalid queries - privilege escalation
		{
			name:          "set role",
			query:         "SET ROLE postgres",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "grant privileges",
			query:         "GRANT ALL ON users TO public",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},

		// Edge cases
		{
			name:          "empty query",
			query:         "",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "whitespace only",
			query:         "   \n\t  ",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "comment only",
			query:         "-- this is just a comment",
			expectError:   true,
			errorContains: "only SELECT and WITH queries are allowed",
		},
		{
			name:          "hidden insert in comment",
			query:         "SELECT * FROM users /* INSERT INTO users VALUES (1, 'hack') */",
			expectError:   false, // Should be safe since it's in a comment
		},
		{
			name:          "unclosed block comment",
			query:         "SELECT * FROM users /* unclosed comment",
			expectError:   true,
			errorContains: "unclosed block comment",
		},

		// Case sensitivity tests for dangerous patterns
		{
			name:          "lowercase dangerous function",
			query:         "SELECT pg_sleep(1)",
			expectError:   true,
			errorContains: "potentially dangerous operation: PG_SLEEP",
		},
		{
			name:          "mixed case dangerous function",
			query:         "SELECT Pg_Sleep(1)",
			expectError:   true,
			errorContains: "potentially dangerous operation: PG_SLEEP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeQuery(tt.query)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for query: %s", tt.query)
					return
				}
				if tt.errorContains != "" && !containsIgnoreCase(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for valid query '%s': %v", tt.query, err)
				}
			}
		})
	}
}

func TestValidateQueryContent(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "safe select",
			query:       "SELECT NAME, EMAIL FROM USERS WHERE ACTIVE = TRUE",
			expectError: false,
		},
		{
			name:          "contains insert",
			query:         "SELECT * FROM USERS; INSERT INTO LOGS VALUES (1)",
			expectError:   true,
			errorContains: "INSERT",
		},
		{
			name:          "contains delete",
			query:         "SELECT COUNT(*) FROM USERS WHERE ID IN (DELETE FROM TEMP_IDS RETURNING ID)",
			expectError:   true,
			errorContains: "DELETE",
		},
		{
			name:          "contains pg_sleep",
			query:         "SELECT *, PG_SLEEP(5) FROM USERS",
			expectError:   true,
			errorContains: "PG_SLEEP",
		},
		{
			name:          "contains do block",
			query:         "SELECT 1; DO $$ BEGIN RAISE NOTICE 'test'; END $$",
			expectError:   true,
			errorContains: "DO $$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryContent(tt.query)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for query: %s", tt.query)
					return
				}
				if tt.errorContains != "" && !containsIgnoreCase(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for safe query '%s': %v", tt.query, err)
				}
			}
		})
	}
}

// Helper function for case-insensitive string contains check
func containsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) && 
		   len(substr) > 0 && 
		   (str == substr || 
		    (len(str) > len(substr) && 
		     (str[:len(substr)] == substr || 
		      str[len(str)-len(substr):] == substr || 
		      stringContains(str, substr))))
}

func stringContains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}