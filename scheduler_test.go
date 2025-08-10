package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

func TestRunCurrentWeatherJobs(t *testing.T) {
	// --- Setup ---
	// Mock server for all weather APIs
	gmpData, err := testData.ReadFile("testdata/current_weather_gmp.json")
	if err != nil {
		t.Fatal(err)
	}
	owmData, err := testData.ReadFile("testdata/current_weather_owm.json")
	if err != nil {
		t.Fatal(err)
	}
	ometeoData, err := testData.ReadFile("testdata/current_weather_ometeo.json")
	if err != nil {
		t.Fatal(err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/gmp") {
			w.Write(gmpData)
		} else if r.URL.Path == "/owm" {
			w.Write(owmData)
		} else if r.URL.Path == "/ometeo" {
			w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	mockDB := &mockQuerier{
		locationsToReturn: []database.Location{
			{ID: uuid.New(), CityName: "Test City 1"},
			{ID: uuid.New(), CityName: "Test City 2"},
		},
	}

	cfg := &apiConfig{
		dbQueries:        mockDB,
		gmpWeatherURL:    mockServer.URL + "/gmp",
		owmWeatherURL:    mockServer.URL + "/owm?",
		ometeoWeatherURL: mockServer.URL + "/ometeo?",
		httpClient:       mockServer.Client(),
	}

	// We can use short intervals for testing since we call the job directly.
	s := NewScheduler(cfg, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	s.runCurrentWeatherJobs()

	// --- Assertions ---
	// For 2 locations and 3 APIs, we expect 6 total persistence attempts.
	// Each attempt first tries to GET an existing record.
	expectedGetCalls := 2 * 3
	if mockDB.getCurrentWeatherFromAPICalls != expectedGetCalls {
		t.Errorf("expected %d calls to GetCurrentWeatherAtLocationFromAPI, got %d", expectedGetCalls, mockDB.getCurrentWeatherFromAPICalls)
	}

	// Since our mock Get... returns sql.ErrNoRows, it should then try to CREATE.
	expectedCreateCalls := 2 * 3
	if mockDB.createCurrentWeatherCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateCurrentWeather, got %d", expectedCreateCalls, mockDB.createCurrentWeatherCalls)
	}

	// It should not try to UPDATE.
	if mockDB.updateCurrentWeatherCalls != 0 {
		t.Errorf("expected 0 calls to UpdateCurrentWeather, got %d", mockDB.updateCurrentWeatherCalls)
	}
}

func TestRunDailyForecastJobs(t *testing.T) {
	// --- Setup ---
	// Mock server for all weather APIs
	gmpData, err := testData.ReadFile("testdata/daily_forecast_gmp.json")
	if err != nil {
		t.Fatal(err)
	}
	owmData, err := testData.ReadFile("testdata/daily_forecast_owm.json")
	if err != nil {
		t.Fatal(err)
	}
	ometeoData, err := testData.ReadFile("testdata/daily_forecast_ometeo.json")
	if err != nil {
		t.Fatal(err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/gmp") {
			w.Write(gmpData)
		} else if r.URL.Path == "/owm" {
			w.Write(owmData)
		} else if r.URL.Path == "/ometeo" {
			w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	mockDB := &mockQuerier{
		locationsToReturn: []database.Location{
			{ID: uuid.New(), CityName: "Test City 1"},
			{ID: uuid.New(), CityName: "Test City 2"},
		},
	}

	cfg := &apiConfig{
		dbQueries:        mockDB,
		gmpWeatherURL:    mockServer.URL + "/gmp",
		owmWeatherURL:    mockServer.URL + "/owm?",
		ometeoWeatherURL: mockServer.URL + "/ometeo?",
		httpClient:       mockServer.Client(),
	}

	s := NewScheduler(cfg, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	s.runDailyForecastJobs()

	// --- Assertions ---
	// For 2 locations, 3 APIs, and 5 days per API, we expect 30 total persistence attempts.
	expectedGetCalls := 2 * 3 * 5
	if mockDB.getDailyForecastFromAPICalls != expectedGetCalls {
		t.Errorf("expected %d calls to GetDailyForecastAtLocationAndDateFromAPI, got %d", expectedGetCalls, mockDB.getDailyForecastFromAPICalls)
	}

	expectedCreateCalls := 2 * 3 * 5
	if mockDB.createDailyForecastCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateDailyForecast, got %d", expectedCreateCalls, mockDB.createDailyForecastCalls)
	}

	if mockDB.updateDailyForecastCalls != 0 {
		t.Errorf("expected 0 calls to UpdateDailyForecast, got %d", mockDB.updateDailyForecastCalls)
	}
}

func TestRunHourlyForecastJobs(t *testing.T) {
	// --- Setup ---
	// Mock server for all weather APIs
	gmpData, err := testData.ReadFile("testdata/hourly_forecast_gmp.json")
	if err != nil {
		t.Fatal(err)
	}
	owmData, err := testData.ReadFile("testdata/hourly_forecast_owm.json")
	if err != nil {
		t.Fatal(err)
	}
	ometeoData, err := testData.ReadFile("testdata/hourly_forecast_ometeo.json")
	if err != nil {
		t.Fatal(err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/gmp") {
			w.Write(gmpData)
		} else if r.URL.Path == "/owm" {
			w.Write(owmData)
		} else if r.URL.Path == "/ometeo" {
			w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	mockDB := &mockQuerier{
		locationsToReturn: []database.Location{
			{ID: uuid.New(), CityName: "Test City 1"},
			{ID: uuid.New(), CityName: "Test City 2"},
		},
	}

	cfg := &apiConfig{
		dbQueries:        mockDB,
		gmpWeatherURL:    mockServer.URL + "/gmp",
		owmWeatherURL:    mockServer.URL + "/owm?",
		ometeoWeatherURL: mockServer.URL + "/ometeo?",
		httpClient:       mockServer.Client(),
	}

	s := NewScheduler(cfg, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	s.runHourlyForecastJobs()

	// --- Assertions ---
	// For 2 locations, 3 APIs, and 24 hours per API, we expect 144 total persistence attempts.
	expectedGetCalls := 2 * 3 * 24
	if mockDB.getHourlyForecastFromAPICalls != expectedGetCalls {
		t.Errorf("expected %d calls to GetHourlyForecastAtLocationAndTimeFromAPI, got %d", expectedGetCalls, mockDB.getHourlyForecastFromAPICalls)
	}

	expectedCreateCalls := 2 * 3 * 24
	if mockDB.createHourlyForecastCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateHourlyForecast, got %d", expectedCreateCalls, mockDB.createHourlyForecastCalls)
	}

	if mockDB.updateHourlyForecastCalls != 0 {
		t.Errorf("expected 0 calls to UpdateHourlyForecast, got %d", mockDB.updateHourlyForecastCalls)
	}
}

func TestScheduler_Ticks(t *testing.T) {
	// --- Setup ---
	// We don't need a real cfg, just a placeholder.
	cfg := &apiConfig{}

	currentChan := make(chan time.Time)
	hourlyChan := make(chan time.Time)
	dailyChan := make(chan time.Time)

	s := &Scheduler{
		cfg:         cfg, // cfg is not used by the mock jobs
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
	// Mock DB that returns an error on ListLocations
	dbErr := errors.New("database connection failed")
	mockDB := &mockQuerier{
		listLocationsErr: dbErr,
	}

	cfg := &apiConfig{
		dbQueries: mockDB,
		// No need for httpClient or API URLs as they shouldn't be called
	}

	// We can use a dummy scheduler instance since we're calling the method directly
	s := &Scheduler{cfg: cfg}

	// Mock update function to track if it's called
	var updateFuncCalled bool
	mockUpdateFunc := func(ctx context.Context, location Location) {
		updateFuncCalled = true
	}

	// --- Action ---
	s.runUpdateForLocations(mockUpdateFunc)

	// --- Assertions ---
	if updateFuncCalled {
		t.Error("expected updateFunc not to be called when ListLocations fails, but it was")
	}
}

func TestRunUpdateForLocations_PartialAPIFailure(t *testing.T) {
	// --- Setup ---
	// Mock server that succeeds for one location and fails for another
	goodCityLat := "1.00"
	badCityLat := "2.00"

	gmpData, err := testData.ReadFile("testdata/current_weather_gmp.json")
	if err != nil {
		t.Fatal(err)
	}
	owmData, err := testData.ReadFile("testdata/current_weather_owm.json")
	if err != nil {
		t.Fatal(err)
	}
	ometeoData, err := testData.ReadFile("testdata/current_weather_ometeo.json")
	if err != nil {
		t.Fatal(err)
	}

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
			w.Write(data)
		} else if lat == badCityLat {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer mockServer.Close()

	mockDB := &mockQuerier{
		locationsToReturn: []database.Location{
			{ID: uuid.New(), CityName: "Good City", Latitude: 1.00},
			{ID: uuid.New(), CityName: "Bad City", Latitude: 2.00},
		},
	}

	cfg := &apiConfig{
		dbQueries:        mockDB,
		gmpWeatherURL:    mockServer.URL + "/gmp/",
		owmWeatherURL:    mockServer.URL + "/owm?",
		ometeoWeatherURL: mockServer.URL + "/ometeo?",
		httpClient:       mockServer.Client(),
		gmpKey:           "dummy",
		owmKey:           "dummy",
	}

	s := NewScheduler(cfg, 1*time.Minute, 1*time.Minute, 1*time.Minute)

	// --- Action ---
	// We test the partial failure logic by calling one of the job functions
	// that uses runUpdateForLocations. runCurrentWeatherJobs is a good candidate.
	s.runCurrentWeatherJobs()

	// --- Assertions ---
	// We expect persistence calls only for the "Good City".
	// There are 3 API providers, so we expect 3 persistence attempts for that city.
	expectedCalls := 3
	if mockDB.createCurrentWeatherCalls != expectedCalls {
		t.Errorf("expected %d calls to CreateCurrentWeather for the successful location, but got %d", expectedCalls, mockDB.createCurrentWeatherCalls)
	}
}

type mockQuerier struct {
	locationsToReturn []database.Location
	listLocationsErr  error

	createCurrentWeatherCalls     int
	updateCurrentWeatherCalls     int
	createDailyForecastCalls      int
	updateDailyForecastCalls      int
	createHourlyForecastCalls     int
	updateHourlyForecastCalls     int
	createLocationCalls           int
	getCurrentWeatherFromAPICalls int
	getDailyForecastFromAPICalls  int
	getHourlyForecastFromAPICalls int
}

func (m *mockQuerier) resetCounters() {
	m.createCurrentWeatherCalls = 0
	m.updateCurrentWeatherCalls = 0
	m.createDailyForecastCalls = 0
	m.updateDailyForecastCalls = 0
	m.createHourlyForecastCalls = 0
	m.updateHourlyForecastCalls = 0
	m.getCurrentWeatherFromAPICalls = 0
	m.getDailyForecastFromAPICalls = 0
	m.getHourlyForecastFromAPICalls = 0
}

func (m *mockQuerier) CreateCurrentWeather(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
	m.createCurrentWeatherCalls++
	return database.CurrentWeather{}, nil
}

func (m *mockQuerier) CreateDailyForecast(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
	m.createDailyForecastCalls++
	return database.DailyForecast{}, nil
}

func (m *mockQuerier) CreateHourlyForecast(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
	m.createHourlyForecastCalls++
	return database.HourlyForecast{}, nil
}

func (m *mockQuerier) CreateLocation(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
	m.createLocationCalls++
	return database.Location{}, nil
}

func (m *mockQuerier) DeleteAllCurrentWeather(ctx context.Context) error { return nil }

func (m *mockQuerier) DeleteAllDailyForecasts(ctx context.Context) error { return nil }

func (m *mockQuerier) DeleteAllHourlyForecasts(ctx context.Context) error { return nil }

func (m *mockQuerier) DeleteAllLocations(ctx context.Context) error { return nil }

func (m *mockQuerier) GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error) {
	return nil, nil
}

func (m *mockQuerier) GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error) {
	return nil, nil
}

func (m *mockQuerier) GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
	return nil, nil
}

func (m *mockQuerier) GetCurrentWeatherAtLocationFromAPI(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
	m.getCurrentWeatherFromAPICalls++
	return database.CurrentWeather{}, sql.ErrNoRows
}

func (m *mockQuerier) GetDailyForecastAtLocationAndDateFromAPI(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
	m.getDailyForecastFromAPICalls++
	return database.DailyForecast{}, sql.ErrNoRows
}

func (m *mockQuerier) GetHourlyForecastAtLocationAndTimeFromAPI(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
	m.getHourlyForecastFromAPICalls++
	return database.HourlyForecast{}, sql.ErrNoRows
}

func (m *mockQuerier) GetLocationByName(ctx context.Context, cityName string) (database.Location, error) {
	return database.Location{}, nil
}

func (m *mockQuerier) ListLocations(ctx context.Context) ([]database.Location, error) {
	return m.locationsToReturn, m.listLocationsErr
}

func (m *mockQuerier) UpdateCurrentWeather(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error) {
	m.updateCurrentWeatherCalls++
	return database.CurrentWeather{}, nil
}

func (m *mockQuerier) UpdateDailyForecast(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error) {
	m.updateDailyForecastCalls++
	return database.DailyForecast{}, nil
}

func (m *mockQuerier) UpdateHourlyForecast(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error) {
	m.updateHourlyForecastCalls++
	return database.HourlyForecast{}, nil
}
