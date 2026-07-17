package checkouts

import (
	"net/http"
	"os"
	"strconv"

	"github.com/polagonow/pola/core"
	stripe "github.com/stripe/stripe-go/v79"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"

	stripelib "saas-starter/lib/stripe"
	"saas-starter/repositories"
)

// Route serves GET /api/stripe/checkout — the Stripe post-checkout success
// redirect. It records the customer, syncs the new subscription onto the team,
// and redirects to the dashboard.
type Route struct {
	teams repositories.TeamRepository
}

// NewRoute resolves the Route's dependencies from the DI registry.
func NewRoute(r *core.Registry) *Route {
	return &Route{teams: core.MustInvoke[repositories.TeamRepository](r)}
}

// Path pins the exact URL.
func (r *Route) Path() string { return "/api/stripe/checkout" }

// GET /api/stripe/checkout?session_id=...
func (r *Route) GET(c core.Context) error {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		return c.Redirect(http.StatusSeeOther, "/pricing")
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("subscription")
	params.AddExpand("subscription.items.data.price.product")
	params.AddExpand("customer")
	s, err := checkoutsession.Get(sessionID, params)
	if err != nil {
		return c.Redirect(http.StatusSeeOther, "/pricing?error=checkout")
	}

	if s.ClientReferenceID != "" {
		if uid, convErr := strconv.Atoi(s.ClientReferenceID); convErr == nil {
			if team, err := r.teams.GetForUser(c.Ctx(), uint(uid)); err == nil {
				if s.Customer != nil {
					cid := s.Customer.ID
					team.StripeCustomerID = &cid
					_ = r.teams.Update(c.Ctx(), team)
				}
				if s.Subscription != nil {
					_ = stripelib.HandleSubscriptionChange(c.Ctx(), s.Subscription, r.teams)
				}
			}
		}
	}

	return c.Redirect(http.StatusSeeOther, "/dashboard")
}
