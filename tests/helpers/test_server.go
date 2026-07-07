package helpers

import (
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/damione1/planning-poker/internal/handlers"
	"github.com/damione1/planning-poker/internal/services"

	// Import migrations to register them (directory path, not package name)
	_ "github.com/damione1/planning-poker/pb_migrations"
)

// TestServer wraps a PocketBase test instance
type TestServer struct {
	App core.App
	t   *testing.T
}

// NewTestServer creates a new test PocketBase instance with in-memory database
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Create temporary directory for test database
	testDir := t.TempDir()

	// Create test app with temporary directory
	app, err := tests.NewTestApp(testDir)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	// Bootstrap app (runs migrations)
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap test app: %v", err)
	}

	return &TestServer{
		App: app,
		t:   t,
	}
}

// NewTestServerWithData creates a test server and applies migrations from the project
func NewTestServerWithData(t *testing.T) *TestServer {
	t.Helper()

	testDir := t.TempDir()

	// Create PocketBase app with test directory
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:   testDir,
		DataMaxOpenConns: 1,
		DataMaxIdleConns: 1,
	})

	// Bootstrap the app to initialize the database
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap app: %v", err)
	}

	// Run all registered migrations
	// (migrations are registered via init() functions when the package is imported)
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return &TestServer{
		App: app,
		t:   t,
	}
}

// Cleanup closes the test server and removes temporary files.
//
// Deliberately does NOT call app.ResetBootstrapState(): that write raced
// (under `go test -race`) against server goroutines started by
// StartTestServer -- notably the Hub's Run() loop and, for auto-reveal
// tests, the server-side time.AfterFunc that calls back into the app ~1.5s
// after the last vote -- which can still be alive when this runs, since the
// test's `defer cleanup()` fires before the `t.Cleanup`-registered
// TestHTTPServer.Close (t.Cleanup callbacks run LIFO *after* the test
// function's own defers). Resetting bootstrap state here is "best effort"
// only, and t.TempDir() already guarantees the on-disk test DB directory is
// removed, so dropping it trades a redundant, racy cleanup step for a
// deterministic, race-free test suite.
func (ts *TestServer) Cleanup() {}

// TestHTTPServer wraps an httptest server whose mux is wired to the real
// production Hub, RoomManager, ACLService and WSHandler (see main.go for the
// production wiring this mirrors). It lets integration tests drive genuine
// WebSocket connections end-to-end against a test PocketBase app.
type TestHTTPServer struct {
	// URL is the host:port of the server (no scheme), so callers can build
	// either "http://" or "ws://" URLs from it, e.g. ConnectTestClient.
	URL string

	Hub         *services.Hub
	RoomManager *services.RoomManager
	ACLService  *services.ACLService

	server *httptest.Server
}

// StartTestServer starts an HTTP test server with WebSocket support, wired
// exactly like production (see main.go): a real Hub with its Run() loop
// started, a RoomManager over app, an ACLService, and a WSHandler registered
// on the /ws/{roomId} route, plus the minimal HTTP routes tests need to
// exercise the room/join flow. app must already be bootstrapped (e.g. via
// SetupTestApp / NewTestServerWithData).
func StartTestServer(t *testing.T, app core.App) *TestHTTPServer {
	t.Helper()

	roomManager := services.NewRoomManager(app)
	aclService := services.NewACLService(roomManager)
	hub := services.NewHub()
	go hub.Run()

	roomHandlers := handlers.NewRoomHandlers(roomManager, hub)
	wsHandler := handlers.NewWSHandler(hub, roomManager, aclService)

	pbRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("Failed to create test router: %v", err)
	}

	// Mirror the production route wiring from main.go's OnServe hook.
	pbRouter.GET("/", handlers.Home)
	pbRouter.POST("/room", roomHandlers.CreateRoom)
	pbRouter.GET("/room/{id}", roomHandlers.RoomView)
	pbRouter.POST("/room/{id}/join", roomHandlers.JoinRoom)
	pbRouter.GET("/room/{id}/participants", roomHandlers.ParticipantGridFragment)
	pbRouter.GET("/ws/{roomId}", wsHandler.HandleWebSocket)

	mux, err := pbRouter.BuildMux()
	if err != nil {
		t.Fatalf("Failed to build test router mux: %v", err)
	}

	httpServer := httptest.NewServer(mux)

	testServer := &TestHTTPServer{
		URL:         httpServer.Listener.Addr().String(),
		Hub:         hub,
		RoomManager: roomManager,
		ACLService:  aclService,
		server:      httpServer,
	}

	t.Cleanup(testServer.Close)

	return testServer
}

// Close shuts down the test HTTP server and its Hub. The httptest server is
// closed first (waiting for in-flight handlers to finish, per
// httptest.Server.Close's contract), then the Hub is shut down, which stops
// its Run() goroutine and closes any still-registered clients. This ensures
// no server-side goroutine (Hub.Run, or a handler's time.AfterFunc reveal
// callback) can still be touching the app by the time this returns.
func (s *TestHTTPServer) Close() {
	if s.server != nil {
		s.server.Close()
	}
	if s.Hub != nil {
		s.Hub.Shutdown()
	}
}
