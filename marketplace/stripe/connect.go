package stripe

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/account"
	"github.com/stripe/stripe-go/v84/accountlink"
	"github.com/stripe/stripe-go/v84/payout"
)

// ConnectService provides Stripe Connect functionality for marketplace sellers.
type ConnectService struct {
	config Config
}

// NewConnectService creates a new Stripe Connect service.
func NewConnectService(cfg Config) *ConnectService {
	stripe.Key = cfg.SecretKey
	return &ConnectService{
		config: cfg,
	}
}

// ConnectAccount represents a Stripe Connect account for a seller.
type ConnectAccount struct {
	// ID is the Stripe account ID.
	ID string

	// OrganizationID is the platform organization ID.
	OrganizationID uuid.UUID

	// Email is the account email.
	Email string

	// ChargesEnabled indicates if the account can accept charges.
	ChargesEnabled bool

	// PayoutsEnabled indicates if the account can receive payouts.
	PayoutsEnabled bool

	// DetailsSubmitted indicates if onboarding is complete.
	DetailsSubmitted bool
}

// CreateConnectAccount creates a new Stripe Connect Express account for a seller.
func (s *ConnectService) CreateConnectAccount(ctx context.Context, orgID uuid.UUID, email string) (*ConnectAccount, error) {
	params := &stripe.AccountParams{
		Type:  stripe.String(string(stripe.AccountTypeExpress)),
		Email: stripe.String(email),
		Metadata: map[string]string{
			"organization_id": orgID.String(),
		},
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
	}

	acct, err := account.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Connect account: %w", err)
	}

	return &ConnectAccount{
		ID:               acct.ID,
		OrganizationID:   orgID,
		Email:            acct.Email,
		ChargesEnabled:   acct.ChargesEnabled,
		PayoutsEnabled:   acct.PayoutsEnabled,
		DetailsSubmitted: acct.DetailsSubmitted,
	}, nil
}

// CreateAccountLink creates an onboarding link for a Connect account.
func (s *ConnectService) CreateAccountLink(ctx context.Context, accountID, refreshURL, returnURL string) (string, error) {
	params := &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		RefreshURL: stripe.String(refreshURL),
		ReturnURL:  stripe.String(returnURL),
		Type:       stripe.String("account_onboarding"),
	}

	link, err := accountlink.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create account link: %w", err)
	}

	return link.URL, nil
}

// GetConnectAccount retrieves a Connect account by ID.
func (s *ConnectService) GetConnectAccount(ctx context.Context, accountID string) (*ConnectAccount, error) {
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Connect account: %w", err)
	}

	var orgID uuid.UUID
	if orgIDStr, ok := acct.Metadata["organization_id"]; ok {
		orgID, _ = uuid.Parse(orgIDStr)
	}

	return &ConnectAccount{
		ID:               acct.ID,
		OrganizationID:   orgID,
		Email:            acct.Email,
		ChargesEnabled:   acct.ChargesEnabled,
		PayoutsEnabled:   acct.PayoutsEnabled,
		DetailsSubmitted: acct.DetailsSubmitted,
	}, nil
}

// CreatePayout initiates a payout to a Connect account.
func (s *ConnectService) CreatePayout(ctx context.Context, accountID string, amountCents int64, currency string) (string, error) {
	params := &stripe.PayoutParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(currency),
	}
	params.SetStripeAccount(accountID)

	p, err := payout.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create payout: %w", err)
	}

	return p.ID, nil
}

// GetAccountBalance retrieves the available balance for a Connect account.
func (s *ConnectService) GetAccountBalance(ctx context.Context, accountID string) (int64, string, error) {
	// Balance retrieval requires using the Balance API
	// For now, return a placeholder
	return 0, "usd", nil
}
