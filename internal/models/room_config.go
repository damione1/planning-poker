package models

// RoomConfig defines permissions and settings for a room
type RoomConfig struct {
	Permissions RoomPermissions `json:"permissions"`
}

// RoomPermissions defines who can perform specific actions
type RoomPermissions struct {
	// AllowAllNewRound: if true, any participant can trigger new round
	// if false, only room creator can trigger new round
	AllowAllNewRound bool `json:"allow_all_new_round"`

	// AllowAllReset: if true, any participant can reset the round
	// if false, only room creator can reset
	AllowAllReset bool `json:"allow_all_reset"`

	// AllowAllReveal: if true, any participant can reveal votes
	// if false, only room creator can reveal (existing behavior)
	AllowAllReveal bool `json:"allow_all_reveal"`

	// AllowChangeVoteAfterReveal: if true, voters can change their vote after reveal
	// if false, votes are locked once revealed
	AllowChangeVoteAfterReveal bool `json:"allow_change_vote_after_reveal"`

	// AutoReveal: if true, automatically reveal votes when all voters have voted
	// if false, manual reveal is required (default behavior)
	AutoReveal bool `json:"auto_reveal"`
}

// DefaultRoomConfig returns default configuration with permissive settings
func DefaultRoomConfig() *RoomConfig {
	return &RoomConfig{
		Permissions: RoomPermissions{
			AllowAllNewRound:           true,  // Default: everyone can trigger new round
			AllowAllReset:              true,  // Default: everyone can reset
			AllowAllReveal:             true,  // Default: everyone can reveal
			AllowChangeVoteAfterReveal: false, // Default: votes locked after reveal
			AutoReveal:                 false, // Default: manual reveal required
		},
	}
}
