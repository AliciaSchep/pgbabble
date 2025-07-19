package display

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveCSV(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		columnNames []string
		rows        [][]interface{}
		filename    string
		expected    string
	}{
		{
			name:        "basic_table",
			columnNames: []string{"id", "name", "age"},
			rows: [][]interface{}{
				{1, "Alice", 30},
				{2, "Bob", 25},
			},
			filename: "test1.csv",
			expected: "id,name,age\n1,Alice,30\n2,Bob,25\n",
		},
		{
			name:        "table_with_nulls_and_empty_strings",
			columnNames: []string{"id", "name", "email"},
			rows: [][]interface{}{
				{1, "Alice", "alice@example.com"},
				{2, nil, ""},
				{3, "Charlie", nil},
			},
			filename: "test2.csv",
			expected: "id,name,email\n1,Alice,alice@example.com\n2,,\n3,Charlie,\n",
		},
		{
			name:        "table_with_special_characters",
			columnNames: []string{"id", "description"},
			rows: [][]interface{}{
				{1, "Text with, comma"},
				{2, "Text with \"quotes\""},
				{3, "Text with\nnewline"},
			},
			filename: "test3.csv",
			expected: "id,description\n1,\"Text with, comma\"\n2,\"Text with \"\"quotes\"\"\"\n3,\"Text with\nnewline\"\n",
		},
		{
			name:        "empty_table",
			columnNames: []string{"id", "name"},
			rows:        [][]interface{}{},
			filename:    "test4.csv",
			expected:    "id,name\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(tempDir, tt.filename)

			err := SaveCSV(tt.columnNames, tt.rows, filename)
			if err != nil {
				t.Fatalf("SaveCSV failed: %v", err)
			}

			// Read the file content
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("Failed to read CSV file: %v", err)
			}

			if string(content) != tt.expected {
				t.Errorf("CSV content mismatch.\nExpected:\n%q\nGot:\n%q", tt.expected, string(content))
			}
		})
	}
}

func TestGenerateDefaultCSVFilename(t *testing.T) {
	filename := GenerateDefaultCSVFilename()

	// Check that it has the expected prefix and suffix
	if !strings.HasPrefix(filename, "pgbabble_results_") {
		t.Errorf("Expected filename to start with 'pgbabble_results_', got: %s", filename)
	}

	if !strings.HasSuffix(filename, ".csv") {
		t.Errorf("Expected filename to end with '.csv', got: %s", filename)
	}

	// Check that it contains a timestamp-like string
	parts := strings.Split(filename, "_")
	if len(parts) < 3 {
		t.Errorf("Expected filename to have timestamp format, got: %s", filename)
	}

	// Verify the timestamp portion is reasonable (should be current time)
	timestampPart := strings.TrimSuffix(strings.Join(parts[2:], "_"), ".csv")
	if len(timestampPart) != 19 { // YYYY-MM-DD_HH-mm-ss format
		t.Errorf("Expected timestamp format YYYY-MM-DD_HH-mm-ss, got: %s", timestampPart)
	}
}

