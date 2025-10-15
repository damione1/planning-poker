package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Get rooms collection
		rooms, err := app.FindCollectionByNameOrId("rooms")
		if err != nil {
			return fmt.Errorf("failed to find rooms collection: %w", err)
		}

		// Add consecutive_consensus_rounds field
		rooms.Fields.Add(&core.NumberField{
			Name:     "consecutive_consensus_rounds",
			Required: false,
			Min:      nil,
			Max:      nil,
		})

		if err := app.Save(rooms); err != nil {
			return fmt.Errorf("failed to update rooms collection: %w", err)
		}

		// Get rounds collection
		rounds, err := app.FindCollectionByNameOrId("rounds")
		if err != nil {
			return fmt.Errorf("failed to find rounds collection: %w", err)
		}

		// Add consensus field
		rounds.Fields.Add(&core.BoolField{
			Name:     "consensus",
			Required: false,
		})

		if err := app.Save(rounds); err != nil {
			return fmt.Errorf("failed to update rounds collection: %w", err)
		}

		return nil

	}, func(app core.App) error {
		// Down migration - remove fields
		rooms, err := app.FindCollectionByNameOrId("rooms")
		if err == nil {
			for i, field := range rooms.Fields {
				if field.GetName() == "consecutive_consensus_rounds" {
					rooms.Fields = append(rooms.Fields[:i], rooms.Fields[i+1:]...)
					break
				}
			}
			_ = app.Save(rooms)
		}

		rounds, err := app.FindCollectionByNameOrId("rounds")
		if err == nil {
			for i, field := range rounds.Fields {
				if field.GetName() == "consensus" {
					rounds.Fields = append(rounds.Fields[:i], rounds.Fields[i+1:]...)
					break
				}
			}
			_ = app.Save(rounds)
		}

		return nil
	})
}
