package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

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

		// Broadcast participant reconnection to notify other users
		participantRecord, err := h.roomManager.GetParticipant(participantID)
		if err == nil {
			participant := &models.Participant{
				ID:        participantRecord.Id,
				Name:      participantRecord.GetString("name"),
				Role:      models.ParticipantRole(participantRecord.GetString("role")),
				Connected: true, // Now connected
				JoinedAt:  participantRecord.GetDateTime("joined_at").Time(),
			}

			h.hub.BroadcastToRoom(roomID, &models.WSMessage{
				Type: models.MsgTypeParticipantJoined,
				Payload: map[string]interface{}{
					"participant": participant,
				},
			})
			log.Printf("Participant reconnected and broadcast: %s (%s)", participant.Name, participantID)
		}
	}

	// Register connection with hub
	h.hub.Register(roomID, conn, participantID)
	defer func() {
		h.hub.Unregister(roomID, conn, participantID)
		// Update participant connection status to disconnected
		if participantID != "" {
			h.roomManager.UpdateParticipantConnection(participantID, false)

			// Broadcast participant left event
			h.hub.BroadcastToRoom(roomID, &models.WSMessage{
				Type: models.MsgTypeParticipantLeft,
				Payload: map[string]any{
					"participantId": participantID,
				},
			})
		}
	}()

	// Send initial room state to this connection immediately after connecting
	if err := h.sendInitialRoomState(conn, roomID, participantID); err != nil {
		log.Printf("Failed to send initial room state: %v", err)
	}

	// Message loop
	ctx := context.Background()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}

		log.Printf("[DEBUG] Raw WebSocket message received: %s", string(data))

		var msg models.WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v, raw data: %s", err, string(data))
			continue
		}

		// Skip HTMX header-only messages (they have no type)
		if msg.Type == "" {
			log.Printf("[DEBUG] Skipping HTMX header-only message")
			continue
		}

		log.Printf("[DEBUG] Parsed message - Type: %s, Payload: %+v", msg.Type, msg.Payload)
		h.handleMessage(roomID, &msg, participantID)
	}

	return nil
}

// isRoomExpired checks if a room has expired based on its expires_at timestamp
func (h *WSHandler) isRoomExpired(roomID string) bool {
	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Error checking room expiration: %v", err)
		return true // Treat errors as expired for safety
	}

	expiresAt := room.GetDateTime("expires_at").Time()
	return time.Now().After(expiresAt)
}

func (h *WSHandler) handleMessage(roomID string, msg *models.WSMessage, participantID string) {
	// Allow name updates regardless of expiration (non-critical actions)
	if msg.Type == models.MsgTypeUpdateName || msg.Type == models.MsgTypeUpdateRoomName {
		switch msg.Type {
		case models.MsgTypeUpdateName:
			h.handleUpdateName(roomID, msg, participantID)
		case models.MsgTypeUpdateRoomName:
			h.handleUpdateRoomName(roomID, msg, participantID)
		}
		return
	}

	// Check room expiration for critical actions (vote, reveal, reset, next_round)
	if h.isRoomExpired(roomID) {
		log.Printf("Action rejected: room %s has expired (type: %s)", roomID, msg.Type)
		// Broadcast expiration message to all connections in this room
		h.hub.BroadcastToRoom(roomID, &models.WSMessage{
			Type: "room_expired",
			Payload: map[string]interface{}{
				"message": "This room has expired. Please create a new room.",
			},
		})
		return
	}

	switch msg.Type {
	case models.MsgTypeVote:
		h.handleVote(roomID, msg, participantID)
	case models.MsgTypeReveal:
		h.handleReveal(roomID)
	case models.MsgTypeReset:
		h.handleReset(roomID, participantID)
	case models.MsgTypeNextRound:
		h.handleNextRound(roomID)
	}
}

