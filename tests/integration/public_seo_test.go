package integration_test

import (
	"encoding/json"
	"encoding/xml"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/damione1/planning-poker/tests/helpers"
)

const canonicalHome = "https://pokerplanning.net/"

func seoResponse(t *testing.T, client *http.Client, url string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Neither canonicals nor sitemap URLs should trust an arbitrary Host.
	req.Host = "untrusted.invalid"
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
	return resp, string(body)
}

func seoHeadValue(body, key, value, result string) string {
	tags := regexp.MustCompile(`<(?:meta|link)\b[^>]*>`).FindAllString(body, -1)
	attributes := regexp.MustCompile(`([a-zA-Z:-]+)="([^"]*)"`)
	for _, tag := range tags {
		attrs := map[string]string{}
		for _, match := range attributes.FindAllStringSubmatch(tag, -1) {
			attrs[match[1]] = html.UnescapeString(match[2])
		}
		if attrs[key] == value {
			return attrs[result]
		}
	}
	return ""
}

func TestPublicSEORoutesAndMetadata(t *testing.T) {
	app := helpers.NewTestServerWithData(t)
	server := helpers.StartTestServer(t, app.App)
	origin := "http://" + server.URL

	resp, body := seoResponse(t, http.DefaultClient, origin+"/?ref=shared")
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("home response: %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if strings.Contains(seoHeadValue(body, "name", "robots", "content"), "noindex") || strings.Contains(resp.Header.Get("X-Robots-Tag"), "noindex") {
		t.Fatal("public landing must remain indexable")
	}
	if got := seoHeadValue(body, "rel", "canonical", "href"); got != canonicalHome {
		t.Fatalf("canonical = %q", got)
	}
	if !strings.Contains(body, "<title>Online Planning Poker - No Sign-up | PokerPlanning</title>") {
		t.Fatal("missing descriptive public title")
	}
	description := seoHeadValue(body, "name", "description", "content")
	if description == "" || description != seoHeadValue(body, "property", "og:description", "content") {
		t.Fatal("description missing or differs from social description")
	}
	if seoHeadValue(body, "property", "og:url", "content") != canonicalHome || seoHeadValue(body, "property", "og:image", "content") != canonicalHome+"static/img/planning-poker-social.jpg" {
		t.Fatal("social URLs must use canonical origin")
	}
	if strings.Contains(body, "teams.microsoft.com/share") || strings.Contains(body, `src="/static/js/ws.js"`) {
		t.Fatal("landing should not load room-only scripts")
	}
	for _, script := range []string{"htmx.min.js", "alpine-components.js", "alpine.min.js"} {
		if !strings.Contains(body, `defer src="/static/js/`+script+`"`) {
			t.Fatalf("required ordered defer script missing: %s", script)
		}
	}
	match := regexp.MustCompile(`(?s)<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatal("missing software identity JSON-LD")
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(match[1]), &schema); err != nil {
		t.Fatal(err)
	}
	if schema["@type"] != "WebApplication" || schema["url"] != canonicalHome || schema["name"] != "PokerPlanning" {
		t.Fatalf("unexpected schema identity: %#v", schema)
	}
	if schema["aggregateRating"] != nil || schema["review"] != nil {
		t.Fatal("software markup must not invent ratings or reviews")
	}

	for _, path := range []string{"/missing-seo-test", "/robots.txt/not-a-file", "/sitemap_index.xml"} {
		t.Run(path, func(t *testing.T) {
			resp, _ := seoResponse(t, http.DefaultClient, origin+path)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("unknown path must not render the homepage: %s => %d", path, resp.StatusCode)
			}
		})
	}
}

func TestPublicSEOFiles(t *testing.T) {
	app := helpers.NewTestServerWithData(t)
	server := helpers.StartTestServer(t, app.App)
	origin := "http://" + server.URL

	resp, robots := seoResponse(t, http.DefaultClient, origin+"/robots.txt")
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain") {
		t.Fatalf("robots response: %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if !strings.Contains(robots, "User-agent: *\n") || !strings.Contains(robots, "Sitemap: "+canonicalHome+"sitemap.xml") || strings.Contains(robots, "Disallow: /room/") {
		t.Fatal("robots must advertise the real sitemap and permit reading room noindex")
	}
	resp, sitemap := seoResponse(t, http.DefaultClient, origin+"/sitemap.xml")
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/xml") {
		t.Fatalf("sitemap response: %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var urls struct {
		XMLName xml.Name `xml:"urlset"`
		URLs    []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal([]byte(sitemap), &urls); err != nil {
		t.Fatal(err)
	}
	if urls.XMLName.Space != "http://www.sitemaps.org/schemas/sitemap/0.9" || len(urls.URLs) != 1 || urls.URLs[0].Loc != canonicalHome {
		t.Fatalf("only the actual public home belongs in sitemap: %#v", urls)
	}
}

func TestPublicSEOErrorAndRoomExclusion(t *testing.T) {
	app := helpers.NewTestServerWithData(t)
	server := helpers.StartTestServer(t, app.App)
	origin := "http://" + server.URL

	for _, errorParam := range []string{"room_not_found", "room_expired", "invalid_room_id", "unknown"} {
		resp, body := seoResponse(t, http.DefaultClient, origin+"/?error="+errorParam)
		if !strings.Contains(resp.Header.Get("X-Robots-Tag"), "noindex") || seoHeadValue(body, "name", "robots", "content") != "noindex" || seoHeadValue(body, "rel", "canonical", "href") != canonicalHome {
			t.Fatalf("error variant must be excluded and canonicalized: %s", errorParam)
		}
	}

	// This room exists only in the test's temporary database, never in production.
	room, err := server.RoomManager.CreateRoom("Private test room", "fibonacci", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, body := seoResponse(t, http.DefaultClient, origin+"/room/"+room.Id)
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("X-Robots-Tag"), "noindex") || seoHeadValue(body, "name", "robots", "content") != "noindex" {
		t.Fatal("private room needs both HTML and HTTP noindex")
	}
	if seoHeadValue(body, "rel", "canonical", "href") != "" || strings.Contains(body, "application/ld+json") {
		t.Fatal("private room must not pretend to be the public landing")
	}
	if !strings.Contains(body, "teams.microsoft.com/share") || !strings.Contains(body, `src="/static/js/ws.js"`) {
		t.Fatal("room sharing and websocket extension must remain available in rooms")
	}

	// A nonexistent room retains its existing return-to-home flow, without
	// becoming an indexable redirect destination with an error query.
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, _ = seoResponse(t, client, origin+"/room/00000000-0000-0000-0000-000000000000")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/?error=room_not_found" || !strings.Contains(resp.Header.Get("X-Robots-Tag"), "noindex") {
		t.Fatalf("unexpected missing-room flow: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}
