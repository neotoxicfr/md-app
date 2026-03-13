package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ---- SSE-based collaborative editing hub ----

const maxClientsPerRoom = 50

// CollabHub manages per-file collaboration rooms.
type CollabHub struct {
	mu    sync.RWMutex
	rooms map[string]*collabRoom // fileID -> room
}

// collabRoom groups SSE clients editing the same file.
type collabRoom struct {
	mu      sync.RWMutex
	fileID  string
	clients map[string]*sseClient
}

// sseClient represents a single connected SSE consumer.
type sseClient struct {
	id     string
	events chan []byte
	done   chan struct{}
}

// NewCollabHub creates a new collaboration hub.
func NewCollabHub() *CollabHub {
	return &CollabHub{rooms: make(map[string]*collabRoom)}
}

// getOrCreateRoom returns (or creates) the room for fileID.
func (h *CollabHub) getOrCreateRoom(fileID string) *collabRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[fileID]
	if !ok {
		room = &collabRoom{
			fileID:  fileID,
			clients: make(map[string]*sseClient),
		}
		h.rooms[fileID] = room
	}
	return room
}

// removeClient removes a client from its room and cleans up empty rooms.
func (h *CollabHub) removeClient(fileID, clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[fileID]
	if !ok {
		return
	}

	room.mu.Lock()
	delete(room.clients, clientID)
	remaining := len(room.clients)
	room.mu.Unlock()

	if remaining == 0 {
		delete(h.rooms, fileID)
	}
}

// ---- handler ----

// collabHandler exposes SSE endpoints for live collaboration.
type collabHandler struct {
	hub *CollabHub
}

func newCollabHandler(hub *CollabHub) *collabHandler {
	return &collabHandler{hub: hub}
}

// GET /api/files/{id}/events — SSE stream for live updates.
//
// Protocol:
//
//	event: connected         — sent once on connect with the client's anonymous ID
//	event: (default/data)    — JSON messages: edit, cursor, presence
//	: keepalive              — sent every 30 s to keep the connection alive
func (ch *collabHandler) events(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	clientID := "anon-" + uuid.New().String()[:8]

	client := &sseClient{
		id:     clientID,
		events: make(chan []byte, 64),
		done:   make(chan struct{}),
	}

	room := ch.hub.getOrCreateRoom(fileID)
	room.mu.Lock()
	if len(room.clients) >= maxClientsPerRoom {
		room.mu.Unlock()
		writeError(w, http.StatusServiceUnavailable, "too many editors in this room")
		return
	}
	room.clients[clientID] = client
	room.mu.Unlock()

	defer func() {
		close(client.done)
		ch.broadcastPresence(room)
		ch.hub.removeClient(fileID, clientID)
		slog.Info("collab: client disconnected", "file_id", fileID, "client", clientID)
	}()

	// SSE response headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // prevent nginx buffering

	// Initial connection event.
	fmt.Fprintf(w, "event: connected\ndata: {\"user\":%q}\n\n", clientID)
	flusher.Flush()

	// Notify all clients of the updated presence list.
	ch.broadcastPresence(room)

	slog.Info("collab: client connected", "file_id", fileID, "client", clientID)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-client.events:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// POST /api/files/{id}/broadcast — push a change event to all listeners.
//
// Request body:
//
//	{
//	  "type":    "edit",          // edit | cursor | selection
//	  "content": "# Hello",      // for edits
//	  "user":    "anon-abc123",   // sender (excluded from broadcast)
//	  "cursor":  {"line":1,"ch":5}
//	}
func (ch *collabHandler) broadcast(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")

	var body struct {
		Type    string `json:"type"`
		Content string `json:"content"`
		User    string `json:"user"`
		Cursor  *struct {
			Line int `json:"line"`
			Ch   int `json:"ch"`
		} `json:"cursor,omitempty"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Type == "" {
		body.Type = "edit"
	}

	msg, err := json.Marshal(map[string]any{
		"type":    body.Type,
		"content": body.Content,
		"user":    body.User,
		"cursor":  body.Cursor,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshal error")
		return
	}

	ch.hub.mu.RLock()
	room, ok := ch.hub.rooms[fileID]
	ch.hub.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"delivered": 0})
		return
	}

	delivered := 0
	room.mu.RLock()
	for _, client := range room.clients {
		if client.id == body.User {
			continue // don't echo to sender
		}
		select {
		case client.events <- msg:
			delivered++
		default:
			slog.Warn("collab: client buffer full, dropping",
				"client", client.id,
				"file_id", fileID,
			)
		}
	}
	room.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{"delivered": delivered})
}

// broadcastPresence sends the current user list to every client in a room.
func (ch *collabHandler) broadcastPresence(room *collabRoom) {
	room.mu.RLock()
	users := make([]string, 0, len(room.clients))
	targets := make([]*sseClient, 0, len(room.clients))
	for _, c := range room.clients {
		users = append(users, c.id)
		targets = append(targets, c)
	}
	room.mu.RUnlock()

	msg, err := json.Marshal(map[string]any{
		"type":  "presence",
		"users": users,
	})
	if err != nil {
		slog.Error("collab: marshal presence failed", "error", err)
		return
	}

	for _, c := range targets {
		select {
		case c.events <- msg:
		default:
			// drop if buffer full
		}
	}
}
