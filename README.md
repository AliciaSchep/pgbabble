# PGBabble

A CLI tool for interacting with PostgreSQL databases using natural language queries powered by LLMs.

## Features

- Natural language to SQL conversion
- Privacy-first design (no data sent to LLM by default)
- Interactive chat interface
- psql-compatible connection handling
- Schema inspection and exploration
- **Enhanced visual output** with intelligent column sizing and pagination

## Usage

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

*Best for: Highly sensitive databases where only structural information should be shared*

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

To test PGBabble with sample data, you can set up a PostgreSQL database with the LEGO dataset:

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

### 4. Explore the Database

Once connected, try these commands:
```
pgbabble> /help              # Show all available commands
pgbabble> /schema            # Database overview
pgbabble> /tables            # List all tables
pgbabble> /describe lego_sets  # Detailed table structure
pgbabble> /describe lego_themes
```

The LEGO database includes tables for sets, themes, parts, colors, and more - perfect for testing schema exploration features!

## Enhanced Visual Output

PGBabble includes intelligent table formatting and pagination features to improve readability of query results:

### Smart Column Sizing
- **Automatic width detection**: Detects your terminal width and distributes column space intelligently
- **Content-aware sizing**: Analyzes data to determine optimal column widths
- **Readable formatting**: Ensures minimum column widths while maximizing use of available space

### Pagination with Less
- **Initial display**: Shows first 25 rows by default for quick viewing
- **Browse all results**: Use `/browse` command to view complete results in the `less` pager
- **Familiar navigation**: Standard `less` controls (space/enter for next page, 'q' to quit, '/' to search)
- **Large result support**: Handles queries with thousands of rows efficiently

### Interactive Commands
```
pgbabble> /help              # Show all available commands
pgbabble> /browse            # Browse last query results in full
pgbabble> /schema            # Database overview  
pgbabble> /tables            # List all tables
pgbabble> /describe <table>  # Detailed table structure
pgbabble> /mode [mode]       # Show or set privacy mode
```

### Example Workflow
1. Run a natural language query that returns many rows
2. View the first 25 rows with intelligent column formatting
3. Type `/browse` to explore all results in the `less` pager
4. Use standard `less` navigation (space, arrows, search with `/pattern`)
5. Press 'q' to return to the pgbabble prompt

The enhanced display works seamlessly with all privacy modes and maintains backward compatibility.

## Development

See [DEVELOPMENT_PLAN.md](DEVELOPMENT_PLAN.md) for detailed development roadmap.

```bash
# Run tests
go test ./...

# Build
go build -o pgbabble cmd/pgbabble/main.go
```