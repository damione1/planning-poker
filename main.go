package main

import (
	"html/template"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	// Your App Here
	"github.com/mannders00/deploysolo/app"
	_ "github.com/mannders00/deploysolo/migrations"
	"github.com/mannders00/deploysolo/utils"
)

func main() {

	pb := pocketbase.New()

	// load/store config
	cfg := utils.LoadConfig()
	pb.Store().Set("cfg", cfg)

	// load/store templates
	funcMap := template.FuncMap{"dict": utils.Dict}
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("web/templates/*/*")
	if err != nil {
		log.Fatal(err)
	}
	pb.Store().Set("tmpl", tmpl)

	// Add HTTP routes
	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Bind(utils.AuthCookieMiddleware())

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
