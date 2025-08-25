package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// This file contains helper functions related to the application's multi-layered
// caching strategy. It includes a generic function for fetching data from the cache
// (Redis or DB) or from external APIs as a final fallback.

// Cache TTL constants define the duration for which different types of data are considered fresh.
const weatherCacheTTL = 10 * time.Minute      // For the database cache of current weather.
const dailyForecastCacheTTL = 12 * time.Hour  // For the database cache of daily forecasts.
const hourlyForecastCacheTTL = 1 * time.Hour  // For the database cache of hourly forecasts.

// Redis cache TTLs are set slightly shorter than the scheduler intervals to prevent serving
// stale data just before a scheduled update.
const redisCurrentWeatherCacheTTL = 9 * time.Minute
const redisDailyForecastCacheTTL = 11*time.Hour + 55*time.Minute
const redisHourlyForecastCacheTTL = 55 * time.Minute

// getCachedOrFetch is a generic helper that abstracts the caching logic for different weather types.
// It implements a multi-layered caching strategy:
// 1. It first checks the Redis cache for fresh data.
// 2. If Redis is a miss or the data is invalid, it checks the PostgreSQL database.
// 3. If the database data is also stale or missing, it fetches fresh data from the external APIs.
// 4. After a successful API fetch, it updates both the database and the Redis cache.
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
	isValidCache func([]T) bool,
) ([]T, error) {
	cacheKey := fmt.Sprintf("%s:%s", cacheKeyPrefix, location.LocationID.String())
	cachedData, err := cfg.cache.Get(ctx, cacheKey)
	if err == nil {
		var items []T
		jsonErr := json.Unmarshal([]byte(cachedData), &items)
		if jsonErr == nil && isValidCache(items) {
			cfg.logger.Debug("cache hit", "key", cacheKey)
			return items, nil
		}
		if jsonErr != nil {
			cfg.logger.Warn("invalid cache entry: unmarshal error", "key", cacheKey, "error", jsonErr)
		} else {
			cfg.logger.Warn("invalid cache entry: validation failed", "key", cacheKey, "actual_count", len(items))
		}
	} else if err != redis.Nil {
		cfg.logger.Warn("error getting from redis", "key", cacheKey, "error", err)
	}

	dbItems, err := dbFetcher(ctx, location.LocationID)
	if err != nil && err != sql.ErrNoRows { // sql.ErrNoRows is handled gracefully
		return nil, fmt.Errorf("database error when fetching %s: %w", cacheKeyPrefix, err)
	}

	if err == nil {
		var freshItems []T
		for _, dbi := range dbItems {
			if getTimestamp(dbi).After(time.Now().UTC().Add(-dbCacheTTL)) {
				freshItems = append(freshItems, modelConverter(dbi, location))
			}
		}

		if isValidCache(freshItems) {
			cfg.logger.Debug("db cache hit", "key", cacheKey)
			if cacheErr := cfg.cache.Set(ctx, cacheKey, freshItems, redisCacheTTL); cacheErr != nil {
				cfg.logger.Warn("error setting to redis", "key", cacheKey, "error", cacheErr)
			}
			return freshItems, nil
		}
	}

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

// The getCachedOrFetch... functions are specific implementations of the generic getCachedOrFetch helper.
// Each one is tailored for a specific forecast type (current, daily, or hourly) by providing the
// appropriate cache keys, TTLs, and data fetching/conversion functions.
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
		func(items []CurrentWeather) bool {
			return len(items) == 3
		},
	)
}

func (cfg *apiConfig) getCachedOrFetchDailyForecast(ctx context.Context, location Location) ([]DailyForecast, error) {
	dbFetcher := func(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error) {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		return cfg.dbQueries.GetUpcomingDailyForecastsAtLocation(ctx, database.GetUpcomingDailyForecastsAtLocationParams{
			LocationID:    locationID,
			ForecastDate:  today,
		})
	}

	return getCachedOrFetch(
		cfg,
		ctx,
		location,
		"dailyforecast",
		dailyForecastCacheTTL,
		redisDailyForecastCacheTTL,
		dbFetcher,
		cfg.requestDailyForecast,
		cfg.persistDailyForecast,
		databaseDailyForecastToDailyForecast,
		func(d database.DailyForecast) time.Time {
			return d.UpdatedAt
		},
		func(items []DailyForecast) bool {
			return len(items) > 0
		},
	)
}

func (cfg *apiConfig) getCachedOrFetchHourlyForecast(ctx context.Context, location Location) ([]HourlyForecast, error) {
	dbFetcher := func(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error) {
		return cfg.dbQueries.GetUpcomingHourlyForecastsAtLocation(ctx, database.GetUpcomingHourlyForecastsAtLocationParams{
			LocationID:          locationID,
			ForecastDatetimeUtc: time.Now().UTC(),
		})
	}

	return getCachedOrFetch(
		cfg,
		ctx,
		location,
		"hourlyforecast",
		hourlyForecastCacheTTL,
		redisHourlyForecastCacheTTL,
		dbFetcher,
		cfg.requestHourlyForecast,
		cfg.persistHourlyForecast,
		databaseHourlyForecastToHourlyForecast,
		func(d database.HourlyForecast) time.Time {
			return d.UpdatedAt
		},
		func(items []HourlyForecast) bool {
			return len(items) > 0
		},
	)
}
