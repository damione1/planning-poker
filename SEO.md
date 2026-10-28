# Public landing and search indexing

The durable public URL is `https://pokerplanning.net/`. The room-creation form stays near the top of the landing; the explanatory copy, voting-scale guide and practical questions are rendered by Go below it. Keep claims grounded in the app: no registration, temporary rooms, independent votes, supported scales and actual session controls. Do not publish invented usage counters or connection benchmarks.

`web/templates/metadata.go` owns the public title, description, canonical origin, social image and WebApplication identity. It never derives public URLs from the request Host. The schema describes functionality without reviews or ratings. The public description and visible content should remain consistent.

`internal/handlers/seo.go` registers the exact root `/{$}`, a text robots file and an XML sitemap. A plain `/` pattern would match unknown paths in the Go/PocketBase router and turn missing URLs into homepage responses. Only durable public pages belong in the sitemap. Room pages and fragments carry noindex; their IDs must not be listed in the sitemap or canonicalized to the homepage. Rooms remain crawlable so crawlers can read noindex. This is an indexing policy, not access control.

Room scripts and Microsoft Teams sharing load only for rooms. Deferred application scripts keep their order. Saved room settings initialize on `alpine:initialized`, after the stores are registered. Successful htmx room creation uses `HX-Redirect`; native form posts retain the 303 redirect.

## Build and verify

Use the generator version pinned to the runtime; a different global templ binary can generate incompatible Go code.

```sh
go run github.com/a-h/templ/cmd/templ@v0.3.943 generate
npm run build:css
go test ./...
node --test tests/frontend/room-settings.test.cjs
go build -o /tmp/planning-poker-preview .
```

Regression tests cover exact routing, unknown URL 404s, robots/sitemap types, canonical Host safety, room/error noindex, page-specific scripts, htmx/native redirects and saved permission initialization. The frontend regression runs in CI alongside the Go tests.

For manual browser checks, use a new temporary `--dir` with the preview server. Do not use the existing `pb_data` or create test rooms on production. Check the form at 320/375 px, scale selection and custom values, settings expansion, create/join navigation, WebSocket connection and persisted permission controls.

After a production release, recheck HTTP responses and submit `https://pokerplanning.net/sitemap.xml` to its Search Console property. Compare search impressions and relevant queries over several weeks; local Lighthouse numbers do not establish production performance or a ranking gain. The site has server-side usage counters; attribution and the acquisition funnel need separate measurement.
