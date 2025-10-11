package migrations

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create rounds collection
		rounds := core.NewBaseCollection("rounds")
		rounds.ListRule = nil
		rounds.ViewRule = nil
		rounds.CreateRule = nil
		rounds.UpdateRule = nil
		rounds.DeleteRule = nil

		// Get rooms collection for relation
		rooms, err := app.FindCollectionByNameOrId("rooms")
		if err != nil {
			return fmt.Errorf("failed to find rooms collection: %w", err)
		}

		// room_id relation
		rounds.Fields.Add(&core.RelationField{
			Name:          "room_id",
			Required:      true,
			MaxSelect:     1,
			CollectionId:  rooms.Id,
			CascadeDelete: true,
		})

		// round_number field
		rounds.Fields.Add(&core.NumberField{
			Name:     "round_number",
			Required: true,
		})

		// state field
		rounds.Fields.Add(&core.SelectField{
			Name:      "state",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"voting", "revealed", "completed"},
		})

		// average_score field (nullable - only set when completed)
		rounds.Fields.Add(&core.NumberField{
			Name:     "average_score",
			Required: false,
		})

		// total_votes field
		rounds.Fields.Add(&core.NumberField{
			Name:     "total_votes",
			Required: false,
		})

		// completed_at field (nullable - only set when completed)
		rounds.Fields.Add(&core.DateField{
			Name:     "completed_at",
			Required: false,
		})

		// Create indexes
		rounds.Indexes = []string{
			"CREATE INDEX idx_rounds_room ON rounds(room_id)",
			"CREATE INDEX idx_rounds_state ON rounds(state)",
			"CREATE UNIQUE INDEX idx_rounds_unique ON rounds(room_id, round_number)",
		}

		if err := app.Save(rounds); err != nil {
			return fmt.Errorf("failed to create rounds collection: %w", err)
		}

		// Get participants collection for creator relation
		participants, err := app.FindCollectionByNameOrId("participants")
		if err != nil {
			return fmt.Errorf("failed to find participants collection: %w", err)
		}

		// Update rooms collection - add creator_participant_id
		rooms.Fields.Add(&core.RelationField{
			Name:          "creator_participant_id",
			Required:      false,
			MaxSelect:     1,
			CollectionId:  participants.Id,
			CascadeDelete: false, // Don't delete room if creator participant is deleted
		})

		// Update rooms collection - add current_round_id
		rooms.Fields.Add(&core.RelationField{
			Name:          "current_round_id",
			Required:      false,
			MaxSelect:     1,
			CollectionId:  rounds.Id,
			CascadeDelete: false,
		})

		if err := app.Save(rooms); err != nil {
			return fmt.Errorf("failed to update rooms collection: %w", err)
		}

		// Migrate existing data
		log.Println("Migrating existing rooms to round system...")

		// Get all existing rooms
		existingRooms, err := app.FindRecordsByFilter("rooms", "", "", 1000, 0, nil)
		if err != nil {
			return fmt.Errorf("failed to get existing rooms: %w", err)
		}

		for _, room := range existingRooms {
			// Create Round 1 for this room
			round := core.NewRecord(rounds)
			round.Set("room_id", room.Id)
			round.Set("round_number", 1)

			// Set state based on current room state
			roomState := room.GetString("state")
			if roomState == "revealed" {
				round.Set("state", "revealed")
			} else {
				round.Set("state", "voting")
			}

			if err := app.Save(round); err != nil {
				return fmt.Errorf("failed to create round 1 for room %s: %w", room.Id, err)
			}

			// Set room's current_round_id
			room.Set("current_round_id", round.Id)

			// Find first participant (oldest joined_at) to set as creator
			roomParticipants, err := app.FindRecordsByFilter(
				"participants",
				"room_id = {:roomId}",
				"joined_at",
				1,
				0,
				map[string]any{"roomId": room.Id},
			)
			if err == nil && len(roomParticipants) > 0 {
				room.Set("creator_participant_id", roomParticipants[0].Id)
			}

			if err := app.Save(room); err != nil {
				return fmt.Errorf("failed to update room %s with round info: %w", room.Id, err)
			}

			log.Printf("✓ Migrated room %s (%s) - Round 1 created", room.Id, room.GetString("name"))
		}

		// Update votes collection - add round_id relation
		votes, err := app.FindCollectionByNameOrId("votes")
		if err != nil {
			return fmt.Errorf("failed to find votes collection: %w", err)
		}

		votes.Fields.Add(&core.RelationField{
			Name:          "round_id",
			Required:      false, // Initially false for migration
			MaxSelect:     1,
			CollectionId:  rounds.Id,
			CascadeDelete: true,
		})

		if err := app.Save(votes); err != nil {
			return fmt.Errorf("failed to update votes collection: %w", err)
		}

		// Migrate existing votes to link to rounds
		log.Println("Migrating existing votes to rounds...")

		existingVotes, err := app.FindRecordsByFilter("votes", "", "", 10000, 0, nil)
		if err != nil {
			return fmt.Errorf("failed to get existing votes: %w", err)
		}

		for _, vote := range existingVotes {
			roomID := vote.GetString("room_id")
			roundNumber := vote.GetInt("round_number")

			// Find the corresponding round
			roundRecords, err := app.FindRecordsByFilter(
				"rounds",
				"room_id = {:roomId} && round_number = {:roundNum}",
				"",
				1,
				0,
				map[string]any{
					"roomId":   roomID,
					"roundNum": roundNumber,
				},
			)

			if err == nil && len(roundRecords) > 0 {
				vote.Set("round_id", roundRecords[0].Id)
				if err := app.Save(vote); err != nil {
					log.Printf("Warning: failed to update vote %s: %v", vote.Id, err)
				}
			}
		}

		log.Println("✓ Migration completed successfully")
		return nil

	}, func(app core.App) error {
		// Down migration - cleanup in reverse order

		// Remove fields from rooms
		rooms, err := app.FindCollectionByNameOrId("rooms")
		if err == nil {
			// Remove creator_participant_id field
			for i, field := range rooms.Fields {
				if field.GetName() == "creator_participant_id" {
					rooms.Fields = append(rooms.Fields[:i], rooms.Fields[i+1:]...)
					break
				}
			}
			// Remove current_round_id field
			for i, field := range rooms.Fields {
				if field.GetName() == "current_round_id" {
					rooms.Fields = append(rooms.Fields[:i], rooms.Fields[i+1:]...)
					break
				}
			}
			_ = app.Save(rooms) // Best effort cleanup
		}

		// Remove round_id field from votes
		votes, err := app.FindCollectionByNameOrId("votes")
		if err == nil {
			for i, field := range votes.Fields {
				if field.GetName() == "round_id" {
					votes.Fields = append(votes.Fields[:i], votes.Fields[i+1:]...)
					break
				}
			}
			_ = app.Save(votes) // Best effort cleanup
		}

		// Delete rounds collection
		rounds, err := app.FindCollectionByNameOrId("rounds")
		if err == nil && rounds != nil {
			return app.Delete(rounds)
		}

		return nil
	})
}
