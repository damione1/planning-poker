package integration_test

import (
	"testing"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsensusStreak_IncrementOnConsensus(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)

	t.Run("increments streak when all votes are identical", func(t *testing.T) {
		// Create room and participants
		room, err := rm.CreateRoom("Consensus Test", "fibonacci", nil, nil)
		require.NoError(t, err)

		alice, err := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		require.NoError(t, err)
		bob, err := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
		require.NoError(t, err)
		carol, err := rm.AddParticipant(room.Id, "Carol", models.RoleVoter, "s3")
		require.NoError(t, err)

		// Round 1: All vote 5 (consensus)
		err = rm.CastVote(room.Id, alice.Id, "5")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "5")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, carol.Id, "5")
		require.NoError(t, err)

		// Reveal votes
		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		// Create next round (this completes the current round)
		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		// Check that streak incremented to 1
		roomRecord, err := rm.GetRoom(room.Id)
		require.NoError(t, err)
		assert.Equal(t, 1, roomRecord.GetInt("consecutive_consensus_rounds"))

		// Round 2: All vote 8 (consensus again)
		err = rm.CastVote(room.Id, alice.Id, "8")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "8")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, carol.Id, "8")
		require.NoError(t, err)

		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		// Check that streak incremented to 2
		roomRecord, err = rm.GetRoom(room.Id)
		require.NoError(t, err)
		assert.Equal(t, 2, roomRecord.GetInt("consecutive_consensus_rounds"))
	})
}

func TestConsensusStreak_ResetOnNonConsensus(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)

	t.Run("resets streak when votes differ", func(t *testing.T) {
		// Create room and participants
		room, err := rm.CreateRoom("Reset Test", "fibonacci", nil, nil)
		require.NoError(t, err)

		alice, err := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		require.NoError(t, err)
		bob, err := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
		require.NoError(t, err)

		// Round 1: Consensus
		err = rm.CastVote(room.Id, alice.Id, "5")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "5")
		require.NoError(t, err)

		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		// Verify streak is 1
		roomRecord, err := rm.GetRoom(room.Id)
		require.NoError(t, err)
		assert.Equal(t, 1, roomRecord.GetInt("consecutive_consensus_rounds"))

		// Round 2: Different votes (no consensus)
		err = rm.CastVote(room.Id, alice.Id, "5")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "8")
		require.NoError(t, err)

		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		// Verify streak reset to 0
		roomRecord, err = rm.GetRoom(room.Id)
		require.NoError(t, err)
		assert.Equal(t, 0, roomRecord.GetInt("consecutive_consensus_rounds"))
	})
}

func TestConsensusStreak_MultipleRoundsPattern(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)

	t.Run("tracks streak correctly across multiple rounds", func(t *testing.T) {
		room, err := rm.CreateRoom("Pattern Test", "fibonacci", nil, nil)
		require.NoError(t, err)

		alice, err := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		require.NoError(t, err)
		bob, err := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
		require.NoError(t, err)
		carol, err := rm.AddParticipant(room.Id, "Carol", models.RoleVoter, "s3")
		require.NoError(t, err)

		testCases := []struct {
			round           int
			votes           map[string]string
			expectedStreak  int
			description     string
		}{
			{
				round: 1,
				votes: map[string]string{
					alice.Id: "5",
					bob.Id:   "5",
					carol.Id: "5",
				},
				expectedStreak: 1,
				description:    "first consensus",
			},
			{
				round: 2,
				votes: map[string]string{
					alice.Id: "8",
					bob.Id:   "8",
					carol.Id: "8",
				},
				expectedStreak: 2,
				description:    "second consensus",
			},
			{
				round: 3,
				votes: map[string]string{
					alice.Id: "13",
					bob.Id:   "13",
					carol.Id: "13",
				},
				expectedStreak: 3,
				description:    "third consensus",
			},
			{
				round: 4,
				votes: map[string]string{
					alice.Id: "5",
					bob.Id:   "8",
					carol.Id: "13",
				},
				expectedStreak: 0,
				description:    "no consensus - reset",
			},
			{
				round: 5,
				votes: map[string]string{
					alice.Id: "3",
					bob.Id:   "3",
					carol.Id: "3",
				},
				expectedStreak: 1,
				description:    "consensus after reset",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				// Cast votes
				for participantID, value := range tc.votes {
					err := rm.CastVote(room.Id, participantID, value)
					require.NoError(t, err)
				}

				// Reveal and complete round
				err = rm.RevealVotes(room.Id)
				require.NoError(t, err)

				_, err = rm.CreateNextRound(room.Id)
				require.NoError(t, err)

				// Verify streak
				roomRecord, err := rm.GetRoom(room.Id)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedStreak, roomRecord.GetInt("consecutive_consensus_rounds"),
					"Round %d (%s): expected streak %d", tc.round, tc.description, tc.expectedStreak)
			})
		}
	})
}

