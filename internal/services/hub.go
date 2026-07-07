package services

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/damione1/planning-poker/internal/config"
	"github.com/damione1/planning-poker/internal/models"
)

var (
	ErrServerAtCapacity = errors.New("server at maximum capacity")
	ErrRoomFull         = errors.New("room has reached maximum participants")
	ErrRoomNotFound     = errors.New("room not found")
)

// MessageHandler processes incoming WebSocket messages
type MessageHandler func(roomID string, participantID string, message []byte)

// clientEvent represents a register/unregister request for a client. Both
// kinds flow through a single channel so that FIFO ordering is preserved
// per client (e.g. a register can never be processed after a later
// unregister for the same client, and vice versa).
type clientEvent struct {
	client   *Client
	register bool // true = register, false = unregister
}

// Hub manages WebSocket connections and message routing.
//
// All room membership state (rooms, per-room client sets, room count, and
// the total connection counter) is guarded by a single mutex (mu). This
// replaces the previous design where a sync.Map guarded only the outer
// room lookup while the inner map[*Client]bool values were mutated by the
// Run() goroutine and read/iterated concurrently by HTTP-request
// goroutines (BroadcastToRoom, CanRegister, GetClient, GetRoomSize) with no
// synchronization on those inner maps -- a data race that could trigger
// Go's "concurrent map iteration and map write" fatal error.
type Hub struct {
	mu        sync.RWMutex
	rooms     map[string]map[*Client]bool
	roomCount int
	// totalConnections is guarded by mu (folded in with the room state so
	// there is a single lock to reason about).
	totalConnections int64

	// Channels
	events        chan clientEvent
	handleMessage chan *ClientMessage

	// Message handler
	messageHandler MessageHandler

	// Metrics
	metrics *Metrics
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		rooms:         make(map[string]map[*Client]bool),
		events:        make(chan clientEvent, config.HubRegisterBufferSize+config.HubUnregisterBufferSize),
		handleMessage: make(chan *ClientMessage, config.HubBroadcastBufferSize),
		metrics:       NewMetrics(),
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	for {
		select {
		case ev := <-h.events:
			if ev.register {
				h.registerClient(ev.client)
			} else {
				h.unregisterClient(ev.client)
			}

		case msg := <-h.handleMessage:
			// Process message through registered handler
			if h.messageHandler != nil {
				h.messageHandler(msg.Client.roomID, msg.Client.participantID, msg.Message)
			}
		}
	}
}

// CanRegister checks if a new connection can be registered. This is a
// best-effort, cheap pre-upgrade check used by callers before accepting a
// websocket connection. It is inherently check-then-act (TOCTOU) under
// concurrency; the authoritative enforcement happens in registerClient
// under the hub's lock, which will refuse and close a client that would
// exceed limits even if CanRegister passed.
func (h *Hub) CanRegister(roomID string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check global connection limit
	if h.totalConnections >= config.MaxTotalConnections {
		return ErrServerAtCapacity
	}

	// Check room-specific limit
	if clients, ok := h.rooms[roomID]; ok {
		if len(clients) >= config.MaxConnectionsPerRoom {
			return ErrRoomFull
		}
	} else if h.roomCount >= config.MaxRoomsPerInstance {
		// Only a problem if this would be a new room.
		return ErrServerAtCapacity
	}

	return nil
}

// Register queues a client for registration
func (h *Hub) Register(roomID string, client *Client) {
	h.events <- clientEvent{client: client, register: true}
}

// Unregister queues a client for unregistration
func (h *Hub) Unregister(roomID string, client *Client) {
	h.events <- clientEvent{client: client, register: false}
}

