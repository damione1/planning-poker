package models_test

import (
	"testing"
	"time"

	"github.com/damione1/planning-poker-new/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewParticipant(t *testing.T) {
	t.Run("creates voter participant", func(t *testing.T) {
		p := models.NewParticipant("p-1", "Alice", models.RoleVoter)

		assert.Equal(t, "p-1", p.ID)
		assert.Equal(t, "Alice", p.Name)
		assert.Equal(t, models.RoleVoter, p.Role)
		assert.False(t, p.Connected) // Default disconnected
		assert.WithinDuration(t, time.Now(), p.JoinedAt, time.Second)
	})

	t.Run("creates spectator participant", func(t *testing.T) {
		p := models.NewParticipant("p-2", "Bob", models.RoleSpectator)

		assert.Equal(t, models.RoleSpectator, p.Role)
		assert.Equal(t, "Bob", p.Name)
	})

	t.Run("creates with different IDs", func(t *testing.T) {
		p1 := models.NewParticipant("p-1", "Alice", models.RoleVoter)
		p2 := models.NewParticipant("p-2", "Bob", models.RoleVoter)

		assert.NotEqual(t, p1.ID, p2.ID)
	})
}

func TestParticipantRole(t *testing.T) {
	t.Run("role constants", func(t *testing.T) {
		assert.Equal(t, models.ParticipantRole("voter"), models.RoleVoter)
		assert.Equal(t, models.ParticipantRole("spectator"), models.RoleSpectator)
	})
}
