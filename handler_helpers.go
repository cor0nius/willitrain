package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
)

const weatherCacheTTL = 15 * time.Minute
const dailyForecastCacheTTL = 24 * time.Hour
const hourlyForecastCacheTTL = 1 * time.Hour

// getOrCreateLocation checks if a location exists in the database by name.
// If it exists, it returns the location data from the DB.
// If not, it calls the geocoding API to get the location details,
// persists the new location to the DB, and then returns the new location data.
func (cfg *apiConfig) getOrCreateLocation(ctx context.Context, cityName string) (Location, error) {
	dbLocation, err := cfg.dbQueries.GetLocationByName(ctx, cityName)
	if err == nil {
		return databaseLocationToLocation(dbLocation), nil
	}

	if err != sql.ErrNoRows {
		return Location{}, fmt.Errorf("database error when fetching location: %w", err)
	}

	geocodedLocation, geoErr := cfg.Geocode(cityName)
	if geoErr != nil {
		return Location{}, fmt.Errorf("could not geocode city: %w", geoErr)
	}

	persistedLocation, createErr := cfg.dbQueries.CreateLocation(ctx, locationToCreateLocationParams(geocodedLocation))
	if createErr != nil {
		log.Printf("Could not persist new location %s: %v", cityName, createErr)
	} else {
		geocodedLocation.LocationID = persistedLocation.ID
	}

	return geocodedLocation, nil
}

// getCachedOrFetchCurrentWeather checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchCurrentWeather(ctx context.Context, location Location) ([]CurrentWeather, error) {
	dbWeathers, err := cfg.dbQueries.GetCurrentWeatherAtLocation(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error when fetching weather: %w", err)
	}

	if err == nil {
		var cachedWeather []CurrentWeather
		for _, dbw := range dbWeathers {
			if dbw.UpdatedAt.After(time.Now().UTC().Add(-weatherCacheTTL)) {
				cachedWeather = append(cachedWeather, databaseCurrentWeatherToCurrentWeather(dbw, location))
			}
		}
		if len(cachedWeather) > 0 {
			return cachedWeather, nil
		}
	}

	weather, err := cfg.requestCurrentWeather(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch current weather: %w", err)
	}

	cfg.persistCurrentWeather(ctx, weather)

	return weather, nil
}

// getCachedOrFetchDailyForecast checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchDailyForecast(ctx context.Context, location Location) ([]DailyForecast, error) {
	dbForecasts, err := cfg.dbQueries.GetAllDailyForecastsAtLocation(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error when fetching daily forecast: %w", err)
	}

	if err == nil {
		var cachedForecasts []DailyForecast
		isCacheFresh := false
		for _, dbf := range dbForecasts {
			if dbf.UpdatedAt.After(time.Now().UTC().Add(-dailyForecastCacheTTL)) {
				isCacheFresh = true
			}
			cachedForecasts = append(cachedForecasts, databaseDailyForecastToDailyForecast(dbf, location))
		}

		if isCacheFresh && len(cachedForecasts) > 0 {
			return cachedForecasts, nil
		}
	}

	forecast, err := cfg.requestDailyForecast(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch daily forecast: %w", err)
	}

	cfg.persistDailyForecast(ctx, forecast)

	return forecast, nil
}

// getCachedOrFetchHourlyForecast checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchHourlyForecast(ctx context.Context, location Location) ([]HourlyForecast, error) {
	dbForecasts, err := cfg.dbQueries.GetAllHourlyForecastsAtLocation(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error when fetching hourly forecast: %w", err)
	}

	if err == nil {
		var cachedForecasts []HourlyForecast
		isCacheFresh := false
		for _, dbf := range dbForecasts {
			if dbf.UpdatedAt.After(time.Now().UTC().Add(-hourlyForecastCacheTTL)) {
				isCacheFresh = true
			}
			cachedForecasts = append(cachedForecasts, databaseHourlyForecastToHourlyForecast(dbf, location))
		}

		if isCacheFresh && len(cachedForecasts) > 0 {
			return cachedForecasts, nil
		}
	}

	forecast, err := cfg.requestHourlyForecast(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch hourly forecast: %w", err)
	}

	cfg.persistHourlyForecast(ctx, forecast)

	return forecast, nil
}

