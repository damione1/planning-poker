package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("rooms")
		collection.ListRule = nil
		collection.ViewRule = nil
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil

		// name field
		collection.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
			Max:      100,
		})

		// pointing_method field
		collection.Fields.Add(&core.SelectField{
			Name:      "pointing_method",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"fibonacci", "custom"},
		})

		// custom_values field
		collection.Fields.Add(&core.JSONField{
			Name:     "custom_values",
			Required: false,
			MaxSize:  2048,
		})

		// state field
		collection.Fields.Add(&core.SelectField{
			Name:      "state",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"voting", "revealed"},
		})

		// config field
		collection.Fields.Add(&core.JSONField{
			Name:     "config",
			Required: false,
			MaxSize:  10240,
		})

		// is_premium field
		collection.Fields.Add(&core.BoolField{
			Name:     "is_premium",
			Required: false, // Bool fields should not be required - default to false/null
		})

		// expires_at field
		collection.Fields.Add(&core.DateField{
			Name:     "expires_at",
			Required: true,
		})

		// last_activity field
		collection.Fields.Add(&core.DateField{
			Name:     "last_activity",
			Required: true,
		})

		// Create indexes
		collection.Indexes = []string{
			"CREATE INDEX idx_rooms_expires ON rooms(expires_at)",
			"CREATE INDEX idx_rooms_activity ON rooms(last_activity)",
			"CREATE INDEX idx_rooms_premium ON rooms(is_premium)",
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		// Create participants collection
		participants := core.NewBaseCollection("participants")
		participants.ListRule = nil
		participants.ViewRule = nil
		participants.CreateRule = nil
		participants.UpdateRule = nil
		participants.DeleteRule = nil

		// room_id relation
		participants.Fields.Add(&core.RelationField{
			Name:          "room_id",
			Required:      true,
			MaxSelect:     1,
			CollectionId:  collection.Id,
			CascadeDelete: true,
		})

		// name field
		participants.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
			Max:      50,
		})

		// role field
		participants.Fields.Add(&core.SelectField{
			Name:      "role",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"voter", "spectator"},
		})

		// connected field
		participants.Fields.Add(&core.BoolField{
			Name:     "connected",
			Required: false, // Bool fields should not be required
		})

		// session_cookie field
		participants.Fields.Add(&core.TextField{
			Name:     "session_cookie",
			Required: true,
			Max:      255,
		})

		// joined_at field
		participants.Fields.Add(&core.DateField{
			Name:     "joined_at",
			Required: true,
		})

		// last_seen field
		participants.Fields.Add(&core.DateField{
			Name:     "last_seen",
			Required: true,
		})

		// Create indexes
		participants.Indexes = []string{
			"CREATE INDEX idx_participants_room ON participants(room_id)",
			"CREATE INDEX idx_participants_cookie ON participants(session_cookie)",
			"CREATE UNIQUE INDEX idx_participants_unique ON participants(session_cookie, room_id)",
		}

		if err := app.Save(participants); err != nil {
			return err
		}

		// Create votes collection
		votes := core.NewBaseCollection("votes")
		votes.ListRule = nil
		votes.ViewRule = nil
		votes.CreateRule = nil
		votes.UpdateRule = nil
		votes.DeleteRule = nil

		// room_id relation
		votes.Fields.Add(&core.RelationField{
			Name:          "room_id",
			Required:      true,
			MaxSelect:     1,
			CollectionId:  collection.Id,
			CascadeDelete: true,
		})

		// participant_id relation
		votes.Fields.Add(&core.RelationField{
			Name:          "participant_id",
			Required:      true,
			MaxSelect:     1,
			CollectionId:  participants.Id,
			CascadeDelete: true,
		})

		// value field
		votes.Fields.Add(&core.TextField{
			Name:     "value",
			Required: true,
			Max:      20,
		})

		// round_number field
		votes.Fields.Add(&core.NumberField{
			Name:     "round_number",
			Required: true,
		})

		// voted_at field
		votes.Fields.Add(&core.DateField{
			Name:     "voted_at",
			Required: true,
		})

		// Create indexes
		votes.Indexes = []string{
			"CREATE INDEX idx_votes_room_round ON votes(room_id, round_number)",
			"CREATE INDEX idx_votes_participant ON votes(participant_id)",
			"CREATE UNIQUE INDEX idx_votes_unique ON votes(participant_id, room_id, round_number)",
		}

		return app.Save(votes)
	}, func(app core.App) error {
		// Down migration - delete in reverse order
		votes, err := app.FindCollectionByNameOrId("votes")
		if err == nil && votes != nil {
			if err := app.Delete(votes); err != nil {
				return err
			}
		}

		participants, err := app.FindCollectionByNameOrId("participants")
		if err == nil && participants != nil {
			if err := app.Delete(participants); err != nil {
				return err
			}
		}

		rooms, err := app.FindCollectionByNameOrId("rooms")
		if err == nil && rooms != nil {
			return app.Delete(rooms)
		}

		return nil
	})
}
