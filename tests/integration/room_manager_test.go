package integration_test

import (
	"testing"
	"time"

	"github.com/damiengoehrig/planning-poker/internal/models"
	"github.com/damiengoehrig/planning-poker/internal/services"
	"github.com/damiengoehrig/planning-poker/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func TestRoomManager_CreateRoom(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)

	t.Run("creates room successfully with fibonacci", func(t *testing.T) {
		room, err := rm.CreateRoom("Sprint Planning", "fibonacci", nil)

		assert.NoError(t, err)
		assert.NotEmpty(t, room.Id)
		assert.Equal(t, "Sprint Planning", room.GetString("name"))
		assert.Equal(t, "fibonacci", room.GetString("pointing_method"))
		assert.Equal(t, string(models.StateVoting), room.GetString("state"))

		// Should have created initial round
		assert.NotEmpty(t, room.GetString("current_round_id"))
	})

	t.Run("creates room with custom values", func(t *testing.T) {
		customValues := []string{"XS", "S", "M", "L", "XL"}
		room, err := rm.CreateRoom("Design Review", "custom", customValues)

		assert.NoError(t, err)
		assert.NotEmpty(t, room.Id)
		assert.Equal(t, "custom", room.GetString("pointing_method"))

		// Custom values are stored as JSON
		assert.NotNil(t, room.Get("custom_values"))
	})

	t.Run("creates room with default pointing method", func(t *testing.T) {
		room, err := rm.CreateRoom("Test Room", "", nil)

		assert.NoError(t, err)
		assert.Equal(t, "custom", room.GetString("pointing_method")) // Defaults to custom
	})
}

func TestRoomManager_GetRoom(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)

	t.Run("retrieves existing room", func(t *testing.T) {
		created, err := rm.CreateRoom("Test", "fibonacci", nil)
		assert.NoError(t, err)

		room, err := rm.GetRoom(created.Id)

		assert.NoError(t, err)
		assert.Equal(t, created.Id, room.Id)
		assert.Equal(t, "Test", room.GetString("name"))
	})

	t.Run("returns error for non-existent room", func(t *testing.T) {
		_, err := rm.GetRoom("non-existent-id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRoomManager_UpdateRoomActivity(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	t.Run("updates last activity timestamp", func(t *testing.T) {
		originalActivity := room.GetDateTime("last_activity")

		// Sleep briefly to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		err := rm.UpdateRoomActivity(room.Id)

		assert.NoError(t, err)

		updated, _ := rm.GetRoom(room.Id)
		newActivity := updated.GetDateTime("last_activity")

		assert.True(t, newActivity.After(originalActivity), "Expected newActivity %v to be after originalActivity %v", newActivity, originalActivity)
	})
}

func TestRoomManager_AddParticipant(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	t.Run("adds voter participant", func(t *testing.T) {
		participant, err := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "session-1")

		assert.NoError(t, err)
		assert.NotEmpty(t, participant.Id)
		assert.Equal(t, "Alice", participant.GetString("name"))
		assert.Equal(t, string(models.RoleVoter), participant.GetString("role"))
		assert.Equal(t, room.Id, participant.GetString("room_id"))
		assert.True(t, participant.GetBool("connected"))
	})

	t.Run("adds spectator participant", func(t *testing.T) {
		participant, err := rm.AddParticipant(room.Id, "Bob", models.RoleSpectator, "session-2")

		assert.NoError(t, err)
		assert.Equal(t, string(models.RoleSpectator), participant.GetString("role"))
	})

	t.Run("sets first participant as room creator", func(t *testing.T) {
		newRoom, _ := rm.CreateRoom("New Room", "fibonacci", nil)
		participant, _ := rm.AddParticipant(newRoom.Id, "Creator", models.RoleVoter, "session-3")

		updatedRoom, _ := rm.GetRoom(newRoom.Id)
		assert.Equal(t, participant.Id, updatedRoom.GetString("creator_participant_id"))
	})
}

func TestRoomManager_GetRoomParticipants(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	t.Run("returns empty list for room with no participants", func(t *testing.T) {
		participants, err := rm.GetRoomParticipants(room.Id)

		assert.NoError(t, err)
		assert.Empty(t, participants)
	})

	t.Run("returns all participants in room", func(t *testing.T) {
		rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
		rm.AddParticipant(room.Id, "Charlie", models.RoleSpectator, "s3")

		participants, err := rm.GetRoomParticipants(room.Id)

		assert.NoError(t, err)
		assert.Len(t, participants, 3)
	})
}

