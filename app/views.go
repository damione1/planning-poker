package app

import (
	"github.com/mannders00/deploysolo/utils"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterViews(se *core.ServeEvent) error {
	se.Router.GET("/", utils.RenderView("index.html", nil))
	se.Router.GET("/app/profile", utils.RenderView("profile.html", nil)).Bind(apis.RequireAuth())
	se.Router.GET("/get", utils.RenderView("get.html", nil))

	se.Router.GET("/auth/login", utils.RenderView("login.html", nil))
	se.Router.GET("/auth/register", utils.RenderView("register.html", nil))
	se.Router.GET("/auth/reset", utils.RenderView("reset.html", nil))

	return nil
}
