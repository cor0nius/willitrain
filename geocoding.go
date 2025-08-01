package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (cfg *apiConfig) WrapForGeocode(cityName string) (string, error) {
	baseURL := "https://maps.googleapis.com/maps/api/geocode/"
	cityClean := strings.ReplaceAll(strings.ToLower(cityName), " ", "%20")
	city := fmt.Sprintf("address=%s", cityClean)
	wrappedURL := fmt.Sprintf("%sjson?%s%s", baseURL, city, cfg.gmpKey)
	return wrappedURL, nil
}

func (cfg *apiConfig) Geocode(cityName string) (Location, error) {
	reqestURL, err := cfg.WrapForGeocode(cityName)
	if err != nil {
		return Location{}, err
	}

	response, err := http.Get(reqestURL)
	if err != nil {
		return Location{}, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return Location{}, err
	}

	var responseJSON Response
	if err = json.Unmarshal(data, &responseJSON); err != nil {
		return Location{}, err
	}

	if len(responseJSON.Results) == 0 {
		return Location{}, err // no results found
	}

	result := responseJSON.Results[0]

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

	return location, nil
}

func (cfg *apiConfig) WrapForReverseGeocode(lat, lng float64) (string, error) {
	baseURL := "https://maps.googleapis.com/maps/api/geocode/"
	latlng := fmt.Sprintf("latlng=%v,%v", lat, lng)
	wrappedURL := fmt.Sprintf("%sjson?%s%s", baseURL, latlng, cfg.gmpKey)
	return wrappedURL, nil
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