func (h *WSHandler) handleVote(roomID string, msg *models.WSMessage, participantID string) {
	log.Printf("[DEBUG] handleVote called: roomID=%s, participantID=%s", roomID, participantID)

	if participantID == "" {
		log.Printf("Vote rejected: no participant ID")
		return
	}

	// Extract vote value from payload
	payload, ok := msg.Payload.(map[string]any)
	if !ok {
		log.Printf("Invalid vote payload format")
		return
	}

	value, ok := payload["value"].(string)
	if !ok {
		log.Printf("Invalid vote value format")
		return
	}
	log.Printf("[DEBUG] Vote value extracted: %s", value)

	// Verify room exists and get state
	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Room not found: %v", err)
		return
	}

	roomState := room.GetString("state")
	log.Printf("[DEBUG] Room state: %s", roomState)

	// Allow voting in both "voting" and "revealed" states
	if roomState != string(models.StateVoting) && roomState != string(models.StateRevealed) {
		log.Printf("Vote rejected: room not in voting or revealed state (current: %s)", roomState)
		return
	}

	// Verify participant exists and is a voter
	participant, err := h.roomManager.GetParticipant(participantID)
	if err != nil {
		log.Printf("Vote rejected: participant not found: %v", err)
		return
	}

	participantRole := participant.GetString("role")
	log.Printf("[DEBUG] Participant role: %s (expected: %s)", participantRole, string(models.RoleVoter))
	if participantRole != string(models.RoleVoter) {
		log.Printf("Vote rejected: participant is not a voter")
		return
	}

	// Save vote to database
	log.Printf("[DEBUG] Calling CastVote: roomID=%s, participantID=%s, value=%s", roomID, participantID, value)
	if err := h.roomManager.CastVote(roomID, participantID, value); err != nil {
		log.Printf("Failed to save vote: %v", err)
		return
	}
	log.Printf("[DEBUG] Vote saved successfully")

	// If room is in revealed state, broadcast the updated vote with value
	// Otherwise, just broadcast vote cast notification without value
	if roomState == string(models.StateRevealed) {
		// Get participant name for the broadcast
		participantName := participant.GetString("name")

		h.hub.BroadcastToRoom(roomID, &models.WSMessage{
			Type: models.MsgTypeVoteUpdated,
			Payload: map[string]any{
				"participantId":   participantID,
				"participantName": participantName,
				"value":           value,
			},
		})
		log.Printf("[DEBUG] Vote update broadcast (revealed state)")
	} else {
		// Broadcast vote cast notification (without revealing the value)
		h.hub.BroadcastToRoom(roomID, &models.WSMessage{
			Type: models.MsgTypeVoteCast,
			Payload: map[string]any{
				"participantId": participantID,
				"hasVoted":      true,
			},
		})
		log.Printf("[DEBUG] Vote cast notification broadcast (voting state)")
	}
}

func (h *WSHandler) handleReveal(roomID string) {
	// Verify room is in voting state
	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Room not found: %v", err)
		return
	}

	if room.GetString("state") != string(models.StateVoting) {
		log.Printf("Reveal rejected: room not in voting state")
		return
	}

	// Update room state to revealed
	if err := h.roomManager.UpdateRoomState(roomID, models.StateRevealed); err != nil {
		log.Printf("Failed to update room state: %v", err)
		return
	}

	// Get all votes for current round
	votes, err := h.roomManager.GetRoomVotes(roomID)
	if err != nil {
		log.Printf("Failed to get votes: %v", err)
		return
	}

	// Get all participants for this room
	participants, err := h.roomManager.GetRoomParticipants(roomID)
	if err != nil {
		log.Printf("Failed to get participants: %v", err)
		return
	}

	// Build vote results map with participant info
	voteResults := make([]map[string]any, 0)
	voteValueCounts := make(map[string]int)

	for _, vote := range votes {
		participantID := vote.GetString("participant_id")
		value := vote.GetString("value")

		// Find participant name
		var participantName string
		for _, p := range participants {
			if p.Id == participantID {
				participantName = p.GetString("name")
				break
			}
		}

		voteResults = append(voteResults, map[string]any{
			"participantId":   participantID,
			"participantName": participantName,
			"value":           value,
		})

		// Count values for statistics
		voteValueCounts[value]++
	}

	// Calculate statistics
	var total int
	var sum float64
	var values []string
	for value, count := range voteValueCounts {
		total += count
		values = append(values, value)
		// Try to parse as number for average calculation
		if num := parseVoteValue(value); num > 0 {
			sum += num * float64(count)
		}
	}

	stats := map[string]any{
		"total":          total,
		"valueBreakdown": voteValueCounts,
	}

	// Add average if we have numeric values
	if sum > 0 && total > 0 {
		stats["average"] = sum / float64(total)
	}

	// Broadcast revealed votes with statistics
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type: models.MsgTypeVotesRevealed,
		Payload: map[string]any{
			"votes": voteResults,
			"stats": stats,
		},
	})
}

func (h *WSHandler) handleReset(roomID string, participantID string) {
	// Verify participant is the room creator
	if !h.roomManager.IsRoomCreator(roomID, participantID) {
		log.Printf("Reset rejected: participant %s is not room creator", participantID)
		return
	}

	// Verify room exists
	_, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Room not found: %v", err)
		return
	}

	// Reset the round (clears votes, returns to voting state, same round)
	if err := h.roomManager.ResetRound(roomID); err != nil {
		log.Printf("Failed to reset round: %v", err)
		return
	}

	// Broadcast room reset
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type:    models.MsgTypeRoomReset,
		Payload: map[string]any{},
	})
}

