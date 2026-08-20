package mock

import (
	"context"
	"fmt"
	"sync/atomic"

	"tap-payment/backend/internal/providers"
)

// Provider is a local/dev payment provider that never calls an external gateway.
// Useful for demos, CI, and Ethiopia-ready local development without Tap keys.
type Provider struct {
	baseURL string
	seq     atomic.Uint64
}

func NewProvider(baseURL string) *Provider {
	return &Provider{baseURL: baseURL}
}

func (p *Provider) Name() string { return "mock" }

func (p *Provider) CreateCharge(_ context.Context, req providers.ChargeRequest) (*providers.ChargeResult, error) {
	n := p.seq.Add(1)
	chargeID := fmt.Sprintf("mock_chg_%d", n)
	redirect := fmt.Sprintf("%s/mock/checkout/%s", trimTrailingSlash(p.baseURL), chargeID)

	return &providers.ChargeResult{
		ProviderChargeID: chargeID,
		RedirectURL:      redirect,
		Status:           "INITIATED",
		Raw: map[string]any{
			"id":       chargeID,
			"orderId":  req.OrderID,
			"amount":   req.Amount,
			"currency": req.Currency,
			"status":   "INITIATED",
		},
	}, nil
}

func trimTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
