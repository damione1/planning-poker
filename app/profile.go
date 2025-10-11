package app

import (
	"html/template"

	"github.com/mannders00/deploysolo/utils"
	"github.com/pocketbase/pocketbase/core"
)

func RenderProfile(tmpl *template.Template) func(e *core.RequestEvent) error {
	return utils.RenderView("profile.html", nil)
}
