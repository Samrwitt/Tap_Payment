package app

import (
	"os"
	"strconv"
)

type Config struct {
	Port               int
	BaseURL            string
	SQLitePath         string
	TapSecretKey       string
	TapWebhookSecretKey string
	TapWebhookURL      string
}

func LoadConfigFromEnv() Config {
	return Config{
		Port:                envInt("PORT", 8080),
		BaseURL:             envString("BASE_URL", "http://localhost:8080"),
		SQLitePath:          envString("SQLITE_PATH", "./data/payments.db"),
		TapSecretKey:        envString("TAP_SECRET_KEY", ""),
		TapWebhookSecretKey: envString("TAP_WEBHOOK_SECRET_KEY", ""),
		TapWebhookURL:       envString("TAP_WEBHOOK_URL", "http://localhost:8080/api/payments/webhooks/tap"),
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

