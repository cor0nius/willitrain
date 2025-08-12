package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const weatherCacheTTL = 10 * time.Minute
const dailyForecastCacheTTL = 12 * time.Hour
const hourlyForecastCacheTTL = 1 * time.Hour

const redisCurrentWeatherCacheTTL = 9 * time.Minute
const redisDailyForecastCacheTTL = 11*time.Hour + 55*time.Minute
const redisHourlyForecastCacheTTL = 55 * time.Minute

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

type apiModel interface {
	CurrentWeather | DailyForecast | HourlyForecast
}

type dbModel interface {
	database.CurrentWeather | database.DailyForecast | database.HourlyForecast
}

// getCachedOrFetch is a generic helper that abstracts the caching logic for different weather types.
// It checks Redis, then the DB, and finally fetches from the API if necessary.
func getCachedOrFetch[T apiModel, D dbModel](
	cfg *apiConfig,
	ctx context.Context,
	location Location,
	cacheKeyPrefix string,
	dbCacheTTL time.Duration,
	redisCacheTTL time.Duration,
	dbFetcher func(context.Context, uuid.UUID) ([]D, error),
	apiFetcher func(Location) ([]T, error),
	persister func(context.Context, []T),
	modelConverter func(D, Location) T,
	getTimestamp func(D) time.Time,
) ([]T, error) {
	// 1. Check Redis cache
	cacheKey := fmt.Sprintf("%s:%s", cacheKeyPrefix, location.LocationID.String())
	cachedData, err := cfg.cache.Get(ctx, cacheKey)
	if err == nil {
		var items []T
		jsonErr := json.Unmarshal([]byte(cachedData), &items)
		if jsonErr == nil {
			return items, nil
		}
		log.Printf("Error unmarshalling %s from Redis: %v", cacheKeyPrefix, jsonErr)
	} else if err != redis.Nil {
		log.Printf("Error getting %s from Redis: %v", cacheKeyPrefix, err)
	}

	// 2. Check Database cache
	dbItems, err := dbFetcher(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error when fetching %s: %w", cacheKeyPrefix, err)
	}

	if err == nil && len(dbItems) > 0 {
		var freshItems []T
		for _, dbi := range dbItems {
			if getTimestamp(dbi).After(time.Now().UTC().Add(-dbCacheTTL)) {
				freshItems = append(freshItems, modelConverter(dbi, location))
			}
		}

		if len(freshItems) > 0 {
			if cacheErr := cfg.cache.Set(ctx, cacheKey, freshItems, redisCacheTTL); cacheErr != nil {
				log.Printf("Error setting %s to Redis: %v", cacheKeyPrefix, cacheErr)
			}
			return freshItems, nil
		}
	}

	// 3. Fetch from API
	apiItems, err := apiFetcher(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch %s: %w", cacheKeyPrefix, err)
	}

	persister(ctx, apiItems)
	if cacheErr := cfg.cache.Set(ctx, cacheKey, apiItems, redisCacheTTL); cacheErr != nil {
		log.Printf("Error setting %s to Redis after API fetch: %v", cacheKeyPrefix, cacheErr)
	}

	return apiItems, nil
}

// getCachedOrFetchCurrentWeather checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchCurrentWeather(ctx context.Context, location Location) ([]CurrentWeather, error) {
	return getCachedOrFetch[CurrentWeather, database.CurrentWeather](
		cfg,
		ctx,
		location,
		"currentweather",
		weatherCacheTTL,
		redisCurrentWeatherCacheTTL,
		cfg.dbQueries.GetCurrentWeatherAtLocation,
		cfg.requestCurrentWeather,
		cfg.persistCurrentWeather,
		databaseCurrentWeatherToCurrentWeather,
		func(d database.CurrentWeather) time.Time {
			return d.UpdatedAt
		},
	)
}

// getCachedOrFetchDailyForecast checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchDailyForecast(ctx context.Context, location Location) ([]DailyForecast, error) {
	return getCachedOrFetch(
		cfg,
		ctx,
		location,
		"dailyforecast",
		dailyForecastCacheTTL,
		redisDailyForecastCacheTTL,
		cfg.dbQueries.GetAllDailyForecastsAtLocation,
		cfg.requestDailyForecast,
		cfg.persistDailyForecast,
		databaseDailyForecastToDailyForecast,
		func(d database.DailyForecast) time.Time {
			return d.UpdatedAt
		},
	)
}

// getCachedOrFetchHourlyForecast checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchHourlyForecast(ctx context.Context, location Location) ([]HourlyForecast, error) {
	return getCachedOrFetch(
		cfg,
		ctx,
		location,
		"hourlyforecast",
		hourlyForecastCacheTTL,
		redisHourlyForecastCacheTTL,
		cfg.dbQueries.GetAllHourlyForecastsAtLocation,
		cfg.requestHourlyForecast,
		cfg.persistHourlyForecast,
		databaseHourlyForecastToHourlyForecast,
		func(d database.HourlyForecast) time.Time {
			return d.UpdatedAt
		},
	)
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
