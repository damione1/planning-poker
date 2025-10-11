# 🧱 App Structure
At its core, DeploySolo uses `net/http` and `html/template` to provide a [hypermedia](https://hypermedia.systems/) web app over HTTP.

Consider this snippet from `main.go`, conveying this idea.

```go
func main() {
  pb := pocketbase.New()
    ...
  pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
  	app.RegisterTasks(se)
  	app.RegisterViews(se)
  	app.RegisterChat(se)
  	app.RegisterInvite(se)

  	utils.RegisterSearch(se, "web/docs")
  	utils.RegisterPayment(se)

  	se.Router.GET("/docs/{doc...}", utils.RenderDocViewHandler(tmpl))
  	se.Router.GET("/public/{path...}", apis.Static(os.DirFS("web/public"), false))

  	return se.Next()
    })

  if err := pb.Start(); err != nil {
  	log.Fatal(err)
  }
}
```

Let's take a closer look at the `RegisterTasks`, which implements the tasks app.

`tasks.go`
```go
package app
func RegisterTasks(se *core.ServeEvent) error {
  g := se.Router.Group("/app")
  g.Bind(utils.RequirePayment())

  g.GET("/tasks", renderTasks)
  g.POST("/tasks", createTask)
  g.GET("/tasks/{id}", editTask)
  g.PUT("/tasks/{id}", saveTask)
  g.DELETE("/tasks/{id}", deleteTask)

  return nil
}

func renderTasks(e *core.RequestEvent) error { ... }
func createTask(e *core.RequestEvent) error { ... }
func editTask(e *core.RequestEvent) error { ... }
func saveTask(e *core.RequestEvent) error { ... }
func deleteTask(e *core.RequestEvent) error { ... }
```

Here we can see the tasks app, from rendering HTML to the browser, to taking form POSTs and updating the database, is all driven by HTTP APIs.

The core idea here is that one could create their app with this method, then add those routes to the shared `mux`, much like the reference tasks app.

## Source Tree
```sh
.
├── app
│   ├── chat.go
│   ├── invite.go
│   ├── profile.go
│   ├── tasks.go
│   └── views.go
├── go.mod
├── go.sum
├── main.go
├── migrations
├── utils
│   ├── config.go
│   ├── middleware.go
│   ├── payment.go
│   ├── render.go
│   └── search.go
└── web
    ├── content
    ├── docs
    ├── public
    └── templates
```

## App
Go files containing app specific code.

## Utils
Go files containing helper functions
