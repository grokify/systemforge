package stripe

import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

// VerifyWebhook verifies the Stripe webhook signature and returns the parsed event.
func VerifyWebhook(payload []byte, signature, secret string) (*stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, secret)
	if err != nil {
		return nil, fmt.Errorf("webhook verification failed: %w", err)
	}
	return &event, nil
}

// parseCheckoutSession extracts a checkout session from an event.
func parseCheckoutSession(event *stripe.Event) (*stripe.CheckoutSession, error) {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return nil, fmt.Errorf("failed to parse checkout session: %w", err)
	}
	return &sess, nil
}

// parseSubscription extracts a subscription from an event.
func parseSubscription(event *stripe.Event) (*stripe.Subscription, error) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return nil, fmt.Errorf("failed to parse subscription: %w", err)
	}
	return &sub, nil
}

// parseInvoice extracts an invoice from an event.
func parseInvoice(event *stripe.Event) (*stripe.Invoice, error) {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return nil, fmt.Errorf("failed to parse invoice: %w", err)
	}
	return &inv, nil
}

// parseAccount extracts a Connect account from an event.
func parseAccount(event *stripe.Event) (*stripe.Account, error) {
	var acct stripe.Account
	if err := json.Unmarshal(event.Data.Raw, &acct); err != nil {
		return nil, fmt.Errorf("failed to parse account: %w", err)
	}
	return &acct, nil
}

// WebhookEventTypes lists the Stripe event types we handle.
var WebhookEventTypes = []string{
	"checkout.session.completed",
	"customer.subscription.created",
	"customer.subscription.updated",
	"customer.subscription.deleted",
	"invoice.paid",
	"invoice.payment_failed",
	"account.updated", // Stripe Connect
}
