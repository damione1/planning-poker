package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/damione1/planning-poker/internal/config"
	"github.com/damione1/planning-poker/internal/handlers"
	"github.com/damione1/planning-poker/internal/security"
	"github.com/damione1/planning-poker/internal/services"
	_ "github.com/damione1/planning-poker/pb_migrations"
)

var (
	Version    string = "dev"
	CommitHash string = "unknown"
	BuildDate  string = "unknown"
)

func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("Planning Poker v%s\n", Version)
		fmt.Printf("Commit: %s\n", CommitHash)
		fmt.Printf("Built: %s\n", BuildDate)
		os.Exit(0)
	}
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

	statsService := services.NewStatsService(app)
	roomManager.SetStatsService(statsService)

	// Initialize handlers
	roomHandlers := handlers.NewRoomHandlers(roomManager, hub)
	wsHandler := handlers.NewWSHandler(hub, roomManager, aclService)

	// Per-IP rate limiters for the unauthenticated room-creation and join
	// endpoints, guarding against disk/DB exhaustion from creation floods.
	createRoomLimiter := security.NewIPRateLimiter(config.RoomCreatesPerIPPerMinute, time.Minute)
	joinRoomLimiter := security.NewIPRateLimiter(config.RoomJoinsPerIPPerMinute, time.Minute)

	// Schedule daily cleanup job for expired rooms (runs at midnight)
	app.Cron().MustAdd("cleanup_expired_rooms", "0 0 * * *", func() {
		cleanupExpiredRooms(app, statsService)
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Periodically observe connection/room gauges and flush accumulated
		// usage counters into the single stats_daily row for today.
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				statsService.ObserveGauges(hub.GetTotalConnections(), hub.GetRoomCount())
				if err := statsService.Flush(); err != nil {
					log.Printf("[Stats] Failed to flush usage stats: %v", err)
				}
			}
		}()

		// Page routes
		se.Router.GET("/", handlers.Home)
		se.Router.POST("/room", handlers.WithRateLimit(createRoomLimiter, roomHandlers.CreateRoom))
		se.Router.GET("/room/{id}", roomHandlers.RoomView)
		se.Router.POST("/room/{id}/join", handlers.WithRateLimit(joinRoomLimiter, roomHandlers.JoinRoom))
		se.Router.GET("/room/{id}/participants", roomHandlers.ParticipantGridFragment)
		se.Router.GET("/room/{id}/qr", roomHandlers.QRCodeHandler)

		// WebSocket route
		se.Router.GET("/ws/{roomId}", wsHandler.HandleWebSocket)

		// Monitoring routes - use /monitoring/* instead of /api/* to avoid conflicts with PocketBase's API
		se.Router.GET("/monitoring/metrics", handlers.HandleMetrics(hub))
		se.Router.GET("/monitoring/health", handlers.HandleHealth(hub))

		// Static files - must be registered last with wildcard path
		// Serves files from web/static directory at /static/* URL path
		se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./web/static"), false))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func cleanupExpiredRooms(app *pocketbase.PocketBase, stats *services.StatsService) {
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

	if stats != nil && len(roomRecords) > 0 {
		stats.Inc("rooms_expired", int64(len(roomRecords)))
	}

	for _, room := range roomRecords {
		if err := app.Delete(room); err != nil {
			log.Printf("[Cleanup] Error deleting expired room %s: %v", room.Id, err)
		} else {
			log.Printf("[Cleanup] Deleted expired room: %s (%s), expired at: %s",
				room.Id, room.GetString("name"), room.GetString("expires_at"))
		}
	}

	if stats != nil {
		if err := stats.Flush(); err != nil {
			log.Printf("[Cleanup] Failed to flush usage stats: %v", err)
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
