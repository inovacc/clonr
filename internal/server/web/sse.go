package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type    string `json:"type"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// SSEHub manages SSE connections and broadcasts
type SSEHub struct {
	clients    map[chan SSEEvent]bool
	broadcast  chan SSEEvent
	register   chan chan SSEEvent
	unregister chan chan SSEEvent
	mu         sync.RWMutex
}

// NewSSEHub creates a new SSE hub
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:    make(map[chan SSEEvent]bool),
		broadcast:  make(chan SSEEvent, 100),
		register:   make(chan chan SSEEvent),
		unregister: make(chan chan SSEEvent),
	}
}

// Run starts the hub's event loop
func (h *SSEHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("SSE client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client)
			}
			h.mu.Unlock()
			log.Printf("SSE client disconnected (total: %d)", len(h.clients))

		case event := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client <- event:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends an event to all connected clients
func (h *SSEHub) Broadcast(event SSEEvent) {
	select {
	case h.broadcast <- event:
	default:
		log.Printf("SSE broadcast channel full, dropping event: %s", event.Type)
	}
}

// ClientCount returns the number of connected clients
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// handleSSE handles SSE connections
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	client := make(chan SSEEvent, 10)

	// Register client
	s.sseHub.register <- client

	// Ensure cleanup on disconnect
	defer func() {
		s.sseHub.unregister <- client
	}()

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	s.sendSSEEvent(w, flusher, SSEEvent{
		Type:    "connected",
		Message: "SSE connection established",
		Data: map[string]any{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})

	// Send heartbeat and events
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case event := <-client:
			s.sendSSEEvent(w, flusher, event)

		case <-heartbeat.C:
			s.sendSSEEvent(w, flusher, SSEEvent{
				Type:    "heartbeat",
				Message: "ping",
				Data: map[string]any{
					"timestamp": time.Now().Format(time.RFC3339),
					"clients":   s.sseHub.ClientCount(),
				},
			})
		}
	}
}

// sendSSEEvent writes an SSE event to the response
func (s *Server) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal SSE event: %v", err)
		return
	}

	_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// BroadcastEvent sends an event to all connected SSE clients
func (s *Server) BroadcastEvent(eventType string, message string, data any) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.Broadcast(SSEEvent{
		Type:    eventType,
		Message: message,
		Data:    data,
	})
}

// Event type constants
const (
	EventProfileCreated        = "profile:created"
	EventProfileUpdated        = "profile:updated"
	EventProfileDeleted        = "profile:deleted"
	EventProfileActivated      = "profile:activated"
	EventWorkspaceCreated      = "workspace:created"
	EventWorkspaceDeleted      = "workspace:deleted"
	EventSlackConnected        = "slack:connected"
	EventSlackDisconnected     = "slack:disconnected"
	EventSlackAccountCreated   = "slack:account:created"
	EventSlackAccountDeleted   = "slack:account:deleted"
	EventSlackAccountActivated = "slack:account:activated"
	EventServerStatus          = "server:status"
	EventNotification          = "notification"
)
