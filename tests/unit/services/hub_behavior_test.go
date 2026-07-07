package services_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/damione1/planning-poker/internal/config"
	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
)

// newDeliveryTestClient creates a real *services.Client wired exactly like
// production (the Client wraps the server-accepted side of the websocket,
// with Start() running its read/write pumps), and returns a channel fed by
// a reader goroutine on the test-owned (dialed) side of the same
// connection. This lets tests observe what the hub/client actually writes
// to the wire, unlike newRaceTestClient (used by the register/unregister
// race tests) which wraps the test-dialed side and never starts the pumps,
// so nothing is ever actually delivered.
func newDeliveryTestClient(t *testing.T, hub *services.Hub, roomID, participantID string) (*services.Client, <-chan []byte, func()) {
	t.Helper()

	clientCh := make(chan *services.Client, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		cl := services.NewClient(c, hub, roomID, participantID)
		clientCh <- cl
		cl.Start()
		<-cl.Done()
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	wsURL := "ws" + srv.URL[len("http"):]
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	cancel()
	if err != nil {
		srv.Close()
		t.Fatalf("failed to dial test websocket server: %v", err)
	}

	var client *services.Client
	select {
	case client = <-clientCh:
	case <-time.After(2 * time.Second):
		srv.Close()
		t.Fatal("timed out waiting for server to accept websocket and construct Client")
	}

	received := make(chan []byte, 16)
	go func() {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			received <- data
		}
	}()

	cleanup := func() {
		client.Close()
		_ = conn.Close(websocket.StatusNormalClosure, "")
		srv.Close()
	}

	return client, received, cleanup
}

// waitUntil polls cond every 10ms until it returns true or the deadline
// elapses, then fails the test. Registration/unregistration is processed
// asynchronously by the hub's Run() goroutine (via the events channel), so
// tests that just called Register/Unregister must poll for the effect
// rather than asserting immediately.
func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("condition not met within %v", timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestHub_Register_ReflectsInRoomSize verifies that registering a client
// makes it visible via GetRoomSize.
func TestHub_Register_ReflectsInRoomSize(t *testing.T) {
	hub := services.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	const roomID = "behavior-register"

	client, cleanup := newRaceTestClient(t, hub, roomID, "p1")
	defer cleanup()

	hub.Register(roomID, client)

	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomID) == 1 })
}

// TestHub_Unregister_RemovesClientAndDropsEmptyRoom verifies that
// unregistering a client removes it from the room, and once the room is
// empty GetRoomCount reflects that the room itself was cleaned up.
func TestHub_Unregister_RemovesClientAndDropsEmptyRoom(t *testing.T) {
	hub := services.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	const roomID = "behavior-unregister"

	client, cleanup := newRaceTestClient(t, hub, roomID, "p1")
	defer cleanup()

	hub.Register(roomID, client)
	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomID) == 1 })

	baselineRoomCount := hub.GetRoomCount()
	if baselineRoomCount != 1 {
		t.Fatalf("expected 1 room after register, got %d", baselineRoomCount)
	}

	hub.Unregister(roomID, client)

	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomID) == 0 })
	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomCount() == baselineRoomCount-1 })

	if got := hub.GetClient(roomID, "p1"); got != nil {
		t.Errorf("expected no client found for p1 after unregister, got %v", got)
	}
}

