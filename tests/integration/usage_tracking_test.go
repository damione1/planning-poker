package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/damione1/planning-poker/internal/models"
	"github.com/damione1/planning-poker/tests/helpers"
	"github.com/damione1/planning-poker/web/templates"
	"github.com/stretchr/testify/require"
)

func usageReceipts(clients ...*helpers.WSClient) []map[string]any {
	var events []map[string]any
	for _, client := range clients {
		for _, message := range client.ReceivedMessages() {
			if message.Type == "usage_event" {
				events = append(events, message.Payload.(map[string]any))
			}
		}
	}
	return events
}

func TestUsageReceiptsCountMutationsOnceAcrossTabsAndParticipants(t *testing.T) {
	app := helpers.NewTestServerWithData(t)
	server := helpers.StartTestServer(t, app.App)
	room, err := server.RoomManager.CreateRoom("Private Room", "custom", []string{"secret", "1"}, nil)
	require.NoError(t, err)
	host, err := server.RoomManager.AddParticipant(room.Id, "Private Host", models.RoleVoter, "private-host-cookie")
	require.NoError(t, err)
	observer, err := server.RoomManager.AddParticipant(room.Id, "Private Watcher", models.RoleSpectator, "private-observer-cookie")
	require.NoError(t, err)
	first := helpers.ConnectTestClientAs(t, server, room.Id, host.Id, "private-host-cookie")
	defer first.Close()
	second := helpers.ConnectTestClientAs(t, server, room.Id, host.Id, "private-host-cookie")
	defer second.Close()
	spectator := helpers.ConnectTestClientAs(t, server, room.Id, observer.Id, "private-observer-cookie")
	defer spectator.Close()
	for _, client := range []*helpers.WSClient{first, second, spectator} {
		require.NotNil(t, client.WaitForMessageType("room_state", time.Second))
	}
	require.Eventually(t, func() bool { return server.Hub.GetRoomSize(room.Id) == 3 }, time.Second, 10*time.Millisecond)
	send := func(name string, payload map[string]any) {
		if payload == nil {
			payload = map[string]any{}
		}
		require.NoError(t, first.SendMessage(map[string]any{"type": name, "payload": payload}))
	}
	wait := func(count int) {
		require.Eventually(t, func() bool { return len(usageReceipts(first, second)) == count }, time.Second, 10*time.Millisecond)
	}
	send("vote", map[string]any{"value": "secret"})
	wait(1)
	require.NotNil(t, spectator.WaitForMessageCount("vote_cast", 1, time.Second))
	send("vote", map[string]any{"value": "secret"}) // replaying the same value is not a second vote action
	require.NotNil(t, spectator.WaitForMessageCount("vote_cast", 2, time.Second))
	require.Len(t, usageReceipts(first, second), 1)
	send("vote", map[string]any{"value": "1"})
	wait(2)
	changed := 0
	for _, event := range usageReceipts(first, second) {
		if event["parameters"].(map[string]any)["vote_action"] == "change" {
			changed++
		}
	}
	require.Equal(t, 1, changed)
	send("reveal", nil)
	wait(3)
	send("reveal", nil) // no-op: already revealed
	send("next_round", nil)
	wait(4)
	var next map[string]any
	for _, event := range usageReceipts(first, second) {
		if event["name"] == "round_started" {
			next = event
		}
	}
	require.NotNil(t, next)
	require.Equal(t, "round_started", next["name"])
	params := next["parameters"].(map[string]any)
	require.Equal(t, float64(2), params["round_number"])
	require.Equal(t, float64(2), params["participant_count"])
	require.Equal(t, float64(1), params["voter_count"])
	require.Equal(t, float64(1), params["spectator_count"])
	send("reset", nil)
	wait(5)
	require.Empty(t, usageReceipts(spectator))
	encoded, err := json.Marshal(usageReceipts(first, second))
	require.NoError(t, err)
	for _, private := range []string{room.Id, host.Id, observer.Id, "Private", "secret", "private-host-cookie"} {
		require.NotContains(t, string(encoded), private)
	}
}

