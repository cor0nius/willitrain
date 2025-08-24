package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// --- Mocks ---

// mockGeocodingService is a mock for the Geocoder interface.
type mockGeocodingService struct {
	GeocodeFunc        func(cityName string) (Location, error)
	ReverseGeocodeFunc func(lat, lng float64) (Location, error)
}

func (m *mockGeocodingService) Geocode(cityName string) (Location, error) {
	if m.GeocodeFunc != nil {
		return m.GeocodeFunc(cityName)
	}
	return Location{}, errors.New("GeocodeFunc not implemented in mock")
}

func (m *mockGeocodingService) ReverseGeocode(lat, lng float64) (Location, error) {
	if m.ReverseGeocodeFunc != nil {
		return m.ReverseGeocodeFunc(lat, lng)
	}
	return Location{}, errors.New("ReverseGeocodeFunc not implemented in mock")
}

// mockCache is a mock for the Cache interface.
type mockCache struct {
	getFunc   func(ctx context.Context, key string) (string, error)
	setFunc   func(ctx context.Context, key string, value any, expiration time.Duration) error
	flushFunc func(ctx context.Context) error
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return "", redis.Nil
}

func (m *mockCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, expiration)
	}
	return nil
}

func (m *mockCache) Flush(ctx context.Context) error {
	if m.flushFunc != nil {
		return m.flushFunc(ctx)
	}
	return nil
}

// mockHandlerHelpersQuerier is a comprehensive, safe mock for the database.Querier interface.
// It fails the test if any unexpected method is called.
type mockHandlerHelpersQuerier struct {
	t *testing.T

	CreateCurrentWeatherFunc                      func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error)
	CreateDailyForecastFunc                       func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error)
	CreateHourlyForecastFunc                      func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error)
	CreateLocationFunc                            func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error)
	CreateLocationAliasFunc                       func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error)
	DeleteAllCurrentWeatherFunc                   func(ctx context.Context) error
	DeleteAllDailyForecastsFunc                   func(ctx context.Context) error
	DeleteAllHourlyForecastsFunc                  func(ctx context.Context) error
	DeleteAllLocationsFunc                        func(ctx context.Context) error
	DeleteCurrentWeatherAtLocationFunc			func(ctx context.Context, locationID uuid.UUID) error
	DeleteDailyForecastsAtLocationFunc			func(ctx context.Context, locationID uuid.UUID) error
	DeleteHourlyForecastsAtLocationFunc			func(ctx context.Context, locationID uuid.UUID) error
	DeleteLocationFunc                            func(ctx context.Context, id uuid.UUID) error
	GetAllDailyForecastsAtLocationFunc            func(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error)
	GetAllHourlyForecastsAtLocationFunc           func(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error)
	GetCurrentWeatherAtLocationFunc               func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error)
	GetCurrentWeatherAtLocationFromAPIFunc        func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error)
	GetDailyForecastAtLocationAndDateFromAPIFunc  func(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error)
	GetHourlyForecastAtLocationAndTimeFromAPIFunc func(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error)
	GetLocationByAliasFunc                        func(ctx context.Context, alias string) (database.Location, error)
	GetLocationByCoordinatesFunc                  func(ctx context.Context, arg database.GetLocationByCoordinatesParams) (database.Location, error)
	GetLocationByNameFunc                         func(ctx context.Context, cityName string) (database.Location, error)
	GetUpcomingDailyForecastsAtLocationFunc       func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error)
	GetUpcomingHourlyForecastsAtLocationFunc      func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error)
	ListLocationsFunc                             func(ctx context.Context) ([]database.Location, error)
	UpdateCurrentWeatherFunc                      func(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error)
	UpdateDailyForecastFunc                       func(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error)
	UpdateHourlyForecastFunc                      func(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error)
	UpdateTimezoneFunc                            func(ctx context.Context, arg database.UpdateTimezoneParams) error
}

// --- mockHandlerHelpersQuerier method implementations ---

func (m *mockHandlerHelpersQuerier) fail(method string) {
	m.t.Fatalf("unexpected call to mockHandlerHelpersQuerier method: %s", method)
}