// TestHub_BroadcastToRoom_ReachesOnlyInRoomClients verifies that
// BroadcastToRoom delivers to clients in the target room and does not
// deliver to a client registered in a different room.
func TestHub_BroadcastToRoom_ReachesOnlyInRoomClients(t *testing.T) {
	hub := services.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	const roomA = "behavior-broadcast-a"
	const roomB = "behavior-broadcast-b"

	inRoom, inRoomReceived, cleanupIn := newDeliveryTestClient(t, hub, roomA, "in-room")
	defer cleanupIn()
	outOfRoom, outOfRoomReceived, cleanupOut := newDeliveryTestClient(t, hub, roomB, "out-of-room")
	defer cleanupOut()

	hub.Register(roomA, inRoom)
	hub.Register(roomB, outOfRoom)

	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomA) == 1 })
	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomB) == 1 })

	hub.BroadcastToRoom(roomA, &models.WSMessage{
		Type:    "vote_cast",
		Payload: map[string]any{"hello": "roomA-only"},
	})

	// inRoom should actually receive the broadcast over the wire.
	select {
	case msg := <-inRoomReceived:
		if len(msg) == 0 {
			t.Error("expected non-empty message delivered to in-room client")
		}
	case <-time.After(2 * time.Second):
		t.Error("expected in-room client to receive broadcast message")
	}

	// outOfRoom must NOT receive anything from a broadcast scoped to roomA.
	select {
	case msg := <-outOfRoomReceived:
		t.Errorf("out-of-room client unexpectedly received message: %s", msg)
	case <-time.After(300 * time.Millisecond):
		// expected: no message delivered
	}
}

// TestHub_CanRegister_ErrRoomFullAtCapacity verifies CanRegister rejects a
// room once it has reached config.MaxConnectionsPerRoom registered clients.
func TestHub_CanRegister_ErrRoomFullAtCapacity(t *testing.T) {
	hub := services.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	const roomID = "behavior-room-full"

	var cleanups []func()
	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()

	for i := 0; i < config.MaxConnectionsPerRoom; i++ {
		client, cleanup := newRaceTestClient(t, hub, roomID, "p"+strconv.Itoa(i))
		cleanups = append(cleanups, cleanup)
		hub.Register(roomID, client)
	}

	waitUntil(t, 5*time.Second, func() bool { return hub.GetRoomSize(roomID) == config.MaxConnectionsPerRoom })

	if err := hub.CanRegister(roomID); err != services.ErrRoomFull {
		t.Errorf("expected ErrRoomFull once room at capacity, got %v", err)
	}
}

// TestHub_HasOtherClient verifies HasOtherClient reports whether a
// different client for the same participantID is registered, and correctly
// excludes the passed-in client from the match.
func TestHub_HasOtherClient(t *testing.T) {
	hub := services.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	const roomID = "behavior-has-other"
	const participantID = "shared-participant"

	client1, cleanup1 := newRaceTestClient(t, hub, roomID, participantID)
	defer cleanup1()

	hub.Register(roomID, client1)
	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomID) == 1 })

	// Only one registered client for this participant: excluding it should
	// leave no "other" match.
	if hub.HasOtherClient(roomID, participantID, client1) {
		t.Error("expected no other client when the sole registrant is excluded")
	}

	// A second connection for the same participant (e.g. a page refresh)
	// registers alongside the first.
	client2, cleanup2 := newRaceTestClient(t, hub, roomID, participantID)
	defer cleanup2()
	hub.Register(roomID, client2)
	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomID) == 2 })

	if !hub.HasOtherClient(roomID, participantID, client1) {
		t.Error("expected client2 to be found as an other client when excluding client1")
	}
	if !hub.HasOtherClient(roomID, participantID, client2) {
		t.Error("expected client1 to be found as an other client when excluding client2")
	}
}

// TestHub_Shutdown_ClosesClientsAndStopsRun verifies that Shutdown closes
// all registered clients and causes Run() to return.
func TestHub_Shutdown_ClosesClientsAndStopsRun(t *testing.T) {
	hub := services.NewHub()

	runReturned := make(chan struct{})
	go func() {
		hub.Run()
		close(runReturned)
	}()

	const roomID = "behavior-shutdown"

	client, cleanup := newRaceTestClient(t, hub, roomID, "p1")
	defer cleanup()

	hub.Register(roomID, client)
	waitUntil(t, 2*time.Second, func() bool { return hub.GetRoomSize(roomID) == 1 })

	hub.Shutdown()

	select {
	case <-runReturned:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("expected Run() to return after Shutdown()")
	}

	select {
	case <-client.Done():
		// expected: Shutdown closed the client
	case <-time.After(2 * time.Second):
		t.Fatal("expected client to be closed by Shutdown()")
	}

	// Calling Shutdown again must not panic (sync.Once-guarded).
	hub.Shutdown()
}
