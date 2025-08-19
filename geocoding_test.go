package main

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestGeocode(t *testing.T) {
	server := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		data, err := testData.ReadFile("testdata/geocode_gmp.json")
		if err != nil {
			t.Fatalf("Failed to read test data: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
	defer server.Close()

	geocoder := NewGmpGeocodingService(
		"dummy-key",
		server.URL+"/",
		server.Client(),
	)

	location, err := geocoder.Geocode("Wroclaw")
	if err != nil {
		t.Fatalf("Geocode() returned an unexpected error: %v", err)
	}

	if location.CityName != "Wrocław" {
		t.Errorf("Expected city name 'Wrocław', got '%s'", location.CityName)
	}
	if location.CountryCode != "PL" {
		t.Errorf("Expected country code 'PL', got '%s'", location.CountryCode)
	}
	expectedLat := 51.1092948
	if math.Abs(location.Latitude-expectedLat) > 0.0001 {
		t.Errorf("Expected latitude %f, got %f", expectedLat, location.Latitude)
	}
	expectedLng := 17.0386019
	if math.Abs(location.Longitude-expectedLng) > 0.0001 {
		t.Errorf("Expected longitude %f, got %f", expectedLng, location.Longitude)
	}
}

func TestReverseGeocode(t *testing.T) {
	server := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		data, err := testData.ReadFile("testdata/reverse_geocode_gmp.json")
		if err != nil {
			t.Fatalf("Failed to read test data: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
	defer server.Close()

	geocoder := NewGmpGeocodingService(
		"dummy-key",
		server.URL+"/",
		server.Client(),
	)

	location, err := geocoder.ReverseGeocode(51.11, 17.04)
	if err != nil {
		t.Fatalf("ReverseGeocode() returned an unexpected error: %v", err)
	}

	if location.CityName != "Wrocław" {
		t.Errorf("Expected city name 'Wrocław', got '%s'", location.CityName)
	}
	if location.CountryCode != "PL" {
		t.Errorf("Expected country code 'PL', got '%s'", location.CountryCode)
	}
	expectedLat := 51.1100303
	if math.Abs(location.Latitude-expectedLat) > 0.0001 {
		t.Errorf("Expected latitude %f, got %f", expectedLat, location.Latitude)
	}
	expectedLng := 17.039911
	if math.Abs(location.Longitude-expectedLng) > 0.0001 {
		t.Errorf("Expected longitude %f, got %f", expectedLng, location.Longitude)
	}
}

func TestGeocode_APIError(t *testing.T) {
	server := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	geocoder := NewGmpGeocodingService(
		"dummy-key",
		server.URL+"/",
		server.Client(),
	)

	_, err := geocoder.Geocode("Wroclaw")
	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}
}

func TestGeocode_ZeroResults(t *testing.T) {
	server := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ZERO_RESULTS", "results": []}`))
	})
	defer server.Close()

	geocoder := NewGmpGeocodingService(
		"dummy-key",
		server.URL+"/",
		server.Client(),
	)

	_, err := geocoder.Geocode("nonexistentcity")
	if !errors.Is(err, ErrNoResultsFound) {
		t.Errorf("Expected ErrNoResultsFound, but got %v", err)
	}
}

func TestGeocode_MalformedJSON(t *testing.T) {
	server := setupMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "OK", "results": [invalid]`)) // Malformed JSON
	})
	defer server.Close()

	geocoder := NewGmpGeocodingService(
		"dummy-key",
		server.URL+"/",
		server.Client(),
	)

	_, err := geocoder.Geocode("anycity")
	if err == nil {
		t.Fatal("Expected an error for malformed JSON, but got nil")
	}

	var syntaxError *json.SyntaxError
	if !errors.As(err, &syntaxError) {
		t.Errorf("Expected a *json.SyntaxError, but got %T", err)
	}
}
