package handlers_test

import (
	"testing"

	"github.com/damione1/planning-poker/internal/handlers"
	"github.com/stretchr/testify/assert"
)

// Helper function to access calculateStats (it's unexported, so we test it indirectly via handlers)
// For now, we'll create a test wrapper to expose it
func TestCalculateStats_ConsensusDetection(t *testing.T) {
	t.Run("detects consensus with identical votes", func(t *testing.T) {
		votes := map[string]string{
			"alice": "5",
			"bob":   "5",
			"carol": "5",
		}

		stats := handlers.CalculateStatsForTest(votes)

		assert.NotNil(t, stats)
		assert.Equal(t, 3, stats["total"])
		assert.Equal(t, 100.0, stats["agreementPercentage"])
		assert.Equal(t, "5", stats["mostCommonValue"])
		assert.Equal(t, true, stats["consensus"])
	})

	t.Run("does not detect consensus with different votes", func(t *testing.T) {
		votes := map[string]string{
			"alice": "5",
			"bob":   "8",
			"carol": "5",
		}

		stats := handlers.CalculateStatsForTest(votes)

		assert.NotNil(t, stats)
		// 2 out of 3 voted 5 = 66.67%
		agreement := stats["agreementPercentage"].(float64)
		assert.InDelta(t, 66.67, agreement, 0.1)
		assert.Equal(t, false, stats["consensus"])
	})

	t.Run("detects consensus with special cards", func(t *testing.T) {
		votes := map[string]string{
			"alice": "?",
			"bob":   "?",
			"carol": "?",
		}

		stats := handlers.CalculateStatsForTest(votes)

		assert.NotNil(t, stats)
		assert.Equal(t, 100.0, stats["agreementPercentage"])
		assert.Equal(t, "?", stats["mostCommonValue"])
		assert.Equal(t, true, stats["consensus"])
	})

	t.Run("detects consensus with coffee break", func(t *testing.T) {
		votes := map[string]string{
			"alice": "☕",
			"bob":   "☕",
		}

		stats := handlers.CalculateStatsForTest(votes)

		assert.NotNil(t, stats)
		assert.Equal(t, 100.0, stats["agreementPercentage"])
		assert.Equal(t, true, stats["consensus"])
	})

	t.Run("handles single vote as consensus", func(t *testing.T) {
		votes := map[string]string{
			"alice": "13",
		}

		stats := handlers.CalculateStatsForTest(votes)

		assert.NotNil(t, stats)
		assert.Equal(t, 100.0, stats["agreementPercentage"])
		assert.Equal(t, true, stats["consensus"])
	})

	t.Run("handles empty votes", func(t *testing.T) {
		votes := map[string]string{}

		stats := handlers.CalculateStatsForTest(votes)

		assert.Nil(t, stats)
	})

	t.Run("calculates correct agreement percentage without consensus", func(t *testing.T) {
		votes := map[string]string{
			"alice": "5",
			"bob":   "5",
			"carol": "5",
			"dave":  "8",
			"eve":   "8",
		}

		stats := handlers.CalculateStatsForTest(votes)

		assert.NotNil(t, stats)
		// 3 out of 5 = 60%
		assert.Equal(t, 60.0, stats["agreementPercentage"])
		assert.Equal(t, "5", stats["mostCommonValue"])
		assert.Equal(t, false, stats["consensus"])
	})

	t.Run("detects consensus with float values", func(t *testing.T) {
		votes := map[string]string{
			"alice": "0.5",
			"bob":   "0.5",
			"carol": "0.5",
		}

		stats := handlers.CalculateStatsForTest(votes)

		assert.NotNil(t, stats)
		assert.Equal(t, 100.0, stats["agreementPercentage"])
		assert.Equal(t, "0.5", stats["mostCommonValue"])
		assert.Equal(t, true, stats["consensus"])
		assert.InDelta(t, 0.5, stats["average"].(float64), 0.01)
	})
}
