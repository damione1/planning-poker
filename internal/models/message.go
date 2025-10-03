package models

type WSMessage struct {
	Type    string      `json:"type"`
	RoomID  string      `json:"roomId,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// Client → Server message types
const (
	MsgTypeJoin   = "join"
	MsgTypeVote   = "vote"
	MsgTypeReveal = "reveal"
	MsgTypeReset  = "reset"
)

// Server → Client message types
const (
	MsgTypeParticipantJoined = "participant_joined"
	MsgTypeParticipantLeft   = "participant_left"
	MsgTypeVoteCast          = "vote_cast"
	MsgTypeVotesRevealed     = "votes_revealed"
	MsgTypeRoomReset         = "room_reset"
)
