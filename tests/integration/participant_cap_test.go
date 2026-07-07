package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/damione1/planning-poker/internal/config"
	"github.com/damione1/planning-poker/tests/helpers"
)

// postJoin submits POST /room/{id}/join exactly like the join form does,
// with a fresh (cookie-less) request so the handler always takes the
// "create a new participant" branch rather than reusing a session.
func postJoin(t *testing.T, baseURL, roomID, name string) *http.Response {
	t.Helper()

	form := url.Values{"name": {name}, "role": {"voter"}}
	resp, err := http.Post(
		fmt.Sprintf("http://%s/room/%s/join", baseURL, roomID),
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("POST /room/%s/join failed: %v", roomID, err)
	}
	return resp
}

// TestJoinRoom_EnforcesParticipantCap verifies that once a room holds
// config.MaxParticipantsPerRoom participants, further join attempts are
// rejected (403) rather than growing the participants table without bound,
// while a join that lands exactly on the cap still succeeds (no off-by-one).
func TestJoinRoom_EnforcesParticipantCap(t *testing.T) {
	app, cleanup := helpers.SetupTestApp(t)
	defer cleanup()

	// Seed the room with MaxParticipantsPerRoom-1 voters directly (fast),
	// so the HTTP join below is the one that would land exactly on the cap.
	roomID := helpers.CreateTestRoomWithParticipants(t, app, config.MaxParticipantsPerRoom-1, nil)

	ts := helpers.StartTestServer(t, app)
	defer ts.Close()

	// This join brings the room to exactly MaxParticipantsPerRoom and must succeed.
	resp := postJoin(t, ts.URL, roomID, "Boundary Participant")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected join at the boundary to succeed with 200, got %d: %s", resp.StatusCode, body)
	}

	participants, err := ts.RoomManager.GetRoomParticipants(roomID)
	if err != nil {
		t.Fatalf("failed to fetch participants: %v", err)
	}
	if len(participants) != config.MaxParticipantsPerRoom {
		t.Fatalf("expected %d participants after boundary join, got %d", config.MaxParticipantsPerRoom, len(participants))
	}

	// The next join is over the cap and must be rejected.
	resp = postJoin(t, ts.URL, roomID, "Over The Cap")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected join over the cap to be rejected with 403, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "full") {
		t.Errorf("expected rejection body to mention the room being full, got: %s", body)
	}

	participants, err = ts.RoomManager.GetRoomParticipants(roomID)
	if err != nil {
		t.Fatalf("failed to fetch participants: %v", err)
	}
	if len(participants) != config.MaxParticipantsPerRoom {
		t.Fatalf("expected participant count to stay at %d after rejected join, got %d", config.MaxParticipantsPerRoom, len(participants))
	}
}
