package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/*.json
var testData embed.FS

func TestParseCurrentWeatherGMP(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/current_weather_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()
		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp, err := time.Parse(time.RFC3339Nano, "2025-08-04T09:44:48.736691285Z")
		if err != nil {
			t.Fatalf("failed to parse timestamp: %v", err)
		}
		expectedWeather := CurrentWeather{
			SourceAPI:     "Google Weather API",
			Timestamp:     timestamp.In(loc),
			Temperature:   18.2,
			Humidity:      74,
			WindSpeed:     6.0,
			Precipitation: 0.1321,
			Condition:     "Cloudy",
		}
		expectedTimezone := "Europe/Warsaw"

		parsedWeather, tz, err := ParseCurrentWeatherGMP(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseCurrentWeatherGMP failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
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
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/current_weather_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedWeather, _, err := ParseCurrentWeatherGMP(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseCurrentWeatherGMP failed with error: %v", err)
		}
		if parsedWeather.Timestamp.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedWeather.Timestamp.Location().String())
		}
	})
}

func TestParseCurrentWeatherOWM(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/current_weather_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()
		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Unix(1754300711, 0).In(loc)
		expectedWeather := CurrentWeather{
			SourceAPI:     "OpenWeatherMap API",
			Timestamp:     timestamp,
			Temperature:   17.,
			Humidity:      79,
			WindSpeed:     Round(2.57*3.6, 4),
			Precipitation: 0.32,
			Condition:     "Rain",
		}
		expectedTimezone := "Europe/Warsaw"

		parsedWeather, tz, err := ParseCurrentWeatherOWM(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseCurrentWeatherOWM failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
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
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/current_weather_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedWeather, _, err := ParseCurrentWeatherOWM(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseCurrentWeatherOWM failed with error: %v", err)
		}
		if parsedWeather.Timestamp.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedWeather.Timestamp.Location().String())
		}
	})
}

func TestParseCurrentWeatherOMeteo(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/current_weather_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()
		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Unix(1754300700, 0).In(loc)
		expectedWeather := CurrentWeather{
			SourceAPI:     "Open-Meteo API",
			Timestamp:     timestamp,
			Temperature:   18.3,
			Humidity:      71,
			WindSpeed:     9.0,
			Precipitation: 0.1,
			Condition:     "slight rain",
		}
		expectedTimezone := "Europe/Warsaw"

		parsedWeather, tz, err := ParseCurrentWeatherOMeteo(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseCurrentWeatherOMeteo failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
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
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/current_weather_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedWeather, _, err := ParseCurrentWeatherOMeteo(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseCurrentWeatherOMeteo failed with error: %v", err)
		}
		if parsedWeather.Timestamp.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedWeather.Timestamp.Location().String())
		}
	})
}

func TestParseCurrentWeatherGMP_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedWeather, _, err := ParseCurrentWeatherGMP(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	expected := CurrentWeather{SourceAPI: "Google Weather API"}
	if parsedWeather != expected {
		t.Errorf("expected weather to be %v, but got %v", expected, parsedWeather)
	}
}

func TestParseCurrentWeatherOWM_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedWeather, _, err := ParseCurrentWeatherOWM(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	expected := CurrentWeather{SourceAPI: "OpenWeatherMap API"}
	if parsedWeather != expected {
		t.Errorf("expected weather to be %v, but got %v", expected, parsedWeather)
	}
}

func TestParseCurrentWeatherOMeteo_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedWeather, _, err := ParseCurrentWeatherOMeteo(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	expected := CurrentWeather{SourceAPI: "Open-Meteo API"}
	if parsedWeather != expected {
		t.Errorf("expected weather to be %v, but got %v", expected, parsedWeather)
	}
}

func TestParseDailyForecastGMP(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Date(2025, 8, 6, 0, 0, 0, 0, loc)

		expectedForecast := DailyForecast{
			SourceAPI:           "Google Weather API",
			ForecastDate:        timestamp,
			MinTemp:             12.9,
			MaxTemp:             25.6,
			Precipitation:       1.5748,
			PrecipitationChance: 50,
			WindSpeed:           16.0,
			Humidity:            68,
		}
		expectedTimezone := "Europe/Warsaw"

		parsedForecast, tz, err := ParseDailyForecastGMP(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastGMP failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
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
		if firstForecast.WindSpeed != expectedForecast.WindSpeed {
			t.Errorf("WindSpeed: got %f, want %f", firstForecast.WindSpeed, expectedForecast.WindSpeed)
		}
		if firstForecast.Humidity != expectedForecast.Humidity {
			t.Errorf("Humidity: got %d, want %d", firstForecast.Humidity, expectedForecast.Humidity)
		}
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedForecast, _, err := ParseDailyForecastGMP(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastGMP failed with error: %v", err)
		}
		if parsedForecast[0].ForecastDate.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedForecast[0].ForecastDate.Location().String())
		}
	})

	t.Run("Success - Truncates long input", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		var response ResponseDailyForecastGMP
		content, _ := io.ReadAll(sampleJSON)
		if err := json.Unmarshal(content, &response); err != nil {
			t.Fatalf("failed to unmarshal test data: %v", err)
		}

		response.ForecastDays = append(response.ForecastDays, response.ForecastDays...)

		modifiedContent, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal modified data: %v", err)
		}
		reader := bytes.NewReader(modifiedContent)

		parsedForecast, _, err := ParseDailyForecastGMP(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastGMP failed with error: %v", err)
		}

		if len(parsedForecast) != 5 {
			t.Errorf("expected forecast to be truncated to 5 days, but got %d", len(parsedForecast))
		}
	})
}

func TestParseDailyForecastGMP_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedForecast, _, err := ParseDailyForecastGMP(invalidJSON, slog.Default())
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
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Date(2025, 8, 5, 13, 0, 0, 0, loc)
		expectedForecast := HourlyForecast{
			SourceAPI:           "Google Weather API",
			ForecastDateTime:    timestamp,
			Temperature:         23.5,
			Humidity:            61,
			WindSpeed:           14.,
			Precipitation:       0.,
			PrecipitationChance: 5,
			Condition:           "Partly sunny",
		}
		expectedTimezone := "Europe/Warsaw"

		parsedForecast, tz, err := ParseHourlyForecastGMP(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastGMP failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
		}
		if len(parsedForecast) == 0 {
			t.Fatal("parsedForecast is empty, expected at least one forecast")
		}
		firstForecast := parsedForecast[0]

		if firstForecast.SourceAPI != expectedForecast.SourceAPI {
			t.Errorf("SourceAPI: got %q, want %q", firstForecast.SourceAPI, expectedForecast.SourceAPI)
		}
		if !firstForecast.ForecastDateTime.Equal(expectedForecast.ForecastDateTime) {
			t.Errorf("ForecastDateTime: got %v, want %v", firstForecast.ForecastDateTime, expectedForecast.ForecastDateTime)
		}
		if firstForecast.Temperature != expectedForecast.Temperature {
			t.Errorf("Temperature: got %f, want %f", firstForecast.Temperature, expectedForecast.Temperature)
		}
		if firstForecast.Humidity != expectedForecast.Humidity {
			t.Errorf("Humidity: got %d, want %d", firstForecast.Humidity, expectedForecast.Humidity)
		}
		if firstForecast.WindSpeed != expectedForecast.WindSpeed {
			t.Errorf("WindSpeed: got %f, want %f", firstForecast.WindSpeed, expectedForecast.WindSpeed)
		}
		if firstForecast.Precipitation != expectedForecast.Precipitation {
			t.Errorf("Precipitation: got %f, want %f", firstForecast.Precipitation, expectedForecast.Precipitation)
		}
		if firstForecast.PrecipitationChance != expectedForecast.PrecipitationChance {
			t.Errorf("PrecipitationChance: got %d, want %d", firstForecast.PrecipitationChance, expectedForecast.PrecipitationChance)
		}
		if firstForecast.Condition != expectedForecast.Condition {
			t.Errorf("Condition: got %q, want %q", firstForecast.Condition, expectedForecast.Condition)
		}
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedForecast, _, err := ParseHourlyForecastGMP(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastGMP failed with error: %v", err)
		}
		if parsedForecast[0].ForecastDateTime.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedForecast[0].ForecastDateTime.Location().String())
		}
	})

	t.Run("Success - Truncates long input", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_gmp.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		var response ResponseHourlyForecastGMP
		content, _ := io.ReadAll(sampleJSON)
		if err := json.Unmarshal(content, &response); err != nil {
			t.Fatalf("failed to unmarshal test data: %v", err)
		}

		response.ForecastHours = append(response.ForecastHours, response.ForecastHours...)

		modifiedContent, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal modified data: %v", err)
		}
		reader := bytes.NewReader(modifiedContent)

		parsedForecast, _, err := ParseHourlyForecastGMP(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastGMP failed with error: %v", err)
		}

		if len(parsedForecast) != 24 {
			t.Errorf("expected forecast to be truncated to 24 hours, but got %d", len(parsedForecast))
		}
	})
}

func TestParseDailyForecastOWM(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Date(2025, 8, 6, 0, 0, 0, 0, loc)
		expectedForecast := DailyForecast{
			SourceAPI:           "OpenWeatherMap API",
			ForecastDate:        timestamp,
			MinTemp:             13.6,
			MaxTemp:             26.63,
			Precipitation:       9.15,
			PrecipitationChance: 100,
			WindSpeed:           Round(7.27*3.6, 4),
			Humidity:            58,
		}
		expectedTimezone := "Europe/Warsaw"

		parsedForecast, tz, err := ParseDailyForecastOWM(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastOWM failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
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
		if firstForecast.WindSpeed != expectedForecast.WindSpeed {
			t.Errorf("WindSpeed: got %f, want %f", firstForecast.WindSpeed, expectedForecast.WindSpeed)
		}
		if firstForecast.Humidity != expectedForecast.Humidity {
			t.Errorf("Humidity: got %d, want %d", firstForecast.Humidity, expectedForecast.Humidity)
		}
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedForecast, _, err := ParseDailyForecastOWM(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastOWM failed with error: %v", err)
		}
		if parsedForecast[0].ForecastDate.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedForecast[0].ForecastDate.Location().String())
		}
	})

	t.Run("Success - Truncates long input", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		var response ResponseDailyForecastOWM
		content, _ := io.ReadAll(sampleJSON)
		if err := json.Unmarshal(content, &response); err != nil {
			t.Fatalf("failed to unmarshal test data: %v", err)
		}

		response.DailyForecast = append(response.DailyForecast, response.DailyForecast...)

		modifiedContent, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal modified data: %v", err)
		}
		reader := bytes.NewReader(modifiedContent)

		parsedForecast, _, err := ParseDailyForecastOWM(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastOWM failed with error: %v", err)
		}

		if len(parsedForecast) != 5 {
			t.Errorf("expected forecast to be truncated to 5 days, but got %d", len(parsedForecast))
		}
	})
}

func TestParseDailyForecastOWM_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedForecast, _, err := ParseDailyForecastOWM(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	if len(parsedForecast) != 1 {
		t.Fatalf("expected a slice with a single item, but got %d items", len(parsedForecast))
	}

	expected := DailyForecast{SourceAPI: "OpenWeatherMap API"}
	if parsedForecast[0] != expected {
		t.Errorf("expected forecast to be %v, but got %v", expected, parsedForecast[0])
	}
}

func TestParseHourlyForecastGMP_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedForecast, _, err := ParseHourlyForecastGMP(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	if len(parsedForecast) != 1 {
		t.Fatalf("expected a slice with a single item, but got %d items", len(parsedForecast))
	}

	expected := HourlyForecast{SourceAPI: "Google Weather API"}
	if parsedForecast[0] != expected {
		t.Errorf("expected forecast to be %v, but got %v", expected, parsedForecast[0])
	}
}

func TestParseHourlyForecastOWM_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedForecast, _, err := ParseHourlyForecastOWM(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	if len(parsedForecast) != 1 {
		t.Fatalf("expected a slice with a single item, but got %d items", len(parsedForecast))
	}

	expected := HourlyForecast{SourceAPI: "OpenWeatherMap API"}
	if parsedForecast[0] != expected {
		t.Errorf("expected forecast to be %v, but got %v", expected, parsedForecast[0])
	}
}

func TestParseHourlyForecastOMeteo_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedForecast, _, err := ParseHourlyForecastOMeteo(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	if len(parsedForecast) != 1 {
		t.Fatalf("expected a slice with a single item, but got %d items", len(parsedForecast))
	}

	expected := HourlyForecast{SourceAPI: "Open-Meteo API"}
	if parsedForecast[0] != expected {
		t.Errorf("expected forecast to be %v, but got %v", expected, parsedForecast[0])
	}
}

func TestParseHourlyForecastOWM(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Unix(1754391600, 0).In(loc)
		expectedForecast := HourlyForecast{
			SourceAPI:           "OpenWeatherMap API",
			ForecastDateTime:    timestamp,
			Temperature:         25.19,
			Humidity:            56,
			WindSpeed:           Round(4.06*3.6, 4),
			Precipitation:       0.,
			PrecipitationChance: 0,
			Condition:           "Clear",
		}
		expectedTimezone := "Europe/Warsaw"

		parsedForecast, tz, err := ParseHourlyForecastOWM(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastOWM failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
		}
		if len(parsedForecast) == 0 {
			t.Fatal("parsedForecast is empty, expected at least one forecast")
		}
		firstForecast := parsedForecast[0]

		if firstForecast.SourceAPI != expectedForecast.SourceAPI {
			t.Errorf("SourceAPI: got %q, want %q", firstForecast.SourceAPI, expectedForecast.SourceAPI)
		}
		if !firstForecast.ForecastDateTime.Equal(expectedForecast.ForecastDateTime) {
			t.Errorf("ForecastDateTime: got %v, want %v", firstForecast.ForecastDateTime, expectedForecast.ForecastDateTime)
		}
		if firstForecast.Temperature != expectedForecast.Temperature {
			t.Errorf("Temperature: got %f, want %f", firstForecast.Temperature, expectedForecast.Temperature)
		}
		if firstForecast.Humidity != expectedForecast.Humidity {
			t.Errorf("Humidity: got %d, want %d", firstForecast.Humidity, expectedForecast.Humidity)
		}
		if firstForecast.WindSpeed != expectedForecast.WindSpeed {
			t.Errorf("WindSpeed: got %f, want %f", firstForecast.WindSpeed, expectedForecast.WindSpeed)
		}
		if firstForecast.Precipitation != expectedForecast.Precipitation {
			t.Errorf("Precipitation: got %f, want %f", firstForecast.Precipitation, expectedForecast.Precipitation)
		}
		if firstForecast.PrecipitationChance != expectedForecast.PrecipitationChance {
			t.Errorf("PrecipitationChance: got %d, want %d", firstForecast.PrecipitationChance, expectedForecast.PrecipitationChance)
		}
		if firstForecast.Condition != expectedForecast.Condition {
			t.Errorf("Condition: got %q, want %q", firstForecast.Condition, expectedForecast.Condition)
		}
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedForecast, _, err := ParseHourlyForecastOWM(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastOWM failed with error: %v", err)
		}
		if parsedForecast[0].ForecastDateTime.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedForecast[0].ForecastDateTime.Location().String())
		}
	})

	t.Run("Success - Truncates long input", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_owm.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		var response ResponseHourlyForecastOWM
		content, _ := io.ReadAll(sampleJSON)
		if err := json.Unmarshal(content, &response); err != nil {
			t.Fatalf("failed to unmarshal test data: %v", err)
		}

		response.HourlyForecast = append(response.HourlyForecast, response.HourlyForecast...)

		modifiedContent, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal modified data: %v", err)
		}
		reader := bytes.NewReader(modifiedContent)

		parsedForecast, _, err := ParseHourlyForecastOWM(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastOWM failed with error: %v", err)
		}

		if len(parsedForecast) != 24 {
			t.Errorf("expected forecast to be truncated to 24 hours, but got %d", len(parsedForecast))
		}
	})
}

func TestParseHourlyForecastOMeteo(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Unix(2785344800, 0).In(loc)
		expectedForecast := HourlyForecast{
			SourceAPI:           "Open-Meteo API",
			ForecastDateTime:    timestamp,
			Temperature:         16.1,
			Humidity:            74,
			WindSpeed:           7.3,
			Precipitation:       0.,
			PrecipitationChance: 0,
			Condition:           "partly cloudy",
		}
		expectedTimezone := "Europe/Warsaw"

		parsedForecast, tz, err := ParseHourlyForecastOMeteo(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastOMeteo failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
		}
		if len(parsedForecast) == 0 {
			t.Fatal("parsedForecast is empty, expected at least one forecast")
		}
		firstForecast := parsedForecast[0]

		if firstForecast.SourceAPI != expectedForecast.SourceAPI {
			t.Errorf("SourceAPI: got %q, want %q", firstForecast.SourceAPI, expectedForecast.SourceAPI)
		}
		if !firstForecast.ForecastDateTime.Equal(expectedForecast.ForecastDateTime) {
			t.Errorf("ForecastDateTime: got %v, want %v", firstForecast.ForecastDateTime, expectedForecast.ForecastDateTime)
		}
		if firstForecast.Temperature != expectedForecast.Temperature {
			t.Errorf("Temperature: got %f, want %f", firstForecast.Temperature, expectedForecast.Temperature)
		}
		if firstForecast.Humidity != expectedForecast.Humidity {
			t.Errorf("Humidity: got %d, want %d", firstForecast.Humidity, expectedForecast.Humidity)
		}
		if firstForecast.WindSpeed != expectedForecast.WindSpeed {
			t.Errorf("WindSpeed: got %f, want %f", firstForecast.WindSpeed, expectedForecast.WindSpeed)
		}
		if firstForecast.Precipitation != expectedForecast.Precipitation {
			t.Errorf("Precipitation: got %f, want %f", firstForecast.Precipitation, expectedForecast.Precipitation)
		}
		if firstForecast.PrecipitationChance != expectedForecast.PrecipitationChance {
			t.Errorf("PrecipitationChance: got %d, want %d", firstForecast.PrecipitationChance, expectedForecast.PrecipitationChance)
		}
		if firstForecast.Condition != expectedForecast.Condition {
			t.Errorf("Condition: got %q, want %q", firstForecast.Condition, expectedForecast.Condition)
		}
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedForecast, _, err := ParseHourlyForecastOMeteo(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastOMeteo failed with error: %v", err)
		}
		if parsedForecast[0].ForecastDateTime.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedForecast[0].ForecastDateTime.Location().String())
		}
	})

	t.Run("Success - Truncates long input", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/hourly_forecast_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		var response ResponseHourlyForecastOMeteo
		content, _ := io.ReadAll(sampleJSON)
		if err := json.Unmarshal(content, &response); err != nil {
			t.Fatalf("failed to unmarshal test data: %v", err)
		}

		response.HourlyForecast.Time = append(response.HourlyForecast.Time, response.HourlyForecast.Time...)
		response.HourlyForecast.Temperature2m = append(response.HourlyForecast.Temperature2m, response.HourlyForecast.Temperature2m...)
		response.HourlyForecast.RelativeHumidity2m = append(response.HourlyForecast.RelativeHumidity2m, response.HourlyForecast.RelativeHumidity2m...)
		response.HourlyForecast.WindSpeed10m = append(response.HourlyForecast.WindSpeed10m, response.HourlyForecast.WindSpeed10m...)
		response.HourlyForecast.Precipitation = append(response.HourlyForecast.Precipitation, response.HourlyForecast.Precipitation...)
		response.HourlyForecast.PrecipitationProbability = append(response.HourlyForecast.PrecipitationProbability, response.HourlyForecast.PrecipitationProbability...)
		response.HourlyForecast.WeatherCode = append(response.HourlyForecast.WeatherCode, response.HourlyForecast.WeatherCode...)

		modifiedContent, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal modified data: %v", err)
		}
		reader := bytes.NewReader(modifiedContent)

		parsedForecast, _, err := ParseHourlyForecastOMeteo(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseHourlyForecastOMeteo failed with error: %v", err)
		}

		if len(parsedForecast) != 24 {
			t.Errorf("expected forecast to be truncated to 24 hours, but got %d", len(parsedForecast))
		}
	})
}

func TestParseDailyForecastOMeteo(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		loc, _ := time.LoadLocation("Europe/Warsaw")
		timestamp := time.Date(2025, 8, 7, 0, 0, 0, 0, loc)

		expectedForecast := DailyForecast{
			SourceAPI:           "Open-Meteo API",
			ForecastDate:        timestamp,
			MinTemp:             11.7,
			MaxTemp:             24.1,
			Precipitation:       0.0,
			PrecipitationChance: 0,
			WindSpeed:           10.0,
			Humidity:            83,
		}
		expectedTimezone := "Europe/Warsaw"

		parsedForecast, tz, err := ParseDailyForecastOMeteo(sampleJSON, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastOMeteo failed with error: %v", err)
		}

		if tz != expectedTimezone {
			t.Errorf("Timezone: got %q, want %q", tz, expectedTimezone)
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
		if firstForecast.WindSpeed != expectedForecast.WindSpeed {
			t.Errorf("WindSpeed: got %f, want %f", firstForecast.WindSpeed, expectedForecast.WindSpeed)
		}
		if firstForecast.Humidity != expectedForecast.Humidity {
			t.Errorf("Humidity: got %d, want %d", firstForecast.Humidity, expectedForecast.Humidity)
		}
	})

	t.Run("Failure - Invalid Timezone", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		content, _ := io.ReadAll(sampleJSON)
		modifiedContent := strings.Replace(string(content), "Europe/Warsaw", "Mars/Olympus_Mons", 1)
		reader := strings.NewReader(modifiedContent)

		parsedForecast, _, err := ParseDailyForecastOMeteo(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastOMeteo failed with error: %v", err)
		}
		if parsedForecast[0].ForecastDate.Location().String() != "UTC" {
			t.Errorf("expected location to be UTC, but got %s", parsedForecast[0].ForecastDate.Location().String())
		}
	})

	t.Run("Success - Truncates long input", func(t *testing.T) {
		sampleJSON, err := testData.Open("testdata/daily_forecast_ometeo.json")
		if err != nil {
			t.Fatalf("failed to open test data: %v", err)
		}
		defer sampleJSON.Close()

		var response ResponseDailyForecastOMeteo
		content, _ := io.ReadAll(sampleJSON)
		if err := json.Unmarshal(content, &response); err != nil {
			t.Fatalf("failed to unmarshal test data: %v", err)
		}

		response.DailyForecast.Time = append(response.DailyForecast.Time, response.DailyForecast.Time...)
		response.DailyForecast.Temperature2mMax = append(response.DailyForecast.Temperature2mMax, response.DailyForecast.Temperature2mMax...)
		response.DailyForecast.Temperature2mMin = append(response.DailyForecast.Temperature2mMin, response.DailyForecast.Temperature2mMin...)
		response.DailyForecast.PrecipitationSum = append(response.DailyForecast.PrecipitationSum, response.DailyForecast.PrecipitationSum...)
		response.DailyForecast.PrecipitationProbabilityMax = append(response.DailyForecast.PrecipitationProbabilityMax, response.DailyForecast.PrecipitationProbabilityMax...)
		response.DailyForecast.WeatherCode = append(response.DailyForecast.WeatherCode, response.DailyForecast.WeatherCode...)
		response.DailyForecast.WindSpeed10mMax = append(response.DailyForecast.WindSpeed10mMax, response.DailyForecast.WindSpeed10mMax...)
		response.DailyForecast.RelativeHumidity2mMax = append(response.DailyForecast.RelativeHumidity2mMax, response.DailyForecast.RelativeHumidity2mMax...)

		modifiedContent, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal modified data: %v", err)
		}
		reader := bytes.NewReader(modifiedContent)

		parsedForecast, _, err := ParseDailyForecastOMeteo(reader, slog.Default())
		if err != nil {
			t.Fatalf("ParseDailyForecastOMeteo failed with error: %v", err)
		}

		if len(parsedForecast) != 5 {
			t.Errorf("expected forecast to be truncated to 5 days, but got %d", len(parsedForecast))
		}
	})
}

func TestParseDailyForecastOMeteo_Error(t *testing.T) {
	invalidJSON := strings.NewReader(`{ "invalid": "json" }`)

	parsedForecast, _, err := ParseDailyForecastOMeteo(invalidJSON, slog.Default())
	if err == nil {
		t.Fatal("expected an error for invalid JSON, but got nil")
	}

	if len(parsedForecast) != 1 {
		t.Fatalf("expected a slice with a single item, but got %d items", len(parsedForecast))
	}

	expected := DailyForecast{SourceAPI: "Open-Meteo API"}
	if parsedForecast[0] != expected {
		t.Errorf("expected forecast to be %v, but got %v", expected, parsedForecast[0])
	}
}

func TestParseCurrentWeatherGMP_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseCurrentWeatherGMP(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseCurrentWeatherOWM_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseCurrentWeatherOWM(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseCurrentWeatherOMeteo_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseCurrentWeatherOMeteo(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseDailyForecastGMP_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseDailyForecastGMP(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseDailyForecastOWM_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseDailyForecastOWM(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseDailyForecastOMeteo_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseDailyForecastOMeteo(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseHourlyForecastGMP_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseHourlyForecastGMP(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseHourlyForecastOWM_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseHourlyForecastOWM(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestParseHourlyForecastOMeteo_DecoderError(t *testing.T) {
	malformedJSON := strings.NewReader(`{,}`)
	_, _, err := ParseHourlyForecastOMeteo(malformedJSON, slog.Default())
	if err == nil {
		t.Fatal("expected a decoder error, but got nil")
	}
}

func TestInterpretWeatherCode(t *testing.T) {
	testCases := []struct {
		code     int
		expected string
	}{
		{0, "clear sky"},
		{1, "mainly clear"},
		{2, "partly cloudy"},
		{3, "overcast"},
		{45, "fog"},
		{48, "depositing rime fog"},
		{51, "light drizzle"},
		{53, "moderate drizzle"},
		{55, "dense drizzle"},
		{56, "light freezing drizzle"},
		{57, "dense freezing drizzle"},
		{61, "slight rain"},
		{63, "moderate rain"},
		{65, "heavy rain"},
		{66, "light freezing rain"},
		{67, "heavy freezing rain"},
		{71, "slight snowfall"},
		{73, "moderate snowfall"},
		{75, "heavy snowfall"},
		{77, "snow grains"},
		{80, "slight showers"},
		{81, "moderate showers"},
		{82, "violent showers"},
		{85, "slight snow showers"},
		{86, "heavy snow showers"},
		{95, "thunderstorm"},
		{96, "thunderstorm with slight hail"},
		{99, "thunderstorm with heavy hail"},
		{100, "unknown code"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := interpretWeatherCode(tc.code)
			if result != tc.expected {
				t.Errorf("for code %d, expected %q but got %q", tc.code, tc.expected, result)
			}
		})
	}
}
