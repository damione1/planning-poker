package handlers

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/damione1/planning-poker/internal/services"
	"github.com/damione1/planning-poker/web/templates"
)

func Home(statsService *services.StatsService) func(*core.RequestEvent) error {
	return func(re *core.RequestEvent) error {
		validator := services.NewVoteValidator()
		templateData := validator.GetAvailableTemplates()

		// Get current statistics
		stats, err := statsService.GetCurrentStats()
		if err != nil {
			// Use empty stats if fetch fails (non-critical)
			stats = &services.StatsSnapshot{}
		}

		// Check for error query parameter
		errorParam := re.Request.URL.Query().Get("error")

		component := templates.Home(templateData, errorParam, stats)
		return templates.Render(re.Response, re.Request, component)
	}
}
