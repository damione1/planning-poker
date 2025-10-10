package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"

	"github.com/damiengoehrig/planning-poker/internal/models"
	"github.com/damiengoehrig/planning-poker/internal/services"
	"github.com/damiengoehrig/planning-poker/web/templates"
)

type RoomHandlers struct {
	roomManager *services.RoomManager
	hub         *services.Hub
}

func NewRoomHandlers(rm *services.RoomManager, hub *services.Hub) *RoomHandlers {
	return &RoomHandlers{
		roomManager: rm,
		hub:         hub,
	}
}

func (h *RoomHandlers) CreateRoom(re *core.RequestEvent) error {
	name := re.Request.FormValue("name")
	pointingMethod := re.Request.FormValue("pointingMethod")
	customValues := re.Request.Form["customValues"]

	// Validation
	if name == "" {
		return re.JSON(http.StatusBadRequest, map[string]string{
			"error": "Room name is required",
		})
	}

	// Create room
	roomID := uuid.New().String()
	h.roomManager.CreateRoom(roomID, name, pointingMethod, customValues)

	// Redirect to room
	return re.Redirect(http.StatusSeeOther, "/room/"+roomID)
}

func (h *RoomHandlers) RoomView(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("id")

	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.Redirect(http.StatusSeeOther, "/")
	}

	// Check for participant cookie
	participantID := getParticipantID(re.Request)
	var participant *models.Participant
	if participantID != "" {
		participant = room.GetParticipant(participantID)
	}

	component := templates.Room(room, participant)
	return templates.Render(re.Response, re.Request, component)
}

func (h *RoomHandlers) JoinRoom(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("id")
	name := re.Request.FormValue("name")
	role := re.Request.FormValue("role")

	// Validation
	if name == "" {
		return re.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}

	// Get or create room
	room, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.JSON(http.StatusNotFound, map[string]string{
			"error": "Room not found",
		})
	}

	// Create participant
	participantID := uuid.New().String()
	participant := &models.Participant{
		ID:   participantID,
		Name: name,
		Role: models.RoleVoter,
	}

	if role == "spectator" {
		participant.Role = models.RoleSpectator
	}

	// Add to room
	room.AddParticipant(participant)

	// Set cookie
	setParticipantID(re.Response, participantID)

	// Broadcast participant joined event
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type: models.MsgTypeParticipantJoined,
		Payload: map[string]interface{}{
			"participant": participant,
		},
	})

	// Return success - htmx will handle the UI update
	return re.NoContent(http.StatusOK)
}

// Session cookie helpers
const participantCookieName = "pp_participant_id"

func getParticipantID(r *http.Request) string {
	cookie, err := r.Cookie(participantCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setParticipantID(w http.ResponseWriter, participantID string) {
	cookie := &http.Cookie{
		Name:     participantCookieName,
		Value:    participantID,
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
