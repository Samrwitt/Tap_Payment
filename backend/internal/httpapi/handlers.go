package httpapi

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"tap-payment/backend/internal/providers/chapa"
	"tap-payment/backend/internal/providers/tap"
	"tap-payment/backend/internal/services"
)

type Handlers struct {
	db               *sql.DB
	svc              *services.PaymentService
	tapWebhookSecret string
	chapaSecret      string
	adminAPIKey      string
}

func NewHandlers(db *sql.DB, svc *services.PaymentService, tapWebhookSecret, chapaSecret, adminAPIKey string) *Handlers {
	return &Handlers{
		db:               db,
		svc:              svc,
		tapWebhookSecret: tapWebhookSecret,
		chapaSecret:      chapaSecret,
		adminAPIKey:      adminAPIKey,
	}
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

	dup, err := h.svc.ApplyWebhook(r.Context(), services.WebhookUpdate{
		Provider: "tap",
		EventKey: ch.ID,
		Status:   ch.Status,
		MarkPaid: ch.Status == "CAPTURED",
		RawJSON:  string(body),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to process webhook")
		return
	}
	if dup {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) ChapaWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "read body failed")
		return
	}
	defer r.Body.Close()

	sig := headerValue(r.Header, "x-chapa-signature")
	if sig == "" {
		sig = headerValue(r.Header, "Chapa-Signature")
	}
	if !chapa.VerifyWebhookSignature(h.chapaSecret, body, sig) {
		writeError(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "invalid webhook signature")
		return
	}

	var payload chapa.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	if payload.TxRef == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "tx_ref required")
		return
	}

	markPaid := strings.EqualFold(payload.Status, "success") || strings.EqualFold(payload.Status, "successful")
	dup, err := h.svc.ApplyWebhook(r.Context(), services.WebhookUpdate{
		Provider: "chapa",
		EventKey: payload.TxRef,
		Status:   payload.Status,
		MarkPaid: markPaid,
		RawJSON:  string(body),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to process webhook")
		return
	}
	if dup {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
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
		ID          string `json:"id"`
		OrderID     string `json:"orderId"`
		Provider    string `json:"provider"`
		ProviderID  string `json:"providerPaymentId"`
		Status      string `json:"status"`
		RedirectURL string `json:"redirectUrl,omitempty"`
		CreatedAt   string `json:"createdAt"`
		UpdatedAt   string `json:"updatedAt"`
	}
	if err := row.Scan(&out.ID, &out.OrderID, &out.Provider, &out.ProviderID, &out.Status, &out.RedirectURL, &out.CreatedAt, &out.UpdatedAt); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "payment not found")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) RefundPayment(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeAdmin(r) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid X-Admin-API-Key required")
		return
	}

	paymentID := chi.URLParam(r, "paymentId")
	var in services.RefundInput
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(w, r, &in); err != nil {
			return
		}
	}

	out, err := h.svc.RefundPayment(r.Context(), paymentID, in)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			writeError(w, http.StatusNotFound, "NOT_FOUND", "payment not found")
		case errors.Is(err, services.ErrInvalidState):
			writeError(w, http.StatusConflict, "INVALID_STATE", err.Error())
		case errors.Is(err, services.ErrProviderUnsupported):
			writeError(w, http.StatusNotImplemented, "NOT_SUPPORTED", err.Error())
		default:
			writeError(w, http.StatusBadGateway, "PROVIDER_ERROR", "refund failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) MockCheckoutPage(w http.ResponseWriter, r *http.Request) {
	chargeID := chi.URLParam(r, "chargeId")
	safeID := html.EscapeString(chargeID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html><head><title>Mock Checkout</title></head>
<body style="font-family:sans-serif;max-width:480px;margin:3rem auto;">
  <h1>Mock Checkout</h1>
  <p>Charge: <code>` + safeID + `</code></p>
  <form method="POST" action="/mock/checkout/` + safeID + `/complete">
    <button type="submit">Pay successfully</button>
  </form>
</body></html>`))
}

func (h *Handlers) MockComplete(w http.ResponseWriter, r *http.Request) {
	chargeID := chi.URLParam(r, "chargeId")
	if err := h.svc.CompleteMockPayment(r.Context(), chargeID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_STATE", err.Error())
		return
	}
	if strings.Contains(r.Header.Get("Accept"), "text/html") || r.Method == http.MethodPost {
		http.Redirect(w, r, "/payment/return?status=paid&chargeId="+chargeID, http.StatusSeeOther)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "chargeId": chargeID})
}

func (h *Handlers) PaymentReturn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	status := html.EscapeString(r.URL.Query().Get("status"))
	_, _ = w.Write([]byte(`<!doctype html><html><body style="font-family:sans-serif;margin:3rem;">
	<h1>Payment return</h1><p>Status: ` + status + `</p></body></html>`))
}

func (h *Handlers) authorizeAdmin(r *http.Request) bool {
	if h.adminAPIKey == "" {
		return false
	}
	got := r.Header.Get("X-Admin-API-Key")
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.adminAPIKey)) == 1
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
	if v := h.Get(key); v != "" {
		return v
	}
	return h.Get(http.CanonicalHeaderKey(key))
}