func (h *WSHandler) handleNextRound(roomID string) {
	// Verify room is in revealed state
	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Room not found: %v", err)
		return
	}

	if room.GetString("state") != string(models.StateRevealed) {
		log.Printf("Next round rejected: room not in revealed state")
		return
	}

	// Create next round (completes current, creates new)
	newRound, err := h.roomManager.CreateNextRound(roomID)
	if err != nil {
		log.Printf("Failed to create next round: %v", err)
		return
	}

	// Broadcast round completed
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type: models.MsgTypeRoundCompleted,
		Payload: map[string]any{
			"newRoundNumber": newRound.GetInt("round_number"),
		},
	})
}

// sendInitialRoomState sends the complete current room state to a newly connected client
func (h *WSHandler) sendInitialRoomState(conn *websocket.Conn, roomID string, participantID string) error {
	// Get all participants for the room
	participantRecords, err := h.roomManager.GetRoomParticipants(roomID)
	if err != nil {
		return err
	}

	// Convert to participant models
	participants := make([]*models.Participant, 0, len(participantRecords))
	for _, pr := range participantRecords {
		participants = append(participants, &models.Participant{
			ID:        pr.Id,
			Name:      pr.GetString("name"),
			Role:      models.ParticipantRole(pr.GetString("role")),
			Connected: pr.GetBool("connected"),
			JoinedAt:  pr.GetDateTime("joined_at").Time(),
		})
	}

	// Get room state
	roomRecord, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return err
	}

	// Get vote count for current round
	votes, _ := h.roomManager.GetRoomVotes(roomID)
	voteCount := len(votes)

	// Check if participant is the room creator
	isCreator := h.roomManager.IsRoomCreator(roomID, participantID)

	// Prepare room state message
	stateMessage := &models.WSMessage{
		Type: models.MsgTypeRoomState,
		Payload: map[string]interface{}{
			"participants":         participants,
			"roomState":            roomRecord.GetString("state"),
			"roundNumber":          nil, // Will be filled if available
			"voteCount":            voteCount,
			"isCreator":            isCreator,
			"currentParticipantId": participantID,
			"expiresAt":            roomRecord.GetDateTime("expires_at").Time().Format("2006-01-02T15:04:05Z07:00"), // ISO 8601 format
		},
	}

	// Get current round number
	if currentRound, err := h.roomManager.GetCurrentRound(roomID); err == nil {
		stateMessage.Payload.(map[string]interface{})["roundNumber"] = currentRound
	}

	// Send message to the connecting client
	ctx := context.Background()
	msgData, err := json.Marshal(stateMessage)
	if err != nil {
		return err
	}

	if err := conn.Write(ctx, websocket.MessageText, msgData); err != nil {
		return err
	}

	log.Printf("Sent initial room state to new connection in room %s (%d participants, %d votes)", roomID, len(participants), voteCount)
	return nil
}

// parseVoteValue attempts to parse vote value as number (supports floats)
func parseVoteValue(value string) float64 {
	// Use VoteValidator for consistent parsing
	validator := services.NewVoteValidator()
	if num, ok := validator.ParseNumericValue(value); ok {
		return num
	}
	return 0
}

func (h *WSHandler) handleUpdateName(roomID string, msg *models.WSMessage, participantID string) {
	if participantID == "" {
		log.Printf("Update name rejected: no participant ID")
		return
	}

	// Extract new name from payload
	payload, ok := msg.Payload.(map[string]any)
	if !ok {
		log.Printf("Invalid update name payload format")
		return
	}

	newName, ok := payload["name"].(string)
	if !ok || newName == "" {
		log.Printf("Invalid or empty name value")
		return
	}

	// Update participant name in database
	if err := h.roomManager.UpdateParticipantName(participantID, newName); err != nil {
		log.Printf("Failed to update participant name: %v", err)
		return
	}

	// Broadcast name update to all clients in the room
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type: models.MsgTypeNameUpdated,
		Payload: map[string]any{
			"participantId": participantID,
			"name":          newName,
		},
	})

	log.Printf("Participant name updated: %s -> %s", participantID, newName)
}

func (h *WSHandler) handleUpdateRoomName(roomID string, msg *models.WSMessage, participantID string) {
	// Verify participant is the room creator
	if !h.roomManager.IsRoomCreator(roomID, participantID) {
		log.Printf("Update room name rejected: participant %s is not room creator", participantID)
		return
	}

	// Extract new name from payload
	payload, ok := msg.Payload.(map[string]any)
	if !ok {
		log.Printf("Invalid update room name payload format")
		return
	}

	newName, ok := payload["name"].(string)
	if !ok || newName == "" {
		log.Printf("Invalid or empty room name value")
		return
	}

	// Update room name in database
	if err := h.roomManager.UpdateRoomName(roomID, newName); err != nil {
		log.Printf("Failed to update room name: %v", err)
		return
	}

	// Broadcast room name update to all clients
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type: models.MsgTypeRoomNameUpdated,
		Payload: map[string]any{
			"name": newName,
		},
	})

	log.Printf("Room name updated: %s -> %s", roomID, newName)
}
