package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/damione1/planning-poker/internal/config"
	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/tests/helpers"
)

// bulkInsertBareRooms directly inserts n minimal room records (skipping the
// round creation and config marshaling CreateRoom normally does) so tests
// can cheaply approach config.MaxRoomsInDB without exercising the full
// CreateRoom path thousands of times.
func bulkInsertBareRooms(t *testing.T, app core.App, n int) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("rooms")
	if err != nil {
		t.Fatalf("failed to find rooms collection: %v", err)
	}

	for i := 0; i < n; i++ {
		record := core.NewRecord(collection)
		record.Set("name", "Bulk Room")
		record.Set("pointing_method", "custom")
		record.Set("state", "voting")
		record.Set("expires_at", time.Now().Add(24*time.Hour))
		record.Set("last_activity", time.Now())
		if err := app.Save(record); err != nil {
			t.Fatalf("failed to bulk insert room %d: %v", i, err)
		}
	}
}

// TestRoomManager_CreateRoom_EnforcesHardDBCap verifies that CreateRoom
// rejects new rooms once the database already holds config.MaxRoomsInDB
// rows, and that it still allows the very last room up to the cap (no
// off-by-one).
func TestRoomManager_CreateRoom_EnforcesHardDBCap(t *testing.T) {
	server := helpers.NewTestServerWithData(t)
	t.Cleanup(server.Cleanup)

	rm := services.NewRoomManager(server.App)

	// Bring the room count to exactly MaxRoomsInDB-1 directly, so the next
	// CreateRoom call is the one that would land exactly on the cap.
	bulkInsertBareRooms(t, server.App, config.MaxRoomsInDB-1)

	count, err := server.App.CountRecords("rooms")
	if err != nil {
		t.Fatalf("failed to count rooms: %v", err)
	}
	if int(count) != config.MaxRoomsInDB-1 {
		t.Fatalf("expected %d rooms before boundary check, got %d", config.MaxRoomsInDB-1, count)
	}

	// At MaxRoomsInDB-1 existing rooms, creating one more should succeed and
	// bring the total to exactly MaxRoomsInDB.
	room, err := rm.CreateRoom("Boundary Room", "fibonacci", nil, nil)
	if err != nil {
		t.Fatalf("expected room creation at the boundary to succeed, got error: %v", err)
	}
	if room == nil {
		t.Fatal("expected a non-nil room record")
	}

	// Now at MaxRoomsInDB rooms, the next creation must be rejected.
	_, err = rm.CreateRoom("Over The Cap", "fibonacci", nil, nil)
	if err == nil {
		t.Fatal("expected room creation over the cap to fail")
	}
	if !errors.Is(err, services.ErrTooManyRooms) {
		t.Fatalf("expected ErrTooManyRooms, got: %v", err)
	}
}
