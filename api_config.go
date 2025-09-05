package main

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// apiConfig serves as the application's dependency injection container.
// It holds all runtime dependencies, such as database connections, external API clients,
// and configuration values. This struct is passed as a receiver to most high-level
// functions, providing them with the necessary context to operate without relying on
// global state. This design improves testability and clarifies dependencies.
type apiConfig struct {
	dbURL                    string
	redisURL                 string
	geocoder                 GeocodingService
	gmpWeatherURL            string
	owmWeatherURL            string
	ometeoWeatherURL         string
	gmpKey                   string
	owmKey                   string
	httpClient               *http.Client
	schedulerCurrentInterval time.Duration
	schedulerHourlyInterval  time.Duration
	schedulerDailyInterval   time.Duration
	port                     string
	devMode                  bool
	logger                   *slog.Logger
	newDBClientFunc          func(driverName, dataSourceName string) (*sql.DB, error)
	dbQueries                dbQuerier
	newCacheClientFunc       func(opt *redis.Options) *redis.Client
	cache                    Cache
}

// getRequiredEnv provides a safe way to read a mandatory environment variable.
// It ensures that the application will not start without critical configuration,
// logging a fatal error and exiting if the variable is not found or is empty.
// This prevents runtime errors due to missing configuration.
func getRequiredEnv(key string, logger *slog.Logger) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		logger.Error("environment variable must be set and not empty", "key", key)
		return "", errors.New("missing required environment variable: " + key)
	}
	return val, nil
}

// getEnv provides a safe way to read an optional environment variable with a fallback.
// This is used for non-critical configuration where a default value is acceptable,
// making the application more flexible and easier to configure.
func getEnv(key, fallback string, logger *slog.Logger) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	logger.Info("environment variable not set, using fallback", "key", key, "fallback", fallback)
	return fallback
}

// getEnvAsInt provides a safe way to read an optional integer environment variable.
// It handles parsing and provides a fallback value if the variable is not set or is
// invalid, preventing configuration errors from crashing the application.
func getEnvAsInt(key string, fallback int, logger *slog.Logger) int {
	valStr := getEnv(key, strconv.Itoa(fallback), logger)
	val, err := strconv.Atoi(valStr)
	if err != nil {
		logger.Warn("invalid integer value for environment variable, using fallback", "key", key, "value", valStr, "error", err)
		return fallback
	}
	return val
}

// config is the application's configuration hub and initialization function.
// It orchestrates the entire setup process by:
//  1. Loading environment variables from a .env file for local development.
//  2. Establishing and verifying connections to the database (PostgreSQL) and cache (Redis).
//  3. Initializing service clients for external APIs (e.g., geocoding).
//  4. Assembling all runtime parameters and dependencies into a single, fully populated
//     apiConfig struct.
//
// This function ensures the application is in a valid state before it starts serving requests.
func NewAPIConfig(output io.Writer) (*apiConfig, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("could not load .env file, proceeding with environment variables")
	}

	devModeStr := os.Getenv("DEV_MODE")
	devMode, err := strconv.ParseBool(devModeStr)
	if err != nil {
		devMode = false
	}

	var logger *slog.Logger
	if devMode {
		logger = slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(output, nil))
	}

	cfg := &apiConfig{
		logger: logger,
	}

	dbURL, err := getRequiredEnv("DB_URL", logger)
	if err != nil {
		return cfg, err
	}

	redisURL, err := getRequiredEnv("REDIS_URL", logger)
	if err != nil {
		return cfg, err
	}

	gmpKey, err := getRequiredEnv("GMP_KEY", logger)
	if err != nil {
		return cfg, err
	}

	gmpGeocodeURL, err := getRequiredEnv("GMP_GEOCODE_URL", logger)
	if err != nil {
		return cfg, err
	}

	gmpWeatherURL, err := getRequiredEnv("GMP_WEATHER_URL", logger)
	if err != nil {
		return cfg, err
	}

	owmWeatherURL, err := getRequiredEnv("OWM_WEATHER_URL", logger)
	if err != nil {
		return cfg, err
	}

	ometeoWeatherURL, err := getRequiredEnv("OMETEO_WEATHER_URL", logger)
	if err != nil {
		return cfg, err
	}

	owmKey, err := getRequiredEnv("OWM_KEY", logger)
	if err != nil {
		return cfg, err
	}

	currentIntervalMin := getEnvAsInt("CURRENT_INTERVAL_MIN", 10, logger)
	hourlyIntervalMin := getEnvAsInt("HOURLY_INTERVAL_MIN", 60, logger)
	dailyIntervalMin := getEnvAsInt("DAILY_INTERVAL_MIN", 720, logger)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &metricsTransport{
			wrapped: http.DefaultTransport,
		},
	}

	geocoder := NewGmpGeocodingService(gmpKey, gmpGeocodeURL, httpClient)

	cfg.dbURL = dbURL
	cfg.redisURL = redisURL
	cfg.geocoder = geocoder
	cfg.gmpWeatherURL = gmpWeatherURL
	cfg.owmWeatherURL = owmWeatherURL
	cfg.ometeoWeatherURL = ometeoWeatherURL
	cfg.gmpKey = gmpKey
	cfg.owmKey = owmKey
	cfg.httpClient = httpClient
	cfg.schedulerCurrentInterval = time.Duration(currentIntervalMin) * time.Minute
	cfg.schedulerHourlyInterval = time.Duration(hourlyIntervalMin) * time.Minute
	cfg.schedulerDailyInterval = time.Duration(dailyIntervalMin) * time.Minute
	cfg.port = getEnv("PORT", "8080", logger)
	cfg.devMode = devMode
	cfg.newDBClientFunc = sql.Open
	cfg.newCacheClientFunc = redis.NewClient

	return cfg, nil
}
