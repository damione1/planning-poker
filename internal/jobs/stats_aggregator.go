package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/damione1/planning-poker/internal/services"
	"github.com/pocketbase/pocketbase/core"
)

// StatsAggregator handles periodic statistics aggregation
type StatsAggregator struct {
	app          core.App
	statsService *services.StatsService
}

// NewStatsAggregator creates a new stats aggregator
func NewStatsAggregator(app core.App, statsService *services.StatsService) *StatsAggregator {
	return &StatsAggregator{
		app:          app,
		statsService: statsService,
	}
}

// RunDailyAggregation aggregates yesterday's events into daily statistics
func (sa *StatsAggregator) RunDailyAggregation() error {
	yesterday := time.Now().AddDate(0, 0, -1)
	dayStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	log.Printf("Running daily aggregation for %s", dayStart.Format("2006-01-02"))

	// Query events for the period
	events, err := sa.app.FindRecordsByFilter(
		"room_stats_events",
		"event_time >= {:start} && event_time < {:end}",
		"",
		100000, // Generous limit
		0,
		map[string]any{
			"start": dayStart,
			"end":   dayEnd,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to fetch events for daily aggregation: %w", err)
	}

	log.Printf("Found %d events to aggregate", len(events))

	// Aggregate the events
	stats := sa.aggregateEvents(events)

	// Save aggregated stats
	if err := sa.saveAggregatedStats("daily", dayStart, dayEnd, stats); err != nil {
		return fmt.Errorf("failed to save daily stats: %w", err)
	}

	log.Printf("Daily aggregation completed: %d rooms, %d votes, %d participants",
		stats.TotalRooms, stats.TotalVotes, stats.UniqueParticipants)

	return nil
}

// UpdateAllTimeStats recalculates all-time statistics from all events
func (sa *StatsAggregator) UpdateAllTimeStats() error {
	log.Println("Updating all-time statistics...")

	// Get the earliest event time
	firstEvent, err := sa.app.FindRecordsByFilter(
		"room_stats_events",
		"",
		"event_time",
		1,
		0,
		nil,
	)
	if err != nil || len(firstEvent) == 0 {
		log.Println("No events found for all-time stats")
		return nil
	}

	startTime := firstEvent[0].GetDateTime("event_time").Time()
	endTime := time.Now()

	// Fetch ALL events (or use daily aggregates for efficiency)
	events, err := sa.app.FindRecordsByFilter(
		"room_stats_events",
		"",
		"",
		1000000, // Very generous limit for all-time
		0,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch all events: %w", err)
	}

	log.Printf("Aggregating %d total events for all-time stats", len(events))

	// Aggregate all events
	stats := sa.aggregateEvents(events)

	// Save all-time stats
	if err := sa.saveAggregatedStats("all_time", startTime, endTime, stats); err != nil {
		return fmt.Errorf("failed to save all-time stats: %w", err)
	}

	log.Printf("All-time stats updated: %d rooms, %d votes, %d participants",
		stats.TotalRooms, stats.TotalVotes, stats.UniqueParticipants)

	return nil
}

// aggregateEvents processes events into statistics
func (sa *StatsAggregator) aggregateEvents(events []*core.Record) *AggregatedStats {
	uniqueRooms := make(map[string]bool)
	uniqueParticipants := make(map[string]bool)
	premiumRooms := make(map[string]bool)
	freeRooms := make(map[string]bool)
	voteCount := 0
	roundCount := 0
	consensusCount := 0

	for _, event := range events {
		eventType := event.GetString("event_type")
		roomID := event.GetString("room_id")
		isPremium := event.GetBool("is_premium")

		switch eventType {
		case "room_created":
			uniqueRooms[roomID] = true
			if isPremium {
				premiumRooms[roomID] = true
			} else {
				freeRooms[roomID] = true
			}

		case "vote_cast":
			voteCount++
			if participantID := event.GetString("participant_id"); participantID != "" {
				uniqueParticipants[participantID] = true
			}

		case "round_completed":
			roundCount++

		case "consensus_achieved":
			consensusCount++

		case "participant_joined":
			if participantID := event.GetString("participant_id"); participantID != "" {
				uniqueParticipants[participantID] = true
			}
		}
	}

	return &AggregatedStats{
		TotalRooms:         len(uniqueRooms),
		TotalVotes:         voteCount,
		UniqueParticipants: len(uniqueParticipants),
		TotalRounds:        roundCount,
		ConsensusRounds:    consensusCount,
		PremiumRooms:       len(premiumRooms),
		FreeRooms:          len(freeRooms),
	}
}

// saveAggregatedStats saves or updates aggregated statistics
func (sa *StatsAggregator) saveAggregatedStats(periodType string, start, end time.Time, stats *AggregatedStats) error {
	collection, err := sa.app.FindCollectionByNameOrId("room_statistics")
	if err != nil {
		return fmt.Errorf("failed to find room_statistics collection: %w", err)
	}

	// Check if record already exists for this period
	existing, err := sa.app.FindRecordsByFilter(
		"room_statistics",
		"period_type = {:type} && period_start = {:start}",
		"",
		1,
		0,
		map[string]any{
			"type":  periodType,
			"start": start,
		},
	)

	var record *core.Record
	if err == nil && len(existing) > 0 {
		// Update existing record
		record = existing[0]
	} else {
		// Create new record
		record = core.NewRecord(collection)
		record.Set("period_type", periodType)
		record.Set("period_start", start)
	}

	// Set/update all fields
	record.Set("period_end", end)
	record.Set("total_rooms", stats.TotalRooms)
	record.Set("total_votes", stats.TotalVotes)
	record.Set("unique_participants", stats.UniqueParticipants)
	record.Set("total_rounds", stats.TotalRounds)
	record.Set("consensus_rounds", stats.ConsensusRounds)
	record.Set("premium_rooms", stats.PremiumRooms)
	record.Set("free_rooms", stats.FreeRooms)

	// Store additional metrics as JSON if needed
	additionalMetrics := map[string]interface{}{
		"aggregated_at": time.Now(),
	}
	metricsJSON, _ := json.Marshal(additionalMetrics)
	record.Set("additional_metrics", metricsJSON)

	if err := sa.app.Save(record); err != nil {
		return fmt.Errorf("failed to save aggregated stats: %w", err)
	}

	return nil
}

// AggregatedStats represents aggregated statistics for a period
type AggregatedStats struct {
	TotalRooms         int
	TotalVotes         int
	UniqueParticipants int
	TotalRounds        int
	ConsensusRounds    int
	PremiumRooms       int
	FreeRooms          int
}
