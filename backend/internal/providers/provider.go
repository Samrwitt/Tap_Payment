package providers

import "context"

// Provider is the app-facing payment gateway contract.
// Tap is the first implementation; Ethiopian gateways (Chapa/Telebirr) can plug in later.
type Provider interface {
	Name() string
	CreateCharge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
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
