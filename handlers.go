package main

import (
	"net/http"
)

func (cfg *apiConfig) handlerCurrentWeather(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	cityName := r.URL.Query().Get("city")
	if cityName == "" {
		respondWithError(w, http.StatusBadRequest, "city query parameter is required", nil)
		return
	}

	location, err := cfg.Geocode(cityName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not geocode city", err)
		return
	}

	weather, err := cfg.requestCurrentWeather(location)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not fetch current weather", err)
		return
	}

	respondWithJSON(w, http.StatusOK, weather)
}

func (cfg *apiConfig) handlerDailyForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	cityName := r.URL.Query().Get("city")
	if cityName == "" {
		respondWithError(w, http.StatusBadRequest, "city query parameter is required", nil)
		return
	}

	location, err := cfg.Geocode(cityName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not geocode city", err)
		return
	}

	forecast, err := cfg.requestDailyForecast(location)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not fetch daily forecast", err)
		return
	}

	respondWithJSON(w, http.StatusOK, forecast)
}

func (cfg *apiConfig) handlerHourlyForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	cityName := r.URL.Query().Get("city")
	if cityName == "" {
		respondWithError(w, http.StatusBadRequest, "city query parameter is required", nil)
		return
	}

	location, err := cfg.Geocode(cityName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not geocode city", err)
		return
	}

	forecast, err := cfg.requestHourlyForecast(location)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not fetch hourly forecast", err)
		return
	}

	respondWithJSON(w, http.StatusOK, forecast)
}
