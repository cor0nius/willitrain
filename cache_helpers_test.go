package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// --- Tests ---

func TestGetCachedOrFetchCurrentWeather(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}
	now := time.Now().UTC()

	apiWeather := []CurrentWeather{
		{SourceAPI: "gmp", Temperature: 22.0},
		{SourceAPI: "owm", Temperature: 23.0},
		{SourceAPI: "ometeo", Temperature: 21.0},
	}
	dbWeather := []database.CurrentWeather{
		{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "gmp", UpdatedAt: now, TemperatureC: sql.NullFloat64{Float64: 22.0, Valid: true}},
		{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "owm", UpdatedAt: now, TemperatureC: sql.NullFloat64{Float64: 23.0, Valid: true}},
		{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "ometeo", UpdatedAt: now, TemperatureC: sql.NullFloat64{Float64: 21.0, Valid: true}},
	}

	testCases := []struct {
		name       string
		setupMocks func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server)
		check      func(t *testing.T, weather []CurrentWeather, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiWeather)
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return string(cachedData), nil
				}
			},
			check: func(t *testing.T, weather []CurrentWeather, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(weather) != 3 {
					t.Fatalf("expected 3 weather items, got %d", len(weather))
				}
			},
		},
		{
			name: "Success: DB Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return "", redis.Nil
				}
				db.GetCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
					return dbWeather, nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error {
					return nil // Expect cache to be warmed
				}
			},
			check: func(t *testing.T, weather []CurrentWeather, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(weather) != 3 {
					t.Fatalf("expected 3 weather items, got %d", len(weather))
				}
			},
		},
		{
			name: "Success: API Fetch",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
					return nil, sql.ErrNoRows
				}
				db.GetCurrentWeatherAtLocationFromAPIFunc = func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, sql.ErrNoRows
				}
				db.CreateCurrentWeatherFunc = func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, nil
				}
				db.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, weather []CurrentWeather, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(weather) != 3 {
					t.Fatalf("expected 3 weather items, got %d", len(weather))
				}
			},
		},
	}

	// --- Mock API Server ---
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		if strings.Contains(r.URL.Path, "gmp") {
			data, _ = os.ReadFile("testdata/current_weather_gmp.json")
		} else if strings.Contains(r.URL.Path, "owm") {
			data, _ = os.ReadFile("testdata/current_weather_owm.json")
		} else {
			data, _ = os.ReadFile("testdata/current_weather_ometeo.json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer mockServer.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockHandlerHelpersQuerier{t: t}
			cacheMock := &mockCache{}
			tc.setupMocks(dbMock, cacheMock, mockServer)

			cfg := &apiConfig{
				dbQueries:        dbMock,
				cache:            cacheMock,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
				gmpWeatherURL:    mockServer.URL + "/gmp",
				owmWeatherURL:    mockServer.URL + "/owm",
				ometeoWeatherURL: mockServer.URL + "/ometeo",
				httpClient:       mockServer.Client(),
				gmpKey:           "dummy",
				owmKey:           "dummy",
			}

			weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
			tc.check(t, weather, err)
		})
	}
}

