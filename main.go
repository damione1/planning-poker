package main

import (
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
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
	roomHandlers := handlers.NewRoomHandlers(roomManager)
	wsHandler := handlers.NewWSHandler(hub, roomManager)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Static files handler
		se.Router.GET("/static/*", func(re *core.RequestEvent) error {
			http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))).ServeHTTP(re.Response, re.Request)
			return nil
		})

		// Page routes
		se.Router.GET("/", handlers.Home)
		se.Router.POST("/room", roomHandlers.CreateRoom)
		se.Router.GET("/room/:id", roomHandlers.RoomView)

		// WebSocket route
		se.Router.GET("/ws/:roomId", wsHandler.HandleWebSocket)

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}


