package helpers

import (
	"fmt"
	"testing"
	"time"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/pocketbase/pocketbase/core"
)

// SetupTestApp creates a test PocketBase app with migrations and returns cleanup function
func SetupTestApp(t *testing.T) (core.App, func()) {
	t.Helper()
	ts := NewTestServerWithData(t)
	return ts.App, ts.Cleanup
}

// CreateTestRoom creates a test room in the database with optional config
func CreateTestRoom(t *testing.T, app core.App, name, pointingMethod string, customValues []string, config *models.RoomConfig) *core.Record {
	t.Helper()

	rm := services.NewRoomManager(app)
	record, err := rm.CreateRoom(name, pointingMethod, customValues, config)
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	return record
}

// CreateTestRoomWithParticipants creates a room with N voter participants
func CreateTestRoomWithParticipants(t *testing.T, app core.App, voterCount int, config *models.RoomConfig) string {
	t.Helper()

	rm := services.NewRoomManager(app)
	record, err := rm.CreateRoom("Test Room", "fibonacci", nil, config)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	roomID := record.Id

	// Add voter participants
	for i := 0; i < voterCount; i++ {
		name := fmt.Sprintf("Voter%d", i+1)
		sessionCookie := fmt.Sprintf("session-%d", i+1)
		_, err := rm.AddParticipant(roomID, name, models.RoleVoter, sessionCookie)
		if err != nil {
			t.Fatalf("Failed to add participant %s: %v", name, err)
		}
	}

	return roomID
}

// CreateTestRoomWithMixedParticipants creates a room with voters and spectators
func CreateTestRoomWithMixedParticipants(t *testing.T, app core.App, voterCount, spectatorCount int, config *models.RoomConfig) string {
	t.Helper()

	rm := services.NewRoomManager(app)
	record, err := rm.CreateRoom("Test Room", "fibonacci", nil, config)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	roomID := record.Id

	// Add voter participants
	for i := 0; i < voterCount; i++ {
		name := fmt.Sprintf("Voter%d", i+1)
		sessionCookie := fmt.Sprintf("voter-session-%d", i+1)
		_, err := rm.AddParticipant(roomID, name, models.RoleVoter, sessionCookie)
		if err != nil {
			t.Fatalf("Failed to add voter %s: %v", name, err)
		}
	}

	// Add spectator participants
	for i := 0; i < spectatorCount; i++ {
		name := fmt.Sprintf("Spectator%d", i+1)
		sessionCookie := fmt.Sprintf("spectator-session-%d", i+1)
		_, err := rm.AddParticipant(roomID, name, models.RoleSpectator, sessionCookie)
		if err != nil {
			t.Fatalf("Failed to add spectator %s: %v", name, err)
		}
	}

	return roomID
}

// CreateTestParticipant creates a test participant
func CreateTestParticipant(id, name string, role models.ParticipantRole) *models.Participant {
	return models.NewParticipant(id, name, role)
}

// CreateTestRound creates a test round
func CreateTestRound(roomID string, roundNumber int) *models.Round {
	return &models.Round{
		RoomID:      roomID,
		RoundNumber: roundNumber,
		State:       models.RoundStateVoting,
		TotalVotes:  0,
	}
}

// CreateTestWSMessage creates a test WebSocket message
func CreateTestWSMessage(msgType string, payload map[string]interface{}) *models.WSMessage {
	return &models.WSMessage{
		Type:    msgType,
		Payload: payload,
	}
}

// AssertTimeRecent checks if a time is within the last second
func AssertTimeRecent(t *testing.T, timestamp time.Time, message string) {
	t.Helper()
	now := time.Now()
	diff := now.Sub(timestamp)
	if diff > time.Second || diff < 0 {
		t.Errorf("%s: expected recent time, got %v (diff: %v)", message, timestamp, diff)
	}
}

// FibonacciValues returns the standard Fibonacci sequence for testing
func FibonacciValues() []string {
	return []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?"}
}

// TShirtValues returns T-shirt sizing values for testing
func TShirtValues() []string {
	return []string{"XS", "S", "M", "L", "XL", "XXL"}
}
