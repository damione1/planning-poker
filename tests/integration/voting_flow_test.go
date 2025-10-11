package integration_test

import (
	"testing"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func TestVotingFlow_BasicFlow(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	bob, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	t.Run("complete voting lifecycle", func(t *testing.T) {
		// 1. Initial state should be voting
		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)

		// 2. Cast votes
		err := rm.CastVote(room.Id, alice.Id, "5")
		assert.NoError(t, err)

		err = rm.CastVote(room.Id, bob.Id, "8")
		assert.NoError(t, err)

		// 3. Verify votes are recorded
		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 2)

		// 4. State should still be voting
		state, _ = rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)

		// 5. Reveal votes
		err = rm.RevealVotes(room.Id)
		assert.NoError(t, err)

		// 6. State should be revealed
		state, _ = rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateRevealed, state)

		// 7. All votes should still be present
		votes, _ = rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 2)
	})
}

func TestVotingFlow_VoteUpdates(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")

	t.Run("allows vote changes before reveal", func(t *testing.T) {
		// Cast initial vote
		rm.CastVote(room.Id, alice.Id, "5")

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 1)
		assert.Equal(t, "5", votes[0].GetString("value"))

		// Change vote
		err := rm.CastVote(room.Id, alice.Id, "8")
		assert.NoError(t, err)

		// Should still be 1 vote with new value
		votes, _ = rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 1)
		assert.Equal(t, "8", votes[0].GetString("value"))
	})

	t.Run("allows vote changes after reveal (no restrictions)", func(t *testing.T) {
		rm.CastVote(room.Id, alice.Id, "5")
		rm.RevealVotes(room.Id)

		// Should allow vote change even after reveal
		err := rm.CastVote(room.Id, alice.Id, "13")
		assert.NoError(t, err)

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Equal(t, "13", votes[0].GetString("value"))
	})
}

func TestVotingFlow_MultipleParticipants(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)

	participants := make([]string, 5)
	for i := 0; i < 5; i++ {
		p, _ := rm.AddParticipant(room.Id, "User"+string(rune('A'+i)), models.RoleVoter, "s"+string(rune('1'+i)))
		participants[i] = p.Id
	}

	t.Run("all participants can vote independently", func(t *testing.T) {
		// Each participant votes
		values := []string{"1", "2", "3", "5", "8"}
		for i, pid := range participants {
			err := rm.CastVote(room.Id, pid, values[i])
			assert.NoError(t, err)
		}

		// All votes should be recorded
		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 5)

		// Verify each vote value
		voteMap := make(map[string]string)
		for _, vote := range votes {
			voteMap[vote.GetString("participant_id")] = vote.GetString("value")
		}

		for i, pid := range participants {
			assert.Equal(t, values[i], voteMap[pid])
		}
	})
}

func TestVotingFlow_SpectatorRestrictions(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	spectator, _ := rm.AddParticipant(room.Id, "Spectator", models.RoleSpectator, "s1")

	t.Run("spectator can cast vote (validation at API level)", func(t *testing.T) {
		// Note: RoomManager doesn't enforce role restrictions
		// That's handled at the WebSocket handler level
		err := rm.CastVote(room.Id, spectator.Id, "5")
		assert.NoError(t, err)
	})
}

func TestVotingFlow_Reset(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	bob, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	t.Run("reset clears votes and returns to voting", func(t *testing.T) {
		// Vote and reveal
		rm.CastVote(room.Id, alice.Id, "5")
		rm.CastVote(room.Id, bob.Id, "8")
		rm.RevealVotes(room.Id)

		// Verify revealed state
		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateRevealed, state)

		// Reset
		err := rm.ResetRound(room.Id)
		assert.NoError(t, err)

		// Should be back to voting
		state, _ = rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)

		// Votes should be cleared
		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Empty(t, votes)
	})

	t.Run("can vote again after reset", func(t *testing.T) {
		rm.CastVote(room.Id, alice.Id, "5")
		rm.RevealVotes(room.Id)
		rm.ResetRound(room.Id)

		// Should be able to vote again
		err := rm.CastVote(room.Id, alice.Id, "13")
		assert.NoError(t, err)

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 1)
		assert.Equal(t, "13", votes[0].GetString("value"))
	})
}

