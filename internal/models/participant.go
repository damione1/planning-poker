package models

import "time"

type ParticipantRole string

const (
	RoleVoter     ParticipantRole = "voter"
	RoleSpectator ParticipantRole = "spectator"
)

type Participant struct {
	ID        string
	Name      string
	Role      ParticipantRole
	Connected bool
	JoinedAt  time.Time
}

func NewParticipant(id, name string, role ParticipantRole) *Participant {
	return &Participant{
		ID:        id,
		Name:      name,
		Role:      role,
		Connected: false,
		JoinedAt:  time.Now(),
	}
}
