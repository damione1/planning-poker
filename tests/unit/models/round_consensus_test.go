package models_test

import (
	"testing"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRound_ConsensusField(t *testing.T) {
	t.Run("new round has consensus set to false by default", func(t *testing.T) {
		round := models.NewRound("room123", 1)

		assert.NotNil(t, round)
		assert.Equal(t, "room123", round.RoomID)
		assert.Equal(t, 1, round.RoundNumber)
		assert.Equal(t, models.RoundStateVoting, round.State)
		assert.False(t, round.Consensus)
	})

	t.Run("can set consensus to true", func(t *testing.T) {
		round := models.NewRound("room123", 1)
		round.Consensus = true

		assert.True(t, round.Consensus)
	})

	t.Run("consensus persists with state changes", func(t *testing.T) {
		round := models.NewRound("room123", 1)
		round.Consensus = true
		round.State = models.RoundStateRevealed

		assert.True(t, round.Consensus)
		assert.Equal(t, models.RoundStateRevealed, round.State)
	})
}

func TestRoom_ConsecutiveConsensusRounds(t *testing.T) {
	t.Run("new room has zero consensus streak", func(t *testing.T) {
		room := models.NewRoom("id123", "Test Room", "custom", []string{"1", "2", "3"})

		assert.NotNil(t, room)
		assert.Equal(t, 0, room.ConsecutiveConsensusRounds)
	})

	t.Run("can increment consensus streak", func(t *testing.T) {
		room := models.NewRoom("id123", "Test Room", "custom", []string{"1", "2", "3"})
		room.ConsecutiveConsensusRounds = 3

		assert.Equal(t, 3, room.ConsecutiveConsensusRounds)
	})
}
