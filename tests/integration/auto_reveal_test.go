package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/tests/helpers"
)

// autoRevealWait is a generous timeout for assertions that must wait past
// the server's config.AutoRevealDelay (1500ms) for the auto-scheduled
// votes_revealed broadcast (see internal/handlers/ws.go's handleVote ->
// time.AfterFunc(wslimits.AutoRevealDelay, ...) -> doReveal).
const autoRevealWait = 5 * time.Second

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
	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	// Create room with auto-reveal enabled
	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	rm := services.NewRoomManager(app)
	room, err := rm.CreateRoom("Test Room", "fibonacci", nil, config)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}
	roomID := room.Id

	p1, err := rm.AddParticipant(roomID, "Voter1", models.RoleVoter, "voter1-session")
	if err != nil {
		t.Fatalf("Failed to add voter1: %v", err)
	}
	p2, err := rm.AddParticipant(roomID, "Voter2", models.RoleVoter, "voter2-session")
	if err != nil {
		t.Fatalf("Failed to add voter2: %v", err)
	}

	// Start test server and connect WebSocket clients
	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	// Connect both voters as their real participants (session-cookie backed,
	// so their votes are accepted by the server's membership gate).
	voter1 := helpers.ConnectTestClientAs(t, ts, roomID, p1.Id, "voter1-session")
	defer voter1.Close()

	voter2 := helpers.ConnectTestClientAs(t, ts, roomID, p2.Id, "voter2-session")
	defer voter2.Close()

	// Wait for initial room state messages
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)

	// Voter 1 casts vote
	voter1.SendVote(t, "3")
	if msgs := voter1.WaitForMessageCount("vote_cast", 1, 2*time.Second); msgs == nil {
		t.Fatal("Expected vote_cast message for voter 1")
	}

	// Voter 2 casts vote - should trigger auto-reveal countdown. Both voters
	// should now have received 2 vote_cast broadcasts (one per vote).
	voter2.SendVote(t, "5")
	if msgs := voter1.WaitForMessageCount("vote_cast", 2, 2*time.Second); msgs == nil {
		t.Fatal("Expected vote_cast message for voter 2 on voter1's connection")
	}
	if msgs := voter2.WaitForMessageCount("vote_cast", 2, 2*time.Second); msgs == nil {
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

	// Auto-reveal is server-authoritative: the countdown is cosmetic and the
	// server schedules the actual reveal itself via time.AfterFunc, rather
	// than waiting for a client to send "reveal". Verify the reveal actually
	// happens once the delay elapses.
	revealMsg := voter1.ExpectMessage(t, "votes_revealed", autoRevealWait)
	if revealMsg == nil {
		t.Fatal("Expected votes_revealed broadcast after auto-reveal delay")
	}
}

// TestAutoRevealDoesNotTriggerWhenDisabled verifies no countdown when auto-reveal is off
func TestAutoRevealDoesNotTriggerWhenDisabled(t *testing.T) {
	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	// Create room with auto-reveal disabled (default)
	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = false

	rm := services.NewRoomManager(app)
	room, err := rm.CreateRoom("Test Room", "fibonacci", nil, config)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}
	roomID := room.Id

	p1, err := rm.AddParticipant(roomID, "Voter1", models.RoleVoter, "voter1-session")
	if err != nil {
		t.Fatalf("Failed to add voter1: %v", err)
	}
	p2, err := rm.AddParticipant(roomID, "Voter2", models.RoleVoter, "voter2-session")
	if err != nil {
		t.Fatalf("Failed to add voter2: %v", err)
	}

	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	voter1 := helpers.ConnectTestClientAs(t, ts, roomID, p1.Id, "voter1-session")
	defer voter1.Close()

	voter2 := helpers.ConnectTestClientAs(t, ts, roomID, p2.Id, "voter2-session")
	defer voter2.Close()

	// Wait for initial state
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)

	// Both voters cast votes
	voter1.SendVote(t, "3")
	voter1.ExpectMessage(t, "vote_cast", 2*time.Second)

	voter2.SendVote(t, "5")
	voter2.ExpectMessage(t, "vote_cast", 2*time.Second)

	// Should NOT receive auto_reveal_countdown message. Bounded drain window
	// well past the point at which the countdown would have been broadcast
	// (broadcast happens synchronously in handleVote, immediately after the
	// second vote_cast), so this is not a race against server timing.
	voter1.ExpectNoMessageType(t, "auto_reveal_countdown", 500*time.Millisecond)
}

