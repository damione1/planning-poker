package migrations

import (
	"log"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		log.Println("Removing rooms.state field - state now managed by rounds only")

		// Get rooms collection
		rooms, err := app.FindCollectionByNameOrId("rooms")
		if err != nil {
			return err
		}

		// Remove state field
		for i, field := range rooms.Fields {
			if field.GetName() == "state" {
				rooms.Fields = append(rooms.Fields[:i], rooms.Fields[i+1:]...)
				log.Println("✓ Removed rooms.state field")
				break
			}
		}

		if err := app.Save(rooms); err != nil {
			return err
		}

		log.Println("✓ Migration completed: Room state now derived from current round")
		return nil

	}, func(app core.App) error {
		// Down migration: Add state field back
		log.Println("Rolling back: Restoring rooms.state field")

		rooms, err := app.FindCollectionByNameOrId("rooms")
		if err != nil {
			return err
		}

		// Add state field back
		rooms.Fields.Add(&core.SelectField{
			Name:      "state",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"voting", "revealed"},
		})

		if err := app.Save(rooms); err != nil {
			return err
		}

		// Populate state from current rounds for existing rooms
		existingRooms, err := app.FindRecordsByFilter("rooms", "", "", 1000, 0, nil)
		if err != nil {
			return err
		}

		for _, room := range existingRooms {
			currentRoundID := room.GetString("current_round_id")
			if currentRoundID != "" {
				round, err := app.FindRecordById("rounds", currentRoundID)
				if err == nil {
					roundState := round.GetString("state")
					// Map round state to room state
					roomState := "voting"
					if roundState == "revealed" {
						roomState = "revealed"
					}
					room.Set("state", roomState)
					_ = app.Save(room) // Best effort rollback
				}
			} else {
				// Default to voting if no current round
				room.Set("state", "voting")
				_ = app.Save(room) // Best effort rollback
			}
		}

		log.Println("✓ Rollback completed: rooms.state field restored")
		return nil
	})
}
