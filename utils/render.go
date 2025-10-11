package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/russross/blackfriday/v2"

	"github.com/pocketbase/pocketbase/core"
)

type RenderData struct {
	Auth *core.Record
	Data interface{}
}

func RenderView(view string, data interface{}) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		tmplData := RenderData{
			Auth: e.Auth,
			Data: data,
		}

		tmpl := e.App.Store().Get("tmpl").(*template.Template)
		if tmpl == nil {
			return e.Error(http.StatusInternalServerError, "tmpl not initialized in store", nil)
		}

		err := tmpl.ExecuteTemplate(e.Response, view, tmplData)
		if err != nil {
			return err
		}
		return nil
	}
}

type RenderDocData struct {
	Auth    *core.Record
	Content template.HTML
}

// RenderDocViewHandler takes a markdown file, renders it into HTML, and
// then renders it a template with a right navigation sidebar.
func RenderDocViewHandler(tmpl *template.Template) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		dir := "./web/docs/"
		doc := e.Request.PathValue("doc")

		// Load markdown
		markdownContent, err := os.ReadFile(dir + doc + ".md")
		if err != nil {
			return err
		}

		// Parse markdown into goquery document
		htmlDocContent := blackfriday.Run(markdownContent)
		queryDoc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlDocContent))
		if err != nil {
			return err
		}

		// Load headings to add right sidebar navigation with slug id
		type Header struct {
			Text string
			ID   string
		}
		var headers []Header

		// Tailwind themes
		queryDoc.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			slug := strings.ToLower(text)
			slug = strings.ReplaceAll(slug, " ", "-")
			headers = append(headers, Header{Text: text, ID: slug})
			s.SetAttr("id", slug)
		})

		// Use goquery to add Bootstrap styling by adding CSS classes
		queryDoc.Find("h1").Each(func(i int, s *goquery.Selection) {
			s.AddClass("display-4 fw-bold mb-4")
		})
		queryDoc.Find("h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
			s.AddClass("h2 fw-bold mb-3")
		})

		queryDoc.Find("p").Each(func(i int, s *goquery.Selection) {
			s.AddClass("mb-3")
		})

		queryDoc.Find("pre").Each(func(i int, s *goquery.Selection) {
			s.AddClass("border rounded border")
		})
		queryDoc.Find("code").Each(func(i int, s *goquery.Selection) {
			s.AddClass("rounded")
		})

		queryDoc.Find("a").Each(func(i int, s *goquery.Selection) {
			s.AddClass("link-opacity-100")
		})

		queryDoc.Find("ul").Each(func(i int, s *goquery.Selection) {
			s.AddClass("mb-3")
		})
		queryDoc.Find("ol").Each(func(i int, s *goquery.Selection) {
			s.AddClass("mb-3")
		})

		queryDoc.Find("img").Each(func(i int, s *goquery.Selection) {
			s.AddClass("code-block shadow-sm rounded border w-100")
		})

		// Render document into HTML after goquery processing
		renderedHtmlDoc, err := queryDoc.Html()
		if err != nil {
			return err
		}

		// Define common and custom data
		tmplData := RenderDocData{
			Auth:    e.Auth,
			Content: template.HTML(renderedHtmlDoc),
		}

		// Render View
		if e.Request.Header.Get("Hx-Target") == "doccontent" {
			return e.HTML(http.StatusOK, renderedHtmlDoc)
		} else {
			return tmpl.ExecuteTemplate(e.Response, "docs.html", tmplData)
		}
	}
}

func Dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}
