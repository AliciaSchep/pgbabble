# Testing Patterns for pgbabble

## Database Interface Mocking Pattern

This guide demonstrates the proper way to test database-dependent methods using interface-based dependency injection.

### The Problem

Original code had database methods that were hard to test:
```go
func (s *Session) describeTable(ctx context.Context, tableName string) error {
    table, err := s.conn.DescribeTable(ctx, schema, tableName)  // Hard to mock
    // ... business logic
}
```

Tests were either:
- ❌ Testing helper functions that reimplemented logic
- ❌ Using nil connections that caused panics
- ❌ Not testing the actual methods at all

### The Solution: Interface-Based Dependency Injection

#### Step 1: Define the Interface

```go
// DatabaseConnection interface defines the database operations needed
type DatabaseConnection interface {
    ListTables(ctx context.Context) ([]db.TableInfo, error)
    DescribeTable(ctx context.Context, schema, tableName string) (*db.TableInfo, error)
    GetForeignKeys(ctx context.Context, schema, tableName string) ([]db.ForeignKeyInfo, error)
}
```

#### Step 2: Update the Struct

```go
type Session struct {
    conn DatabaseConnection  // Use interface, not concrete type
    // ... other fields
}

func NewSession(conn DatabaseConnection, mode string) *Session {
    return &Session{conn: conn, mode: mode}
}
```

#### Step 3: Create Mock Implementation

```go
type MockDBConnection struct {
    tables       []db.TableInfo
    tableDetails map[string]*db.TableInfo
    foreignKeys  map[string][]db.ForeignKeyInfo
    shouldFail   string // Which method should fail for error testing
}

func (m *MockDBConnection) DescribeTable(ctx context.Context, schema, tableName string) (*db.TableInfo, error) {
    if m.shouldFail == "DescribeTable" {
        return nil, fmt.Errorf("mock database error: DescribeTable failed")
    }
    if table, exists := m.tableDetails[tableName]; exists {
        return table, nil
    }
    return nil, fmt.Errorf("table %s.%s not found", schema, tableName)
}
// ... implement other interface methods
```

#### Step 4: Write Proper Tests

```go
func TestSession_DescribeTable_WithMockDB(t *testing.T) {
    // Arrange: Create mock and session
    mockDB := NewMockDBConnection()
    session := NewSession(mockDB, "default")
    ctx := context.Background()

    t.Run("describe_existing_table", func(t *testing.T) {
        // Act: Call the ACTUAL session method
        err := session.describeTable(ctx, "users")
        
        // Assert: Verify behavior
        if err != nil {
            t.Fatalf("describeTable failed: %v", err)
        }
    })

    t.Run("describe_nonexistent_table", func(t *testing.T) {
        // Test error cases
        err := session.describeTable(ctx, "nonexistent")
        if err == nil {
            t.Error("Expected error for nonexistent table")
        }
    })
}
```

### Key Benefits

1. **Tests actual methods** - Not helper functions that reimplement logic
2. **Full error path coverage** - Can simulate database failures
3. **Fast execution** - No real database needed
4. **Deterministic** - Predictable mock data
5. **High coverage** - Can test edge cases easily

### Results

- ✅ `describeTable` coverage: **10.0% → 95.0%**
- ✅ Overall chat package coverage: **32.8% → 49.3%**
- ✅ Tests actual business logic and error handling
- ✅ Fast, reliable, deterministic tests

### Anti-Patterns to Avoid

❌ **Don't create helper functions that reimplement logic:**
```go
// BAD: This doesn't test the actual session method
func testDescribeTableHelper(mockDB DatabaseConnection, tableName string) {
    // Reimplementing session.describeTable logic here
}
```

❌ **Don't test with nil connections:**
```go
// BAD: This will panic
session := NewSession(nil, "default")
session.describeTable(ctx, "users")  // panic!
```

❌ **Don't test only the mock:**
```go
// BAD: This only tests the mock, not the session
mockDB.DescribeTable(ctx, "public", "users")
```

### Next Steps

Apply this pattern to other database-dependent methods:
- `showSchema()`
- `listTables()`
- Any other methods that use `s.conn`

The key is: **Mock the dependency, inject the mock, test the actual method.**