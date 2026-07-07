package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("stats_daily")
		collection.ListRule = nil
		collection.ViewRule = nil
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		// day field - "YYYY-MM-DD", one row per calendar day
		collection.Fields.Add(&core.TextField{
			Name:     "day",
			Required: true,
			Max:      10,
		})

		// Counters accumulated in-memory and flushed on a 60s ticker
		// (see StatsService), plus running peak gauges for the day.
		collection.Fields.Add(&core.NumberField{Name: "rooms_created", Required: false})
		collection.Fields.Add(&core.NumberField{Name: "participants_joined", Required: false})
		collection.Fields.Add(&core.NumberField{Name: "votes_cast", Required: false})
		collection.Fields.Add(&core.NumberField{Name: "rounds_played", Required: false})
		collection.Fields.Add(&core.NumberField{Name: "rooms_expired", Required: false})
		collection.Fields.Add(&core.NumberField{Name: "peak_connections", Required: false})
		collection.Fields.Add(&core.NumberField{Name: "peak_rooms", Required: false})

		// No RelationField here by design - cleanupExpiredRooms cascade-deletes
		// only relation-linked records, so this collection is untouchable by
		// cleanup. Do not add a relation to rooms/participants/votes.
		collection.Indexes = []string{
			"CREATE UNIQUE INDEX idx_stats_daily_day ON stats_daily(day)",
		}

		return app.Save(collection)
	}, func(app core.App) error {
		stats, err := app.FindCollectionByNameOrId("stats_daily")
		if err == nil && stats != nil {
			return app.Delete(stats)
		}
		return nil
	})
}
