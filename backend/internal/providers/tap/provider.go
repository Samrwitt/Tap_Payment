package tap

import (
	"context"
	"fmt"

	"tap-payment/backend/internal/providers"
)

// Provider adapts the Tap HTTP client to the shared providers.Provider interface.
type Provider struct {
	client *Client
}

func NewProvider(secretKey string) *Provider {
	return &Provider{client: NewClient(secretKey)}
}

func (p *Provider) Name() string { return "tap" }

func (p *Provider) CreateCharge(ctx context.Context, req providers.ChargeRequest) (*providers.ChargeResult, error) {
	tapReq := CreateChargeRequest{
		Amount:            req.Amount,
		Currency:          req.Currency,
		CustomerInitiated: true,
		ThreeDSecure:      true,
		Description:       req.Description,
		Metadata:          req.Metadata,
		Customer: CreateChargeCustomer{
			FirstName: req.Customer.FirstName,
			LastName:  req.Customer.LastName,
			Email:     req.Customer.Email,
			Phone: PhoneNumber{
				CountryCode: req.Customer.Phone.CountryCode,
				Number:      req.Customer.Phone.Number,
			},
		},
		Source:   CreateChargeSource{ID: "src_all"},
		Redirect: CreateChargeRedirect{URL: req.RedirectURL},
		Post:     &CreateChargePost{URL: req.WebhookURL},
		Reference: map[string]string{
			"order": req.OrderID,
		},
	}

	resp, err := p.client.CreateCharge(ctx, tapReq)
	if err != nil {
		return nil, fmt.Errorf("tap create charge: %w", err)
	}

	return &providers.ChargeResult{
		ProviderChargeID: resp.ID,
		RedirectURL:      resp.Transaction.URL,
		Status:           resp.Status,
		Raw:              resp,
	}, nil
}

func (p *Provider) Refund(ctx context.Context, req providers.RefundRequest) (*providers.RefundResult, error) {
	return p.client.Refund(ctx, req)
}

func (p *Provider) SavePaymentMethod(_ context.Context, _ providers.SaveMethodRequest) (*providers.SavedMethodResult, error) {
	return nil, fmt.Errorf("%w: enable Tap save_card / payment agreements in a follow-up", providers.ErrNotSupported)
}

func (p *Provider) OneTapCharge(_ context.Context, _ providers.OneTapChargeRequest) (*providers.ChargeResult, error) {
	return nil, fmt.Errorf("%w: Tap one-tap requires a saved card token", providers.ErrNotSupported)
}
