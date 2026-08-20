package services

import (
	"context"
	"path/filepath"
	"testing"

	"tap-payment/backend/internal/db"
	"tap-payment/backend/internal/providers/mock"
)

func TestOneTapPayWithSavedWallet(t *testing.T) {
	database, err := db.OpenAndMigrate(filepath.Join(t.TempDir(), "payments.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	svc := NewPaymentService(database, mock.NewProvider("http://localhost:8080"), PaymentServiceConfig{
		BaseURL: "http://localhost:8080",
	})

	method, err := svc.SavePaymentMethod(context.Background(), SaveMethodInput{
		MethodType: "wallet",
		Customer: CustomerInput{
			FirstName: "Abebe",
			Email:     "abebe@example.com",
			Phone:     PhoneInput{CountryCode: "251", Number: "911234567"},
		},
	})
	if err != nil {
		t.Fatalf("SavePaymentMethod: %v", err)
	}
	if method.ID == "" || method.Label == "" {
		t.Fatalf("unexpected method: %+v", method)
	}

	out, err := svc.OneTapPay(context.Background(), OneTapInput{
		OrderID:         "ord_ot_1",
		Amount:          75,
		Currency:        "ETB",
		PaymentMethodID: method.ID,
	})
	if err != nil {
		t.Fatalf("OneTapPay: %v", err)
	}
	if out.Status != "CAPTURED" {
		t.Fatalf("expected CAPTURED, got %q", out.Status)
	}
	if out.RedirectURL != "" {
		t.Fatalf("one-tap should not redirect, got %q", out.RedirectURL)
	}

	var orderStatus, paymentStatus string
	err = database.QueryRow(`
		SELECT o.status, p.status
		FROM payments p JOIN orders o ON o.id=p.order_id
		WHERE p.id=?
	`, out.PaymentID).Scan(&orderStatus, &paymentStatus)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if orderStatus != "paid" || paymentStatus != "captured" {
		t.Fatalf("unexpected statuses order=%q payment=%q", orderStatus, paymentStatus)
	}
}
