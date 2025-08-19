package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// mockParserSuccess simulates a successful parse of an API response.
func mockParserSuccess(body io.Reader, logger *slog.Logger) (CurrentWeather, error) {
	return CurrentWeather{
		SourceAPI:   "TestAPI",
		Temperature: 25.0,
	}, nil
}

// mockParserError simulates a failed parse.
func mockParserError(body io.Reader, logger *slog.Logger) (CurrentWeather, error) {
	return CurrentWeather{SourceAPI: "TestAPI"}, errors.New("parsing failed")
}

func TestFetchForecastFromAPI(t *testing.T) {
	testCases := []struct {
		name          string
		serverHandler http.HandlerFunc
		parser        func(io.Reader, *slog.Logger) (CurrentWeather, error)
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
				server = httptest.NewServer(tc.serverHandler)
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

func TestRequestHourlyForecast(t *testing.T) {
	// Mock server for GMP
	gmpData, err := testData.ReadFile("testdata/hourly_forecast_gmp.json")
	if err != nil {
		t.Fatal(err)
	}
	gmpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(gmpData)
	}))
	defer gmpServer.Close()

	// Mock server for OWM
	owmData, err := testData.ReadFile("testdata/hourly_forecast_owm.json")
	if err != nil {
		t.Fatal(err)
	}
	owmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(owmData)
	}))
	defer owmServer.Close()

	// Mock server for O-Meteo
	ometeoData, err := testData.ReadFile("testdata/hourly_forecast_ometeo.json")
	if err != nil {
		t.Fatal(err)
	}
	ometeoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(ometeoData)
	}))
	defer ometeoServer.Close()

	// Mock server that always fails
	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverFail.Close()

	location := Location{Latitude: 51.11, Longitude: 17.04}

	testCases := []struct {
		name        string
		cfg         apiConfig
		expectedLen int
		expectError bool
	}{
		{
			name: "All providers succeed",
			cfg: apiConfig{
				gmpWeatherURL:    gmpServer.URL + "/",
				owmWeatherURL:    owmServer.URL + "?",
				ometeoWeatherURL: ometeoServer.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 72, // 24 from each provider
			expectError: false,
		},
		{
			name: "One provider fails",
			cfg: apiConfig{
				gmpWeatherURL:    gmpServer.URL + "/",
				owmWeatherURL:    serverFail.URL + "?",
				ometeoWeatherURL: ometeoServer.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 48, // 24 from GMP, 24 from O-Meteo
			expectError: false,
		},
		{
			name: "All providers fail",
			cfg: apiConfig{
				gmpWeatherURL:    serverFail.URL + "/",
				owmWeatherURL:    serverFail.URL + "?",
				ometeoWeatherURL: serverFail.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 0,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := tc.cfg.requestHourlyForecast(location)
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if len(results) != tc.expectedLen {
				t.Errorf("Expected %d results, but got %d", tc.expectedLen, len(results))
			}
		})
	}
}

func TestRequestDailyForecast(t *testing.T) {
	// Mock server for GMP
	gmpData, err := testData.ReadFile("testdata/daily_forecast_gmp.json")
	if err != nil {
		t.Fatal(err)
	}
	gmpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(gmpData)
	}))
	defer gmpServer.Close()

	// Mock server for OWM
	owmData, err := testData.ReadFile("testdata/daily_forecast_owm.json")
	if err != nil {
		t.Fatal(err)
	}
	owmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(owmData)
	}))
	defer owmServer.Close()

	// Mock server for O-Meteo
	ometeoData, err := testData.ReadFile("testdata/daily_forecast_ometeo.json")
	if err != nil {
		t.Fatal(err)
	}
	ometeoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(ometeoData)
	}))
	defer ometeoServer.Close()

	// Mock server that always fails
	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverFail.Close()

	location := Location{Latitude: 51.11, Longitude: 17.04}

	testCases := []struct {
		name        string
		cfg         apiConfig
		expectedLen int
		expectError bool
	}{
		{
			name: "All providers succeed",
			cfg: apiConfig{
				gmpWeatherURL:    gmpServer.URL + "/",
				owmWeatherURL:    owmServer.URL + "?",
				ometeoWeatherURL: ometeoServer.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 15, // 5 from GMP, 5 from OWM, 5 from O-Meteo
			expectError: false,
		},
		{
			name: "One provider fails",
			cfg: apiConfig{
				gmpWeatherURL:    gmpServer.URL + "/",
				owmWeatherURL:    serverFail.URL + "?",
				ometeoWeatherURL: ometeoServer.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 10, // 5 from GMP, 5 from O-Meteo
			expectError: false,
		},
		{
			name: "All providers fail",
			cfg: apiConfig{
				gmpWeatherURL:    serverFail.URL + "/",
				owmWeatherURL:    serverFail.URL + "?",
				ometeoWeatherURL: serverFail.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 0,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := tc.cfg.requestDailyForecast(location)
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if len(results) != tc.expectedLen {
				t.Errorf("Expected %d results, but got %d", tc.expectedLen, len(results))
			}
		})
	}
}

func TestRequestCurrentWeather(t *testing.T) {
	// Mock server for GMP
	gmpData, err := testData.ReadFile("testdata/current_weather_gmp.json")
	if err != nil {
		t.Fatal(err)
	}
	gmpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(gmpData)
	}))
	defer gmpServer.Close()

	// Mock server for OWM
	owmData, err := testData.ReadFile("testdata/current_weather_owm.json")
	if err != nil {
		t.Fatal(err)
	}
	owmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(owmData)
	}))
	defer owmServer.Close()

	// Mock server for O-Meteo
	ometeoData, err := testData.ReadFile("testdata/current_weather_ometeo.json")
	if err != nil {
		t.Fatal(err)
	}
	ometeoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(ometeoData)
	}))
	defer ometeoServer.Close()

	// Mock server that always fails
	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer serverFail.Close()

	location := Location{Latitude: 51.11, Longitude: 17.04}

	testCases := []struct {
		name        string
		cfg         apiConfig
		expectedLen int
		expectError bool
	}{
		{
			name: "All providers succeed",
			cfg: apiConfig{
				gmpWeatherURL:    gmpServer.URL + "/",
				owmWeatherURL:    owmServer.URL + "?",
				ometeoWeatherURL: ometeoServer.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 3,
			expectError: false,
		},
		{
			name: "One provider fails",
			cfg: apiConfig{
				gmpWeatherURL:    gmpServer.URL + "/",
				owmWeatherURL:    serverFail.URL + "?",
				ometeoWeatherURL: ometeoServer.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 2,
			expectError: false,
		},
		{
			name: "All providers fail",
			cfg: apiConfig{
				gmpWeatherURL:    serverFail.URL + "/",
				owmWeatherURL:    serverFail.URL + "?",
				ometeoWeatherURL: serverFail.URL + "?",
				gmpKey:           "dummy",
				owmKey:           "dummy",
				httpClient:       http.DefaultClient,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			expectedLen: 0,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := tc.cfg.requestCurrentWeather(location)
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}
			if len(results) != tc.expectedLen {
				t.Errorf("Expected %d results, but got %d", tc.expectedLen, len(results))
			}
		})
	}
}

