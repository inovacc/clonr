package grpcserver

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// IdleTracker tracks server activity and determines when the server is idle.
type IdleTracker struct {
	mu           sync.RWMutex
	lastActivity time.Time
	idleTimeout  time.Duration
	shutdownChan chan struct{}
	done         bool
}

// NewIdleTracker creates a new idle tracker with the specified timeout.
// If timeout is 0, the tracker is disabled (never triggers shutdown).
func NewIdleTracker(timeout time.Duration) *IdleTracker {
	return &IdleTracker{
		lastActivity: time.Now(),
		idleTimeout:  timeout,
		shutdownChan: make(chan struct{}),
	}
}

// Touch updates the last activity time.
func (t *IdleTracker) Touch() {
	t.mu.Lock()
	t.lastActivity = time.Now()
	t.mu.Unlock()
}

// IsEnabled returns true if idle timeout is enabled.
func (t *IdleTracker) IsEnabled() bool {
	return t.idleTimeout > 0
}

// ShutdownChan returns a channel that will be closed when idle timeout is reached.
func (t *IdleTracker) ShutdownChan() <-chan struct{} {
	return t.shutdownChan
}

// Start begins monitoring for idle timeout.
// This should be called in a goroutine.
func (t *IdleTracker) Start() {
	if !t.IsEnabled() {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		t.mu.RLock()
		idle := time.Since(t.lastActivity)
		done := t.done
		t.mu.RUnlock()

		if done {
			return
		}

		if idle >= t.idleTimeout {
			close(t.shutdownChan)
			return
		}
	}
}

// Stop stops the idle tracker.
func (t *IdleTracker) Stop() {
	t.mu.Lock()
	t.done = true
	t.mu.Unlock()
}

// IdleTimeout returns the configured idle timeout.
func (t *IdleTracker) IdleTimeout() time.Duration {
	return t.idleTimeout
}

// activityInterceptor creates an interceptor that updates the idle tracker on each request.
func activityInterceptor(tracker *IdleTracker) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		tracker.Touch()
		return handler(ctx, req)
	}
}
