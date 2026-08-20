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

func (p *Provider) Refund(_ context.Context, req providers.RefundRequest) (*providers.RefundResult, error) {
	n := p.seq.Add(1)
	return &providers.RefundResult{
		ProviderRefundID: fmt.Sprintf("mock_rf_%d", n),
		Status:           "REFUNDED",
		Raw: map[string]any{
			"chargeId": req.ProviderChargeID,
			"amount":   req.Amount,
			"currency": req.Currency,
			"status":   "REFUNDED",
		},
	}, nil
}

func (p *Provider) SavePaymentMethod(_ context.Context, req providers.SaveMethodRequest) (*providers.SavedMethodResult, error) {
	n := p.seq.Add(1)
	token := fmt.Sprintf("mock_tok_%d", n)

	brand := "WALLET"
	last4 := ""
	label := "Phone wallet"
	if req.MethodType == "card" {
		brand = "CARD"
		digits := digitsOnly(req.CardNumber)
		if len(digits) >= 4 {
			last4 = digits[len(digits)-4:]
		} else {
			last4 = "0000"
		}
		label = "Card •••• " + last4
	} else {
		phone := digitsOnly(req.Customer.Phone.Number)
		if len(phone) >= 4 {
			last4 = phone[len(phone)-4:]
		}
		label = "Wallet •••• " + last4
	}

	return &providers.SavedMethodResult{
		ProviderToken: token,
		Label:         label,
		Brand:         brand,
		Last4:         last4,
		Raw: map[string]any{
			"token": token,
			"brand": brand,
			"last4": last4,
		},
	}, nil
}

func (p *Provider) OneTapCharge(_ context.Context, req providers.OneTapChargeRequest) (*providers.ChargeResult, error) {
	if req.ProviderToken == "" {
		return nil, fmt.Errorf("provider token is required")
	}
	n := p.seq.Add(1)
	chargeID := fmt.Sprintf("mock_ot_%d", n)
	return &providers.ChargeResult{
		ProviderChargeID: chargeID,
		RedirectURL:      "", // one-tap: no redirect
		Status:           "CAPTURED",
		Raw: map[string]any{
			"id":       chargeID,
			"token":    req.ProviderToken,
			"amount":   req.Amount,
			"currency": req.Currency,
			"status":   "CAPTURED",
			"mode":     "one_tap",
		},
	}, nil
}

func digitsOnly(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return string(out)
}

func trimTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
