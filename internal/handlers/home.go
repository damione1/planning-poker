package handlers

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/damiengoehrig/planning-poker/internal/services"
	"github.com/damiengoehrig/planning-poker/web/templates"
)

func Home(re *core.RequestEvent) error {
	validator := services.NewVoteValidator()
	templateData := validator.GetAvailableTemplates()
	component := templates.Home(templateData)
	return templates.Render(re.Response, re.Request, component)
}
