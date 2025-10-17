package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/security"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/web/templates"
)

type RoomHandlers struct {
	roomManager   *services.RoomManager
	hub           *services.Hub
	voteValidator *services.VoteValidator
}

func NewRoomHandlers(rm *services.RoomManager, hub *services.Hub) *RoomHandlers {
	return &RoomHandlers{
		roomManager:   rm,
		hub:           hub,
		voteValidator: services.NewVoteValidator(),
	}
}

func (h *RoomHandlers) CreateRoom(re *core.RequestEvent) error {
	name := re.Request.FormValue("name")
	pointingMethod := re.Request.FormValue("pointingMethod")
	customValuesRaw := re.Request.FormValue("customValues")

	// Validate and sanitize room name
	sanitizedName, err := security.ValidateRoomName(name)
	if err != nil {
		// Return HTML error for htmx or regular form submission
		component := templates.ErrorDisplay(err.Error())
		re.Response.WriteHeader(http.StatusBadRequest)
		return templates.Render(re.Response, re.Request, component)
	}
	name = sanitizedName

	// Default to custom pointing method
	if pointingMethod == "" {
		pointingMethod = "custom"
	}

	// Parse and validate custom values
	var customValues []string
	switch pointingMethod {
	case "fibonacci":
		// Use predefined fibonacci values
		customValues = h.voteValidator.GetFibonacciValues()
	case "custom":
		if customValuesRaw == "" {
			component := templates.ErrorDisplay("Custom values are required when using custom pointing method")
			re.Response.WriteHeader(http.StatusBadRequest)
			return templates.Render(re.Response, re.Request, component)
		}

		parsedValues, err := h.voteValidator.ParseCustomValues(customValuesRaw)
		if err != nil {
			component := templates.ErrorDisplay(fmt.Sprintf("Invalid custom values: %s", err.Error()))
			re.Response.WriteHeader(http.StatusBadRequest)
			return templates.Render(re.Response, re.Request, component)
		}
		customValues = parsedValues
	}

	// Parse room config from form values (defaults are all false)
	config := models.DefaultRoomConfig()
	config.Permissions.AllowAllReveal = re.Request.FormValue("allow_all_reveal") == "on"
	config.Permissions.AllowAllReset = re.Request.FormValue("allow_all_reset") == "on"
	config.Permissions.AllowAllNewRound = re.Request.FormValue("allow_all_new_round") == "on"
	config.Permissions.AllowChangeVoteAfterReveal = re.Request.FormValue("allow_change_vote_after_reveal") == "on"
	config.Permissions.AutoReveal = re.Request.FormValue("auto_reveal") == "on"

	// Create room in database with config
	roomRecord, err := h.roomManager.CreateRoom(name, pointingMethod, customValues, config)
	if err != nil {
		component := templates.ErrorDisplay("Failed to create room. Please try again.")
		re.Response.WriteHeader(http.StatusInternalServerError)
		return templates.Render(re.Response, re.Request, component)
	}

	// Redirect to room
	return re.Redirect(http.StatusSeeOther, "/room/"+roomRecord.Id)
}

func (h *RoomHandlers) RoomView(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("id")

	// Validate room ID
	if err := security.ValidateUUID(roomID); err != nil {
		return re.Redirect(http.StatusSeeOther, "/?error=invalid_room_id")
	}

	// Get room from database
	roomRecord, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.Redirect(http.StatusSeeOther, "/?error=room_not_found")
	}

	// Convert DB record to model for template
	room := recordToRoom(roomRecord)

	// Populate current round to derive state
	_ = h.populateCurrentRound(room) // Error is non-critical, room defaults to voting state

	// Check if room is expired
	if room.ExpiresAt.Before(time.Now()) {
		return re.Redirect(http.StatusSeeOther, "/?error=room_expired")
	}

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

	// Validate room ID
	if err := security.ValidateUUID(roomID); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid room ID"})
	}

	// Get room from database
	roomRecord, err := h.roomManager.GetRoom(roomID)
	if err != nil {
		return re.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
	}

	// Convert DB record to model for template
	room := recordToRoom(roomRecord)

	// Populate current round to derive state
	_ = h.populateCurrentRound(room) // Error is non-critical, room defaults to voting state

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
	statistics := templates.Statistics(room.State, stats, currentRound, room.ConsecutiveConsensusRounds)

	// Combine both components
	combined := templ.Join(participantGrid, statistics)
	return templates.Render(re.Response, re.Request, combined)
}

