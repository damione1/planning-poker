package services_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/internal/services"
)

// newRaceTestClient creates a real, usable *services.Client backed by an
// actual WebSocket connection (via an in-process httptest server). Hub.Close
// on unregister/capacity-rejection calls client.conn.Close(), so the client
// needs a genuine *websocket.Conn rather than a nil/fake one.
//
// It returns the client plus a cleanup func that tears down both ends of
// the connection.
func newRaceTestClient(t *testing.T, hub *services.Hub, roomID, participantID string) (*services.Client, func()) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		// Keep the server side alive until the client disconnects; we don't
		// need to do anything with incoming frames for this test.
		defer func() { _ = c.Close(websocket.StatusNormalClosure, "") }()
		ctx := r.Context()
		for {
			if _, _, err := c.Read(ctx); err != nil {
				return
			}
		}
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	wsURL := "ws" + srv.URL[len("http"):]
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	cancel()
	if err != nil {
		srv.Close()
		t.Fatalf("failed to dial test websocket server: %v", err)
	}

	client := services.NewClient(conn, hub, roomID, participantID)

	cleanup := func() {
		client.Close()
		srv.Close()
	}

	return client, cleanup
}

// TestHub_ConcurrentAccess_NoRace hammers the hub with many goroutines doing
// Register, Unregister, BroadcastToRoom and GetClient concurrently, across
// both a shared room and per-goroutine rooms. It is designed to trip the
// historical "concurrent map iteration and map write" fatal panic that
// existed when Hub.rooms was a sync.Map whose map[*Client]bool values were
// mutated by the hub's Run() goroutine while being read/iterated directly
// by BroadcastToRoom/GetClient/CanRegister/GetRoomSize from other
// goroutines. Run with `go test -race` to catch any reintroduced races.
func TestHub_ConcurrentAccess_NoRace(t *testing.T) {
	hub := services.NewHub()
	go hub.Run()

	const sharedRoomID = "shared-room"
	const numWorkers = 40
	const opsPerWorker = 25

	var wg sync.WaitGroup
	var cleanupMu sync.Mutex
	var cleanups []func()

	registerCleanup := func(fn func()) {
		cleanupMu.Lock()
		cleanups = append(cleanups, fn)
		cleanupMu.Unlock()
	}
	defer func() {
		cleanupMu.Lock()
		defer cleanupMu.Unlock()
		for _, fn := range cleanups {
			fn()
		}
	}()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()

			// Half the workers hammer a single shared room (maximizing
			// contention on one room's client set), the other half use
			// their own room (exercising concurrent room creation/cleanup).
			roomID := sharedRoomID
			if workerID%2 == 1 {
				roomID = fmt.Sprintf("room-%d", workerID)
			}

			for j := 0; j < opsPerWorker; j++ {
				participantID := "participant-" + strconv.Itoa(workerID) + "-" + strconv.Itoa(j)

				client, cleanup := newRaceTestClient(t, hub, roomID, participantID)
				registerCleanup(cleanup)

				// Register and immediately fire off concurrent readers
				// against the hub while the register/unregister for THIS
				// client races through the hub's event loop.
				hub.Register(roomID, client)

				var innerWG sync.WaitGroup
				innerWG.Add(3)

				go func() {
					defer innerWG.Done()
					hub.BroadcastToRoom(roomID, &models.WSMessage{
						Type:    "vote_cast",
						Payload: map[string]any{"participantId": participantID},
					})
				}()

				go func() {
					defer innerWG.Done()
					_ = hub.GetClient(roomID, participantID)
				}()

				go func() {
					defer innerWG.Done()
					_ = hub.GetRoomSize(roomID)
					_ = hub.GetRoomCount()
					_ = hub.GetTotalConnections()
					_ = hub.CanRegister(roomID)
				}()

				innerWG.Wait()

				hub.Unregister(roomID, client)
			}
		}()
	}

	wg.Wait()

	// Give the hub's event loop a moment to drain the last unregister
	// events before asserting final state.
	deadline := time.Now().Add(2 * time.Second)
	for hub.GetTotalConnections() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if got := hub.GetTotalConnections(); got != 0 {
		t.Errorf("expected 0 total connections after all unregisters drained, got %d", got)
	}
	if got := hub.GetRoomCount(); got != 0 {
		t.Errorf("expected 0 rooms after all unregisters drained, got %d", got)
	}
}

// TestHub_RegisterUnregister_FIFOOrdering verifies that a client's register
// and unregister events are processed in the order they were issued. With
// the old design (separate register/unregister channels combined via a
// random `select`), a fast Register-then-Unregister pair from the same
// caller could have its unregister processed before its register,
// permanently leaking a "phantom" client entry. Using a single events
// channel preserves per-client FIFO ordering.
func TestHub_RegisterUnregister_FIFOOrdering(t *testing.T) {
	hub := services.NewHub()
	go hub.Run()

	const roomID = "fifo-room"
	const iterations = 200

	for i := 0; i < iterations; i++ {
		client, cleanup := newRaceTestClient(t, hub, roomID, "participant-fifo")
		hub.Register(roomID, client)
		hub.Unregister(roomID, client)
		cleanup()
	}

	deadline := time.Now().Add(2 * time.Second)
	for hub.GetTotalConnections() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if got := hub.GetTotalConnections(); got != 0 {
		t.Fatalf("expected 0 leaked connections after FIFO register/unregister, got %d", got)
	}
	if got := hub.GetRoomSize(roomID); got != 0 {
		t.Fatalf("expected empty room after FIFO register/unregister, got size %d", got)
	}
}
