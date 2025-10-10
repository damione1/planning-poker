package handlers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/coder/websocket"
	"github.com/pocketbase/pocketbase/core"

	"github.com/damiengoehrig/planning-poker/internal/models"
	"github.com/damiengoehrig/planning-poker/internal/services"
)

type WSHandler struct {
	hub         *services.Hub
	roomManager *services.RoomManager
}

func NewWSHandler(hub *services.Hub, rm *services.RoomManager) *WSHandler {
	return &WSHandler{
		hub:         hub,
		roomManager: rm,
	}
}

func (h *WSHandler) HandleWebSocket(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("roomId")

	// Verify room exists
	_, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.JSON(404, map[string]string{"error": "Room not found"})
	}

	// Get participant from session cookie
	sessionCookie := getParticipantID(re.Request)
	var participantID string
	if sessionCookie != "" {
		participantRecord, err := h.roomManager.GetParticipantBySession(roomID, sessionCookie)
		if err == nil {
			participantID = participantRecord.Id
		}
	}

	// Upgrade to WebSocket
	conn, err := websocket.Accept(re.Response, re.Request, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Configure based on environment
	})
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusInternalError, "")

	// Update participant connection status to connected
	if participantID != "" {
		h.roomManager.UpdateParticipantConnection(participantID, true)
	}

	// Register connection with hub
	h.hub.Register(roomID, conn, participantID)
	defer func() {
		h.hub.Unregister(roomID, conn, participantID)
		// Update participant connection status to disconnected
		if participantID != "" {
			h.roomManager.UpdateParticipantConnection(participantID, false)
		}
	}()

	// Message loop
	ctx := context.Background()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}

		var msg models.WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		h.handleMessage(roomID, &msg)
	}

	return nil
}

func (h *WSHandler) handleMessage(roomID string, msg *models.WSMessage) {
	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return
	}

	switch msg.Type {
	case models.MsgTypeVote:
		// Will implement in Phase 7
		log.Printf("Vote message received for room %s", room.Id)
	case models.MsgTypeReveal:
		// Will implement in Phase 7
		log.Printf("Reveal message received for room %s", room.Id)
	case models.MsgTypeReset:
		// Will implement in Phase 7
		log.Printf("Reset message received for room %s", room.Id)
	}
}
