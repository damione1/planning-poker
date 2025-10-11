# 📖 Documentation
One of the best features of DeploySolo is the documentation system.

Documentation is stored as a folder of plain markdown files. This makes it as painless as possible to create visually appealing documentation pages, and allows seamless version control in the same codebase.

## Markdown Rendering
Consider the structure of markdown files
```sh
├── about
│   └── introduction.md
└── **/
    └── *.md
```

And the `/docs/` handler in `main.go`

```go
se.Router.GET("/docs/{doc...}", utils.RenderDocViewHandler(tmpl))
```

DeploySolo will automatically fetch the directory after `/docs/`, fetch the markdown file at the local directory, and render the formatted markdown as formatted HTML.

For example, navigating to `{URL}/docs/about/introduction.md` will cause the server to parse `./docs/about/introduction.md`, render it as formatted HTML, and display the page.

## Active Search
Instead of offloading search onto an external provider, DeploySolo uses [bleve](https://github.com/blevesearch/bleve) to index the local folder of markdown files, and exposes a `POST /search` endpoint which returns an HTML formatted list of links to search matches.This is used by the "search documentation" popup.

Without any effort from your part, your users can live search your markdown files, and navigate to matches.
