package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

func TestRunCurrentWeatherJobs(t *testing.T) {
	gmpData, _ := os.ReadFile("testdata/current_weather_gmp.json")
	owmData, _ := os.ReadFile("testdata/current_weather_owm.json")
	ometeoData, _ := os.ReadFile("testdata/current_weather_ometeo.json")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/gmp") {
			_, _ = w.Write(gmpData)
		} else if strings.Contains(r.URL.Path, "/owm") {
			_, _ = w.Write(owmData)
		} else if strings.Contains(r.URL.Path, "/ometeo") {
			_, _ = w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	dbErr := errors.New("DB error")
	apiErr := errors.New("API error")

	tests := []struct {
		name                string
		setup               func(t *testing.T, cfg *testAPIConfig)
		expectedCreateCalls int
		expectedLogContains string
		expectErrorInLog    bool
		expectSuccessInLog  bool
	}{
		{
			name: "success",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{
						{ID: uuid.New(), CityName: "Test City 1"},
						{ID: uuid.New(), CityName: "Test City 2"},
					}, nil
				}
				cfg.mockDB.GetCurrentWeatherAtLocationFromAPIFunc = func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateCurrentWeatherFunc = func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, nil
				}
				cfg.apiConfig.httpClient = mockServer.Client()
			},
			expectedCreateCalls: 2 * 3, // 2 locations, 3 APIs
			expectSuccessInLog:  true,
		},
		{
			name: "db delete error",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{{ID: uuid.New(), CityName: "Test City 1"}}, nil
				}
				cfg.mockDB.DeleteCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) error {
					return dbErr
				}
			},
			expectedCreateCalls: 0,
			expectedLogContains: "failed to delete current weather",
			expectErrorInLog:    true,
		},
		{
			name: "forecast request error",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{{ID: uuid.New(), CityName: "Test City 1"}}, nil
				}
				cfg.apiConfig.httpClient = &http.Client{
					Transport: &errorTransport{err: apiErr},
				}
			},
			expectedCreateCalls: 0,
			expectedLogContains: "failed to request current weather",
			expectErrorInLog:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			tt.setup(t, testCfg)

			var logBuffer bytes.Buffer
			testCfg.apiConfig.logger = slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

			testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
			testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
			testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"

			s := NewScheduler(testCfg.apiConfig, 1*time.Minute, 1*time.Minute, 1*time.Minute)
			s.runCurrentWeatherJobs()

			if testCfg.mockDB.createCurrentWeatherCalls != tt.expectedCreateCalls {
				t.Errorf("expected %d calls to CreateCurrentWeather, got %d", tt.expectedCreateCalls, testCfg.mockDB.createCurrentWeatherCalls)
			}

			logOutput := logBuffer.String()
			if tt.expectErrorInLog && !strings.Contains(logOutput, tt.expectedLogContains) {
				t.Errorf("expected log to contain %q, but it didn't. Log: %s", tt.expectedLogContains, logOutput)
			}
			if tt.expectSuccessInLog && !strings.Contains(logOutput, "updated current weather") {
				t.Errorf("expected log to contain success message, but it didn't. Log: %s", logOutput)
			}
		})
	}
}

