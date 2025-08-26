package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// This file provides the application's geocoding capabilities, which are essential
// for converting between city names and geographical coordinates (latitude/longitude).
// It abstracts the geocoding provider behind a `GeocodingService` interface, making
// the application independent of a specific service like the Google Maps Platform.
// This design allows for easier testing and future replacement of the geocoding provider.

// ErrNoResultsFound is returned when a geocoding query yields no results.
var ErrNoResultsFound = errors.New("no results found for the given query")

// GeocodingService defines a generic interface for geocoding operations.
// Using an interface decouples the application's core logic from the concrete
// implementation of a geocoding client, which simplifies testing and allows for
// different providers to be used.
type GeocodingService interface {
	Geocode(cityName string) (Location, error)
	ReverseGeocode(lat, lng float64) (Location, error)
}

// GmpGeocodingService is an implementation of GeocodingService that uses the Google Maps Platform API.
type GmpGeocodingService struct {
	gmpKey        string
	gmpGeocodeURL string
	httpClient    *http.Client
}

// NewGmpGeocodingService creates a new GmpGeocodingService.
func NewGmpGeocodingService(gmpKey, gmpGeocodeURL string, httpClient *http.Client) *GmpGeocodingService {
	return &GmpGeocodingService{
		gmpKey:        gmpKey,
		gmpGeocodeURL: gmpGeocodeURL,
		httpClient:    httpClient,
	}
}

// Geocode and ReverseGeocode are wrappers that prepare the specific parameters
// for their respective operations (by address or by lat/lng) and then delegate
// the core API call logic to performGeocodeRequest.
func (s *GmpGeocodingService) Geocode(cityName string) (Location, error) {
	params := map[string]string{
		"address": cityName,
	}
	return s.performGeocodeRequest(params)
}

func (s *GmpGeocodingService) ReverseGeocode(lat, lng float64) (Location, error) {
	params := map[string]string{
		"latlng": fmt.Sprintf("%.2f,%.2f", lat, lng),
	}
	return s.performGeocodeRequest(params)
}

// performGeocodeRequest handles the actual HTTP request to the Google Geocoding API.
func (s *GmpGeocodingService) performGeocodeRequest(queryParams map[string]string) (Location, error) {
	baseURL, err := url.Parse(s.gmpGeocodeURL + "json")
	if err != nil {
		return Location{}, fmt.Errorf("failed to parse base geocode URL: %w", err)
	}

	q := baseURL.Query()
	q.Set("key", s.gmpKey)
	for key, value := range queryParams {
		q.Set(key, value)
	}
	baseURL.RawQuery = q.Encode()

	resp, err := s.httpClient.Get(baseURL.String())
	if err != nil {
		return Location{}, fmt.Errorf("geocoding API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Location{}, fmt.Errorf("geocoding API request returned non-200 status: %s", resp.Status)
	}

	var responseJSON Response
	if err := json.NewDecoder(resp.Body).Decode(&responseJSON); err != nil {
		return Location{}, fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	if responseJSON.Status != "OK" {
		if responseJSON.Status == "ZERO_RESULTS" {
			return Location{}, ErrNoResultsFound
		}
		return Location{}, fmt.Errorf("geocoding API returned status: %s", responseJSON.Status)
	}

	if len(responseJSON.Results) == 0 {
		return Location{}, ErrNoResultsFound
	}

	location := parseLocationFromResult(responseJSON.Results[0])
	return location, nil
}

// parseLocationFromResult extracts Location data from a single geocoding API result.
func parseLocationFromResult(result Result) Location {
	var location Location
	location.Latitude = result.Geometry.Location.Latitude
	location.Longitude = result.Geometry.Location.Longitude

	for _, component := range result.AddressComponents {
		for _, componentType := range component.Types {
			switch componentType {
			case "locality":
				location.CityName = component.LongName
			case "country":
				location.CountryCode = component.ShortName
			}
		}
	}
	return location
}

// The following structs represent the structure of the Google Geocoding API JSON response.
// They are used by the json decoder to parse the API's output.
type Response struct {
	Results []Result `json:"results"`
	Status  string   `json:"status"`
}

type Result struct {
	AddressComponents []AddressComponent `json:"address_components"`
	Geometry          Geometry           `json:"geometry"`
}

type AddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

type Geometry struct {
	Location LocationData `json:"location"`
}

type LocationData struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}