// calculateStats computes vote statistics (supports float values)
func calculateStats(votes map[string]string) map[string]interface{} {
	if len(votes) == 0 {
		return nil
	}

	stats := make(map[string]interface{})
	valueBreakdown := make(map[string]int)
	var sum float64
	var count int

	// Use validator for consistent numeric parsing
	validator := services.NewVoteValidator()

	// Track most common value for agreement percentage
	var mostCommonValue string
	var mostCommonCount int

	for _, vote := range votes {
		valueBreakdown[vote]++

		// Track most common value
		if valueBreakdown[vote] > mostCommonCount {
			mostCommonCount = valueBreakdown[vote]
			mostCommonValue = vote
		}

		// Try to parse as number for average (supports floats)
		if num, ok := validator.ParseNumericValue(vote); ok {
			sum += num
			count++
		}
	}

	stats["total"] = len(votes)
	stats["valueBreakdown"] = valueBreakdown

	// Calculate agreement percentage
	if len(votes) > 0 && mostCommonCount > 0 {
		agreementPercentage := (float64(mostCommonCount) / float64(len(votes))) * 100
		stats["agreementPercentage"] = agreementPercentage
		stats["mostCommonValue"] = mostCommonValue

		// Detect consensus (100% agreement)
		stats["consensus"] = agreementPercentage == 100.0
	}

	if count > 0 {
		stats["average"] = sum / float64(count)
	}

	return stats
}

// CalculateStatsForTest is a test helper that exposes calculateStats for testing
func CalculateStatsForTest(votes map[string]string) map[string]interface{} {
	return calculateStats(votes)
}

func (h *RoomHandlers) JoinRoom(re *core.RequestEvent) error {
	roomID := re.Request.PathValue("id")
	name := re.Request.FormValue("name")
	role := re.Request.FormValue("role")

	// Validate room ID
	if err := security.ValidateUUID(roomID); err != nil {
		component := templates.ErrorDisplay("Invalid room ID")
		re.Response.WriteHeader(http.StatusBadRequest)
		return templates.Render(re.Response, re.Request, component)
	}

	// Validate and sanitize participant name
	sanitizedName, err := security.ValidateParticipantName(name)
	if err != nil {
		component := templates.ErrorDisplay(err.Error())
		re.Response.WriteHeader(http.StatusBadRequest)
		return templates.Render(re.Response, re.Request, component)
	}
	name = sanitizedName

	// Verify room exists
	_, err = h.roomManager.GetRoom(roomID)
	if err != nil {
		component := templates.ErrorDisplay("Room not found")
		re.Response.WriteHeader(http.StatusNotFound)
		return templates.Render(re.Response, re.Request, component)
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
		component := templates.ErrorDisplay("Failed to join room. Please try again.")
		re.Response.WriteHeader(http.StatusInternalServerError)
		return templates.Render(re.Response, re.Request, component)
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
		Secure:   !isDevMode(), // Only send over HTTPS in production
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// isDevMode checks if the application is running in development mode
// Returns true if DEV_MODE environment variable is set to "true"
func isDevMode() bool {
	devMode := strings.ToLower(strings.TrimSpace(os.Getenv("DEV_MODE")))
	return devMode == "true" || devMode == "1"
}

// Database record converters
func recordToRoom(record *core.Record) *models.Room {
	room := &models.Room{
		ID:                         record.Id,
		Name:                       record.GetString("name"),
		PointingMethod:             record.GetString("pointing_method"),
		ConsecutiveConsensusRounds: record.GetInt("consecutive_consensus_rounds"),
		// State will be derived from CurrentRound after it's populated
		Participants: make(map[string]*models.Participant),
		Votes:        make(map[string]string),
		CreatedAt:    record.GetDateTime("created").Time(),
		LastActivity: record.GetDateTime("last_activity").Time(),
		ExpiresAt:    record.GetDateTime("expires_at").Time(),
	}

	// Parse custom values if present
	if customValuesJSON := record.GetString("custom_values"); customValuesJSON != "" {
		var customValues []string
		if err := record.UnmarshalJSONField("custom_values", &customValues); err == nil {
			room.CustomValues = customValues
		}
	}

	// Parse config if present, otherwise use default
	if configJSON := record.GetString("config"); configJSON != "" {
		var config models.RoomConfig
		if err := record.UnmarshalJSONField("config", &config); err == nil {
			room.Config = &config
		} else {
			room.Config = models.DefaultRoomConfig()
		}
	} else {
		room.Config = models.DefaultRoomConfig()
	}

	return room
}

// populateCurrentRound loads the current round for a room and sets state
func (h *RoomHandlers) populateCurrentRound(room *models.Room) error {
	roundRecord, err := h.roomManager.GetCurrentRoundRecord(room.ID)
	if err != nil {
		// If no round found, default to voting state
		room.State = models.StateVoting
		return err
	}

	// Populate CurrentRound
	room.CurrentRound = &models.Round{
		ID:          roundRecord.Id,
		RoomID:      roundRecord.GetString("room_id"),
		RoundNumber: roundRecord.GetInt("round_number"),
		State:       models.RoundState(roundRecord.GetString("state")),
	}

	// Set room state from round state
	room.State = room.GetState()

	return nil
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

	// Validate room ID
	if err := security.ValidateUUID(roomID); err != nil {
		return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid room ID"})
	}

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
