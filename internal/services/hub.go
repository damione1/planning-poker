package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/coder/websocket"
	"github.com/damione1/planning-poker/internal/models"
)

type Hub struct {
	// Room connections: roomId -> set of connections
	rooms map[string]map[*websocket.Conn]bool

	// Connection to participant mapping
	connToParticipant map[*websocket.Conn]string

	// Broadcast message to room
	broadcast chan *BroadcastMessage

	// Register connection to room
	register chan *Registration

	// Unregister connection from room
	unregister chan *Registration

	mu sync.RWMutex
}

type Registration struct {
	RoomID        string
	Conn          *websocket.Conn
	ParticipantID string
}

type BroadcastMessage struct {
	RoomID  string
	Message *models.WSMessage
}

func NewHub() *Hub {
	return &Hub{
		rooms:             make(map[string]map[*websocket.Conn]bool),
		connToParticipant: make(map[*websocket.Conn]string),
		broadcast:         make(chan *BroadcastMessage, 256),
		register:          make(chan *Registration),
		unregister:        make(chan *Registration),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.registerConnection(reg)

		case reg := <-h.unregister:
			h.unregisterConnection(reg)

		case msg := <-h.broadcast:
			h.broadcastToRoom(msg)
		}
	}
}

func (h *Hub) registerConnection(reg *Registration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[reg.RoomID] == nil {
		h.rooms[reg.RoomID] = make(map[*websocket.Conn]bool)
	}
	h.rooms[reg.RoomID][reg.Conn] = true

	// Track participant connection if provided
	if reg.ParticipantID != "" {
		h.connToParticipant[reg.Conn] = reg.ParticipantID
	}

	log.Printf("✓ WebSocket registered: room=%s participant=%s (total connections in room: %d)",
		reg.RoomID, reg.ParticipantID, len(h.rooms[reg.RoomID]))
}

func (h *Hub) unregisterConnection(reg *Registration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if connections, ok := h.rooms[reg.RoomID]; ok {
		if _, exists := connections[reg.Conn]; exists {
			delete(connections, reg.Conn)
			delete(h.connToParticipant, reg.Conn)
			reg.Conn.Close(websocket.StatusNormalClosure, "")

			// Clean up empty rooms
			if len(connections) == 0 {
				delete(h.rooms, reg.RoomID)
			}
		}
	}
}

func (h *Hub) broadcastToRoom(msg *BroadcastMessage) {
	h.mu.RLock()
	connections := h.rooms[msg.RoomID]
	h.mu.RUnlock()

	if connections == nil {
		log.Printf("⚠️  No connections in room %s", msg.RoomID)
		return
	}

	data, err := json.Marshal(msg.Message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	log.Printf("📤 Broadcasting to room %s (%d connections): %s", msg.RoomID, len(connections), string(data))

	for conn := range connections {
		go func(c *websocket.Conn) {
			err := c.Write(context.Background(), websocket.MessageText, data)
			if err != nil {
				log.Printf("Error writing to WebSocket: %v", err)
			} else {
				log.Printf("✓ Message sent to connection")
			}
		}(conn)
	}
}

func (h *Hub) BroadcastToRoom(roomID string, message *models.WSMessage) {
	h.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: message,
	}
}

func (h *Hub) Register(roomID string, conn *websocket.Conn, participantID string) {
	h.register <- &Registration{
		RoomID:        roomID,
		Conn:          conn,
		ParticipantID: participantID,
	}
}

func (h *Hub) Unregister(roomID string, conn *websocket.Conn, participantID string) {
	h.unregister <- &Registration{
		RoomID:        roomID,
		Conn:          conn,
		ParticipantID: participantID,
	}
}
