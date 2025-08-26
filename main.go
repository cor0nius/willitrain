package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
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
//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	// Initialize the application configuration, which includes setting up
	// the logger, database connections, and other dependencies.
	cfg := config()
	cfg.logger.Debug("configuration loaded")

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
	mux.HandleFunc("/metrics", cfg.handlerMetrics)

	// Register development-only endpoints if dev mode is enabled.
	if cfg.devMode {
		cfg.logger.Debug("development mode enabled. Registering /dev/reset-db, /dev/runschedulerjobs endpoints.")
		mux.HandleFunc("/dev/reset-db", cfg.handlerResetDB)
		mux.HandleFunc("/dev/runschedulerjobs", scheduler.handlerRunSchedulerJobs)
	}

	// Set up the file server to serve the embedded frontend assets.
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		cfg.logger.Error("failed to create frontend file system", "error", err)
		os.Exit(1)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	// Configure and start the HTTP server, wrapping the router with middleware.
	server := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           metricsMiddleware(corsMiddleware(mux)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	cfg.logger.Info("starting server", "port", cfg.port)
	err = server.ListenAndServe()
	if err != nil {
		cfg.logger.Error("server startup failed", "error", err)
		os.Exit(1)
	}
}