func TestRoomManager_UpdateParticipantConnection(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	participant, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")

	t.Run("updates connection status to disconnected", func(t *testing.T) {
		err := rm.UpdateParticipantConnection(participant.Id, false)

		assert.NoError(t, err)

		updated, _ := rm.GetParticipant(participant.Id)
		assert.False(t, updated.GetBool("connected"))
	})

	t.Run("updates connection status to connected", func(t *testing.T) {
		rm.UpdateParticipantConnection(participant.Id, false)

		err := rm.UpdateParticipantConnection(participant.Id, true)

		assert.NoError(t, err)

		updated, _ := rm.GetParticipant(participant.Id)
		assert.True(t, updated.GetBool("connected"))
	})
}

func TestRoomManager_GetParticipantBySession(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	participant, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "session-123")

	t.Run("finds participant by session cookie", func(t *testing.T) {
		found, err := rm.GetParticipantBySession(room.Id, "session-123")

		assert.NoError(t, err)
		assert.Equal(t, participant.Id, found.Id)
	})

	t.Run("returns error for invalid session", func(t *testing.T) {
		_, err := rm.GetParticipantBySession(room.Id, "invalid-session")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRoomManager_CastVote(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	participant, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")

	t.Run("records new vote successfully", func(t *testing.T) {
		err := rm.CastVote(room.Id, participant.Id, "5")

		assert.NoError(t, err)

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 1)
		assert.Equal(t, "5", votes[0].GetString("value"))
		assert.Equal(t, participant.Id, votes[0].GetString("participant_id"))
	})

	t.Run("updates existing vote", func(t *testing.T) {
		rm.CastVote(room.Id, participant.Id, "5")

		err := rm.CastVote(room.Id, participant.Id, "8")

		assert.NoError(t, err)

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 1) // Should still be 1 vote
		assert.Equal(t, "8", votes[0].GetString("value")) // But with updated value
	})

	t.Run("multiple participants can vote", func(t *testing.T) {
		p2, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
		p3, _ := rm.AddParticipant(room.Id, "Charlie", models.RoleVoter, "s3")

		rm.CastVote(room.Id, participant.Id, "5")
		rm.CastVote(room.Id, p2.Id, "8")
		rm.CastVote(room.Id, p3.Id, "13")

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 3)
	})
}

func TestRoomManager_GetRoomVotes(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	t.Run("returns empty for no votes", func(t *testing.T) {
		votes, err := rm.GetRoomVotes(room.Id)

		assert.NoError(t, err)
		assert.Empty(t, votes)
	})

	t.Run("returns all votes for current round", func(t *testing.T) {
		p1, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		p2, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

		rm.CastVote(room.Id, p1.Id, "5")
		rm.CastVote(room.Id, p2.Id, "8")

		votes, err := rm.GetRoomVotes(room.Id)

		assert.NoError(t, err)
		assert.Len(t, votes, 2)
	})
}

func TestRoomManager_RevealVotes(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	p1, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	p2, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	rm.CastVote(room.Id, p1.Id, "5")
	rm.CastVote(room.Id, p2.Id, "8")

	t.Run("transitions to revealed state", func(t *testing.T) {
		err := rm.RevealVotes(room.Id)

		assert.NoError(t, err)

		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateRevealed, state)
	})

	t.Run("updates current round state", func(t *testing.T) {
		rm.RevealVotes(room.Id)

		currentRound, _ := rm.GetCurrentRoundRecord(room.Id)
		assert.Equal(t, string(models.RoundStateRevealed), currentRound.GetString("state"))
	})
}

func TestRoomManager_GetRoomState(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	t.Run("returns voting state by default", func(t *testing.T) {
		state, err := rm.GetRoomState(room.Id)

		assert.NoError(t, err)
		assert.Equal(t, models.StateVoting, state)
	})

	t.Run("returns revealed state after reveal", func(t *testing.T) {
		p, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		rm.CastVote(room.Id, p.Id, "5")
		rm.RevealVotes(room.Id)

		state, err := rm.GetRoomState(room.Id)

		assert.NoError(t, err)
		assert.Equal(t, models.StateRevealed, state)
	})
}

