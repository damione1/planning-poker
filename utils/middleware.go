package utils

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// AuthCookieMiddleware is an HTTP middleware that takes the Pocketbase auth token from the `pb_auth` cookie
// It then manually retrieves the auth state from this token, and places it in the echo context, accessible by HTTP handlers.
func AuthCookieMiddleware() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:   "AuthCookieMiddleware",
		Func: authCookie,
	}
}

func authCookie(e *core.RequestEvent) error {
	if e.Auth != nil {
		return e.Next()
	}

	tokenCookie, err := e.Request.Cookie("pb_auth")
	if err != nil {
		return e.Next()
	}

	decodedCookie, err := url.QueryUnescape(tokenCookie.Value)
	if err != nil {
		return e.Next()
	}

	var cookieObject map[string]interface{}
	err = json.Unmarshal([]byte(decodedCookie), &cookieObject)
	if err != nil {
		return e.Next()
	}

	token := cookieObject["token"].(string)

	m, err := e.App.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return e.Next()
	}

	e.Auth = m
	return e.Next()
}

// LoginRedirect moves to default /auth/login view if auth token isn't set
func LoginRedirect() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:   "LoginRedirect",
		Func: checkLogin,
	}
}

func checkLogin(e *core.RequestEvent) error {
	if e.Auth == nil {
		return e.Redirect(http.StatusSeeOther, "/auth/login")
	}

	return e.Next()
}

// RequirePayment is an HTTP middleware that takes the Pocketbase auth token from the `pb_auth` cookie
// It then manually retrieves the auth state from this token, and places it in the echo context, accessible by HTTP handlers.
func RequirePayment() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:   "PaidGuard",
		Func: paidGuard,
	}
}

func paidGuard(e *core.RequestEvent) error {
	if e.Auth == nil {
		return e.Redirect(http.StatusSeeOther, "/get")
	}

	if e.Auth.Get("paid").(bool) != true {
		return e.Redirect(http.StatusSeeOther, "/get")
	}

	return e.Next()
}
