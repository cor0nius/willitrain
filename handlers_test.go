package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestHandlerConfig(t *testing.T) {
	testCases := []struct {
		name       string
		devMode    bool
		wantStatus int
		wantBody   string
	}{
		{
			name:       "Dev Mode True",
			devMode:    true,
			wantStatus: http.StatusOK,
			wantBody:   `{"dev_mode":true}`,
		},
		{
			name:       "Dev Mode False",
			devMode:    false,
			wantStatus: http.StatusOK,
			wantBody:   `{"dev_mode":false}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiCfg := &apiConfig{
				devMode: tc.devMode,
			}

			req := httptest.NewRequest("GET", "/api/config", nil)
			rr := httptest.NewRecorder()

			apiCfg.handlerConfig(rr, req)

			if status := rr.Code; status != tc.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.wantStatus)
			}

			if rr.Body.String() != tc.wantBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestHandlerResetDB(t *testing.T) {
	testCases := []struct {
		name          string
		setupMocks    func(cfg *testAPIConfig)
		wantStatus    int
		wantBody      string
		checkMocks    func(t *testing.T, cfg *testAPIConfig)
		requestMethod string
	}{
		{
			name: "Success",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.DeleteAllLocationsFunc = func(ctx context.Context) error {
					return nil
				}
				cfg.mockCache.flushFunc = func(ctx context.Context) error {
					return nil
				}
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"database and cache reset"}`,
			checkMocks: func(t *testing.T, cfg *testAPIConfig) {
				// No checks needed for success case
			},
			requestMethod: http.MethodPost,
		},
		{
			name: "DB Fails",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.DeleteAllLocationsFunc = func(ctx context.Context) error {
					return errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Failed to reset database"}`,
			checkMocks: func(t *testing.T, cfg *testAPIConfig) {
				// No checks needed for this case
			},
			requestMethod: http.MethodPost,
		},
		{
			name: "Cache Fails",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.DeleteAllLocationsFunc = func(ctx context.Context) error {
					return nil
				}
				cfg.mockCache.flushFunc = func(ctx context.Context) error {
					return errors.New("cache error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"Failed to flush cache"}`,
			checkMocks: func(t *testing.T, cfg *testAPIConfig) {
				// No checks needed for this case
			},
			requestMethod: http.MethodPost,
		},
		{
			name: "Wrong Method",
			setupMocks: func(cfg *testAPIConfig) {
				// No mocks needed
			},
			wantStatus:    http.StatusMethodNotAllowed,
			wantBody:      `{"error":"Method Not Allowed"}`,
			checkMocks:    func(t *testing.T, cfg *testAPIConfig) {},
			requestMethod: http.MethodGet,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg)

			req := httptest.NewRequest(tc.requestMethod, "/api/reset", nil)
			rr := httptest.NewRecorder()

			testCfg.apiConfig.handlerResetDB(rr, req)

			if status := rr.Code; status != tc.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.wantStatus)
			}

			if rr.Body.String() != tc.wantBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.wantBody)
			}
			tc.checkMocks(t, testCfg)
		})
	}
}

