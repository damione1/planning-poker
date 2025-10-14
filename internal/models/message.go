package models

type WSMessage struct {
	Type    string      `json:"type"`
	RoomID  string      `json:"roomId,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// Client → Server message types
const (
	MsgTypeJoin           = "join"
	MsgTypeVote           = "vote"
	MsgTypeReveal         = "reveal"
	MsgTypeReset          = "reset"
	MsgTypeNextRound      = "next_round"
	MsgTypeUpdateName     = "update_name"
	MsgTypeUpdateRoomName = "update_room_name"
	MsgTypeUpdateConfig   = "update_config"
)

// Server → Client message types
const (
	MsgTypeRoomState         = "room_state" // Initial state sync on connection
	MsgTypeParticipantJoined = "participant_joined"
	MsgTypeParticipantLeft   = "participant_left"
	MsgTypeVoteCast          = "vote_cast"
	MsgTypeVotesRevealed     = "votes_revealed"
	MsgTypeVoteUpdated       = "vote_updated" // Vote changed after reveal
	MsgTypeRoomReset         = "room_reset"
	MsgTypeRoundCompleted    = "round_completed"
	MsgTypeNameUpdated       = "name_updated"
	MsgTypeRoomNameUpdated   = "room_name_updated"
	MsgTypeConfigUpdated     = "config_updated"
	MsgTypeError             = "error" // Error message to client
)
