package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

// mockParserSuccess simulates a successful parse of an API response.
func mockParserSuccess(body io.Reader, logger *slog.Logger) (CurrentWeather, string, error) {
	return CurrentWeather{
		SourceAPI:   "TestAPI",
		Temperature: 25.0,
	}, "Europe/Warsaw", nil
}

// mockParserError simulates a failed parse.
func mockParserError(body io.Reader, logger *slog.Logger) (CurrentWeather, string, error) {
	return CurrentWeather{SourceAPI: "TestAPI"}, "", errors.New("parsing failed")
}

func TestFetchForecastFromAPI(t *testing.T) {
	testCases := []struct {
		name          string
		serverHandler http.HandlerFunc
		parser        func(io.Reader, *slog.Logger) (CurrentWeather, string, error)
		expectError   bool
		expectedTemp  float64
	}{
		{
			name: "Successful fetch and parse",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"temp": 25.0}`)) // Dummy JSON, our mock parser handles it
			},
			parser:       mockParserSuccess,
			expectError:  false,
			expectedTemp: 25.0,
		},
		{
			name: "Server returns 500 error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			parser:      mockParserSuccess, // Parser won't be called
			expectError: true,
		},
		{
			name: "Parser returns error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`invalid json`)) // This will cause our mock parser to fail
			},
			parser:      mockParserError,
			expectError: true,
		},
		{
			name: "HTTP Get fails",
			// No server handler, URL will be invalid
			parser:      mockParserSuccess,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var server *httptest.Server
			var url string

			if tc.serverHandler != nil {
				server = setupMockServer(tc.serverHandler)
				defer server.Close()
				url = server.URL
			} else {
				// For the HTTP Get failure case, use a non-existent URL
				url = "http://localhost:12345/nonexistent"
			}

			cfg := &apiConfig{
				httpClient: http.DefaultClient,
			}

			var wg sync.WaitGroup
			results := make(chan struct {
				t   CurrentWeather
				tz  string
				err error
			}, 1)

			wg.Add(1)
			errorVal := CurrentWeather{SourceAPI: "TestAPI"}
			go fetchForecastFromAPI(cfg, url, tc.parser, errorVal, &wg, results)

			res := <-results
			wg.Wait()

			if tc.expectError && res.err == nil {
				t.Errorf("Expected an error, but got nil")
			}

			if !tc.expectError && res.err != nil {
				t.Errorf("Expected no error, but got: %v", res.err)
			}
		})
	}
}

func TestProcessForecastRequests(t *testing.T) {
	handlerSuccess := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"temp": 25.0}`))
	}
	serverSuccess := setupMockServer(handlerSuccess)
	defer serverSuccess.Close()

	handlerFail := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	serverFail := setupMockServer(handlerFail)
	defer serverFail.Close()

	providers := map[string]forecastProvider[CurrentWeather]{
		"provider1": {
			parser:   mockParserSuccess,
			errorVal: CurrentWeather{SourceAPI: "Provider 1"},
		},
		"provider2": {
			parser:   mockParserSuccess,
			errorVal: CurrentWeather{SourceAPI: "Provider 2"},
		},
	}

	testCases := []struct {
		name             string
		urls             map[string]string
		providers        map[string]forecastProvider[CurrentWeather]
		expectedLen      int
		expectedTimezone string
		expectError      bool
	}{
		{
			name: "All providers succeed",
			urls: map[string]string{
				"provider1": serverSuccess.URL,
				"provider2": serverSuccess.URL,
			},
			providers:        providers,
			expectedLen:      2,
			expectedTimezone: "Europe/Warsaw",
			expectError:      false,
		},
		{
			name: "One provider fails",
			urls: map[string]string{
				"provider1": serverSuccess.URL,
				"provider2": serverFail.URL,
			},
			providers:        providers,
			expectedLen:      1,
			expectedTimezone: "Europe/Warsaw",
			expectError:      false,
		},
		{
			name: "All providers fail",
			urls: map[string]string{
				"provider1": serverFail.URL,
				"provider2": serverFail.URL,
			},
			providers:        providers,
			expectedLen:      0,
			expectedTimezone: "",
			expectError:      true,
		},
		{
			name: "No provider found for URL",
			urls: map[string]string{
				"provider1":       serverSuccess.URL,
				"unknownProvider": serverSuccess.URL,
			},
			providers:        providers,
			expectedLen:      1,
			expectedTimezone: "Europe/Warsaw",
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &apiConfig{
				logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
				httpClient: http.DefaultClient,
			}

			results, tz, err := processForecastRequests(cfg, tc.urls, tc.providers)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}

			if len(results) != tc.expectedLen {
				t.Errorf("Expected %d results, but got %d", tc.expectedLen, len(results))
			}

			if tz != tc.expectedTimezone {
				t.Errorf("Expected timezone %q, but got %q", tc.expectedTimezone, tz)
			}
		})
	}
}