func (m *mockHandlerHelpersQuerier) CreateCurrentWeather(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
	if m.CreateCurrentWeatherFunc != nil {
		return m.CreateCurrentWeatherFunc(ctx, arg)
	}
	m.fail("CreateCurrentWeather")
	return database.CurrentWeather{}, nil
}
func (m *mockHandlerHelpersQuerier) CreateDailyForecast(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
	if m.CreateDailyForecastFunc != nil {
		return m.CreateDailyForecastFunc(ctx, arg)
	}
	m.fail("CreateDailyForecast")
	return database.DailyForecast{}, nil
}
func (m *mockHandlerHelpersQuerier) CreateHourlyForecast(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
	if m.CreateHourlyForecastFunc != nil {
		return m.CreateHourlyForecastFunc(ctx, arg)
	}
	m.fail("CreateHourlyForecast")
	return database.HourlyForecast{}, nil
}
func (m *mockHandlerHelpersQuerier) CreateLocation(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
	if m.CreateLocationFunc != nil {
		return m.CreateLocationFunc(ctx, arg)
	}
	m.fail("CreateLocation")
	return database.Location{}, nil
}
func (m *mockHandlerHelpersQuerier) CreateLocationAlias(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
	if m.CreateLocationAliasFunc != nil {
		return m.CreateLocationAliasFunc(ctx, arg)
	}
	m.fail("CreateLocationAlias")
	return database.LocationAlias{}, nil
}
func (m *mockHandlerHelpersQuerier) DeleteAllCurrentWeather(ctx context.Context) error {
	if m.DeleteAllCurrentWeatherFunc != nil {
		return m.DeleteAllCurrentWeatherFunc(ctx)
	}
	m.fail("DeleteAllCurrentWeather")
	return nil
}
func (m *mockHandlerHelpersQuerier) DeleteAllDailyForecasts(ctx context.Context) error {
	if m.DeleteAllDailyForecastsFunc != nil {
		return m.DeleteAllDailyForecastsFunc(ctx)
	}
	m.fail("DeleteAllDailyForecasts")
	return nil
}
func (m *mockHandlerHelpersQuerier) DeleteAllHourlyForecasts(ctx context.Context) error {
	if m.DeleteAllHourlyForecastsFunc != nil {
		return m.DeleteAllHourlyForecastsFunc(ctx)
	}
	m.fail("DeleteAllHourlyForecasts")
	return nil
}
func (m *mockHandlerHelpersQuerier) DeleteAllLocations(ctx context.Context) error {
	if m.DeleteAllLocationsFunc != nil {
		return m.DeleteAllLocationsFunc(ctx)
	}
	m.fail("DeleteAllLocations")
	return nil
}

func (m *mockHandlerHelpersQuerier) DeleteCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) error {
	if m.DeleteCurrentWeatherAtLocationFunc != nil {
		return m.DeleteCurrentWeatherAtLocationFunc(ctx, locationID)
	}
	m.fail("DeleteCurrentWeatherAtLocation")
	return nil
}

func (m *mockHandlerHelpersQuerier) DeleteDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) error {
	if m.DeleteDailyForecastsAtLocationFunc != nil {
		return m.DeleteDailyForecastsAtLocationFunc(ctx, locationID)
	}
	m.fail("DeleteDailyForecastsAtLocation")
	return nil
}

func (m *mockHandlerHelpersQuerier) DeleteHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) error {
	if m.DeleteHourlyForecastsAtLocationFunc != nil {
		return m.DeleteHourlyForecastsAtLocationFunc(ctx, locationID)
	}
	m.fail("DeleteHourlyForecastsAtLocation")
	return nil
}

