package display

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// validateFilePath validates that the filename is safe and doesn't contain path traversal attempts
func validateFilePath(filename string) error {
	// Clean the path to resolve any relative components
	cleanPath := filepath.Clean(filename)
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return errors.New("path traversal detected: filename cannot contain '..'")
	}
	
	// Check for absolute paths that might escape the current directory
	if filepath.IsAbs(cleanPath) {
		// Allow absolute paths but warn that they should be used carefully
		// In a production environment, you might want to restrict this further
	}
	
	// Check for empty filename
	if cleanPath == "" || cleanPath == "." {
		return errors.New("invalid filename: cannot be empty or current directory")
	}
	
	return nil
}

// SaveCSV saves query results to a CSV file
func SaveCSV(columnNames []string, allRows [][]interface{}, filename string) error {
	// Validate the file path for security
	if err := validateFilePath(filename); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	
	// Create the output file
	file, err := os.Create(filepath.Clean(filename))
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header row
	if err := writer.Write(columnNames); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, row := range allRows {
		// Convert all values to strings
		stringRow := make([]string, len(row))
		for i, value := range row {
			stringRow[i] = formatValueForCSV(value)
		}

		if err := writer.Write(stringRow); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// GenerateDefaultCSVFilename creates a default filename with timestamp
func GenerateDefaultCSVFilename() string {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return fmt.Sprintf("pgbabble_results_%s.csv", timestamp)
}

// formatValueForCSV converts various Go types to string for CSV output
func formatValueForCSV(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// SaveQueryResultToCSV is a convenience function for saving query results
func SaveQueryResultToCSV(columnNames []string, allRows [][]interface{}, filename string) (string, error) {
	// Use default filename if not provided
	if filename == "" {
		filename = GenerateDefaultCSVFilename()
	}

	// Ensure .csv extension
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		filename += ".csv"
	}

	// Validate the file path for security (after processing filename)
	if err := validateFilePath(filename); err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	// Clean the filename
	cleanFilename := filepath.Clean(filename)

	// Get absolute path for display
	absPath, err := filepath.Abs(cleanFilename)
	if err != nil {
		absPath = cleanFilename // fallback to relative path
	}

	// Save the CSV
	if err := SaveCSV(columnNames, allRows, cleanFilename); err != nil {
		return "", err
	}

	return absPath, nil
}
