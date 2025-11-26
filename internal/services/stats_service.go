package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// StatsService handles statistics event recording and querying
type StatsService struct {
	app core.App
}

// NewStatsService creates a new statistics service
func NewStatsService(app core.App) *StatsService {
	return &StatsService{
		app: app,
	}
}

// StatsSnapshot represents current aggregated statistics
type StatsSnapshot struct {
	TotalRooms         int       `json:"total_rooms"`
	TotalVotes         int       `json:"total_votes"`
	UniqueParticipants int       `json:"unique_participants"`
	TotalRounds        int       `json:"total_rounds"`
	ConsensusRounds    int       `json:"consensus_rounds"`
	PremiumRooms       int       `json:"premium_rooms"`
	FreeRooms          int       `json:"free_rooms"`
	LastUpdated        time.Time `json:"last_updated"`
}

// RecordEvent records a statistics event asynchronously (non-blocking)
func (s *StatsService) RecordEvent(eventType, roomID string, isPremium bool, payload map[string]interface{}) {
	go func() {
		if err := s.recordEventSync(eventType, roomID, isPremium, payload); err != nil {
			log.Printf("Failed to record stats event %s for room %s: %v", eventType, roomID, err)
		}
	}()
}

// recordEventSync performs the actual event recording (internal, synchronous)
func (s *StatsService) recordEventSync(eventType, roomID string, isPremium bool, payload map[string]interface{}) error {
	collection, err := s.app.FindCollectionByNameOrId("room_stats_events")
	if err != nil {
		return fmt.Errorf("failed to find room_stats_events collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("event_type", eventType)
	record.Set("room_id", roomID)
	record.Set("is_premium", isPremium)
	record.Set("event_time", time.Now())

	// Set optional fields from payload
	if payload != nil {
		if participantID, ok := payload["participant_id"].(string); ok {
			record.Set("participant_id", participantID)
		}
		if roundNumber, ok := payload["round_number"].(int); ok {
			record.Set("round_number", roundNumber)
		}

		// Store full payload as JSON
		payloadJSON, _ := json.Marshal(payload)
		record.Set("payload", payloadJSON)
	}

	if err := s.app.Save(record); err != nil {
		return fmt.Errorf("failed to save stats event: %w", err)
	}

	return nil
}

// GetCurrentStats retrieves the latest aggregated statistics
func (s *StatsService) GetCurrentStats() (*StatsSnapshot, error) {
	// Query for all_time period stats
	records, err := s.app.FindRecordsByFilter(
		"room_statistics",
		"period_type = 'all_time'",
		"-period_start",
		1,
		0,
		nil,
	)

	if err != nil || len(records) == 0 {
		// Fallback to live calculation if no aggregates exist
		return s.calculateLiveStats()
	}

	record := records[0]
	return &StatsSnapshot{
		TotalRooms:         record.GetInt("total_rooms"),
		TotalVotes:         record.GetInt("total_votes"),
		UniqueParticipants: record.GetInt("unique_participants"),
		TotalRounds:        record.GetInt("total_rounds"),
		ConsensusRounds:    record.GetInt("consensus_rounds"),
		PremiumRooms:       record.GetInt("premium_rooms"),
		FreeRooms:          record.GetInt("free_rooms"),
		LastUpdated:        record.GetDateTime("period_end").Time(),
	}, nil
}

// calculateLiveStats calculates statistics from raw events (fallback method)
func (s *StatsService) calculateLiveStats() (*StatsSnapshot, error) {
	events, err := s.app.FindRecordsByFilter(
		"room_stats_events",
		"",
		"",
		100000, // Generous limit for initial calculation
		0,
		nil,
	)
	if err != nil {
		return &StatsSnapshot{}, fmt.Errorf("failed to fetch events: %w", err)
	}

	stats := s.aggregateEvents(events)
	stats.LastUpdated = time.Now()

	return stats, nil
}

// aggregateEvents processes a list of events into aggregated statistics
func (s *StatsService) aggregateEvents(events []*core.Record) *StatsSnapshot {
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

	return &StatsSnapshot{
		TotalRooms:         len(uniqueRooms),
		TotalVotes:         voteCount,
		UniqueParticipants: len(uniqueParticipants),
		TotalRounds:        roundCount,
		ConsensusRounds:    consensusCount,
		PremiumRooms:       len(premiumRooms),
		FreeRooms:          len(freeRooms),
	}
}

// GetPeriodStats retrieves statistics for a specific period
func (s *StatsService) GetPeriodStats(periodType string, start time.Time) (*StatsSnapshot, error) {
	records, err := s.app.FindRecordsByFilter(
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

	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("no statistics found for period")
	}

	record := records[0]
	return &StatsSnapshot{
		TotalRooms:         record.GetInt("total_rooms"),
		TotalVotes:         record.GetInt("total_votes"),
		UniqueParticipants: record.GetInt("unique_participants"),
		TotalRounds:        record.GetInt("total_rounds"),
		ConsensusRounds:    record.GetInt("consensus_rounds"),
		PremiumRooms:       record.GetInt("premium_rooms"),
		FreeRooms:          record.GetInt("free_rooms"),
		LastUpdated:        record.GetDateTime("period_end").Time(),
	}, nil
}