func TestHandlerCurrentWeather(t *testing.T) {
	mockLocationWithTimezone := MockLocation
	mockLocationWithTimezone.Timezone = "Europe/Warsaw"
	mockDBLocationWithTimezone := MockDBLocation
	mockDBLocationWithTimezone.Timezone = sql.NullString{String: "Europe/Warsaw", Valid: true}

	testCases := []struct {
		name       string
		setupMocks func(cfg *testAPIConfig)
		wantStatus int
		wantBody   string
		checkMocks func(t *testing.T, cfg *testAPIConfig)
	}{
		{
			name: "Success : DB Cache Hit",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return mockDBLocationWithTimezone, nil
				}
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) {
					return "", redis.Nil
				}
				cfg.mockDB.GetCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
					return []database.CurrentWeather{MockDBCurrentWeather, MockDBCurrentWeather, MockDBCurrentWeather}, nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
					return nil
				}
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"location":{"location_id":"` + mockLocationWithTimezone.LocationID.String() + `","city_name":"Wroclaw","latitude":51.1,"longitude":17.03,"country_code":"PL","timezone":"Europe/Warsaw"},"weather":[{"source_api":"test","timestamp":"` + MockDBCurrentWeather.UpdatedAt.In(time.FixedZone("Europe/Warsaw", 7200)).Format("2006-01-02 15:04") + `","temperature_c":10,"humidity":50,"wind_speed_kmh":5,"precipitation_mm":0,"condition_text":"sunny"},{"source_api":"test","timestamp":"` + MockDBCurrentWeather.UpdatedAt.In(time.FixedZone("Europe/Warsaw", 7200)).Format("2006-01-02 15:04") + `","temperature_c":10,"humidity":50,"wind_speed_kmh":5,"precipitation_mm":0,"condition_text":"sunny"},{"source_api":"test","timestamp":"` + MockDBCurrentWeather.UpdatedAt.In(time.FixedZone("Europe/Warsaw", 7200)).Format("2006-01-02 15:04") + `","temperature_c":10,"humidity":50,"wind_speed_kmh":5,"precipitation_mm":0,"condition_text":"sunny"}]}`,
			checkMocks: func(t *testing.T, cfg *testAPIConfig) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg)

			req := httptest.NewRequest("GET", "/?city=wroclaw", nil)
			rr := httptest.NewRecorder()

			testCfg.apiConfig.handlerCurrentWeather(rr, req)

			if status := rr.Code; status != tc.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.wantStatus)
			}

			if rr.Body.String() != tc.wantBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.wantBody)
			}
			tc.checkMocks(t, testCfg)
		})
	}
}

func TestHandlerDailyForecast(t *testing.T) {
	mockLocationWithTimezone := MockLocation
	mockLocationWithTimezone.Timezone = "Europe/Warsaw"
	mockDBLocationWithTimezone := MockDBLocation
	mockDBLocationWithTimezone.Timezone = sql.NullString{String: "Europe/Warsaw", Valid: true}

	testCases := []struct {
		name       string
		setupMocks func(cfg *testAPIConfig)
		wantStatus int
		wantBody   string
		checkMocks func(t *testing.T, cfg *testAPIConfig)
	}{
		{
			name: "Success: DB Cache Hit",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return mockDBLocationWithTimezone, nil
				}
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) {
					return "", redis.Nil
				}
				cfg.mockDB.GetUpcomingDailyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
					return []database.DailyForecast{MockDBDailyForecast}, nil
				}
				cfg.mockCache.setFunc = func(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
					return nil
				}
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"location":{"location_id":"` + mockLocationWithTimezone.LocationID.String() + `","city_name":"Wroclaw","latitude":51.1,"longitude":17.03,"country_code":"PL","timezone":"Europe/Warsaw"},"forecasts":[{"source_api":"test","forecast_date":"2035-01-01","min_temp_c":5,"max_temp_c":15,"precipitation_mm":1,"precipitation_chance":50,"wind_speed_kmh":10,"humidity":60}]}`,
			checkMocks: func(t *testing.T, cfg *testAPIConfig) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg)

			req := httptest.NewRequest("GET", "/?city=wroclaw", nil)
			rr := httptest.NewRecorder()

			testCfg.apiConfig.handlerDailyForecast(rr, req)

			if status := rr.Code; status != tc.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.wantStatus)
			}

			if rr.Body.String() != tc.wantBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.wantBody)
			}
			tc.checkMocks(t, testCfg)
		})
	}
}

func TestHandlerHourlyForecast(t *testing.T) {
	mockLocationWithTimezone := MockLocation
	mockLocationWithTimezone.Timezone = "Europe/Warsaw"
	mockDBLocationWithTimezone := MockDBLocation
	mockDBLocationWithTimezone.Timezone = sql.NullString{String: "Europe/Warsaw", Valid: true}

	testCases := []struct {
		name       string
		setupMocks func(cfg *testAPIConfig)
		wantStatus int
		wantBody   string
		checkMocks func(t *testing.T, cfg *testAPIConfig)
	}{
		{
			name: "Success: DB Cache Hit",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return mockDBLocationWithTimezone, nil
				}
				cfg.mockCache.getFunc = func(ctx context.Context, key string) (string, error) {
					return "", errors.New("not in cache")
				}
				cfg.mockDB.GetUpcomingHourlyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
					return []database.HourlyForecast{MockDBHourlyForecast}, nil
				}
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"location":{"location_id":"` + mockLocationWithTimezone.LocationID.String() + `","city_name":"Wroclaw","latitude":51.1,"longitude":17.03,"country_code":"PL","timezone":"Europe/Warsaw"},"forecasts":[{"source_api":"test","forecast_datetime":"2035-01-01 13:00","temperature_c":10,"humidity":50,"wind_speed_kmh":5,"precipitation_mm":0,"precipitation_chance":10,"condition_text":"cloudy"}]}`,
			checkMocks: func(t *testing.T, cfg *testAPIConfig) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg)

			req := httptest.NewRequest("GET", "/?city=wroclaw", nil)
			rr := httptest.NewRecorder()

			testCfg.apiConfig.handlerHourlyForecast(rr, req)

			if status := rr.Code; status != tc.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.wantStatus)
			}

			if rr.Body.String() != tc.wantBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.wantBody)
			}
			tc.checkMocks(t, testCfg)
		})
	}
}