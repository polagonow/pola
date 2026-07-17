// Package stripe wraps the Stripe Go SDK for the SaaS starter: subscription
// checkout (with a 14-day trial), the customer billing portal, webhook-driven
// subscription sync, and price/product listing for the pricing page. It mirrors
// the Next.js starter's lib/payments/stripe.ts.
package stripe

import (
	"context"
	"errors"
	"fmt"
	"os"

	stripe "github.com/stripe/stripe-go/v79"
	bpsession "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/price"
	"github.com/stripe/stripe-go/v79/product"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

func setKey() { stripe.Key = os.Getenv("STRIPE_SECRET_KEY") }

func baseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:3000"
}

// CreateCheckoutSession starts a subscription checkout with a 14-day trial and
// returns the hosted Stripe Checkout URL to redirect the user to.
func CreateCheckoutSession(team *models.Team, userID uint, priceID string) (string, error) {
	setKey()
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
		},
		SuccessURL:        stripe.String(baseURL() + "/api/stripe/checkout?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:         stripe.String(baseURL() + "/pricing"),
		ClientReferenceID: stripe.String(fmt.Sprint(userID)),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(14),
		},
	}
	if team != nil && team.StripeCustomerID != nil && *team.StripeCustomerID != "" {
		params.Customer = stripe.String(*team.StripeCustomerID)
	}
	s, err := checkoutsession.New(params)
	if err != nil {
		return "", err
	}
	return s.URL, nil
}

// CreateCustomerPortalSession returns a Stripe billing-portal URL for the team's
// customer to manage or cancel their subscription.
func CreateCustomerPortalSession(team *models.Team) (string, error) {
	setKey()
	if team == nil || team.StripeCustomerID == nil || *team.StripeCustomerID == "" {
		return "", errors.New("team has no Stripe customer")
	}
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(*team.StripeCustomerID),
		ReturnURL: stripe.String(baseURL() + "/dashboard"),
	}
	s, err := bpsession.New(params)
	if err != nil {
		return "", err
	}
	return s.URL, nil
}

// HandleSubscriptionChange syncs a Stripe subscription onto the owning team:
// active/trialing subscriptions store the plan; canceled/unpaid ones clear it.
func HandleSubscriptionChange(ctx context.Context, sub *stripe.Subscription, teams repositories.TeamRepository) error {
	if sub == nil || sub.Customer == nil {
		return errors.New("subscription has no customer")
	}
	team, err := teams.GetByStripeCustomerID(ctx, sub.Customer.ID)
	if err != nil {
		return err
	}

	status := string(sub.Status)
	switch status {
	case "active", "trialing":
		subID := sub.ID
		var productID, planName string
		if len(sub.Items.Data) > 0 {
			item := sub.Items.Data[0]
			if item.Price != nil {
				if item.Price.Product != nil {
					productID = item.Price.Product.ID
				}
				planName = item.Price.Nickname
			}
		}
		if planName == "" {
			planName = productID
		}
		team.StripeSubscriptionID = &subID
		team.StripeProductID = &productID
		team.PlanName = &planName
		team.SubscriptionStatus = &status
	case "canceled", "unpaid":
		team.StripeSubscriptionID = nil
		team.StripeProductID = nil
		team.PlanName = nil
		team.SubscriptionStatus = &status
	default:
		team.SubscriptionStatus = &status
	}
	return teams.Update(ctx, team)
}

// PriceView is the pricing-page shape of a Stripe price.
type PriceView struct {
	ID         string `json:"id"`
	ProductID  string `json:"productId"`
	UnitAmount int64  `json:"unitAmount"`
	Currency   string `json:"currency"`
	Interval   string `json:"interval"`
}

// GetPrices returns active recurring prices for the pricing page.
func GetPrices() ([]PriceView, error) {
	setKey()
	params := &stripe.PriceListParams{Active: stripe.Bool(true)}
	params.AddExpand("data.product")
	it := price.List(params)
	out := []PriceView{}
	for it.Next() {
		p := it.Price()
		pv := PriceView{ID: p.ID, UnitAmount: p.UnitAmount, Currency: string(p.Currency)}
		if p.Product != nil {
			pv.ProductID = p.Product.ID
		}
		if p.Recurring != nil {
			pv.Interval = string(p.Recurring.Interval)
		}
		out = append(out, pv)
	}
	return out, it.Err()
}

// ProductView is the pricing-page shape of a Stripe product.
type ProductView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	DefaultPriceID string `json:"defaultPriceId"`
}

// GetProducts returns active products with their default price.
func GetProducts() ([]ProductView, error) {
	setKey()
	params := &stripe.ProductListParams{Active: stripe.Bool(true)}
	params.AddExpand("data.default_price")
	it := product.List(params)
	out := []ProductView{}
	for it.Next() {
		pr := it.Product()
		pv := ProductView{ID: pr.ID, Name: pr.Name, Description: pr.Description}
		if pr.DefaultPrice != nil {
			pv.DefaultPriceID = pr.DefaultPrice.ID
		}
		out = append(out, pv)
	}
	return out, it.Err()
}
