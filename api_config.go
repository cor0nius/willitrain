package main

import (
	"context"
	"database/sql"
	"log"
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

type apiConfig struct {
	dbQueries                dbQuerier
	gmpGeocodeURL            string
	cache                    Cache
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
}

// getRequiredEnv retrieves an environment variable by key, and fatals if it's not set.
func getRequiredEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("Environment variable %s must be set", key)
	}
	return val
}

// getEnv retrieves an environment variable by key, with a fallback value.
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	log.Printf("%s not set, defaulting to %s", key, fallback)
	return fallback
}

// getEnvAsInt retrieves an environment variable as an integer, with a fallback value.
func getEnvAsInt(key string, fallback int) int {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		log.Printf("%s not set, defaulting to %d", key, fallback)
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		log.Printf("%s is not a valid integer, defaulting to %d: %v", key, fallback, err)
		return fallback
	}
	return val
}

func config() *apiConfig {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	dbURL := getRequiredEnv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Couldn't prepare connection to database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}
	dbQueries := database.New(db)

	redisURL := getRequiredEnv("REDIS_URL")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Could not parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	cache := NewRedisCache(redisClient)

	currentIntervalMin := getEnvAsInt("CURRENT_INTERVAL_MIN", 10)
	hourlyIntervalMin := getEnvAsInt("HOURLY_INTERVAL_MIN", 60)
	dailyIntervalMin := getEnvAsInt("DAILY_INTERVAL_MIN", 720)

	devModeStr := getEnv("DEV_MODE", "false")
	devMode, err := strconv.ParseBool(devModeStr)
	if err != nil {
		log.Printf("Invalid value for DEV_MODE: %s, defaulting to false", devModeStr)
		devMode = false
	}

	cfg := apiConfig{
		dbQueries:        dbQueries,
		cache:            cache,
		gmpGeocodeURL:    getRequiredEnv("GMP_GEOCODE_URL"),
		gmpWeatherURL:    getRequiredEnv("GMP_WEATHER_URL"),
		owmWeatherURL:    getRequiredEnv("OWM_WEATHER_URL"),
		ometeoWeatherURL: getRequiredEnv("OMETEO_WEATHER_URL"),
		gmpKey:           getRequiredEnv("GMP_KEY"),
		owmKey:           getRequiredEnv("OWM_KEY"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		schedulerCurrentInterval: time.Duration(currentIntervalMin) * time.Minute,
		schedulerHourlyInterval:  time.Duration(hourlyIntervalMin) * time.Minute,
		schedulerDailyInterval:   time.Duration(dailyIntervalMin) * time.Minute,
		port:                     getEnv("PORT", "8080"),
		devMode:                  devMode,
	}

	return &cfg
}

type dbQuerier interface {
	CreateCurrentWeather(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error)
	CreateDailyForecast(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error)
	CreateHourlyForecast(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error)
	CreateLocation(ctx context.Context, arg database.CreateLocationParams) (database.Location, error)
	DeleteAllCurrentWeather(ctx context.Context) error
	DeleteAllDailyForecasts(ctx context.Context) error
	DeleteAllHourlyForecasts(ctx context.Context) error
	DeleteAllLocations(ctx context.Context) error
	GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error)
	GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error)
	GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error)
	GetCurrentWeatherAtLocationFromAPI(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error)
	GetDailyForecastAtLocationAndDateFromAPI(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error)
	GetHourlyForecastAtLocationAndTimeFromAPI(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error)
	GetLocationByName(ctx context.Context, cityName string) (database.Location, error)
	ListLocations(ctx context.Context) ([]database.Location, error)
	UpdateCurrentWeather(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error)
	UpdateDailyForecast(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error)
	UpdateHourlyForecast(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error)
}
