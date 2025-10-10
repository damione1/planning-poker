package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/damiengoehrig/planning-poker/internal/handlers"
	"github.com/damiengoehrig/planning-poker/internal/services"
)

func main() {
	app := pocketbase.New()

	// Initialize services
	roomManager := services.NewRoomManager()
	hub := services.NewHub()
	go hub.Run()

	// Initialize handlers
	roomHandlers := handlers.NewRoomHandlers(roomManager, hub)
	wsHandler := handlers.NewWSHandler(hub, roomManager)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Page routes
		se.Router.GET("/", handlers.Home)
		se.Router.POST("/room", roomHandlers.CreateRoom)
		se.Router.GET("/room/{id}", roomHandlers.RoomView)
		se.Router.POST("/room/{id}/join", roomHandlers.JoinRoom)

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


