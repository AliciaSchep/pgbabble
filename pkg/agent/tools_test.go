package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalToolsToJSON(t *testing.T) {
	// Create test tools
	tools := []*Tool{
		{
			Name:        "test_tool_1",
			Description: "First test tool",
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
				return &ToolResult{Content: "test"}, nil
			},
		},
		{
			Name:        "test_tool_2", 
			Description: "Second test tool",
			InputSchema: ToolSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
				Required:   []string{},
			},
			Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
				return &ToolResult{Content: "test2"}, nil
			},
		},
	}

	// Test JSON marshaling
	jsonStr, err := MarshalToolsToJSON(tools)
	if err != nil {
		t.Errorf("unexpected error marshaling tools: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Verify JSON contains expected tool names
	if !strings.Contains(jsonStr, "test_tool_1") {
		t.Error("expected JSON to contain 'test_tool_1'")
	}
	if !strings.Contains(jsonStr, "test_tool_2") {
		t.Error("expected JSON to contain 'test_tool_2'")
	}

	// Verify it's valid JSON by unmarshaling
	var result []interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Errorf("failed to unmarshal generated JSON: %v", err)
	}
}

func TestToolResult_Basic(t *testing.T) {
	// Test successful result
	result := &ToolResult{
		Content: "success message",
		IsError: false,
	}

	if result.Content != "success message" {
		t.Errorf("expected content 'success message', got '%s'", result.Content)
	}
	if result.IsError {
		t.Error("expected IsError to be false")
	}

	// Test error result
	errorResult := &ToolResult{
		Content: "error message",
		IsError: true,
	}

	if !errorResult.IsError {
		t.Error("expected IsError to be true")
	}
}

func TestToolSchema_Basic(t *testing.T) {
	schema := ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"param1": map[string]interface{}{
				"type":        "string",
				"description": "Test parameter",
			},
			"param2": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
		},
		Required: []string{"param1"},
	}

	if schema.Type != "object" {
		t.Errorf("expected type 'object', got '%s'", schema.Type)
	}

	if len(schema.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(schema.Properties))
	}

	if len(schema.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(schema.Required))
	}

	if schema.Required[0] != "param1" {
		t.Errorf("expected required field 'param1', got '%s'", schema.Required[0])
	}
}

func TestTool_BasicExecution(t *testing.T) {
	tool := &Tool{
		Name:        "echo_tool",
		Description: "Echoes input back",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"message"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			message, ok := input["message"].(string)
			if !ok {
				return &ToolResult{
					Content: "Invalid message type",
					IsError: true,
				}, nil
			}
			return &ToolResult{
				Content: "Echo: " + message,
				IsError: false,
			}, nil
		},
	}

	ctx := context.Background()

	// Test successful execution
	input := map[string]interface{}{
		"message": "hello world",
	}
	result, err := tool.Handler(ctx, input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected successful execution")
	}
	if !strings.Contains(result.Content, "hello world") {
		t.Errorf("expected result to contain 'hello world', got '%s'", result.Content)
	}

	// Test with invalid input
	invalidInput := map[string]interface{}{
		"message": 123, // Invalid type
	}
	result, err = tool.Handler(ctx, invalidInput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result with invalid input")
	}
}