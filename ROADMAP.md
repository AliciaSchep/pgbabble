# PGBabble Roadmap

### Improved conversation handling
* Add information about current "mode" to the system prompt
* After changing "mode", send a message to agent to inform it.
* Update system prompt to recommend doing list_tables rather than jumping into describe_table unless a specific table name has been given.
* Prevent agent from doing two query or explain tool calls in a row without user input.
* Improve messages send back to LLM after tool calls

### Result limiting

* Add an argument ROWS_LIMIT that default to 1000 for number of rows to limit fetching to
* Limit should be at database level
* Limit can be set to 0 for no limit
* Limit can be modified in-session with a slash command

### Improved Testing
* Analyze test coverage to find gaps in coverage
* Review currents tests to identify areas of improvement
* Add integration tests

### CI-worfklow
* Setup basic github action that will run test suite & build binaries

### Add option to save
* Add a slash command /save to enable saving result set to csv

### Multi-provider LLM support
* Initial version of project uses anthropic sdk. Make the project more flexible to be able to support a variety of OpenAI-compatible API providers, including local ones. In particular for local want to support use of llama.cpp server https://github.com/ggml-org/llama.cpp.

### Query history
- Save query history and give tools to browse earlier results and queries




### Advanced Features
- Local LLM support via go-llama.cpp
- Advanced output formatting (JSON, CSV export)
- Full data mode implementation
- Query history and favorites
- Advanced schema analysis tools
- Performance optimization
- Multiple LLM provider support