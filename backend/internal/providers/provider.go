package providers

import (
	"context"
	"errors"
)

var ErrNotSupported = errors.New("operation not supported by provider")

// Provider is the app-facing payment gateway contract.
// Tap / mock / Chapa implement this so the public API stays stable.
type Provider interface {
	Name() string
	CreateCharge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
	Refund(ctx context.Context, req RefundRequest) (*RefundResult, error)
}

type ChargeRequest struct {
	PaymentID   string
	OrderID     string
	Amount      float64
	Currency    string
	Description string
	Customer    Customer
	Metadata    map[string]string
	RedirectURL string
	WebhookURL  string
}

type Customer struct {
	FirstName string
	LastName  string
	Email     string
	Phone     Phone
}

type Phone struct {
	CountryCode string
	Number      string
}

type ChargeResult struct {
	ProviderChargeID string
	RedirectURL      string
	Status           string
	Raw              any
}

type RefundRequest struct {
	ProviderChargeID string
	Amount           float64
	Currency         string
	Reason           string
}

type RefundResult struct {
	ProviderRefundID string
	Status           string
	Raw              any
}
