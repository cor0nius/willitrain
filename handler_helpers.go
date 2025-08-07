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
	// Step 1: Check for Location in Database First
	dbLocation, err := cfg.dbQueries.GetLocationByName(ctx, cityName)
	if err == nil {
		// Location found in DB, map and return it.
		return databaseLocationToLocation(dbLocation), nil
	}

	if err != sql.ErrNoRows {
		// A different database error occurred.
		return Location{}, fmt.Errorf("database error when fetching location: %w", err)
	}

	// Step 2: Location not found, so geocode it.
	geocodedLocation, geoErr := cfg.Geocode(cityName)
	if geoErr != nil {
		return Location{}, fmt.Errorf("could not geocode city: %w", geoErr)
	}

	// Step 3: Persist the new location to the database.
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
	// Step 1: Check for Cached Weather Data
	dbWeathers, err := cfg.dbQueries.GetCurrentWeatherAtLocation(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error when fetching weather: %w", err)
	}

	if err == nil { // This means sql.ErrNoRows was not returned, so some rows were found.
		var cachedWeather []CurrentWeather
		for _, dbw := range dbWeathers {
			if dbw.UpdatedAt.After(time.Now().UTC().Add(-weatherCacheTTL)) {
				cachedWeather = append(cachedWeather, databaseCurrentWeatherToCurrentWeather(dbw, location))
			}
		}

		// If we have any fresh data, return it.
		// A more robust implementation might fetch only for the stale sources.
		if len(cachedWeather) > 0 {
			return cachedWeather, nil
		}
	}

	// Step 2: Fetch from APIs if Weather Cache is Stale or Empty
	weather, err := cfg.requestCurrentWeather(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch current weather: %w", err)
	}

	// Step 3: Persist New Weather Data using the helper function
	cfg.persistCurrentWeather(ctx, weather)

	return weather, nil
}

// getCachedOrFetchDailyForecast checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchDailyForecast(ctx context.Context, location Location) ([]DailyForecast, error) {
	// Step 1: Check for Cached Weather Data
	// NOTE: This assumes a `GetDailyForecastsByLocation` method exists in dbQueries,
	// which would fetch all forecast entries for a given location. This query needs to be added.
	dbForecasts, err := cfg.dbQueries.GetDailyForecastsByLocation(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error when fetching daily forecast: %w", err)
	}

	if err == nil { // This means sql.ErrNoRows was not returned, so some rows were found.
		var cachedForecasts []DailyForecast
		isCacheFresh := false
		// NOTE: This also assumes the `database.DailyForecast` struct has an `UpdatedAt` field.
		for _, dbf := range dbForecasts {
			// Check if at least one of the forecast entries is fresh.
			if dbf.UpdatedAt.After(time.Now().UTC().Add(-dailyForecastCacheTTL)) {
				isCacheFresh = true
			}
			cachedForecasts = append(cachedForecasts, databaseDailyForecastToDailyForecast(dbf, location))
		}

		// If we have any fresh data, return the whole cached set.
		if isCacheFresh && len(cachedForecasts) > 0 {
			return cachedForecasts, nil
		}
	}

	// Step 2: Fetch from APIs if Cache is Stale or Empty
	forecast, err := cfg.requestDailyForecast(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch daily forecast: %w", err)
	}

	// Step 3: Persist New Weather Data using the helper function
	// This assumes persistDailyForecast will correctly set the new UpdatedAt timestamp.
	cfg.persistDailyForecast(ctx, forecast)

	return forecast, nil
}

// getCachedOrFetchHourlyForecast checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchHourlyForecast(ctx context.Context, location Location) ([]HourlyForecast, error) {
	// Step 1: Check for Cached Weather Data
	// NOTE: This assumes a `GetHourlyForecastsByLocation` method exists in dbQueries,
	// which would fetch all forecast entries for a given location. This query needs to be added.
	dbForecasts, err := cfg.dbQueries.GetHourlyForecastsByLocation(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error when fetching hourly forecast: %w", err)
	}

	if err == nil { // This means sql.ErrNoRows was not returned, so some rows were found.
		var cachedForecasts []HourlyForecast
		isCacheFresh := false
		// NOTE: This also assumes the `database.HourlyForecast` struct has an `UpdatedAt` field.
		for _, dbf := range dbForecasts {
			// Check if at least one of the forecast entries is fresh.
			if dbf.UpdatedAt.After(time.Now().UTC().Add(-hourlyForecastCacheTTL)) {
				isCacheFresh = true
			}
			cachedForecasts = append(cachedForecasts, databaseHourlyForecastToHourlyForecast(dbf, location))
		}

		// If we have any fresh data, return the whole cached set.
		if isCacheFresh && len(cachedForecasts) > 0 {
			return cachedForecasts, nil
		}
	}

	// Step 2: Fetch from APIs if Cache is Stale or Empty
	forecast, err := cfg.requestHourlyForecast(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch hourly forecast: %w", err)
	}

	// Step 3: Persist New Weather Data using the helper function
	// This assumes persistHourlyForecast will correctly set the new UpdatedAt timestamp.
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
			// Record doesn't exist, so create it.
			if _, createErr := createItemFunc(); createErr != nil {
				log.Printf("Error creating cache for %s at %s from %s: %v", logInfo["type"], logInfo["location"], logInfo["api"], createErr)
			}
		} else {
			// A different database error occurred while checking for an existing record.
			log.Printf("Error getting cache for %s at %s from %s: %v", logInfo["type"], logInfo["location"], logInfo["api"], err)
		}
		return
	}

	// Record exists, so update it.
	if _, updateErr := updateItemFunc(existing); updateErr != nil {
		log.Printf("Error updating cache for %s at %s from %s: %v", logInfo["type"], logInfo["location"], logInfo["api"], updateErr)
	}
}

// persistCurrentWeather handles persisting current weather data using the generic upsert helper.
func (cfg *apiConfig) persistCurrentWeather(ctx context.Context, weatherData []CurrentWeather) {
	for _, weather := range weatherData {
		// This closure captures the specific 'weather' item for the functions below.
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
		// This closure captures the specific 'forecast' item for the functions below.
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
		// This closure captures the specific 'forecast' item for the functions below.
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
