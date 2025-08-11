package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestGetCachedOrFetchCurrentWeather_RedisHit(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville"}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	expectedWeather := []CurrentWeather{{Location: location, SourceAPI: "TestCacheAPI", Temperature: 22.0, Timestamp: fixedTime}}
	expectedData, err := json.Marshal(expectedWeather)
	if err != nil {
		t.Fatalf("failed to marshal expected weather: %v", err)
	}

	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return string(expectedData), nil
		},
	}

	cfg := &apiConfig{
		cache:     cache,
		dbQueries: &mockFailingQuerier{t: t},
	}

	weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchCurrentWeather returned an unexpected error: %v", err)
	}

	if cache.getCalls != 1 {
		t.Errorf("expected 1 call to cache.Get, got %d", cache.getCalls)
	}
	if cache.setCalls != 0 {
		t.Errorf("expected 0 calls to cache.Set, got %d", cache.setCalls)
	}
	if !reflect.DeepEqual(weather, expectedWeather) {
		t.Errorf("returned weather does not match expected weather")
	}
}

func TestGetCachedOrFetchCurrentWeather_DBHit(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville"}
	fixedTime := time.Now().UTC() // Use current time for freshness check

	dbWeather := []database.CurrentWeather{
		{
			ID:           uuid.New(),
			LocationID:   location.LocationID,
			SourceApi:    "TestDB_API",
			UpdatedAt:    fixedTime,
			TemperatureC: sql.NullFloat64{Float64: 20.0, Valid: true},
			Humidity:     sql.NullInt32{Int32: 60, Valid: true},
		},
	}
	expectedWeather := []CurrentWeather{
		databaseCurrentWeatherToCurrentWeather(dbWeather[0], location),
	}

	// 1. Mock Cache should miss
	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "", redis.Nil
		},
	}

	// 2. Mock DB should return fresh data
	db := &mockQuerierForDBHit{
		mockFailingQuerier: mockFailingQuerier{t: t},
		weatherToReturn:    dbWeather,
	}

	// 3. API call should not happen. httpClient is nil, so it would panic.
	cfg := &apiConfig{
		cache:     cache,
		dbQueries: db,
	}

	weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchCurrentWeather returned an unexpected error: %v", err)
	}

	if cache.getCalls != 1 {
		t.Errorf("expected 1 call to cache.Get, got %d", cache.getCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected 1 call to cache.Set to warm the cache, got %d", cache.setCalls)
	}

	// We need to ignore the timestamp for comparison as it's set inside the function
	weather[0].Timestamp = expectedWeather[0].Timestamp
	if !reflect.DeepEqual(weather, expectedWeather) {
		t.Errorf("returned weather does not match expected weather.\nGot: %v\nWant: %v", weather, expectedWeather)
	}
}

func TestGetCachedOrFetchCurrentWeather_APIFetch(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}

	// --- Setup Mock API Server ---
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
		} else if strings.Contains(r.URL.Path, "/owm") {
			w.Write(owmData)
		} else if strings.Contains(r.URL.Path, "/ometeo") {
			w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	// --- Setup Mocks ---
	// 1. Mock Cache should miss
	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "", redis.Nil
		},
	}

	// 2. Mock DB should miss
	db := &mockQuerierForAPIFetch{}

	// 3. Setup apiConfig with mocks
	cfg := &apiConfig{
		cache:            cache,
		dbQueries:        db,
		gmpWeatherURL:    mockServer.URL + "/gmp/",
		owmWeatherURL:    mockServer.URL + "/owm?",
		ometeoWeatherURL: mockServer.URL + "/ometeo?",
		httpClient:       mockServer.Client(),
		gmpKey:           "dummy",
		owmKey:           "dummy",
	}

	// --- Action ---
	weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchCurrentWeather returned an unexpected error: %v", err)
	}

	// --- Assertions ---
	if cache.getCalls != 1 {
		t.Errorf("expected 1 call to cache.Get, got %d", cache.getCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected 1 call to cache.Set, got %d", cache.setCalls)
	}

	if db.createCurrentWeatherCalls != 3 {
		t.Errorf("expected 3 calls to CreateCurrentWeather, got %d", db.createCurrentWeatherCalls)
	}

	if len(weather) != 3 {
		t.Errorf("expected 3 weather results from APIs, got %d", len(weather))
	}
}

