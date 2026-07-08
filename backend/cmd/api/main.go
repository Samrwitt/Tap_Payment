package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"tap-payment/backend/internal/app"
)

func main() {
	_ = godotenv.Load()

	cfg := app.LoadConfigFromEnv()
	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           a.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Printf("shutdown complete")
}

