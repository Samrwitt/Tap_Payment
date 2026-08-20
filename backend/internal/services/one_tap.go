package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tap-payment/backend/internal/providers"
)

type SaveMethodInput struct {
	CustomerKey string        `json:"customerKey"`
	MethodType  string        `json:"methodType"` // wallet | card
	CardNumber  string        `json:"cardNumber,omitempty"`
	Customer    CustomerInput `json:"customer"`
}

type PaymentMethodOutput struct {
	ID          string `json:"id"`
	CustomerKey string `json:"customerKey"`
	Provider    string `json:"provider"`
	Label       string `json:"label"`
	Brand       string `json:"brand,omitempty"`
	Last4       string `json:"last4,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

type OneTapInput struct {
	OrderID         string            `json:"orderId"`
	Amount          float64           `json:"amount"`
	Currency        string            `json:"currency"`
	PaymentMethodID string            `json:"paymentMethodId"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

func (s *PaymentService) SavePaymentMethod(ctx context.Context, in SaveMethodInput) (*PaymentMethodOutput, error) {
	in.CustomerKey = strings.TrimSpace(in.CustomerKey)
	in.MethodType = strings.ToLower(strings.TrimSpace(in.MethodType))
	if in.MethodType == "" {
		in.MethodType = "wallet"
	}
	if in.MethodType != "wallet" && in.MethodType != "card" {
		return nil, fmt.Errorf("%w: methodType must be wallet or card", ErrInvalidInput)
	}
	if err := validateCreateChargeInput(&CreateChargeInput{
		OrderID:  "setup",
		Amount:   1,
		Currency: "ETB",
		Customer: in.Customer,
	}); err != nil {
		// reuse phone validation; strip order-specific noise
		return nil, err
	}
	if in.CustomerKey == "" {
		in.CustomerKey = customerKeyFromPhone(in.Customer.Phone)
	}

	saved, err := s.provider.SavePaymentMethod(ctx, providers.SaveMethodRequest{
		CustomerKey: in.CustomerKey,
		Customer: providers.Customer{
			FirstName: in.Customer.FirstName,
			LastName:  in.Customer.LastName,
			Email:     in.Customer.Email,
			Phone: providers.Phone{
				CountryCode: in.Customer.Phone.CountryCode,
				Number:      in.Customer.Phone.Number,
			},
		},
		MethodType: in.MethodType,
		CardNumber: in.CardNumber,
	})
	if err != nil {
		if errors.Is(err, providers.ErrNotSupported) {
			return nil, fmt.Errorf("%w: %v", ErrProviderUnsupported, err)
		}
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := "pm_" + randID()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO payment_methods (id, customer_key, provider, provider_token, label, brand, last4, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, in.CustomerKey, s.provider.Name(), saved.ProviderToken, saved.Label, saved.Brand, saved.Last4, now, now)
	if err != nil {
		return nil, err
	}

	return &PaymentMethodOutput{
		ID:          id,
		CustomerKey: in.CustomerKey,
		Provider:    s.provider.Name(),
		Label:       saved.Label,
		Brand:       saved.Brand,
		Last4:       saved.Last4,
		CreatedAt:   now,
	}, nil
}

func (s *PaymentService) ListPaymentMethods(ctx context.Context, customerKey string) ([]PaymentMethodOutput, error) {
	customerKey = strings.TrimSpace(customerKey)
	if customerKey == "" {
		return nil, fmt.Errorf("%w: customerKey is required", ErrInvalidInput)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, customer_key, provider, label, brand, last4, created_at
		FROM payment_methods
		WHERE customer_key=?
		ORDER BY created_at DESC
	`, customerKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []PaymentMethodOutput{}
	for rows.Next() {
		var m PaymentMethodOutput
		if err := rows.Scan(&m.ID, &m.CustomerKey, &m.Provider, &m.Label, &m.Brand, &m.Last4, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *PaymentService) OneTapPay(ctx context.Context, in OneTapInput) (*CreateChargeOutput, error) {
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.PaymentMethodID = strings.TrimSpace(in.PaymentMethodID)

	switch {
	case in.OrderID == "":
		return nil, fmt.Errorf("%w: orderId is required", ErrInvalidInput)
	case in.Amount <= 0:
		return nil, fmt.Errorf("%w: amount must be > 0", ErrInvalidInput)
	case !isoCurrencyCodeRe.MatchString(in.Currency):
		return nil, fmt.Errorf("%w: currency must be a 3-letter ISO code", ErrInvalidInput)
	case in.PaymentMethodID == "":
		return nil, fmt.Errorf("%w: paymentMethodId is required", ErrInvalidInput)
	}

	var customerKey, providerName, token string
	err := s.db.QueryRowContext(ctx, `
		SELECT customer_key, provider, provider_token FROM payment_methods WHERE id=?
	`, in.PaymentMethodID).Scan(&customerKey, &providerName, &token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if providerName != s.provider.Name() {
		return nil, fmt.Errorf("%w: payment method belongs to a different provider", ErrInvalidState)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	paymentID := "pay_" + randID()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, amount, currency, status, created_at, updated_at)
		VALUES (?, ?, ?, 'pending', ?, ?)
		ON CONFLICT(id) DO UPDATE SET amount=excluded.amount, currency=excluded.currency, updated_at=excluded.updated_at
	`, in.OrderID, in.Amount, in.Currency, now, now)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (id, order_id, provider, status, created_at, updated_at)
		VALUES (?, ?, ?, 'initiated', ?, ?)
	`, paymentID, in.OrderID, providerName, now, now)
	if err != nil {
		return nil, err
	}

	result, err := s.provider.OneTapCharge(ctx, providers.OneTapChargeRequest{
		PaymentID:     paymentID,
		OrderID:       in.OrderID,
		Amount:        in.Amount,
		Currency:      in.Currency,
		ProviderToken: token,
		CustomerKey:   customerKey,
		Metadata:      mergeMetadata(in.Metadata, map[string]string{"paymentMethodId": in.PaymentMethodID, "mode": "one_tap"}),
	})
	if err != nil {
		if errors.Is(err, providers.ErrNotSupported) {
			return nil, fmt.Errorf("%w: %v", ErrProviderUnsupported, err)
		}
		return nil, err
	}

	status := strings.ToLower(result.Status)
	raw, _ := json.Marshal(result.Raw)
	_, err = tx.ExecContext(ctx, `
		UPDATE payments
		SET provider_payment_id=?, redirect_url=?, status=?, raw_last_event=?, updated_at=?
		WHERE id=?
	`, result.ProviderChargeID, result.RedirectURL, status, string(raw), now, paymentID)
	if err != nil {
		return nil, err
	}

	if status == "captured" || status == "success" || status == "successful" || status == "paid" {
		_, err = tx.ExecContext(ctx, `UPDATE orders SET status='paid', updated_at=? WHERE id=?`, now, in.OrderID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &CreateChargeOutput{
		PaymentID:        paymentID,
		Provider:         providerName,
		ProviderChargeID: result.ProviderChargeID,
		RedirectURL:      result.RedirectURL,
		Status:           result.Status,
	}, nil
}

func customerKeyFromPhone(p PhoneInput) string {
	return "+" + p.CountryCode + p.Number
}