func (m *mockHandlerHelpersQuerier) DeleteLocation(ctx context.Context, id uuid.UUID) error {
	if m.DeleteLocationFunc != nil {
		return m.DeleteLocationFunc(ctx, id)
	}
	m.fail("DeleteLocation")
	return nil
}
func (m *mockHandlerHelpersQuerier) GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error) {
	if m.GetAllDailyForecastsAtLocationFunc != nil {
		return m.GetAllDailyForecastsAtLocationFunc(ctx, locationID)
	}
	m.fail("GetAllDailyForecastsAtLocation")
	return nil, nil
}
func (m *mockHandlerHelpersQuerier) GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error) {
	if m.GetAllHourlyForecastsAtLocationFunc != nil {
		return m.GetAllHourlyForecastsAtLocationFunc(ctx, locationID)
	}
	m.fail("GetAllHourlyForecastsAtLocation")
	return nil, nil
}
func (m *mockHandlerHelpersQuerier) GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
	if m.GetCurrentWeatherAtLocationFunc != nil {
		return m.GetCurrentWeatherAtLocationFunc(ctx, locationID)
	}
	m.fail("GetCurrentWeatherAtLocation")
	return nil, nil
}
func (m *mockHandlerHelpersQuerier) GetCurrentWeatherAtLocationFromAPI(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
	if m.GetCurrentWeatherAtLocationFromAPIFunc != nil {
		return m.GetCurrentWeatherAtLocationFromAPIFunc(ctx, arg)
	}
	m.fail("GetCurrentWeatherAtLocationFromAPI")
	return database.CurrentWeather{}, nil
}
func (m *mockHandlerHelpersQuerier) GetDailyForecastAtLocationAndDateFromAPI(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
	if m.GetDailyForecastAtLocationAndDateFromAPIFunc != nil {
		return m.GetDailyForecastAtLocationAndDateFromAPIFunc(ctx, arg)
	}
	m.fail("GetDailyForecastAtLocationAndDateFromAPI")
	return database.DailyForecast{}, nil
}
func (m *mockHandlerHelpersQuerier) GetHourlyForecastAtLocationAndTimeFromAPI(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
	if m.GetHourlyForecastAtLocationAndTimeFromAPIFunc != nil {
		return m.GetHourlyForecastAtLocationAndTimeFromAPIFunc(ctx, arg)
	}
	m.fail("GetHourlyForecastAtLocationAndTimeFromAPI")
	return database.HourlyForecast{}, nil
}
func (m *mockHandlerHelpersQuerier) GetLocationByAlias(ctx context.Context, alias string) (database.Location, error) {
	if m.GetLocationByAliasFunc != nil {
		return m.GetLocationByAliasFunc(ctx, alias)
	}
	m.fail("GetLocationByAlias")
	return database.Location{}, nil
}
func (m *mockHandlerHelpersQuerier) GetLocationByCoordinates(ctx context.Context, arg database.GetLocationByCoordinatesParams) (database.Location, error) {
	if m.GetLocationByCoordinatesFunc != nil {
		return m.GetLocationByCoordinatesFunc(ctx, arg)
	}
	m.fail("GetLocationByCoordinates")
	return database.Location{}, nil
}
func (m *mockHandlerHelpersQuerier) GetLocationByName(ctx context.Context, cityName string) (database.Location, error) {
	if m.GetLocationByNameFunc != nil {
		return m.GetLocationByNameFunc(ctx, cityName)
	}
	m.fail("GetLocationByName")
	return database.Location{}, nil
}
func (m *mockHandlerHelpersQuerier) GetUpcomingDailyForecastsAtLocation(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
	if m.GetUpcomingDailyForecastsAtLocationFunc != nil {
		return m.GetUpcomingDailyForecastsAtLocationFunc(ctx, arg)
	}
	m.fail("GetUpcomingDailyForecastsAtLocation")
	return nil, nil
}
func (m *mockHandlerHelpersQuerier) GetUpcomingHourlyForecastsAtLocation(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
	if m.GetUpcomingHourlyForecastsAtLocationFunc != nil {
		return m.GetUpcomingHourlyForecastsAtLocationFunc(ctx, arg)
	}
	m.fail("GetUpcomingHourlyForecastsAtLocation")
	return nil, nil
}
func (m *mockHandlerHelpersQuerier) ListLocations(ctx context.Context) ([]database.Location, error) {
	if m.ListLocationsFunc != nil {
		return m.ListLocationsFunc(ctx)
	}
	m.fail("ListLocations")
	return nil, nil
}
func (m *mockHandlerHelpersQuerier) UpdateCurrentWeather(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error) {
	if m.UpdateCurrentWeatherFunc != nil {
		return m.UpdateCurrentWeatherFunc(ctx, arg)
	}
	m.fail("UpdateCurrentWeather")
	return database.CurrentWeather{}, nil
}
func (m *mockHandlerHelpersQuerier) UpdateDailyForecast(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error) {
	if m.UpdateDailyForecastFunc != nil {
		return m.UpdateDailyForecastFunc(ctx, arg)
	}
	m.fail("UpdateDailyForecast")
	return database.DailyForecast{}, nil
}
func (m *mockHandlerHelpersQuerier) UpdateHourlyForecast(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error) {
	if m.UpdateHourlyForecastFunc != nil {
		return m.UpdateHourlyForecastFunc(ctx, arg)
	}
	m.fail("UpdateHourlyForecast")
	return database.HourlyForecast{}, nil
}
func (m *mockHandlerHelpersQuerier) UpdateTimezone(ctx context.Context, arg database.UpdateTimezoneParams) error {
	if m.UpdateTimezoneFunc != nil {
		return m.UpdateTimezoneFunc(ctx, arg)
	}
	m.fail("UpdateTimezone")
	return nil
}

