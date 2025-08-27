package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
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

// mockQuerier is a comprehensive, safe mock for the database.Querier interface.
// It fails the test if any unexpected method is called.
type mockQuerier struct {
	t *testing.T

	// Scheduler test fields
	locationsToReturn []database.Location
	listLocationsErr  error
	mu                sync.Mutex

	createCurrentWeatherCalls     int
	createDailyForecastCalls      int
	createHourlyForecastCalls     int
	getCurrentWeatherFromAPICalls int
	getDailyForecastFromAPICalls  int
	getHourlyForecastFromAPICalls int
	updateCurrentWeatherCalls     int
	updateDailyForecastCalls      int
	updateHourlyForecastCalls     int

	// Handler helpers test fields
	CreateCurrentWeatherFunc                      func(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error)
	CreateDailyForecastFunc                       func(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error)
	CreateHourlyForecastFunc                      func(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error)
	CreateLocationFunc                            func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error)
	CreateLocationAliasFunc                       func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error)
	DeleteAllCurrentWeatherFunc                   func(ctx context.Context) error
	DeleteAllDailyForecastsFunc                   func(ctx context.Context) error
	DeleteAllHourlyForecastsFunc                  func(ctx context.Context) error
	DeleteAllLocationsFunc                        func(ctx context.Context) error
	DeleteCurrentWeatherAtLocationFunc            func(ctx context.Context, locationID uuid.UUID) error
	DeleteDailyForecastsAtLocationFunc            func(ctx context.Context, locationID uuid.UUID) error
	DeleteHourlyForecastsAtLocationFunc           func(ctx context.Context, locationID uuid.UUID) error
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

func (m *mockQuerier) fail(method string) {
	m.t.Fatalf("unexpected call to mockQuerier method: %s", method)
}

func (m *mockQuerier) CreateCurrentWeather(ctx context.Context, arg database.CreateCurrentWeatherParams) (database.CurrentWeather, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCurrentWeatherCalls++
	if m.CreateCurrentWeatherFunc != nil {
		return m.CreateCurrentWeatherFunc(ctx, arg)
	}
	return database.CurrentWeather{}, nil
}
func (m *mockQuerier) CreateDailyForecast(ctx context.Context, arg database.CreateDailyForecastParams) (database.DailyForecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createDailyForecastCalls++
	if m.CreateDailyForecastFunc != nil {
		return m.CreateDailyForecastFunc(ctx, arg)
	}
	return database.DailyForecast{}, nil
}
func (m *mockQuerier) CreateHourlyForecast(ctx context.Context, arg database.CreateHourlyForecastParams) (database.HourlyForecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createHourlyForecastCalls++
	if m.CreateHourlyForecastFunc != nil {
		return m.CreateHourlyForecastFunc(ctx, arg)
	}
	return database.HourlyForecast{}, nil
}
func (m *mockQuerier) CreateLocation(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
	if m.CreateLocationFunc != nil {
		return m.CreateLocationFunc(ctx, arg)
	}
	m.fail("CreateLocation")
	return database.Location{}, nil
}
func (m *mockQuerier) CreateLocationAlias(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
	if m.CreateLocationAliasFunc != nil {
		return m.CreateLocationAliasFunc(ctx, arg)
	}
	m.fail("CreateLocationAlias")
	return database.LocationAlias{}, nil
}
func (m *mockQuerier) DeleteAllCurrentWeather(ctx context.Context) error {
	if m.DeleteAllCurrentWeatherFunc != nil {
		return m.DeleteAllCurrentWeatherFunc(ctx)
	}
	return nil
}
func (m *mockQuerier) DeleteAllDailyForecasts(ctx context.Context) error {
	if m.DeleteAllDailyForecastsFunc != nil {
		return m.DeleteAllDailyForecastsFunc(ctx)
	}
	return nil
}
func (m *mockQuerier) DeleteAllHourlyForecasts(ctx context.Context) error {
	if m.DeleteAllHourlyForecastsFunc != nil {
		return m.DeleteAllHourlyForecastsFunc(ctx)
	}
	return nil
}
func (m *mockQuerier) DeleteAllLocations(ctx context.Context) error {
	if m.DeleteAllLocationsFunc != nil {
		return m.DeleteAllLocationsFunc(ctx)
	}
	return nil
}

