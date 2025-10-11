package app

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/mannders00/deploysolo/utils"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterInvite(se *core.ServeEvent) error {
	g := se.Router.Group("/app")
	g.Bind(utils.RequirePayment())

	g.GET("/invite", utils.RenderView("invite.html", nil))
	g.POST("/invite", gitHubInviteHandler)

	return nil
}

// GitHubInviteHandler sends a one-time invitation for the GitHub repo
// to the username provided in the form in invite.html
// The repository, username, and PAT are specified as an environment variable
func gitHubInviteHandler(e *core.RequestEvent) error {

	// Load config
	cfg := e.App.Store().Get("cfg").(*utils.Config)

	// Ensure only one invitation per user
	if e.Auth.Get("invited") == true {
		return e.String(http.StatusOK, "Invitation has already been sent.")
	}

	username := e.Request.FormValue("username")
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", cfg.GHUsername, cfg.GHRepo, username)
	reqBody := []byte(`{"permission": "pull"}`)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+cfg.GHPAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return e.String(http.StatusInternalServerError, err.Error())
	}

	if resp.StatusCode == 201 {
		e.Auth.Set("invited", true)
		if err := e.App.Save(e.Auth); err != nil {
			return err
		}
		return e.String(http.StatusOK, fmt.Sprintf("Successfully invited %s as a collaborator!", username))
	} else {
		return e.String(http.StatusOK, string(body))
	}
}
