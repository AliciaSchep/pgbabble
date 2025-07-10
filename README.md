# PGBabble

A CLI tool for interacting with PostgreSQL databases using natural language queries powered by LLMs.

## Features

- Natural language to SQL conversion
- Privacy-first design (no data sent to LLM by default)
- Interactive chat interface
- psql-compatible connection handling
- Schema inspection and exploration

## Usage

```bash
# Connect with URI
pgbabble "postgresql://user:pass@localhost/mydb"

# Connect with individual parameters
pgbabble --host localhost --port 5432 --user myuser --dbname mydb

# Use environment variables (like psql)
export PGHOST=localhost PGUSER=myuser PGDATABASE=mydb
pgbabble
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

## Development

See [DEVELOPMENT_PLAN.md](DEVELOPMENT_PLAN.md) for detailed development roadmap.

```bash
# Run tests
go test ./...

# Build
go build -o pgbabble cmd/pgbabble/main.go
```