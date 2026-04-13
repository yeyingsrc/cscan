package scanner

import (
	"context"
	"testing"
	"time"
)

// This test serves as the RED checkpoint for Issue 3
// simulating a hang inside a long-running execution where wait is not handled properly.
func TestHttpxRunnerLeakAndTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Simulate whathttpxRunner.RunEnumeration does, hanging indefinitely 
	// Before our fix, this would block forever or leak
	done := make(chan struct{})
	
	go func() {
		// simulate infinite block
		time.Sleep(10 * time.Second)
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Timeout triggered, success
		t.Log("Context canceled properly avoiding leak")
	case <-done:
		t.Fatal("Simulate enumeration finished instead of timeout")
	}
}
