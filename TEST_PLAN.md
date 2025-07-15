# Test Improvement Plan for PGBabble

## Overview

This document outlines the comprehensive plan to improve test coverage and quality for the PGBabble project, with a focus on avoiding excessive mocking and implementing real database integration tests.

## Current State Analysis

### Coverage by Package
- **pkg/config**: 98.3% ✅ (excellent)
- **pkg/display**: 65.4% ✅ (good)
- **pkg/agent**: 30.5% ⚠️ (needs improvement)
- **pkg/db**: 14.8% ❌ (critical - core database functionality)
- **pkg/chat**: 0.5% ❌ (critical - session management)
- **cmd/pgbabble**: 15.4% ❌ (main CLI entry point)

### Current Testing Approach
- Mostly unit tests with minimal mocking
- Some integration tests exist but are limited
- Database tests currently skip if no local PostgreSQL available
- No standardized test database infrastructure

## Implementation Phases

### Phase 1: Test Database Infrastructure

#### Goals
- Create portable, consistent test database setup
- Enable integration tests without requiring manual PostgreSQL setup
- Provide simple but realistic test schema

#### Implementation
1. **Dependencies**
   - Add `testcontainers-go` for portable PostgreSQL containers
   - Add fallback support for environment-based testing

2. **Test Database Schema**
   ```sql
   -- Simple but realistic schema for testing
   CREATE TABLE users (
       id SERIAL PRIMARY KEY,
       username VARCHAR(50) UNIQUE NOT NULL,
       email VARCHAR(100) UNIQUE NOT NULL,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );

   CREATE TABLE products (
       id SERIAL PRIMARY KEY,
       name VARCHAR(100) NOT NULL,
       price DECIMAL(10,2) NOT NULL,
       category VARCHAR(50),
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );

   CREATE TABLE orders (
       id SERIAL PRIMARY KEY,
       user_id INTEGER REFERENCES users(id),
       total_amount DECIMAL(10,2) NOT NULL,
       status VARCHAR(20) DEFAULT 'pending',
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );

   CREATE TABLE order_items (
       id SERIAL PRIMARY KEY,
       order_id INTEGER REFERENCES orders(id),
       product_id INTEGER REFERENCES products(id),
       quantity INTEGER NOT NULL,
       price DECIMAL(10,2) NOT NULL
   );
   ```

3. **Test Utilities**
   - `internal/testutil/database.go`: Database setup/teardown
   - Transaction-based test isolation
   - Seed data fixtures
   - Helper functions for common test scenarios

### Phase 2: Database Package Testing (pkg/db)

#### Current Issues
- Only 14.8% coverage despite being core functionality
- Connection logic undertested
- Schema introspection not tested with real data
- Reconnection logic hard to test without real failures

#### Improvements
1. **Connection Testing**
   - Test real connection establishment
   - Test reconnection logic with container restarts
   - Test timeout and cancellation behavior
   - Test connection string generation and parsing

2. **Query Interface Testing**
   - Test Query/QueryRow/Exec methods with real data
   - Test transaction handling
   - Test error scenarios with real PostgreSQL errors

3. **Schema Introspection Testing**
   - Test table listing with known test schema
   - Test column information extraction
   - Test foreign key relationship discovery
   - Test index and constraint information

#### Target Coverage: 80%+

### Phase 3: Agent Package Testing (pkg/agent)

#### Current Issues
- 30.5% coverage, but critical SQL safety and tool execution undertested
- SQL validation tested but not with real database responses
- Result formatting tested with mock data only

#### Improvements
1. **SQL Tool Integration Tests**
   - Test `execute_sql` tool with real database and known data
   - Test schema inspection tools against test database
   - Test query result formatting with actual PostgreSQL data types
   - Test privacy mode filtering with real query results

2. **Safety and Validation**
   - Test SQL safety validation with real PostgreSQL parsing
   - Test dangerous query detection in real execution context
   - Test timeout and cancellation with real long-running queries

3. **Error Handling**
   - Test database error formatting with real PostgreSQL errors
   - Test connection failure handling during tool execution
   - Test malformed SQL handling

#### Target Coverage: 70%+

### Phase 4: Chat Session Testing (pkg/chat)

#### Current Issues
- Only 0.5% coverage - critical session management logic untested
- User approval workflow not tested
- Integration with agent and database not tested

#### Improvements
1. **Session Management**
   - Test session creation and lifecycle
   - Test conversation state management
   - Test mode switching and validation

2. **User Approval Workflow**
   - Test approval prompt generation
   - Test response parsing and validation
   - Test SQL execution flow with approvals

3. **Integration Testing**
   - Test full chat session with test database
   - Test error recovery in conversation flow
   - Test memory management for long conversations

#### Target Coverage: 60%+

### Phase 5: CLI Integration Testing (cmd/pgbabble)

#### Current Issues
- 15.4% coverage of main entry point
- Command-line parsing and database connection not well tested
- End-to-end workflows not covered

#### Improvements
1. **Command-Line Interface**
   - Test argument parsing and validation
   - Test configuration loading from environment
   - Test database connection establishment

2. **End-to-End Workflows**
   - Test basic query execution flow
   - Test interactive session management
   - Test error handling and user feedback

#### Target Coverage: 50%+

## Testing Strategy

### Principles
1. **Avoid Over-Mocking**: Use real database for meaningful integration tests
2. **Fast Unit Tests**: Keep existing unit tests, add integration layer
3. **Realistic Scenarios**: Test with actual PostgreSQL behavior and data types
4. **CI-Friendly**: Testcontainers with environment variable fallbacks
5. **Incremental**: Each phase builds on previous infrastructure

### Test Categories
1. **Unit Tests**: Fast, isolated, no external dependencies
2. **Integration Tests**: Real database, realistic scenarios
3. **End-to-End Tests**: Full workflow testing with CLI

### Infrastructure Requirements
- Go 1.24.3+
- Docker for testcontainers (optional with env fallback)
- PostgreSQL test container image
- Test data fixtures and utilities

## Implementation Timeline

1. **Phase 1** (Foundation): 2-3 days
   - Test database infrastructure
   - Basic utilities and helpers

2. **Phase 2** (Database Core): 2-3 days
   - Connection and query testing
   - Schema introspection testing

3. **Phase 3** (Agent Logic): 3-4 days
   - SQL tool integration tests
   - Safety and validation improvements

4. **Phase 4** (Session Management): 2-3 days
   - Chat session testing
   - User approval workflow testing

5. **Phase 5** (CLI Integration): 1-2 days
   - End-to-end CLI testing
   - Workflow validation

## Success Metrics

- **Overall coverage**: 30.9% → 60%+
- **Critical packages**: pkg/db, pkg/chat, pkg/agent all above 60%
- **Integration test coverage**: Comprehensive testing of core workflows
- **CI reliability**: Tests pass consistently across environments
- **Test maintainability**: Clear, readable tests that are easy to update

## Maintenance

- Regular review of test coverage as features are added
- Update test data and schema as application evolves
- Monitor test execution time and optimize as needed
- Keep test utilities and helpers up to date with best practices