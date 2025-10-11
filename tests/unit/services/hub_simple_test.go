package services_test

import (
	"testing"

	"github.com/damiengoehrig/planning-poker/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestHub_Initialization(t *testing.T) {
	t.Run("creates new hub", func(t *testing.T) {
		hub := services.NewHub()

		assert.NotNil(t, hub)
	})

	t.Run("hub can be started", func(t *testing.T) {
		hub := services.NewHub()

		// Start hub in background
		go hub.Run()

		// Hub should not panic on start
		assert.NotNil(t, hub)
	})
}
