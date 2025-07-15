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
	client       *anthropic.Client
	tools        []ToolDefinition
	conversation []anthropic.MessageParam
	mode         string
}

// ToolDefinition matches the ampcode.com pattern
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema anthropic.ToolInputSchemaParam
	Function    func(input json.RawMessage) (string, error)
}

// NewAgent creates a new agent following the ampcode.com pattern
func NewAgent(apiKey string, mode string) (*Agent, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required (set ANTHROPIC_API_KEY environment variable)")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Agent{
		client:       &client,
		tools:        []ToolDefinition{},
		conversation: []anthropic.MessageParam{},
		mode:         mode,
	}, nil
}

// AddTool adds a tool to the agent
func (a *Agent) AddTool(tool ToolDefinition) {
	a.tools = append(a.tools, tool)
}

// ClearConversation clears the conversation history
func (a *Agent) ClearConversation() {
	a.conversation = []anthropic.MessageParam{}
}

// generateSystemMessage creates a system message that includes mode-specific information
func (a *Agent) generateSystemMessage() string {
	var modeDescription string
	switch a.mode {
	case "default":
		modeDescription = `IMPORTANT -- Running queries will display results to the user, but not share them with you.
DO NOT MAKE UP results when talking with user. You can see schema information, table sizes, and query execution metadata, but NOT actual query result data. You can examine EXPLAIN query plans to help with optimization.
If the user asks for a result from a prior query, you should clarify that you cannot see the results, but that they can use the "/browse" command to view the results from last query
or you can run the same or a modified query again for them to see the results.
`
	case "schema-only":
		modeDescription = `IMPORTANT -- Running queries will display results to the user, but not share them with you.
DO NOT MAKE UP results when talking with user. You can see schema information but NOT actual query result data, table sizes, or explain execution plans.
If the user asks for a result from a prior query, you should clarify that you cannot see the results, but that they can use the "/browse" command to view the results from last query
or you can run the same or a modified query again for them to see the results.
`
	default:
		modeDescription = ""
	}

	return fmt.Sprintf(`You are a PostgreSQL expert assistant that helps users write SQL queries.

%s

The user can change this mode by restarting the session with a different --mode flag, but cannot change it during this session.

CRITICAL: You MUST use the available tools to interact with the database. Never just describe SQL - always use tools.

Available tools:
- list_tables: See all tables and views in the database
- describe_table: Get detailed information about a specific table including columns and types
- get_relationships: Find foreign key relationships for a table
- search_columns: Find columns matching a pattern across tables
- execute_sql: Execute a SQL query after user approval
- explain_query: Analyze query execution plans for performance optimization

MANDATORY Workflow:
1. Unless the user has provided specific table names, ALWAYS start by calling list_tables to understand the database or search_columns to understand what tables to focus on.
2. Use describe_table and get_relationships to better understand tables and relationships
3. Generate SQL based on actual schema information
4. ALWAYS call execute_sql tool to run queries - never just show SQL text
5. Let the tool handle user approval and execution
6. For performance questions or complex queries, use explain_query to analyze execution plans
7. If a SQL query or explain execution is rejected by the user, always ask for clarification before proposing another sql
query to execute
8. Don't run multiple queries in a row without checking in with the user in between each query.

Use a conversational tone, do not mention specific tool names.
Do NOT provide raw SQL in text. Use execute_sql tool for all query execution.`, modeDescription)
}

// SendMessage sends a message and handles tool calling (simplified from the ampcode.com pattern)
func (a *Agent) SendMessage(ctx context.Context, userMessage string) (string, error) {
	// Add user message to conversation history
	a.conversation = append(a.conversation, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	systemMessage := a.generateSystemMessage()

	for {
		message, err := a.runInference(ctx, a.conversation, systemMessage)
		if err != nil {
			return "", err
		}
		a.conversation = append(a.conversation, message.ToParam())

		var textResponse string
		toolResults := []anthropic.ContentBlockParamUnion{}

		for _, content := range message.Content {
			switch content.Type {
			case "text":
				textResponse += content.Text
			case "tool_use":
				fmt.Printf("🛠️  LLM called tool: %s\n", content.Name)
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}

		// If no tools were used, return the text response
		if len(toolResults) == 0 {
			return textResponse, nil
		}

		// Add tool results and continue the conversation
		a.conversation = append(a.conversation, anthropic.NewUserMessage(toolResults...))
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
				return "", fmt.Errorf("%s", result.Content)
			}

			return result.Content, nil
		},
	}
}
