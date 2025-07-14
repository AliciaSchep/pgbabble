package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestContextCancellationBehavior(t *testing.T) {
	t.Run("cancelled context stops progress indicator", func(t *testing.T) {
		// Create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())

		// Track if progress goroutine terminates
		var progressStopped bool
		var wg sync.WaitGroup
		wg.Add(1)

		// Simulate the progress indicator goroutine from executeSelectQuery
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(100 * time.Millisecond) // faster for test
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					// Progress continues
				case <-ctx.Done():
					// Context cancelled - stop progress indicator
					progressStopped = true
					return
				}
			}
		}()

		// Let the goroutine start
		time.Sleep(50 * time.Millisecond)

		// Cancel the context (simulating Ctrl-C)
		cancel()

		// Wait for goroutine to terminate with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Goroutine terminated as expected
		case <-time.After(1 * time.Second):
			t.Error("Progress indicator goroutine did not terminate when context was cancelled")
		}

		if !progressStopped {
			t.Error("Progress indicator should have been marked as stopped")
		}
	})

	t.Run("cancelled context propagates to child operations", func(t *testing.T) {
		// Create cancellable parent context
		parentCtx, cancel := context.WithCancel(context.Background())

		// Create child context with timeout (like in executeSelectQuery)
		childCtx, childCancel := context.WithTimeout(parentCtx, 5*time.Second)
		defer childCancel()

		// Cancel parent context (simulating Ctrl-C)
		cancel()

		// Child context should be cancelled too
		select {
		case <-childCtx.Done():
			// Expected - child context was cancelled
			if childCtx.Err() != context.Canceled {
				t.Errorf("Expected context.Canceled, got %v", childCtx.Err())
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Child context should have been cancelled when parent was cancelled")
		}
	})

	t.Run("timeout vs cancellation can be distinguished", func(t *testing.T) {
		// Test timeout scenario
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		// Wait for timeout
		<-timeoutCtx.Done()

		if timeoutCtx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded for timeout, got %v", timeoutCtx.Err())
		}

		// Test cancellation scenario
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Immediately cancel

		<-cancelCtx.Done()

		if cancelCtx.Err() != context.Canceled {
			t.Errorf("Expected Canceled for cancellation, got %v", cancelCtx.Err())
		}
	})
}

func TestGracefulCleanup(t *testing.T) {
	t.Run("resources are cleaned up on cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Track resource cleanup
		var resourcesCleaned bool
		var goroutineFinished bool

		// Simulate a long-running operation that needs cleanup
		go func() {
			defer func() {
				resourcesCleaned = true
				goroutineFinished = true
			}()

			// Simulate work that can be interrupted
			select {
			case <-time.After(5 * time.Second):
				// This should not happen in our test
				t.Error("Operation should have been cancelled")
			case <-ctx.Done():
				// Context cancelled - clean up and exit
				return
			}
		}()

		// Let operation start
		time.Sleep(10 * time.Millisecond)

		// Cancel context (simulate Ctrl-C)
		cancel()

		// Wait for cleanup to complete
		timeout := time.After(1 * time.Second)
		for !goroutineFinished {
			select {
			case <-timeout:
				t.Error("Goroutine did not finish within timeout")
				return
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}

		if !resourcesCleaned {
			t.Error("Resources should have been cleaned up")
		}
	})

	t.Run("multiple goroutines terminate on cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		const numGoroutines = 3
		var wg sync.WaitGroup
		var terminatedCount int32
		var mu sync.Mutex

		// Start multiple goroutines that listen for cancellation
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				select {
				case <-time.After(5 * time.Second):
					t.Errorf("Goroutine %d should have been cancelled", id)
				case <-ctx.Done():
					mu.Lock()
					terminatedCount++
					mu.Unlock()
					return
				}
			}(i)
		}

		// Let goroutines start
		time.Sleep(10 * time.Millisecond)

		// Cancel context
		cancel()

		// Wait for all goroutines to finish
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All goroutines finished
		case <-time.After(1 * time.Second):
			t.Error("Not all goroutines terminated within timeout")
		}

		mu.Lock()
		finalCount := terminatedCount
		mu.Unlock()

		if finalCount != numGoroutines {
			t.Errorf("Expected %d goroutines to terminate, got %d", numGoroutines, finalCount)
		}
	})
}

func TestContextPropagationChain(t *testing.T) {
	t.Run("cancellation propagates through function chain", func(t *testing.T) {
		// Simulate the chain: main -> session -> agent -> tools -> db
		rootCtx, cancel := context.WithCancel(context.Background())

		var chainCancelled []string
		var mu sync.Mutex

		// Simulate main level
		mainCtx := rootCtx

		// Simulate session level
		sessionFunc := func(ctx context.Context) {
			defer func() {
				mu.Lock()
				chainCancelled = append(chainCancelled, "session")
				mu.Unlock()
			}()

			// Simulate agent level
			agentFunc := func(ctx context.Context) {
				defer func() {
					mu.Lock()
					chainCancelled = append(chainCancelled, "agent")
					mu.Unlock()
				}()

				// Simulate tool level
				toolFunc := func(ctx context.Context) {
					defer func() {
						mu.Lock()
						chainCancelled = append(chainCancelled, "tool")
						mu.Unlock()
					}()

					// Wait for cancellation
					<-ctx.Done()
				}

				toolFunc(ctx)
			}

			agentFunc(ctx)
		}

		// Start the chain
		go sessionFunc(mainCtx)

		// Let chain start
		time.Sleep(10 * time.Millisecond)

		// Cancel root context
		cancel()

		// Wait for cancellation to propagate
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		cancelled := chainCancelled
		mu.Unlock()

		expectedChain := []string{"tool", "agent", "session"}
		if len(cancelled) != len(expectedChain) {
			t.Errorf("Expected %d levels to be cancelled, got %d: %v",
				len(expectedChain), len(cancelled), cancelled)
		}

		// Verify cancellation propagated through all levels
		for _, expected := range expectedChain {
			found := false
			for _, actual := range cancelled {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %s to be cancelled, but it wasn't in %v", expected, cancelled)
			}
		}
	})
}
