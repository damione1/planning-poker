package handlers

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/damione1/planning-poker/web/templates"
)

// RegisterPublicRoutes is shared with the integration-test router. The /{$}
// pattern is deliberately exact: GET / would swallow unknown paths, including
// robots.txt and sitemap.xml, and return a misleading 200 landing page.
func RegisterPublicRoutes(r *router.Router[*core.RequestEvent]) {
	r.GET("/{$}", Home)
	r.GET("/robots.txt", Robots)
	r.GET("/sitemap.xml", Sitemap)
}

func Robots(re *core.RequestEvent) error {
	re.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	re.Response.Header().Set("Cache-Control", "public, max-age=3600")
	// Room pages stay crawlable so crawlers can read their noindex directive.
	_, err := re.Response.Write([]byte("User-agent: *\nAllow: /\nDisallow: /_/\nDisallow: /api/\nDisallow: /ws/\nDisallow: /monitoring/\n\nSitemap: " + templates.PublicSiteURL + "sitemap.xml\n"))
	return err
}

func Sitemap(re *core.RequestEvent) error {
	re.Response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	re.Response.Header().Set("Cache-Control", "public, max-age=3600")
	// Only durable public pages belong here. Room IDs and query variants are
	// deliberately omitted; there is no artificial daily modification date.
	_, err := re.Response.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\"><url><loc>" + templates.PublicSiteURL + "</loc></url></urlset>\n"))
	return err
}

// WithNoIndex also covers non-HTML room fragments and redirect responses.
func WithNoIndex(next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		re.Response.Header().Set("X-Robots-Tag", "noindex")
		re.Response.Header().Set("Cache-Control", "no-store")
		return next(re)
	}
}

// Home errors are useful to people returning from an expired/missing room, but
// must not become separate search landing pages.
func setHomeHeaders(re *core.RequestEvent, hasError bool) {
	re.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	re.Response.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	if hasError {
		re.Response.Header().Set("X-Robots-Tag", "noindex")
		re.Response.Header().Set("Cache-Control", "no-store")
	}
}
