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
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Couldn't connect to database: %v", err)
	}
	dbQueries := database.New(db)

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL must be set")
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Could not parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(opt)
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	cache := NewRedisCache(redisClient)

	gmpGeocodeURL := os.Getenv("GMP_GEOCODE_URL")
	if gmpGeocodeURL == "" {
		log.Fatal("GMP_GEOCODE_URL must be set")
	}

	gmpWeatherURL := os.Getenv("GMP_WEATHER_URL")
	if gmpWeatherURL == "" {
		log.Fatal("GMP_WEATHER_URL must be set")
	}

	owmWeatherURL := os.Getenv("OWM_WEATHER_URL")
	if owmWeatherURL == "" {
		log.Fatal("OWM_WEATHER_URL must be set")
	}

	ometeoWeatherURL := os.Getenv("OMETEO_WEATHER_URL")
	if ometeoWeatherURL == "" {
		log.Fatal("OMETEO_WEATHER_URL must be set")
	}

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		log.Fatal("Missing API Key for Google Maps Platform")
	}

	owmKey := os.Getenv("OWM_KEY")
	if owmKey == "" {
		log.Fatal("Missing API Key for OpenWeatherMap")
	}

	currentIntervalMin, err := strconv.Atoi(os.Getenv("CURRENT_INTERVAL_MIN"))
	if err != nil {
		log.Printf("CURRENT_INTERVAL_MIN not set or invalid, defaulting to 10 minutes: %v", err)
		currentIntervalMin = 10
	}

	hourlyIntervalMin, err := strconv.Atoi(os.Getenv("HOURLY_INTERVAL_MIN"))
	if err != nil {
		log.Printf("HOURLY_INTERVAL_MIN not set or invalid, defaulting to 60 minutes: %v", err)
		hourlyIntervalMin = 60
	}

	dailyIntervalMin, err := strconv.Atoi(os.Getenv("DAILY_INTERVAL_MIN"))
	if err != nil {
		log.Printf("DAILY_INTERVAL_MIN not set or invalid, defaulting to 720 minutes: %v", err)
		dailyIntervalMin = 720
	}

	devMode := os.Getenv("DEV_MODE")

	cfg := apiConfig{
		dbQueries:        dbQueries,
		cache:            cache,
		gmpGeocodeURL:    gmpGeocodeURL,
		gmpWeatherURL:    gmpWeatherURL,
		owmWeatherURL:    owmWeatherURL,
		ometeoWeatherURL: ometeoWeatherURL,
		gmpKey:           gmpKey,
		owmKey:           owmKey,
		httpClient: &http.Client{
			Timeout: http.DefaultClient.Timeout,
		},
		schedulerCurrentInterval: time.Duration(currentIntervalMin) * time.Minute,
		schedulerHourlyInterval:  time.Duration(hourlyIntervalMin) * time.Minute,
		schedulerDailyInterval:   time.Duration(dailyIntervalMin) * time.Minute,
	}

	scheduler := NewScheduler(&cfg,
		cfg.schedulerCurrentInterval,
		cfg.schedulerHourlyInterval,
		cfg.schedulerDailyInterval,
	)
	scheduler.Start()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/currentweather", cfg.handlerCurrentWeather)
	mux.HandleFunc("/dailyforecast", cfg.handlerDailyForecast)
	mux.HandleFunc("/hourlyforecast", cfg.handlerHourlyForecast)

	if devMode == "true" {
		log.Println("Development mode enabled. Registering /dev/reset-db endpoint.")
		mux.HandleFunc("/dev/reset-db", cfg.handlerResetDB)
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
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
