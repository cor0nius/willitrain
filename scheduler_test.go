package main

import (
	"context"
	"database/sql"
	"errors"
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
	// --- Setup ---
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

	testCfg := newTestAPIConfig(t)
	testCfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
		return []database.Location{
			{ID: uuid.New(), CityName: "Test City 1"},
			{ID: uuid.New(), CityName: "Test City 2"},
		}, nil
	}
	testCfg.mockDB.GetCurrentWeatherAtLocationFromAPIFunc = func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
		return database.CurrentWeather{}, sql.ErrNoRows
	}
	testCfg.mockDB.CreateCurrentWeatherFunc = func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
		return database.CurrentWeather{}, nil
	}

	testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
	testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
	testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"
	testCfg.apiConfig.httpClient = mockServer.Client()

	s := NewScheduler(testCfg.apiConfig, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	s.runCurrentWeatherJobs()

	// --- Assertions ---
	expectedCreateCalls := 2 * 3 // 2 locations, 3 APIs
	if testCfg.mockDB.createCurrentWeatherCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateCurrentWeather, got %d", expectedCreateCalls, testCfg.mockDB.createCurrentWeatherCalls)
	}
}

func TestRunDailyForecastJobs(t *testing.T) {
	// --- Setup ---
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

	testCfg := newTestAPIConfig(t)
	testCfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
		return []database.Location{
			{ID: uuid.New(), CityName: "Test City 1"},
			{ID: uuid.New(), CityName: "Test City 2"},
		}, nil
	}
	testCfg.mockDB.GetDailyForecastAtLocationAndDateFromAPIFunc = func(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
		return database.DailyForecast{}, sql.ErrNoRows
	}
	testCfg.mockDB.CreateDailyForecastFunc = func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
		return database.DailyForecast{}, nil
	}

	testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
	testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
	testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"
	testCfg.apiConfig.httpClient = mockServer.Client()

	s := NewScheduler(testCfg.apiConfig, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	s.runDailyForecastJobs()

	// --- Assertions ---
	expectedCreateCalls := 2 * 3 * 5 // 2 locations, 3 APIs, 5 days
	if testCfg.mockDB.createDailyForecastCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateDailyForecast, got %d", expectedCreateCalls, testCfg.mockDB.createDailyForecastCalls)
	}
}

func TestRunHourlyForecastJobs(t *testing.T) {
	// --- Setup ---
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

	testCfg := newTestAPIConfig(t)
	testCfg.mockDB.ListLocationsFunc = func(ctx context.Context) ([]database.Location, error) {
		return []database.Location{
			{ID: uuid.New(), CityName: "Test City 1"},
			{ID: uuid.New(), CityName: "Test City 2"},
		}, nil
	}
	testCfg.mockDB.GetHourlyForecastAtLocationAndTimeFromAPIFunc = func(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
		return database.HourlyForecast{}, sql.ErrNoRows
	}
	testCfg.mockDB.CreateHourlyForecastFunc = func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
		return database.HourlyForecast{}, nil
	}

	testCfg.apiConfig.gmpWeatherURL = mockServer.URL + "/gmp"
	testCfg.apiConfig.owmWeatherURL = mockServer.URL + "/owm"
	testCfg.apiConfig.ometeoWeatherURL = mockServer.URL + "/ometeo"
	testCfg.apiConfig.httpClient = mockServer.Client()

	s := NewScheduler(testCfg.apiConfig, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	s.runHourlyForecastJobs()

	// --- Assertions ---
	expectedCreateCalls := 2 * 3 * 24 // 2 locations, 3 APIs, 24 hours
	if testCfg.mockDB.createHourlyForecastCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateHourlyForecast, got %d", expectedCreateCalls, testCfg.mockDB.createHourlyForecastCalls)
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