package core

import (
	"context"
	"time"
)

// Common timeout durations used throughout the codebase
const (
	TimeoutShort  = 30 * time.Second // For quick API calls
	TimeoutMedium = 2 * time.Minute  // For standard operations
	TimeoutLong   = 5 * time.Minute  // For longer operations like uploads
	TimeoutXLong  = 10 * time.Minute // For very long operations like downloads
)

// WithShortTimeout creates a context with a 30-second timeout.
// Use for quick API calls like fetching single resources.
func WithShortTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TimeoutShort)
}

// WithMediumTimeout creates a context with a 2-minute timeout.
// Use for standard operations like listing resources.
func WithMediumTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TimeoutMedium)
}

// WithLongTimeout creates a context with a 5-minute timeout.
// Use for longer operations like file uploads.
func WithLongTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TimeoutLong)
}

// WithXLongTimeout creates a context with a 10-minute timeout.
// Use for very long operations like large file downloads.
func WithXLongTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TimeoutXLong)
}

// WithTimeout creates a context with a custom timeout duration.
// Prefer the predefined timeout functions when applicable.
func WithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// WithTimeoutFrom creates a context with timeout derived from parent context.
// Use when you need to inherit cancellation from a parent context.
func WithTimeoutFrom(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
