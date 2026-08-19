package services

import (
	"errors"
	"testing"
)

func TestValidateCreateChargeInputNormalizesValues(t *testing.T) {
	in := CreateChargeInput{
		OrderID:  "  ord_123  ",
		Amount:   10,
		Currency: " etb ",
		Customer: CustomerInput{
			FirstName: "  Sam ",
			LastName:  "  W ",
			Email:     " user@example.com ",
			Phone: PhoneInput{
				CountryCode: "+251",
				Number:      "912345678",
			},
		},
	}

	if err := validateCreateChargeInput(&in); err != nil {
		t.Fatalf("expected validation to pass: %v", err)
	}

	if in.OrderID != "ord_123" {
		t.Fatalf("expected trimmed orderId, got %q", in.OrderID)
	}
	if in.Currency != "ETB" {
		t.Fatalf("expected uppercase currency, got %q", in.Currency)
	}
	if in.Customer.Phone.CountryCode != "251" {
		t.Fatalf("expected normalized country code, got %q", in.Customer.Phone.CountryCode)
	}
	if in.Customer.Email != "user@example.com" {
		t.Fatalf("expected trimmed email, got %q", in.Customer.Email)
	}
}

func TestValidateCreateChargeInputRejectsBadCountryCode(t *testing.T) {
	in := CreateChargeInput{
		OrderID:  "ord_123",
		Amount:   10,
		Currency: "ETB",
		Customer: CustomerInput{
			Phone: PhoneInput{
				CountryCode: "251a",
				Number:      "912345678",
			},
		},
	}

	err := validateCreateChargeInput(&in)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

