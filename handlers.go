package main

import (
	"net/http"
	"sort"
	"sync"
	"time"
)

// This file contains the main HTTP handlers for the application. Each handler is responsible
// for processing incoming API requests, calling the appropriate helper functions to fetch
// and process data, and writing the final JSON response.

// The weather handlers (handlerCurrentWeather, handlerDailyForecast, handlerHourlyForecast)
// follow a similar pattern:
// 1. They ensure the request method is GET.
// 2. They extract the location from the request using getLocationFromRequest.
// 3. They fetch the relevant forecast data using the appropriate getCachedOrFetch... function.
// 4. They sort the results for a consistent response order.
// 5. They format the data into the final JSON response structure.
// 6. They send the JSON response to the client.
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

	sort.Slice(weather, func(i, j int) bool {
		if weather[i].Timestamp.Equal(weather[j].Timestamp) {
			return weather[i].SourceAPI < weather[j].SourceAPI
		}
		return weather[i].Timestamp.Before(weather[j].Timestamp)
	})

	loc, err := time.LoadLocation(location.Timezone)
	if err != nil {
		cfg.logger.Warn("could not load location timezone, falling back to UTC", "timezone", location.Timezone, "error", err)
		loc = time.UTC
	}

	weatherJSON := make([]CurrentWeatherJSON, len(weather))
	for i, w := range weather {
		weatherJSON[i] = CurrentWeatherJSON{
			SourceAPI:     w.SourceAPI,
			Timestamp:     w.Timestamp.In(loc).Format("2006-01-02 15:04"),
			Temperature:   w.Temperature,
			Humidity:      w.Humidity,
			WindSpeed:     w.WindSpeed,
			Precipitation: w.Precipitation,
			Condition:     w.Condition,
		}
	}

	response := CurrentWeatherResponse{
		Location: location,
		Weather:  weatherJSON,
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

	sort.Slice(forecast, func(i, j int) bool {
		if forecast[i].ForecastDate.Equal(forecast[j].ForecastDate) {
			return forecast[i].SourceAPI < forecast[j].SourceAPI
		}
		return forecast[i].ForecastDate.Before(forecast[j].ForecastDate)
	})

	loc, err := time.LoadLocation(location.Timezone)
	if err != nil {
		cfg.logger.Warn("could not load location timezone, falling back to UTC", "timezone", location.Timezone, "error", err)
		loc = time.UTC
	}

	forecastsJSON := make([]DailyForecastJSON, len(forecast))
	for i, f := range forecast {
		forecastsJSON[i] = DailyForecastJSON{
			SourceAPI:           f.SourceAPI,
			ForecastDate:        f.ForecastDate.In(loc).Format("2006-01-02"),
			MinTemp:             f.MinTemp,
			MaxTemp:             f.MaxTemp,
			Precipitation:       f.Precipitation,
			PrecipitationChance: f.PrecipitationChance,
			WindSpeed:           f.WindSpeed,
			Humidity:            f.Humidity,
		}
	}

	response := DailyForecastsResponse{
		Location:  location,
		Forecasts: forecastsJSON,
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

	sort.Slice(forecast, func(i, j int) bool {
		if forecast[i].ForecastDateTime.Equal(forecast[j].ForecastDateTime) {
			return forecast[i].SourceAPI < forecast[j].SourceAPI
		}
		return forecast[i].ForecastDateTime.Before(forecast[j].ForecastDateTime)
	})

	loc, err := time.LoadLocation(location.Timezone)
	if err != nil {
		cfg.logger.Warn("could not load location timezone, falling back to UTC", "timezone", location.Timezone, "error", err)
		loc = time.UTC
	}

	forecastsJSON := make([]HourlyForecastJSON, len(forecast))
	for i, f := range forecast {
		forecastsJSON[i] = HourlyForecastJSON{
			SourceAPI:           f.SourceAPI,
			ForecastDateTime:    f.ForecastDateTime.In(loc).Format("2006-01-02 15:04"),
			Temperature:         f.Temperature,
			Humidity:            f.Humidity,
			WindSpeed:           f.WindSpeed,
			Precipitation:       f.Precipitation,
			PrecipitationChance: f.PrecipitationChance,
			Condition:           f.Condition,
		}
	}

	response := HourlyForecastsResponse{
		Location:  location,
		Forecasts: forecastsJSON,
	}

	cfg.respondWithJSON(w, http.StatusOK, response)
}

// handlerResetDB is a development-only endpoint that completely wipes the database and the Redis cache.
// Deleting all locations cascades and clears all related weather and forecast data.
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

// handlerRunSchedulerJobs is a development-only endpoint that manually triggers
// a run of all scheduled data update jobs.
func (s *Scheduler) handlerRunSchedulerJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.cfg.respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}
	s.cfg.logger.Info("manual scheduler run triggered")

	// Reset tickers
	s.tickers[0].Reset(s.cfg.schedulerCurrentInterval)
	s.tickers[1].Reset(s.cfg.schedulerHourlyInterval)
	s.tickers[2].Reset(s.cfg.schedulerDailyInterval)

	go func() {
		s.cfg.logger.Info("starting manual scheduler jobs")
		var wg sync.WaitGroup
		wg.Add(3)

		go func() {
			defer wg.Done()
			s.currentWeatherJobs()
		}()
		go func() {
			defer wg.Done()
			s.hourlyForecastJobs()
		}()
		go func() {
			defer wg.Done()
			s.dailyForecastJobs()
		}()

		wg.Wait()
		s.cfg.logger.Info("manual scheduler run finished")
	}()

	s.cfg.respondWithJSON(w, http.StatusAccepted, map[string]string{"status": "scheduler jobs triggered"})
}

// handlerConfig provides client-side applications with necessary configuration,
// such as whether the application is running in development mode.
func (cfg *apiConfig) handlerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		cfg.respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", nil)
		return
	}

	type configResponse struct {
		DevMode bool `json:"dev_mode"`
	}

	response := configResponse{
		DevMode: cfg.devMode,
	}

	cfg.respondWithJSON(w, http.StatusOK, response)
}
