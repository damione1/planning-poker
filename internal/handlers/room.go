package handlers

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	qrcode "github.com/skip2/go-qrcode"

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
	var isCreator bool
	if sessionCookie != "" {
		participantRecord, err := h.roomManager.GetParticipantBySession(roomID, sessionCookie)
		if err == nil {
			participant = recordToParticipant(participantRecord)
			// Check if this participant is the room creator
			isCreator = h.roomManager.IsRoomCreator(roomID, participant.ID)
		}
	}

	// Get all participants for the room
	participantRecords, _ := h.roomManager.GetRoomParticipants(roomID)
	for _, pr := range participantRecords {
		p := recordToParticipant(pr)
		room.Participants[p.ID] = p
	}

	// Get current votes (GetRoomVotes already filters by current round)
	voteRecords, _ := h.roomManager.GetRoomVotes(roomID)
	for _, vr := range voteRecords {
		room.Votes[vr.GetString("participant_id")] = vr.GetString("value")
	}

	component := templates.Room(room, participant, isCreator)
	return templates.Render(re.Response, re.Request, component)
}

// ParticipantGridFragment returns just the participant grid HTML for htmx updates
func (h *RoomHandlers) ParticipantGridFragment(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("id")

	// Get room from database
	roomRecord, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
	}

	// Convert DB record to model for template
	room := recordToRoom(roomRecord)

	// Get current participant from session cookie
	sessionCookie := getParticipantID(re.Request)
	var currentParticipant *models.Participant
	if sessionCookie != "" {
		participantRecord, err := h.roomManager.GetParticipantBySession(roomID, sessionCookie)
		if err == nil {
			currentParticipant = recordToParticipant(participantRecord)
		}
	}

	// Get all participants for the room
	participantRecords, _ := h.roomManager.GetRoomParticipants(roomID)
	for _, pr := range participantRecords {
		p := recordToParticipant(pr)
		room.Participants[p.ID] = p
	}

	// Get current votes (GetRoomVotes already filters by current round)
	voteRecords, _ := h.roomManager.GetRoomVotes(roomID)
	for _, vr := range voteRecords {
		room.Votes[vr.GetString("participant_id")] = vr.GetString("value")
	}

	// Return both participant grid and statistics fragments
	// Use OOB version for WebSocket-triggered updates (this endpoint is called by refreshParticipants())
	participantGrid := templates.ParticipantGridOOB(room.Participants, room.State, room.Votes, currentParticipant)

	// Calculate statistics if in revealed state
	var stats map[string]interface{}
	if room.State == models.StateRevealed {
		stats = calculateStats(room.Votes)
	}
	currentRound, _ := h.roomManager.GetCurrentRound(roomID)
	statistics := templates.Statistics(room.State, stats, currentRound)

	// Combine both components
	combined := templ.Join(participantGrid, statistics)
	return templates.Render(re.Response, re.Request, combined)
}

// calculateStats computes vote statistics
func calculateStats(votes map[string]string) map[string]interface{} {
	if len(votes) == 0 {
		return nil
	}

	stats := make(map[string]interface{})
	valueBreakdown := make(map[string]int)
	var sum, count int

	for _, vote := range votes {
		valueBreakdown[vote]++

		// Try to parse as number for average
		var val int
		if _, err := fmt.Sscanf(vote, "%d", &val); err == nil {
			sum += val
			count++
		}
	}

	stats["total"] = len(votes)
	stats["valueBreakdown"] = valueBreakdown

	if count > 0 {
		stats["average"] = float64(sum) / float64(count)
	}

	return stats
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

// QRCodeHandler generates a QR code for the room URL
func (h *RoomHandlers) QRCodeHandler(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("id")

	// Verify room exists
	_, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
	}

	// Build the full room URL
	scheme := "http"
	if re.Request.TLS != nil {
		scheme = "https"
	}
	roomURL := fmt.Sprintf("%s://%s/room/%s", scheme, re.Request.Host, roomID)

	// Generate QR code
	png, err := qrcode.Encode(roomURL, qrcode.Medium, 256)
	if err != nil {
		return re.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate QR code"})
	}

	// Set proper headers
	re.Response.Header().Set("Content-Type", "image/png")
	re.Response.Header().Set("Cache-Control", "public, max-age=3600")

	// Write the PNG data
	_, err = re.Response.Write(png)
	return err
}
