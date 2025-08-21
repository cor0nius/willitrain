package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

// getOrCreateLocation is an intelligent helper to retrieve a location from the database.
// It handles city name aliases to avoid duplicate entries and minimize external API calls.
//
// The logic is as follows:
// 1. Normalize the input cityName to create a standardized alias.
// 2. Attempt to find the location using this alias in the `location_aliases` table.
// 3. If found, return the location.
// 4. If not found, call the geocoding service to get the canonical location data.
// 5. Check if a location with the canonical name already exists in the `locations` table.
// 6. If it exists, create a new alias for the user's original input and link it to the existing location.
// 7. If no location exists by either alias or canonical name, create a new location record.
// 8. Finally, create aliases for both the user's normalized input and the canonical name to ensure future lookups are successful.
func (cfg *apiConfig) getOrCreateLocation(ctx context.Context, cityName string) (Location, error) {
	alias, err := normalizeCityName(cityName)
	if err != nil {
		return Location{}, fmt.Errorf("could not normalize city name: %w", err)
	}

	dbLocation, err := cfg.dbQueries.GetLocationByAlias(ctx, alias)
	if err == nil {
		cfg.logger.Debug("location found by alias", "alias", alias, "city", dbLocation.CityName)
		return databaseLocationToLocation(dbLocation), nil
	}
	if err != sql.ErrNoRows {
		return Location{}, fmt.Errorf("database error when fetching location by alias: %w", err)
	}

	cfg.logger.Debug("alias not found, geocoding", "alias", alias, "original_city", cityName)
	geocodedLocation, geoErr := cfg.geocoder.Geocode(cityName)
	if geoErr != nil {
		return Location{}, fmt.Errorf("could not geocode city '%s': %w", cityName, geoErr)
	}

	dbLocation, err = cfg.dbQueries.GetLocationByName(ctx, geocodedLocation.CityName)
	if err == nil {
		cfg.logger.Debug("canonical location found in db, creating new alias", "city", dbLocation.CityName, "alias", alias)
		_, aliasErr := cfg.dbQueries.CreateLocationAlias(ctx, database.CreateLocationAliasParams{Alias: alias, LocationID: dbLocation.ID})
		if aliasErr != nil {
			cfg.logger.Warn("could not create location alias", "alias", alias, "location_id", dbLocation.ID, "error", aliasErr)
		}
		return databaseLocationToLocation(dbLocation), nil
	}
	if err != sql.ErrNoRows {
		return Location{}, fmt.Errorf("database error when fetching location by canonical name: %w", err)
	}

	cfg.logger.Debug("no location found, creating new location and aliases", "city", geocodedLocation.CityName)
	persistedLocation, createErr := cfg.dbQueries.CreateLocation(ctx, locationToCreateLocationParams(geocodedLocation))
	if createErr != nil {
		return Location{}, fmt.Errorf("could not persist new location: %w", createErr)
	}

	_, aliasErr := cfg.dbQueries.CreateLocationAlias(ctx, database.CreateLocationAliasParams{Alias: alias, LocationID: persistedLocation.ID})
	if aliasErr != nil {
		cfg.logger.Warn("could not create user input alias", "alias", alias, "location_id", persistedLocation.ID, "error", aliasErr)
	}

	canonicalAlias, err := normalizeCityName(persistedLocation.CityName)
	if err != nil {
		cfg.logger.Error("could not normalize canonical city name", "city", persistedLocation.CityName, "error", err)
	} else if alias != canonicalAlias {
		_, aliasErr = cfg.dbQueries.CreateLocationAlias(ctx, database.CreateLocationAliasParams{Alias: canonicalAlias, LocationID: persistedLocation.ID})
		if aliasErr != nil {
			cfg.logger.Warn("could not create canonical alias", "alias", canonicalAlias, "location_id", persistedLocation.ID, "error", aliasErr)
		}
	}

	return databaseLocationToLocation(persistedLocation), nil
}

func (cfg *apiConfig) getLocationFromRequest(r *http.Request) (Location, error) {
	ctx := r.Context()
	cityName := r.URL.Query().Get("city")
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if cityName != "" {
		return cfg.getOrCreateLocation(ctx, cityName)
	}

	if latStr != "" && lonStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return Location{}, fmt.Errorf("invalid latitude: %v", err)
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return Location{}, fmt.Errorf("invalid longitude: %v", err)
		}

		// Reverse geocode to get a city name
		location, err := cfg.geocoder.ReverseGeocode(lat, lon)
		if err != nil {
			return Location{}, fmt.Errorf("could not reverse geocode coordinates: %w", err)
		}

		// Use the same unified logic to get or create the location
		return cfg.getOrCreateLocation(ctx, location.CityName)
	}

	return Location{}, fmt.Errorf("either city or lat/lon query parameters are required")
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
			cfg.logger.Debug("cache hit", "key", cacheKey)
			switch v := any(&items).(type) {
			case *[]CurrentWeather:
				for i := range *v {
					(*v)[i].Location = location
				}
			case *[]DailyForecast:
				for i := range *v {
					(*v)[i].Location = location
				}
			case *[]HourlyForecast:
				for i := range *v {
					(*v)[i].Location = location
				}
			}
			return items, nil
		}
		cfg.logger.Warn("error unmarshalling from redis", "key", cacheKey, "error", jsonErr)
	} else if err != redis.Nil {
		cfg.logger.Warn("error getting from redis", "key", cacheKey, "error", err)
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
			cfg.logger.Debug("db cache hit", "key", cacheKey)
			if cacheErr := cfg.cache.Set(ctx, cacheKey, freshItems, redisCacheTTL); cacheErr != nil {
				cfg.logger.Warn("error setting to redis", "key", cacheKey, "error", cacheErr)
			}
			return freshItems, nil
		}
	}

	// 3. Fetch from API
	apiItems, err := apiFetcher(location)
	if err != nil {
		return nil, fmt.Errorf("could not fetch %s: %w", cacheKeyPrefix, err)
	}
	cfg.logger.Debug("api fetch successful", "key", cacheKey)

	persister(ctx, apiItems)
	if cacheErr := cfg.cache.Set(ctx, cacheKey, apiItems, redisCacheTTL); cacheErr != nil {
		cfg.logger.Warn("error setting to redis after api fetch", "key", cacheKey, "error", cacheErr)
	} else {
		cfg.logger.Debug("set to cache", "key", cacheKey)
	}

	return apiItems, nil
}

// getCachedOrFetchCurrentWeather checks for fresh cached data and fetches from APIs if it's stale or missing.
func (cfg *apiConfig) getCachedOrFetchCurrentWeather(ctx context.Context, location Location) ([]CurrentWeather, error) {
	return getCachedOrFetch(
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
