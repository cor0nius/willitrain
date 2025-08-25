package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// apiConfig holds all the dependencies and configuration required by the application.
// This includes database connections, API clients, and other settings.
type apiConfig struct {
	dbQueries                dbQuerier
	cache                    Cache
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
}

// getRequiredEnv retrieves an environment variable by key, and fatals if it's not set.
func getRequiredEnv(key string, logger *slog.Logger) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		logger.Error("environment variable must be set", "key", key)
		os.Exit(1)
	}
	return val
}

// getEnv retrieves an environment variable by key, with a fallback value.
func getEnv(key, fallback string, logger *slog.Logger) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	logger.Info("environment variable not set, using fallback", "key", key, "fallback", fallback)
	return fallback
}

// getEnvAsInt retrieves an environment variable as an integer, with a fallback value.
func getEnvAsInt(key string, fallback int, logger *slog.Logger) int {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		logger.Info("environment variable not set, using fallback", "key", key, "fallback", fallback)
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		logger.Warn("invalid integer value for environment variable, using fallback", "key", key, "value", valStr, "error", err)
		return fallback
	}
	return val
}

// config initializes and returns a new apiConfig struct.
// It loads configuration from environment variables, establishes database and cache connections,
// and sets up the necessary clients and services for the application to run.
// The function will exit the application if any required configuration is missing or invalid.
func config() *apiConfig {
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
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	dbURL := getRequiredEnv("DB_URL", logger)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("couldn't prepare connection to database", "error", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		logger.Error("couldn't connect to database", "error", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)

	redisURL := getRequiredEnv("REDIS_URL", logger)
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("could not parse Redis URL", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(opt)
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		logger.Error("could not connect to Redis", "error", err)
		os.Exit(1)
	}
	cache := NewRedisCache(redisClient)

	currentIntervalMin := getEnvAsInt("CURRENT_INTERVAL_MIN", 10, logger)
	hourlyIntervalMin := getEnvAsInt("HOURLY_INTERVAL_MIN", 60, logger)
	dailyIntervalMin := getEnvAsInt("DAILY_INTERVAL_MIN", 720, logger)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	geocoder := NewGmpGeocodingService(getRequiredEnv("GMP_KEY", logger), getRequiredEnv("GMP_GEOCODE_URL", logger), httpClient)

	cfg := apiConfig{
		dbQueries:                dbQueries,
		cache:                    cache,
		geocoder:                 geocoder,
		gmpWeatherURL:            getRequiredEnv("GMP_WEATHER_URL", logger),
		owmWeatherURL:            getRequiredEnv("OWM_WEATHER_URL", logger),
		ometeoWeatherURL:         getRequiredEnv("OMETEO_WEATHER_URL", logger),
		gmpKey:                   getRequiredEnv("GMP_KEY", logger),
		owmKey:                   getRequiredEnv("OWM_KEY", logger),
		httpClient:               httpClient,
		schedulerCurrentInterval: time.Duration(currentIntervalMin) * time.Minute,
		schedulerHourlyInterval:  time.Duration(hourlyIntervalMin) * time.Minute,
		schedulerDailyInterval:   time.Duration(dailyIntervalMin) * time.Minute,
		port:                     getEnv("PORT", "8080", logger),
		devMode:                  devMode,
		logger:                   logger,
	}

	return &cfg
}

// dbQuerier is an interface that defines all the database operations required by the application.
// It is implemented by the sqlc-generated Queries struct, allowing for easy mocking in tests.
type dbQuerier interface {
	CreateCurrentWeather(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error)
	CreateDailyForecast(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error)
	CreateHourlyForecast(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error)
	CreateLocation(ctx context.Context, arg database.CreateLocationParams) (database.Location, error)
	CreateLocationAlias(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error)
	DeleteAllCurrentWeather(ctx context.Context) error
	DeleteAllDailyForecasts(ctx context.Context) error
	DeleteAllHourlyForecasts(ctx context.Context) error
	DeleteAllLocations(ctx context.Context) error
	DeleteCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) error
	DeleteDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) error
	DeleteHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) error
	DeleteLocation(ctx context.Context, id uuid.UUID) error
	GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error)
	GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error)
	GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error)
	GetCurrentWeatherAtLocationFromAPI(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error)
	GetDailyForecastAtLocationAndDateFromAPI(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error)
	GetHourlyForecastAtLocationAndTimeFromAPI(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error)
	GetLocationByAlias(ctx context.Context, alias string) (database.Location, error)
	GetLocationByCoordinates(ctx context.Context, arg database.GetLocationByCoordinatesParams) (database.Location, error)
	GetLocationByName(ctx context.Context, cityName string) (database.Location, error)
	GetUpcomingDailyForecastsAtLocation(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error)
	GetUpcomingHourlyForecastsAtLocation(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error)
	ListLocations(ctx context.Context) ([]database.Location, error)
	UpdateCurrentWeather(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error)
	UpdateDailyForecast(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error)
	UpdateHourlyForecast(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error)
	UpdateTimezone(ctx context.Context, arg database.UpdateTimezoneParams) error
}
