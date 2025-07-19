package display

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SaveCSV saves query results to a CSV file
func SaveCSV(columnNames []string, allRows [][]interface{}, filename string) error {
	// Create the output file
	file, err := os.Create(filename)
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

	// Get absolute path for display
	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename // fallback to relative path
	}

	// Save the CSV
	if err := SaveCSV(columnNames, allRows, filename); err != nil {
		return "", err
	}

	return absPath, nil
}