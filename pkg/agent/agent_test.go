package agent

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestNewAgent(t *testing.T) {
	// Test with valid API key
	agent, err := NewAgent("test-api-key", "default")
	if err != nil {
		t.Errorf("unexpected error creating agent: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if len(agent.tools) != 0 {
		t.Error("expected empty tools on creation")
	}
	if len(agent.conversation) != 0 {
		t.Error("expected empty conversation on creation")
	}

	// Test with empty API key but environment variable set
	if err := os.Setenv("ANTHROPIC_API_KEY", "env-api-key"); err != nil {
		t.Fatalf("failed to set environment variable: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("ANTHROPIC_API_KEY"); err != nil {
			t.Logf("failed to unset environment variable: %v", err)
		}
	}()

	agent2, err := NewAgent("", "default")
	if err != nil {
		t.Errorf("unexpected error creating agent with env var: %v", err)
	}
	if agent2 == nil {
		t.Error("expected non-nil agent with env var")
	}

	// Test with no API key
	if err := os.Unsetenv("ANTHROPIC_API_KEY"); err != nil {
		t.Logf("failed to unset environment variable: %v", err)
	}
	agent3, err := NewAgent("", "default")
	if err == nil {
		t.Error("expected error when no API key provided")
	}
	if agent3 != nil {
		t.Error("expected nil agent when no API key")
	}
}

func TestAgent_AddTool(t *testing.T) {
	agent := &Agent{
		tools:        []ToolDefinition{},
		conversation: []anthropic.MessageParam{},
	}

	toolDef := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
		},
		Function: func(input json.RawMessage) (string, error) {
			return "test result", nil
		},
	}

	initialCount := len(agent.tools)
	agent.AddTool(toolDef)

	if len(agent.tools) != initialCount+1 {
		t.Errorf("expected %d tools, got %d", initialCount+1, len(agent.tools))
	}

	addedTool := agent.tools[len(agent.tools)-1]
	if addedTool.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", addedTool.Name)
	}
	if addedTool.Description != "A test tool" {
		t.Errorf("expected tool description 'A test tool', got '%s'", addedTool.Description)
	}
}

func TestAgent_ClearConversation(t *testing.T) {
	agent := &Agent{
		tools: []ToolDefinition{},
		conversation: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("test message")),
		},
	}

	// Verify conversation has content
	if len(agent.conversation) == 0 {
		t.Error("expected conversation to have content for test setup")
	}

	agent.ClearConversation()

	if len(agent.conversation) != 0 {
		t.Errorf("expected empty conversation after clear, got %d messages", len(agent.conversation))
	}
}

func TestConvertToolToDefinition(t *testing.T) {
	// Create a test tool
	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool for conversion",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "Test parameter",
				},
			},
			Required: []string{"param1"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{
				Content: "test result",
				IsError: false,
			}, nil
		},
	}

	// Convert to ToolDefinition
	toolDef := ConvertToolToDefinition(tool)

	// Verify basic properties
	if toolDef.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got '%s'", toolDef.Name)
	}
	if toolDef.Description != "A test tool for conversion" {
		t.Errorf("expected description 'A test tool for conversion', got '%s'", toolDef.Description)
	}

	// Verify schema conversion
	if toolDef.InputSchema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", toolDef.InputSchema.Type)
	}
	if len(toolDef.InputSchema.Required) != 1 || toolDef.InputSchema.Required[0] != "param1" {
		t.Errorf("expected required field 'param1', got %v", toolDef.InputSchema.Required)
	}

	// Test function execution with valid input
	testInput := json.RawMessage(`{"param1": "test_value"}`)
	result, err := toolDef.Function(testInput)
	if err != nil {
		t.Errorf("unexpected error executing converted tool: %v", err)
	}
	if result != "test result" {
		t.Errorf("expected result 'test result', got '%s'", result)
	}
}

func TestConvertToolToDefinition_ErrorHandling(t *testing.T) {
	// Test tool that returns an error
	errorTool := &Tool{
		Name:        "error_tool",
		Description: "A tool that returns errors",
		InputSchema: ToolSchema{Type: "object"},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{
				Content: "Tool execution failed",
				IsError: true,
			}, nil
		},
	}

	toolDef := ConvertToolToDefinition(errorTool)

	// Test function execution with error result
	testInput := json.RawMessage(`{}`)
	result, err := toolDef.Function(testInput)
	if err == nil {
		t.Error("expected error when tool returns IsError=true")
	}
	if result != "" {
		t.Errorf("expected empty result on error, got '%s'", result)
	}

	// Test with invalid JSON input
	invalidInput := json.RawMessage(`{invalid json}`)
	_, err = toolDef.Function(invalidInput)
	if err == nil {
		t.Error("expected error with invalid JSON input")
	}
}

func TestAgent_executeTool(t *testing.T) {
	agent := &Agent{
		tools: []ToolDefinition{
			{
				Name:        "test_tool",
				Description: "Test tool",
				Function: func(input json.RawMessage) (string, error) {
					return "success", nil
				},
			},
		},
		conversation: []anthropic.MessageParam{},
	}

	// Test successful tool execution
	input := json.RawMessage(`{"test": "value"}`)
	result := agent.executeTool("test-id", "test_tool", input)

	// Verify the result structure
	if result.OfToolResult == nil {
		t.Error("expected OfToolResult to be set")
	}
	if result.OfToolResult.ToolUseID != "test-id" {
		t.Errorf("expected ToolUseID 'test-id', got '%s'", result.OfToolResult.ToolUseID)
	}
	// For successful execution, IsError should be false or not set
	if len(result.OfToolResult.Content) == 0 {
		t.Error("expected content to be set")
	}

	// Test tool not found - should return error structure
	result = agent.executeTool("test-id", "nonexistent_tool", input)
	if result.OfToolResult == nil {
		t.Error("expected OfToolResult to be set for error case")
	}
	if result.OfToolResult.ToolUseID != "test-id" {
		t.Errorf("expected ToolUseID 'test-id' for error case, got '%s'", result.OfToolResult.ToolUseID)
	}
}
