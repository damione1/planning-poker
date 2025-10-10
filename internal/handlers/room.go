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

	// Create room in database
	roomRecord, err := h.roomManager.CreateRoom(name, pointingMethod, customValues)
	if err != nil {
		return re.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create room",
		})
	}

	// Redirect to room
	return re.Redirect(http.StatusSeeOther, "/room/"+roomRecord.Id)
}

func (h *RoomHandlers) RoomView(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("id")

	// Get room from database
	roomRecord, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.Redirect(http.StatusSeeOther, "/")
	}

	// Convert DB record to model for template
	room := recordToRoom(roomRecord)

	// Check for participant cookie and load from DB
	sessionCookie := getParticipantID(re.Request)
	var participant *models.Participant
	if sessionCookie != "" {
		participantRecord, err := h.roomManager.GetParticipantBySession(roomID, sessionCookie)
		if err == nil {
			participant = recordToParticipant(participantRecord)
		}
	}

	// Get all participants for the room
	participantRecords, _ := h.roomManager.GetRoomParticipants(roomID)
	for _, pr := range participantRecords {
		p := recordToParticipant(pr)
		room.Participants[p.ID] = p
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

	// Verify room exists
	_, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.JSON(http.StatusNotFound, map[string]string{
			"error": "Room not found",
		})
	}

	// Determine role
	participantRole := models.RoleVoter
	if role == "spectator" {
		participantRole = models.RoleSpectator
	}

	// Create session cookie
	sessionCookie := uuid.New().String()

	// Create participant in database
	participantRecord, err := h.roomManager.AddParticipant(roomID, name, participantRole, sessionCookie)
	if err != nil {
		return re.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to join room",
		})
	}

	// Set cookie
	setParticipantID(re.Response, sessionCookie)

	// Convert to model for broadcast
	participant := recordToParticipant(participantRecord)

	// Broadcast participant joined event
	h.hub.BroadcastToRoom(roomID, &models.WSMessage{
		Type: models.MsgTypeParticipantJoined,
		Payload: map[string]interface{}{
			"participant": participant,
		},
	})

	// Redirect back to room page - will reload with participant context
	re.Response.Header().Set("HX-Redirect", "/room/"+roomID)
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

// Database record converters
func recordToRoom(record *core.Record) *models.Room {
	room := &models.Room{
		ID:             record.Id,
		Name:           record.GetString("name"),
		PointingMethod: record.GetString("pointing_method"),
		State:          models.RoomState(record.GetString("state")),
		Participants:   make(map[string]*models.Participant),
		Votes:          make(map[string]string),
		CreatedAt:      record.GetDateTime("created").Time(),
		LastActivity:   record.GetDateTime("last_activity").Time(),
	}

	// Parse custom values if present
	if customValuesJSON := record.GetString("custom_values"); customValuesJSON != "" {
		var customValues []string
		if err := record.UnmarshalJSONField("custom_values", &customValues); err == nil {
			room.CustomValues = customValues
		}
	}

	return room
}

func recordToParticipant(record *core.Record) *models.Participant {
	return &models.Participant{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Role:      models.ParticipantRole(record.GetString("role")),
		Connected: record.GetBool("connected"),
		JoinedAt:  record.GetDateTime("joined_at").Time(),
	}
}
