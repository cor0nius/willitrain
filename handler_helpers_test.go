package main

import (
	"context"
	"database/sql"
	"io"
	"log/slog"

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

type mockLocationQuerier struct {
	mockFailingQuerier
	GetLocationByAliasFunc   func(ctx context.Context, alias string) (database.Location, error)
	GetLocationByNameFunc    func(ctx context.Context, cityName string) (database.Location, error)
	CreateLocationFunc       func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error)
	CreateLocationAliasFunc  func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error)
	getLocationByAliasCalls  int
	getLocationByNameCalls   int
	createLocationCalls      int
	createLocationAliasCalls int
}

func (m *mockLocationQuerier) GetLocationByAlias(ctx context.Context, alias string) (database.Location, error) {
	m.getLocationByAliasCalls++
	return m.GetLocationByAliasFunc(ctx, alias)
}
func (m *mockLocationQuerier) GetLocationByName(ctx context.Context, cityName string) (database.Location, error) {
	m.getLocationByNameCalls++
	return m.GetLocationByNameFunc(ctx, cityName)
}
func (m *mockLocationQuerier) CreateLocation(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
	m.createLocationCalls++
	return m.CreateLocationFunc(ctx, arg)
}
func (m *mockLocationQuerier) CreateLocationAlias(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
	m.createLocationAliasCalls++
	return m.CreateLocationAliasFunc(ctx, arg)
}

type mockGeocodingService struct {
	GeocodeFunc         func(cityName string) (Location, error)
	ReverseGeocodeFunc  func(lat, lng float64) (Location, error)
	geocodeCalls        int
	reverseGeocodeCalls int
}

func (m *mockGeocodingService) Geocode(cityName string) (Location, error) {
	m.geocodeCalls++
	if m.GeocodeFunc != nil {
		return m.GeocodeFunc(cityName)
	}
	return Location{}, errors.New("GeocodeFunc not implemented in mock")
}

func (m *mockGeocodingService) ReverseGeocode(lat, lng float64) (Location, error) {
	m.reverseGeocodeCalls++
	if m.ReverseGeocodeFunc != nil {
		return m.ReverseGeocodeFunc(lat, lng)
	}
	return Location{}, errors.New("ReverseGeocodeFunc not implemented in mock")
}

