package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dbpkg "tap-payment/backend/internal/db"
	"tap-payment/backend/internal/providers/tap"
)

func TestTapWebhookIsIdempotentAndUpdatesPayment(t *testing.T) {
	database := openTestDB(t)
	defer database.Close()

	seedPayment(t, database, "ord_1", "pay_1", "chg_test_123")

	h := NewHandlers(database, nil, "secret123")
	payload := tap.WebhookCharge{}
	payload.ID = "chg_test_123"
	payload.Object = "charge"
	payload.Status = "CAPTURED"
	payload.Amount = 1
	payload.Currency = "SAR"
	payload.Reference.Gateway = "gw_123"
	payload.Reference.Payment = "payref_123"
	payload.Transaction.Created = "1698392202943"

	_, hashstring := tap.VerifyHashstring("secret123", "invalid", payload)
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req1 := httptest.NewRequest(http.MethodPost, "/api/payments/webhooks/tap", strings.NewReader(string(bodyBytes)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("hashstring", hashstring)
	rec1 := httptest.NewRecorder()
	h.TapWebhook(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first webhook 200, got %d body=%s", rec1.Code, rec1.Body.String())
	}
	if !strings.Contains(rec1.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected ok response, got %s", rec1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/payments/webhooks/tap", strings.NewReader(string(bodyBytes)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("hashstring", hashstring)
	rec2 := httptest.NewRecorder()
	h.TapWebhook(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected duplicate webhook 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), `"status":"duplicate"`) {
		t.Fatalf("expected duplicate response, got %s", rec2.Body.String())
	}

	var paymentStatus, orderStatus string
	err = database.QueryRow(`
		SELECT p.status, o.status
		FROM payments p
		JOIN orders o ON o.id = p.order_id
		WHERE p.id = ?
	`, "pay_1").Scan(&paymentStatus, &orderStatus)
	if err != nil {
		t.Fatalf("query statuses: %v", err)
	}

	if paymentStatus != "captured" {
		t.Fatalf("expected payment status captured, got %q", paymentStatus)
	}
	if orderStatus != "paid" {
		t.Fatalf("expected order status paid, got %q", orderStatus)
	}

	var count int
	err = database.QueryRow(`SELECT COUNT(*) FROM webhook_events WHERE provider='tap' AND event_key=?`, payload.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count webhook_events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 webhook event, got %d", count)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := dbpkg.OpenAndMigrate(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return database
}

func seedPayment(t *testing.T, db *sql.DB, orderID, paymentID, providerPaymentID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.Exec(`
		INSERT INTO orders (id, amount, currency, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, orderID, 1.0, "SAR", "pending", now, now)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO payments (id, order_id, provider, provider_payment_id, status, redirect_url, raw_last_event, created_at, updated_at)
		VALUES (?, ?, 'tap', ?, 'initiated', '', '', ?, ?)
	`, paymentID, orderID, providerPaymentID, now, now)
	if err != nil {
		t.Fatalf("insert payment: %v", err)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