func TestVotingFlow_NextRound(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
	bob, _ := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")

	t.Run("next round starts fresh voting session", func(t *testing.T) {
		// Complete round 1
		rm.CastVote(room.Id, alice.Id, "5")
		rm.CastVote(room.Id, bob.Id, "8")
		rm.RevealVotes(room.Id)

		roundNum, _ := rm.GetCurrentRound(room.Id)
		assert.Equal(t, 1, roundNum)

		// Create next round
		newRound, err := rm.CreateNextRound(room.Id)
		assert.NoError(t, err)
		assert.Equal(t, 2, newRound.GetInt("round_number"))

		// Should be in voting state
		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)

		// No votes in new round
		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Empty(t, votes)

		// Current round should be 2
		roundNum, _ = rm.GetCurrentRound(room.Id)
		assert.Equal(t, 2, roundNum)
	})

	t.Run("can vote in new round", func(t *testing.T) {
		rm.CastVote(room.Id, alice.Id, "5")
		rm.RevealVotes(room.Id)
		rm.CreateNextRound(room.Id)

		// Vote in round 2
		err := rm.CastVote(room.Id, alice.Id, "13")
		assert.NoError(t, err)

		votes, _ := rm.GetRoomVotes(room.Id)
		assert.Len(t, votes, 1)
		assert.Equal(t, "13", votes[0].GetString("value"))
	})

	t.Run("previous round votes are isolated", func(t *testing.T) {
		// Round 1: cast votes
		rm.CastVote(room.Id, alice.Id, "5")
		rm.CastVote(room.Id, bob.Id, "8")
		rm.RevealVotes(room.Id)

		round1, _ := rm.GetCurrentRoundRecord(room.Id)

		// Create round 2
		rm.CreateNextRound(room.Id)

		// Round 1 votes should still exist in database
		votes, _ := server.App.FindRecordsByFilter(
			"votes",
			"round_id = {:roundId}",
			"",
			100,
			0,
			map[string]any{"roundId": round1.Id},
		)
		assert.Len(t, votes, 2)

		// But GetRoomVotes should only return current round votes (none)
		currentVotes, _ := rm.GetRoomVotes(room.Id)
		assert.Empty(t, currentVotes)
	})
}

func TestVotingFlow_StateTransitions(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	alice, _ := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")

	t.Run("voting → revealed → voting (reset)", func(t *testing.T) {
		// Start: voting
		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)

		// Cast and reveal
		rm.CastVote(room.Id, alice.Id, "5")
		rm.RevealVotes(room.Id)

		state, _ = rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateRevealed, state)

		// Reset back to voting
		rm.ResetRound(room.Id)

		state, _ = rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)
	})

	t.Run("voting → revealed → voting (next round)", func(t *testing.T) {
		// Cast and reveal
		rm.CastVote(room.Id, alice.Id, "5")
		rm.RevealVotes(room.Id)

		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateRevealed, state)

		// Next round returns to voting
		rm.CreateNextRound(room.Id)

		state, _ = rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)
	})
}

func TestVotingFlow_EmptyVotingSession(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)
	room, _ := rm.CreateRoom("Test", "fibonacci", nil)
	rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")

	t.Run("can reveal with no votes", func(t *testing.T) {
		err := rm.RevealVotes(room.Id)

		assert.NoError(t, err)

		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateRevealed, state)
	})

	t.Run("can reset with no votes", func(t *testing.T) {
		rm.RevealVotes(room.Id)

		err := rm.ResetRound(room.Id)

		assert.NoError(t, err)

		state, _ := rm.GetRoomState(room.Id)
		assert.Equal(t, models.StateVoting, state)
	})

	t.Run("can create next round with no votes", func(t *testing.T) {
		_, err := rm.CreateNextRound(room.Id)

		assert.NoError(t, err)

		roundNum, _ := rm.GetCurrentRound(room.Id)
		assert.Equal(t, 2, roundNum)
	})
}
