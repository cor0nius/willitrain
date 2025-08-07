package main

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestURLWrappers(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Error loading .env file")
	}

	gmpWeatherURL := os.Getenv("GMP_WEATHER_URL")
	if gmpWeatherURL == "" {
		t.Fatal("GMP_WEATHER_URL must be set")
	}

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		t.Fatal("Missing API Key for Google Maps Platform")
	}

	owmWeatherURL := os.Getenv("OWM_WEATHER_URL")
	if owmWeatherURL == "" {
		t.Fatal("OWM_WEATHER_URL must be set")
	}

	owmKey := os.Getenv("OWM_KEY")
	if owmKey == "" {
		t.Fatal("Missing API Key for OpenWeatherMap")
	}

	ometeoWeatherURL := os.Getenv("OMETEO_WEATHER_URL")
	if ometeoWeatherURL == "" {
		t.Fatal("OMETEO_WEATHER_URL must be set")
	}

	cfg := apiConfig{
		gmpWeatherURL:    gmpWeatherURL,
		gmpKey:           gmpKey,
		owmWeatherURL:    owmWeatherURL,
		owmKey:           owmKey,
		ometeoWeatherURL: ometeoWeatherURL,
	}

	location := Location{Latitude: 51.1093, Longitude: 17.0386} // Example coordinates for Wrocław

	testCases := []struct {
		name         string
		wrapperFunc  func(Location) map[string]string
		expectedURLs map[string]string
	}{
		{
			name:        "CurrentWeather",
			wrapperFunc: cfg.WrapForCurrentWeather,
			expectedURLs: map[string]string{
				"gmpWrappedURL":    "https://weather.googleapis.com/v1/currentConditions:lookup?key=" + gmpKey + "&location.latitude=51.11&location.longitude=17.04",
				"owmWrappedURL":    "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=minutely,hourly,daily,alerts&units=metric&appid=" + owmKey,
				"ometeoWrappedURL": "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&current=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,weather_code&timezone=auto&timeformat=unixtime",
			},
		},
		{
			name:        "DailyForecast",
			wrapperFunc: cfg.WrapForDailyForecast,
			expectedURLs: map[string]string{
				"gmpWrappedURL":    "https://weather.googleapis.com/v1/forecast/days:lookup?key=" + gmpKey + "&location.latitude=51.11&location.longitude=17.04",
				"owmWrappedURL":    "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=current,minutely,hourly,alerts&units=metric&appid=" + owmKey,
				"ometeoWrappedURL": "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&daily=temperature_2m_max,temperature_2m_min,precipitation_sum,precipitation_probability_max,wind_speed_10m_max,weather_code,relative_humidity_2m_max&timezone=auto&timeformat=unixtime",
			},
		},
		{
			name:        "HourlyForecast",
			wrapperFunc: cfg.WrapForHourlyForecast,
			expectedURLs: map[string]string{
				"gmpWrappedURL":    "https://weather.googleapis.com/v1/forecast/hours:lookup?key=" + gmpKey + "&location.latitude=51.11&location.longitude=17.04",
				"owmWrappedURL":    "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=current,minutely,daily,alerts&units=metric&appid=" + owmKey,
				"ometeoWrappedURL": "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&hourly=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,precipitation_probability,weather_code&forecast_days=2&timezone=auto&timeformat=unixtime",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrappedURLs := tc.wrapperFunc(location)

			if len(wrappedURLs) != len(tc.expectedURLs) {
				t.Errorf("Expected %d URLs, but got %d", len(tc.expectedURLs), len(wrappedURLs))
			}

			for key, url := range tc.expectedURLs {
				if wrappedURLs[key] == "" {
					t.Errorf("%s should not be empty", key)
				}
				if wrappedURLs[key] != url {
					t.Errorf("Expected %s for %s, but got %s", url, key, wrappedURLs[key])
				}
			}
		})
	}
}
