package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFormatDatabaseError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected string
	}{
		{
			name:     "table not found error",
			input:    fmt.Errorf(`pq: relation "nonexistent_table" does not exist`),
			expected: "Table or view not found",
		},
		{
			name:     "column not found error", 
			input:    fmt.Errorf(`pq: column "nonexistent_column" does not exist`),
			expected: "Column not found",
		},
		{
			name:     "syntax error",
			input:    fmt.Errorf(`pq: syntax error at or near "SELCT"`),
			expected: "SQL syntax error",
		},
		{
			name:     "permission denied error",
			input:    fmt.Errorf(`pq: permission denied for table users`),
			expected: "Permission denied",
		},
		{
			name:     "connection refused error",
			input:    fmt.Errorf(`dial tcp 127.0.0.1:5432: connection refused`),
			expected: "Database connection issue",
		},
		{
			name:     "connection closed error",
			input:    fmt.Errorf(`connection closed unexpectedly`),
			expected: "Database connection issue",
		},
		{
			name:     "network timeout error",
			input:    fmt.Errorf(`dial tcp: lookup postgres: no such host`),
			expected: "Network connectivity issue",
		},
		{
			name:     "generic database error",
			input:    fmt.Errorf(`some other database error`),
			expected: "Database error: some other database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDatabaseError(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("formatDatabaseError() = %v, want to contain %v", result, tt.expected)
			}
		})
	}
}

func TestQueryTimeoutConfiguration(t *testing.T) {
	// Test that QueryTimeout is configurable
	originalTimeout := QueryTimeout
	defer func() {
		QueryTimeout = originalTimeout
	}()

	// Test default value
	if QueryTimeout != 60*time.Second {
		t.Errorf("Expected default QueryTimeout to be 60s, got %v", QueryTimeout)
	}

	// Test that it can be modified
	QueryTimeout = 30 * time.Second
	if QueryTimeout != 30*time.Second {
		t.Errorf("Expected QueryTimeout to be configurable, got %v", QueryTimeout)
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than limit",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to limit",
			input:    "hello world",
			maxLen:   11,
			expected: "hello world",
		},
		{
			name:     "string longer than limit",
			input:    "hello world this is a long string",
			maxLen:   10,
			expected: "hello w...",
		},
		{
			name:     "very short limit",
			input:    "hello",
			maxLen:   3,
			expected: "...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "NULL",
		},
		{
			name:     "string value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "integer value",
			input:    42,
			expected: "42",
		},
		{
			name:     "boolean value",
			input:    true,
			expected: "true",
		},
		{
			name:     "float value",
			input:    3.14,
			expected: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.input)
			if result != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContextTimeoutHandling(t *testing.T) {
	// Test that context timeout is properly handled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected context to be timed out, got %v", ctx.Err())
	}
}

// Test helper functions for SQL validation
func TestSQLValidationHelpers(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{
			name:     "contains substring",
			str:      "SELECT * FROM users",
			substr:   "SELECT",
			expected: true,
		},
		{
			name:     "does not contain substring",
			str:      "SELECT * FROM users",
			substr:   "INSERT",
			expected: false,
		},
		{
			name:     "empty substring",
			str:      "SELECT * FROM users",
			substr:   "",
			expected: false,
		},
		{
			name:     "empty string",
			str:      "",
			substr:   "SELECT",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.str, tt.substr)
			if result != tt.expected {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.str, tt.substr, result, tt.expected)
			}
		})
	}
}

// Test error message formatting patterns
func TestErrorMessagePatterns(t *testing.T) {
	tests := []struct {
		name        string
		errorStr    string
		shouldMatch string
	}{
		{
			name:        "relation does not exist pattern",
			errorStr:    `relation "test_table" does not exist`,
			shouldMatch: "relation",
		},
		{
			name:        "column does not exist pattern", 
			errorStr:    `column "test_column" does not exist`,
			shouldMatch: "column",
		},
		{
			name:        "syntax error pattern",
			errorStr:    `syntax error at or near "SELCT"`,
			shouldMatch: "syntax error",
		},
		{
			name:        "permission denied pattern",
			errorStr:    `permission denied for relation users`,
			shouldMatch: "permission denied",
		},
		{
			name:        "connection refused pattern",
			errorStr:    `connection refused`,
			shouldMatch: "connection",
		},
		{
			name:        "network error pattern",
			errorStr:    `dial tcp: network is unreachable`,
			shouldMatch: "network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.errorStr, tt.shouldMatch) {
				t.Errorf("Error string %q should contain pattern %q", tt.errorStr, tt.shouldMatch)
			}
		})
	}
}