type mockCache struct {
	getFunc  func(ctx context.Context, key string) (string, error)
	setFunc  func(ctx context.Context, key string, value any, expiration time.Duration) error
	getCalls int
	setCalls int
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	m.getCalls++
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return "", redis.Nil
}

func (m *mockCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	m.setCalls++
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, expiration)
	}
	return nil
}

type mockFailingQuerier struct {
	t *testing.T
}

func (m *mockFailingQuerier) fail(methodName string) {
	m.t.Errorf("%s should not have been called", methodName)
}

func (m *mockFailingQuerier) CreateCurrentWeather(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
	m.fail("CreateCurrentWeather")
	return database.CurrentWeather{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) CreateDailyForecast(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
	m.fail("CreateDailyForecast")
	return database.DailyForecast{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) CreateHourlyForecast(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
	m.fail("CreateHourlyForecast")
	return database.HourlyForecast{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) CreateLocation(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
	m.fail("CreateLocation")
	return database.Location{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) DeleteAllCurrentWeather(ctx context.Context) error {
	m.fail("DeleteAllCurrentWeather")
	return errors.New("unexpected call")
}
func (m *mockFailingQuerier) DeleteAllDailyForecasts(ctx context.Context) error {
	m.fail("DeleteAllDailyForecasts")
	return errors.New("unexpected call")
}
func (m *mockFailingQuerier) DeleteAllHourlyForecasts(ctx context.Context) error {
	m.fail("DeleteAllHourlyForecasts")
	return errors.New("unexpected call")
}
func (m *mockFailingQuerier) DeleteAllLocations(ctx context.Context) error {
	m.fail("DeleteAllLocations")
	return errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error) {
	m.fail("GetAllDailyForecastsAtLocation")
	return nil, errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error) {
	m.fail("GetAllHourlyForecastsAtLocation")
	return nil, errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
	m.fail("GetCurrentWeatherAtLocation")
	return nil, errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetCurrentWeatherAtLocationFromAPI(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
	m.fail("GetCurrentWeatherAtLocationFromAPI")
	return database.CurrentWeather{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetDailyForecastAtLocationAndDateFromAPI(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
	m.fail("GetDailyForecastAtLocationAndDateFromAPI")
	return database.DailyForecast{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetHourlyForecastAtLocationAndTimeFromAPI(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
	m.fail("GetHourlyForecastAtLocationAndTimeFromAPI")
	return database.HourlyForecast{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetLocationByName(ctx context.Context, cityName string) (database.Location, error) {
	m.fail("GetLocationByName")
	return database.Location{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) ListLocations(ctx context.Context) ([]database.Location, error) {
	m.fail("ListLocations")
	return nil, errors.New("unexpected call")
}
func (m *mockFailingQuerier) UpdateCurrentWeather(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error) {
	m.fail("UpdateCurrentWeather")
	return database.CurrentWeather{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) UpdateDailyForecast(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error) {
	m.fail("UpdateDailyForecast")
	return database.DailyForecast{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) UpdateHourlyForecast(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error) {
	m.fail("UpdateHourlyForecast")
	return database.HourlyForecast{}, errors.New("unexpected call")
}

type mockQuerierForDBHit struct {
	mockFailingQuerier
	weatherToReturn []database.CurrentWeather
}

func (m *mockQuerierForDBHit) GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
	return m.weatherToReturn, nil
}

type mockQuerierForAPIFetch struct {
	mockFailingQuerier
	createCurrentWeatherCalls int
}

func (m *mockQuerierForAPIFetch) GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
	return nil, sql.ErrNoRows
}

func (m *mockQuerierForAPIFetch) GetCurrentWeatherAtLocationFromAPI(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
	return database.CurrentWeather{}, sql.ErrNoRows
}

func (m *mockQuerierForAPIFetch) CreateCurrentWeather(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
	m.createCurrentWeatherCalls++
	return database.CurrentWeather{}, nil
}