func TestProcessForecastRequests(t *testing.T) {
	// Mock server that always succeeds
	serverSuccess := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"temp": 25.0}`))
	}))
	defer serverSuccess.Close()

	// Mock server that always fails
	serverFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
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
		name        string
		urls        map[string]string
		providers   map[string]forecastProvider[CurrentWeather]
		expectedLen int
		expectError bool
	}{
		{
			name: "All providers succeed",
			urls: map[string]string{
				"provider1": serverSuccess.URL,
				"provider2": serverSuccess.URL,
			},
			providers:   providers,
			expectedLen: 2,
			expectError: false,
		},
		{
			name: "One provider fails",
			urls: map[string]string{
				"provider1": serverSuccess.URL,
				"provider2": serverFail.URL,
			},
			providers:   providers,
			expectedLen: 1,
			expectError: false, // The function logs errors but doesn't return one
		},
		{
			name: "All providers fail",
			urls: map[string]string{
				"provider1": serverFail.URL,
				"provider2": serverFail.URL,
			},
			providers:   providers,
			expectedLen: 0,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal apiConfig with a logger that discards output
			cfg := &apiConfig{
				logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
				httpClient: http.DefaultClient,
			}

			results, err := processForecastRequests(cfg, tc.urls, tc.providers)

			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}

			if len(results) != tc.expectedLen {
				t.Errorf("Expected %d results, but got %d", tc.expectedLen, len(results))
			}
		})
	}
}
