package main

import (
	"net/http"
)

func (cfg *apiConfig) handlerCurrentWeather(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	cityName := r.URL.Query().Get("city")
	if cityName == "" {
		respondWithError(w, http.StatusBadRequest, "city query parameter is required", nil)
		return
	}

	location, err := cfg.getOrCreateLocation(ctx, cityName)
	if err != nil {
		// getOrCreateLocation handles logging persistence errors, so we just need
		// to handle the case where we can't get a location at all.
		respondWithError(w, http.StatusInternalServerError, "Error getting location data", err)
		return
	}

	// Get weather data, either from cache or by fetching from APIs
	weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting current weather data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, weather)
}

func (cfg *apiConfig) handlerDailyForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	cityName := r.URL.Query().Get("city")
	if cityName == "" {
		respondWithError(w, http.StatusBadRequest, "city query parameter is required", nil)
		return
	}

	location, err := cfg.getOrCreateLocation(ctx, cityName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting location data", err)
		return
	}

	forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting daily forecast data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, forecast)
}

func (cfg *apiConfig) handlerHourlyForecast(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	cityName := r.URL.Query().Get("city")
	if cityName == "" {
		respondWithError(w, http.StatusBadRequest, "city query parameter is required", nil)
		return
	}

	location, err := cfg.getOrCreateLocation(ctx, cityName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting location data", err)
		return
	}

	forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting hourly forecast data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, forecast)
}
