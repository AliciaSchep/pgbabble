package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AliciaSchep/pgbabble/pkg/agent"
)

// Test that context cancellation returns a proper tool result instead of an error
func main() {
	// Create a cancellable context that simulates Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel the context after a short delay to simulate Ctrl+C
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	
	// Call executeSelectQuery with a query that would take time
	// but will be cancelled by the context
	result, err := agent.TestExecuteSelectQuery(ctx, nil, "SELECT pg_sleep(10)", "default")
	
	if err != nil {
		fmt.Printf("ERROR: executeSelectQuery returned an error instead of handling cancellation: %v\n", err)
		return
	}
	
	if strings.Contains(result, "cancelled") {
		fmt.Println("SUCCESS: Query cancellation returned proper tool result")
		fmt.Printf("Result: %s\n", result)
	} else {
		fmt.Printf("ERROR: Expected cancellation result, got: %s\n", result)
	}
}