func TestGetCachedOrFetchDailyForecast(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}
	now := time.Now().UTC()

	apiForecast := []DailyForecast{{SourceAPI: "gmp", MaxTemp: 25.0}}
	dbForecast := []database.DailyForecast{{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "gmp", UpdatedAt: now, ForecastDate: now}}

	testCases := []struct {
		name       string
		setupMocks func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server)
		check      func(t *testing.T, forecast []DailyForecast, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiForecast)
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return string(cachedData), nil
				}
			},
			check: func(t *testing.T, forecast []DailyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) == 0 {
					t.Fatal("expected forecast items, got 0")
				}
			},
		},
		{
			name: "Success: DB Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingDailyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
					return dbForecast, nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []DailyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) != 1 {
					t.Fatalf("expected 1 forecast item, got %d", len(forecast))
				}
			},
		},
		{
			name: "Success: API Fetch",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingDailyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
					return nil, sql.ErrNoRows
				}
				db.GetDailyForecastAtLocationAndDateFromAPIFunc = func(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, sql.ErrNoRows
				}
				db.CreateDailyForecastFunc = func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, nil
				}
				db.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []DailyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				// 5 days from 3 APIs
				if len(forecast) != 15 {
					t.Fatalf("expected 15 forecast items, got %d", len(forecast))
				}
			},
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		if strings.Contains(r.URL.Path, "gmp") {
			data, _ = os.ReadFile("testdata/daily_forecast_gmp.json")
		} else if strings.Contains(r.URL.Path, "owm") {
			data, _ = os.ReadFile("testdata/daily_forecast_owm.json")
		} else {
			data, _ = os.ReadFile("testdata/daily_forecast_ometeo.json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer mockServer.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockHandlerHelpersQuerier{t: t}
			cacheMock := &mockCache{}
			tc.setupMocks(dbMock, cacheMock, mockServer)

			cfg := &apiConfig{
				dbQueries:        dbMock,
				cache:            cacheMock,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
				gmpWeatherURL:    mockServer.URL + "/gmp",
				owmWeatherURL:    mockServer.URL + "/owm",
				ometeoWeatherURL: mockServer.URL + "/ometeo",
				httpClient:       mockServer.Client(),
				gmpKey:           "dummy",
				owmKey:           "dummy",
			}

			forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
			tc.check(t, forecast, err)
		})
	}
}

func TestGetCachedOrFetchHourlyForecast(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}
	now := time.Now().UTC()

	apiForecast := []HourlyForecast{{SourceAPI: "gmp", Temperature: 15.0}}
	dbForecast := []database.HourlyForecast{{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "gmp", UpdatedAt: now, ForecastDatetimeUtc: now}}

	testCases := []struct {
		name       string
		setupMocks func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server)
		check      func(t *testing.T, forecast []HourlyForecast, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiForecast)
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return string(cachedData), nil
				}
			},
			check: func(t *testing.T, forecast []HourlyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) == 0 {
					t.Fatal("expected forecast items, got 0")
				}
			},
		},
		{
			name: "Success: DB Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingHourlyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
					return dbForecast, nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []HourlyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) != 1 {
					t.Fatalf("expected 1 forecast item, got %d", len(forecast))
				}
			},
		},
		{
			name: "Success: API Fetch",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingHourlyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
					return nil, sql.ErrNoRows
				}
				db.GetHourlyForecastAtLocationAndTimeFromAPIFunc = func(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, sql.ErrNoRows
				}
				db.CreateHourlyForecastFunc = func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, nil
				}
				db.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []HourlyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				// 24 hours from 3 APIs
				if len(forecast) != 72 {
					t.Fatalf("expected 72 forecast items, got %d", len(forecast))
				}
			},
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		if strings.Contains(r.URL.Path, "gmp") {
			data, _ = os.ReadFile("testdata/hourly_forecast_gmp.json")
		} else if strings.Contains(r.URL.Path, "owm") {
			data, _ = os.ReadFile("testdata/hourly_forecast_owm.json")
		} else {
			data, _ = os.ReadFile("testdata/hourly_forecast_ometeo.json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer mockServer.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockHandlerHelpersQuerier{t: t}
			cacheMock := &mockCache{}
			tc.setupMocks(dbMock, cacheMock, mockServer)

			cfg := &apiConfig{
				dbQueries:        dbMock,
				cache:            cacheMock,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
				gmpWeatherURL:    mockServer.URL + "/gmp",
				owmWeatherURL:    mockServer.URL + "/owm",
				ometeoWeatherURL: mockServer.URL + "/ometeo",
				httpClient:       mockServer.Client(),
				gmpKey:           "dummy",
				owmKey:           "dummy",
			}

			forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
			tc.check(t, forecast, err)
		})
	}
}
