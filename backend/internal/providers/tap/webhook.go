package tap

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// WebhookCharge is a partial representation of Tap's charge webhook payload.
// We only parse the fields needed for signature verification and state updates.
type WebhookCharge struct {
	ID        string  `json:"id"`
	Object    string  `json:"object"`
	Status    string  `json:"status"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Reference struct {
		Gateway  string `json:"gateway"`
		Payment  string `json:"payment"`
	} `json:"reference"`
	Transaction struct {
		Created string `json:"created"`
	} `json:"transaction"`
}

// VerifyHashstring verifies Tap's `hashstring` header using Tap's documented concatenation:
//
// toBeHashedString = x_id{id}x_amount{amount}x_currency{currency}x_gateway_reference{gateway}x_payment_reference{payment}x_status{status}x_created{created}
//
// amount must be formatted with the currency's standard decimal places.
func VerifyHashstring(secret, headerHashstring string, ch WebhookCharge) (bool, string) {
	headerHashstring = strings.TrimSpace(headerHashstring)
	if headerHashstring == "" {
		return false, ""
	}

	amt := formatAmount(ch.Currency, ch.Amount)
	toHash := "x_id" + ch.ID +
		"x_amount" + amt +
		"x_currency" + ch.Currency +
		"x_gateway_reference" + ch.Reference.Gateway +
		"x_payment_reference" + ch.Reference.Payment +
		"x_status" + ch.Status +
		"x_created" + ch.Transaction.Created

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(toHash))
	sum := mac.Sum(nil)
	expected := hex.EncodeToString(sum)
	return hmac.Equal([]byte(expected), []byte(headerHashstring)), expected
}

func formatAmount(currency string, amount float64) string {
	// Tap docs explicitly call out 2 or 3 decimals depending on currency.
	// We implement a small mapping with safe default=2.
	decimals := 2
	switch strings.ToUpper(currency) {
	case "BHD", "KWD", "OMR", "JOD":
		decimals = 3
	}

	pow := math.Pow10(decimals)
	rounded := math.Round(amount*pow) / pow
	return strconv.FormatFloat(rounded, 'f', decimals, 64)
}

func HashstringFromWebhook(secret string, header string, ch WebhookCharge) error {
	ok, expected := VerifyHashstring(secret, header, ch)
	if !ok {
		return fmt.Errorf("invalid tap hashstring: expected=%s got=%s", expected, strings.TrimSpace(header))
	}
	return nil
}

