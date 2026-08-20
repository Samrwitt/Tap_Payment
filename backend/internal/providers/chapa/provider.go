package chapa

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tap-payment/backend/internal/providers"
)

const baseURL = "https://api.chapa.co/v1"

// Provider implements Ethiopian settlement via Chapa.
type Provider struct {
	secretKey string
	http      *http.Client
}

func NewProvider(secretKey string) *Provider {
	return &Provider{
		secretKey: secretKey,
		http:      &http.Client{Timeout: 20 * time.Second},
	}
}

func (p *Provider) Name() string { return "chapa" }

type initializeRequest struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	TxRef       string `json:"tx_ref"`
	CallbackURL string `json:"callback_url,omitempty"`
	ReturnURL   string `json:"return_url,omitempty"`
}

type initializeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		CheckoutURL string `json:"checkout_url"`
	} `json:"data"`
}

func (p *Provider) CreateCharge(ctx context.Context, req providers.ChargeRequest) (*providers.ChargeResult, error) {
	if p.secretKey == "" {
		return nil, fmt.Errorf("chapa secret key is empty")
	}
	if strings.ToUpper(req.Currency) != "ETB" {
		return nil, fmt.Errorf("%w: chapa currently expects ETB", providers.ErrNotSupported)
	}
	email := req.Customer.Email
	if email == "" {
		email = fmt.Sprintf("%s@payments.local", req.PaymentID)
	}

	phone := strings.TrimSpace(req.Customer.Phone.Number)
	if phone != "" && !strings.HasPrefix(phone, "0") && !strings.HasPrefix(phone, "+") {
		phone = "0" + phone
	}

	body := initializeRequest{
		Amount:      formatAmount(req.Amount),
		Currency:    "ETB",
		Email:       email,
		FirstName:   req.Customer.FirstName,
		LastName:    req.Customer.LastName,
		PhoneNumber: phone,
		TxRef:       req.PaymentID,
		CallbackURL: req.WebhookURL,
		ReturnURL:   req.RedirectURL,
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/transaction/initialize", bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chapa http: %w", err)
	}
	defer resp.Body.Close()

	var out initializeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("chapa decode: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || strings.ToLower(out.Status) != "success" {
		return nil, fmt.Errorf("chapa initialize failed: status=%d message=%s", resp.StatusCode, out.Message)
	}
	if out.Data.CheckoutURL == "" {
		return nil, fmt.Errorf("chapa initialize missing checkout_url")
	}

	return &providers.ChargeResult{
		ProviderChargeID: req.PaymentID, // Chapa tx_ref is our payment id
		RedirectURL:      out.Data.CheckoutURL,
		Status:           "INITIATED",
		Raw:              out,
	}, nil
}

func (p *Provider) Refund(_ context.Context, _ providers.RefundRequest) (*providers.RefundResult, error) {
	return nil, fmt.Errorf("%w: chapa refunds are not wired in this demo", providers.ErrNotSupported)
}

func (p *Provider) SavePaymentMethod(_ context.Context, _ providers.SaveMethodRequest) (*providers.SavedMethodResult, error) {
	return nil, fmt.Errorf("%w: chapa one-tap enrollment not wired yet", providers.ErrNotSupported)
}

func (p *Provider) OneTapCharge(_ context.Context, _ providers.OneTapChargeRequest) (*providers.ChargeResult, error) {
	return nil, fmt.Errorf("%w: chapa one-tap charge not wired yet", providers.ErrNotSupported)
}

// VerifyWebhookSignature validates Chapa webhook authenticity (HMAC-SHA256 of raw body).
func VerifyWebhookSignature(secret string, rawBody []byte, signatureHeader string) bool {
	signatureHeader = strings.TrimSpace(signatureHeader)
	if secret == "" || signatureHeader == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signatureHeader))) ||
		hmac.Equal([]byte(expected), []byte(signatureHeader))
}

type WebhookPayload struct {
	TxRef     string `json:"tx_ref"`
	Status    string `json:"status"`
	Reference string `json:"reference"`
	Amount    any    `json:"amount"`
	Currency  string `json:"currency"`
}

func formatAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}
