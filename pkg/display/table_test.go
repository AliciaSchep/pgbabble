package display

import (
	"strings"
	"testing"
)

func TestTableFormatter(t *testing.T) {
	columnNames := []string{"id", "name", "email"}
	formatter := NewTableFormatter(columnNames)

	// Test sample data analysis
	sampleData := [][]interface{}{
		{1, "Alice", "alice@example.com"},
		{2, "Bob", "bob@example.com"},
		{3, "Charlie", "charlie@example.com"},
	}

	formatter.AnalyzeData(sampleData)

	// Test column width calculation
	widths := formatter.CalculateColumnWidths()

	if len(widths) != 3 {
		t.Errorf("Expected 3 column widths, got %d", len(widths))
	}

	// All widths should be at least the minimum width
	for i, width := range widths {
		if width < formatter.columns[i].MinWidth {
			t.Errorf("Column %d width %d is less than minimum %d", i, width, formatter.columns[i].MinWidth)
		}
	}

	// Test header formatting
	header := formatter.FormatHeader(widths)
	if !strings.Contains(header, "id") || !strings.Contains(header, "name") || !strings.Contains(header, "email") {
		t.Error("Header should contain all column names")
	}

	// Test row formatting
	row := formatter.FormatRow(sampleData[0], widths)
	if !strings.Contains(row, "Alice") || !strings.Contains(row, "alice@example.com") {
		t.Error("Row should contain the data values")
	}
}

func TestTerminalWidth(t *testing.T) {
	width := GetTerminalWidth()
	if width <= 0 {
		t.Error("Terminal width should be positive")
	}

	// Should return at least the fallback width
	if width < 80 {
		t.Error("Terminal width should be at least 80 (fallback)")
	}
}

func TestGenerateFullTableContent(t *testing.T) {
	columnNames := []string{"id", "name"}
	rows := [][]interface{}{
		{1, "Alice"},
		{2, "Bob"},
	}

	content := GenerateFullTableContent(columnNames, rows, "Test Results")

	if !strings.Contains(content, "Test Results") {
		t.Error("Content should contain the title")
	}

	if !strings.Contains(content, "Alice") || !strings.Contains(content, "Bob") {
		t.Error("Content should contain the row data")
	}

	if !strings.Contains(content, "Total rows: 2") {
		t.Error("Content should contain row count")
	}
}
