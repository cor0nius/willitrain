package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cor0nius/willitrain/internal/database"
)

// This file contains helper functions related to location management.
// It includes logic for retrieving or creating canonical location records
// from the database, handling aliases, and parsing location data from HTTP requests.

// getOrCreateLocation is an intelligent helper to retrieve a location from the database.
// It handles city name aliases to avoid duplicate entries and minimize external API calls.
//
// The logic is as follows:
// 1. Normalize the input cityName to create a standardized alias.
// 2. Attempt to find the location using this alias in the `location_aliases` table.
// 3. If found, return the location.
// 4. If not found, call the geocoding service to get the canonical location data.
// 5. Check if a location with the canonical name already exists in the `locations` table.
// 6. If it exists, create a new alias for the user's original input and link it to the existing location.
// 7. If no location exists by either alias or canonical name, create a new location record.
// 8. Finally, create aliases for both the user's normalized input and the canonical name to ensure future lookups are successful.
func (cfg *apiConfig) getOrCreateLocation(ctx context.Context, cityName string) (Location, error) {
	alias, err := normalizeCityName(cityName)
	if err != nil {
		return Location{}, fmt.Errorf("could not normalize city name: %w", err)
	}

	dbLocation, err := cfg.dbQueries.GetLocationByAlias(ctx, alias)
	if err == nil {
		cfg.logger.Debug("location found by alias", "alias", alias, "city", dbLocation.CityName)
		return databaseLocationToLocation(dbLocation), nil
	}
	if err != sql.ErrNoRows {
		return Location{}, fmt.Errorf("database error when fetching location by alias: %w", err)
	}

	cfg.logger.Debug("alias not found, geocoding", "alias", alias, "original_city", cityName)
	geocodedLocation, geoErr := cfg.geocoder.Geocode(cityName)
	if geoErr != nil {
		return Location{}, fmt.Errorf("could not geocode city '%s': %w", cityName, geoErr)
	}

	dbLocation, err = cfg.dbQueries.GetLocationByName(ctx, geocodedLocation.CityName)
	if err == nil {
		cfg.logger.Debug("canonical location found in db, creating new alias", "city", dbLocation.CityName, "alias", alias)
		_, aliasErr := cfg.dbQueries.CreateLocationAlias(ctx, database.CreateLocationAliasParams{Alias: alias, LocationID: dbLocation.ID})
		if aliasErr != nil {
			cfg.logger.Warn("could not create location alias", "alias", alias, "location_id", dbLocation.ID, "error", aliasErr)
		}
		return databaseLocationToLocation(dbLocation), nil
	}
	if err != sql.ErrNoRows {
		return Location{}, fmt.Errorf("database error when fetching location by canonical name: %w", err)
	}

	cfg.logger.Debug("no location found, creating new location and aliases", "city", geocodedLocation.CityName)
	persistedLocation, createErr := cfg.dbQueries.CreateLocation(ctx, locationToCreateLocationParams(geocodedLocation))
	if createErr != nil {
		return Location{}, fmt.Errorf("could not persist new location: %w", createErr)
	}

	_, aliasErr := cfg.dbQueries.CreateLocationAlias(ctx, database.CreateLocationAliasParams{Alias: alias, LocationID: persistedLocation.ID})
	if aliasErr != nil {
		cfg.logger.Warn("could not create user input alias", "alias", alias, "location_id", persistedLocation.ID, "error", aliasErr)
	}

	canonicalAlias, err := normalizeCityName(persistedLocation.CityName)
	if err != nil {
		cfg.logger.Error("could not normalize canonical city name", "city", persistedLocation.CityName, "error", err)
	} else if alias != canonicalAlias {
		_, aliasErr = cfg.dbQueries.CreateLocationAlias(ctx, database.CreateLocationAliasParams{Alias: canonicalAlias, LocationID: persistedLocation.ID})
		if aliasErr != nil {
			cfg.logger.Warn("could not create canonical alias", "alias", canonicalAlias, "location_id", persistedLocation.ID, "error", aliasErr)
		}
	}

	return databaseLocationToLocation(persistedLocation), nil
}

// getLocationFromRequest extracts location details from an HTTP request, supporting both
// city name and latitude/longitude query parameters. It uses getOrCreateLocation to
// ensure a consistent and canonical location record is used.
func (cfg *apiConfig) getLocationFromRequest(r *http.Request) (Location, error) {
	ctx := r.Context()
	cityName := r.URL.Query().Get("city")
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if cityName != "" {
		return cfg.getOrCreateLocation(ctx, cityName)
	}

	if latStr != "" && lonStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return Location{}, fmt.Errorf("invalid latitude: %v", err)
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return Location{}, fmt.Errorf("invalid longitude: %v", err)
		}

		location, err := cfg.geocoder.ReverseGeocode(lat, lon)
		if err != nil {
			return Location{}, fmt.Errorf("could not reverse geocode coordinates: %w", err)
		}

		return cfg.getOrCreateLocation(ctx, location.CityName)
	}

	return Location{}, fmt.Errorf("either city or lat/lon query parameters are required")
}