func TestGetOrCreateLocation(t *testing.T) {
	ctx := context.Background()
	expectedLocation := Location{
		LocationID:  uuid.New(),
		CityName:    "Wroclaw",
		Latitude:    51.1,
		Longitude:   17.03,
		CountryCode: "PL",
	}
	dbLocation := database.Location{
		ID:          expectedLocation.LocationID,
		CityName:    expectedLocation.CityName,
		Latitude:    expectedLocation.Latitude,
		Longitude:   expectedLocation.Longitude,
		CountryCode: expectedLocation.CountryCode,
	}

	testCases := []struct {
		name           string
		cityName       string
		setupMocks     func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService)
		checkResult    func(t *testing.T, loc Location, err error)
		checkMockCalls func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService)
	}{
		{
			name:     "Alias Exists in DB",
			cityName: "wroclaw",
			setupMocks: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					if alias != "wroclaw" {
						t.Errorf("expected alias 'wroclaw', got '%s'", alias)
					}
					return dbLocation, nil
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					t.Error("Geocode should not be called when alias exists in DB")
					return Location{}, errors.New("unexpected geocode call")
				}
			},
			checkResult: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, expectedLocation) {
					t.Errorf("unexpected location returned. got %+v, want %+v", loc, expectedLocation)
				}
			},
			checkMockCalls: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				if db.getLocationByAliasCalls != 1 {
					t.Errorf("expected GetLocationByAlias to be called once, got %d", db.getLocationByAliasCalls)
				}
				if geo.geocodeCalls != 0 {
					t.Errorf("expected Geocode not to be called, but was called %d times", geo.geocodeCalls)
				}
			},
		},
		{
			name:     "Canonical Name Exists in DB",
			cityName: "wroclaw",
			setupMocks: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					return expectedLocation, nil
				}
				db.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					if cityName != "Wroclaw" {
						t.Errorf("expected city name 'Wroclaw', got '%s'", cityName)
					}
					return dbLocation, nil
				}
				db.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					if arg.Alias != "wroclaw" {
						t.Errorf("expected alias 'wroclaw', got '%s'", arg.Alias)
					}
					if arg.LocationID != expectedLocation.LocationID {
						t.Errorf("expected location ID %v, got %v", expectedLocation.LocationID, arg.LocationID)
					}
					return database.LocationAlias{}, nil
				}
			},
			checkResult: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, expectedLocation) {
					t.Errorf("unexpected location returned. got %+v, want %+v", loc, expectedLocation)
				}
			},
			checkMockCalls: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				if db.getLocationByAliasCalls != 1 {
					t.Errorf("expected GetLocationByAlias to be called once, got %d", db.getLocationByAliasCalls)
				}
				if geo.geocodeCalls != 1 {
					t.Errorf("expected Geocode to be called once, got %d", geo.geocodeCalls)
				}
				if db.getLocationByNameCalls != 1 {
					t.Errorf("expected GetLocationByName to be called once, got %d", db.getLocationByNameCalls)
				}
				if db.createLocationAliasCalls != 1 {
					t.Errorf("expected CreateLocationAlias to be called once, got %d", db.createLocationAliasCalls)
				}
				if db.createLocationCalls != 0 {
					t.Errorf("expected CreateLocation not to be called, but was called %d times", db.createLocationCalls)
				}
			},
		},
		{
			name:     "New City",
			cityName: "newville town",
			setupMocks: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				newLocation := Location{LocationID: uuid.New(), CityName: "Newville", CountryCode: "NV"}
				dbNewLocation := database.Location{ID: newLocation.LocationID, CityName: newLocation.CityName, CountryCode: newLocation.CountryCode}

				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					return newLocation, nil
				}
				db.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				db.CreateLocationFunc = func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
					return dbNewLocation, nil
				}
				db.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					return database.LocationAlias{}, nil
				}
			},
			checkResult: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if loc.CityName != "Newville" {
					t.Errorf("expected city name 'Newville', got '%s'", loc.CityName)
				}
			},
			checkMockCalls: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				if db.getLocationByAliasCalls != 1 {
					t.Errorf("expected GetLocationByAlias to be called once, got %d", db.getLocationByAliasCalls)
				}
				if geo.geocodeCalls != 1 {
					t.Errorf("expected Geocode to be called once, got %d", geo.geocodeCalls)
				}
				if db.getLocationByNameCalls != 1 {
					t.Errorf("expected GetLocationByName to be called once, got %d", db.getLocationByNameCalls)
				}
				if db.createLocationCalls != 1 {
					t.Errorf("expected CreateLocation to be called once, got %d", db.createLocationCalls)
				}
				if db.createLocationAliasCalls != 2 {
					t.Errorf("expected CreateLocationAlias to be called twice, got %d", db.createLocationAliasCalls)
				}
			},
		},
		{
			name:     "Geocoder Error",
			cityName: "errorcity",
			setupMocks: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					return Location{}, errors.New("geocoder service unavailable")
				}
			},
			checkResult: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
			checkMockCalls: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				if db.getLocationByAliasCalls != 1 {
					t.Errorf("expected GetLocationByAlias to be called once, got %d", db.getLocationByAliasCalls)
				}
				if geo.geocodeCalls != 1 {
					t.Errorf("expected Geocode to be called once, got %d", geo.geocodeCalls)
				}
				if db.getLocationByNameCalls != 0 {
					t.Errorf("expected GetLocationByName not to be called, got %d", db.getLocationByNameCalls)
				}
			},
		},
		{
			name:     "DB Error on GetLocationByAlias",
			cityName: "dberrorcity",
			setupMocks: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, errors.New("db connection lost")
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					t.Error("Geocode should not be called when GetLocationByAlias fails")
					return Location{}, nil
				}
			},
			checkResult: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
			checkMockCalls: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				if db.getLocationByAliasCalls != 1 {
					t.Errorf("expected GetLocationByAlias to be called once, got %d", db.getLocationByAliasCalls)
				}
				if geo.geocodeCalls != 0 {
					t.Errorf("expected Geocode not to be called, got %d", geo.geocodeCalls)
				}
			},
		},
		{
			name:     "DB Error on GetLocationByName",
			cityName: "dberrorcity",
			setupMocks: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					return expectedLocation, nil
				}
				db.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, errors.New("db connection lost")
				}
			},
			checkResult: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
			checkMockCalls: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				if db.getLocationByAliasCalls != 1 {
					t.Errorf("expected GetLocationByAlias to be called once, got %d", db.getLocationByAliasCalls)
				}
				if geo.geocodeCalls != 1 {
					t.Errorf("expected Geocode to be called once, got %d", geo.geocodeCalls)
				}
				if db.getLocationByNameCalls != 1 {
					t.Errorf("expected GetLocationByName to be called once, got %d", db.getLocationByNameCalls)
				}
			},
		},
		{
			name:     "DB Error on CreateLocation",
			cityName: "dberrorcity",
			setupMocks: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) { return expectedLocation, nil }
				db.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				db.CreateLocationFunc = func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
					return database.Location{}, errors.New("db connection lost")
				}
			},
			checkResult: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
			checkMockCalls: func(t *testing.T, db *mockLocationQuerier, geo *mockGeocodingService) {
				if db.createLocationCalls != 1 {
					t.Errorf("expected CreateLocation to be called once, got %d", db.createLocationCalls)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockLocationQuerier{}
			geoMock := &mockGeocodingService{}
			tc.setupMocks(t, dbMock, geoMock)
			cfg := &apiConfig{dbQueries: dbMock, geocoder: geoMock, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			loc, err := cfg.getOrCreateLocation(ctx, tc.cityName)
			tc.checkResult(t, loc, err)
			tc.checkMockCalls(t, dbMock, geoMock)
		})
	}
}

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
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
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

