package mock

import (
	"context"
	"strings"
	"testing"

	"tap-payment/backend/internal/providers"
)

func TestMockCreateChargeReturnsRedirect(t *testing.T) {
	p := NewProvider("http://localhost:8080")
	res, err := p.CreateCharge(context.Background(), providers.ChargeRequest{
		PaymentID: "pay_1",
		OrderID:   "ord_1",
		Amount:    25,
		Currency:  "ETB",
		Customer: providers.Customer{
			Phone: providers.Phone{CountryCode: "251", Number: "911000000"},
		},
	})
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if p.Name() != "mock" {
		t.Fatalf("expected provider name mock, got %q", p.Name())
	}
	if !strings.HasPrefix(res.ProviderChargeID, "mock_chg_") {
		t.Fatalf("unexpected charge id %q", res.ProviderChargeID)
	}
	if !strings.Contains(res.RedirectURL, "/mock/checkout/") {
		t.Fatalf("unexpected redirect url %q", res.RedirectURL)
	}
	if res.Status != "INITIATED" {
		t.Fatalf("expected INITIATED, got %q", res.Status)
	}
}
