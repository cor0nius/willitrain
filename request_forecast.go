package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
)

// This file contains the high-level logic for fetching weather forecasts.
// It orchestrates the process of preparing API requests, fetching data concurrently from multiple sources,
// parsing the responses, and updating the database with new information.

// The request... functions are the main entry points for fetching a specific type of forecast.
// Each function prepares the necessary URLs and provider configurations for its forecast type
// (current, daily, or hourly) and then passes them to the generic processForecastRequests function
// to handle the concurrent API calls. They also handle post-processing, such as updating
// the location's timezone in the database if it's discovered during the fetch.
func (cfg *apiConfig) requestCurrentWeather(location Location) ([]CurrentWeather, error) {
	urls := cfg.WrapForCurrentWeather(location)

	providers := map[string]forecastProvider[CurrentWeather]{
		"gmpWrappedURL": {
			parser:   ParseCurrentWeatherGMP,
			errorVal: CurrentWeather{SourceAPI: "Google Weather API"},
		},
		"owmWrappedURL": {
			parser:   ParseCurrentWeatherOWM,
			errorVal: CurrentWeather{SourceAPI: "OpenWeatherMap API"},
		},
		"ometeoWrappedURL": {
			parser:   ParseCurrentWeatherOMeteo,
			errorVal: CurrentWeather{SourceAPI: "Open-Meteo API"},
		},
	}

	results, tz, err := processForecastRequests(cfg, urls, providers)
	if err != nil {
		return nil, err
	}

	if tz != "" && location.Timezone == "" {
		params := database.UpdateTimezoneParams{
			ID:       location.LocationID,
			Timezone: sql.NullString{String: tz, Valid: true},
		}
		if err := cfg.dbQueries.UpdateTimezone(context.Background(), params); err != nil {
			cfg.logger.Warn("failed to update timezone", "location", location.CityName, "error", err)
		}
	}

	for i := range results {
		results[i].Location = location
	}

	return results, nil
}

func (cfg *apiConfig) requestDailyForecast(location Location) ([]DailyForecast, error) {
	fetchedAt := time.Now().UTC()
	urls := cfg.WrapForDailyForecast(location)

	providers := map[string]forecastProvider[[]DailyForecast]{
		"gmpWrappedURL": {
			parser:   ParseDailyForecastGMP,
			errorVal: []DailyForecast{{SourceAPI: "Google Weather API"}},
		},
		"owmWrappedURL": {
			parser:   ParseDailyForecastOWM,
			errorVal: []DailyForecast{{SourceAPI: "OpenWeatherMap API"}},
		},
		"ometeoWrappedURL": {
			parser:   ParseDailyForecastOMeteo,
			errorVal: []DailyForecast{{SourceAPI: "Open-Meteo API"}},
		},
	}

	results, tz, err := processForecastRequests(cfg, urls, providers)
	if err != nil {
		return nil, err
	}

	if tz != "" && location.Timezone == "" {
		params := database.UpdateTimezoneParams{
			ID:       location.LocationID,
			Timezone: sql.NullString{String: tz, Valid: true},
		}
		if err := cfg.dbQueries.UpdateTimezone(context.Background(), params); err != nil {
			cfg.logger.Warn("failed to update timezone", "location", location.CityName, "error", err)
		}
	}

	var allForecasts []DailyForecast
	for _, forecastSlice := range results {
		allForecasts = append(allForecasts, forecastSlice...)
	}

	for i := range allForecasts {
		allForecasts[i].Location = location
		allForecasts[i].Timestamp = fetchedAt
	}

	return allForecasts, nil
}

func (cfg *apiConfig) requestHourlyForecast(location Location) ([]HourlyForecast, error) {
	fetchedAt := time.Now().UTC()
	urls := cfg.WrapForHourlyForecast(location)

	providers := map[string]forecastProvider[[]HourlyForecast]{
		"gmpWrappedURL": {
			parser:   ParseHourlyForecastGMP,
			errorVal: []HourlyForecast{{SourceAPI: "Google Weather API"}},
		},
		"owmWrappedURL": {
			parser:   ParseHourlyForecastOWM,
			errorVal: []HourlyForecast{{SourceAPI: "OpenWeatherMap API"}},
		},
		"ometeoWrappedURL": {
			parser:   ParseHourlyForecastOMeteo,
			errorVal: []HourlyForecast{{SourceAPI: "Open-Meteo API"}},
		},
	}

	results, tz, err := processForecastRequests(cfg, urls, providers)
	if err != nil {
		return nil, err
	}

	if tz != "" && location.Timezone == "" {
		params := database.UpdateTimezoneParams{
			ID:       location.LocationID,
			Timezone: sql.NullString{String: tz, Valid: true},
		}
		if err := cfg.dbQueries.UpdateTimezone(context.Background(), params); err != nil {
			cfg.logger.Warn("failed to update timezone", "location", location.CityName, "error", err)
		}
	}

	var allForecasts []HourlyForecast
	for _, forecastSlice := range results {
		allForecasts = append(allForecasts, forecastSlice...)
	}

	for i := range allForecasts {
		allForecasts[i].Location = location
		allForecasts[i].Timestamp = fetchedAt
	}

	return allForecasts, nil
}

// processForecastRequests is a generic function that manages the concurrent fetching of forecasts.
// It takes a map of URLs and a corresponding map of providers, launches a goroutine for each,
// waits for them to complete, and then aggregates the results.
func processForecastRequests[T Forecast](
	cfg *apiConfig,
	urls map[string]string,
	providers map[string]forecastProvider[T],
) ([]T, string, error) {
	var wg sync.WaitGroup
	results := make(chan struct {
		t   T
		tz  string
		err error
	}, len(urls))

	for key, url := range urls {
		if provider, ok := providers[key]; ok {
			wg.Add(1)
			go fetchForecastFromAPI(cfg, url, provider.parser, provider.errorVal, &wg, results)
		} else {
			cfg.logger.Error("no provider found for key", "key", key)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []T
	var timezone string
	for res := range results {
		if res.err != nil {
			var sourceAPI string
			v := any(res.t)
			switch v := v.(type) {
			case CurrentWeather:
				sourceAPI = v.SourceAPI
			case []DailyForecast:
				if len(v) > 0 {
					sourceAPI = v[0].SourceAPI
				}
			case []HourlyForecast:
				if len(v) > 0 {
					sourceAPI = v[0].SourceAPI
				}
			}
			if sourceAPI != "" {
				cfg.logger.Warn("error fetching forecast from provider", "provider", sourceAPI, "error", res.err)
			} else {
				cfg.logger.Warn("error fetching forecast from unknown provider", "error", res.err)
			}
		} else {
			allResults = append(allResults, res.t)
			if timezone == "" && res.tz != "" {
				timezone = res.tz
			}
		}
	}

	if len(allResults) == 0 {
		cfg.logger.Error("all forecast fetches failed")
		return nil, "", errors.New("all forecast fetches failed")
	}

	return allResults, timezone, nil
}

// forecastProvider is a helper struct that bundles a parser function with its corresponding zero-value.
// This allows the generic fetcher to know which parser to use for a given API response.
type forecastProvider[T Forecast] struct {
	parser   func(io.Reader, *slog.Logger) (T, string, error)
	errorVal T
}
