package main

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

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
			setupMocks: func(cfg *testAPIConfig) {
				cfg.mockDB.GetLocationByAliasFunc = func(ctx context.Context, alias string) (database.Location, error) {
					return database.Location{}, sql.ErrNoRows
				}
				cfg.mockGeo.GeocodeFunc = func(cityName string) (Location, error) {
					return expectedLocation, nil
				}
				cfg.mockDB.GetLocationByNameFunc = func(ctx context.Context, cityName string) (database.Location, error) {
					return dbLocation, nil
				}
				cfg.mockDB.CreateLocationAliasFunc = func(ctx context.Context, arg database.CreateLocationAliasParams) (database.LocationAlias, error) {
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