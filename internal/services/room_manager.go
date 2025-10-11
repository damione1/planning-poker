package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/damione1/planning-poker-new/internal/models"
	"github.com/damione1/planning-poker-new/internal/security"
)

type RoomManager struct {
	app core.App
}

func NewRoomManager(app core.App) *RoomManager {
	return &RoomManager{
		app: app,
	}
}

// CreateRoom creates a new room in the database with initial round
func (rm *RoomManager) CreateRoom(name, pointingMethod string, customValues []string) (*core.Record, error) {
	collection, err := rm.app.FindCollectionByNameOrId("rooms")
	if err != nil {
		return nil, fmt.Errorf("failed to find rooms collection: %w", err)
	}

	record := core.NewRecord(collection)
	// Don't set ID manually - PocketBase will auto-generate it
	record.Set("name", name)

	// Default to custom with modified fibonacci if no pointing method specified
	if pointingMethod == "" {
		pointingMethod = "custom"
	}
	record.Set("pointing_method", pointingMethod)

	// Always store custom values (even for fibonacci preset)
	if len(customValues) > 0 {
		// Marshal custom values to JSON
		customValuesJSON, err := json.Marshal(customValues)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal custom values: %w", err)
		}
		record.Set("custom_values", customValuesJSON)
	}

	record.Set("state", string(models.StateVoting))
	record.Set("is_premium", false)
	record.Set("expires_at", time.Now().Add(24*time.Hour))
	record.Set("last_activity", time.Now())
	// creator_participant_id will be set when first participant joins
	// current_round_id will be set after creating first round

	if err := rm.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save room record: %w", err)
	}

	// Create Round 1 for this room
	round, err := rm.CreateRoundForRoom(record.Id, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial round: %w", err)
	}

	// Update room with current round
	record.Set("current_round_id", round.Id)
	if err := rm.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to update room with round: %w", err)
	}

	return record, nil
}

// GetRoom retrieves a room by ID from the database
func (rm *RoomManager) GetRoom(id string) (*core.Record, error) {
	record, err := rm.app.FindRecordById("rooms", id)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}
	return record, nil
}

// UpdateRoomActivity updates the last_activity timestamp
func (rm *RoomManager) UpdateRoomActivity(roomID string) error {
	record, err := rm.GetRoom(roomID)
	if err != nil {
		return err
	}

	record.Set("last_activity", time.Now())
	return rm.app.Save(record)
}

// UpdateRoomState updates the room state (voting/revealed)
// DEPRECATED: State is now managed through rounds only
// This method is kept for backward compatibility during migration
func (rm *RoomManager) UpdateRoomState(roomID string, state models.RoomState) error {
	// State is now managed by rounds, this is a no-op
	// Update activity timestamp only
	return rm.UpdateRoomActivity(roomID)
}

// RevealVotes updates the current round to revealed state
func (rm *RoomManager) RevealVotes(roomID string) error {
	currentRound, err := rm.GetCurrentRoundRecord(roomID)
	if err != nil {
		return fmt.Errorf("failed to get current round: %w", err)
	}

	// Update round state to revealed
	currentRound.Set("state", string(models.RoundStateRevealed))
	if err := rm.app.Save(currentRound); err != nil {
		return fmt.Errorf("failed to reveal votes: %w", err)
	}

	return rm.UpdateRoomActivity(roomID)
}

// GetRoomState derives the current room state from the current round
func (rm *RoomManager) GetRoomState(roomID string) (models.RoomState, error) {
	round, err := rm.GetCurrentRoundRecord(roomID)
	if err != nil {
		return models.StateVoting, err
	}
	return models.RoomState(round.GetString("state")), nil
}

