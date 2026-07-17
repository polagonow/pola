package actions

import (
	"context"

	"github.com/polagonow/pola/core"

	stripelib "saas-starter/lib/stripe"
	"saas-starter/repositories"
)

// PaymentsAction exposes Stripe checkout, the customer portal, and pricing data.
// Bridged to React as `PaymentsAction`.
type PaymentsAction struct {
	teams repositories.TeamRepository
}

// NewPaymentsAction resolves the PaymentsAction's dependencies from the DI registry.
func NewPaymentsAction(r *core.Registry) *PaymentsAction {
	return &PaymentsAction{
		teams: core.MustInvoke[repositories.TeamRepository](r),
	}
}

// Checkout starts a Stripe subscription checkout for the caller's team and
// returns the hosted-checkout URL to redirect to.
func (p *PaymentsAction) Checkout(ctx context.Context, priceID string) (*ActionResult, error) {
	uid, ok := sessionUserID(ctx)
	if !ok {
		return &ActionResult{Redirect: "/sign-in"}, nil
	}
	team, err := p.teams.GetForUser(ctx, uid)
	if err != nil {
		return &ActionResult{Error: "No team found for the current user."}, nil
	}
	url, err := stripelib.CreateCheckoutSession(team, uid, priceID)
	if err != nil {
		return &ActionResult{Error: err.Error()}, nil
	}
	return &ActionResult{Redirect: url}, nil
}

// CustomerPortal returns a Stripe billing-portal URL for the caller's team.
func (p *PaymentsAction) CustomerPortal(ctx context.Context) (*ActionResult, error) {
	uid, ok := sessionUserID(ctx)
	if !ok {
		return &ActionResult{Redirect: "/sign-in"}, nil
	}
	team, err := p.teams.GetForUser(ctx, uid)
	if err != nil {
		return &ActionResult{Error: "No team found for the current user."}, nil
	}
	url, err := stripelib.CreateCustomerPortalSession(team)
	if err != nil {
		return &ActionResult{Error: err.Error()}, nil
	}
	return &ActionResult{Redirect: url}, nil
}

// Prices returns active recurring prices for the pricing page.
func (p *PaymentsAction) Prices(ctx context.Context) ([]stripelib.PriceView, error) {
	return stripelib.GetPrices()
}

// Products returns active products for the pricing page.
func (p *PaymentsAction) Products(ctx context.Context) ([]stripelib.ProductView, error) {
	return stripelib.GetProducts()
}
