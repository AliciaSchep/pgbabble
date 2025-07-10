package agent

import (
	"context"
	"fmt"
	"sync"
)

// ToolRegistry manages available tools for the LLM
type ToolRegistry struct {
	tools map[string]*Tool
	mutex sync.RWMutex
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

// RegisterTool adds a tool to the registry
func (tr *ToolRegistry) RegisterTool(tool *Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}
	
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	
	tr.tools[tool.Name] = tool
	return nil
}

// GetTool retrieves a tool by name
func (tr *ToolRegistry) GetTool(name string) (*Tool, bool) {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()
	
	tool, exists := tr.tools[name]
	return tool, exists
}

// GetAllTools returns all registered tools
func (tr *ToolRegistry) GetAllTools() []*Tool {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()
	
	tools := make([]*Tool, 0, len(tr.tools))
	for _, tool := range tr.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolsForAnthropic returns tools in Anthropic API format
func (tr *ToolRegistry) GetToolsForAnthropic() []map[string]interface{} {
	tools := tr.GetAllTools()
	anthropicTools := make([]map[string]interface{}, len(tools))
	
	for i, tool := range tools {
		anthropicTools[i] = map[string]interface{}{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.InputSchema,
		}
	}
	
	return anthropicTools
}

// ExecuteTool executes a tool by name with given input
func (tr *ToolRegistry) ExecuteTool(ctx context.Context, name string, input map[string]interface{}) (*ToolResult, error) {
	tool, exists := tr.GetTool(name)
	if !exists {
		return &ToolResult{
			Content: fmt.Sprintf("Tool '%s' not found", name),
			IsError: true,
		}, fmt.Errorf("tool '%s' not found", name)
	}
	
	// Execute the tool
	result, err := tool.Handler(ctx, input)
	if err != nil {
		return &ToolResult{
			Content: fmt.Sprintf("Tool execution failed: %v", err),
			IsError: true,
		}, err
	}
	
	return result, nil
}

// ListToolNames returns a list of all tool names
func (tr *ToolRegistry) ListToolNames() []string {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()
	
	names := make([]string, 0, len(tr.tools))
	for name := range tr.tools {
		names = append(names, name)
	}
	return names
}

// RemoveTool removes a tool from the registry
func (tr *ToolRegistry) RemoveTool(name string) bool {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	
	if _, exists := tr.tools[name]; exists {
		delete(tr.tools, name)
		return true
	}
	return false
}

// Clear removes all tools from the registry
func (tr *ToolRegistry) Clear() {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	
	tr.tools = make(map[string]*Tool)
}