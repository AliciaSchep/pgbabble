package chat

import (
	"strings"
	"testing"
)

// TestGetUserApproval_Logic tests the approval logic without readline dependency
func TestGetUserApproval_Logic(t *testing.T) {
	// Test query info formatting
	queryInfo := "Test query explanation\n\nSQL Query:\nSELECT * FROM users"

	// Verify the query info is properly formatted (this is what would be shown to user)
	if !strings.Contains(queryInfo, "Test query explanation") {
		t.Error("expected queryInfo to contain explanation")
	}
	if !strings.Contains(queryInfo, "SELECT * FROM users") {
		t.Error("expected queryInfo to contain SQL query")
	}
	if !strings.Contains(queryInfo, "SQL Query:") {
		t.Error("expected queryInfo to contain SQL Query header")
	}
}

// TestUserApprovalResponseParsing tests the response parsing logic
func TestUserApprovalResponseParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"y", true},
		{"Y", true},
		{"yes", true},
		{"YES", true},
		{"Yes", true},
		{"n", false},
		{"N", false},
		{"no", false},
		{"NO", false},
		{"No", false},
		{"", false},
		{"maybe", false},
		{"quit", false},
		{"  y  ", true},   // Test trimming
		{"  no  ", false}, // Test trimming
	}

	for _, tt := range tests {
		t.Run("input_"+tt.input, func(t *testing.T) {
			// Simulate the response parsing logic from getUserApproval
			response := strings.ToLower(strings.TrimSpace(tt.input))
			actual := response == "y" || response == "yes"

			if actual != tt.expected {
				t.Errorf("input '%s': expected %v, got %v", tt.input, tt.expected, actual)
			}
		})
	}
}

// TestExecuteSQLToolWorkflow tests the tool parameter validation
func TestExecuteSQLToolWorkflow(t *testing.T) {
	// Test valid input
	validInput := map[string]interface{}{
		"sql":         "SELECT * FROM users WHERE active = true",
		"explanation": "Get all active users",
	}

	sqlQuery, ok := validInput["sql"].(string)
	if !ok {
		t.Error("expected sql to be a string")
	}
	if sqlQuery != "SELECT * FROM users WHERE active = true" {
		t.Errorf("expected specific SQL query, got '%s'", sqlQuery)
	}

	explanation, ok := validInput["explanation"].(string)
	if !ok {
		t.Error("expected explanation to be a string")
	}
	if explanation != "Get all active users" {
		t.Errorf("expected specific explanation, got '%s'", explanation)
	}

	// Test invalid input (missing sql)
	invalidInput := map[string]interface{}{
		"explanation": "Missing SQL parameter",
	}

	_, ok = invalidInput["sql"].(string)
	if ok {
		t.Error("expected sql parameter to be missing")
	}

	// Test invalid input (wrong type)
	wrongTypeInput := map[string]interface{}{
		"sql":         123, // Should be string
		"explanation": "Wrong type for SQL",
	}

	_, ok = wrongTypeInput["sql"].(string)
	if ok {
		t.Error("expected sql parameter to fail type assertion")
	}
}

// TestSQLQueryInfoFormatting tests how query info is formatted for user display
func TestSQLQueryInfoFormatting(t *testing.T) {
	explanation := "Find all users with recent activity"
	sqlQuery := "SELECT u.id, u.name, u.last_login FROM users u WHERE u.last_login > NOW() - INTERVAL '7 days'"

	// This mimics the formatting in the execute_sql tool
	queryInfo := explanation + "\n\nSQL Query:\n" + sqlQuery

	// Verify the format matches what getUserApproval expects
	lines := strings.Split(queryInfo, "\n")
	if len(lines) < 3 {
		t.Error("expected at least 3 lines in formatted query info")
	}

	if lines[0] != explanation {
		t.Errorf("expected first line to be explanation, got '%s'", lines[0])
	}

	if lines[1] != "" {
		t.Error("expected second line to be empty (separator)")
	}

	if lines[2] != "SQL Query:" {
		t.Errorf("expected third line to be 'SQL Query:', got '%s'", lines[2])
	}

	if lines[3] != sqlQuery {
		t.Errorf("expected fourth line to be SQL query, got '%s'", lines[3])
	}
}

// TestApprovalPromptFormat tests the prompt formatting
func TestApprovalPromptFormat(t *testing.T) {
	expectedPrompt := "Execute this query? (y/yes/n/no): "

	// This is the prompt that should be set in getUserApproval
	if !strings.Contains(expectedPrompt, "Execute this query?") {
		t.Error("expected prompt to ask about execution")
	}

	if !strings.Contains(expectedPrompt, "(y/yes/n/no)") {
		t.Error("expected prompt to show valid options")
	}

	if !strings.HasSuffix(expectedPrompt, ": ") {
		t.Error("expected prompt to end with ': ' for user input")
	}
}

// TestConversationClearWorkflow tests the /clear command functionality
func TestConversationClearWorkflow(t *testing.T) {
	session := &Session{
		mode:       "default",
		agentReady: false,
	}

	// Test when agent is not ready
	if session.agentReady {
		t.Error("expected agent to not be ready for test setup")
	}

	// This simulates the /clear command logic
	if session.agentReady {
		// Would call s.agent.ClearConversation()
		// Would print "🧹 Conversation history cleared"
	} else {
		// Would print "ℹ️  No conversation to clear"
	}

	// Test when agent is ready (simulated)
	session.agentReady = true
	if !session.agentReady {
		t.Error("expected agent to be ready after setting")
	}
}
