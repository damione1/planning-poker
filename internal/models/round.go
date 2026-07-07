package models

import (
	"time"
)

type RoundState string

const (
	RoundStateVoting    RoundState = "voting"
	RoundStateRevealed  RoundState = "revealed"
	RoundStateCompleted RoundState = "completed"
)

type Round struct {
	ID           string
	RoomID       string
	RoundNumber  int
	State        RoundState
	AverageScore *float64 // Nullable - only set when completed
	TotalVotes   int
	Consensus    bool   // True if all votes were identical
	CreatedAt    time.Time
	CompletedAt  *time.Time // Nullable - only set when completed
}

func NewRound(roomID string, roundNumber int) *Round {
	return &Round{
		RoomID:      roomID,
		RoundNumber: roundNumber,
		State:       RoundStateVoting,
		TotalVotes:  0,
		CreatedAt:   time.Now(),
	}
}

