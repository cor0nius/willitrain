package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

func TestUpsertWeatherItem(t *testing.T) {
	type testCase struct {
		name             string
		getItemFunc      func() (any, error)
		createItemFunc   func() (any, error)
		updateItemFunc   func(existingItem any) (any, error)
		expectedLog      string
		expectedLogLevel string
	}

	testCases := []testCase{
		{
			name: "Failure - Get item fails with a generic error",
			getItemFunc: func() (any, error) {
				return nil, errors.New("something went wrong")
			},
			createItemFunc: func() (any, error) {
				return nil, nil // Should not be called
			},
			updateItemFunc: func(existingItem any) (any, error) {
				return nil, nil // Should not be called
			},
			expectedLog:      "error getting cache",
			expectedLogLevel: "ERROR",
		},
		{
			name: "Failure - Create item fails",
			getItemFunc: func() (any, error) {
				return nil, sql.ErrNoRows
			},
			createItemFunc: func() (any, error) {
				return nil, errors.New("create failed")
			},
			updateItemFunc: func(existingItem any) (any, error) {
				return nil, nil // Should not be called
			},
			expectedLog:      "error creating cache",
			expectedLogLevel: "ERROR",
		},
		{
			name: "Failure - Update item fails",
			getItemFunc: func() (any, error) {
				return "existing item", nil
			},
			createItemFunc: func() (any, error) {
				return nil, nil // Should not be called
			},
			updateItemFunc: func(existingItem any) (any, error) {
				return nil, errors.New("update failed")
			},
			expectedLog:      "error updating cache",
			expectedLogLevel: "ERROR",
		},
		{
			name: "Success - Update item succeeds",
			getItemFunc: func() (any, error) {
				return "existing item", nil
			},
			createItemFunc: func() (any, error) {
				return nil, nil // Should not be called
			},
			updateItemFunc: func(existingItem any) (any, error) {
				return "updated item", nil
			},
			expectedLog:      "updated cache item",
			expectedLogLevel: "DEBUG",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
			cfg := &apiConfig{logger: logger}

			logInfo := map[string]string{
				"type":     "test type",
				"location": "test location",
				"api":      "test api",
			}

			cfg.upsertWeatherItem(context.Background(), tc.getItemFunc, tc.createItemFunc, tc.updateItemFunc, logInfo)

			logOutput := logBuffer.String()
			if !strings.Contains(logOutput, tc.expectedLog) {
				t.Errorf("expected log to contain %q, but got %q", tc.expectedLog, logOutput)
			}
			if !strings.Contains(logOutput, "level="+tc.expectedLogLevel) {
				t.Errorf("expected log level to be %q, but got %q", tc.expectedLogLevel, logOutput)
			}
		})
	}
}

func TestPersistCurrentWeather(t *testing.T) {
	ctx := context.Background()
	mockWeather := []CurrentWeather{
		{
			Location:  MockLocation,
			SourceAPI: "test-api",
		},
	}
	mockExistingDBWeather := database.CurrentWeather{
		ID:         uuid.New(),
		LocationID: MockLocation.LocationID,
		SourceApi:  "test-api",
	}

	t.Run("Success - Item is Updated", func(t *testing.T) {
		// Setup
		testCfg := newTestAPIConfig(t)
		updateCalled := 0
		createCalled := 0

		// Mock the DB to return an existing item
		testCfg.mockDB.GetCurrentWeatherAtLocationFromAPIFunc = func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
			return mockExistingDBWeather, nil
		}

		// Track calls to Update and Create
		testCfg.mockDB.UpdateCurrentWeatherFunc = func(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error) {
			updateCalled++
			// Check if the correct ID is passed
			if arg.ID != mockExistingDBWeather.ID {
				t.Errorf("expected ID %v but got %v", mockExistingDBWeather.ID, arg.ID)
			}
			return database.CurrentWeather{}, nil
		}
		testCfg.mockDB.CreateCurrentWeatherFunc = func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
			createCalled++
			return database.CurrentWeather{}, nil
		}

		// Execute
		testCfg.apiConfig.persistCurrentWeather(ctx, mockWeather)

		// Verify
		if updateCalled != 1 {
			t.Errorf("expected UpdateCurrentWeather to be called once, but got %d", updateCalled)
		}
		if createCalled != 0 {
			t.Errorf("expected CreateCurrentWeather not to be called, but got %d", createCalled)
		}
	})
}

