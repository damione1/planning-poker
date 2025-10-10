package handlers

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/damiengoehrig/planning-poker/web/templates"
)

func Home(re *core.RequestEvent) error {
	component := templates.Home()
	return templates.Render(re.Response, re.Request, component)
}