func TestRunDailyForecastJobs(t *testing.T) {
	gmpData, _ := os.ReadFile("testdata/daily_forecast_gmp.json")
	owmData, _ := os.ReadFile("testdata/daily_forecast_owm.json")
	ometeoData, _ := os.ReadFile("testdata/daily_forecast_ometeo.json")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/gmp") {
			_, _ = w.Write(gmpData)
		} else if strings.Contains(r.URL.Path, "/owm") {
			_, _ = w.Write(owmData)
		} else if strings.Contains(r.URL.Path, "/ometeo") {
			_, _ = w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	dbErr := errors.New("DB error")
	apiErr := errors.New("API error")

	tests := []struct {
		name                string
		setup               func(t *testing.T, cfg *testAPIConfig)
		expectedCreateCalls int
		expectedLogContains string
		expectErrorInLog    bool
		expectSuccessInLog  bool
	}{
		{
			name: "success",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{
						{ID: uuid.New(), CityName: "Test City 1"},
						{ID: uuid.New(), CityName: "Test City 2"},
					}, nil
				}
				cfg.mockDB.GetDailyForecastAtLocationAndDateFromAPIFunc = func(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateDailyForecastFunc = func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, nil
				}
				cfg.apiConfig.httpClient = mockServer.Client()
			},
			expectedCreateCalls: 2 * 3 * 5, // 2 locations, 3 APIs, 5 days
			expectSuccessInLog:  true,
		},
		{
			name: "db delete error",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{{ID: uuid.New(), CityName: "Test City 1"}}, nil
				}
				cfg.mockDB.DeleteDailyForecastsAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) error {
					return dbErr
				}
			},
			expectedCreateCalls: 0,
			expectedLogContains: "failed to delete daily forecasts",
			expectErrorInLog:    true,
		},
		{
			name: "forecast request error",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{{ID: uuid.New(), CityName: "Test City 1"}}, nil
				}
				cfg.apiConfig.httpClient = &http.Client{
					Transport: &errorTransport{err: apiErr},
				}
			},
			expectedCreateCalls: 0,
			expectedLogContains: "failed to request daily forecast",
			expectErrorInLog:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			tt.setup(t, testCfg)

			var logBuffer bytes.Buffer
			testCfg.apiConfig.logger = slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

			testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
			testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
			testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"

			s := NewScheduler(testCfg.apiConfig, 1*time.Minute, 1*time.Minute, 1*time.Minute)
			s.runDailyForecastJobs()

			if testCfg.mockDB.createDailyForecastCalls != tt.expectedCreateCalls {
				t.Errorf("expected %d calls to CreateDailyForecast, got %d", tt.expectedCreateCalls, testCfg.mockDB.createDailyForecastCalls)
			}

			logOutput := logBuffer.String()
			if tt.expectErrorInLog && !strings.Contains(logOutput, tt.expectedLogContains) {
				t.Errorf("expected log to contain %q, but it didn't. Log: %s", tt.expectedLogContains, logOutput)
			}
			if tt.expectSuccessInLog && !strings.Contains(logOutput, "updated daily forecast") {
				t.Errorf("expected log to contain success message, but it didn't. Log: %s", logOutput)
			}
		})
	}
}

