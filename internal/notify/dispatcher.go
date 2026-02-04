package notify

import (
	"context"
	"log"
	"sync"
	"time"
)

// Dispatcher routes events to registered senders.
type Dispatcher struct {
	senders []Sender
	mu      sync.RWMutex
	async   bool
}

// NewDispatcher creates a new notification dispatcher.
// If async is true, notifications are sent in goroutines.
func NewDispatcher(async bool) *Dispatcher {
	return &Dispatcher{
		senders: make([]Sender, 0),
		async:   async,
	}
}

// Register adds a sender to the dispatcher.
func (d *Dispatcher) Register(sender Sender) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.senders = append(d.senders, sender)
}

// Unregister removes a sender from the dispatcher by name.
func (d *Dispatcher) Unregister(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	filtered := make([]Sender, 0, len(d.senders))
	for _, s := range d.senders {
		if s.Name() != name {
			filtered = append(filtered, s)
		}
	}
	d.senders = filtered
}

// Dispatch sends an event to all registered senders.
func (d *Dispatcher) Dispatch(ctx context.Context, event *Event) {
	d.mu.RLock()
	senders := make([]Sender, len(d.senders))
	copy(senders, d.senders)
	d.mu.RUnlock()

	if len(senders) == 0 {
		return
	}

	if d.async {
		for _, sender := range senders {
			go d.sendWithRecover(ctx, sender, event)
		}
	} else {
		for _, sender := range senders {
			d.sendWithRecover(ctx, sender, event)
		}
	}
}

// sendWithRecover sends an event and recovers from panics.
func (d *Dispatcher) sendWithRecover(ctx context.Context, sender Sender, event *Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("notify: panic in sender %s: %v", sender.Name(), r)
		}
	}()

	// Create a timeout context for the send operation
	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := sender.Send(sendCtx, event); err != nil {
		log.Printf("notify: error sending to %s: %v", sender.Name(), err)
	}
}

// HasSenders returns true if any senders are registered.
func (d *Dispatcher) HasSenders() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.senders) > 0
}

// Senders returns a copy of the registered senders.
func (d *Dispatcher) Senders() []Sender {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]Sender, len(d.senders))
	copy(result, d.senders)
	return result
}

// defaultDispatcher is the global dispatcher instance.
var (
	defaultDispatcher     *Dispatcher
	defaultDispatcherOnce sync.Once
)

// GetDispatcher returns the global dispatcher instance.
func GetDispatcher() *Dispatcher {
	defaultDispatcherOnce.Do(func() {
		defaultDispatcher = NewDispatcher(true) // async by default
	})
	return defaultDispatcher
}

// Send is a convenience function to dispatch an event using the global dispatcher.
func Send(ctx context.Context, event *Event) {
	GetDispatcher().Dispatch(ctx, event)
}
