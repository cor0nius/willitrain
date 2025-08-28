package main

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"testing"
)

func TestGmpGeocodingService(t *testing.T) {
	testCases := []struct {
		name             string
		isReverse        bool
		handler          http.HandlerFunc
		expectedLocation Location
		expectErr        bool
		expectedErrType  error
	}{
		{
			name:      "Successful Geocode",
			isReverse: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				data, err := testData.ReadFile("testdata/geocode_gmp.json")
				if err != nil {
					t.Fatalf("Failed to read test data: %v", err)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			},
			expectedLocation: Location{
				CityName:    "Wrocław",
				CountryCode: "PL",
				Latitude:    51.1092948,
				Longitude:   17.0386019,
			},
			expectErr: false,
		},
		{
			name:      "Successful Reverse Geocode",
			isReverse: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				data, err := testData.ReadFile("testdata/reverse_geocode_gmp.json")
				if err != nil {
					t.Fatalf("Failed to read test data: %v", err)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			},
			expectedLocation: Location{
				CityName:    "Wrocław",
				CountryCode: "PL",
				Latitude:    51.1100303,
				Longitude:   17.039911,
			},
			expectErr: false,
		},
		{
			name:      "API Error",
			isReverse: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectErr: true,
		},
		{
			name:      "Zero Results",
			isReverse: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status": "ZERO_RESULTS", "results": []}`))
			},
			expectErr:       true,
			expectedErrType: ErrNoResultsFound,
		},
		{
			name:      "Malformed JSON response",
			isReverse: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status": "OK", "results": [invalid]`))
			},
			expectErr:       true,
			expectedErrType: &json.SyntaxError{},
		},
		{
			name:      "Malformed base URL",
			isReverse: false,
			handler:   func(w http.ResponseWriter, r *http.Request) {}, // Will not be called
			expectErr: true,
		},
		{
			name:      "API Request Failure",
			isReverse: false,
			handler:   func(w http.ResponseWriter, r *http.Request) {}, // Will not be called
			expectErr: true,
		},
		{
			name:      "Unexpected API Status",
			isReverse: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status": "REQUEST_DENIED"}`))
			},
			expectErr: true,
		},
		{
			name:      "OK Status with Empty Results",
			isReverse: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status": "OK", "results": []}`))
			},
			expectErr:       true,
			expectedErrType: ErrNoResultsFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := setupMockServer(tc.handler)
			defer server.Close()

			client := server.Client()
			serviceURL := server.URL + "/"

			if tc.name == "API Request Failure" {
				client = &http.Client{
					Transport: &errorTransport{err: errors.New("network error")},
				}
			}

			if tc.name == "Malformed base URL" {
				serviceURL = "http://\x7f"
			}

			geocoder := NewGmpGeocodingService(
				"dummy-key",
				serviceURL,
				client,
			)

			var location Location
			var err error

			if tc.isReverse {
				location, err = geocoder.ReverseGeocode(51.11, 17.04)
			} else {
				location, err = geocoder.Geocode("some-city")
			}

			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected an error, but got nil")
				}
				if tc.expectedErrType != nil {
					if errors.Is(err, tc.expectedErrType) {
						return // Correct error type
					}
					// Check for syntax error type specifically
					var syntaxError *json.SyntaxError
					if errors.As(err, &syntaxError) {
						_, ok := tc.expectedErrType.(*json.SyntaxError)
						if ok {
							return // Correctly identified as json.SyntaxError
						}
					}
					t.Errorf("Expected error of type %T, but got %T", tc.expectedErrType, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Returned an unexpected error: %v", err)
				}
				if location.CityName != tc.expectedLocation.CityName {
					t.Errorf("Expected city name '%s', got '%s'", tc.expectedLocation.CityName, location.CityName)
				}
				if location.CountryCode != tc.expectedLocation.CountryCode {
					t.Errorf("Expected country code '%s', got '%s'", tc.expectedLocation.CountryCode, location.CountryCode)
				}
				if math.Abs(location.Latitude-tc.expectedLocation.Latitude) > 0.0001 {
					t.Errorf("Expected latitude %f, got %f", tc.expectedLocation.Latitude, location.Latitude)
				}
				if math.Abs(location.Longitude-tc.expectedLocation.Longitude) > 0.0001 {
					t.Errorf("Expected longitude %f, got %f", tc.expectedLocation.Longitude, location.Longitude)
				}
			}
		})
	}
}