func TestRunHourlyForecastJobs(t *testing.T) {
	gmpData, _ := os.ReadFile("testdata/hourly_forecast_gmp.json")
	owmData, _ := os.ReadFile("testdata/hourly_forecast_owm.json")
	ometeoData, _ := os.ReadFile("testdata/hourly_forecast_ometeo.json")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/gmp") {
			_, _ = w.Write(gmpData)
		} else if strings.Contains(r.URL.Path, "/owm") {
			_, _ = w.Write(owmData)
		} else if strings.Contains(r.URL.Path, "/ometeo") {
			_, _ = w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	dbErr := errors.New("DB error")
	apiErr := errors.New("API error")

	tests := []struct {
		name                string
		setup               func(t *testing.T, cfg *testAPIConfig)
		expectedCreateCalls int
		expectedLogContains string
		expectErrorInLog    bool
		expectSuccessInLog  bool
	}{
		{
			name: "success",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{
						{ID: uuid.New(), CityName: "Test City 1"},
						{ID: uuid.New(), CityName: "Test City 2"},
					}, nil
				}
				cfg.mockDB.GetHourlyForecastAtLocationAndTimeFromAPIFunc = func(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateHourlyForecastFunc = func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, nil
				}
				cfg.apiConfig.httpClient = mockServer.Client()
			},
			expectedCreateCalls: 2 * 3 * 24, // 2 locations, 3 APIs, 24 hours
			expectSuccessInLog:  true,
		},
		{
			name: "db delete error",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{{ID: uuid.New(), CityName: "Test City 1"}}, nil
				}
				cfg.mockDB.DeleteHourlyForecastsAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) error {
					return dbErr
				}
			},
			expectedCreateCalls: 0,
			expectedLogContains: "failed to delete hourly forecasts",
			expectErrorInLog:    true,
		},
		{
			name: "forecast request error",
			setup: func(t *testing.T, cfg *testAPIConfig) {
				cfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
					return []database.Location{{ID: uuid.New(), CityName: "Test City 1"}}, nil
				}
				cfg.apiConfig.httpClient = &http.Client{
					Transport: &errorTransport{err: apiErr},
				}
			},
			expectedCreateCalls: 0,
			expectedLogContains: "failed to request hourly forecast",
			expectErrorInLog:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			tt.setup(t, testCfg)

			var logBuffer bytes.Buffer
			testCfg.apiConfig.logger = slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

			testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
			testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
			testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"

			s := NewScheduler(testCfg.apiConfig, 1*time.Minute, 1*time.Minute, 1*time.Minute)
			s.runHourlyForecastJobs()

			if testCfg.mockDB.createHourlyForecastCalls != tt.expectedCreateCalls {
				t.Errorf("expected %d calls to CreateHourlyForecast, got %d", tt.expectedCreateCalls, testCfg.mockDB.createHourlyForecastCalls)
			}

			logOutput := logBuffer.String()
			if tt.expectErrorInLog && !strings.Contains(logOutput, tt.expectedLogContains) {
				t.Errorf("expected log to contain %q, but it didn't. Log: %s", tt.expectedLogContains, logOutput)
			}
			if tt.expectSuccessInLog && !strings.Contains(logOutput, "updated hourly forecast") {
				t.Errorf("expected log to contain success message, but it didn't. Log: %s", logOutput)
			}
		})
	}
}

func TestScheduler_Ticks(t *testing.T) {
	// --- Setup ---
	testCfg := newTestAPIConfig(t)

	currentChan := make(chan time.Time)
	hourlyChan := make(chan time.Time)
	dailyChan := make(chan time.Time)

	s := &Scheduler{
		cfg:         testCfg.apiConfig,
		currentChan: currentChan,
		hourlyChan:  hourlyChan,
		dailyChan:   dailyChan,
		stop:        make(chan struct{}),
	}

	// --- Mock Job Functions ---
	var wg sync.WaitGroup
	var currentCalled, hourlyCalled, dailyCalled bool

	s.currentWeatherJobs = func() {
		currentCalled = true
		wg.Done()
	}
	s.hourlyForecastJobs = func() {
		hourlyCalled = true
		wg.Done()
	}
	s.dailyForecastJobs = func() {
		dailyCalled = true
		wg.Done()
	}

	// --- Action & Assertions ---
	s.Start()
	defer s.Stop()

	t.Run("CurrentWeatherTick", func(t *testing.T) {
		currentCalled, hourlyCalled, dailyCalled = false, false, false
		wg.Add(1)
		currentChan <- time.Now()
		wg.Wait()

		if !currentCalled {
			t.Error("expected current weather job to be called, but it wasn't")
		}
		if hourlyCalled || dailyCalled {
			t.Error("hourly or daily jobs were called unexpectedly")
		}
	})

	t.Run("DailyForecastTick", func(t *testing.T) {
		currentCalled, hourlyCalled, dailyCalled = false, false, false
		wg.Add(1)
		dailyChan <- time.Now()
		wg.Wait()

		if !dailyCalled {
			t.Error("expected daily forecast job to be called, but it wasn't")
		}
	})

	t.Run("HourlyForecastTick", func(t *testing.T) {
		currentCalled, hourlyCalled, dailyCalled = false, false, false
		wg.Add(1)
		hourlyChan <- time.Now()
		wg.Wait()

		if !hourlyCalled {
			t.Error("expected hourly forecast job to be called, but it wasn't")
		}
	})
}

