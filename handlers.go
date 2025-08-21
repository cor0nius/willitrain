package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (cfg *apiConfig) handlerCurrentWeather(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		cfg.respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	location, err := cfg.getLocationFromRequest(r)
	if err != nil {
		cfg.respondWithError(w, http.StatusBadRequest, "Error getting location data", err)
		return
	}
	cfg.logger.Debug("current weather request", "city", location.CityName)

	weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Error getting current weather data", err)
		return
	}

	response := CurrentWeatherResponse{
		Location: location,
		Weather:  weather,
	}

	cfg.respondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) handlerDailyForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		cfg.respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	location, err := cfg.getLocationFromRequest(r)
	if err != nil {
		cfg.respondWithError(w, http.StatusBadRequest, "Error getting location data", err)
		return
	}
	cfg.logger.Debug("daily forecast request", "city", location.CityName)

	forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Error getting daily forecast data", err)
		return
	}

	response := DailyForecastsResponse{
		Location:  location,
		Forecasts: forecast,
	}

	cfg.respondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) handlerHourlyForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		cfg.respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	location, err := cfg.getLocationFromRequest(r)
	if err != nil {
		cfg.respondWithError(w, http.StatusBadRequest, "Error getting location data", err)
		return
	}
	cfg.logger.Debug("hourly forecast request", "city", location.CityName)

	forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Error getting hourly forecast data", err)
		return
	}

	response := HourlyForecastsResponse{
		Location:  location,
		Forecasts: forecast,
	}

	cfg.respondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

func (cfg *apiConfig) handlerResetDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		cfg.respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}
	cfg.logger.Debug("database reset request received")

	ctx := r.Context()

	err := cfg.dbQueries.DeleteAllLocations(ctx)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Failed to reset database", err)
		return
	}

	err = cfg.cache.Flush(ctx)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Failed to flush cache", err)
		return
	}

	cfg.respondWithJSON(w, http.StatusOK, map[string]string{"status": "database and cache reset"})
}
