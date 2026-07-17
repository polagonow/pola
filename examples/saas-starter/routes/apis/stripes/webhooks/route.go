package webhooks

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/polagonow/pola/core"
	stripe "github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"

	stripelib "saas-starter/lib/stripe"
	"saas-starter/repositories"
)

// Route serves POST /api/stripe/webhook — Stripe subscription lifecycle events.
// This path is CSRF-exempt (see Polafile csrf { exempt }); Stripe authenticates
// via the Stripe-Signature header instead.
type Route struct {
	teams repositories.TeamRepository
}

// NewRoute resolves the Route's dependencies from the DI registry.
func NewRoute(r *core.Registry) *Route {
	return &Route{teams: core.MustInvoke[repositories.TeamRepository](r)}
}

// Path pins the exact URL.
func (r *Route) Path() string { return "/api/stripe/webhook" }

// POST /api/stripe/webhook
func (r *Route) POST(c core.Context) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": "cannot read body"})
	}

	sig := c.Request().Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": "invalid signature"})
	}

	switch event.Type {
	case "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return c.JSON(http.StatusBadRequest, core.M{"error": "bad payload"})
		}
		if err := stripelib.HandleSubscriptionChange(c.Ctx(), &sub, r.teams); err != nil {
			return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, core.M{"received": true})
}
