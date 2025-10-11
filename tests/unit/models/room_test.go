package models_test

import (
	"testing"
	"time"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewRoom(t *testing.T) {
	t.Run("creates room with fibonacci pointing", func(t *testing.T) {
		room := models.NewRoom("room-1", "Sprint Planning", "fibonacci", nil)

		assert.Equal(t, "room-1", room.ID)
		assert.Equal(t, "Sprint Planning", room.Name)
		assert.Equal(t, "fibonacci", room.PointingMethod)
		assert.Equal(t, models.StateVoting, room.State)
		assert.NotNil(t, room.Participants)
		assert.NotNil(t, room.Votes)
		assert.WithinDuration(t, time.Now(), room.CreatedAt, time.Second)
		assert.WithinDuration(t, time.Now(), room.LastActivity, time.Second)
	})

	t.Run("creates room with custom values", func(t *testing.T) {
		custom := []string{"XS", "S", "M", "L", "XL"}
		room := models.NewRoom("room-2", "Design Review", "custom", custom)

		assert.Equal(t, "custom", room.PointingMethod)
		assert.Equal(t, custom, room.CustomValues)
	})

	t.Run("initializes empty collections", func(t *testing.T) {
		room := models.NewRoom("room-3", "Test", "fibonacci", nil)

		assert.Empty(t, room.Participants)
		assert.Empty(t, room.Votes)
		assert.Len(t, room.Participants, 0)
		assert.Len(t, room.Votes, 0)
	})
}

func TestRoom_GetState(t *testing.T) {
	t.Run("returns state from current round when available", func(t *testing.T) {
		room := models.NewRoom("room-1", "Test", "fibonacci", nil)
		room.CurrentRound = &models.Round{
			State: models.RoundStateRevealed,
		}

		assert.Equal(t, models.RoomState(models.RoundStateRevealed), room.GetState())
	})

	t.Run("falls back to state field when no current round", func(t *testing.T) {
		room := models.NewRoom("room-1", "Test", "fibonacci", nil)
		room.State = models.StateVoting

		assert.Equal(t, models.StateVoting, room.GetState())
	})

	t.Run("current round takes precedence over state field", func(t *testing.T) {
		room := models.NewRoom("room-1", "Test", "fibonacci", nil)
		room.State = models.StateVoting
		room.CurrentRound = &models.Round{
			State: models.RoundStateRevealed,
		}

		// Should return state from round, not room.State
		assert.Equal(t, models.RoomState(models.RoundStateRevealed), room.GetState())
		assert.NotEqual(t, room.State, room.GetState())
	})
}

func TestRoomState(t *testing.T) {
	t.Run("state constants", func(t *testing.T) {
		assert.Equal(t, models.RoomState("voting"), models.StateVoting)
		assert.Equal(t, models.RoomState("revealed"), models.StateRevealed)
	})
}