// GetRoomParticipants retrieves all participants for a room
func (rm *RoomManager) GetRoomParticipants(roomID string) ([]*core.Record, error) {
	records, err := rm.app.FindRecordsByFilter(
		"participants",
		"room_id = {:roomId}",
		"",
		100,
		0,
		map[string]any{"roomId": roomID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	return records, nil
}

// AddParticipant creates a new participant in the database
func (rm *RoomManager) AddParticipant(roomID, name string, role models.ParticipantRole, sessionCookie string) (*core.Record, error) {
	collection, err := rm.app.FindCollectionByNameOrId("participants")
	if err != nil {
		return nil, fmt.Errorf("failed to find participants collection: %w", err)
	}

	record := core.NewRecord(collection)
	// Don't set ID manually - PocketBase will auto-generate it
	record.Set("room_id", roomID)
	record.Set("name", name)
	record.Set("role", string(role))
	record.Set("connected", true) // Set to true - participant is joining and will connect via WebSocket
	record.Set("session_cookie", sessionCookie)
	record.Set("joined_at", time.Now())
	record.Set("last_seen", time.Now())

	if err := rm.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save participant: %w", err)
	}

	// Set as room creator if this is the first participant
	room, err := rm.GetRoom(roomID)
	if err == nil && room.GetString("creator_participant_id") == "" {
		room.Set("creator_participant_id", record.Id)
		rm.app.Save(room)
	}

	// Update room activity
	rm.UpdateRoomActivity(roomID)

	return record, nil
}

// UpdateParticipantConnection updates participant connection status
func (rm *RoomManager) UpdateParticipantConnection(participantID string, connected bool) error {
	record, err := rm.app.FindRecordById("participants", participantID)
	if err != nil {
		return fmt.Errorf("participant not found: %w", err)
	}

	record.Set("connected", connected)
	record.Set("last_seen", time.Now())
	return rm.app.Save(record)
}

// GetParticipantBySession retrieves a participant by session cookie and room
func (rm *RoomManager) GetParticipantBySession(roomID, sessionCookie string) (*core.Record, error) {
	records, err := rm.app.FindRecordsByFilter(
		"participants",
		"room_id = {:roomId} && session_cookie = {:session}",
		"",
		1,
		0,
		map[string]any{
			"roomId":  roomID,
			"session": sessionCookie,
		},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("participant not found")
	}
	return records[0], nil
}

// GetParticipant retrieves a participant by ID
func (rm *RoomManager) GetParticipant(participantID string) (*core.Record, error) {
	return rm.app.FindRecordById("participants", participantID)
}

// GetCurrentRound retrieves the current round number for a room
func (rm *RoomManager) GetCurrentRound(roomID string) (int, error) {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return 1, nil
	}

	currentRoundID := room.GetString("current_round_id")
	if currentRoundID == "" {
		return 1, nil // Fallback to round 1
	}

	round, err := rm.app.FindRecordById("rounds", currentRoundID)
	if err != nil {
		return 1, nil
	}

	return round.GetInt("round_number"), nil
}

// GetCurrentRoundRecord retrieves the current round record for a room
func (rm *RoomManager) GetCurrentRoundRecord(roomID string) (*core.Record, error) {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return nil, fmt.Errorf("room not found: %w", err)
	}

	currentRoundID := room.GetString("current_round_id")
	if currentRoundID == "" {
		return nil, fmt.Errorf("no current round set for room")
	}

	round, err := rm.app.FindRecordById("rounds", currentRoundID)
	if err != nil {
		return nil, fmt.Errorf("current round not found: %w", err)
	}

	return round, nil
}

// CastVote records or updates a participant's vote in the database
func (rm *RoomManager) CastVote(roomID, participantID, value string) error {
	fmt.Printf("[DEBUG] CastVote called: roomID=%s, participantID=%s, value=%s\n", roomID, participantID, value)

	// Get current round record
	currentRound, err := rm.GetCurrentRoundRecord(roomID)
	if err != nil {
		return fmt.Errorf("failed to get current round: %w", err)
	}
	currentRoundID := currentRound.Id
	fmt.Printf("[DEBUG] Current round ID: %s, round number: %d\n", currentRoundID, currentRound.GetInt("round_number"))

	// Check if vote already exists for this participant in this round
	existingVotes, err := rm.app.FindRecordsByFilter(
		"votes",
		"participant_id = {:participantId} && round_id = {:roundId}",
		"",
		1,
		0,
		map[string]any{
			"participantId": participantID,
			"roundId":       currentRoundID,
		},
	)

	var record *core.Record
	if err == nil && len(existingVotes) > 0 {
		// Update existing vote
		fmt.Printf("[DEBUG] Updating existing vote\n")
		record = existingVotes[0]
	} else {
		// Create new vote
		fmt.Printf("[DEBUG] Creating new vote (err=%v, count=%d)\n", err, len(existingVotes))
		collection, err := rm.app.FindCollectionByNameOrId("votes")
		if err != nil {
			return fmt.Errorf("failed to find votes collection: %w", err)
		}
		record = core.NewRecord(collection)
		record.Set("participant_id", participantID)
		record.Set("room_id", roomID)
		record.Set("round_id", currentRoundID)
		// Keep round_number for backward compatibility during migration
		record.Set("round_number", currentRound.GetInt("round_number"))
	}

	record.Set("value", value)
	record.Set("voted_at", time.Now())

	fmt.Printf("[DEBUG] About to save vote record with: participant_id=%s, room_id=%s, round_id=%s, value=%s\n",
		record.GetString("participant_id"),
		record.GetString("room_id"),
		record.GetString("round_id"),
		record.GetString("value"))

	if err := rm.app.Save(record); err != nil {
		fmt.Printf("[DEBUG] Failed to save vote: %v\n", err)
		return fmt.Errorf("failed to save vote: %w", err)
	}

	fmt.Printf("[DEBUG] Vote saved successfully with ID: %s\n", record.Id)

	// Update room activity
	return rm.UpdateRoomActivity(roomID)
}

