package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create room_stats_events collection (append-only event log)
		events := core.NewBaseCollection("room_stats_events")
		events.ListRule = nil
		events.ViewRule = nil
		events.CreateRule = nil
		events.UpdateRule = nil
		events.DeleteRule = nil

		// Event type
		events.Fields.Add(&core.SelectField{
			Name:      "event_type",
			Required:  true,
			MaxSelect: 1,
			Values: []string{
				"room_created",
				"room_completed",
				"vote_cast",
				"participant_joined",
				"round_completed",
				"consensus_achieved",
			},
		})

		// Room context
		events.Fields.Add(&core.TextField{
			Name:     "room_id",
			Required: true,
			Max:      15, // PocketBase ID length
		})

		events.Fields.Add(&core.BoolField{
			Name:     "is_premium",
			Required: false,
		})

		// Optional participant context
		events.Fields.Add(&core.TextField{
			Name:     "participant_id",
			Required: false,
			Max:      15,
		})

		// Optional round context
		events.Fields.Add(&core.NumberField{
			Name:     "round_number",
			Required: false,
		})

		// Flexible event payload
		events.Fields.Add(&core.JSONField{
			Name:     "payload",
			Required: false,
			MaxSize:  2048,
		})

		// Timestamp
		events.Fields.Add(&core.DateField{
			Name:     "event_time",
			Required: true,
		})

		// Indexes for efficient queries
		events.Indexes = []string{
			"CREATE INDEX idx_stats_events_type ON room_stats_events(event_type)",
			"CREATE INDEX idx_stats_events_time ON room_stats_events(event_time)",
			"CREATE INDEX idx_stats_events_room ON room_stats_events(room_id)",
			"CREATE INDEX idx_stats_events_premium ON room_stats_events(is_premium)",
		}

		if err := app.Save(events); err != nil {
			return fmt.Errorf("failed to create room_stats_events collection: %w", err)
		}

		// Create room_statistics collection (pre-computed aggregates)
		stats := core.NewBaseCollection("room_statistics")
		stats.ListRule = nil
		stats.ViewRule = nil
		stats.CreateRule = nil
		stats.UpdateRule = nil
		stats.DeleteRule = nil

		// Aggregation period
		stats.Fields.Add(&core.SelectField{
			Name:      "period_type",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"daily", "weekly", "monthly", "all_time"},
		})

		stats.Fields.Add(&core.DateField{
			Name:     "period_start",
			Required: true,
		})

		stats.Fields.Add(&core.DateField{
			Name:     "period_end",
			Required: true,
		})

		// Core metrics
		stats.Fields.Add(&core.NumberField{
			Name:     "total_rooms",
			Required: true,
		})

		stats.Fields.Add(&core.NumberField{
			Name:     "total_votes",
			Required: true,
		})

		stats.Fields.Add(&core.NumberField{
			Name:     "unique_participants",
			Required: true,
		})

		stats.Fields.Add(&core.NumberField{
			Name:     "total_rounds",
			Required: false,
		})

		stats.Fields.Add(&core.NumberField{
			Name:     "consensus_rounds",
			Required: false,
		})

		// Premium vs Free breakdown
		stats.Fields.Add(&core.NumberField{
			Name:     "premium_rooms",
			Required: false,
		})

		stats.Fields.Add(&core.NumberField{
			Name:     "free_rooms",
			Required: false,
		})

		// Additional metrics (JSON for flexibility)
		stats.Fields.Add(&core.JSONField{
			Name:     "additional_metrics",
			Required: false,
			MaxSize:  4096,
		})

		// Indexes
		stats.Indexes = []string{
			"CREATE UNIQUE INDEX idx_stats_period ON room_statistics(period_type, period_start)",
			"CREATE INDEX idx_stats_time ON room_statistics(period_start)",
		}

		if err := app.Save(stats); err != nil {
			return fmt.Errorf("failed to create room_statistics collection: %w", err)
		}

		return nil

	}, func(app core.App) error {
		// Down migration - delete collections in reverse order
		stats, err := app.FindCollectionByNameOrId("room_statistics")
		if err == nil && stats != nil {
			if err := app.Delete(stats); err != nil {
				return err
			}
		}

		events, err := app.FindCollectionByNameOrId("room_stats_events")
		if err == nil && events != nil {
			return app.Delete(events)
		}

		return nil
	})
}
