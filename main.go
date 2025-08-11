package main

import (
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config()

	scheduler := NewScheduler(cfg,
		cfg.schedulerCurrentInterval,
		cfg.schedulerHourlyInterval,
		cfg.schedulerDailyInterval,
	)
	scheduler.Start()

	mux := http.NewServeMux()

	mux.HandleFunc("/currentweather", cfg.handlerCurrentWeather)
	mux.HandleFunc("/dailyforecast", cfg.handlerDailyForecast)
	mux.HandleFunc("/hourlyforecast", cfg.handlerHourlyForecast)

	if cfg.devMode {
		log.Println("Development mode enabled. Registering /dev/reset-db endpoint.")
		mux.HandleFunc("/dev/reset-db", cfg.handlerResetDB)
	}

	server := &http.Server{
		Addr:    ":" + cfg.port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", cfg.port)
	log.Fatal(server.ListenAndServe())
}
