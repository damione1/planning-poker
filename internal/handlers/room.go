package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"

	"github.com/damiengoehrig/planning-poker/internal/services"
)

type RoomHandlers struct {
	roomManager *services.RoomManager
}

func NewRoomHandlers(rm *services.RoomManager) *RoomHandlers {
	return &RoomHandlers{roomManager: rm}
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

	// Temporary placeholder
	html := `<!DOCTYPE html>
<html>
<head>
    <title>` + room.Name + `</title>
</head>
<body>
    <h1>` + room.Name + `</h1>
    <p>Room ID: ` + room.ID + `</p>
    <p>Pointing Method: ` + room.PointingMethod + `</p>
    <p>WebSocket will connect to: /ws/` + room.ID + `</p>
</body>
</html>`
	return re.HTML(200, html)
}
