package main

import (
	"embed"
	"testing"
	"time"
)

//go:embed testdata/*.json
var testData embed.FS

func TestParseCurrentWeatherGMP(t *testing.T) {
	sampleJSON, _ := testData.Open("testdata/current_weather_gmp.json")
	defer sampleJSON.Close()
	timestamp, _ := time.Parse(time.RFC3339Nano, "2025-08-04T09:44:48.736691285Z")
	weather := CurrentWeather{
		SourceAPI:     "Google Weather API",
		Timestamp:     timestamp,
		Temperature:   18.2,
		Humidity:      74,
		WindSpeed:     6.,
		Precipitation: 0.1321,
		Condition:     "Cloudy",
		Error:         nil,
	}

	parsedWeather := ParseCurrentWeatherGMP(sampleJSON)

	if parsedWeather != weather {
		t.Errorf("Expected parsed weather to be %v, got %v", weather, parsedWeather)
	}
}

func TestParseCurrentWeatherOWM(t *testing.T) {
	sampleJSON, _ := testData.Open("testdata/current_weather_owm.json")
	defer sampleJSON.Close()
	timestamp := time.Unix(1754300711, 0)
	weather := CurrentWeather{
		SourceAPI:     "OpenWeatherMap API",
		Timestamp:     timestamp,
		Temperature:   17.,
		Humidity:      79,
		WindSpeed:     Round(2.57/3.6, 4),
		Precipitation: 0.32,
		Condition:     "Rain",
		Error:         nil,
	}

	parsedWeather := ParseCurrentWeatherOWM(sampleJSON)

	if parsedWeather != weather {
		t.Errorf("Expected parsed weather to be %v, got %v", weather, parsedWeather)
	}
}

func TestParseCurrentWeatherOMeteo(t *testing.T) {
	sampleJSON, _ := testData.Open("testdata/current_weather_ometeo.json")
	defer sampleJSON.Close()
	timestamp := time.Unix(1754300700, 0)
	weather := CurrentWeather{
		SourceAPI:     "Open-Meteo API",
		Timestamp:     timestamp,
		Temperature:   18.3,
		Humidity:      71,
		WindSpeed:     9.,
		Precipitation: 0.1,
		Condition:     "slight rain",
		Error:         nil,
	}

	parsedWeather := ParseCurrentWeatherOMeteo(sampleJSON)

	if parsedWeather != weather {
		t.Errorf("Expected parsed weather to be %v, got %v", weather, parsedWeather)
	}
}

func TestParseDailyForecastGMP(t *testing.T) {
	sampleJSON, _ := testData.Open("testdata/daily_forecast_gmp.json")
	defer sampleJSON.Close()
	forecast := make([]DailyForecast, 5)

	timestamp, _ := time.Parse(time.RFC3339, "2025-08-05T05:00:00Z")
	forecast[0] = DailyForecast{
		SourceAPI:           "Google Weather API",
		ForecastDate:        timestamp,
		MinTemp:             12.9,
		MaxTemp:             25.6,
		Precipitation:       1.5748,
		PrecipitationChance: 50,
	}

	parsedForecast, _ := ParseDailyForecastGMP(sampleJSON)

	if parsedForecast[0] != forecast[0] {
		t.Errorf("Expected parsed weather to be %v, got %v", forecast[0], parsedForecast[0])
	}
}
