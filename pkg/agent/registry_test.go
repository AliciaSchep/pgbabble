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

	if err := registry.RegisterTool(tool); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

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

	if err := registry.RegisterTool(tool); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

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

	if err := registry.RegisterTool(tool1); err != nil {
		t.Fatalf("failed to register tool1: %v", err)
	}
	if err := registry.RegisterTool(tool2); err != nil {
		t.Fatalf("failed to register tool2: %v", err)
	}

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

	if err := registry.RegisterTool(tool); err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

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

	if err := registry.RegisterTool(tool1); err != nil {
		t.Fatalf("failed to register tool1: %v", err)
	}
	if err := registry.RegisterTool(tool2); err != nil {
		t.Fatalf("failed to register tool2: %v", err)
	}

	// Clear registry
	registry.Clear()

	// Verify empty
	names := registry.ListToolNames()
	if len(names) != 0 {
		t.Errorf("expected 0 tools after clear, got %d", len(names))
	}
}

func TestToolRegistry_GetAllTools(t *testing.T) {
	registry := NewToolRegistry()

	// Test empty registry
	tools := registry.GetAllTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools in empty registry, got %d", len(tools))
	}

	// Add some tools
	tool1 := &Tool{
		Name:        "list_tables",
		Description: "List database tables",
		InputSchema: ToolSchema{Type: "object"},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "table1, table2"}, nil
		},
	}
	tool2 := &Tool{
		Name:        "describe_table",
		Description: "Describe table structure",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"table": map[string]interface{}{"type": "string"},
			},
			Required: []string{"table"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "column details"}, nil
		},
	}

	if err := registry.RegisterTool(tool1); err != nil {
		t.Fatalf("failed to register tool1: %v", err)
	}
	if err := registry.RegisterTool(tool2); err != nil {
		t.Fatalf("failed to register tool2: %v", err)
	}

	// Get all tools
	tools = registry.GetAllTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// Verify tools are returned correctly
	toolMap := make(map[string]*Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	if tool, exists := toolMap["list_tables"]; !exists {
		t.Error("expected 'list_tables' in returned tools")
	} else {
		if tool.Description != "List database tables" {
			t.Errorf("expected description 'List database tables', got '%s'", tool.Description)
		}
	}

	if tool, exists := toolMap["describe_table"]; !exists {
		t.Error("expected 'describe_table' in returned tools")
	} else {
		if tool.Description != "Describe table structure" {
			t.Errorf("expected description 'Describe table structure', got '%s'", tool.Description)
		}
		if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "table" {
			t.Errorf("expected required field 'table', got %v", tool.InputSchema.Required)
		}
	}
}

func TestToolRegistry_GetToolsForAnthropic(t *testing.T) {
	registry := NewToolRegistry()

	// Test empty registry
	anthropicTools := registry.GetToolsForAnthropic()
	if len(anthropicTools) != 0 {
		t.Errorf("expected 0 tools for empty registry, got %d", len(anthropicTools))
	}

	// Add tools with different schema complexities
	simpleTool := &Tool{
		Name:        "simple_tool",
		Description: "A simple tool",
		InputSchema: ToolSchema{Type: "object"},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "simple"}, nil
		},
	}

	complexTool := &Tool{
		Name:        "complex_tool",
		Description: "A complex tool with parameters",
		InputSchema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "SQL query to execute",
				},
				"limit": map[string]interface{}{
					"type":    "integer",
					"minimum": 1,
					"maximum": 1000,
					"default": 100,
				},
			},
			Required: []string{"query"},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: "complex result"}, nil
		},
	}

	if err := registry.RegisterTool(simpleTool); err != nil {
		t.Fatalf("failed to register simple tool: %v", err)
	}
	if err := registry.RegisterTool(complexTool); err != nil {
		t.Fatalf("failed to register complex tool: %v", err)
	}

	// Get tools formatted for Anthropic
	anthropicTools = registry.GetToolsForAnthropic()
	if len(anthropicTools) != 2 {
		t.Errorf("expected 2 anthropic tools, got %d", len(anthropicTools))
	}

	// Verify tool conversion
	toolMap := make(map[string]map[string]interface{})
	for _, tool := range anthropicTools {
		if name, ok := tool["name"].(string); ok {
			toolMap[name] = tool
		}
	}

	// Check simple tool
	if simpleDef, exists := toolMap["simple_tool"]; !exists {
		t.Error("expected 'simple_tool' in anthropic tools")
	} else {
		if desc, ok := simpleDef["description"].(string); !ok || desc != "A simple tool" {
			t.Errorf("expected description 'A simple tool', got '%v'", simpleDef["description"])
		}
		if inputSchema, ok := simpleDef["input_schema"].(ToolSchema); ok {
			if inputSchema.Type != "object" {
				t.Errorf("expected schema type 'object', got '%s'", inputSchema.Type)
			}
		} else {
			t.Error("expected input_schema to be ToolSchema type")
		}
	}

	// Check complex tool
	if complexDef, exists := toolMap["complex_tool"]; !exists {
		t.Error("expected 'complex_tool' in anthropic tools")
	} else {
		if desc, ok := complexDef["description"].(string); !ok || desc != "A complex tool with parameters" {
			t.Errorf("expected description 'A complex tool with parameters', got '%v'", complexDef["description"])
		}
		if inputSchema, ok := complexDef["input_schema"].(ToolSchema); ok {
			if inputSchema.Type != "object" {
				t.Errorf("expected schema type 'object', got '%s'", inputSchema.Type)
			}
			
			// Check properties were converted
			if inputSchema.Properties == nil {
				t.Error("expected properties to be converted")
			}
			
			// Check required fields were converted
			if len(inputSchema.Required) != 1 || inputSchema.Required[0] != "query" {
				t.Errorf("expected required field 'query', got %v", inputSchema.Required)
			}
		} else {
			t.Error("expected input_schema to be ToolSchema type")
		}
	}
}
