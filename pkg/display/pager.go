package display

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PageResult sends formatted table data to less for interactive viewing
func PageResult(title string, content string) error {
	// Check if less is available
	if _, err := exec.LookPath("less"); err != nil {
		fmt.Println("⚠️  'less' command not found, displaying all content:")
		fmt.Print(content)
		return nil
	}

	// Create the command
	cmd := exec.Command("less", "-S", "-R")
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command and wait for completion
	fmt.Printf("📖 Opening %s in less (press 'q' to exit)...\n", title)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running less: %w", err)
	}

	fmt.Println("✅ Returned from less viewer")
	return nil
}

// GenerateFullTableContent creates complete formatted table content for paging
func GenerateFullTableContent(columnNames []string, allRows [][]interface{}, title string) string {
	var content strings.Builder

	// Add title
	content.WriteString(fmt.Sprintf("📊 %s\n", title))
	content.WriteString(strings.Repeat("=", len(title)+4) + "\n\n")

	if len(allRows) == 0 {
		content.WriteString("No data to display.\n")
		return content.String()
	}

	// Create table formatter
	formatter := NewTableFormatter(columnNames)

	// Analyze sample data (first 10 rows for performance)
	sampleSize := min(len(allRows), 10)
	formatter.AnalyzeData(allRows[:sampleSize])

	// Calculate column widths
	widths := formatter.CalculateColumnWidths()

	// Generate header
	content.WriteString(formatter.FormatHeader(widths))

	// Generate all rows
	for _, row := range allRows {
		content.WriteString(formatter.FormatRow(row, widths))
	}

	// Add footer with row count
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Total rows: %d\n", len(allRows)))

	return content.String()
}

// QueryResultData represents the structure for query results (re-defined here to avoid import cycles)
type QueryResultData struct {
	ColumnNames []string
	Rows        [][]interface{}
	TotalRows   int
	Truncated   bool
}

// PageQueryResult is a convenience function for paging query results
func PageQueryResult(data *QueryResultData, queryInfo string) error {
	if data == nil || len(data.Rows) == 0 {
		fmt.Println("No data to browse.")
		return nil
	}

	title := fmt.Sprintf("Query Results (%d rows)", data.TotalRows)
	content := GenerateFullTableContent(data.ColumnNames, data.Rows, title)

	// Add query info to the top
	fullContent := fmt.Sprintf("Query: %s\n\n%s", queryInfo, content)

	return PageResult(title, fullContent)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CheckLessAvailable returns true if the less command is available
func CheckLessAvailable() bool {
	_, err := exec.LookPath("less")
	return err == nil
}

// PageWithContext is a context-aware version of PageResult
func PageWithContext(ctx context.Context, title string, content string) error {
	if !CheckLessAvailable() {
		fmt.Println("⚠️  'less' command not found, displaying all content:")
		fmt.Print(content)
		return nil
	}

	cmd := exec.CommandContext(ctx, "less", "-S", "-R")
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("📖 Opening %s in less (press 'q' to exit)...\n", title)
	
	if err := cmd.Run(); err != nil {
		// Check if it was cancelled due to context
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("error running less: %w", err)
	}

	fmt.Println("✅ Returned from less viewer")
	return nil
}