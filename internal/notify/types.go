// Package notify provides notification dispatching for clonr events.
package notify

import (
	"context"
	"time"
)

// Event represents a notification event with all context needed for formatting.
type Event struct {
	// Type is the event type (push, clone, pull, commit, etc.)
	Type string

	// Repository is the repository name (owner/repo)
	Repository string

	// Branch is the branch name (if applicable)
	Branch string

	// Commit is the commit SHA (if applicable)
	Commit string

	// CommitMessage is the commit message (if applicable)
	CommitMessage string

	// Author is the author of the action
	Author string

	// URL is a link to the relevant resource
	URL string

	// Profile is the profile name that triggered the event
	Profile string

	// Workspace is the workspace name
	Workspace string

	// Timestamp is when the event occurred
	Timestamp time.Time

	// Success indicates if the operation succeeded
	Success bool

	// Error contains error details if the operation failed
	Error string

	// Extra contains additional event-specific data
	Extra map[string]string
}

// Sender is the interface for notification senders.
type Sender interface {
	// Send sends a notification for the given event.
	// Returns an error if the notification could not be sent.
	Send(ctx context.Context, event *Event) error

	// Name returns the sender's name for logging purposes.
	Name() string

	// Test sends a test notification to verify configuration.
	Test(ctx context.Context) error
}

// Filter determines whether an event should be sent to a specific channel.
type Filter interface {
	// Match returns true if the event matches the filter criteria.
	Match(event *Event) bool
}

// Priority levels for notifications.
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

// Event types that can trigger notifications.
const (
	EventPush     = "push"
	EventClone    = "clone"
	EventPull     = "pull"
	EventCommit   = "commit"
	EventPRCreate = "pr-create"
	EventPRMerge  = "pr-merge"
	EventCIPass   = "ci-pass"
	EventCIFail   = "ci-fail"
	EventRelease  = "release"
	EventSync     = "sync"
	EventError    = "error"
)

// NewEvent creates a new event with the given type and sets the timestamp.
func NewEvent(eventType string) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Success:   true,
		Extra:     make(map[string]string),
	}
}

// WithRepository sets the repository on the event.
func (e *Event) WithRepository(repo string) *Event {
	e.Repository = repo
	return e
}

// WithBranch sets the branch on the event.
func (e *Event) WithBranch(branch string) *Event {
	e.Branch = branch
	return e
}

// WithCommit sets the commit SHA and message on the event.
func (e *Event) WithCommit(sha, message string) *Event {
	e.Commit = sha
	e.CommitMessage = message

	return e
}

// WithAuthor sets the author on the event.
func (e *Event) WithAuthor(author string) *Event {
	e.Author = author
	return e
}

// WithURL sets the URL on the event.
func (e *Event) WithURL(url string) *Event {
	e.URL = url
	return e
}

// WithProfile sets the profile on the event.
func (e *Event) WithProfile(profile string) *Event {
	e.Profile = profile
	return e
}

// WithWorkspace sets the workspace on the event.
func (e *Event) WithWorkspace(workspace string) *Event {
	e.Workspace = workspace
	return e
}

// WithError sets the error on the event and marks it as failed.
func (e *Event) WithError(err string) *Event {
	e.Error = err
	e.Success = false

	return e
}

// WithExtra adds extra data to the event.
func (e *Event) WithExtra(key, value string) *Event {
	if e.Extra == nil {
		e.Extra = make(map[string]string)
	}

	e.Extra[key] = value

	return e
}
