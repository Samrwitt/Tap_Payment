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

var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidState      = errors.New("invalid payment state")
	ErrProviderUnsupported = errors.New("provider operation unsupported")
)

type WebhookUpdate struct {
	Provider  string
	EventKey  string
	Status    string
	MarkPaid  bool
	RawJSON   string
}

func (s *PaymentService) ApplyWebhook(ctx context.Context, in WebhookUpdate) (duplicate bool, err error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	eventID := "wev_" + randID()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO webhook_events (id, provider, event_key, received_at)
		VALUES (?, ?, ?, ?)
	`, eventID, in.Provider, in.EventKey, now)
	if err != nil {
		if isUniqueConstraintError(err) {
			return true, nil
		}
		return false, fmt.Errorf("persist webhook: %w", err)
	}

	status := strings.ToLower(in.Status)
	_, err = s.db.ExecContext(ctx, `
		UPDATE payments SET status=?, raw_last_event=?, updated_at=?
		WHERE provider=? AND provider_payment_id=?
	`, status, in.RawJSON, now, in.Provider, in.EventKey)
	if err != nil {
		return false, fmt.Errorf("update payment: %w", err)
	}

	if in.MarkPaid {
		_, err = s.db.ExecContext(ctx, `
			UPDATE orders SET status='paid', updated_at=?
			WHERE id IN (SELECT order_id FROM payments WHERE provider=? AND provider_payment_id=?)
		`, now, in.Provider, in.EventKey)
		if err != nil {
			return false, fmt.Errorf("update order: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE webhook_events SET processed_at=? WHERE provider=? AND event_key=?
	`, now, in.Provider, in.EventKey)
	if err != nil {
		return false, fmt.Errorf("mark webhook processed: %w", err)
	}
	return false, nil
}

type RefundInput struct {
	Amount float64 `json:"amount,omitempty"`
	Reason string  `json:"reason,omitempty"`
}

type RefundOutput struct {
	PaymentID        string `json:"paymentId"`
	Provider         string `json:"provider"`
	ProviderRefundID string `json:"providerRefundId"`
	Status           string `json:"status"`
}

func (s *PaymentService) RefundPayment(ctx context.Context, paymentID string, in RefundInput) (*RefundOutput, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT p.provider, p.provider_payment_id, p.status, o.amount, o.currency
		FROM payments p
		JOIN orders o ON o.id = p.order_id
		WHERE p.id=?
	`, paymentID)

	var providerName, providerPaymentID, status, currency string
	var amount float64
	if err := row.Scan(&providerName, &providerPaymentID, &status, &amount, &currency); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if providerPaymentID == "" {
		return nil, fmt.Errorf("%w: payment has no provider charge id", ErrInvalidState)
	}
	if strings.EqualFold(status, "refunded") {
		return nil, fmt.Errorf("%w: payment already refunded", ErrInvalidState)
	}

	refundAmount := in.Amount
	if refundAmount <= 0 {
		refundAmount = amount
	}

	result, err := s.provider.Refund(ctx, providers.RefundRequest{
		ProviderChargeID: providerPaymentID,
		Amount:           refundAmount,
		Currency:         currency,
		Reason:           in.Reason,
	})
	if err != nil {
		if errors.Is(err, providers.ErrNotSupported) {
			return nil, fmt.Errorf("%w: %v", ErrProviderUnsupported, err)
		}
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	raw, _ := json.Marshal(result.Raw)
	_, err = s.db.ExecContext(ctx, `
		UPDATE payments SET status=?, raw_last_event=?, updated_at=? WHERE id=?
	`, "refunded", string(raw), now, paymentID)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE orders SET status='refunded', updated_at=?
		WHERE id IN (SELECT order_id FROM payments WHERE id=?)
	`, now, paymentID)

	return &RefundOutput{
		PaymentID:        paymentID,
		Provider:         providerName,
		ProviderRefundID: result.ProviderRefundID,
		Status:           result.Status,
	}, nil
}

func (s *PaymentService) CompleteMockPayment(ctx context.Context, providerChargeID string) error {
	if s.provider.Name() != "mock" {
		return fmt.Errorf("%w: mock completion only works with mock provider", ErrInvalidState)
	}
	raw, _ := json.Marshal(map[string]string{
		"id":     providerChargeID,
		"status": "CAPTURED",
	})
	_, err := s.ApplyWebhook(ctx, WebhookUpdate{
		Provider: "mock",
		EventKey: providerChargeID,
		Status:   "CAPTURED",
		MarkPaid: true,
		RawJSON:  string(raw),
	})
	return err
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
