package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/damiengoehrig/planning-poker/internal/models"
)

type RoomManager struct {
	app core.App
}

func NewRoomManager(app core.App) *RoomManager {
	return &RoomManager{
		app: app,
	}
}

// CreateRoom creates a new room in the database
func (rm *RoomManager) CreateRoom(name, pointingMethod string, customValues []string) (*core.Record, error) {
	collection, err := rm.app.FindCollectionByNameOrId("rooms")
	if err != nil {
		return nil, fmt.Errorf("failed to find rooms collection: %w", err)
	}

	record := core.NewRecord(collection)
	// Don't set ID manually - PocketBase will auto-generate it
	record.Set("name", name)

	// Default to fibonacci if no pointing method specified
	if pointingMethod == "" {
		pointingMethod = "fibonacci"
	}
	record.Set("pointing_method", pointingMethod)

	if pointingMethod == "custom" && len(customValues) > 0 {
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

	if err := rm.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save room record: %w", err)
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
func (rm *RoomManager) UpdateRoomState(roomID string, state models.RoomState) error {
	record, err := rm.GetRoom(roomID)
	if err != nil {
		return err
	}

	record.Set("state", string(state))
	record.Set("last_activity", time.Now())
	return rm.app.Save(record)
}

// GetRoomParticipants retrieves all participants for a room
func (rm *RoomManager) GetRoomParticipants(roomID string) ([]*core.Record, error) {
	records, err := rm.app.FindRecordsByFilter(
		"participants",
		"room_id = {:roomId}",
		"",
		100,
		0,
		map[string]interface{}{"roomId": roomID},
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
	record.Set("connected", false)
	record.Set("session_cookie", sessionCookie)
	record.Set("joined_at", time.Now())
	record.Set("last_seen", time.Now())

	if err := rm.app.Save(record); err != nil {
		return nil, fmt.Errorf("failed to save participant: %w", err)
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
		map[string]interface{}{
			"roomId":  roomID,
			"session": sessionCookie,
		},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("participant not found")
	}
	return records[0], nil
}