// --- Tests ---

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
		name       string
		cityName   string
		setupMocks func(db *mockHandlerHelpersQuerier, geo *mockGeocodingService)
		check      func(t *testing.T, loc Location, err error)
	}{
		{
			name:     "Success: Alias Exists in DB",
			cityName: "wroclaw",
			setupMocks: func(db *mockHandlerHelpersQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					if alias != "wroclaw" {
						t.Errorf("expected alias 'wroclaw', got '%s'", alias)
					}
					return dbLocation, nil
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, expectedLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, expectedLocation)
				}
			},
		},
		{
			name:     "Success: Canonical Name Exists in DB",
			cityName: "wroclaw",
			setupMocks: func(db *mockHandlerHelpersQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					return expectedLocation, nil
				}
				db.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return dbLocation, nil
				}
				db.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					return database.LocationAlias{}, nil
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, expectedLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, expectedLocation)
				}
			},
		},
		{
			name:     "Success: New City",
			cityName: "newville",
			setupMocks: func(db *mockHandlerHelpersQuerier, geo *mockGeocodingService) {
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
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if loc.CityName != "Newville" {
					t.Errorf("expected city name 'Newville', got '%s'", loc.CityName)
				}
			},
		},
		{
			name:     "Failure: Geocoder Error",
			cityName: "errorcity",
			setupMocks: func(db *mockHandlerHelpersQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				geo.GeocodeFunc = func(cityName string) (Location, error) {
					return Location{}, errors.New("geocoder service unavailable")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name:     "Failure: DB Error on GetLocationByAlias",
			cityName: "dberror",
			setupMocks: func(db *mockHandlerHelpersQuerier, geo *mockGeocodingService) {
				db.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, errors.New("db connection lost")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockHandlerHelpersQuerier{t: t}
			geoMock := &mockGeocodingService{}
			tc.setupMocks(dbMock, geoMock)

			cfg := &apiConfig{dbQueries: dbMock, geocoder: geoMock, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			loc, err := cfg.getOrCreateLocation(ctx, tc.cityName)
			tc.check(t, loc, err)
		})
	}
}

func TestGetCachedOrFetchCurrentWeather(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}
	now := time.Now().UTC()

	apiWeather := []CurrentWeather{
		{SourceAPI: "gmp", Temperature: 22.0},
		{SourceAPI: "owm", Temperature: 23.0},
		{SourceAPI: "ometeo", Temperature: 21.0},
	}
	dbWeather := []database.CurrentWeather{
		{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "gmp", UpdatedAt: now, TemperatureC: sql.NullFloat64{Float64: 22.0, Valid: true}},
		{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "owm", UpdatedAt: now, TemperatureC: sql.NullFloat64{Float64: 23.0, Valid: true}},
		{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "ometeo", UpdatedAt: now, TemperatureC: sql.NullFloat64{Float64: 21.0, Valid: true}},
	}

	testCases := []struct {
		name       string
		setupMocks func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server)
		check      func(t *testing.T, weather []CurrentWeather, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiWeather)
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return string(cachedData), nil
				}
			},
			check: func(t *testing.T, weather []CurrentWeather, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(weather) != 3 {
					t.Fatalf("expected 3 weather items, got %d", len(weather))
				}
			},
		},
		{
			name: "Success: DB Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return "", redis.Nil
				}
				db.GetCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
					return dbWeather, nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error {
					return nil // Expect cache to be warmed
				}
			},
			check: func(t *testing.T, weather []CurrentWeather, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(weather) != 3 {
					t.Fatalf("expected 3 weather items, got %d", len(weather))
				}
			},
		},
		{
			name: "Success: API Fetch",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetCurrentWeatherAtLocationFunc = func(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
					return nil, sql.ErrNoRows
				}
				db.GetCurrentWeatherAtLocationFromAPIFunc = func(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, sql.ErrNoRows
				}
				db.CreateCurrentWeatherFunc = func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
					return database.CurrentWeather{}, nil
				}
				db.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, weather []CurrentWeather, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(weather) != 3 {
					t.Fatalf("expected 3 weather items, got %d", len(weather))
				}
			},
		},
	}

	// --- Mock API Server ---
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		if strings.Contains(r.URL.Path, "gmp") {
			data, _ = os.ReadFile("testdata/current_weather_gmp.json")
		} else if strings.Contains(r.URL.Path, "owm") {
			data, _ = os.ReadFile("testdata/current_weather_owm.json")
		} else {
			data, _ = os.ReadFile("testdata/current_weather_ometeo.json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer mockServer.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockHandlerHelpersQuerier{t: t}
			cacheMock := &mockCache{}
			tc.setupMocks(dbMock, cacheMock, mockServer)

			cfg := &apiConfig{
				dbQueries:        dbMock,
				cache:            cacheMock,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
				gmpWeatherURL:    mockServer.URL + "/gmp",
				owmWeatherURL:    mockServer.URL + "/owm",
				ometeoWeatherURL: mockServer.URL + "/ometeo",
				httpClient:       mockServer.Client(),
				gmpKey:           "dummy",
				owmKey:           "dummy",
			}

			weather, err := cfg.getCachedOrFetchCurrentWeather(ctx, location)
			tc.check(t, weather, err)
		})
	}
}