// registerClient adds a client to a room. This is the authoritative
// capacity check: it re-validates limits under the lock so that a burst of
// concurrent connections that all passed the best-effort CanRegister check
// cannot collectively exceed the configured limits.
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()

	clients, roomExists := h.rooms[client.roomID]
	isNewRoom := !roomExists

	if h.totalConnections >= config.MaxTotalConnections ||
		(roomExists && len(clients) >= config.MaxConnectionsPerRoom) ||
		(isNewRoom && h.roomCount >= config.MaxRoomsPerInstance) {
		h.mu.Unlock()
		log.Printf("⛔ Rejecting client registration at capacity: room=%s participant=%s", client.roomID, client.participantID)
		client.Close()
		return
	}

	if isNewRoom {
		clients = make(map[*Client]bool)
		h.rooms[client.roomID] = clients
		h.roomCount++
	}

	clients[client] = true
	h.totalConnections++
	roomSize := len(clients)
	total := h.totalConnections

	h.mu.Unlock()

	h.metrics.IncrementConnections()
	if isNewRoom {
		h.metrics.IncrementRooms()
	}

	log.Printf("✓ Client registered: room=%s participant=%s (room size: %d, total connections: %d)",
		client.roomID, client.participantID, roomSize, total)
}

// unregisterClient removes a client from a room
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()

	clients, ok := h.rooms[client.roomID]
	if !ok {
		h.mu.Unlock()
		return
	}

	if _, exists := clients[client]; !exists {
		h.mu.Unlock()
		return
	}

	delete(clients, client)
	h.totalConnections--
	roomSize := len(clients)
	total := h.totalConnections

	roomDeleted := false
	if roomSize == 0 {
		delete(h.rooms, client.roomID)
		h.roomCount--
		roomDeleted = true
	}

	h.mu.Unlock()

	client.Close()
	h.metrics.DecrementConnections()

	if roomDeleted {
		h.metrics.DecrementRooms()
		log.Printf("🧹 Room cleaned up: %s", client.roomID)
	}

	log.Printf("✓ Client unregistered: room=%s participant=%s (room size: %d, total connections: %d)",
		client.roomID, client.participantID, roomSize, total)
}

// BroadcastToRoom sends a message to all clients in a room (non-blocking)
func (h *Hub) BroadcastToRoom(roomID string, message *models.WSMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.rooms[roomID]
	var snapshot []*Client
	if ok {
		snapshot = make([]*Client, 0, len(clients))
		for client := range clients {
			snapshot = append(snapshot, client)
		}
	}
	h.mu.RUnlock()

	if !ok {
		log.Printf("⚠️  Room not found: %s", roomID)
		return
	}

	log.Printf("📤 Broadcasting to room %s (%d clients): type=%s", roomID, len(snapshot), message.Type)

	// Send to all clients outside the lock (client.Send can briefly block
	// on its own internal mutex / channel send).
	successCount := 0
	for _, client := range snapshot {
		if client.Send(data) {
			successCount++
		}
	}

	log.Printf("✓ Broadcast complete: %d/%d clients received message", successCount, len(snapshot))
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(client *Client, message *models.WSMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	client.Send(data)
}

// GetRoomSize returns the number of clients in a room
func (h *Hub) GetRoomSize(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.rooms[roomID])
}

// GetMetrics returns the current metrics snapshot
func (h *Hub) GetMetrics() MetricsSnapshot {
	return h.metrics.Snapshot()
}

// GetTotalConnections returns the current number of active connections
func (h *Hub) GetTotalConnections() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.totalConnections
}

// GetRoomCount returns the current number of active rooms
func (h *Hub) GetRoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.roomCount
}

// SetMessageHandler sets the callback function for processing incoming messages
func (h *Hub) SetMessageHandler(handler MessageHandler) {
	h.messageHandler = handler
}

// GetClient finds a client in a room by participant ID
func (h *Hub) GetClient(roomID string, participantID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return nil
	}

	for client := range clients {
		if client.participantID == participantID {
			return client
		}
	}

	return nil
}

// HasOtherClient reports whether a room has a registered client for
// participantID other than exclude. Used by the WebSocket handler's
// disconnect cleanup to avoid marking a participant disconnected (and
// broadcasting participant_left) when another live connection for the same
// participant still exists -- e.g. a page refresh where the new socket has
// already registered, or a second open tab.
//
// exclude must be the client being torn down. It is intentionally excluded
// from the match rather than relying on it having already been removed from
// the hub: Unregister() only queues the removal through the async events
// channel, so at the time this is called the old client may still be
// present in the room's client set.
func (h *Hub) HasOtherClient(roomID string, participantID string, exclude *Client) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return false
	}

	for client := range clients {
		if client == exclude {
			continue
		}
		if client.participantID == participantID {
			return true
		}
	}

	return false
}
