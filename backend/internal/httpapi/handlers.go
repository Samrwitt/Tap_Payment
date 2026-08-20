package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"tap-payment/backend/internal/providers/tap"
	"tap-payment/backend/internal/services"
)

type Handlers struct {
	db               *sql.DB
	svc              *services.PaymentService
	tapWebhookSecret string
}

func NewHandlers(db *sql.DB, svc *services.PaymentService, tapWebhookSecret string) *Handlers {
	return &Handlers{db: db, svc: svc, tapWebhookSecret: tapWebhookSecret}
}

func (h *Handlers) CreateCharge(w http.ResponseWriter, r *http.Request) {
	var in services.CreateChargeInput
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}

	out, err := h.svc.CreateCharge(r.Context(), in)
	if err != nil {
		if errors.Is(err, services.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, "PROVIDER_ERROR", "payment provider request failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) TapWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "read body failed")
		return
	}
	defer r.Body.Close()

	var ch tap.WebhookCharge
	if err := json.Unmarshal(body, &ch); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}

	hashstring := headerValue(r.Header, "hashstring")
	if err := tap.HashstringFromWebhook(h.tapWebhookSecret, hashstring, ch); err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "invalid webhook signature")
		return
	}

	receivedAt := time.Now().UTC().Format(time.RFC3339Nano)
	// Idempotency: only process a given charge.id once.
	eventID := "wev_" + services.RandID()
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO webhook_events (id, provider, event_key, received_at)
		VALUES (?, 'tap', ?, ?)
	`, eventID, ch.ID, receivedAt)
	if err != nil {
		// Only unique-key violations are considered duplicates.
		if isUniqueConstraintError(err) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to persist webhook event")
		return
	}

	// Update payment by provider_payment_id (charge id)
	status := strings.ToLower(ch.Status)
	updatedAt := receivedAt
	raw := string(body)
	_, err = h.db.ExecContext(r.Context(), `
		UPDATE payments SET status=?, raw_last_event=?, updated_at=?
		WHERE provider='tap' AND provider_payment_id=?
	`, status, raw, updatedAt, ch.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update payment status")
		return
	}

	// If captured, mark order paid (best effort).
	if ch.Status == "CAPTURED" {
		_, err = h.db.ExecContext(r.Context(), `
			UPDATE orders SET status='paid', updated_at=?
			WHERE id IN (SELECT order_id FROM payments WHERE provider='tap' AND provider_payment_id=?)
		`, updatedAt, ch.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update order status")
			return
		}
	}

	_, err = h.db.ExecContext(r.Context(), `
		UPDATE webhook_events SET processed_at=? WHERE provider='tap' AND event_key=?
	`, updatedAt, ch.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to mark webhook as processed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) GetPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "paymentId")
	if paymentID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "paymentId required")
		return
	}

	row := h.db.QueryRowContext(r.Context(), `
		SELECT id, order_id, provider, provider_payment_id, status, redirect_url, created_at, updated_at
		FROM payments WHERE id=?
	`, paymentID)

	var out struct {
		ID              string `json:"id"`
		OrderID         string `json:"orderId"`
		Provider        string `json:"provider"`
		ProviderID      string `json:"providerPaymentId"`
		Status          string `json:"status"`
		RedirectURL     string `json:"redirectUrl,omitempty"`
		CreatedAt       string `json:"createdAt"`
		UpdatedAt       string `json:"updatedAt"`
	}
	if err := row.Scan(&out.ID, &out.OrderID, &out.Provider, &out.ProviderID, &out.Status, &out.RedirectURL, &out.CreatedAt, &out.UpdatedAt); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "payment not found")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE", "content-type must be application/json")
		return errors.New("bad content type")
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": msg,
		},
	})
}

func headerValue(h http.Header, key string) string {
	// net/http canonicalizes header names; try direct then canonical.
	if v := h.Get(key); v != "" {
		return v
	}
	return h.Get(http.CanonicalHeaderKey(key))
}

func isUniqueConstraintError(err error) bool {
	// modernc sqlite returns driver errors as strings containing this phrase.
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

