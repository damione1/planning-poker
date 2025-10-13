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

// Hub manages WebSocket connections and message routing
type Hub struct {
	// Rooms: roomID -> set of clients (using sync.Map for fine-grained locking)
	rooms sync.Map // map[string]map[*Client]bool

	// Connection tracking
	totalConnections int64
	mu               sync.RWMutex

	// Channels
	register      chan *Client
	unregister    chan *Client
	handleMessage chan *ClientMessage

	// Metrics
	metrics *Metrics
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		register:      make(chan *Client, config.HubRegisterBufferSize),
		unregister:    make(chan *Client, config.HubUnregisterBufferSize),
		handleMessage: make(chan *ClientMessage, config.HubBroadcastBufferSize),
		metrics:       NewMetrics(),
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.handleMessage:
			// Message handling will be implemented by WebSocket handler
			// This channel exists for future extensibility
			_ = msg
		}
	}
}

// CanRegister checks if a new connection can be registered
func (h *Hub) CanRegister(roomID string) error {
	h.mu.RLock()
	totalConns := h.totalConnections
	h.mu.RUnlock()

	// Check global connection limit
	if totalConns >= config.MaxTotalConnections {
		return ErrServerAtCapacity
	}

	// Check room-specific limit
	if value, ok := h.rooms.Load(roomID); ok {
		clients := value.(map[*Client]bool)
		if len(clients) >= config.MaxConnectionsPerRoom {
			return ErrRoomFull
		}
	}

	// Check total rooms limit
	roomCount := 0
	h.rooms.Range(func(key, value interface{}) bool {
		roomCount++
		return true
	})

	if roomCount >= config.MaxRoomsPerInstance {
		return ErrServerAtCapacity
	}

	return nil
}

// Register queues a client for registration
func (h *Hub) Register(roomID string, client *Client) {
	h.register <- client
}

// Unregister queues a client for unregistration
func (h *Hub) Unregister(roomID string, client *Client) {
	h.unregister <- client
}

// registerClient adds a client to a room
func (h *Hub) registerClient(client *Client) {
	// Get or create room's client set
	value, _ := h.rooms.LoadOrStore(client.roomID, make(map[*Client]bool))
	clients := value.(map[*Client]bool)

	// Add client to room
	clients[client] = true
	h.rooms.Store(client.roomID, clients)

	// Update global connection count
	h.mu.Lock()
	h.totalConnections++
	h.mu.Unlock()

	h.metrics.IncrementConnections()

	// Increment room count if this is a new room
	if len(clients) == 1 {
		h.metrics.IncrementRooms()
	}

	log.Printf("✓ Client registered: room=%s participant=%s (room size: %d, total connections: %d)",
		client.roomID, client.participantID, len(clients), h.totalConnections)
}

// unregisterClient removes a client from a room
func (h *Hub) unregisterClient(client *Client) {
	value, ok := h.rooms.Load(client.roomID)
	if !ok {
		return
	}

	clients := value.(map[*Client]bool)
	if _, exists := clients[client]; !exists {
		return
	}

	// Remove client from room
	delete(clients, client)
	client.Close()

	// Update global connection count
	h.mu.Lock()
	h.totalConnections--
	h.mu.Unlock()

	h.metrics.DecrementConnections()

	// Clean up empty room
	if len(clients) == 0 {
		h.rooms.Delete(client.roomID)
		h.metrics.DecrementRooms()
		log.Printf("🧹 Room cleaned up: %s", client.roomID)
	} else {
		h.rooms.Store(client.roomID, clients)
	}

	log.Printf("✓ Client unregistered: room=%s participant=%s (room size: %d, total connections: %d)",
		client.roomID, client.participantID, len(clients), h.totalConnections)
}

// BroadcastToRoom sends a message to all clients in a room (non-blocking)
func (h *Hub) BroadcastToRoom(roomID string, message *models.WSMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error marshaling message: %v", err)
		return
	}

	value, ok := h.rooms.Load(roomID)
	if !ok {
		log.Printf("⚠️  Room not found: %s", roomID)
		return
	}

	clients := value.(map[*Client]bool)
	log.Printf("📤 Broadcasting to room %s (%d clients): type=%s", roomID, len(clients), message.Type)

	// Send to all clients in parallel (non-blocking)
	successCount := 0
	for client := range clients {
		if client.Send(data) {
			successCount++
		}
	}

	log.Printf("✓ Broadcast complete: %d/%d clients received message", successCount, len(clients))
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
	value, ok := h.rooms.Load(roomID)
	if !ok {
		return 0
	}

	clients := value.(map[*Client]bool)
	return len(clients)
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
	count := 0
	h.rooms.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
