package app

import (
	"os"
	"strconv"
)

type Config struct {
	Port                int
	BaseURL             string
	FrontendURL         string
	CORSOrigin          string
	SQLitePath          string
	PaymentProvider     string
	AdminAPIKey         string
	TapSecretKey        string
	TapWebhookSecretKey string
	ChapaSecretKey      string
	WebhookURL          string
}

func LoadConfigFromEnv() Config {
	frontend := envString("FRONTEND_URL", "http://localhost:3000")
	return Config{
		Port:                envInt("PORT", 8080),
		BaseURL:             envString("BASE_URL", "http://localhost:8080"),
		FrontendURL:         frontend,
		CORSOrigin:          envString("CORS_ORIGIN", frontend),
		SQLitePath:          envString("SQLITE_PATH", "./data/payments.db"),
		PaymentProvider:     envString("PAYMENT_PROVIDER", "mock"),
		AdminAPIKey:         envString("ADMIN_API_KEY", ""),
		TapSecretKey:        envString("TAP_SECRET_KEY", ""),
		TapWebhookSecretKey: envString("TAP_WEBHOOK_SECRET_KEY", ""),
		ChapaSecretKey:      envString("CHAPA_SECRET_KEY", ""),
		WebhookURL:          envString("WEBHOOK_URL", ""),
	}
}

func (c Config) Addr() string {
	return ":" + strconv.Itoa(c.Port)
}

func envString(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
