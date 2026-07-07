package services_test

import (
	"testing"
	"time"

	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsService_IncAndFlush_AccumulatesIntoSingleDailyRow(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	t.Cleanup(server.Cleanup)

	stats := services.NewStatsService(server.App)

	stats.Inc("rooms_created", 2)
	stats.Inc("participants_joined", 3)
	stats.Inc("votes_cast", 5)

	require.NoError(t, stats.Flush())

	dayKey := time.Now().Format("2006-01-02")
	record, err := server.App.FindFirstRecordByFilter("stats_daily", "day = {:d}", map[string]any{"d": dayKey})
	require.NoError(t, err)

	assert.Equal(t, 2, record.GetInt("rooms_created"))
	assert.Equal(t, 3, record.GetInt("participants_joined"))
	assert.Equal(t, 5, record.GetInt("votes_cast"))

	// Second round of increments must accumulate onto the same row, not overwrite it.
	stats.Inc("rooms_created", 1)
	stats.Inc("votes_cast", 10)

	require.NoError(t, stats.Flush())

	records, err := server.App.FindRecordsByFilter("stats_daily", "day = {:d}", "", 10, 0, map[string]any{"d": dayKey})
	require.NoError(t, err)
	require.Len(t, records, 1, "expected exactly one row per day")

	updated := records[0]
	assert.Equal(t, 3, updated.GetInt("rooms_created"))
	assert.Equal(t, 3, updated.GetInt("participants_joined"))
	assert.Equal(t, 15, updated.GetInt("votes_cast"))
}

func TestStatsService_ObserveGauges_TracksPeaksNotLatest(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	t.Cleanup(server.Cleanup)

	stats := services.NewStatsService(server.App)

	stats.ObserveGauges(5, 3)
	stats.ObserveGauges(2, 1) // lower than the peak - must not overwrite it

	require.NoError(t, stats.Flush())

	dayKey := time.Now().Format("2006-01-02")
	record, err := server.App.FindFirstRecordByFilter("stats_daily", "day = {:d}", map[string]any{"d": dayKey})
	require.NoError(t, err)

	assert.Equal(t, 5, record.GetInt("peak_connections"))
	assert.Equal(t, 3, record.GetInt("peak_rooms"))
}