func (m *mockQuerier) DeleteCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) error {
	if m.DeleteCurrentWeatherAtLocationFunc != nil {
		return m.DeleteCurrentWeatherAtLocationFunc(ctx, locationID)
	}
	return nil
}

func (m *mockQuerier) DeleteDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) error {
	if m.DeleteDailyForecastsAtLocationFunc != nil {
		return m.DeleteDailyForecastsAtLocationFunc(ctx, locationID)
	}
	return nil
}

func (m *mockQuerier) DeleteHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) error {
	if m.DeleteHourlyForecastsAtLocationFunc != nil {
		return m.DeleteHourlyForecastsAtLocationFunc(ctx, locationID)
	}
	return nil
}

func (m *mockQuerier) DeleteLocation(ctx context.Context, id uuid.UUID) error {
	if m.DeleteLocationFunc != nil {
		return m.DeleteLocationFunc(ctx, id)
	}
	m.fail("DeleteLocation")
	return nil
}
func (m *mockQuerier) GetAllDailyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.DailyForecast, error) {
	if m.GetAllDailyForecastsAtLocationFunc != nil {
		return m.GetAllDailyForecastsAtLocationFunc(ctx, locationID)
	}
	m.fail("GetAllDailyForecastsAtLocation")
	return nil, nil
}
func (m *mockQuerier) GetAllHourlyForecastsAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.HourlyForecast, error) {
	if m.GetAllHourlyForecastsAtLocationFunc != nil {
		return m.GetAllHourlyForecastsAtLocationFunc(ctx, locationID)
	}
	m.fail("GetAllHourlyForecastsAtLocation")
	return nil, nil
}
func (m *mockQuerier) GetCurrentWeatherAtLocation(ctx context.Context, locationID uuid.UUID) ([]database.CurrentWeather, error) {
	if m.GetCurrentWeatherAtLocationFunc != nil {
		return m.GetCurrentWeatherAtLocationFunc(ctx, locationID)
	}
	m.fail("GetCurrentWeatherAtLocation")
	return nil, nil
}
func (m *mockQuerier) GetCurrentWeatherAtLocationFromAPI(ctx context.Context, arg database.GetCurrentWeatherAtLocationFromAPIParams) (database.CurrentWeather, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCurrentWeatherFromAPICalls++
	if m.GetCurrentWeatherAtLocationFromAPIFunc != nil {
		return m.GetCurrentWeatherAtLocationFromAPIFunc(ctx, arg)
	}
	m.fail("GetCurrentWeatherAtLocationFromAPI")
	return database.CurrentWeather{}, nil
}
func (m *mockQuerier) GetDailyForecastAtLocationAndDateFromAPI(ctx context.Context, arg database.GetDailyForecastAtLocationAndDateFromAPIParams) (database.DailyForecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getDailyForecastFromAPICalls++
	if m.GetDailyForecastAtLocationAndDateFromAPIFunc != nil {
		return m.GetDailyForecastAtLocationAndDateFromAPIFunc(ctx, arg)
	}
	m.fail("GetDailyForecastAtLocationAndDateFromAPI")
	return database.DailyForecast{}, nil
}
func (m *mockQuerier) GetHourlyForecastAtLocationAndTimeFromAPI(ctx context.Context, arg database.GetHourlyForecastAtLocationAndTimeFromAPIParams) (database.HourlyForecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getHourlyForecastFromAPICalls++
	if m.GetHourlyForecastAtLocationAndTimeFromAPIFunc != nil {
		return m.GetHourlyForecastAtLocationAndTimeFromAPIFunc(ctx, arg)
	}
	m.fail("GetHourlyForecastAtLocationAndTimeFromAPI")
	return database.HourlyForecast{}, nil
}
func (m *mockQuerier) GetLocationByAlias(ctx context.Context, alias string) (database.Location, error) {
	if m.GetLocationByAliasFunc != nil {
		return m.GetLocationByAliasFunc(ctx, alias)
	}
	m.fail("GetLocationByAlias")
	return database.Location{}, nil
}
func (m *mockQuerier) GetLocationByCoordinates(ctx context.Context, arg database.GetLocationByCoordinatesParams) (database.Location, error) {
	if m.GetLocationByCoordinatesFunc != nil {
		return m.GetLocationByCoordinatesFunc(ctx, arg)
	}
	m.fail("GetLocationByCoordinates")
	return database.Location{}, nil
}
func (m *mockQuerier) GetLocationByName(ctx context.Context, cityName string) (database.Location, error) {
	if m.GetLocationByNameFunc != nil {
		return m.GetLocationByNameFunc(ctx, cityName)
	}
	m.fail("GetLocationByName")
	return database.Location{}, nil
}
func (m *mockQuerier) GetUpcomingDailyForecastsAtLocation(ctx context.Context, arg database.GetUpcomingDailyForecastsAtLocationParams) ([]database.DailyForecast, error) {
	if m.GetUpcomingDailyForecastsAtLocationFunc != nil {
		return m.GetUpcomingDailyForecastsAtLocationFunc(ctx, arg)
	}
	m.fail("GetUpcomingDailyForecastsAtLocation")
	return nil, nil
}
func (m *mockQuerier) GetUpcomingHourlyForecastsAtLocation(ctx context.Context, arg database.GetUpcomingHourlyForecastsAtLocationParams) ([]database.HourlyForecast, error) {
	if m.GetUpcomingHourlyForecastsAtLocationFunc != nil {
		return m.GetUpcomingHourlyForecastsAtLocationFunc(ctx, arg)
	}
	m.fail("GetUpcomingHourlyForecastsAtLocation")
	return nil, nil
}
func (m *mockQuerier) ListLocations(ctx context.Context) ([]database.Location, error) {
	if m.ListLocationsFunc != nil {
		return m.ListLocationsFunc(ctx)
	}
	m.fail("ListLocations")
	return nil, nil
}
func (m *mockQuerier) UpdateCurrentWeather(ctx context.Context, arg database.UpdateCurrentWeatherParams) (database.CurrentWeather, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCurrentWeatherCalls++
	if m.UpdateCurrentWeatherFunc != nil {
		return m.UpdateCurrentWeatherFunc(ctx, arg)
	}
	m.fail("UpdateCurrentWeather")
	return database.CurrentWeather{}, nil
}
func (m *mockQuerier) UpdateDailyForecast(ctx context.Context, arg database.UpdateDailyForecastParams) (database.DailyForecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateDailyForecastCalls++
	if m.UpdateDailyForecastFunc != nil {
		return m.UpdateDailyForecastFunc(ctx, arg)
	}
	m.fail("UpdateDailyForecast")
	return database.DailyForecast{}, nil
}
func (m *mockQuerier) UpdateHourlyForecast(ctx context.Context, arg database.UpdateHourlyForecastParams) (database.HourlyForecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateHourlyForecastCalls++
	if m.UpdateHourlyForecastFunc != nil {
		return m.UpdateHourlyForecastFunc(ctx, arg)
	}
	m.fail("UpdateHourlyForecast")
	return database.HourlyForecast{}, nil
}
func (m *mockQuerier) UpdateTimezone(ctx context.Context, arg database.UpdateTimezoneParams) error {
	if m.UpdateTimezoneFunc != nil {
		return m.UpdateTimezoneFunc(ctx, arg)
	}
	return nil
}

type testAPIConfig struct {
	*apiConfig
	mockDB    *mockQuerier
	mockCache *mockCache
	mockGeo   *mockGeocodingService
}

func newTestAPIConfig(t *testing.T) *testAPIConfig {
	mockDB := &mockQuerier{t: t}
	mockCache := &mockCache{}
	mockGeo := &mockGeocodingService{}

	return &testAPIConfig{
		apiConfig: &apiConfig{
			dbQueries:  mockDB,
			cache:      mockCache,
			geocoder:   mockGeo,
			logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			httpClient: &http.Client{},
		},
		mockDB:    mockDB,
		mockCache: mockCache,
		mockGeo:   mockGeo,
	}
}
