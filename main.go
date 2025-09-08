package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// This file is the main entrypoint for the WillItRain application.
// It orchestrates the entire startup sequence:
// 1. Initializes the application configuration.
// 2. Starts the background scheduler for periodic data updates.
// 3. Sets up the HTTP router with all API and frontend routes.
// 4. Wraps the router in middleware for metrics and CORS.
// 5. Starts the web server.

// frontendFS embeds the compiled frontend assets into the Go binary.
// This allows the application to be deployed as a single, self-contained executable.
//
//go:embed all:frontend/dist
var frontendFS embed.FS

func run(ctx context.Context) error {
	// Initialize the application configuration, which includes setting up
	// the logger, database connections, and other dependencies.
	cfg, err := NewAPIConfig(os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	cfg.logger.Debug("configuration loaded")

	// Establish connections to the database and cache.
	err = cfg.ConnectDB()
	if err != nil {
		return fmt.Errorf("couldn't connect to database: %w", err)
	}

	err = cfg.ConnectCache()
	if err != nil {
		return fmt.Errorf("couldn't connect to cache: %w", err)
	}

	// Create and start the scheduler for periodic weather data updates.
	scheduler := NewScheduler(cfg,
		cfg.schedulerCurrentInterval,
		cfg.schedulerHourlyInterval,
		cfg.schedulerDailyInterval,
	)
	cfg.logger.Info(
		"starting scheduler",
		"current", cfg.schedulerCurrentInterval.String(),
		"hourly", cfg.schedulerHourlyInterval.String(),
		"daily", cfg.schedulerDailyInterval.String(),
	)
	scheduler.Start()

	// Set up the HTTP request multiplexer (router).
	mux := http.NewServeMux()

	// Register the public API endpoints.
	mux.HandleFunc("/api/config", cfg.handlerConfig)
	mux.HandleFunc("/api/currentweather", cfg.handlerCurrentWeather)
	mux.HandleFunc("/api/dailyforecast", cfg.handlerDailyForecast)
	mux.HandleFunc("/api/hourlyforecast", cfg.handlerHourlyForecast)
	mux.Handle("/metrics", promhttp.Handler())

	// Register development-only endpoints if dev mode is enabled.
	if cfg.devMode {
		cfg.logger.Debug("development mode enabled. Registering /dev/reset-db, /dev/runschedulerjobs endpoints.")
		mux.HandleFunc("/dev/reset-db", cfg.handlerResetDB)
		mux.HandleFunc("/dev/runschedulerjobs", scheduler.handlerRunSchedulerJobs)
	}

	// Set up the file server to serve the embedded frontend assets.
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		return fmt.Errorf("failed to create frontend file system: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	// Configure and start the HTTP server, wrapping the router with middleware.
	// The /metrics endpoint is excluded from metricsMiddleware.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			corsMiddleware(mux).ServeHTTP(w, r)
		} else {
			metricsMiddleware(corsMiddleware(mux)).ServeHTTP(w, r)
		}
	})

	server := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Goroutine for graceful shutdown
	go func() {
		<-ctx.Done() // Block until context is cancelled
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			cfg.logger.Error("server shutdown failed", "error", err)
		}
	}()

	cfg.logger.Info("starting server", "port", cfg.port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server startup failed: %w", err)
	}
	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}