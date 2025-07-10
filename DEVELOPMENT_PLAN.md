# PGBabble Development Plan

## Phase 1: Foundation & Core Architecture (Weeks 1-2)
**Goal**: Establish CLI with psql-style connection handling and interactive chat

### 1.1 Project Setup & Dependencies
- Initialize Go module with minimal dependencies:
  - `github.com/jackc/pgx/v5` - PostgreSQL driver
  - `github.com/spf13/cobra` - CLI framework  
  - `github.com/anthropics/anthropic-sdk-go` - Anthropic client
  - `github.com/chzyer/readline` - Interactive terminal input
- Set up clean project structure

### 1.2 Connection Management (TDD)
- **Tests First**: Connection parsing and validation tests
- Implement psql-compatible connection handling:
  - Parse `postgresql://` URIs 
  - Support individual flags: `--host`, `--port`, `--user`, `--password`, `--dbname`
  - Environment variable fallbacks: `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`
  - Connection validation on startup
- Graceful error handling for connection failures

### 1.3 Interactive Chat Interface
- **Tests First**: Chat loop and input handling tests
- Implement interactive readline-based chat session
- Basic prompt: `pgbabble> `
- Commands: `/quit`, `/help`, `/schema`
- Message history management

### 1.4 Database Schema Layer
- **Tests First**: Schema inspection tests with mock DB
- Build PostgreSQL schema inspection:
  - List tables/views/functions
  - Describe table structure (columns, types, constraints)
  - Foreign key relationships
  - Index information

## Phase 2: LLM Integration & Agent Core (Weeks 3-4)
**Goal**: Integrate Anthropic API with tool-based SQL generation

### 2.1 LLM Integration
- **Tests First**: Mock LLM client for testing
- Implement Anthropic client wrapper
- Tool registry system following ampcode.com patterns
- Conversation context management

### 2.2 Schema Tools for Agent
- **Tests First**: Tool execution and formatting tests
- Schema inspection tools for LLM:
  - `describe_table(table_name)` 
  - `list_tables()` 
  - `get_relationships(table_name)`
  - `search_columns(pattern)`
- Format schema info for LLM context

### 2.3 SQL Generation Flow
- **Tests First**: SQL generation and review workflow tests
- Natural language → SQL generation
- Present generated SQL for user review
- User approval flow: approve/reject/iterate
- Track conversation context for iterations

## Phase 3: Query Execution & Results (Weeks 5-6)
**Goal**: Execute approved queries with privacy-first result handling

### 3.1 Query Execution Engine
- **Tests First**: Safe execution and result handling tests
- Execute user-approved SQL queries
- Return only success/failure + error details to LLM
- No query result data sent to LLM by default
- Resource limits and timeouts

### 3.2 Result Display
- **Tests First**: Table formatting and display tests
- Simple ASCII table output for users
- Handle various PostgreSQL data types
- Pagination for large result sets
- Export options (future: CSV, JSON)

### 3.3 Safety & Error Handling
- **Tests First**: Validation and safety tests
- Basic SQL validation (prevent DROP/DELETE without confirmation)
- User-friendly error messages
- Graceful connection recovery

## Phase 4: Enhanced Data Modes Foundation (Week 7)
**Goal**: Architecture for summary_data and full_data modes

### 4.1 Mode System
- **Tests First**: Mode switching and configuration tests
- Command-line flags: `--mode=default|summary_data|full_data`
- Runtime mode switching: `/mode summary_data`
- Mode-specific tool availability

### 4.2 Summary Data Mode
- **Tests First**: Summary calculation tests
- Additional tools for LLM in summary_data mode:
  - `get_row_count(table_name)`
  - `get_column_stats(table_name, column_name)` (cardinality, nulls)
  - `sample_values(table_name, column_name, limit=5)`

## Phase 5: Testing Strategy
**Throughout all phases - TDD approach**

### 5.1 Unit Testing
- >90% test coverage target
- Mock external dependencies (DB, Anthropic API)
- Table-driven tests for multiple scenarios
- Error condition testing

### 5.2 Integration Testing  
- End-to-end CLI testing
- Real PostgreSQL integration (testcontainers)
- Recorded LLM response testing

### 5.3 Manual Testing
- Various PostgreSQL schema types
- Complex query generation scenarios
- User experience flow validation

## CLI Usage Examples
```bash
# Direct URI connection
pgbabble "postgresql://user:pass@localhost/mydb"

# Individual parameters
pgbabble --host localhost --port 5432 --user myuser --dbname mydb

# Environment variables (like psql)
export PGHOST=localhost PGUSER=myuser PGDATABASE=mydb
pgbabble

# With enhanced mode
pgbabble --mode=summary_data "postgresql://user:pass@localhost/mydb"
```

## Interactive Session Flow
```
$ pgbabble "postgresql://user:pass@localhost/mydb"
Connected to PostgreSQL database: mydb
Type /help for commands, /quit to exit

pgbabble> Show me all customers who made orders last month
Generated SQL:
SELECT c.name, c.email, COUNT(o.id) as order_count 
FROM customers c 
JOIN orders o ON c.id = o.customer_id 
WHERE o.created_at >= date_trunc('month', CURRENT_DATE - interval '1 month')
  AND o.created_at < date_trunc('month', CURRENT_DATE)
GROUP BY c.id, c.name, c.email;

Approve this query? (y/n/iterate): y
Executing query...
┌──────────────┬─────────────────────┬─────────────┐
│ name         │ email               │ order_count │
├──────────────┼─────────────────────┼─────────────┤
│ John Smith   │ john@example.com    │ 3           │
│ Jane Doe     │ jane@example.com    │ 1           │
└──────────────┴─────────────────────┴─────────────┘
Query executed successfully (2 rows)

pgbabble> 
```

## Key Architecture Principles
1. **psql-compatible connection handling** for familiar UX
2. **Interactive chat-first design** with immediate DB connection
3. **Privacy-first with explicit data exposure controls**
4. **Test-driven development** with comprehensive coverage
5. **Tool-based LLM integration** for extensibility
6. **Minimal, battle-tested dependencies**

## Future Enhancements (Beyond MVP)
- Local LLM support via go-llama.cpp
- Advanced output formatting (JSON, CSV export)
- Full data mode implementation
- Query history and favorites
- Advanced schema analysis tools
- Performance optimization
- Multiple LLM provider support