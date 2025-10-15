package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/tests/helpers"
)

// TestAutoRevealDisabledByDefault verifies that auto-reveal is off by default
func TestAutoRevealDisabledByDefault(t *testing.T) {
	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	// Create room with default config
	roomRecord := helpers.CreateTestRoom(t, app, "Test Room", "custom", []string{"1", "2", "3"}, nil)
	if roomRecord == nil {
		t.Fatal("Failed to create test room")
	}

	// Parse config from database record
	var config models.RoomConfig
	if err := json.Unmarshal([]byte(roomRecord.GetString("config")), &config); err != nil {
		t.Fatalf("Failed to parse room config: %v", err)
	}

	// Verify auto-reveal is false by default
	if config.Permissions.AutoReveal {
		t.Error("Expected AutoReveal to be false by default, got true")
	}
}

// TestAutoRevealEnabledInConfig verifies auto-reveal can be enabled
func TestAutoRevealEnabledInConfig(t *testing.T) {
	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	// Create room with auto-reveal enabled
	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	roomRecord := helpers.CreateTestRoom(t, app, "Test Room", "custom", []string{"1", "2", "3"}, config)
	if roomRecord == nil {
		t.Fatal("Failed to create test room")
	}

	// Parse config from database record
	var savedConfig models.RoomConfig
	if err := json.Unmarshal([]byte(roomRecord.GetString("config")), &savedConfig); err != nil {
		t.Fatalf("Failed to parse room config: %v", err)
	}

	// Verify auto-reveal is enabled
	if !savedConfig.Permissions.AutoReveal {
		t.Error("Expected AutoReveal to be true, got false")
	}
}

// TestAutoRevealTriggersCountdown verifies countdown is triggered when all voters vote
func TestAutoRevealTriggersCountdown(t *testing.T) {
	t.Skip("WebSocket integration test requires full HTTP server infrastructure - skipped until server setup is implemented")

	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	// Create room with auto-reveal enabled
	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	roomID := helpers.CreateTestRoomWithParticipants(t, app, 2, config) // 2 voters

	// Start test server and connect WebSocket clients
	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	// Connect both voters
	voter1 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter1.Close()

	voter2 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter2.Close()

	// Wait for initial room state messages
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)

	// Voter 1 casts vote
	voter1.SendVote(t, "3")
	msg1 := voter1.ExpectMessage(t, "vote_cast", 2*time.Second)
	if msg1 == nil {
		t.Fatal("Expected vote_cast message for voter 1")
	}

	// Voter 2 casts vote - should trigger auto-reveal countdown
	voter2.SendVote(t, "5")

	// Both voters should receive vote_cast for voter 2
	msg2a := voter1.ExpectMessage(t, "vote_cast", 2*time.Second)
	if msg2a == nil {
		t.Fatal("Expected vote_cast message for voter 2 on voter1's connection")
	}

	msg2b := voter2.ExpectMessage(t, "vote_cast", 2*time.Second)
	if msg2b == nil {
		t.Fatal("Expected vote_cast message for voter 2 on voter2's connection")
	}

	// Should receive auto_reveal_countdown message
	countdownMsg := voter1.ExpectMessage(t, "auto_reveal_countdown", 2*time.Second)
	if countdownMsg == nil {
		t.Fatal("Expected auto_reveal_countdown message")
	}

	// Verify countdown payload
	payload, ok := countdownMsg.Payload.(map[string]interface{})
	if !ok {
		t.Fatal("Invalid countdown payload format")
	}

	duration, ok := payload["duration"].(float64)
	if !ok || duration != 1500 {
		t.Errorf("Expected duration 1500ms, got %v", payload["duration"])
	}
}

// TestAutoRevealDoesNotTriggerWhenDisabled verifies no countdown when auto-reveal is off
func TestAutoRevealDoesNotTriggerWhenDisabled(t *testing.T) {
	t.Skip("WebSocket integration test requires full HTTP server infrastructure - skipped until server setup is implemented")

	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	// Create room with auto-reveal disabled (default)
	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = false

	roomID := helpers.CreateTestRoomWithParticipants(t, app, 2, config)

	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	voter1 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter1.Close()

	voter2 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter2.Close()

	// Wait for initial state
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)

	// Both voters cast votes
	voter1.SendVote(t, "3")
	voter1.ExpectMessage(t, "vote_cast", 2*time.Second)

	voter2.SendVote(t, "5")
	voter2.ExpectMessage(t, "vote_cast", 2*time.Second)

	// Should NOT receive auto_reveal_countdown message
	// Wait a bit to ensure no countdown message arrives
	time.Sleep(500 * time.Millisecond)

	// Try to read message with very short timeout - should get nothing
	countdownMsg := voter1.TryReadMessage(100 * time.Millisecond)
	if countdownMsg != nil && countdownMsg.Type == "auto_reveal_countdown" {
		t.Error("Expected no auto_reveal_countdown when auto-reveal is disabled")
	}
}

// TestAutoRevealOnlyTriggersWhenAllVotersVoted verifies partial votes don't trigger
func TestAutoRevealOnlyTriggersWhenAllVotersVoted(t *testing.T) {
	t.Skip("WebSocket integration test requires full HTTP server infrastructure - skipped until server setup is implemented")

	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	roomID := helpers.CreateTestRoomWithParticipants(t, app, 3, config) // 3 voters

	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	voter1 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter1.Close()

	voter2 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter2.Close()

	voter3 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter3.Close()

	// Wait for initial state
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)
	voter3.ExpectMessage(t, "room_state", 2*time.Second)

	// Only 2 out of 3 voters vote
	voter1.SendVote(t, "3")
	voter1.ExpectMessage(t, "vote_cast", 2*time.Second)

	voter2.SendVote(t, "5")
	voter2.ExpectMessage(t, "vote_cast", 2*time.Second)

	// Wait and verify no countdown triggered
	time.Sleep(500 * time.Millisecond)
	countdownMsg := voter1.TryReadMessage(100 * time.Millisecond)
	if countdownMsg != nil && countdownMsg.Type == "auto_reveal_countdown" {
		t.Error("Expected no countdown when not all voters have voted")
	}

	// Now third voter votes - should trigger countdown
	voter3.SendVote(t, "8")
	voter3.ExpectMessage(t, "vote_cast", 2*time.Second)

	// Should receive countdown now
	countdownMsg = voter1.ExpectMessage(t, "auto_reveal_countdown", 2*time.Second)
	if countdownMsg == nil {
		t.Error("Expected countdown after all voters voted")
	}
}

// TestAutoRevealWithSpectators verifies spectators don't affect auto-reveal trigger
func TestAutoRevealWithSpectators(t *testing.T) {
	t.Skip("WebSocket integration test requires full HTTP server infrastructure - skipped until server setup is implemented")

	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	// Create room with 2 voters and 1 spectator
	roomID := helpers.CreateTestRoomWithMixedParticipants(t, app, 2, 1, config)

	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	voter1 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter1.Close()

	voter2 := helpers.ConnectTestClient(t, ts, roomID)
	defer voter2.Close()

	// Wait for initial state
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)

	// Both voters vote (spectator doesn't vote)
	voter1.SendVote(t, "3")
	voter1.ExpectMessage(t, "vote_cast", 2*time.Second)

	voter2.SendVote(t, "5")
	voter2.ExpectMessage(t, "vote_cast", 2*time.Second)

	// Should trigger countdown even though spectator hasn't voted
	countdownMsg := voter1.ExpectMessage(t, "auto_reveal_countdown", 2*time.Second)
	if countdownMsg == nil {
		t.Error("Expected countdown when all voters (excluding spectators) have voted")
	}
}
