package agent

import (
	"context"
	"encoding/json"
	"testing"
)

// Test Tool type functionality
func TestTool_Structure(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"param1": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"param1"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "test", IsError: false}, nil
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got '%s'", tool.Name)
	}

	if tool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%s'", tool.Description)
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", tool.InputSchema.Type)
	}

	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "param1" {
		t.Errorf("expected required field 'param1', got %v", tool.InputSchema.Required)
	}

	// Test handler execution
	result, err := tool.Handler(context.Background(), map[string]interface{}{"param1": "test"})
	if err != nil {
		t.Errorf("unexpected error from handler: %v", err)
	}
	if result.Content != "test" {
		t.Errorf("expected result content 'test', got '%s'", result.Content)
	}
	if result.IsError {
		t.Error("expected IsError to be false")
	}
}

// Test ToolSchema validation and structure
func TestToolSchema_Validation(t *testing.T) {
	schema := ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name parameter",
			},
			"age": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
		},
		Required: []string{"name"},
	}

	// Test JSON marshaling/unmarshaling
	jsonData, err := json.Marshal(schema)
	if err != nil {
		t.Errorf("failed to marshal schema: %v", err)
	}

	var unmarshaled ToolSchema
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Errorf("failed to unmarshal schema: %v", err)
	}

	if unmarshaled.Type != "object" {
		t.Errorf("expected type 'object', got '%s'", unmarshaled.Type)
	}

	if len(unmarshaled.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(unmarshaled.Properties))
	}

	if len(unmarshaled.Required) != 1 || unmarshaled.Required[0] != "name" {
		t.Errorf("expected required field 'name', got %v", unmarshaled.Required)
	}
}

