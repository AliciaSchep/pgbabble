# PGBabble Roadmap

## Priority Levels
- **P0 (Critical)**: Core functionality improvements needed for stability and basic usability
- **P1 (High)**: Important features that significantly improve user experience
- **P2 (Medium)**: Nice-to-have features that add value but aren't essential

---

## P0 - Critical Priority

### Improved conversation handling
* **Context awareness**: Add information about current "mode" to the system prompt
* **Mode change notifications**: After changing "mode", send a message to agent to inform it
* **Smart table exploration**: Update system prompt to recommend doing list_tables rather than jumping into describe_table unless a specific table name has been given
* **Query flow control**: Prevent agent from doing two query or explain tool calls in a row without user input
* **Enhanced LLM feedback**: Improve messages sent back to LLM after tool calls with richer context
* **Conversation state tracking**: Maintain context about previous queries and results within the session
* **Error context preservation**: When queries fail, maintain conversation context to help LLM understand what went wrong

### Result limiting
* **CLI argument**: Add ROWS_LIMIT argument that defaults to 1000 for number of rows to limit fetching
* **Database-level enforcement**: Apply LIMIT clause at database level for efficiency
* **Unlimited option**: Allow limit to be set to 0 for no restriction
* **Runtime modification**: Add `/limit` slash command to modify limit in-session
* **Smart pagination**: Implement pagination for large result sets with navigation controls
* **Memory management**: Ensure large result sets don't consume excessive memory
* **Progressive loading**: Load results in chunks for better responsiveness with very large datasets

### Improved Testing
* **Coverage analysis**: Use coverage tools to identify untested code paths and aim for >80% coverage
* **Test quality review**: Review current tests to identify areas of improvement and test reliability
* **Integration tests**: Add end-to-end tests that verify complete workflows with real databases
* **Unit test expansion**: Increase unit test coverage for core modules (agent, chat, db, display)
* **Error scenario testing**: Add tests for edge cases, connection failures, and malformed queries
* **Performance testing**: Add benchmarks for query processing and result formatting
* **LLM interaction mocking**: Create robust mocks for LLM interactions to ensure deterministic testing

### Error handling & recovery
* **Connection resilience**: Implement automatic reconnection on database connection loss
* **Graceful degradation**: Handle LLM API failures with fallback options or manual mode
* **User-friendly error messages**: Convert technical database errors into helpful guidance
* **Query validation**: Pre-validate SQL before execution to catch syntax errors early
* **Timeout handling**: Implement configurable timeouts for long-running queries with cancellation
* **Transaction safety**: Ensure proper transaction rollback on errors
* **Network error recovery**: Handle network interruptions gracefully with retry mechanisms

### CI workflow  
* **Basic automation**: Setup GitHub Actions to run test suite on all PRs and pushes
* **Cross-platform builds**: Build binaries for Linux, macOS, and Windows
* **Security scanning**: Add vulnerability scanning for dependencies
* **Release automation**: Automate release creation with tagged versions
* **Artifact publishing**: Publish binaries to GitHub releases
* **Code quality checks**: Integrate linting and formatting checks
* **Performance regression testing**: Run benchmarks to detect performance issues

---

## P1 - High Priority

### Multi-provider LLM support
* **Provider abstraction**: Create a unified interface for different LLM providers
* **OpenAI API support**: Add support for OpenAI GPT models via API
* **Local LLM support**: Integration with llama.cpp server for local model inference
* **Provider configuration**: Allow users to configure API keys and endpoints per provider
* **Fallback providers**: Support multiple providers with automatic fallback on failures
* **Model selection**: Allow users to choose specific models within a provider
* **Cost optimization**: Track token usage and costs across different providers

### Add option to save
* **CSV export**: Add `/save csv` command to export current result set to CSV format
* **JSON export**: Add `/save json` command for structured data export
* **TSV export**: Add `/save tsv` command for tab-separated values
* **Custom delimiters**: Allow custom field separators for export formats
* **Header options**: Option to include/exclude column headers in exports
* **File naming**: Smart default file naming with timestamps and query context
* **Append mode**: Option to append results to existing files

---

## P2 - Medium Priority

### Add manual sql option with autocomplete
* **SQL mode command**: Add `/sql` command to enter manual SQL editing mode
* **Tab completion**: Implement tab completion for table names, column names, and SQL keywords
* **Syntax highlighting**: Add basic syntax highlighting for SQL queries in terminal
* **Query validation**: Real-time validation of SQL syntax as user types
* **Schema-aware completion**: Context-aware suggestions based on current database schema
* **History integration**: Access previous SQL queries using arrow keys
* **Multi-line editing**: Support for complex multi-line SQL queries with proper indentation

### Performance optimizations
* **Schema caching**: Cache database schema information to reduce repeated metadata queries
* **Query result caching**: Implement optional caching for identical queries with configurable TTL
* **Connection pooling**: Reuse database connections to reduce connection overhead
* **Lazy loading**: Load table/column details only when needed for large schemas
* **Query optimization hints**: Provide suggestions for improving slow queries
* **Memory usage optimization**: Minimize memory footprint for large result sets
* **Startup performance**: Optimize initial connection and schema discovery time

### Query history
* **Persistent storage**: Save query history to local file system with encryption
* **Session browsing**: Command to browse previous queries and results from current session
* **Historical search**: Search through query history by keywords or patterns
* **Result replay**: Re-execute previous queries with `/replay <query_id>` command
* **Export history**: Export query history to various formats for backup
* **History cleanup**: Commands to manage and clean up old history entries
* **Cross-session persistence**: Maintain history across different pgbabble sessions