func TestGetCachedOrFetchDailyForecast_RedisHit(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville"}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	expectedForecast := []DailyForecast{
		{
			Location:            location,
			SourceAPI:           "TestCacheAPI",
			Timestamp:           fixedTime,
			ForecastDate:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			MinTemp:             5.0,
			MaxTemp:             15.0,
			Precipitation:       1.2,
			PrecipitationChance: 30,
			WindSpeed:           10.0,
			Humidity:            80,
		},
	}
	expectedData, err := json.Marshal(expectedForecast)
	if err != nil {
		t.Fatalf("failed to marshal expected forecast: %v", err)
	}

	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return string(expectedData), nil
		},
	}

	cfg := &apiConfig{
		cache:     cache,
		dbQueries: &mockFailingQuerier{t: t},
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchDailyForecast returned an unexpected error: %v", err)
	}

	if !reflect.DeepEqual(forecast, expectedForecast) {
		t.Errorf("returned forecast does not match expected forecast.\nGot: %v\nWant: %v", forecast, expectedForecast)
	}
}

func TestGetCachedOrFetchDailyForecast_DBHit(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville"}
	fixedTime := time.Now().UTC() // Use current time for freshness check

	dbForecasts := []database.DailyForecast{
		{
			ID:                         uuid.New(),
			LocationID:                 location.LocationID,
			SourceApi:                  "TestDB_API",
			UpdatedAt:                  fixedTime,
			ForecastDate:               time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			MinTempC:                   sql.NullFloat64{Float64: 5.0, Valid: true},
			MaxTempC:                   sql.NullFloat64{Float64: 15.0, Valid: true},
			PrecipitationMm:            sql.NullFloat64{Float64: 1.2, Valid: true},
			PrecipitationChancePercent: sql.NullInt32{Int32: 30, Valid: true},
			WindSpeedKmh:               sql.NullFloat64{Float64: 10.0, Valid: true},
			Humidity:                   sql.NullInt32{Int32: 80, Valid: true},
		},
	}
	expectedForecast := []DailyForecast{
		databaseDailyForecastToDailyForecast(dbForecasts[0], location),
	}

	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "", redis.Nil
		},
	}

	db := &mockQuerierForDailyDBHit{
		mockFailingQuerier: mockFailingQuerier{t: t},
		forecastToReturn:   dbForecasts,
	}

	cfg := &apiConfig{
		cache:     cache,
		dbQueries: db,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchDailyForecast returned an unexpected error: %v", err)
	}

	forecast[0].Timestamp = expectedForecast[0].Timestamp
	if !reflect.DeepEqual(forecast, expectedForecast) {
		t.Errorf("returned forecast does not match expected forecast.\nGot: %v\nWant: %v", forecast, expectedForecast)
	}
}

