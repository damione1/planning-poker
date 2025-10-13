package handlers

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/security"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/pocketbase/pocketbase/core"
)

type WSHandler struct {
	hub             *services.Hub
	roomManager     *services.RoomManager
	aclService      *services.ACLService
	originValidator *security.OriginValidator
}

func NewWSHandler(hub *services.Hub, rm *services.RoomManager, acl *services.ACLService) *WSHandler {
	// Configure origin validator from environment or use defaults
	allowedOrigins := getWebSocketOrigins()
	originValidator := security.NewOriginValidator(allowedOrigins)

	return &WSHandler{
		hub:             hub,
		roomManager:     rm,
		aclService:      acl,
		originValidator: originValidator,
	}
}

// getWebSocketOrigins returns allowed WebSocket origins from environment
func getWebSocketOrigins() []string {
	// Check for environment variable
	if origins := os.Getenv("WS_ALLOWED_ORIGINS"); origins != "" {
		return strings.Split(origins, ",")
	}

	// Default origins for development
	return []string{
		"localhost:*",
		"127.0.0.1:*",
	}
}

// HandleWebSocket is the optimized WebSocket handler using Client architecture
func (h *WSHandler) HandleWebSocket(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("roomId")

	// Validate room ID
	if err := security.ValidateUUID(roomID); err != nil {
		return re.JSON(400, map[string]string{"error": "Invalid room ID"})
	}

	// Verify room exists
	_, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.JSON(404, map[string]string{"error": "Room not found"})
	}

	// Check if hub can accept new connection
	if err := h.hub.CanRegister(roomID); err != nil {
		log.Printf("Connection rejected: %v", err)
		if err == services.ErrServerAtCapacity {
			return re.JSON(503, map[string]string{"error": "Server at capacity. Please try again later."})
		}
		if err == services.ErrRoomFull {
			return re.JSON(429, map[string]string{"error": "Room is full. Maximum participants reached."})
		}
		return re.JSON(500, map[string]string{"error": "Unable to accept connection"})
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

	// Upgrade to WebSocket with origin validation
	conn, err := websocket.Accept(re.Response, re.Request, h.originValidator.GetAcceptOptions())
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return err
	}

	// Create client instance
	client := services.NewClient(conn, h.hub, roomID, participantID)

	// Update participant connection status to connected
	if participantID != "" {
		_ = h.roomManager.UpdateParticipantConnection(participantID, true) // Best effort

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

	// Set up cleanup on disconnect
	defer func() {
		// Update participant connection status to disconnected
		if participantID != "" {
			_ = h.roomManager.UpdateParticipantConnection(participantID, false) // Best effort

			// Broadcast participant left event
			h.hub.BroadcastToRoom(roomID, &models.WSMessage{
				Type: models.MsgTypeParticipantLeft,
				Payload: map[string]any{
					"participantId": participantID,
				},
			})
		}
	}()

	// Register client with hub (this queues the registration)
	h.hub.Register(roomID, client)

	// Send initial room state to this client
	if err := h.sendInitialRoomStateToClient(client, roomID, participantID); err != nil {
		log.Printf("Failed to send initial room state: %v", err)
	}

	// Start client's read and write pumps (these run in separate goroutines)
	client.Start()

	// Process messages from client (the client's readPump handles reading)
	// We just handle the business logic here
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

		// Validate message type
		if !security.IsValidMessageType(msg.Type) {
			log.Printf("Invalid message type received: %s", msg.Type)
			continue
		}

		// Validate payload structure
		if err := security.ValidateMessagePayload(msg.Type, msg.Payload); err != nil {
			log.Printf("Invalid message payload: %v", err)
			continue
		}

		log.Printf("[DEBUG] Parsed message - Type: %s, Payload: %+v", msg.Type, msg.Payload)
		h.handleMessage(roomID, &msg, participantID)
	}

	return nil
}

