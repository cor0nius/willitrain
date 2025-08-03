package main

import (
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestWrapForGeocode(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Error loading .env file")
	}

	gmpGeocodeURL := os.Getenv("GMP_GEOCODE_URL")
	if gmpGeocodeURL == "" {
		t.Fatal("GMP_GEOCODE_URL must be set")
	}

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		t.Fatal("Missing API Key for Google Maps Platform")
	}

	cfg := apiConfig{gmpGeocodeURL: gmpGeocodeURL, gmpKey: gmpKey}

	cityName := "New York"
	expectedURL := "https://maps.googleapis.com/maps/api/geocode/json?address=new%20york&key=" + gmpKey

	wrappedURL := cfg.WrapForGeocode(cityName)
	if err != nil {
		t.Fatalf("WrapForGeocode failed: %v", err)
	}

	if wrappedURL != expectedURL {
		t.Errorf("Expected %s, got %s", expectedURL, wrappedURL)
	}

	if !strings.Contains(wrappedURL, "address=new%20york") {
		t.Error("Wrapped URL does not contain the expected address parameter")
	}

	if !strings.Contains(wrappedURL, gmpKey) {
		t.Error("Wrapped URL does not contain the API key")
	}

	if !strings.HasPrefix(wrappedURL, "https://maps.googleapis.com/maps/api/geocode/") {
		t.Error("Wrapped URL does not start with the expected base URL")
	}

	if strings.Contains(wrappedURL, " ") {
		t.Error("Wrapped URL contains spaces, which should be replaced with '%20'")
	}

	if strings.Contains(wrappedURL, "New York") {
		t.Error("Wrapped URL should not contain the original city name with spaces, it should be URL-encoded")
	}

	if strings.Contains(wrappedURL, "new york") {
		t.Error("Wrapped URL should not contain the original city name in lowercase with spaces, it should be URL-encoded")
	}
}

func TestGeocode(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Error loading .env file")
	}

	gmpGeocodeURL := os.Getenv("GMP_GEOCODE_URL")
	if gmpGeocodeURL == "" {
		t.Fatal("GMP_GEOCODE_URL must be set")
	}

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		t.Fatal("Missing API Key for Google Maps Platform")
	}

	cfg := apiConfig{gmpGeocodeURL: gmpGeocodeURL, gmpKey: gmpKey}

	cityName := "New York"

	location, err := cfg.Geocode(cityName)
	if err != nil {
		t.Fatalf("Geocode failed: %v", err)
	}

	if location.CityName == "" || location.CountryCode == "" {
		t.Error("Geocode did not return valid location data")
	}

	if location.Latitude == 0 || location.Longitude == 0 {
		t.Error("Geocode did not return valid latitude or longitude")
	}

	if location.CityName != "New York" {
		t.Errorf("Expected city name 'New York', got '%s'", location.CityName)
	}

	if location.CountryCode == "" {
		t.Error("Expected a valid country code, got empty string")
	}
}

func TestWrapForReverseGeocode(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Error loading .env file")
	}

	gmpGeocodeURL := os.Getenv("GMP_GEOCODE_URL")
	if gmpGeocodeURL == "" {
		t.Fatal("GMP_GEOCODE_URL must be set")
	}

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		t.Fatal("Missing API Key for Google Maps Platform")
	}

	cfg := apiConfig{gmpGeocodeURL: gmpGeocodeURL, gmpKey: gmpKey}

	lat, lng := 40.7128, -74.0061 // Coordinates for New York City
	expectedURL := "https://maps.googleapis.com/maps/api/geocode/json?latlng=40.7128,-74.0061&key=" + gmpKey

	wrappedURL := cfg.WrapForReverseGeocode(lat, lng)
	if err != nil {
		t.Fatalf("WrapForReverseGeocode failed: %v", err)
	}

	if wrappedURL != expectedURL {
		t.Errorf("Expected %s, got %s", expectedURL, wrappedURL)
	}

	if !strings.Contains(wrappedURL, "latlng=40.7128,-74.0061") {
		t.Error("Wrapped URL does not contain the expected latlng parameter")
	}

	if !strings.Contains(wrappedURL, gmpKey) {
		t.Error("Wrapped URL does not contain the API key")
	}

	if !strings.HasPrefix(wrappedURL, "https://maps.googleapis.com/maps/api/geocode/") {
		t.Error("Wrapped URL does not start with the expected base URL")
	}
}

func TestReverseGeocode(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Error loading .env file")
	}

	gmpGeocodeURL := os.Getenv("GMP_GEOCODE_URL")
	if gmpGeocodeURL == "" {
		t.Fatal("GMP_GEOCODE_URL must be set")
	}

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		t.Fatal("Missing API Key for Google Maps Platform")
	}

	cfg := apiConfig{gmpGeocodeURL: gmpGeocodeURL, gmpKey: gmpKey}
	lat, lng := 40.7128, -74.0061 // Coordinates for New York City

	location, err := cfg.ReverseGeocode(lat, lng)
	if err != nil {
		t.Fatalf("ReverseGeocode failed: %v", err)
	}

	if location.CityName == "" || location.CountryCode == "" {
		t.Error("ReverseGeocode did not return valid location data")
	}

	if location.CityName != "New York" {
		t.Errorf("Expected city name 'New York', got '%s'", location.CityName)
	}

	if location.CountryCode == "" {
		t.Error("Expected a valid country code, got empty string")
	}

	if location.Latitude == 0 || location.Longitude == 0 {
		t.Error("Expected valid latitude and longitude, got zero values")
	}
}
