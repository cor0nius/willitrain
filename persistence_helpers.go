package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cor0nius/willitrain/internal/database"
)

// This file contains helper functions for persisting data to the database.
// It includes a generic "upsert" (update or insert) function and specific
// implementations for each type of weather data.

// upsertWeatherItem is a generic helper for the "upsert" (update or insert) logic.
// It abstracts the common pattern of checking if a database record exists, and then either
// updating it or creating a new one. This is used to persist weather data.
func (cfg *apiConfig) upsertWeatherItem(
	ctx context.Context,
	getItemFunc func() (any, error),
	createItemFunc func() (any, error),
	updateItemFunc func(existingItem any) (any, error),
	logInfo map[string]string,
) {
	existing, err := getItemFunc()
	if err != nil {
		if err == sql.ErrNoRows {
			_, createErr := createItemFunc()
			if createErr != nil {
				cfg.logger.Error("error creating cache", "type", logInfo["type"], "location", logInfo["location"], "api", logInfo["api"], "error", createErr)
			} else {
				cfg.logger.Debug("created cache item", "type", logInfo["type"], "location", logInfo["location"], "api", logInfo["api"])
			}
		} else {
			cfg.logger.Error("error getting cache", "type", logInfo["type"], "location", logInfo["location"], "api", logInfo["api"], "error", err)
		}
		return
	}

	if _, updateErr := updateItemFunc(existing); updateErr != nil {
		cfg.logger.Error("error updating cache", "type", logInfo["type"], "location", logInfo["location"], "api", logInfo["api"], "error", updateErr)
	} else {
		cfg.logger.Debug("updated cache item", "type", logInfo["type"], "location", logInfo["location"], "api", logInfo["api"])
	}
}

// The persist... functions use the generic upsertWeatherItem helper to save weather data to the database.
// Each function is specific to a forecast type and provides the necessary getItem, createItem, and updateItem
// functions to the upsert helper.
func (cfg *apiConfig) persistCurrentWeather(ctx context.Context, weatherData []CurrentWeather) {
	for _, weather := range weatherData {
		cfg.upsertWeatherItem(ctx,
			func() (any, error) {
				return cfg.dbQueries.GetCurrentWeatherAtLocationFromAPI(ctx, database.GetCurrentWeatherAtLocationFromAPIParams{
					LocationID: weather.Location.LocationID,
					SourceApi:  weather.SourceAPI,
				})
			},
			func() (any, error) {
				return cfg.dbQueries.CreateCurrentWeather(ctx, currentWeatherToCreateCurrentWeatherParams(weather))
			},
			func(existing any) (any, error) {
				existingWeather, ok := existing.(database.CurrentWeather)
				if !ok {
					return nil, fmt.Errorf("unexpected type for existing item: %T", existing)
				}
				return cfg.dbQueries.UpdateCurrentWeather(ctx, currentWeatherToUpdateCurrentWeatherParams(weather, existingWeather.ID))
			},
			map[string]string{
				"location": weather.Location.CityName,
				"api":      weather.SourceAPI,
				"type":     "current weather",
			},
		)
	}
}

func (cfg *apiConfig) persistDailyForecast(ctx context.Context, forecastData []DailyForecast) {
	for _, forecast := range forecastData {
		cfg.upsertWeatherItem(ctx,
			func() (any, error) {
				return cfg.dbQueries.GetDailyForecastAtLocationAndDateFromAPI(ctx, database.GetDailyForecastAtLocationAndDateFromAPIParams{
					LocationID:   forecast.Location.LocationID,
					ForecastDate: forecast.ForecastDate,
					SourceApi:    forecast.SourceAPI,
				})
			},
			func() (any, error) {
				return cfg.dbQueries.CreateDailyForecast(ctx, dailyForecastToCreateDailyForecastParams(forecast))
			},
			func(existing any) (any, error) {
				existingForecast, ok := existing.(database.DailyForecast)
				if !ok {
					return nil, fmt.Errorf("unexpected type for existing item: %T", existing)
				}
				return cfg.dbQueries.UpdateDailyForecast(ctx, dailyForecastToUpdateDailyForecastParams(forecast, existingForecast.ID))
			},
			map[string]string{
				"location": forecast.Location.CityName,
				"api":      forecast.SourceAPI,
				"type":     "daily forecast",
			},
		)
	}
}

func (cfg *apiConfig) persistHourlyForecast(ctx context.Context, forecastData []HourlyForecast) {
	for _, forecast := range forecastData {
		cfg.upsertWeatherItem(ctx,
			func() (any, error) {
				return cfg.dbQueries.GetHourlyForecastAtLocationAndTimeFromAPI(ctx, database.GetHourlyForecastAtLocationAndTimeFromAPIParams{
					LocationID:          forecast.Location.LocationID,
					ForecastDatetimeUtc: forecast.ForecastDateTime,
					SourceApi:           forecast.SourceAPI,
				})
			},
			func() (any, error) {
				return cfg.dbQueries.CreateHourlyForecast(ctx, hourlyForecastToCreateHourlyForecastParams(forecast))
			},
			func(existing any) (any, error) {
				existingForecast, ok := existing.(database.HourlyForecast)
				if !ok {
					return nil, fmt.Errorf("unexpected type for existing item: %T", existing)
				}
				return cfg.dbQueries.UpdateHourlyForecast(ctx, hourlyForecastToUpdateHourlyForecastParams(forecast, existingForecast.ID))
			},
			map[string]string{
				"location": forecast.Location.CityName,
				"api":      forecast.SourceAPI,
				"type":     "hourly forecast",
			},
		)
	}
}