// upsertWeatherItem is a generic helper for the "upsert" (update or insert) logic.
// It takes functions as arguments to perform the specific DB operations,
// allowing it to be reused for current, daily, and hourly weather data.
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
			if _, createErr := createItemFunc(); createErr != nil {
				log.Printf("Error creating cache for %s at %s from %s: %v", logInfo["type"], logInfo["location"], logInfo["api"], createErr)
			}
		} else {
			log.Printf("Error getting cache for %s at %s from %s: %v", logInfo["type"], logInfo["location"], logInfo["api"], err)
		}
		return
	}

	if _, updateErr := updateItemFunc(existing); updateErr != nil {
		log.Printf("Error updating cache for %s at %s from %s: %v", logInfo["type"], logInfo["location"], logInfo["api"], updateErr)
	}
}

// persistCurrentWeather handles persisting current weather data using the generic upsert helper.
func (cfg *apiConfig) persistCurrentWeather(ctx context.Context, weatherData []CurrentWeather) {
	for _, weather := range weatherData {
		cfg.upsertWeatherItem(ctx,
			func() (any, error) { // getItem
				return cfg.dbQueries.GetCurrentWeatherAtLocationFromAPI(ctx, database.GetCurrentWeatherAtLocationFromAPIParams{
					LocationID: weather.Location.LocationID,
					SourceApi:  weather.SourceAPI,
				})
			},
			func() (any, error) { // createItem
				return cfg.dbQueries.CreateCurrentWeather(ctx, currentWeatherToCreateCurrentWeatherParams(weather))
			},
			func(existing any) (any, error) { // updateItem
				existingWeather, ok := existing.(database.CurrentWeather)
				if !ok {
					return nil, fmt.Errorf("unexpected type for existing item: %T", existing)
				}
				return cfg.dbQueries.UpdateCurrentWeather(ctx, currentWeatherToUpdateCurrentWeatherParams(weather, existingWeather.ID))
			},
			map[string]string{ // logInfo
				"location": weather.Location.CityName,
				"api":      weather.SourceAPI,
				"type":     "current weather",
			},
		)
	}
}

// persistDailyForecast handles persisting daily forecast data using the generic upsert helper.
func (cfg *apiConfig) persistDailyForecast(ctx context.Context, forecastData []DailyForecast) {
	for _, forecast := range forecastData {
		cfg.upsertWeatherItem(ctx,
			func() (any, error) { // getItem
				return cfg.dbQueries.GetDailyForecastAtLocationAndDateFromAPI(ctx, database.GetDailyForecastAtLocationAndDateFromAPIParams{
					LocationID:   forecast.Location.LocationID,
					ForecastDate: forecast.ForecastDate,
					SourceApi:    forecast.SourceAPI,
				})
			},
			func() (any, error) { // createItem
				return cfg.dbQueries.CreateDailyForecast(ctx, dailyForecastToCreateDailyForecastParams(forecast))
			},
			func(existing any) (any, error) { // updateItem
				existingForecast, ok := existing.(database.DailyForecast)
				if !ok {
					return nil, fmt.Errorf("unexpected type for existing item: %T", existing)
				}
				return cfg.dbQueries.UpdateDailyForecast(ctx, dailyForecastToUpdateDailyForecastParams(forecast, existingForecast.ID))
			},
			map[string]string{ // logInfo
				"location": forecast.Location.CityName,
				"api":      forecast.SourceAPI,
				"type":     "daily forecast",
			},
		)
	}
}

// persistHourlyForecast handles persisting hourly forecast data using the generic upsert helper.
func (cfg *apiConfig) persistHourlyForecast(ctx context.Context, forecastData []HourlyForecast) {
	for _, forecast := range forecastData {
		cfg.upsertWeatherItem(ctx,
			func() (any, error) { // getItem
				return cfg.dbQueries.GetHourlyForecastAtLocationAndTimeFromAPI(ctx, database.GetHourlyForecastAtLocationAndTimeFromAPIParams{
					LocationID:          forecast.Location.LocationID,
					ForecastDatetimeUtc: forecast.ForecastDateTime,
					SourceApi:           forecast.SourceAPI,
				})
			},
			func() (any, error) { // createItem
				return cfg.dbQueries.CreateHourlyForecast(ctx, hourlyForecastToCreateHourlyForecastParams(forecast))
			},
			func(existing any) (any, error) { // updateItem
				existingForecast, ok := existing.(database.HourlyForecast)
				if !ok {
					return nil, fmt.Errorf("unexpected type for existing item: %T", existing)
				}
				return cfg.dbQueries.UpdateHourlyForecast(ctx, hourlyForecastToUpdateHourlyForecastParams(forecast, existingForecast.ID))
			},
			map[string]string{ // logInfo
				"location": forecast.Location.CityName,
				"api":      forecast.SourceAPI,
				"type":     "hourly forecast",
			},
		)
	}
}
