package providers

import (
	"context"
	"errors"
)

var ErrNotSupported = errors.New("operation not supported by provider")

// Provider is the app-facing payment gateway contract.
type Provider interface {
	Name() string
	CreateCharge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
	Refund(ctx context.Context, req RefundRequest) (*RefundResult, error)
	SavePaymentMethod(ctx context.Context, req SaveMethodRequest) (*SavedMethodResult, error)
	OneTapCharge(ctx context.Context, req OneTapChargeRequest) (*ChargeResult, error)
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

// SaveMethodRequest enrolls a reusable payment method for later one-tap charges.
type SaveMethodRequest struct {
	CustomerKey string
	Customer    Customer
	// MethodType: "wallet" (phone/Telebirr-style) or "card"
	MethodType string
	// CardNumber is only used by mock for last4 display; never store full PAN.
	CardNumber string
}

type SavedMethodResult struct {
	ProviderToken string
	Label         string
	Brand         string
	Last4         string
	Raw           any
}

type OneTapChargeRequest struct {
	PaymentID     string
	OrderID       string
	Amount        float64
	Currency      string
	ProviderToken string
	CustomerKey   string
	Metadata      map[string]string
}
