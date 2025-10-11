package main

import (
	"log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/damione1/planning-poker/internal/handlers"
	"github.com/damione1/planning-poker/internal/services"
	_ "github.com/damione1/planning-poker/pb_migrations"
)

func main() {
	app := pocketbase.New()

	// Register migrate command with automigrate enabled
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true, // Auto-run migrations on app.Start()
	})

	// Initialize services
	roomManager := services.NewRoomManager(app)
	aclService := services.NewACLService(roomManager)
	hub := services.NewHub()
	go hub.Run()

	// Initialize handlers
	roomHandlers := handlers.NewRoomHandlers(roomManager, hub)
	wsHandler := handlers.NewWSHandler(hub, roomManager, aclService)

	// Schedule daily cleanup job for expired rooms (runs at midnight)
	app.Cron().MustAdd("cleanup_expired_rooms", "0 0 * * *", func() {
		cleanupExpiredRooms(app)
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Page routes
		se.Router.GET("/", handlers.Home)
		se.Router.POST("/room", roomHandlers.CreateRoom)
		se.Router.GET("/room/{id}", roomHandlers.RoomView)
		se.Router.POST("/room/{id}/join", roomHandlers.JoinRoom)
		se.Router.GET("/room/{id}/participants", roomHandlers.ParticipantGridFragment)
		se.Router.GET("/room/{id}/qr", roomHandlers.QRCodeHandler)

		// WebSocket route
		se.Router.GET("/ws/{roomId}", wsHandler.HandleWebSocket)

		// Static files - must be registered last with wildcard path
		// Serves files from web/static directory at /static/* URL path
		se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./web/static"), false))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func cleanupExpiredRooms(app *pocketbase.PocketBase) {
	log.Printf("[Cleanup] Starting cleanup job at %s", time.Now().Format(time.RFC3339))

	// Delete expired rooms (cascade deletes rounds and votes via database constraints)
	// PocketBase supports @now macro for current datetime comparison
	roomRecords, err := app.FindRecordsByFilter(
		"rooms",
		"expires_at < @now",
		"expires_at", // Sort by expiration date (oldest first)
		100,
		0,
	)

	if err != nil {
		log.Printf("[Cleanup] Error finding expired rooms: %v", err)
		return
	}

	log.Printf("[Cleanup] Found %d expired rooms to delete", len(roomRecords))

	for _, room := range roomRecords {
		if err := app.Delete(room); err != nil {
			log.Printf("[Cleanup] Error deleting expired room %s: %v", room.Id, err)
		} else {
			log.Printf("[Cleanup] Deleted expired room: %s (%s), expired at: %s",
				room.Id, room.GetString("name"), room.GetString("expires_at"))
		}
	}

	// Delete orphaned participants (participants whose room no longer exists)
	// This handles participants from rooms that were deleted
	participantRecords, err := app.FindRecordsByFilter(
		"participants",
		"room_id != '' && room_id.id = ''",
		"", // No sorting needed for cleanup
		500,
		0,
	)

	if err != nil {
		log.Printf("[Cleanup] Error finding orphaned participants: %v", err)
		return
	}

	log.Printf("[Cleanup] Found %d orphaned participants to delete", len(participantRecords))

	for _, participant := range participantRecords {
		if err := app.Delete(participant); err != nil {
			log.Printf("[Cleanup] Error deleting orphaned participant %s: %v", participant.Id, err)
		} else {
			log.Printf("[Cleanup] Deleted orphaned participant: %s (%s)", participant.Id, participant.GetString("name"))
		}
	}

	log.Printf("[Cleanup] Cleanup job completed successfully")
}
