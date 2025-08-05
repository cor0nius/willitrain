package main

import (
	"embed"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/*.json
var testData embed.FS

func TestParseCurrentWeatherGMP(t *testing.T) {
	sampleJSON, err := testData.Open("testdata/current_weather_gmp.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer sampleJSON.Close()
	timestamp, err := time.Parse(time.RFC3339Nano, "2025-08-04T09:44:48.736691285Z")
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}
	expectedWeather := CurrentWeather{
		SourceAPI:     "Google Weather API",
		Timestamp:     timestamp,
		Temperature:   18.2,
		Humidity:      74,
		WindSpeed:     6.0,
		Precipitation: 0.1321,
		Condition:     "Cloudy",
	}

	parsedWeather, err := ParseCurrentWeatherGMP(sampleJSON)
	if err != nil {
		t.Fatalf("ParseCurrentWeatherGMP failed with error: %v", err)
	}

	if parsedWeather.SourceAPI != expectedWeather.SourceAPI {
		t.Errorf("SourceAPI: got %q, want %q", parsedWeather.SourceAPI, expectedWeather.SourceAPI)
	}
	if !parsedWeather.Timestamp.Equal(expectedWeather.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", parsedWeather.Timestamp, expectedWeather.Timestamp)
	}
	if parsedWeather.Temperature != expectedWeather.Temperature {
		t.Errorf("Temperature: got %f, want %f", parsedWeather.Temperature, expectedWeather.Temperature)
	}
	if parsedWeather.Humidity != expectedWeather.Humidity {
		t.Errorf("Humidity: got %d, want %d", parsedWeather.Humidity, expectedWeather.Humidity)
	}
	if parsedWeather.WindSpeed != expectedWeather.WindSpeed {
		t.Errorf("WindSpeed: got %f, want %f", parsedWeather.WindSpeed, expectedWeather.WindSpeed)
	}
	if parsedWeather.Precipitation != expectedWeather.Precipitation {
		t.Errorf("Precipitation: got %f, want %f", parsedWeather.Precipitation, expectedWeather.Precipitation)
	}
	if parsedWeather.Condition != expectedWeather.Condition {
		t.Errorf("Condition: got %q, want %q", parsedWeather.Condition, expectedWeather.Condition)
	}
}

func TestParseCurrentWeatherOWM(t *testing.T) {
	sampleJSON, err := testData.Open("testdata/current_weather_owm.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer sampleJSON.Close()
	timestamp := time.Unix(1754300711, 0)
	expectedWeather := CurrentWeather{
		SourceAPI:     "OpenWeatherMap API",
		Timestamp:     timestamp,
		Temperature:   17.,
		Humidity:      79,
		WindSpeed:     Round(2.57*3.6, 4),
		Precipitation: 0.32,
		Condition:     "Rain",
	}

	parsedWeather, err := ParseCurrentWeatherOWM(sampleJSON)
	if err != nil {
		t.Fatalf("ParseCurrentWeatherOWM failed with error: %v", err)
	}

	if parsedWeather.SourceAPI != expectedWeather.SourceAPI {
		t.Errorf("SourceAPI: got %q, want %q", parsedWeather.SourceAPI, expectedWeather.SourceAPI)
	}
	if !parsedWeather.Timestamp.Equal(expectedWeather.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", parsedWeather.Timestamp, expectedWeather.Timestamp)
	}
	if parsedWeather.Temperature != expectedWeather.Temperature {
		t.Errorf("Temperature: got %f, want %f", parsedWeather.Temperature, expectedWeather.Temperature)
	}
	if parsedWeather.Humidity != expectedWeather.Humidity {
		t.Errorf("Humidity: got %d, want %d", parsedWeather.Humidity, expectedWeather.Humidity)
	}
	if parsedWeather.WindSpeed != expectedWeather.WindSpeed {
		t.Errorf("WindSpeed: got %f, want %f", parsedWeather.WindSpeed, expectedWeather.WindSpeed)
	}
	if parsedWeather.Precipitation != expectedWeather.Precipitation {
		t.Errorf("Precipitation: got %f, want %f", parsedWeather.Precipitation, expectedWeather.Precipitation)
	}
	if parsedWeather.Condition != expectedWeather.Condition {
		t.Errorf("Condition: got %q, want %q", parsedWeather.Condition, expectedWeather.Condition)
	}
}

func TestParseCurrentWeatherOMeteo(t *testing.T) {
	sampleJSON, err := testData.Open("testdata/current_weather_ometeo.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer sampleJSON.Close()
	timestamp := time.Unix(1754300700, 0)
	expectedWeather := CurrentWeather{
		SourceAPI:     "Open-Meteo API",
		Timestamp:     timestamp,
		Temperature:   18.3,
		Humidity:      71,
		WindSpeed:     9.0,
		Precipitation: 0.1,
		Condition:     "slight rain",
	}

	parsedWeather, err := ParseCurrentWeatherOMeteo(sampleJSON)
	if err != nil {
		t.Fatalf("ParseCurrentWeatherOMeteo failed with error: %v", err)
	}

	if parsedWeather.SourceAPI != expectedWeather.SourceAPI {
		t.Errorf("SourceAPI: got %q, want %q", parsedWeather.SourceAPI, expectedWeather.SourceAPI)
	}
	if !parsedWeather.Timestamp.Equal(expectedWeather.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", parsedWeather.Timestamp, expectedWeather.Timestamp)
	}
	if parsedWeather.Temperature != expectedWeather.Temperature {
		t.Errorf("Temperature: got %f, want %f", parsedWeather.Temperature, expectedWeather.Temperature)
	}
	if parsedWeather.Humidity != expectedWeather.Humidity {
		t.Errorf("Humidity: got %d, want %d", parsedWeather.Humidity, expectedWeather.Humidity)
	}
	if parsedWeather.WindSpeed != expectedWeather.WindSpeed {
		t.Errorf("WindSpeed: got %f, want %f", parsedWeather.WindSpeed, expectedWeather.WindSpeed)
	}
	if parsedWeather.Precipitation != expectedWeather.Precipitation {
		t.Errorf("Precipitation: got %f, want %f", parsedWeather.Precipitation, expectedWeather.Precipitation)
	}
	if parsedWeather.Condition != expectedWeather.Condition {
		t.Errorf("Condition: got %q, want %q", parsedWeather.Condition, expectedWeather.Condition)
	}
}

func TestParseDailyForecastGMP(t *testing.T) {
	sampleJSON, err := testData.Open("testdata/daily_forecast_gmp.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer sampleJSON.Close()

	timestamp, err := time.Parse(time.RFC3339, "2025-08-05T05:00:00Z")
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}
	expectedForecast := DailyForecast{
		SourceAPI:           "Google Weather API",
		ForecastDate:        timestamp,
		MinTemp:             12.9,
		MaxTemp:             25.6,
		Precipitation:       1.5748,
		PrecipitationChance: 50,
	}

	parsedForecast, err := ParseDailyForecastGMP(sampleJSON)
	if err != nil {
		t.Fatalf("ParseDailyForecastGMP failed with error: %v", err)
	}

	if len(parsedForecast) == 0 {
		t.Fatal("parsedForecast is empty, expected at least one forecast")
	}
	firstForecast := parsedForecast[0]

	if firstForecast.SourceAPI != expectedForecast.SourceAPI {
		t.Errorf("SourceAPI: got %q, want %q", firstForecast.SourceAPI, expectedForecast.SourceAPI)
	}
	if !firstForecast.ForecastDate.Equal(expectedForecast.ForecastDate) {
		t.Errorf("ForecastDate: got %v, want %v", firstForecast.ForecastDate, expectedForecast.ForecastDate)
	}
	if firstForecast.MinTemp != expectedForecast.MinTemp {
		t.Errorf("MinTemp: got %f, want %f", firstForecast.MinTemp, expectedForecast.MinTemp)
	}
	if firstForecast.MaxTemp != expectedForecast.MaxTemp {
		t.Errorf("MaxTemp: got %f, want %f", firstForecast.MaxTemp, expectedForecast.MaxTemp)
	}
	if firstForecast.Precipitation != expectedForecast.Precipitation {
		t.Errorf("Precipitation: got %f, want %f", firstForecast.Precipitation, expectedForecast.Precipitation)
	}
	if firstForecast.PrecipitationChance != expectedForecast.PrecipitationChance {
		t.Errorf("PrecipitationChance: got %d, want %d", firstForecast.PrecipitationChance, expectedForecast.PrecipitationChance)
	}
}

func TestParseDailyForecastGMP_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedForecast, err := ParseDailyForecastGMP(invalidJSON)
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	if len(parsedForecast) != 1 {
		t.Fatalf("expected a slice with a single item, but got %d items", len(parsedForecast))
	}

	expected := DailyForecast{SourceAPI: "Google Weather API"}
	if parsedForecast[0] != expected {
		t.Errorf("expected forecast to be %v, but got %v", expected, parsedForecast[0])
	}
}

func TestParseHourlyForecastGMP(t *testing.T) {
	sampleJSON, err := testData.Open("testdata/hourly_forecast_gmp.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer sampleJSON.Close()

	timestamp, _ := time.Parse(time.RFC3339, "2025-08-05T11:00:00Z")
	expectedForecast := HourlyForecast{
		SourceAPI:        "Google Weather API",
		ForecastDateTime: timestamp,
		Temperature:      23.5,
		Humidity:         61,
		WindSpeed:        14.,
		Precipitation:    0.,
	}

	parsedForecast, err := ParseHourlyForecastGMP(sampleJSON)
	if err != nil {
		t.Fatalf("ParseHourlyForecastGMP failed with error: %v", err)
	}
	if len(parsedForecast) < 1 {
		t.Fatal("parsedForecast is empty")
	}
	if parsedForecast[0] != expectedForecast {
		t.Errorf("Expected parsed weather to be %v, got %v", expectedForecast, parsedForecast[0])
	}
}

func TestParseDailyForecastOWM(t *testing.T) {
	sampleJSON, _ := testData.Open("testdata/daily_forecast_owm.json")
	defer sampleJSON.Close()
	forecast := make([]DailyForecast, 5)

	timestamp := time.Unix(1754388000, 0)
	forecast[0] = DailyForecast{
		SourceAPI:     "OpenWeatherMap API",
		ForecastDate:  timestamp,
		MinTemp:       13.6,
		MaxTemp:       26.63,
		Precipitation: 9.15,
	}

	parsedForecast, _ := ParseDailyForecastOWM(sampleJSON)

	if parsedForecast[0] != forecast[0] {
		t.Errorf("Expected parsed weather to be %v, got %v", forecast[0], parsedForecast[0])
	}
}

func TestParseHourlyForecastOWM(t *testing.T) {
	sampleJSON, err := testData.Open("testdata/hourly_forecast_owm.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer sampleJSON.Close()

	timestamp := time.Unix(1754391600, 0)
	expectedForecast := HourlyForecast{
		SourceAPI:        "OpenWeatherMap API",
		ForecastDateTime: timestamp,
		Temperature:      25.19,
		Humidity:         56,
		WindSpeed:        Round(4.06*3.6, 4),
		Precipitation:    0.,
	}

	parsedForecast, err := ParseHourlyForecastOWM(sampleJSON)
	if err != nil {
		t.Fatalf("ParseHourlyForecastOWM failed with error: %v", err)
	}
	if len(parsedForecast) < 1 {
		t.Fatal("parsedForecast is empty")
	}
	if parsedForecast[0] != expectedForecast {
		t.Errorf("Expected parsed weather to be %v, got %v", expectedForecast, parsedForecast[0])
	}
}

func TestParseHourlyForecastOMeteo(t *testing.T) {
	sampleJSON, err := testData.Open("testdata/hourly_forecast_ometeo.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer sampleJSON.Close()

	timestamp := time.Unix(1754344800, 0)
	expectedForecast := HourlyForecast{
		SourceAPI:        "Open-Meteo API",
		ForecastDateTime: timestamp,
		Temperature:      16.1,
		Humidity:         74,
		WindSpeed:        7.3,
		Precipitation:    0.,
	}

	parsedForecast, err := ParseHourlyForecastOMeteo(sampleJSON)
	if err != nil {
		t.Fatalf("ParseHourlyForecastOMeteo failed with error: %v", err)
	}
	if len(parsedForecast) < 1 {
		t.Fatal("parsedForecast is empty, expected at least one forecast")
	}
	if parsedForecast[0] != expectedForecast {
		t.Errorf("Expected parsed weather to be %v, got %v", expectedForecast, parsedForecast[0])
	}
}

func TestParseDailyForecastOMeteo(t *testing.T) {
	sampleJSON, _ := testData.Open("testdata/daily_forecast_ometeo.json")
	defer sampleJSON.Close()
	forecast := make([]DailyForecast, 5)

	timestamp := time.Unix(1754344800, 0)
	forecast[0] = DailyForecast{
		SourceAPI:           "Open-Meteo API",
		ForecastDate:        timestamp,
		MinTemp:             13.1,
		MaxTemp:             25.4,
		Precipitation:       13.3,
		PrecipitationChance: 100,
	}

	parsedForecast, _ := ParseDailyForecastOMeteo(sampleJSON)

	if parsedForecast[0] != forecast[0] {
		t.Errorf("Expected parsed weather to be %v, got %v", forecast[0], parsedForecast[0])
	}
}
