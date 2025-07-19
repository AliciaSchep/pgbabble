# PGBabble

A CLI tool for interacting with PostgreSQL databases using natural language queries powered by LLMs, while restricting what data is shared back to LLM and keeping a human-in-the-loop for query approval.

## Features

- Natural language to SQL conversion, but with human-in-the-loop for approving LLM-generated SQL before it is run.
- Privacy-first design (only metadata sent to LLM by default)
- Interactive chat interface
- psql-compatible connection handling
- Schema inspection and exploration

## Usage

### API Key Setup & model selection

Before using pgbabble, you need to set up your Anthropic API key:

```bash
export ANTHROPIC_API_KEY=your_api_key_here
```

You can get an API key from [Anthropic's Console](https://console.anthropic.com/).

**Note**: Currently, pgbabble only supports Anthropic's Claude models. We have plans to support other model providers (OpenAI, local models, etc.) in the future, but this has not yet been implemented.

The model can be specified using the `--model` flag with a valid anthropic model alias like `claude-sonnet-4-0`. The default is `claude-3-7-sonnet-latest`.

### Connection Examples

```bash
# Connect with URI
pgbabble "postgresql://user:pass@localhost/mydb"

# Connect with individual parameters
pgbabble --host localhost --port 5432 --user myuser --dbname mydb

# Use environment variables (like psql)
export PGHOST=localhost PGUSER=myuser PGDATABASE=mydb
pgbabble

# Specify data sharing mode (optional)
pgbabble --mode schema-only "postgresql://user:pass@localhost/mydb"
```

## Privacy Modes

PGBabble offers three privacy modes to control what information is shared with the LLM:

### `default` (default mode)
- ✅ **Schema information** (table names, column names, types)
- ✅ **EXPLAIN query plans** (for query optimization)
- ✅ **Table size estimates** (approximate row counts)
- ✅ **Query execution metadata** (row counts, execution time)
- ❌ **Actual query result data**

*Best for: General database exploration and query development with privacy protection*

### `schema-only` (maximum privacy)
- ✅ **Schema information** (table names, column names, types)
- ❌ **EXPLAIN query plans** (execution details hidden)
- ❌ **Table size estimates** (size information hidden)
- ❌ **Query execution metadata** (minimal feedback)
- ❌ **Actual query result data**

*Best for: Highly sensitive databases where even table size and query result counts should not be shared*

### `share-results` (full access)
- ✅ **Schema information** (table names, column names, types)
- ✅ **EXPLAIN query plans** (for query optimization)
- ✅ **Table size estimates** (approximate row counts)
- ✅ **Query execution metadata** (row counts, execution time)
- ✅ **Actual query result data** (limited to 50 rows per query)

*Best for: Development/testing environments where full data access is acceptable*

### Example Usage
```bash
# Maximum privacy mode
pgbabble --mode schema-only "postgresql://user:pass@localhost/mydb"

# Default balanced mode
pgbabble --mode default "postgresql://user:pass@localhost/mydb"

# Full data sharing mode
pgbabble --mode share-results "postgresql://user:pass@localhost/mydb"
```

## Quick Start with Sample Data

To test PGBabble with sample data, you can set up a PostgreSQL database with the LEGO dataset, which includes tables for sets, themes, parts, colors, and more.


### 1. Start PostgreSQL with Docker

```bash
# Start PostgreSQL 17 container
docker run --name postgres-lego \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_DB=postgres \
  -p 5432:5432 \
  -d postgres:17
```

### 2. Create and Load LEGO Database

```bash
# Wait a moment for PostgreSQL to start, then create the lego database
docker exec -it postgres-lego psql -U postgres -c "CREATE DATABASE lego;"

# Download the LEGO dataset
curl -O https://raw.githubusercontent.com/neondatabase/postgres-sample-dbs/main/lego.sql

# Load the data into the database
docker exec -i postgres-lego psql -U postgres -d lego < lego.sql
```

### 3. Connect with PGBabble

```bash
# Build pgbabble
go build -o pgbabble cmd/pgbabble/main.go

# Connect to the LEGO database
./pgbabble "postgresql://postgres:password@localhost/lego"
```

### 4. Ask LLM to help with queries

Once connected, try asking about the data or for specific queries:
```
pgbabble> Can you describe what tables are in this database?
pgbabble> How many themes have a parent theme?
pgbabble> Can you provide a summary of all colors and how many parts they are used in?
```

### 5. Use Interactive Commands
```
pgbabble> /help              # Show all available commands
pgbabble> /browse            # Browse last query results in full
pgbabble> /save [filename]   # Save last query results to CSV file
pgbabble> /schema            # Database overview
pgbabble> /tables            # List all tables
pgbabble> /describe <table>  # Detailed table structure
pgbabble> /mode              # Show privacy mode
```

### Example Workflow
1. Run a natural language query that returns many rows
2. View the first 25 rows with intelligent column formatting
3. Type `/browse` to explore all results in the `less` pager
4. Use standard `less` navigation (space, arrows, search with `/pattern`)
5. Press 'q' to return to the pgbabble prompt
6. Type `/save` to export results to CSV file with default filename
7. Or use `/save my_analysis.csv` to specify a custom filename

## Development

```bash
# Run tests
make test

# Build
make build
```
