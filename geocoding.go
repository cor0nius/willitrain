package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var ErrNoResultsFound = errors.New("no results found for the given query")

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

func (cfg *apiConfig) performGeocodeRequest(queryParams map[string]string) (Location, error) {
	baseURL, err := url.Parse(cfg.gmpGeocodeURL + "json")
	if err != nil {
		return Location{}, fmt.Errorf("failed to parse base geocode URL: %w", err)
	}

	q := baseURL.Query()
	q.Set("key", cfg.gmpKey)
	for key, value := range queryParams {
		q.Set(key, value)
	}
	baseURL.RawQuery = q.Encode()

	resp, err := cfg.httpClient.Get(baseURL.String())
	if err != nil {
		return Location{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Location{}, fmt.Errorf("geocoding API request failed with status: %s", resp.Status)
	}

	var responseJSON Response
	if err := json.NewDecoder(resp.Body).Decode(&responseJSON); err != nil {
		return Location{}, err
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

func (cfg *apiConfig) Geocode(cityName string) (Location, error) {
	params := map[string]string{
		"address": cityName,
	}
	return cfg.performGeocodeRequest(params)
}

func (cfg *apiConfig) ReverseGeocode(lat, lng float64) (Location, error) {
	params := map[string]string{
		"latlng": fmt.Sprintf("%.2f,%.2f", lat, lng),
	}
	return cfg.performGeocodeRequest(params)
}

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
