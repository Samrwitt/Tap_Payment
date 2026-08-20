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

	webhookURL := cfg.TapWebhookURL
	if webhookURL == "" {
		webhookURL = cfg.BaseURL + "/api/payments/webhooks/tap"
	}

	svc := services.NewPaymentService(database, provider, services.PaymentServiceConfig{
		BaseURL:    cfg.BaseURL,
		WebhookURL: webhookURL,
	})
	handlers := httpapi.NewHandlers(database, svc, tapWebhookSecret)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/payments", func(r chi.Router) {
		r.Post("/charges", handlers.CreateCharge)
		r.Post("/webhooks/tap", handlers.TapWebhook)
		r.Get("/{paymentId}", handlers.GetPayment)
	})

	return &App{Router: r, DB: database}, nil
}

func newProvider(cfg Config) (providers.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.PaymentProvider)) {
	case "", "tap":
		return tap.NewProvider(cfg.TapSecretKey), nil
	case "mock":
		return mock.NewProvider(cfg.BaseURL), nil
	default:
		return nil, fmt.Errorf("unsupported PAYMENT_PROVIDER %q (use tap or mock)", cfg.PaymentProvider)
	}
}
