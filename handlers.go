package main

import (
	"net/http"
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

	weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Error getting current weather data", err)
		return
	}

	cfg.respondWithJSON(w, http.StatusOK, weather)
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

	forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Error getting daily forecast data", err)
		return
	}

	cfg.respondWithJSON(w, http.StatusOK, forecast)
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

	forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Error getting hourly forecast data", err)
		return
	}

	cfg.respondWithJSON(w, http.StatusOK, forecast)
}

func (cfg *apiConfig) handlerResetDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		cfg.respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	ctx := r.Context()

	err := cfg.dbQueries.DeleteAllLocations(ctx)
	if err != nil {
		cfg.respondWithError(w, http.StatusInternalServerError, "Failed to reset database", err)
		return
	}

	cfg.respondWithJSON(w, http.StatusOK, map[string]string{"status": "database reset"})
}