func TestGetCachedOrFetchHourlyForecast_RedisHit(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville"}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	expectedForecast := []HourlyForecast{
		{
			Location:            location,
			SourceAPI:           "TestCacheAPI",
			Timestamp:           fixedTime,
			ForecastDateTime:    time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			Temperature:         10.0,
			Humidity:            75,
			WindSpeed:           15.0,
			Precipitation:       0.5,
			PrecipitationChance: 40,
			Condition:           "Cloudy",
		},
	}
	expectedData, err := json.Marshal(expectedForecast)
	if err != nil {
		t.Fatalf("failed to marshal expected forecast: %v", err)
	}

	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return string(expectedData), nil
		},
	}

	cfg := &apiConfig{
		cache:     cache,
		dbQueries: &mockFailingQuerier{t: t},
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchHourlyForecast returned an unexpected error: %v", err)
	}

	if !reflect.DeepEqual(forecast, expectedForecast) {
		t.Errorf("returned forecast does not match expected forecast.\nGot: %v\nWant: %v", forecast, expectedForecast)
	}
}

func TestGetCachedOrFetchHourlyForecast_DBHit(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville"}
	fixedTime := time.Now().UTC() // Use current time for freshness check

	dbForecasts := []database.HourlyForecast{
		{
			ID:                         uuid.New(),
			LocationID:                 location.LocationID,
			SourceApi:                  "TestDB_API",
			UpdatedAt:                  fixedTime,
			ForecastDatetimeUtc:        time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			TemperatureC:               sql.NullFloat64{Float64: 10.0, Valid: true},
			Humidity:                   sql.NullInt32{Int32: 75, Valid: true},
			WindSpeedKmh:               sql.NullFloat64{Float64: 15.0, Valid: true},
			PrecipitationMm:            sql.NullFloat64{Float64: 0.5, Valid: true},
			PrecipitationChancePercent: sql.NullInt32{Int32: 40, Valid: true},
			ConditionText:              sql.NullString{String: "Cloudy", Valid: true},
		},
	}
	expectedForecast := []HourlyForecast{
		databaseHourlyForecastToHourlyForecast(dbForecasts[0], location),
	}

	// 1. Mock Cache should miss
	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "", redis.Nil
		},
	}

	// 2. Mock DB should return fresh data
	db := &mockQuerierForHourlyDBHit{
		mockFailingQuerier: mockFailingQuerier{t: t},
		forecastToReturn:   dbForecasts,
	}

	// 3. API call should not happen.
	cfg := &apiConfig{
		cache:     cache,
		dbQueries: db,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchHourlyForecast returned an unexpected error: %v", err)
	}

	if cache.getCalls != 1 {
		t.Errorf("expected 1 call to cache.Get, got %d", cache.getCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected 1 call to cache.Set to warm the cache, got %d", cache.setCalls)
	}
	if !reflect.DeepEqual(forecast, expectedForecast) {
		t.Errorf("returned forecast does not match expected forecast.\nGot: %v\nWant: %v", forecast, expectedForecast)
	}
}