// sendInitialRoomStateToClient sends the complete current room state to a newly connected client
func (h *WSHandler) sendInitialRoomStateToClient(client *services.Client, roomID string, participantID string) error {
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

	// Get room record
	roomRecord, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return err
	}

	// Get room state from current round
	roomState, err := h.getRoomState(roomID)
	if err != nil {
		roomState = models.StateVoting // Default if error
	}

	// Get vote count for current round
	votes, _ := h.roomManager.GetRoomVotes(roomID)
	voteCount := len(votes)

	// Check if participant is the room creator
	isCreator := h.roomManager.IsRoomCreator(roomID, participantID)

	// Get permissions for this participant
	canReset, _ := h.aclService.CanReset(roomID, participantID)
	canNewRound, _ := h.aclService.CanTriggerNewRound(roomID, participantID)
	canReveal, _ := h.aclService.CanReveal(roomID, participantID)
	canChangeVoteAfterReveal, _ := h.aclService.CanChangeVoteAfterReveal(roomID)

	// Prepare room state message
	stateMessage := &models.WSMessage{
		Type: models.MsgTypeRoomState,
		Payload: map[string]any{
			"participants":         participants,
			"roomState":            string(roomState),
			"roundNumber":          nil, // Will be filled if available
			"voteCount":            voteCount,
			"isCreator":            isCreator,
			"currentParticipantId": participantID,
			"expiresAt":            roomRecord.GetDateTime("expires_at").Time().Format("2006-01-02T15:04:05Z07:00"), // ISO 8601 format
			"permissions": map[string]any{
				"canReset":                 canReset,
				"canNewRound":              canNewRound,
				"canReveal":                canReveal,
				"canChangeVoteAfterReveal": canChangeVoteAfterReveal,
			},
		},
	}

	// Get current round number
	if currentRound, err := h.roomManager.GetCurrentRound(roomID); err == nil {
		stateMessage.Payload.(map[string]any)["roundNumber"] = currentRound
	}

	// Send message via hub
	h.hub.SendToClient(client, stateMessage)

	log.Printf("Sent initial room state to new connection in room %s (%d participants, %d votes)", roomID, len(participants), voteCount)
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
			Payload: map[string]any{
				"message": "This room has expired. Please create a new room.",
			},
		})
		return
	}

	switch msg.Type {
	case models.MsgTypeVote:
		h.handleVote(roomID, msg, participantID)
	case models.MsgTypeReveal:
		h.handleReveal(roomID, participantID)
	case models.MsgTypeReset:
		h.handleReset(roomID, participantID)
	case models.MsgTypeNextRound:
		h.handleNextRound(roomID, participantID)
	case models.MsgTypeUpdateConfig:
		h.handleUpdateConfig(roomID, msg, participantID)
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

	// Verify room exists
	_, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Room not found: %v", err)
		return
	}

	// Get current room state from round
	roomState, err := h.getRoomState(roomID)
	if err != nil {
		log.Printf("Failed to get room state: %v", err)
		return
	}
	log.Printf("[DEBUG] Room state: %s", roomState)

	// Check if voting is allowed based on room state and permissions
	switch roomState {
	case models.StateVoting:
		// Always allow voting in voting state
	case models.StateRevealed:
		// Check if changing votes after reveal is allowed
		canChange, err := h.aclService.CanChangeVoteAfterReveal(roomID)
		if err != nil {
			log.Printf("Failed to check change vote permission: %v", err)
			return
		}
		if !canChange {
			log.Printf("Vote rejected: changing votes after reveal is not allowed")
			return
		}
	default:
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
	if roomState == models.StateRevealed {
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

func (h *WSHandler) handleReveal(roomID string, participantID string) {
	// ACL Check: Verify participant has permission
	canReveal, err := h.aclService.CanReveal(roomID, participantID)
	if err != nil {
		log.Printf("ACL check failed: %v", err)
		return
	}

	if !canReveal {
		log.Printf("Reveal rejected: participant %s not authorized", participantID)
		return
	}

	// Verify room exists
	_, err = h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Room not found: %v", err)
		return
	}

	// Get current room state from round
	roomState, err := h.getRoomState(roomID)
	if err != nil {
		log.Printf("Failed to get room state: %v", err)
		return
	}

	if roomState != models.StateVoting {
		log.Printf("Reveal rejected: room not in voting state")
		return
	}

	// Reveal votes (updates round state to revealed)
	if err := h.roomManager.RevealVotes(roomID); err != nil {
		log.Printf("Failed to reveal votes: %v", err)
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
	validator := services.NewVoteValidator()
	for value, count := range voteValueCounts {
		total += count
		// Try to parse as number for average calculation
		if num, ok := validator.ParseNumericValue(value); ok && num > 0 {
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
	// ACL Check: Verify participant has permission
	canReset, err := h.aclService.CanReset(roomID, participantID)
	if err != nil {
		log.Printf("ACL check failed: %v", err)
		return
	}

	if !canReset {
		log.Printf("Reset rejected: participant %s not authorized", participantID)
		return
	}

	// Verify room exists
	_, err = h.roomManager.GetRoom(roomID)
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

func (h *WSHandler) handleNextRound(roomID string, participantID string) {
	// ACL Check: Verify participant has permission
	canTrigger, err := h.aclService.CanTriggerNewRound(roomID, participantID)
	if err != nil {
		log.Printf("ACL check failed: %v", err)
		return
	}

	if !canTrigger {
		log.Printf("Next round rejected: participant %s not authorized", participantID)
		return
	}

	// Verify room exists
	_, err = h.roomManager.GetRoom(roomID)
	if err != nil {
		log.Printf("Room not found: %v", err)
		return
	}

	// Get current room state from round
	roomState, err := h.getRoomState(roomID)
	if err != nil {
		log.Printf("Failed to get room state: %v", err)
		return
	}

	if roomState != models.StateRevealed {
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

// getRoomState gets the current room state from the current round
func (h *WSHandler) getRoomState(roomID string) (models.RoomState, error) {
	return h.roomManager.GetRoomState(roomID)
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
	if !ok {
		log.Printf("Invalid name value type")
		return
	}

	// Validate and sanitize name
	sanitizedName, err := security.ValidateParticipantName(newName)
	if err != nil {
		log.Printf("Invalid participant name: %v", err)
		return
	}
	newName = sanitizedName

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
	if !ok {
		log.Printf("Invalid room name value type")
		return
	}

	// Validate and sanitize room name
	sanitizedName, err := security.ValidateRoomName(newName)
	if err != nil {
		log.Printf("Invalid room name: %v", err)
		return
	}
	newName = sanitizedName

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

func (h *WSHandler) handleUpdateConfig(roomID string, msg *models.WSMessage, participantID string) {
	// Verify participant is the room creator
	if !h.roomManager.IsRoomCreator(roomID, participantID) {
		log.Printf("Config update rejected: participant %s is not room creator", participantID)
		return
	}

	// Extract config from payload
	payload, ok := msg.Payload.(map[string]any)
	if !ok {
		log.Printf("Invalid update config payload format")
		return
	}

	configData, ok := payload["config"]
	if !ok {
		log.Printf("Config data missing from payload")
		return
	}

	// Convert to JSON and parse as RoomConfig
	configJSON, err := json.Marshal(configData)
	if err != nil {
		log.Printf("Failed to marshal config data: %v", err)
		return
	}

	var config models.RoomConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		log.Printf("Failed to parse config: %v", err)
		return
	}

	// Update room config
	if err := h.aclService.UpdateRoomConfig(roomID, participantID, &config); err != nil {
		log.Printf("Failed to update room config: %v", err)
		return
	}

	// Broadcast config update to all participants
	// Clients will recalculate their permissions based on config + isCreator flag
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type: models.MsgTypeConfigUpdated,
		Payload: map[string]any{
			"config": config,
		},
	})

	log.Printf("Room config updated for room %s", roomID)
}
