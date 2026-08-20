package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"tap-payment/backend/internal/providers"
)

type PaymentServiceConfig struct {
	BaseURL    string
	WebhookURL string
}

type PaymentService struct {
	db       *sql.DB
	provider providers.Provider
	cfg      PaymentServiceConfig
}

func NewPaymentService(db *sql.DB, provider providers.Provider, cfg PaymentServiceConfig) *PaymentService {
	return &PaymentService{db: db, provider: provider, cfg: cfg}
}

type CreateChargeInput struct {
	OrderID  string            `json:"orderId"`
	Amount   float64           `json:"amount"`
	Currency string            `json:"currency"`
	Customer CustomerInput     `json:"customer"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type CustomerInput struct {
	FirstName string     `json:"firstName,omitempty"`
	LastName  string     `json:"lastName,omitempty"`
	Email     string     `json:"email,omitempty"`
	Phone     PhoneInput `json:"phone"`
}

type PhoneInput struct {
	CountryCode string `json:"countryCode"`
	Number      string `json:"number"`
}

type CreateChargeOutput struct {
	PaymentID        string `json:"paymentId"`
	Provider         string `json:"provider"`
	ProviderChargeID string `json:"providerChargeId"`
	RedirectURL      string `json:"redirectUrl"`
	Status           string `json:"status"`
}

var (
	ErrInvalidInput = errors.New("invalid input")
)

var isoCurrencyCodeRe = regexp.MustCompile(`^[A-Z]{3}$`)
var countryCodeDigitsRe = regexp.MustCompile(`^[0-9]{1,3}$`)
var phoneNumberDigitsRe = regexp.MustCompile(`^[0-9]{6,15}$`)

func (s *PaymentService) CreateCharge(ctx context.Context, in CreateChargeInput) (*CreateChargeOutput, error) {
	if err := validateCreateChargeInput(&in); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	paymentID := "pay_" + randID()
	providerName := s.provider.Name()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, amount, currency, status, created_at, updated_at)
		VALUES (?, ?, ?, 'pending', ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			amount=excluded.amount,
			currency=excluded.currency,
			updated_at=excluded.updated_at
	`, in.OrderID, in.Amount, in.Currency, now, now)
	if err != nil {
		return nil, fmt.Errorf("upsert order: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (id, order_id, provider, status, created_at, updated_at)
		VALUES (?, ?, ?, 'initiated', ?, ?)
	`, paymentID, in.OrderID, providerName, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert payment: %w", err)
	}

	result, err := s.provider.CreateCharge(ctx, providers.ChargeRequest{
		PaymentID:   paymentID,
		OrderID:     in.OrderID,
		Amount:      in.Amount,
		Currency:    in.Currency,
		Description: "Order " + in.OrderID,
		Metadata:    mergeMetadata(in.Metadata, map[string]string{"orderId": in.OrderID, "paymentId": paymentID}),
		Customer: providers.Customer{
			FirstName: in.Customer.FirstName,
			LastName:  in.Customer.LastName,
			Email:     in.Customer.Email,
			Phone: providers.Phone{
				CountryCode: in.Customer.Phone.CountryCode,
				Number:      in.Customer.Phone.Number,
			},
		},
		RedirectURL: s.cfg.BaseURL + "/payment/return",
		WebhookURL:  s.cfg.WebhookURL,
	})
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(result.Raw)
	_, err = tx.ExecContext(ctx, `
		UPDATE payments
		SET provider_payment_id=?, redirect_url=?, raw_last_event=?, updated_at=?
		WHERE id=?
	`, result.ProviderChargeID, result.RedirectURL, string(raw), now, paymentID)
	if err != nil {
		return nil, fmt.Errorf("update payment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &CreateChargeOutput{
		PaymentID:        paymentID,
		Provider:         providerName,
		ProviderChargeID: result.ProviderChargeID,
		RedirectURL:      result.RedirectURL,
		Status:           result.Status,
	}, nil
}

func validateCreateChargeInput(in *CreateChargeInput) error {
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.Customer.Email = strings.TrimSpace(in.Customer.Email)
	in.Customer.FirstName = strings.TrimSpace(in.Customer.FirstName)
	in.Customer.LastName = strings.TrimSpace(in.Customer.LastName)
	in.Customer.Phone.CountryCode = strings.TrimPrefix(strings.TrimSpace(in.Customer.Phone.CountryCode), "+")
	in.Customer.Phone.Number = strings.TrimSpace(in.Customer.Phone.Number)

	switch {
	case in.OrderID == "":
		return fmt.Errorf("%w: orderId is required", ErrInvalidInput)
	case in.Amount <= 0:
		return fmt.Errorf("%w: amount must be > 0", ErrInvalidInput)
	case in.Currency == "":
		return fmt.Errorf("%w: currency is required", ErrInvalidInput)
	case !isoCurrencyCodeRe.MatchString(in.Currency):
		return fmt.Errorf("%w: currency must be a 3-letter ISO code", ErrInvalidInput)
	case in.Customer.Phone.CountryCode == "":
		return fmt.Errorf("%w: customer.phone.countryCode is required", ErrInvalidInput)
	case !countryCodeDigitsRe.MatchString(in.Customer.Phone.CountryCode):
		return fmt.Errorf("%w: customer.phone.countryCode must be digits only", ErrInvalidInput)
	case in.Customer.Phone.Number == "":
		return fmt.Errorf("%w: customer.phone.number is required", ErrInvalidInput)
	case !phoneNumberDigitsRe.MatchString(in.Customer.Phone.Number):
		return fmt.Errorf("%w: customer.phone.number must be digits only and 6-15 chars", ErrInvalidInput)
	}

	return nil
}

func mergeMetadata(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
