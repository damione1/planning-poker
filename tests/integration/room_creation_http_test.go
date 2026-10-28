package integration_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/damione1/planning-poker/tests/helpers"
)

func TestRoomCreationNavigation(t *testing.T) {
	for _, mode := range []string{"htmx", "native"} {
		t.Run(mode, func(t *testing.T) {
			app := helpers.NewTestServerWithData(t)
			server := helpers.StartTestServer(t, app.App)
			form := url.Values{
				"name":           {"Room creation navigation"},
				"pointingMethod": {"custom"},
				"customValues":   {"1,2,3,5"},
			}
			req, err := http.NewRequest(http.MethodPost, "http://"+server.URL+"/room", strings.NewReader(form.Encode()))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if mode == "htmx" {
				req.Header.Set("HX-Request", "true")
				req.Header.Set("HX-Target", "form-errors")
			}
			// Inspect the original response instead of hiding a 303 by following it.
			client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			var roomURL string
			if mode == "htmx" {
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("htmx must receive a non-3xx redirect instruction, got %d", resp.StatusCode)
				}
				roomURL = resp.Header.Get("HX-Redirect")
				if len(body) != 0 || resp.Header.Get("Location") != "" {
					t.Fatal("htmx creation must navigate, not swap a room document into #form-errors")
				}
			} else {
				if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("HX-Redirect") != "" {
					t.Fatalf("native form must retain ordinary 303 navigation, got %d", resp.StatusCode)
				}
				roomURL = resp.Header.Get("Location")
			}
			roomID := strings.TrimPrefix(roomURL, "/room/")
			if roomID == roomURL || roomID == "" || strings.ContainsAny(roomID, "/?#") {
				t.Fatalf("unexpected navigation target: %q", roomURL)
			}
			room, err := server.RoomManager.GetRoom(roomID)
			if err != nil || room.GetString("name") != form.Get("name") {
				t.Fatalf("navigation must point to the room that was actually saved: %v", err)
			}
		})
	}
}

func TestRoomCreationHTMXValidationStaysInForm(t *testing.T) {
	app := helpers.NewTestServerWithData(t)
	server := helpers.StartTestServer(t, app.App)
	form := url.Values{"name": {""}, "pointingMethod": {"custom"}, "customValues": {"1,2,3"}}
	req, err := http.NewRequest(http.MethodPost, "http://"+server.URL+"/room", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "form-errors")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest || resp.Header.Get("HX-Redirect") != "" || resp.Header.Get("Location") != "" {
		t.Fatal("invalid form must remain an error response rather than navigate")
	}
	if len(body) == 0 || strings.Contains(strings.ToLower(string(body)), "<!doctype") {
		t.Fatal("validation should return an error fragment, not a full page")
	}
	rooms, err := app.App.FindAllRecords("rooms")
	if err != nil || len(rooms) != 0 {
		t.Fatalf("invalid form must not create a room: count=%d err=%v", len(rooms), err)
	}
}
