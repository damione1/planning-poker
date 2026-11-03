package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
)

type usageEventName string

const (
	usageRoomCreated     usageEventName = "room_created"
	usageRoomJoined      usageEventName = "room_joined"
	usageRoundStarted    usageEventName = "round_started"
	usageRoundRevealed   usageEventName = "round_revealed"
	usageRoundReset      usageEventName = "round_reset"
	usageVoteCast        usageEventName = "vote_cast"
	usageSettingsChanged usageEventName = "room_settings_changed"
)

// Only these aggregate fields can cross the analytics boundary. Room IDs,
// names, session cookies, card values and estimates never enter this struct.
type usageParameters struct {
	ParticipantRole  string `json:"participant_role,omitempty"`
	MemberType       string `json:"member_type,omitempty"`
	DeckType         string `json:"deck_type"`
	AutoReveal       string `json:"auto_reveal"`
	PermissionMode   string `json:"permission_mode"`
	RoundNumber      int    `json:"round_number"`
	ParticipantCount int    `json:"participant_count"`
	ConnectedCount   int    `json:"connected_count"`
	VoterCount       int    `json:"voter_count"`
	SpectatorCount   int    `json:"spectator_count"`
	VoteCount        int    `json:"vote_count"`
	RevealMethod     string `json:"reveal_method,omitempty"`
	VoteAction       string `json:"vote_action,omitempty"`
	RoundSource      string `json:"round_source,omitempty"`
}
type usageEvent struct {
	Name       usageEventName  `json:"name"`
	Parameters usageParameters `json:"parameters"`
}

func usageSnapshot(rm *services.RoomManager, roomID, participantID string, name usageEventName) *usageEvent {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return nil
	}
	participants, err := rm.GetRoomParticipants(roomID)
	if err != nil {
		return nil
	}
	round, err := rm.GetCurrentRoundRecord(roomID)
	if err != nil {
		return nil
	}
	votes, err := rm.GetRoomVotes(roomID)
	if err != nil {
		return nil
	}
	params := usageParameters{DeckType: "custom", AutoReveal: "disabled", PermissionMode: "mixed",
		RoundNumber: round.GetInt("round_number"), ParticipantCount: len(participants), VoteCount: len(votes), MemberType: "visitor"}
	if room.GetString("pointing_method") == "fibonacci" {
		params.DeckType = "fibonacci"
	}
	var values []string
	if err := room.UnmarshalJSONField("custom_values", &values); err == nil {
		validator := services.NewVoteValidator()
		for _, template := range validator.GetAvailableTemplates() {
			preset, err := validator.ParseCustomValues(template.Values)
			if err == nil && slices.Equal(values, preset) {
				params.DeckType = template.ID
				break
			}
		}
	}
	config := models.DefaultRoomConfig()
	if err := room.UnmarshalJSONField("config", config); err == nil {
		p := config.Permissions
		if p.AutoReveal {
			params.AutoReveal = "enabled"
		}
		if p.AllowAllReveal && p.AllowAllReset && p.AllowAllNewRound {
			params.PermissionMode = "open"
		}
		if !p.AllowAllReveal && !p.AllowAllReset && !p.AllowAllNewRound {
			params.PermissionMode = "host_only"
		}
	}
	for _, participant := range participants {
		role := models.ParticipantRole(participant.GetString("role"))
		switch role {
		case models.RoleVoter:
			params.VoterCount++
		case models.RoleSpectator:
			params.SpectatorCount++
		}
		if participant.GetBool("connected") {
			params.ConnectedCount++
		}
		if participant.Id == participantID {
			switch role {
			case models.RoleVoter, models.RoleSpectator:
				params.ParticipantRole = string(role)
			}
			params.MemberType = "participant"
			if room.GetString("creator_participant_id") == participantID {
				params.MemberType = "host"
			}
		}
	}
	return &usageEvent{Name: name, Parameters: params}
}

func setUsageTrigger(w http.ResponseWriter, events ...*usageEvent) {
	filtered := make([]*usageEvent, 0, len(events))
	for _, event := range events {
		if event != nil {
			filtered = append(filtered, event)
		}
	}
	if len(filtered) == 0 {
		return
	}
	data, err := json.Marshal(map[string]any{"pp:usage": map[string]any{"events": filtered}})
	if err != nil {
		log.Print("Cannot encode aggregate usage receipt")
		return
	}
	w.Header().Set("HX-Trigger", string(data))
}

// A mutation receipt goes to exactly one socket, never to every participant.
// Automatic reveals can fall back to another connected participant if the last
// voter disconnected during the countdown. The server counters remain intact
// even when no browser is connected or Analytics is blocked.
func (h *WSHandler) sendUsage(roomID, participantID string, event *usageEvent, fallback bool) {
	if event == nil {
		return
	}
	client := h.hub.GetClient(roomID, participantID)
	if client == nil && fallback {
		participants, err := h.roomManager.GetRoomParticipants(roomID)
		if err != nil {
			return
		}
		for _, participant := range participants {
			if client = h.hub.GetClient(roomID, participant.Id); client != nil {
				break
			}
		}
	}
	if client != nil {
		h.hub.SendToClient(client, &models.WSMessage{Type: models.MsgTypeUsageEvent, Payload: event})
	}
}
