package services

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"tap-payment/backend/internal/db"
	"tap-payment/backend/internal/providers/mock"
)

func TestCreateChargeWithMockProvider(t *testing.T) {
	database, err := db.OpenAndMigrate(filepath.Join(t.TempDir(), "payments.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	svc := NewPaymentService(database, mock.NewProvider("http://localhost:8080"), PaymentServiceConfig{
		BaseURL:    "http://localhost:8080",
		WebhookURL: "http://localhost:8080/api/payments/webhooks/tap",
	})

	out, err := svc.CreateCharge(context.Background(), CreateChargeInput{
		OrderID:  "ord_et_1",
		Amount:   100,
		Currency: "ETB",
		Customer: CustomerInput{
			FirstName: "Abebe",
			LastName:  "Kebede",
			Email:     "abebe@example.com",
			Phone: PhoneInput{
				CountryCode: "251",
				Number:      "911234567",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}

	if out.Provider != "mock" {
		t.Fatalf("expected mock provider, got %q", out.Provider)
	}
	if !strings.HasPrefix(out.PaymentID, "pay_") {
		t.Fatalf("unexpected payment id %q", out.PaymentID)
	}
	if !strings.Contains(out.RedirectURL, "/mock/checkout/") {
		t.Fatalf("unexpected redirect url %q", out.RedirectURL)
	}

	var status, provider string
	err = database.QueryRow(`SELECT status, provider FROM payments WHERE id=?`, out.PaymentID).Scan(&status, &provider)
	if err != nil {
		t.Fatalf("query payment: %v", err)
	}
	if status != "initiated" || provider != "mock" {
		t.Fatalf("unexpected payment row status=%q provider=%q", status, provider)
	}
}