// GetRoomVotes retrieves all votes for a room's current round
func (rm *RoomManager) GetRoomVotes(roomID string) ([]*core.Record, error) {
	currentRound, err := rm.GetCurrentRoundRecord(roomID)
	if err != nil {
		return nil, err
	}

	records, err := rm.app.FindRecordsByFilter(
		"votes",
		"round_id = {:roundId}",
		"",
		100,
		0,
		map[string]any{
			"roundId": currentRound.Id,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes: %w", err)
	}
	return records, nil
}

// ResetRound clears votes for current round and returns to voting state
// Does NOT create a new round - just clears the current one
func (rm *RoomManager) ResetRound(roomID string) error {
	// Get current round
	currentRound, err := rm.GetCurrentRoundRecord(roomID)
	if err != nil {
		return fmt.Errorf("failed to get current round: %w", err)
	}

	// Delete all votes for this round
	votes, err := rm.app.FindRecordsByFilter(
		"votes",
		"round_id = {:roundId}",
		"",
		1000,
		0,
		map[string]any{"roundId": currentRound.Id},
	)
	if err == nil {
		for _, vote := range votes {
			rm.app.Delete(vote)
		}
	}

	// Update round state back to voting
	currentRound.Set("state", string(models.RoundStateVoting))
	if err := rm.app.Save(currentRound); err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}

	// Room state is automatically derived from round state
	return rm.UpdateRoomActivity(roomID)
}

// IsRoomCreator checks if a participant is the room creator
func (rm *RoomManager) IsRoomCreator(roomID, participantID string) bool {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return false
	}
	return room.GetString("creator_participant_id") == participantID
}

// UpdateParticipantName updates a participant's name
func (rm *RoomManager) UpdateParticipantName(participantID, newName string) error {
	// Validate name (should already be validated by caller, but defense in depth)
	sanitizedName, err := security.ValidateParticipantName(newName)
	if err != nil {
		return err
	}

	participant, err := rm.GetParticipant(participantID)
	if err != nil {
		log.Printf("Failed to get participant %s: %v", participantID, err)
		return fmt.Errorf("participant not found")
	}

	participant.Set("name", sanitizedName)
	if err := rm.app.Save(participant); err != nil {
		log.Printf("Failed to save participant name update: %v", err)
		return fmt.Errorf("failed to update participant name")
	}

	return nil
}

// UpdateRoomName updates a room's name
func (rm *RoomManager) UpdateRoomName(roomID, newName string) error {
	// Validate name (should already be validated by caller, but defense in depth)
	sanitizedName, err := security.ValidateRoomName(newName)
	if err != nil {
		return err
	}

	room, err := rm.GetRoom(roomID)
	if err != nil {
		log.Printf("Failed to get room %s: %v", roomID, err)
		return fmt.Errorf("room not found")
	}

	room.Set("name", sanitizedName)
	room.Set("last_activity", time.Now())
	if err := rm.app.Save(room); err != nil {
		log.Printf("Failed to save room name update: %v", err)
		return fmt.Errorf("failed to update room name")
	}

	return nil
}

// CreateRoundForRoom creates a new round for a room
func (rm *RoomManager) CreateRoundForRoom(roomID string, roundNumber int) (*core.Record, error) {
	collection, err := rm.app.FindCollectionByNameOrId("rounds")
	if err != nil {
		return nil, fmt.Errorf("failed to find rounds collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("room_id", roomID)
	record.Set("round_number", roundNumber)
	record.Set("state", string(models.RoundStateVoting))
	record.Set("total_votes", 0)

	if err := rm.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save round: %w", err)
	}

	return record, nil
}

// CompleteRound marks a round as completed and saves statistics
func (rm *RoomManager) CompleteRound(roundID string, avgScore float64, totalVotes int) error {
	round, err := rm.app.FindRecordById("rounds", roundID)
	if err != nil {
		return fmt.Errorf("round not found: %w", err)
	}

	round.Set("state", string(models.RoundStateCompleted))
	round.Set("average_score", avgScore)
	round.Set("total_votes", totalVotes)
	round.Set("completed_at", time.Now())

	if err := rm.app.Save(round); err != nil {
		return fmt.Errorf("failed to complete round: %w", err)
	}

	return nil
}

// CreateNextRound completes the current round and creates a new one
func (rm *RoomManager) CreateNextRound(roomID string) (*core.Record, error) {
	// Get current round
	currentRound, err := rm.GetCurrentRoundRecord(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current round: %w", err)
	}

	// Get votes to calculate statistics
	votes, err := rm.GetRoomVotes(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes: %w", err)
	}

	// Calculate average (supports float values)
	var sum float64
	var count int
	validator := NewVoteValidator()
	for _, vote := range votes {
		value := vote.GetString("value")
		if num, ok := validator.ParseNumericValue(value); ok && num > 0 {
			sum += num
			count++
		}
	}

	var avgScore float64
	if count > 0 {
		avgScore = sum / float64(count)
	}

	// Complete current round with stats
	if err := rm.CompleteRound(currentRound.Id, avgScore, len(votes)); err != nil {
		return nil, fmt.Errorf("failed to complete round: %w", err)
	}

	// Create new round
	nextRoundNumber := currentRound.GetInt("round_number") + 1
	newRound, err := rm.CreateRoundForRoom(roomID, nextRoundNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create next round: %w", err)
	}

	// Update room's current round
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	room.Set("current_round_id", newRound.Id)
	// Room state is automatically derived from round state
	if err := rm.app.Save(room); err != nil {
		return nil, fmt.Errorf("failed to update room: %w", err)
	}

	return newRound, nil
}

