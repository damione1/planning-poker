package models

import (
	"sync"
	"time"
)

type RoomState string

const (
	StateVoting   RoomState = "voting"
	StateRevealed RoomState = "revealed"
)

type Room struct {
	ID              string
	Name            string
	PointingMethod  string // "fibonacci" or "custom"
	CustomValues    []string
	State           RoomState
	Participants    map[string]*Participant
	Votes           map[string]string
	CreatedAt       time.Time
	LastActivity    time.Time
	ExpiresAt       time.Time
	mu              sync.RWMutex
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

func (r *Room) AddParticipant(p *Participant) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Participants[p.ID] = p
	r.LastActivity = time.Now()
}

func (r *Room) GetParticipant(participantID string) *Participant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Participants[participantID]
}

func (r *Room) RemoveParticipant(participantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Participants, participantID)
	delete(r.Votes, participantID)
	r.LastActivity = time.Now()
}

func (r *Room) CastVote(participantID, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Votes[participantID] = value
	r.LastActivity = time.Now()
}

func (r *Room) RevealVotes() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.State = StateRevealed
	r.LastActivity = time.Now()
}

func (r *Room) ResetVoting() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.State = StateVoting
	r.Votes = make(map[string]string)
	r.LastActivity = time.Now()
}

func (r *Room) GetVoteStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"total": len(r.Votes),
	}
}

func (r *Room) GetLastActivity() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.LastActivity
}