func TestGetCachedOrFetchDailyForecast(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}
	now := time.Now().UTC()

	apiForecast := []DailyForecast{{SourceAPI: "gmp", MaxTemp: 25.0}}
	dbForecast := []database.DailyForecast{{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "gmp", UpdatedAt: now, ForecastDate: now}}

	testCases := []struct {
		name       string
		setupMocks func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server)
		check      func(t *testing.T, forecast []DailyForecast, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiForecast)
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return string(cachedData), nil
				}
			},
			check: func(t *testing.T, forecast []DailyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) == 0 {
					t.Fatal("expected forecast items, got 0")
				}
			},
		},
		{
			name: "Success: DB Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingDailyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
					return dbForecast, nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []DailyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) != 1 {
					t.Fatalf("expected 1 forecast item, got %d", len(forecast))
				}
			},
		},
		{
			name: "Success: API Fetch",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingDailyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
					return nil, sql.ErrNoRows
				}
				db.GetDailyForecastAtLocationAndDateFromAPIFunc = func(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, sql.ErrNoRows
				}
				db.CreateDailyForecastFunc = func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
					return database.DailyForecast{}, nil
				}
				db.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []DailyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				// 5 days from 3 APIs
				if len(forecast) != 15 {
					t.Fatalf("expected 15 forecast items, got %d", len(forecast))
				}
			},
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		if strings.Contains(r.URL.Path, "gmp") {
			data, _ = os.ReadFile("testdata/daily_forecast_gmp.json")
		} else if strings.Contains(r.URL.Path, "owm") {
			data, _ = os.ReadFile("testdata/daily_forecast_owm.json")
		} else {
			data, _ = os.ReadFile("testdata/daily_forecast_ometeo.json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer mockServer.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockHandlerHelpersQuerier{t: t}
			cacheMock := &mockCache{}
			tc.setupMocks(dbMock, cacheMock, mockServer)

			cfg := &apiConfig{
				dbQueries:        dbMock,
				cache:            cacheMock,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
				gmpWeatherURL:    mockServer.URL + "/gmp",
				owmWeatherURL:    mockServer.URL + "/owm",
				ometeoWeatherURL: mockServer.URL + "/ometeo",
				httpClient:       mockServer.Client(),
				gmpKey:           "dummy",
				owmKey:           "dummy",
			}

			forecast, err := cfg.getCachedOrFetchDailyForecast(ctx, location)
			tc.check(t, forecast, err)
		})
	}
}

