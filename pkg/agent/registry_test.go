package agent

import (
	"context"
	"testing"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	if registry == nil {
		t.Error("expected non-nil registry")
	}

	if len(registry.ListToolNames()) != 0 {
		t.Error("expected empty registry on creation")
	}
}

func TestToolRegistry_RegisterTool(t *testing.T) {
	registry := NewToolRegistry()

	// Test registering a valid tool
	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: ToolSchema{Type: "object"},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "test"}, nil
		},
	}

	err := registry.RegisterTool(tool)
	if err != nil {
		t.Errorf("unexpected error registering tool: %v", err)
	}

	// Test registering nil tool
	err = registry.RegisterTool(nil)
	if err == nil {
		t.Error("expected error registering nil tool")
	}

	// Test registering tool with empty name
	invalidTool := &Tool{
		Name:        "",
		Description: "Invalid tool",
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "test"}, nil
		},
	}

	err = registry.RegisterTool(invalidTool)
	if err == nil {
		t.Error("expected error registering tool with empty name")
	}

	// Test registering tool with nil handler
	invalidTool2 := &Tool{
		Name:        "invalid_tool",
		Description: "Invalid tool",
		Handler:     nil,
	}

	err = registry.RegisterTool(invalidTool2)
	if err == nil {
		t.Error("expected error registering tool with nil handler")
	}
}

func TestToolRegistry_GetTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: ToolSchema{Type: "object"},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "test"}, nil
		},
	}

	registry.RegisterTool(tool)

	// Test getting existing tool
	retrieved, exists := registry.GetTool("test_tool")
	if !exists {
		t.Error("expected tool to exist")
	}
	if retrieved.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", retrieved.Name)
	}

	// Test getting non-existing tool
	_, exists = registry.GetTool("nonexistent")
	if exists {
		t.Error("expected tool not to exist")
	}
}

func TestToolRegistry_ExecuteTool(t *testing.T) {
	registry := NewToolRegistry()
	ctx := context.Background()

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: ToolSchema{Type: "object"},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "test result"}, nil
		},
	}

	registry.RegisterTool(tool)

	// Test executing existing tool
	result, err := registry.ExecuteTool(ctx, "test_tool", map[string]interface{}{})
	if err != nil {
		t.Errorf("unexpected error executing tool: %v", err)
	}
	if result.Content != "test result" {
		t.Errorf("expected result content 'test result', got '%s'", result.Content)
	}

	// Test executing non-existing tool
	result, err = registry.ExecuteTool(ctx, "nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("expected error executing non-existent tool")
	}
	if result == nil || !result.IsError {
		t.Error("expected error result for non-existent tool")
	}
}

func TestToolRegistry_ListToolNames(t *testing.T) {
	registry := NewToolRegistry()

	// Test empty registry
	names := registry.ListToolNames()
	if len(names) != 0 {
		t.Errorf("expected 0 tool names, got %d", len(names))
	}

	// Add tools
	tool1 := &Tool{
		Name:    "tool1",
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) { return nil, nil },
	}
	tool2 := &Tool{
		Name:    "tool2",
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) { return nil, nil },
	}

	registry.RegisterTool(tool1)
	registry.RegisterTool(tool2)

	names = registry.ListToolNames()
	if len(names) != 2 {
		t.Errorf("expected 2 tool names, got %d", len(names))
	}

	// Check names are present (order doesn't matter)
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["tool1"] {
		t.Error("expected 'tool1' in tool names")
	}
	if !nameMap["tool2"] {
		t.Error("expected 'tool2' in tool names")
	}
}

func TestToolRegistry_RemoveTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := &Tool{
		Name:    "test_tool",
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) { return nil, nil },
	}

	registry.RegisterTool(tool)

	// Test removing existing tool
	removed := registry.RemoveTool("test_tool")
	if !removed {
		t.Error("expected tool to be removed")
	}

	// Verify tool is gone
	_, exists := registry.GetTool("test_tool")
	if exists {
		t.Error("expected tool to be removed from registry")
	}

	// Test removing non-existing tool
	removed = registry.RemoveTool("nonexistent")
	if removed {
		t.Error("expected false when removing non-existent tool")
	}
}

func TestToolRegistry_Clear(t *testing.T) {
	registry := NewToolRegistry()

	// Add some tools
	tool1 := &Tool{
		Name:    "tool1",
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) { return nil, nil },
	}
	tool2 := &Tool{
		Name:    "tool2",
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) { return nil, nil },
	}

	registry.RegisterTool(tool1)
	registry.RegisterTool(tool2)

	// Clear registry
	registry.Clear()

	// Verify empty
	names := registry.ListToolNames()
	if len(names) != 0 {
		t.Errorf("expected 0 tools after clear, got %d", len(names))
	}
}
