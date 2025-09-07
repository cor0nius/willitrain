package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

// --- Tests ---

func TestGetOrCreateLocation(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name       string
		cityName   string
		setupMocks func(cfg *testAPIConfig)
		check      func(t *testing.T, loc Location, err error)
	}{
		{
			name:     "Success: Alias Exists in DB",
			cityName: "wroclaw",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					if alias != "wroclaw" {
						t.Errorf("expected alias 'wroclaw', got '%s'", alias)
					}
					return MockDBLocation, nil
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, MockLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, MockLocation)
				}
			},
		},
		{
			name:     "Success: Canonical Name Exists in DB",
			cityName: "wroclaw",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return MockLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return MockDBLocation, nil
				}
				cfg.mockDB.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					return database.LocationAlias{}, nil
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, MockLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, MockLocation)
				}
			},
		},
		{
			name:     "Success: New City",
			cityName: "newville",
			setupMocks: func(cfg *testAPIConfig) {
				newLocation := Location{LocationID: uuid.New(), CityName: "Newville", CountryCode: "NV"}
				dbNewLocation := database.Location{ID: newLocation.LocationID, CityName: newLocation.CityName, CountryCode: newLocation.CountryCode}
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return newLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateLocationFunc = func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
					return dbNewLocation, nil
				}
				cfg.mockDB.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
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
			name:     "Failure: City Name Contains Invalid UTF-8",
			cityName: "abc\x80def",
			setupMocks: func(cfg *testAPIConfig) {
				// No mocks needed since normalization will fail first
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name:     "Failure: Geocoder Error",
			cityName: "errorcity",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
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
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, errors.New("db connection lost")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name:     "Failure: DB Error on GetLocationByName",
			cityName: "dberror",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return MockLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, errors.New("db connection lost")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name:     "Failure: Alias Creation Error after GetLocationByName",
			cityName: "aliascreationerror",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return MockLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return MockDBLocation, nil
				}
				cfg.mockDB.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					return database.LocationAlias{}, errors.New("could not create alias")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, MockLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, MockLocation)
				}
			},
		},
		{
			name:     "Failure: Location Creation Error",
			cityName: "locationcreationerror",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return MockLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateLocationFunc = func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
					return database.Location{}, errors.New("could not create location")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name:     "Failure: User Alias Creation Error",
			cityName: "useraliascreationerror",
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return MockLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateLocationFunc = func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
					return MockDBLocation, nil
				}
				cfg.mockDB.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					return database.LocationAlias{}, errors.New("could not create user alias")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, MockLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, MockLocation)
				}
			},
		},
		{
			name:     "Failure: Canonical Name Normalization Error",
			cityName: "canonicalnormalizationerror",
			setupMocks: func(cfg *testAPIConfig) {
				// This mock location has an invalid city name that will cause normalizeCityName to fail
				mockLocationWithInvalidName := Location{LocationID: uuid.New(), CityName: "abc\x80def", CountryCode: "PL"}
				dbMockLocationWithInvalidName := database.Location{ID: mockLocationWithInvalidName.LocationID, CityName: mockLocationWithInvalidName.CityName, CountryCode: mockLocationWithInvalidName.CountryCode}

				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return mockLocationWithInvalidName, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateLocationFunc = func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
					return dbMockLocationWithInvalidName, nil
				}
				cfg.mockDB.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					return database.LocationAlias{}, nil
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if loc.CityName != "abc\x80def" {
					t.Errorf("unexpected city name. got %s", loc.CityName)
				}
			},
		},
		{
			name:     "Failure: Canonical Alias Creation Error",
			cityName: "Breslau", // User input that geocodes to "Wrocław"
			setupMocks: func(cfg *testAPIConfig) {
				geocodedLocation := Location{LocationID: uuid.New(), CityName: "Wrocław", CountryCode: "PL"}
				dbGeocodedLocation := database.Location{ID: geocodedLocation.LocationID, CityName: geocodedLocation.CityName, CountryCode: geocodedLocation.CountryCode}

				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return geocodedLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockDB.CreateLocationFunc = func(ctx context.Context, arg database.CreateLocationParams) (database.Location, error) {
					return dbGeocodedLocation, nil
				}

				// This mock will be called twice. First for the user alias, then for the canonical alias.
				// We want the second call to fail.
				callCount := 0
				cfg.mockDB.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
					callCount++
					if callCount == 1 { // First call for "breslau"
						return database.LocationAlias{}, nil
					}
					// Second call for "wroclaw"
					return database.LocationAlias{}, errors.New("could not create canonical alias")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if loc.CityName != "Wrocław" {
					t.Errorf("unexpected city name. got %s, want Wrocław", loc.CityName)
				}
			},
		},
		{
			name:     "Failure: transformer.String returned an error (artificial)",
			cityName: "wroclaw",
			setupMocks: func(cfg *testAPIConfig) {
				mockTransformer := &mockTransformer{
					errToReturn: errors.New("artificial transform error"),
				}
				originalTransformer := transformer
				transformer = mockTransformer
				t.Cleanup(func() {
					transformer = originalTransformer
				})
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
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg)

			loc, err := testCfg.apiConfig.getOrCreateLocation(ctx, tc.cityName)
			tc.check(t, loc, err)
		})
	}
}

func TestGetLocationFromRequest(t *testing.T) {
	testCases := []struct {
		name       string
		req        *http.Request
		setupMocks func(cfg *testAPIConfig)
		check      func(t *testing.T, loc Location, err error)
	}{
		{
			name: "Success: With City",
			req:  httptest.NewRequest("GET", "/?city=wroclaw", nil),
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					if alias != "wroclaw" {
						t.Errorf("expected alias 'wroclaw', got '%s'", alias)
					}
					return MockDBLocation, nil
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, MockLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, MockLocation)
				}
			},
		},
		{
			name: "Success: With Lat/Lon",
			req:  httptest.NewRequest("GET", "/?lat=51.1&lon=17.03", nil),
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockGeo.ReverseGeocodeFunc = func(lat, lon float64) (Location, error) {
					return MockLocation, nil
				}
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return MockDBLocation, nil
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !reflect.DeepEqual(loc, MockLocation) {
					t.Errorf("unexpected location. got %+v, want %+v", loc, MockLocation)
				}
			},
		},
		{
			name: "Failure: Invalid Latitude",
			req:  httptest.NewRequest("GET", "/?lat=invalid&lon=17.03", nil),
			setupMocks: func(cfg *testAPIConfig) {
				// No mocks needed
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name: "Failure: Invalid Longitude",
			req:  httptest.NewRequest("GET", "/?lat=51.1&lon=invalid", nil),
			setupMocks: func(cfg *testAPIConfig) {
				// No mocks needed
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name: "Failure: ReverseGeocode Error",
			req:  httptest.NewRequest("GET", "/?lat=51.1&lon=17.03", nil),
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockGeo.ReverseGeocodeFunc = func(lat, lon float64) (Location, error) {
					return Location{}, errors.New("reverse geocode failed")
				}
			},
			check: func(t *testing.T, loc Location, err error) {
				if err == nil {
					t.Fatal("expected an error, but got nil")
				}
			},
		},
		{
			name: "Failure: Missing Parameters",
			req:  httptest.NewRequest("GET", "/", nil),
			setupMocks: func(cfg *testAPIConfig) {
				// No mocks needed
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
			testCfg := newTestAPIConfig(t)
			tc.setupMocks(testCfg)

			loc, err := testCfg.apiConfig.getLocationFromRequest(tc.req)
			tc.check(t, loc, err)
		})
	}
}