func TestRequestWeatherFunctions(t *testing.T) {
	location := Location{LocationID: uuid.New(), CityName: "Testville"}

	// This handler will now serve the correct data based on the URL path,
	// ensuring that each real parser gets the data it expects.
	handlerSuccess := createWeatherAPIHandler(t, "current_weather")
	serverSuccess := setupMockServer(handlerSuccess)
	defer serverSuccess.Close()

	testCases := []struct {
		name           string
		functionToTest string // "current", "daily", "hourly"
		setupMocks     func(cfg *testAPIConfig)
		check          func(t *testing.T, err error)
	}{
		{
			name:           "requestCurrentWeather - All providers fail",
			functionToTest: "current",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.apiConfig.httpClient = &http.Client{Transport: &errorTransport{err: errors.New("network error")}}
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
		},
		{
			name:           "requestDailyForecast - All providers fail",
			functionToTest: "daily",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.apiConfig.httpClient = &http.Client{Transport: &errorTransport{err: errors.New("network error")}}
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
		},
		{
			name:           "requestHourlyForecast - All providers fail",
			functionToTest: "hourly",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.apiConfig.httpClient = &http.Client{Transport: &errorTransport{err: errors.New("network error")}}
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
		},
		{
			name:           "requestCurrentWeather - UpdateTimezone fails",
			functionToTest: "current",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return errors.New("db error")
				}
			},
			check: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected no error, but got: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := newTestAPIConfig(t)
			testCfg.apiConfig.gmpWeatherURL = serverSuccess.URL + "/gmp"
			testCfg.apiConfig.owmWeatherURL = serverSuccess.URL + "/owm"
			testCfg.apiConfig.ometeoWeatherURL = serverSuccess.URL + "/ometeo"
			testCfg.apiConfig.gmpKey = "dummy"
			testCfg.apiConfig.owmKey = "dummy"

			tc.setupMocks(testCfg)

			var err error
			switch tc.functionToTest {
			case "current":
				_, err = testCfg.apiConfig.requestCurrentWeather(location)
			case "daily":
				// We need a different handler for daily/hourly to ensure parsers don't fail
				dailyHandler := createWeatherAPIHandler(t, "daily_forecast")
				dailyServer := setupMockServer(dailyHandler)
				testCfg.apiConfig.gmpWeatherURL = dailyServer.URL + "/gmp"
				testCfg.apiConfig.owmWeatherURL = dailyServer.URL + "/owm"
				testCfg.apiConfig.ometeoWeatherURL = dailyServer.URL + "/ometeo"
				_, err = testCfg.apiConfig.requestDailyForecast(location)
				dailyServer.Close()
			case "hourly":
				hourlyHandler := createWeatherAPIHandler(t, "hourly_forecast")
				hourlyServer := setupMockServer(hourlyHandler)
				testCfg.apiConfig.gmpWeatherURL = hourlyServer.URL + "/gmp"
				testCfg.apiConfig.owmWeatherURL = hourlyServer.URL + "/owm"
				testCfg.apiConfig.ometeoWeatherURL = hourlyServer.URL + "/ometeo"
				_, err = testCfg.apiConfig.requestHourlyForecast(location)
				hourlyServer.Close()
			default:
				t.Fatalf("unknown function to test: %s", tc.functionToTest)
			}

			tc.check(t, err)
		})
	}
}
