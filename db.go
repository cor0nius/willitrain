package main

import (
	"context"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// ConnectDB establishes a connection to the PostgreSQL database using the provided
// connection string in the apiConfig struct. It initializes the dbQueries field with
// a sqlc-generated Queries struct, which provides type-safe methods for all database
// operations. This method should be called during application startup to ensure that
// the database is reachable before handling any requests.
func (cfg *apiConfig) ConnectDB() error {
	db, err := cfg.newDBClientFunc("postgres", cfg.dbURL)
	if err != nil {
		cfg.logger.Error("couldn't prepare connection to database", "error", err)
		return err
	}
	if err := db.Ping(); err != nil {
		cfg.logger.Error("couldn't connect to database", "error", err)
		return err
	}
	cfg.dbQueries = database.New(db)
	cfg.logger.Info("connected to database")
	return nil
}

// dbQuerier is an interface that abstracts all database operations.
// It is implemented by the sqlc-generated Queries struct, allowing for dependency
// injection and easy mocking in tests. This decouples business logic from the data layer.
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