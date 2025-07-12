package display

import (
	"fmt"
	"strings"
)

// ColumnInfo represents information about a table column for formatting
type ColumnInfo struct {
	Name      string
	MaxWidth  int
	MinWidth  int
	IsNumeric bool
}

// TableFormatter handles intelligent table formatting with dynamic column widths
type TableFormatter struct {
	columns      []ColumnInfo
	terminalWidth int
	borderWidth   int // Space for borders and padding
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(columnNames []string) *TableFormatter {
	columns := make([]ColumnInfo, len(columnNames))
	for i, name := range columnNames {
		columns[i] = ColumnInfo{
			Name:      name,
			MaxWidth:  len(name), // Start with header width
			MinWidth:  5,         // Minimum readable width
			IsNumeric: false,     // Will be determined from data
		}
	}

	return &TableFormatter{
		columns:       columns,
		terminalWidth: GetTerminalWidth(),
		borderWidth:   len(columnNames)*3 + 1, // "| " before each column + "|" at end
	}
}

// AnalyzeData analyzes sample data to determine optimal column characteristics
func (tf *TableFormatter) AnalyzeData(sampleRows [][]interface{}) {
	for _, row := range sampleRows {
		for i, value := range row {
			if i >= len(tf.columns) {
				continue
			}

			valueStr := formatValue(value)
			valueLen := len(valueStr)

			// Update max width
			if valueLen > tf.columns[i].MaxWidth {
				tf.columns[i].MaxWidth = valueLen
			}

			// Try to detect numeric columns
			if !tf.columns[i].IsNumeric && isNumericValue(value) {
				tf.columns[i].IsNumeric = true
			}
		}
	}
}

// CalculateColumnWidths determines optimal column widths based on terminal size
func (tf *TableFormatter) CalculateColumnWidths() []int {
	availableWidth := tf.terminalWidth - tf.borderWidth
	totalColumns := len(tf.columns)

	if totalColumns == 0 {
		return []int{}
	}

	widths := make([]int, totalColumns)

	// First pass: assign minimum widths
	remainingWidth := availableWidth
	for i := range tf.columns {
		minWidth := max(tf.columns[i].MinWidth, len(tf.columns[i].Name))
		widths[i] = minWidth
		remainingWidth -= minWidth
	}

	// If we don't have enough space, truncate all columns equally
	if remainingWidth < 0 {
		evenWidth := availableWidth / totalColumns
		for i := range widths {
			widths[i] = max(3, evenWidth) // At least 3 characters
		}
		return widths
	}

	// Second pass: distribute remaining width based on content needs
	for remainingWidth > 0 {
		distributed := false
		for i := range tf.columns {
			if remainingWidth <= 0 {
				break
			}

			// Can this column use more space?
			maxUseful := tf.columns[i].MaxWidth
			if widths[i] < maxUseful {
				widths[i]++
				remainingWidth--
				distributed = true
			}
		}

		// If no column could use more space, break
		if !distributed {
			break
		}
	}

	return widths
}

// FormatHeader creates the table header
func (tf *TableFormatter) FormatHeader(widths []int) string {
	var header strings.Builder
	var separator strings.Builder

	header.WriteString("| ")
	separator.WriteString(strings.Repeat("-", tf.terminalWidth))

	for i, col := range tf.columns {
		if i >= len(widths) {
			continue
		}

		headerText := truncateString(col.Name, widths[i])
		if col.IsNumeric {
			headerText = padRight(headerText, widths[i])
		} else {
			headerText = padLeft(headerText, widths[i])
		}

		header.WriteString(headerText)
		header.WriteString(" | ")
	}

	return header.String() + "\n" + separator.String() + "\n"
}

// FormatRow formats a single data row
func (tf *TableFormatter) FormatRow(row []interface{}, widths []int) string {
	var result strings.Builder
	result.WriteString("| ")

	for i, value := range row {
		if i >= len(widths) || i >= len(tf.columns) {
			continue
		}

		valueStr := truncateString(formatValue(value), widths[i])
		if tf.columns[i].IsNumeric {
			valueStr = padRight(valueStr, widths[i])
		} else {
			valueStr = padLeft(valueStr, widths[i])
		}

		result.WriteString(valueStr)
		result.WriteString(" | ")
	}

	return result.String() + "\n"
}

// Helper functions

func formatValue(value interface{}) string {
	if value == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", value)
}

func isNumericValue(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	default:
		return false
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}