// TestAutoRevealOnlyTriggersWhenAllVotersVoted verifies partial votes don't trigger
func TestAutoRevealOnlyTriggersWhenAllVotersVoted(t *testing.T) {
	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	rm := services.NewRoomManager(app)
	room, err := rm.CreateRoom("Test Room", "fibonacci", nil, config)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}
	roomID := room.Id

	p1, err := rm.AddParticipant(roomID, "Voter1", models.RoleVoter, "voter1-session")
	if err != nil {
		t.Fatalf("Failed to add voter1: %v", err)
	}
	p2, err := rm.AddParticipant(roomID, "Voter2", models.RoleVoter, "voter2-session")
	if err != nil {
		t.Fatalf("Failed to add voter2: %v", err)
	}
	p3, err := rm.AddParticipant(roomID, "Voter3", models.RoleVoter, "voter3-session")
	if err != nil {
		t.Fatalf("Failed to add voter3: %v", err)
	}

	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	voter1 := helpers.ConnectTestClientAs(t, ts, roomID, p1.Id, "voter1-session")
	defer voter1.Close()

	voter2 := helpers.ConnectTestClientAs(t, ts, roomID, p2.Id, "voter2-session")
	defer voter2.Close()

	voter3 := helpers.ConnectTestClientAs(t, ts, roomID, p3.Id, "voter3-session")
	defer voter3.Close()

	// Wait for initial state
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)
	voter3.ExpectMessage(t, "room_state", 2*time.Second)

	// Only 2 out of 3 voters vote
	voter1.SendVote(t, "3")
	voter1.ExpectMessage(t, "vote_cast", 2*time.Second)

	voter2.SendVote(t, "5")
	if msgs := voter1.WaitForMessageCount("vote_cast", 2, 2*time.Second); msgs == nil {
		t.Fatal("Expected 2 vote_cast broadcasts after voter1 and voter2 voted")
	}

	// Verify no countdown triggered within a bounded window
	voter1.ExpectNoMessageType(t, "auto_reveal_countdown", 500*time.Millisecond)

	// Now third voter votes - should trigger countdown
	voter3.SendVote(t, "8")
	if msgs := voter1.WaitForMessageCount("vote_cast", 3, 2*time.Second); msgs == nil {
		t.Fatal("Expected 3 vote_cast broadcasts after all voters voted")
	}

	// Should receive countdown now
	countdownMsg := voter1.ExpectMessage(t, "auto_reveal_countdown", 2*time.Second)
	if countdownMsg == nil {
		t.Fatal("Expected countdown after all voters voted")
	}

	// And the server-scheduled reveal should follow after the delay.
	revealMsg := voter1.ExpectMessage(t, "votes_revealed", autoRevealWait)
	if revealMsg == nil {
		t.Fatal("Expected votes_revealed broadcast after auto-reveal delay")
	}
}

// TestAutoRevealWithSpectators verifies spectators don't affect auto-reveal trigger
func TestAutoRevealWithSpectators(t *testing.T) {
	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true

	rm := services.NewRoomManager(app)
	room, err := rm.CreateRoom("Test Room", "fibonacci", nil, config)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}
	roomID := room.Id

	// Create room with 2 voters and 1 spectator
	p1, err := rm.AddParticipant(roomID, "Voter1", models.RoleVoter, "voter1-session")
	if err != nil {
		t.Fatalf("Failed to add voter1: %v", err)
	}
	p2, err := rm.AddParticipant(roomID, "Voter2", models.RoleVoter, "voter2-session")
	if err != nil {
		t.Fatalf("Failed to add voter2: %v", err)
	}
	if _, err := rm.AddParticipant(roomID, "Spectator1", models.RoleSpectator, "spectator1-session"); err != nil {
		t.Fatalf("Failed to add spectator: %v", err)
	}

	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	voter1 := helpers.ConnectTestClientAs(t, ts, roomID, p1.Id, "voter1-session")
	defer voter1.Close()

	voter2 := helpers.ConnectTestClientAs(t, ts, roomID, p2.Id, "voter2-session")
	defer voter2.Close()

	// Wait for initial state
	voter1.ExpectMessage(t, "room_state", 2*time.Second)
	voter2.ExpectMessage(t, "room_state", 2*time.Second)

	// Both voters vote (spectator doesn't vote and never connects)
	voter1.SendVote(t, "3")
	voter1.ExpectMessage(t, "vote_cast", 2*time.Second)

	voter2.SendVote(t, "5")
	if msgs := voter1.WaitForMessageCount("vote_cast", 2, 2*time.Second); msgs == nil {
		t.Fatal("Expected 2 vote_cast broadcasts after both voters voted")
	}

	// Should trigger countdown even though spectator hasn't voted
	countdownMsg := voter1.ExpectMessage(t, "auto_reveal_countdown", 2*time.Second)
	if countdownMsg == nil {
		t.Fatal("Expected countdown when all voters (excluding spectators) have voted")
	}

	revealMsg := voter1.ExpectMessage(t, "votes_revealed", autoRevealWait)
	if revealMsg == nil {
		t.Fatal("Expected votes_revealed broadcast after auto-reveal delay")
	}
}
