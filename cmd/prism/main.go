package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/w-h-a/prism/internal/handler/http/health"
)

func main() {
	defaultPort := 8081
	if env := os.Getenv("PRISM_HEALTH_PORT"); env != "" {
		if p, err := strconv.Atoi(env); err == nil && p > 0 {
			defaultPort = p
		}
	}

	healthPort := flag.Int("health-port", defaultPort, "HTTP health check port")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	h := health.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /readyz", h.Readyz)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *healthPort),
		Handler: mux,
	}

	go func() {
		logger.Info("health server starting", "port", *healthPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health server", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	logger.Info("shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "error", err)
		return
	}

	logger.Info("shutdown complete")
}
