package app

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"tap-payment/backend/internal/db"
	"tap-payment/backend/internal/httpapi"
	"tap-payment/backend/internal/providers"
	"tap-payment/backend/internal/providers/chapa"
	"tap-payment/backend/internal/providers/mock"
	"tap-payment/backend/internal/providers/tap"
	"tap-payment/backend/internal/services"
)

type App struct {
	Router http.Handler
	DB     *sql.DB
}

func New(cfg Config) (*App, error) {
	if cfg.SQLitePath == "" {
		return nil, errors.New("SQLITE_PATH is required")
	}

	database, err := db.OpenAndMigrate(cfg.SQLitePath)
	if err != nil {
		return nil, err
	}

	provider, err := newProvider(cfg)
	if err != nil {
		_ = database.Close()
		return nil, err
	}

	tapWebhookSecret := cfg.TapWebhookSecretKey
	if tapWebhookSecret == "" {
		tapWebhookSecret = cfg.TapSecretKey
	}

	webhookURL := cfg.WebhookURL
	if webhookURL == "" {
		switch provider.Name() {
		case "chapa":
			webhookURL = cfg.BaseURL + "/api/payments/webhooks/chapa"
		default:
			webhookURL = cfg.BaseURL + "/api/payments/webhooks/tap"
		}
	}

	svc := services.NewPaymentService(database, provider, services.PaymentServiceConfig{
		BaseURL:    cfg.BaseURL,
		WebhookURL: webhookURL,
	})
	handlers := httpapi.NewHandlers(database, svc, tapWebhookSecret, cfg.ChapaSecretKey, cfg.AdminAPIKey, cfg.FrontendURL)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware(cfg.CORSOrigin))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/payment/return", handlers.PaymentReturn)
	r.Get("/mock/checkout/{chargeId}", handlers.MockCheckoutPage)
	r.Post("/mock/checkout/{chargeId}/complete", handlers.MockComplete)

	r.Route("/api/payments", func(r chi.Router) {
		r.Post("/charges", handlers.CreateCharge)
		r.Post("/methods", handlers.SavePaymentMethod)
		r.Get("/methods", handlers.ListPaymentMethods)
		r.Post("/one-tap", handlers.OneTapPay)
		r.Post("/webhooks/tap", handlers.TapWebhook)
		r.Post("/webhooks/chapa", handlers.ChapaWebhook)
		r.Get("/{paymentId}", handlers.GetPayment)
		r.Post("/{paymentId}/refund", handlers.RefundPayment)
	})

	return &App{Router: r, DB: database}, nil
}

func corsMiddleware(origin string) func(http.Handler) http.Handler {
	if origin == "" {
		origin = "http://localhost:3000"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-API-Key, hashstring, x-chapa-signature, Chapa-Signature")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func newProvider(cfg Config) (providers.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.PaymentProvider)) {
	case "", "tap":
		return tap.NewProvider(cfg.TapSecretKey), nil
	case "mock":
		return mock.NewProvider(cfg.BaseURL), nil
	case "chapa":
		return chapa.NewProvider(cfg.ChapaSecretKey), nil
	default:
		return nil, fmt.Errorf("unsupported PAYMENT_PROVIDER %q (use tap, mock, or chapa)", cfg.PaymentProvider)
	}
}
