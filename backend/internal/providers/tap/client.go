package tap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	secretKey string
	http      *http.Client
}

func NewClient(secretKey string) *Client {
	return &Client{
		secretKey: secretKey,
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type CreateChargeRequest struct {
	Amount             float64                `json:"amount"`
	Currency           string                 `json:"currency"`
	CustomerInitiated  bool                   `json:"customer_initiated"`
	ThreeDSecure       bool                   `json:"threeDSecure"`
	Description        string                 `json:"description,omitempty"`
	Metadata           map[string]string      `json:"metadata,omitempty"`
	Customer           CreateChargeCustomer   `json:"customer"`
	Source             CreateChargeSource     `json:"source"`
	Redirect           CreateChargeRedirect   `json:"redirect"`
	Post               *CreateChargePost      `json:"post,omitempty"`
	Reference          map[string]string      `json:"reference,omitempty"`
}

type CreateChargeCustomer struct {
	FirstName string      `json:"first_name,omitempty"`
	LastName  string      `json:"last_name,omitempty"`
	Email     string      `json:"email,omitempty"`
	Phone     PhoneNumber `json:"phone"`
}

type PhoneNumber struct {
	CountryCode string `json:"country_code"`
	Number      string `json:"number"`
}

type CreateChargeSource struct {
	ID string `json:"id"`
}

type CreateChargeRedirect struct {
	URL string `json:"url"`
}

type CreateChargePost struct {
	URL string `json:"url"`
}

type CreateChargeResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Transaction struct {
		URL string `json:"url"`
	} `json:"transaction"`
	Response *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"response,omitempty"`
}

func (c *Client) CreateCharge(ctx context.Context, req CreateChargeRequest) (*CreateChargeResponse, error) {
	if c.secretKey == "" {
		return nil, fmt.Errorf("tap secret key is empty")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tap.company/v2/charges", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("tap http: %w", err)
	}
	defer resp.Body.Close()

	var out CreateChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := ""
		msg := ""
		if out.Response != nil {
			code = out.Response.Code
			msg = out.Response.Message
		}
		return &out, fmt.Errorf("tap create charge failed: status=%d code=%s message=%s", resp.StatusCode, code, msg)
	}

	return &out, nil
}

