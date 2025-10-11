package utils

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/pocketbase/pocketbase/core"
)

var index bleve.Index

func RegisterSearch(se *core.ServeEvent, docDir string) {
	initSearchIndex(docDir)
	se.Router.POST("/search", searchHandler)
}

type MDFile struct {
	Path    string
	Content string
}

func searchHandler(e *core.RequestEvent) error {
	query := e.Request.FormValue("searchInput")
	matches := queryIndex(query)
	tmpl := e.App.Store().Get("tmpl").(*template.Template)
	tmpl.ExecuteTemplate(e.Response, "searchresults", matches)
	return nil
}

func queryIndex(search string) []map[string]interface{} {
	query := bleve.NewMatchQuery(search)
	request := bleve.NewSearchRequest(query)
	request.Highlight = bleve.NewHighlight()
	searchResults, err := index.Search(request)
	if err != nil {
		panic(err)
	}

	var matches []map[string]interface{}
	for _, hit := range searchResults.Hits {
		result := hit.ID
		trimmedFile := strings.TrimSuffix(strings.TrimPrefix(result, "web/docs/"), ".md")
		highlight := strings.Join(hit.Fragments["Content"], "")

		href := trimmedFile
		if href == "index" {
			href = ""
		}

		matches = append(matches, map[string]interface{}{
			"File":      trimmedFile,
			"Highlight": template.HTML(highlight),
			"Href":      "/docs/" + href,
		})
	}

	return matches
}

func initSearchIndex(docDir string) {
	var err error
	indexMapping := bleve.NewIndexMapping()
	index, err = bleve.NewMemOnly(indexMapping)
	if err != nil {
		panic(err)
	}

	err = filepath.Walk(docDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				panic(err)
			}

			file := strings.TrimSuffix(strings.TrimPrefix(path, "docs/"), ".md")
			mdFile := MDFile{
				Path:    file,
				Content: string(content),
			}

			err = index.Index(path, mdFile)
			if err != nil {
				panic(err)
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