func TestFormatValueForCSV(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"nil_value", nil, ""},
		{"string_value", "hello", "hello"},
		{"int_value", 42, "42"},
		{"float_value", 3.14, "3.14"},
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
		{"byte_slice", []byte("bytes"), "bytes"},
		{"time_value", time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC), "2023-01-15 10:30:00 +0000 UTC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValueForCSV(tt.value)
			if result != tt.expected {
				t.Errorf("formatValueForCSV(%v) = %q, expected %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestSaveQueryResultToCSV(t *testing.T) {
	tempDir := t.TempDir()

	columnNames := []string{"id", "name", "status"}
	rows := [][]interface{}{
		{1, "Alice", true},
		{2, "Bob", false},
	}

	tests := []struct {
		name           string
		inputFilename  string
		expectedSuffix string
	}{
		{
			name:           "with_filename",
			inputFilename:  "custom_results",
			expectedSuffix: "custom_results.csv",
		},
		{
			name:           "with_csv_extension",
			inputFilename:  "results.csv",
			expectedSuffix: "results.csv",
		},
		{
			name:           "empty_filename_uses_default",
			inputFilename:  "",
			expectedSuffix: ".csv", // Will have timestamp prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change to temp directory for this test
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current working directory: %v", err)
			}
			
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalWd); err != nil {
					t.Errorf("Failed to restore working directory: %v", err)
				}
			}()

			savedPath, err := SaveQueryResultToCSV(columnNames, rows, tt.inputFilename)
			if err != nil {
				t.Fatalf("SaveQueryResultToCSV failed: %v", err)
			}

			// Check that the file exists
			if _, err := os.Stat(savedPath); os.IsNotExist(err) {
				t.Fatalf("CSV file was not created at: %s", savedPath)
			}

			// Check that the filename ends with expected suffix
			if !strings.HasSuffix(savedPath, tt.expectedSuffix) {
				t.Errorf("Expected path to end with %q, got: %s", tt.expectedSuffix, savedPath)
			}

			// Verify the content is correct
			content, err := os.ReadFile(savedPath)
			if err != nil {
				t.Fatalf("Failed to read CSV file: %v", err)
			}

			expectedContent := "id,name,status\n1,Alice,true\n2,Bob,false\n"
			if string(content) != expectedContent {
				t.Errorf("CSV content mismatch.\nExpected:\n%q\nGot:\n%q", expectedContent, string(content))
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_simple_filename",
			filename:    "results.csv",
			expectError: false,
		},
		{
			name:        "valid_filename_with_path",
			filename:    "data/results.csv",
			expectError: false,
		},
		{
			name:        "valid_absolute_path",
			filename:    "/tmp/results.csv",
			expectError: false,
		},
		{
			name:        "path_traversal_attack",
			filename:    "../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "path_traversal_with_csv",
			filename:    "../../../tmp/malicious.csv",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "current_directory",
			filename:    ".",
			expectError: true,
			errorMsg:    "invalid filename",
		},
		{
			name:        "empty_filename",
			filename:    "",
			expectError: true,
			errorMsg:    "invalid filename",
		},
		{
			name:        "relative_path_with_dots",
			filename:    "./results.csv",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.filename)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for filename %q, but got none", tt.filename)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for filename %q, but got: %v", tt.filename, err)
				}
			}
		})
	}
}

func TestSaveCSVError(t *testing.T) {
	t.Run("invalid_path_traversal", func(t *testing.T) {
		// Test path traversal protection
		maliciousPath := "../../../etc/passwd"
		
		err := SaveCSV([]string{"col1"}, [][]interface{}{{"data"}}, maliciousPath)
		if err == nil {
			t.Error("Expected error for path traversal attempt, but got none")
		}
		
		if !strings.Contains(err.Error(), "path traversal detected") {
			t.Errorf("Expected path traversal error, got: %v", err)
		}
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		// Test with invalid path (directory that doesn't exist)
		invalidPath := "/nonexistent/directory/test.csv"
		
		err := SaveCSV([]string{"col1"}, [][]interface{}{{"data"}}, invalidPath)
		if err == nil {
			t.Error("Expected error when saving to invalid path, but got none")
		}
		
		if !strings.Contains(err.Error(), "failed to create file") {
			t.Errorf("Expected error message about file creation, got: %v", err)
		}
	})

	t.Run("empty_filename", func(t *testing.T) {
		err := SaveCSV([]string{"col1"}, [][]interface{}{{"data"}}, "")
		if err == nil {
			t.Error("Expected error for empty filename, but got none")
		}
		
		if !strings.Contains(err.Error(), "invalid filename") {
			t.Errorf("Expected invalid filename error, got: %v", err)
		}
	})
}

func TestSaveQueryResultToCSVSecurity(t *testing.T) {
	columnNames := []string{"id", "name"}
	rows := [][]interface{}{
		{1, "Alice"},
		{2, "Bob"},
	}

	t.Run("path_traversal_protection", func(t *testing.T) {
		maliciousPath := "../../../etc/passwd"
		
		_, err := SaveQueryResultToCSV(columnNames, rows, maliciousPath)
		if err == nil {
			t.Error("Expected error for path traversal attempt, but got none")
		}
		
		if !strings.Contains(err.Error(), "path traversal detected") {
			t.Errorf("Expected path traversal error, got: %v", err)
		}
	})

	t.Run("path_traversal_with_csv_extension", func(t *testing.T) {
		// Test that adding .csv extension doesn't bypass validation
		maliciousPath := "../../../tmp/malicious"  // .csv will be added automatically
		
		_, err := SaveQueryResultToCSV(columnNames, rows, maliciousPath)
		if err == nil {
			t.Error("Expected error for path traversal attempt, but got none")
		}
		
		if !strings.Contains(err.Error(), "path traversal detected") {
			t.Errorf("Expected path traversal error, got: %v", err)
		}
	})
}