func TestRunUpdateForLocations_DBError(t *testing.T) {
	// --- Setup ---
	testCfg := newTestAPIConfig(t)
	dbErr := errors.New("database connection failed")
	testCfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
		return nil, dbErr
	}

	s := &Scheduler{cfg: testCfg.apiConfig}

	var updateFuncCalled bool
	mockUpdateFunc := func(ctx context.Context, location Location) {
		updateFuncCalled = true
	}

	// --- Action ---
	s.runUpdateForLocations("test job", mockUpdateFunc)

	// --- Assertions ---
	if updateFuncCalled {
		t.Error("expected updateFunc not to be called when ListLocations fails, but it was")
	}
}

func TestRunUpdateForLocations_PartialAPIFailure(t *testing.T) {
	// --- Setup ---
	goodCityLat := "1.00"
	badCityLat := "2.00"

	gmpData, _ := os.ReadFile("testdata/current_weather_gmp.json")
	owmData, _ := os.ReadFile("testdata/current_weather_owm.json")
	ometeoData, _ := os.ReadFile("testdata/current_weather_ometeo.json")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var lat string
		var data []byte

		switch {
		case strings.Contains(r.URL.Path, "/gmp"):
			lat = r.URL.Query().Get("location.latitude")
			data = gmpData
		case strings.Contains(r.URL.Path, "/owm"):
			lat = r.URL.Query().Get("lat")
			data = owmData
		case strings.Contains(r.URL.Path, "/ometeo"):
			lat = r.URL.Query().Get("latitude")
			data = ometeoData
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if lat == goodCityLat {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else if lat == badCityLat {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer mockServer.Close()

	testCfg := newTestAPIConfig(t)
	testCfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
		return []database.Location{
			{ID: uuid.New(), CityName: "Good City", Latitude: 1.00},
			{ID: uuid.New(), CityName: "Bad City", Latitude: 2.00},
		}, nil
	}
	testCfg.mockDB.GetCurrentWeatherAtLocationFromAPIFunc = func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
		return database.CurrentWeather{}, sql.ErrNoRows
	}
	testCfg.mockDB.CreateCurrentWeatherFunc = func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
		return database.CurrentWeather{}, nil
	}

	testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp/"
	testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm?"
	testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo?"
	testCfg.apiConfig.httpClient = mockServer.Client()
	testCfg.apiConfig.gmpKey = "dummy"
	testCfg.apiConfig.owmKey = "dummy"

	s := NewScheduler(testCfg.apiConfig, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	s.runCurrentWeatherJobs()

	// --- Assertions ---
	expectedCalls := 3
	if testCfg.mockDB.createCurrentWeatherCalls != expectedCalls {
		t.Errorf("expected %d calls to CreateCurrentWeather for the successful location, but got %d", expectedCalls, testCfg.mockDB.createCurrentWeatherCalls)
	}
}

func TestScheduler_Stop(t *testing.T) {
	testCfg := newTestAPIConfig(t)
	s := NewScheduler(testCfg.apiConfig, 10*time.Millisecond, 10*time.Millisecond, 10*time.Millisecond)

	// Mock the job functions to prevent real work and isolate the test
	// to the scheduler's lifecycle management.
	s.currentWeatherJobs = func() {}
	s.hourlyForecastJobs = func() {}
	s.dailyForecastJobs = func() {}

	s.Start()

	// Allow the scheduler goroutine to start and potentially run a job cycle.
	time.Sleep(20 * time.Millisecond)

	stopChan := make(chan struct{})
	go func() {
		s.Stop()
		close(stopChan)
	}()

	select {
	case <-stopChan:
		// Test passed, Stop() returned correctly.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("scheduler.Stop() timed out, worker goroutine may not have exited")
	}
}
