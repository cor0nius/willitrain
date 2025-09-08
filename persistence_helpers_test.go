package main

import (
	"bytes"
	"context"

	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"
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
		// We will add test cases here in the next steps.
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
