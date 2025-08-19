package main

import (
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

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

	mux.HandleFunc("/currentweather", cfg.handlerCurrentWeather)
	mux.HandleFunc("/dailyforecast", cfg.handlerDailyForecast)
	mux.HandleFunc("/hourlyforecast", cfg.handlerHourlyForecast)
	mux.HandleFunc("/metrics", cfg.handlerMetrics)

	if cfg.devMode {
		cfg.logger.Debug("development mode enabled. Registering /dev/reset-db endpoint.")
		mux.HandleFunc("/dev/reset-db", cfg.handlerResetDB)
	}

	server := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           metricsMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	cfg.logger.Info("starting server", "port", cfg.port)
	err := server.ListenAndServe()
	if err != nil {
		cfg.logger.Error("server startup failed", "error", err)
		os.Exit(1)
	}
}
