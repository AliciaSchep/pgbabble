# PGBabble Roadmap

## P1 - High Priority

### Add option to save results
* **CSV export**: Add `/save` command to export current result set to CSV format
* **File naming**: Smart default file naming with timestamps and query context, but option to give a file name `/save <path>`


### Multi-provider LLM support
* **Provider abstraction**: Create a unified interface for different LLM providers
* **OpenAI API support**: Add support for OpenAI GPT models via API
* **Local LLM support**: Integration with llama.cpp server for local model inference
* **Provider configuration**: Allow users to configure API keys and endpoints per provider
* **Fallback providers**: Support multiple providers with automatic fallback on failures
* **Model selection**: Allow users to choose specific models within a provider
* **Cost optimization**: Track token usage and costs across different providers

---

## P2 - Medium Priority

### Turn limiting
* **Query flow control**: Harder restriction on agent calling tool right away after rejection
* **General limiting**: Set a limit for number of consecutive tool calls

### Result limiting
* **CLI argument**: Add ROWS_LIMIT argument that defaults to 1000 for number of rows to limit fetching
* **Database-level enforcement**: Apply LIMIT clause at database level for efficiency
* **Unlimited option**: Allow limit to be set to 0 for no restriction
* **Runtime modification**: Add `/limit` slash command to modify limit in-session
* **Smart pagination**: Implement pagination for large result sets with navigation controls
* **Memory management**: Ensure large result sets don't consume excessive memory
* **Progressive loading**: Load results in chunks for better responsiveness with very large datasets

### Add manual sql option with autocomplete
* **SQL mode command**: Add `/sql` command to enter manual SQL editing mode
* **Tab completion**: Implement tab completion for table names, column names, and SQL keywords
* **Syntax highlighting**: Add basic syntax highlighting for SQL queries in terminal
* **Query validation**: Real-time validation of SQL syntax as user types
* **Schema-aware completion**: Context-aware suggestions based on current database schema
* **History integration**: Access previous SQL queries using arrow keys
* **Multi-line editing**: Support for complex multi-line SQL queries with proper indentation

---

## P3 - Low Priority

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
