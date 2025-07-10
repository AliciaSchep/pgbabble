package agent

import (
	"context"
	"encoding/json"
)

// Tool represents a function that the LLM can call
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema ToolSchema  `json:"input_schema"`
	Handler     ToolHandler `json:"-"`
}

// ToolSchema defines the JSON schema for tool input
type ToolSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// ToolHandler is a function that executes a tool
type ToolHandler func(ctx context.Context, input map[string]interface{}) (*ToolResult, error)

// ToolResult represents the result of tool execution
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

// Message represents a conversation message
type Message struct {
	Role    string      `json:"role"`    // "user", "assistant"
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ContentBlock represents a piece of message content
type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result"
	Text string `json:"text,omitempty"`
	
	// For tool_use
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Input    map[string]interface{} `json:"input,omitempty"`
	
	// For tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ConversationHistory manages the conversation state
type ConversationHistory struct {
	Messages []Message `json:"messages"`
}

// AddUserMessage adds a user message to the conversation
func (ch *ConversationHistory) AddUserMessage(content string) {
	ch.Messages = append(ch.Messages, Message{
		Role:    "user",
		Content: content,
	})
}

// AddAssistantMessage adds an assistant message to the conversation
func (ch *ConversationHistory) AddAssistantMessage(content interface{}) {
	ch.Messages = append(ch.Messages, Message{
		Role:    "assistant",
		Content: content,
	})
}

// AddToolResult adds a tool result to the conversation
func (ch *ConversationHistory) AddToolResult(toolUseID string, result *ToolResult) {
	// Find the last assistant message and add tool result
	if len(ch.Messages) > 0 {
		lastMsg := &ch.Messages[len(ch.Messages)-1]
		if lastMsg.Role == "assistant" {
			// Convert content to []ContentBlock if it's not already
			var blocks []ContentBlock
			
			switch content := lastMsg.Content.(type) {
			case string:
				blocks = []ContentBlock{{Type: "text", Text: content}}
			case []ContentBlock:
				blocks = content
			default:
				// Try to unmarshal from JSON
				if jsonBytes, err := json.Marshal(content); err == nil {
					json.Unmarshal(jsonBytes, &blocks)
				}
			}
			
			// Add tool result
			blocks = append(blocks, ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolUseID,
				Content:   result.Content,
				IsError:   result.IsError,
			})
			
			lastMsg.Content = blocks
		}
	}
}

// GetLastUserMessage returns the last user message content
func (ch *ConversationHistory) GetLastUserMessage() string {
	for i := len(ch.Messages) - 1; i >= 0; i-- {
		if ch.Messages[i].Role == "user" {
			if content, ok := ch.Messages[i].Content.(string); ok {
				return content
			}
		}
	}
	return ""
}

// Clear resets the conversation history
func (ch *ConversationHistory) Clear() {
	ch.Messages = nil
}