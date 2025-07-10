package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Agent implements the simple pattern from ampcode.com
type Agent struct {
	client *anthropic.Client
	tools  []ToolDefinition
}

// ToolDefinition matches the ampcode.com pattern
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema anthropic.ToolInputSchemaParam
	Function    func(input json.RawMessage) (string, error)
}

// NewAgent creates a new agent following the ampcode.com pattern
func NewAgent(apiKey string) (*Agent, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required (set ANTHROPIC_API_KEY environment variable)")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Agent{
		client: &client,
		tools:  []ToolDefinition{},
	}, nil
}

// AddTool adds a tool to the agent
func (a *Agent) AddTool(tool ToolDefinition) {
	a.tools = append(a.tools, tool)
}

// SendMessage sends a message and handles tool calling (simplified from the ampcode.com pattern)
func (a *Agent) SendMessage(ctx context.Context, userMessage string) (string, error) {
	conversation := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
	}

	systemMessage := `You are a PostgreSQL expert assistant that helps users write SQL queries.

IMPORTANT: You have access to database schema tools. Always use these tools FIRST to understand the database structure before writing SQL queries.

Available tools:
- list_tables: See all tables and views in the database  
- describe_table: Get detailed information about a specific table including columns and types
- get_relationships: Find foreign key relationships for a table
- search_columns: Find columns matching a pattern across tables

Always start by using list_tables or describe_table to understand the database structure, then generate accurate SQL queries based on the actual schema.`

	for {
		message, err := a.runInference(ctx, conversation, systemMessage)
		if err != nil {
			return "", err
		}
		conversation = append(conversation, message.ToParam())

		var textResponse string
		toolResults := []anthropic.ContentBlockParamUnion{}
		
		for _, content := range message.Content {
			switch content.Type {
			case "text":
				textResponse += content.Text
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}

		// If no tools were used, return the text response
		if len(toolResults) == 0 {
			return textResponse, nil
		}

		// Add tool results and continue the conversation
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
	}
}

// runInference makes the API call (from ampcode.com pattern)
func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam, systemMessage string) (*anthropic.Message, error) {
	// Convert tools to Anthropic format
	anthropicTools := make([]anthropic.ToolUnionParam, len(a.tools))
	for i, tool := range a.tools {
		anthropicTools[i] = anthropic.ToolUnionParamOfTool(tool.InputSchema, tool.Name)
	}

	params := anthropic.MessageNewParams{
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 4000,
		System: []anthropic.TextBlockParam{
			{Text: systemMessage},
		},
		Messages: conversation,
	}

	if len(anthropicTools) > 0 {
		params.Tools = anthropicTools
	}

	return a.client.Messages.New(ctx, params)
}

// executeTool executes a tool (adapted from ampcode.com pattern for current SDK)
func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	var toolDef ToolDefinition
	var found bool
	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}
	if !found {
		// Create a tool result block for error case
		return anthropic.ContentBlockParamUnion{
			OfToolResult: &anthropic.ToolResultBlockParam{
				ToolUseID: id,
				IsError:   anthropic.Bool(true),
				Content: []anthropic.ToolResultBlockParamContentUnion{
					{OfText: &anthropic.TextBlockParam{Text: "Tool not found"}},
				},
			},
		}
	}

	response, err := toolDef.Function(input)
	if err != nil {
		// Create a tool result block for error case
		return anthropic.ContentBlockParamUnion{
			OfToolResult: &anthropic.ToolResultBlockParam{
				ToolUseID: id,
				IsError:   anthropic.Bool(true),
				Content: []anthropic.ToolResultBlockParamContentUnion{
					{OfText: &anthropic.TextBlockParam{Text: err.Error()}},
				},
			},
		}
	}
	
	// Create a tool result block for success case with the actual response
	return anthropic.ContentBlockParamUnion{
		OfToolResult: &anthropic.ToolResultBlockParam{
			ToolUseID: id,
			IsError:   anthropic.Bool(false),
			Content: []anthropic.ToolResultBlockParamContentUnion{
				{OfText: &anthropic.TextBlockParam{Text: response}},
			},
		},
	}
}

// ConvertToolToDefinition converts our Tool to ToolDefinition format
func ConvertToolToDefinition(tool *Tool) ToolDefinition {
	return ToolDefinition{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: anthropic.ToolInputSchemaParam{
			Type:       "object",
			Properties: tool.InputSchema.Properties,
			Required:   tool.InputSchema.Required,
		},
		Function: func(input json.RawMessage) (string, error) {
			// Convert JSON to map for our tool handler
			var inputMap map[string]interface{}
			if err := json.Unmarshal(input, &inputMap); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}

			// Call our tool handler
			result, err := tool.Handler(context.Background(), inputMap)
			if err != nil {
				return "", err
			}

			if result.IsError {
				return "", fmt.Errorf(result.Content)
			}

			return result.Content, nil
		},
	}
}