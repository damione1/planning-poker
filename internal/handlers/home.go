package handlers

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/damione1/planning-poker-new/internal/services"
	"github.com/damione1/planning-poker-new/web/templates"
)

func Home(re *core.RequestEvent) error {
	validator := services.NewVoteValidator()
	templateData := validator.GetAvailableTemplates()

	// Check for error query parameter
	errorParam := re.Request.URL.Query().Get("error")

	component := templates.Home(templateData, errorParam)
	return templates.Render(re.Response, re.Request, component)
}