func TestRoomManager_ResetRound(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	p1, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	p2, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	rm.CastVote(room.Id, p1.Id, "5")
	rm.CastVote(room.Id, p2.Id, "8")
	rm.RevealVotes(room.Id)

	t.Run("clears all votes for current round", func(t *testing.T) {
		err := rm.ResetRound(room.Id)

		assert.NoError(t, err)

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Empty(t, votes)
	})

	t.Run("returns to voting state", func(t *testing.T) {
		rm.ResetRound(room.Id)

		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)
	})

	t.Run("does not create new round", func(t *testing.T) {
		currentRound, _ := rm.GetCurrentRound(room.Id)

		rm.ResetRound(room.Id)

		newRound, _ := rm.GetCurrentRound(room.Id)
		assert.Equal(t, currentRound, newRound) // Same round number
	})
}

func TestRoomManager_CreateNextRound(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	p1, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	p2, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	rm.CastVote(room.Id, p1.Id, "5")
	rm.CastVote(room.Id, p2.Id, "8")
	rm.RevealVotes(room.Id)

	t.Run("creates new round with incremented number", func(t *testing.T) {
		currentRoundNum, _ := rm.GetCurrentRound(room.Id)

		newRound, err := rm.CreateNextRound(room.Id)

		assert.NoError(t, err)
		assert.Greater(t, newRound.GetInt("round_number"), currentRoundNum)
	})

	t.Run("completes previous round", func(t *testing.T) {
		oldRound, _ := rm.GetCurrentRoundRecord(room.Id)

		rm.CreateNextRound(room.Id)

		// Old round should be completed
		refreshed, _ := server.App.FindRecordById("rounds", oldRound.Id)
		assert.Equal(t, string(models.RoundStateCompleted), refreshed.GetString("state"))
	})

	t.Run("new round starts in voting state", func(t *testing.T) {
		newRound, _ := rm.CreateNextRound(room.Id)

		assert.Equal(t, string(models.RoundStateVoting), newRound.GetString("state"))
	})

	t.Run("new round has no votes", func(t *testing.T) {
		rm.CreateNextRound(room.Id)

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Empty(t, votes)
	})
}

func TestRoomManager_GetCurrentRound(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	t.Run("returns 1 for initial round", func(t *testing.T) {
		roundNum, err := rm.GetCurrentRound(room.Id)

		assert.NoError(t, err)
		assert.Equal(t, 1, roundNum)
	})

	t.Run("returns incremented number after next round", func(t *testing.T) {
		p, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		rm.CastVote(room.Id, p.Id, "5")
		rm.RevealVotes(room.Id)
		rm.CreateNextRound(room.Id)

		roundNum, _ := rm.GetCurrentRound(room.Id)
		assert.Equal(t, 2, roundNum)
	})
}

func TestRoomManager_IsRoomCreator(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	creator, _ := rm.AddParticipant(room.Id, "Creator", models.RoleVoter, "s1")
	other, _ := rm.AddParticipant(room.Id, "Other", models.RoleVoter, "s2")

	t.Run("returns true for room creator", func(t *testing.T) {
		isCreator := rm.IsRoomCreator(room.Id, creator.Id)

		assert.True(t, isCreator)
	})

	t.Run("returns false for non-creator", func(t *testing.T) {
		isCreator := rm.IsRoomCreator(room.Id, other.Id)

		assert.False(t, isCreator)
	})
}

func TestRoomManager_UpdateParticipantName(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	participant, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")

	t.Run("updates participant name successfully", func(t *testing.T) {
		err := rm.UpdateParticipantName(participant.Id, "Alice Smith")

		assert.NoError(t, err)

		updated, _ := rm.GetParticipant(participant.Id)
		assert.Equal(t, "Alice Smith", updated.GetString("name"))
	})

	t.Run("validates name", func(t *testing.T) {
		err := rm.UpdateParticipantName(participant.Id, "<script>alert('xss')</script>")

		assert.Error(t, err)
	})
}

func TestRoomManager_UpdateRoomName(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	t.Run("updates room name successfully", func(t *testing.T) {
		err := rm.UpdateRoomName(room.Id, "New Room Name")

		assert.NoError(t, err)

		updated, _ := rm.GetRoom(room.Id)
		assert.Equal(t, "New Room Name", updated.GetString("name"))
	})

	t.Run("validates name", func(t *testing.T) {
		err := rm.UpdateRoomName(room.Id, "<script>alert('xss')</script>")

		assert.Error(t, err)
	})
}
