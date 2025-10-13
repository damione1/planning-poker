package handlers

import (
	"net/http"

	"github.com/damione1/planning-poker/internal/services"
	"github.com/pocketbase/pocketbase/core"
)

// HandleMetrics returns WebSocket server metrics
func HandleMetrics(hub *services.Hub) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		snapshot := hub.GetMetrics()
		return e.JSON(http.StatusOK, snapshot)
	}
}

// HandleHealth returns server health status
func HandleHealth(hub *services.Hub) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		snapshot := hub.GetMetrics()

		status := http.StatusOK
		switch snapshot.HealthStatus {
		case "critical":
			status = http.StatusServiceUnavailable
		case "warning":
			status = http.StatusOK // Still return 200, but with warning in body
		default:
			status = http.StatusOK
		}

		response := map[string]interface{}{
			"status":             snapshot.HealthStatus,
			"active_connections": snapshot.ActiveConnections,
			"active_rooms":       snapshot.ActiveRooms,
			"uptime_seconds":     snapshot.UptimeSeconds,
		}

		return e.JSON(status, response)
	}
}
