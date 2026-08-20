package chapa

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	body := []byte(`{"tx_ref":"pay_1","status":"success"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !VerifyWebhookSignature("secret", body, sig) {
		t.Fatal("expected valid signature")
	}
	if VerifyWebhookSignature("secret", body, "bad") {
		t.Fatal("expected invalid signature")
	}
}