func TestGetCachedOrFetchHourlyForecast(t *testing.T) {
	ctx := context.Background()
	location := Location{LocationID: uuid.New(), CityName: "Testville", Latitude: 51.11, Longitude: 17.04}
	now := time.Now().UTC()

	apiForecast := []HourlyForecast{{SourceAPI: "gmp", Temperature: 15.0}}
	dbForecast := []database.HourlyForecast{{ID: uuid.New(), LocationID: location.LocationID, SourceApi: "gmp", UpdatedAt: now, ForecastDatetimeUtc: now}}

	testCases := []struct {
		name       string
		setupMocks func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server)
		check      func(t *testing.T, forecast []HourlyForecast, err error)
	}{
		{
			name: "Success: Redis Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cachedData, _ := json.Marshal(apiForecast)
				cache.getFunc = func(ctx context.Context, key string) (string, error) {
					return string(cachedData), nil
				}
			},
			check: func(t *testing.T, forecast []HourlyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) == 0 {
					t.Fatal("expected forecast items, got 0")
				}
			},
		},
		{
			name: "Success: DB Hit",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingHourlyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
					return dbForecast, nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []HourlyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(forecast) != 1 {
					t.Fatalf("expected 1 forecast item, got %d", len(forecast))
				}
			},
		},
		{
			name: "Success: API Fetch",
			setupMocks: func(db *mockHandlerHelpersQuerier, cache *mockCache, server *httptest.Server) {
				cache.getFunc = func(ctx context.Context, key string) (string, error) { return "", redis.Nil }
				db.GetUpcomingHourlyForecastsAtLocationFunc = func(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
					return nil, sql.ErrNoRows
				}
				db.GetHourlyForecastAtLocationAndTimeFromAPIFunc = func(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, sql.ErrNoRows
				}
				db.CreateHourlyForecastFunc = func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
					return database.HourlyForecast{}, nil
				}
				db.UpdateTimezoneFunc = func(ctx context.Context, arg database.UpdateTimezoneParams) error {
					return nil
				}
				cache.setFunc = func(ctx context.Context, key string, value any, expiration time.Duration) error { return nil }
			},
			check: func(t *testing.T, forecast []HourlyForecast, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				// 24 hours from 3 APIs
				if len(forecast) != 72 {
					t.Fatalf("expected 72 forecast items, got %d", len(forecast))
				}
			},
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		if strings.Contains(r.URL.Path, "gmp") {
			data, _ = os.ReadFile("testdata/hourly_forecast_gmp.json")
		} else if strings.Contains(r.URL.Path, "owm") {
			data, _ = os.ReadFile("testdata/hourly_forecast_owm.json")
		} else {
			data, _ = os.ReadFile("testdata/hourly_forecast_ometeo.json")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer mockServer.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbMock := &mockHandlerHelpersQuerier{t: t}
			cacheMock := &mockCache{}
			tc.setupMocks(dbMock, cacheMock, mockServer)

			cfg := &apiConfig{
				dbQueries:        dbMock,
				cache:            cacheMock,
				logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
				gmpWeatherURL:    mockServer.URL + "/gmp",
				owmWeatherURL:    mockServer.URL + "/owm",
				ometeoWeatherURL: mockServer.URL + "/ometeo",
				httpClient:       mockServer.Client(),
				gmpKey:           "dummy",
				owmKey:           "dummy",
			}

			forecast, err := cfg.getCachedOrFetchHourlyForecast(ctx, location)
			tc.check(t, forecast, err)
		})
	}
}
