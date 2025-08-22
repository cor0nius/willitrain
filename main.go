package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	cfg := config()
	cfg.logger.Debug("configuration loaded")

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

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/currentweather", cfg.handlerCurrentWeather)
	mux.HandleFunc("/api/dailyforecast", cfg.handlerDailyForecast)
	mux.HandleFunc("/api/hourlyforecast", cfg.handlerHourlyForecast)
	mux.HandleFunc("/metrics", cfg.handlerMetrics)

	if cfg.devMode {
		cfg.logger.Debug("development mode enabled. Registering /dev/reset-db, /dev/runschedulerjobs endpoints.")
		mux.HandleFunc("/dev/reset-db", cfg.handlerResetDB)
		mux.HandleFunc("/dev/runschedulerjobs", scheduler.handlerRunSchedulerJobs)
	}

	// Frontend file server
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		cfg.logger.Error("failed to create frontend file system", "error", err)
		os.Exit(1)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

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