func TestAutomaticRevealProducesOneAggregateReceipt(t *testing.T) {
	app := helpers.NewTestServerWithData(t)
	server := helpers.StartTestServer(t, app.App)
	config := models.DefaultRoomConfig()
	config.Permissions.AutoReveal = true
	room, err := server.RoomManager.CreateRoom("Private Automatic", "custom", []string{"1", "2"}, config)
	require.NoError(t, err)
	voter, err := server.RoomManager.AddParticipant(room.Id, "Private Voter", models.RoleVoter, "auto-session")
	require.NoError(t, err)
	client := helpers.ConnectTestClientAs(t, server, room.Id, voter.Id, "auto-session")
	defer client.Close()
	require.NotNil(t, client.WaitForMessageType("room_state", time.Second))
	require.NoError(t, client.SendMessage(map[string]any{"type": "vote", "payload": map[string]any{"value": "1"}}))
	require.NotNil(t, client.WaitForMessageType("votes_revealed", 5*time.Second))
	events := usageReceipts(client)
	require.Len(t, events, 2)
	require.Equal(t, "round_revealed", events[1]["name"])
	params := events[1]["parameters"].(map[string]any)
	require.Equal(t, "automatic", params["reveal_method"])
	require.Equal(t, "system", params["member_type"])
	require.NotContains(t, params, "participant_role")
}

func TestHTTPUsageReceiptsOnlyFollowSuccessfulNewMembership(t *testing.T) {
	app := helpers.NewTestServerWithData(t)
	server := helpers.StartTestServer(t, app.App)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}
	cookieHeader := ""
	post := func(path string, form url.Values) *http.Response {
		req, err := http.NewRequest(http.MethodPost, "http://"+server.URL+path, strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		if cookieHeader != "" {
			req.Header.Set("Cookie", cookieHeader)
		}
		response, err := client.Do(req)
		require.NoError(t, err)
		for _, cookie := range response.Cookies() {
			cookieHeader = cookie.Name + "=" + cookie.Value
		}
		_, err = io.Copy(io.Discard, response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		return response
	}
	invalid := post("/room", url.Values{"name": {""}})
	require.Empty(t, invalid.Header.Get("HX-Trigger"))
	created := post("/room", url.Values{"name": {"Private Team"}, "pointingMethod": {"custom"}, "customValues": {"1, 2, 3"}})
	require.Equal(t, http.StatusOK, created.StatusCode)
	trigger := created.Header.Get("HX-Trigger")
	require.Contains(t, trigger, "room_created")
	require.Contains(t, trigger, "round_started")
	require.NotContains(t, trigger, "Private Team")
	roomPath := created.Header.Get("HX-Redirect")
	joined := post(roomPath+"/join", url.Values{"name": {"Private Spectator"}, "role": {"spectator"}})
	require.Contains(t, joined.Header.Get("HX-Trigger"), "room_joined")
	require.Contains(t, joined.Header.Get("HX-Trigger"), `"participant_role":"spectator"`)
	require.Contains(t, joined.Header.Get("HX-Trigger"), `"member_type":"host"`)
	rejoined := post(roomPath+"/join", url.Values{"name": {"Private Spectator"}, "role": {"spectator"}})
	require.Empty(t, rejoined.Header.Get("HX-Trigger"))
}

func TestAnalyticsMeasurementComesFromBuildConfiguration(t *testing.T) {
	previous := templates.AnalyticsMeasurementID
	t.Cleanup(func() { templates.AnalyticsMeasurementID = previous })
	for _, id := range []string{"", "G-TEST123", "G-TEST123<script>"} {
		templates.AnalyticsMeasurementID = id
		var output bytes.Buffer
		require.NoError(t, templates.Home(nil, "").Render(context.Background(), &output))
		if id == "G-TEST123" {
			require.Contains(t, output.String(), `data-measurement-id="G-TEST123"`)
		} else {
			require.NotContains(t, output.String(), "/static/js/analytics.js")
		}
	}
}
