package services

import (
	"context"
	"path/filepath"
	"testing"

	"tap-payment/backend/internal/db"
	"tap-payment/backend/internal/providers/mock"
)

func TestRefundWithMockProvider(t *testing.T) {
	database, err := db.OpenAndMigrate(filepath.Join(t.TempDir(), "payments.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	svc := NewPaymentService(database, mock.NewProvider("http://localhost:8080"), PaymentServiceConfig{
		BaseURL:    "http://localhost:8080",
		WebhookURL: "http://localhost:8080/webhooks",
	})

	created, err := svc.CreateCharge(context.Background(), CreateChargeInput{
		OrderID:  "ord_rf_1",
		Amount:   50,
		Currency: "ETB",
		Customer: CustomerInput{
			Phone: PhoneInput{CountryCode: "251", Number: "911234567"},
		},
	})
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}

	if err := svc.CompleteMockPayment(context.Background(), created.ProviderChargeID); err != nil {
		t.Fatalf("CompleteMockPayment: %v", err)
	}

	out, err := svc.RefundPayment(context.Background(), created.PaymentID, RefundInput{Reason: "customer request"})
	if err != nil {
		t.Fatalf("RefundPayment: %v", err)
	}
	if out.Status != "REFUNDED" {
		t.Fatalf("expected REFUNDED, got %q", out.Status)
	}

	var status string
	if err := database.QueryRow(`SELECT status FROM payments WHERE id=?`, created.PaymentID).Scan(&status); err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "refunded" {
		t.Fatalf("expected refunded payment, got %q", status)
	}
}
