package main

import (
	"fmt"
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
	reqURL, err := cfg.WrapForGeocode(cityName)
	if err != nil {
		return Location{}, err
	}
	resp, err := http.Get(reqURL)
	if err != nil {
		return Location{}, err
	}
	defer resp.Body.Close()
	return Location{}, nil
}

func (cfg *apiConfig) WrapForReverseGeocode(lat, lng float64) (string, error) {
	baseURL := "https://maps.googleapis.com/maps/api/geocode/"
	latlng := fmt.Sprintf("latlng=%v,%v", lat, lng)
	wrappedURL := fmt.Sprintf("%sjson?%s%s", baseURL, latlng, cfg.gmpKey)
	return wrappedURL, nil
}