// Test ToolResult structure and JSON handling
func TestToolResult_JSON(t *testing.T) {
	tests := []struct {
		name     string
		result   ToolResult
		expected string
	}{
		{
			name: "success result",
			result: ToolResult{
				Content: "operation successful",
				IsError: false,
			},
			expected: `{"content":"operation successful"}`,
		},
		{
			name: "error result",
			result: ToolResult{
				Content: "operation failed",
				IsError: true,
			},
			expected: `{"content":"operation failed","is_error":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.result)
			if err != nil {
				t.Errorf("failed to marshal result: %v", err)
			}

			if string(jsonData) != tt.expected {
				t.Errorf("expected JSON %s, got %s", tt.expected, string(jsonData))
			}

			// Test unmarshaling
			var unmarshaled ToolResult
			if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
				t.Errorf("failed to unmarshal result: %v", err)
			}

			if unmarshaled.Content != tt.result.Content {
				t.Errorf("expected content '%s', got '%s'", tt.result.Content, unmarshaled.Content)
			}

			if unmarshaled.IsError != tt.result.IsError {
				t.Errorf("expected IsError %v, got %v", tt.result.IsError, unmarshaled.IsError)
			}
		})
	}
}

// Test Message type structure
func TestMessage_Structure(t *testing.T) {
	tests := []struct {
		name    string
		message Message
	}{
		{
			name: "text message",
			message: Message{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
		{
			name: "content blocks message",
			message: Message{
				Role: "assistant",
				Content: []ContentBlock{
					{Type: "text", Text: "I'm doing well, thank you!"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.message)
			if err != nil {
				t.Errorf("failed to marshal message: %v", err)
			}

			// Test JSON unmarshaling
			var unmarshaled Message
			if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
				t.Errorf("failed to unmarshal message: %v", err)
			}

			if unmarshaled.Role != tt.message.Role {
				t.Errorf("expected role '%s', got '%s'", tt.message.Role, unmarshaled.Role)
			}
		})
	}
}

// Test ContentBlock variations
func TestContentBlock_Types(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
	}{
		{
			name: "text block",
			block: ContentBlock{
				Type: "text",
				Text: "This is a text block",
			},
		},
		{
			name: "tool_use block",
			block: ContentBlock{
				Type:  "tool_use",
				ID:    "tool-123",
				Name:  "search_tool",
				Input: map[string]interface{}{"query": "test"},
			},
		},
		{
			name: "tool_result block",
			block: ContentBlock{
				Type:      "tool_result",
				ToolUseID: "tool-123",
				Content:   "search completed",
				IsError:   false,
			},
		},
		{
			name: "tool_result error block",
			block: ContentBlock{
				Type:      "tool_result",
				ToolUseID: "tool-456",
				Content:   "search failed",
				IsError:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			jsonData, err := json.Marshal(tt.block)
			if err != nil {
				t.Errorf("failed to marshal content block: %v", err)
			}

			var unmarshaled ContentBlock
			if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
				t.Errorf("failed to unmarshal content block: %v", err)
			}

			if unmarshaled.Type != tt.block.Type {
				t.Errorf("expected type '%s', got '%s'", tt.block.Type, unmarshaled.Type)
			}

			// Verify type-specific fields
			switch tt.block.Type {
			case "text":
				if unmarshaled.Text != tt.block.Text {
					t.Errorf("expected text '%s', got '%s'", tt.block.Text, unmarshaled.Text)
				}
			case "tool_use":
				if unmarshaled.ID != tt.block.ID {
					t.Errorf("expected ID '%s', got '%s'", tt.block.ID, unmarshaled.ID)
				}
				if unmarshaled.Name != tt.block.Name {
					t.Errorf("expected name '%s', got '%s'", tt.block.Name, unmarshaled.Name)
				}
			case "tool_result":
				if unmarshaled.ToolUseID != tt.block.ToolUseID {
					t.Errorf("expected tool_use_id '%s', got '%s'", tt.block.ToolUseID, unmarshaled.ToolUseID)
				}
				if unmarshaled.Content != tt.block.Content {
					t.Errorf("expected content '%s', got '%s'", tt.block.Content, unmarshaled.Content)
				}
				if unmarshaled.IsError != tt.block.IsError {
					t.Errorf("expected is_error %v, got %v", tt.block.IsError, unmarshaled.IsError)
				}
			}
		})
	}
}

// Test ConversationHistory functionality
func TestConversationHistory_UserMessage(t *testing.T) {
	ch := &ConversationHistory{}

	// Test adding user message
	ch.AddUserMessage("Hello, world!")

	if len(ch.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(ch.Messages))
	}

	message := ch.Messages[0]
	if message.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", message.Role)
	}

	if content, ok := message.Content.(string); !ok || content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got %v", message.Content)
	}

	// Test GetLastUserMessage
	lastUser := ch.GetLastUserMessage()
	if lastUser != "Hello, world!" {
		t.Errorf("expected last user message 'Hello, world!', got '%s'", lastUser)
	}
}

func TestConversationHistory_AssistantMessage(t *testing.T) {
	ch := &ConversationHistory{}

	// Test adding assistant message with string content
	ch.AddAssistantMessage("Hello back!")

	if len(ch.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(ch.Messages))
	}

	message := ch.Messages[0]
	if message.Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", message.Role)
	}

	// Test adding assistant message with content blocks
	blocks := []ContentBlock{
		{Type: "text", Text: "I can help you with that."},
	}
	ch.AddAssistantMessage(blocks)

	if len(ch.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(ch.Messages))
	}
}

func TestConversationHistory_ToolResult(t *testing.T) {
	ch := &ConversationHistory{}

	// Add an assistant message first
	ch.AddAssistantMessage("I'll search for that.")

	// Add a tool result
	result := &ToolResult{
		Content: "Search completed successfully",
		IsError: false,
	}
	ch.AddToolResult("tool-123", result)

	if len(ch.Messages) != 1 {
		t.Errorf("expected 1 message (tool result added to existing), got %d", len(ch.Messages))
	}

	message := ch.Messages[0]
	blocks, ok := message.Content.([]ContentBlock)
	if !ok {
		t.Errorf("expected content to be []ContentBlock, got %T", message.Content)
		return
	}

	if len(blocks) != 2 {
		t.Errorf("expected 2 content blocks, got %d", len(blocks))
		return
	}

	// First block should be the original text
	if blocks[0].Type != "text" || blocks[0].Text != "I'll search for that." {
		t.Errorf("expected first block to be original text, got %+v", blocks[0])
	}

	// Second block should be the tool result
	if blocks[1].Type != "tool_result" {
		t.Errorf("expected second block type 'tool_result', got '%s'", blocks[1].Type)
	}
	if blocks[1].ToolUseID != "tool-123" {
		t.Errorf("expected tool_use_id 'tool-123', got '%s'", blocks[1].ToolUseID)
	}
	if blocks[1].Content != "Search completed successfully" {
		t.Errorf("expected content 'Search completed successfully', got '%s'", blocks[1].Content)
	}
	if blocks[1].IsError {
		t.Error("expected IsError to be false")
	}
}

func TestConversationHistory_Clear(t *testing.T) {
	ch := &ConversationHistory{}

	// Add some messages
	ch.AddUserMessage("Hello")
	ch.AddAssistantMessage("Hi there")
	ch.AddUserMessage("How are you?")

	if len(ch.Messages) != 3 {
		t.Errorf("expected 3 messages before clear, got %d", len(ch.Messages))
	}

	// Clear the conversation
	ch.Clear()

	if len(ch.Messages) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(ch.Messages))
	}

	// Test GetLastUserMessage after clear
	lastUser := ch.GetLastUserMessage()
	if lastUser != "" {
		t.Errorf("expected empty string after clear, got '%s'", lastUser)
	}
}

func TestConversationHistory_GetLastUserMessage(t *testing.T) {
	ch := &ConversationHistory{}

	// Test with no messages
	if lastUser := ch.GetLastUserMessage(); lastUser != "" {
		t.Errorf("expected empty string with no messages, got '%s'", lastUser)
	}

	// Add various messages
	ch.AddUserMessage("First message")
	ch.AddAssistantMessage("Response 1")
	ch.AddUserMessage("Second message")
	ch.AddAssistantMessage("Response 2")
	ch.AddUserMessage("Third message")

	// Should return the last user message
	lastUser := ch.GetLastUserMessage()
	if lastUser != "Third message" {
		t.Errorf("expected 'Third message', got '%s'", lastUser)
	}
}

func TestConversationHistory_ComplexContentHandling(t *testing.T) {
	ch := &ConversationHistory{}

	// Test with complex content that needs JSON marshaling/unmarshaling
	complexContent := map[string]interface{}{
		"type": "complex",
		"data": []interface{}{1, 2, 3},
	}
	ch.AddAssistantMessage(complexContent)

	result := &ToolResult{
		Content: "Processed complex data",
		IsError: false,
	}
	ch.AddToolResult("tool-456", result)

	if len(ch.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(ch.Messages))
	}

	message := ch.Messages[0]
	blocks, ok := message.Content.([]ContentBlock)
	if !ok {
		t.Errorf("expected content to be []ContentBlock, got %T", message.Content)
		return
	}

	if len(blocks) != 2 {
		t.Errorf("expected 2 content blocks, got %d", len(blocks))
		return
	}

	// First block should contain the complex content as text
	if blocks[0].Type != "text" {
		t.Errorf("expected first block type 'text', got '%s'", blocks[0].Type)
	}

	// Second block should be the tool result
	if blocks[1].Type != "tool_result" {
		t.Errorf("expected second block type 'tool_result', got '%s'", blocks[1].Type)
	}
}