func TestGetCachedOrFetchHourlyForecast_APIFetch(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}

	// --- Setup Mock API Server ---
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
			_, _ = w.Write(gmpData)
		} else if strings.Contains(r.URL.Path, "/owm") {
			_, _ = w.Write(owmData)
		} else if strings.Contains(r.URL.Path, "/ometeo") {
			_, _ = w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	// --- Setup Mocks ---
	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "", redis.Nil
		},
	}

	db := &mockQuerierForHourlyAPIFetch{}

	cfg := &apiConfig{
		cache:            cache,
		dbQueries:        db,
		gmpWeatherURL:    mockServer.URL + "/gmp/",
		owmWeatherURL:    mockServer.URL + "/owm?",
		ometeoWeatherURL: mockServer.URL + "/ometeo?",
		httpClient:       mockServer.Client(),
		gmpKey:           "dummy",
		owmKey:           "dummy",
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchHourlyForecast returned an unexpected error: %v", err)
	}

	if cache.getCalls != 1 {
		t.Errorf("expected 1 call to cache.Get, got %d", cache.getCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected 1 call to cache.Set, got %d", cache.setCalls)
	}
	if db.getHourlyForecastCalls != 1 {
		t.Errorf("expected 1 call to GetAllHourlyForecastsAtLocation, got %d", db.getHourlyForecastCalls)
	}
	// Each of the 3 APIs returns 24 hours of data.
	expectedCreateCalls := 24 * 3
	if db.createHourlyForecastCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateHourlyForecast, got %d", expectedCreateCalls, db.createHourlyForecastCalls)
	}
	if len(forecast) != expectedCreateCalls {
		t.Errorf("expected %d forecast results from APIs, got %d", expectedCreateCalls, len(forecast))
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
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
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

func TestGetCachedOrFetchDailyForecast_APIFetch(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}

	// --- Setup Mock API Server ---
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
			_, _ = w.Write(gmpData)
		} else if strings.Contains(r.URL.Path, "/owm") {
			_, _ = w.Write(owmData)
		} else if strings.Contains(r.URL.Path, "/ometeo") {
			_, _ = w.Write(ometeoData)
		}
	}))
	defer mockServer.Close()

	// --- Setup Mocks ---
	cache := &mockCache{
		getFunc: func(ctx context.Context, key string) (string, error) {
			return "", redis.Nil
		},
	}

	db := &mockQuerierForDailyAPIFetch{}

	cfg := &apiConfig{
		cache:            cache,
		dbQueries:        db,
		gmpWeatherURL:    mockServer.URL + "/gmp/",
		owmWeatherURL:    mockServer.URL + "/owm?",
		ometeoWeatherURL: mockServer.URL + "/ometeo?",
		httpClient:       mockServer.Client(),
		gmpKey:           "dummy",
		owmKey:           "dummy",
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
	if err != nil {
		t.Fatalf("getCachedOrFetchDailyForecast returned an unexpected error: %v", err)
	}

	if cache.getCalls != 1 {
		t.Errorf("expected 1 call to cache.Get, got %d", cache.getCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected 1 call to cache.Set, got %d", cache.setCalls)
	}
	if db.getDailyForecastCalls != 1 {
		t.Errorf("expected 1 call to GetAllDailyForecastsAtLocation, got %d", db.getDailyForecastCalls)
	}
	// The test data files contain 5, 8, and 7 days of forecasts respectively.
	// Our parsers limit this to 5 days each. So 5 * 3 = 15.
	expectedCreateCalls := 5 * 3
	if db.createDailyForecastCalls != expectedCreateCalls {
		t.Errorf("expected %d calls to CreateDailyForecast, got %d", expectedCreateCalls, db.createDailyForecastCalls)
	}
	if len(forecast) != expectedCreateCalls {
		t.Errorf("expected %d forecast results from APIs, got %d", expectedCreateCalls, len(forecast))
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
			_, _ = w.Write(gmpData)
		} else if strings.Contains(r.URL.Path, "/owm") {
			_, _ = w.Write(owmData)
		} else if strings.Contains(r.URL.Path, "/ometeo") {
			_, _ = w.Write(ometeoData)
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
		logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
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

func TestHandlerResetDB(t *testing.T) {
	dbMock := &mockDBQuerier{}
	cacheMock := &mockCache{}

	cfg := &apiConfig{
		dbQueries: dbMock,
		cache:     cacheMock,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest("POST", "/dev/reset-db", nil)
	rr := httptest.NewRecorder()

	cfg.handlerResetDB(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", rr.Code)
	}

	if dbMock.deleteAllLocationsCalls != 1 {
		t.Errorf("expected DeleteAllLocations to be called once, but got %d", dbMock.deleteAllLocationsCalls)
	}

	if cacheMock.flushCalls != 1 {
		t.Errorf("expected cache.Flush to be called once, but got %d", cacheMock.flushCalls)
	}

	expectedBody := `{"status":"database and cache reset"}`
	if strings.TrimSpace(rr.Body.String()) != expectedBody {
		t.Errorf("expected body to be %s, but got %s", expectedBody, rr.Body.String())
	}
}

type mockCache struct {
	getFunc    func(ctx context.Context, key string) (string, error)
	setFunc    func(ctx context.Context, key string, value any, expiration time.Duration) error
	flushFunc  func(ctx context.Context) error
	getCalls   int
	setCalls   int
	flushCalls int
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

func (m *mockCache) Flush(ctx context.Context) error {
	m.flushCalls++
	if m.flushFunc != nil {
		return m.flushFunc(ctx)
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
func (m *mockFailingQuerier) CreateLocationAlias(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
	m.fail("CreateLocationAlias")
	return database.LocationAlias{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetLocationByAlias(ctx context.Context, alias string) (database.Location, error) {
	m.fail("GetLocationByAlias")
	return database.Location{}, errors.New("unexpected call")
}
func (m *mockFailingQuerier) DeleteLocation(ctx context.Context, id uuid.UUID) error {
	m.fail("DeleteLocation")
	return errors.New("unexpected call")
}
func (m *mockFailingQuerier) GetLocationByCoordinates(ctx context.Context, arg database.GetLocationByCoordinatesParams) (database.Location, error) {
	m.fail("GetLocationByCoordinates")
	return database.Location{}, errors.New("unexpected call")
}

type mockQuerierForDBHit struct {
	mockFailingQuerier
	weatherToReturn []database.CurrentWeather
}

func (m *mockQuerierForDBHit) GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
	return m.weatherToReturn, nil
}

type mockQuerierForDailyDBHit struct {
	mockFailingQuerier
	forecastToReturn []database.DailyForecast
}

func (m *mockQuerierForDailyDBHit) GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error) {
	return m.forecastToReturn, nil
}

type mockQuerierForHourlyDBHit struct {
	mockFailingQuerier
	forecastToReturn           []database.HourlyForecast
	getAllHourlyForecastsCalls int
}

func (m *mockQuerierForHourlyDBHit) GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error) {
	m.getAllHourlyForecastsCalls++
	return m.forecastToReturn, nil
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

type mockQuerierForDailyAPIFetch struct {
	mockFailingQuerier
	createDailyForecastCalls int
	getDailyForecastCalls    int
}

func (m *mockQuerierForDailyAPIFetch) GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error) {
	m.getDailyForecastCalls++
	return nil, sql.ErrNoRows
}

func (m *mockQuerierForDailyAPIFetch) GetDailyForecastAtLocationAndDateFromAPI(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
	// This simulates a miss when checking if a forecast for a specific day/API already exists,
	// which then triggers a Create call.
	return database.DailyForecast{}, sql.ErrNoRows
}

func (m *mockQuerierForDailyAPIFetch) CreateDailyForecast(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
	m.createDailyForecastCalls++
	// Return a dummy forecast to satisfy the interface, the return value isn't used in the calling code.
	return database.DailyForecast{}, nil
}

type mockQuerierForHourlyAPIFetch struct {
	mockFailingQuerier
	createHourlyForecastCalls int
	getHourlyForecastCalls    int
}

func (m *mockQuerierForHourlyAPIFetch) GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error) {
	m.getHourlyForecastCalls++
	return nil, sql.ErrNoRows
}

func (m *mockQuerierForHourlyAPIFetch) GetHourlyForecastAtLocationAndTimeFromAPI(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
	// This simulates a miss when checking if a forecast for a specific day/API already exists,
	// which then triggers a Create call.
	return database.HourlyForecast{}, sql.ErrNoRows
}

func (m *mockQuerierForHourlyAPIFetch) CreateHourlyForecast(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
	m.createHourlyForecastCalls++
	// Return a dummy forecast to satisfy the interface, the return value isn't used in the calling code.
	return database.HourlyForecast{}, nil
}

type mockDBQuerier struct {
	mockFailingQuerier
	deleteAllLocationsFunc  func(ctx context.Context) error
	deleteAllLocationsCalls int
}

func (m *mockDBQuerier) DeleteAllLocations(ctx context.Context) error {
	m.deleteAllLocationsCalls++
	if m.deleteAllLocationsFunc != nil {
		return m.deleteAllLocationsFunc(ctx)
	}
	return nil
}
