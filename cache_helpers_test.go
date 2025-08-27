package main

import (
	"context"
	"database/sql"
	"encoding/json"
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
		setupMocks func(cfg *testAPIConfig, server *httptest.Server)
		check      func(t *testing.T, weather []CurrentWeather, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiWeather)
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) {
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
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) {
					return "", redis.Nil
				}
				cfg.mockDB.GetCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
					return dbWeather, nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error {
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
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				cfg.mockDB.GetCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
					return nil, sql.ErrNoRows
				}
				cfg.mockDB.GetCurrentWeatherAtLocationFromAPIFunc = func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateCurrentWeatherFunc = func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, nil
				}
				cfg.mockDB.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
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
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg, mockServer)

			testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
			testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
			testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"
			testCfg.apiConfig.httpClient = mockServer.Client()
			testCfg.apiConfig.gmpKey = "dummy"
			testCfg.apiConfig.owmKey = "dummy"

			weather, err := testCfg.apiConfig.getCachedOrFetchCurrentWeather(ctx, location)
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
		setupMocks func(cfg *testAPIConfig, server *httptest.Server)
		check      func(t *testing.T, forecast []DailyForecast, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiForecast)
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) {
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
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				cfg.mockDB.GetUpcomingDailyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
					return dbForecast, nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
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
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				cfg.mockDB.GetUpcomingDailyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
					return nil, sql.ErrNoRows
				}
				cfg.mockDB.GetDailyForecastAtLocationAndDateFromAPIFunc = func(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateDailyForecastFunc = func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, nil
				}
				cfg.mockDB.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
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
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg, mockServer)

			testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
			testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
			testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"
			testCfg.apiConfig.httpClient = mockServer.Client()
			testCfg.apiConfig.gmpKey = "dummy"
			testCfg.apiConfig.owmKey = "dummy"

			forecast, err := testCfg.apiConfig.getCachedOrFetchDailyForecast(ctx, location)
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
		setupMocks func(cfg *testAPIConfig, server *httptest.Server)
		check      func(t *testing.T, forecast []HourlyForecast, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiForecast)
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) {
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
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				cfg.mockDB.GetUpcomingHourlyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
					return dbForecast, nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
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
			setupMocks: func(cfg *testAPIConfig, server *httptest.Server) {
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				cfg.mockDB.GetUpcomingHourlyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
					return nil, sql.ErrNoRows
				}
				cfg.mockDB.GetHourlyForecastAtLocationAndTimeFromAPIFunc = func(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateHourlyForecastFunc = func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, nil
				}
				cfg.mockDB.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
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
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg, mockServer)

			testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
			testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
			testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"
			testCfg.apiConfig.httpClient = mockServer.Client()
			testCfg.apiConfig.gmpKey = "dummy"
			testCfg.apiConfig.owmKey = "dummy"

			forecast, err := testCfg.apiConfig.getCachedOrFetchHourlyForecast(ctx, location)
			tc.check(t, forecast, err)
		})
	}
}