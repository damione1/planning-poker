package utils

import (
	"io"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

func RegisterPayment(se *core.ServeEvent) {
	se.Router.GET("/checkout", checkoutHandler).Bind(LoginRedirect())
	se.Router.POST("/webhook", webhookHandler)
}

// CheckoutHandler creates a Stripe checkout session using a pre-configured Stripe product
// Also fetches the current user email, and attaches it to metadata. This is used later
// To find which user is associated with subsequent webhook requests.
func checkoutHandler(e *core.RequestEvent) error {
	cfg := e.App.Store().Get("cfg").(*Config)
	stripe.Key = cfg.StripeSecretKey

	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				Price:    stripe.String(cfg.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:         stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:   stripe.String(cfg.DomainName + "/login"),
		CancelURL:    stripe.String(cfg.DomainName + "/get"),
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)},
	}

	params.AddMetadata("email", e.Auth.Email())

	s, err := session.New(params)
	if err != nil {
		log.Printf("session.New: %v", err)
		return err
	}

	return e.Redirect(http.StatusSeeOther, s.URL)
}

// WebhookHandler is the handler that stripe's servers call. It contains the (pocketbase)
// email of the user as request metadata. This function activates the "paid" status of that email.
func webhookHandler(e *core.RequestEvent) error {
	cfg := e.App.Store().Get("cfg").(*Config)
	const maxBytes = int64(65536)
	limitedReader := http.MaxBytesReader(e.Response, e.Request.Body, maxBytes)

	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		return err
	}

	event, err := webhook.ConstructEvent(payload, e.Request.Header.Get("Stripe-Signature"), cfg.StripeWebhookSecret)
	if err != nil {
		return err
	}

	if event.Type == "checkout.session.completed" {
		// Successful payment. Retrieve user by email and update paid status in database.
		metadata, _ := event.Data.Object["metadata"].(map[string]interface{})
		email, _ := metadata["email"].(string)
		record, err := e.App.FindAuthRecordByEmail("users", email)
		if err != nil {
			return err
		}
		record.Set("paid", true)
		if err := e.App.Save(record); err != nil {
			return err
		}
	}

	return e.String(http.StatusOK, "OK")
}