func TestConsensusStreak_SpecialCards(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)

	t.Run("counts special cards as consensus", func(t *testing.T) {
		room, err := rm.CreateRoom("Special Cards Test", "fibonacci", nil, nil)
		require.NoError(t, err)

		alice, err := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		require.NoError(t, err)
		bob, err := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
		require.NoError(t, err)

		// Round 1: All vote "?" (consensus)
		err = rm.CastVote(room.Id, alice.Id, "?")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "?")
		require.NoError(t, err)

		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		roomRecord, err := rm.GetRoom(room.Id)
		require.NoError(t, err)
		assert.Equal(t, 1, roomRecord.GetInt("consecutive_consensus_rounds"))

		// Round 2: All vote "☕" (consensus)
		err = rm.CastVote(room.Id, alice.Id, "☕")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "☕")
		require.NoError(t, err)

		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		roomRecord, err = rm.GetRoom(room.Id)
		require.NoError(t, err)
		assert.Equal(t, 2, roomRecord.GetInt("consecutive_consensus_rounds"))
	})
}

func TestConsensusStreak_RoundRecordsConsensus(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	defer server.Cleanup()

	rm := services.NewRoomManager(server.App)

	t.Run("completed rounds record consensus status", func(t *testing.T) {
		room, err := rm.CreateRoom("Round Records Test", "fibonacci", nil, nil)
		require.NoError(t, err)

		alice, err := rm.AddParticipant(room.Id, "Alice", models.RoleVoter, "s1")
		require.NoError(t, err)
		bob, err := rm.AddParticipant(room.Id, "Bob", models.RoleVoter, "s2")
		require.NoError(t, err)

		// Round 1: Consensus
		err = rm.CastVote(room.Id, alice.Id, "5")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "5")
		require.NoError(t, err)

		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		round1, err := rm.GetCurrentRoundRecord(room.Id)
		require.NoError(t, err)

		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		// Verify round 1 marked as consensus
		completedRound, err := server.App.FindRecordById("rounds", round1.Id)
		require.NoError(t, err)
		assert.True(t, completedRound.GetBool("consensus"))
		assert.Equal(t, "completed", completedRound.GetString("state"))

		// Round 2: No consensus
		err = rm.CastVote(room.Id, alice.Id, "5")
		require.NoError(t, err)
		err = rm.CastVote(room.Id, bob.Id, "8")
		require.NoError(t, err)

		err = rm.RevealVotes(room.Id)
		require.NoError(t, err)

		round2, err := rm.GetCurrentRoundRecord(room.Id)
		require.NoError(t, err)

		_, err = rm.CreateNextRound(room.Id)
		require.NoError(t, err)

		// Verify round 2 marked as no consensus
		completedRound2, err := server.App.FindRecordById("rounds", round2.Id)
		require.NoError(t, err)
		assert.False(t, completedRound2.GetBool("consensus"))
	})
}