func TestPersistDailyForecast(t *testing.T) {
	ctx := context.Background()
	mockForecast := []DailyForecast{
		{
			Location:     MockLocation,
			SourceAPI:    "test-api",
			ForecastDate: time.Now(),
		},
	}
	mockExistingDBForecast := database.DailyForecast{
		ID:         uuid.New(),
		LocationID: MockLocation.LocationID,
		SourceApi:  "test-api",
	}

	t.Run("Success - Item is Updated", func(t *testing.T) {
		// Setup
		testCfg := newTestAPIConfig(t)
		updateCalled := 0
		createCalled := 0

		// Mock the DB to return an existing item
		testCfg.mockDB.GetDailyForecastAtLocationAndDateFromAPIFunc = func(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
			return mockExistingDBForecast, nil
		}

		// Track calls to Update and Create
		testCfg.mockDB.UpdateDailyForecastFunc = func(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error) {
			updateCalled++
			if arg.ID != mockExistingDBForecast.ID {
				t.Errorf("expected ID %v but got %v", mockExistingDBForecast.ID, arg.ID)
			}
			return database.DailyForecast{}, nil
		}
		testCfg.mockDB.CreateDailyForecastFunc = func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
			createCalled++
			return database.DailyForecast{}, nil
		}

		// Execute
		testCfg.apiConfig.persistDailyForecast(ctx, mockForecast)

		// Verify
		if updateCalled != 1 {
			t.Errorf("expected UpdateDailyForecast to be called once, but got %d", updateCalled)
		}
		if createCalled != 0 {
			t.Errorf("expected CreateDailyForecast not to be called, but got %d", createCalled)
		}
	})
}

func TestPersistHourlyForecast(t *testing.T) {
	ctx := context.Background()
	mockForecast := []HourlyForecast{
		{
			Location:         MockLocation,
			SourceAPI:        "test-api",
			ForecastDateTime: time.Now(),
		},
	}
	mockExistingDBForecast := database.HourlyForecast{
		ID:         uuid.New(),
		LocationID: MockLocation.LocationID,
		SourceApi:  "test-api",
	}

	t.Run("Success - Item is Updated", func(t *testing.T) {
		// Setup
		testCfg := newTestAPIConfig(t)
		updateCalled := 0
		createCalled := 0

		// Mock the DB to return an existing item
		testCfg.mockDB.GetHourlyForecastAtLocationAndTimeFromAPIFunc = func(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
			return mockExistingDBForecast, nil
		}

		// Track calls to Update and Create
		testCfg.mockDB.UpdateHourlyForecastFunc = func(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error) {
			updateCalled++
			if arg.ID != mockExistingDBForecast.ID {
				t.Errorf("expected ID %v but got %v", mockExistingDBForecast.ID, arg.ID)
			}
			return database.HourlyForecast{}, nil
		}
		testCfg.mockDB.CreateHourlyForecastFunc = func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
			createCalled++
			return database.HourlyForecast{}, nil
		}

		// Execute
		testCfg.apiConfig.persistHourlyForecast(ctx, mockForecast)

		// Verify
		if updateCalled != 1 {
			t.Errorf("expected UpdateHourlyForecast to be called once, but got %d", updateCalled)
		}
		if createCalled != 0 {
			t.Errorf("expected CreateHourlyForecast not to be called, but got %d", createCalled)
		}
	})
}