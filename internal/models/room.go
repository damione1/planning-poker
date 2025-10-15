package models

import (
	"time"
)

type RoomState string

const (
	StateVoting   RoomState = "voting"
	StateRevealed RoomState = "revealed"
)

// Room is a data transfer object for room state.
// All persistent state is managed in the database via RoomManager.
// This struct is used for rendering templates and passing data between handlers.
type Room struct {
	ID                         string
	Name                       string
	PointingMethod             string // "fibonacci" or "custom"
	CustomValues               []string
	Config                     *RoomConfig // Room configuration and permissions
	State                      RoomState   // Derived from CurrentRound.State
	CurrentRound               *Round      // Current round for state derivation
	Participants               map[string]*Participant
	Votes                      map[string]string // Current round votes for rendering
	ConsecutiveConsensusRounds int               // Number of consecutive rounds with 100% agreement
	CreatedAt                  time.Time
	LastActivity               time.Time
	ExpiresAt                  time.Time
}

func NewRoom(id, name, pointingMethod string, customValues []string) *Room {
	return &Room{
		ID:             id,
		Name:           name,
		PointingMethod: pointingMethod,
		CustomValues:   customValues,
		State:          StateVoting,
		Participants:   make(map[string]*Participant),
		Votes:          make(map[string]string),
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
	}
}

// GetState returns the current room state from the current round.
// If CurrentRound is populated, derives state from it; otherwise returns the State field.
func (r *Room) GetState() RoomState {
	// If CurrentRound is populated, derive state from it
	if r.CurrentRound != nil {
		return RoomState(r.CurrentRound.State)
	}

	// Fallback to State field for backward compatibility
	return r.State
}
