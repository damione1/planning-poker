package helpers

import (
	"testing"
	"time"

	"github.com/damione1/planning-poker-new/internal/models"
)

// CreateTestRoom creates a test room with default values
func CreateTestRoom(name, pointingMethod string, customValues []string) *models.Room {
	return models.NewRoom(
		"test-room-id",
		name,
		pointingMethod,
		customValues,
	)
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
