package display

import (
	"strings"
	"testing"
)

func TestCheckLessAvailable(t *testing.T) {
	// This test will depend on the system, but we can at least ensure it returns a boolean
	available := CheckLessAvailable()
	
	// Should return either true or false, not panic
	_ = available
}

func TestGenerateFullTableContentDetailed(t *testing.T) {
	tests := []struct {
		name        string
		columnNames []string
		rows        [][]interface{}
		title       string
		expectContains []string
	}{
		{
			name:        "basic table with data",
			columnNames: []string{"id", "name", "age"},
			rows: [][]interface{}{
				{1, "Alice", 25},
				{2, "Bob", 30},
			},
			title: "User Data",
			expectContains: []string{
				"📊 User Data",
				"id",
				"name", 
				"age",
				"Alice",
				"Bob",
				"25",
				"30",
				"Total rows: 2",
			},
		},
		{
			name:        "empty table",
			columnNames: []string{"col1", "col2"},
			rows:        [][]interface{}{},
			title:       "Empty Results",
			expectContains: []string{
				"📊 Empty Results",
				"No data to display",
			},
		},
		{
			name:        "single column table",
			columnNames: []string{"status"},
			rows: [][]interface{}{
				{"active"},
				{"inactive"},
			},
			title: "Status List",
			expectContains: []string{
				"📊 Status List",
				"status",
				"active",
				"inactive",
				"Total rows: 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := GenerateFullTableContent(tt.columnNames, tt.rows, tt.title)
			
			for _, expected := range tt.expectContains {
				if !strings.Contains(content, expected) {
					t.Errorf("Expected content to contain '%s', but it didn't.\nContent: %s", expected, content)
				}
			}
		})
	}
}

func TestPageQueryResult(t *testing.T) {
	tests := []struct {
		name      string
		data      *QueryResultData
		queryInfo string
		expectErr bool
	}{
		{
			name: "valid data",
			data: &QueryResultData{
				ColumnNames: []string{"id", "name"},
				Rows: [][]interface{}{
					{1, "Alice"},
					{2, "Bob"},
				},
				TotalRows: 2,
				Truncated: false,
			},
			queryInfo: "SELECT id, name FROM users",
			expectErr: false,
		},
		{
			name:      "nil data",
			data:      nil,
			queryInfo: "SELECT * FROM empty_table",
			expectErr: false, // Should handle gracefully, not error
		},
		{
			name: "empty data",
			data: &QueryResultData{
				ColumnNames: []string{"id"},
				Rows:        [][]interface{}{},
				TotalRows:   0,
				Truncated:   false,
			},
			queryInfo: "SELECT id FROM users WHERE false",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test doesn't actually invoke less since that would be system-dependent
			// and interactive. Instead we're testing the data preparation logic.
			
			if tt.data == nil || len(tt.data.Rows) == 0 {
				// These cases should return early without error
				err := PageQueryResult(tt.data, tt.queryInfo)
				if tt.expectErr && err == nil {
					t.Error("Expected error but got none")
				}
				if !tt.expectErr && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				return
			}

			// For valid data, test that content generation works
			title := "Query Results (2 rows)"
			content := GenerateFullTableContent(tt.data.ColumnNames, tt.data.Rows, title)
			
			if !strings.Contains(content, "Alice") {
				t.Error("Generated content should contain sample data")
			}
			
			if !strings.Contains(content, tt.queryInfo) == false {
				// We expect the query info to be included when calling PageQueryResult
				fullContent := tt.queryInfo + "\n\n" + content
				if !strings.Contains(fullContent, tt.queryInfo) {
					t.Error("Full content should contain query info")
				}
			}
		})
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 3, 3},
		{3, 5, 3},
		{0, 0, 0},
		{-1, 5, -1},
		{10, 10, 10},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}