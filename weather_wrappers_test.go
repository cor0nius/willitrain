package main

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestWrapForCurrentWeather(t *testing.T) {
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

	location := Location{Latitude: 40.7128, Longitude: -74.0060} // Example coordinates for New York City
	wrappedURLs := cfg.WrapForCurrentWeather(location)
	if wrappedURLs.gmpWrappedURL == "" || wrappedURLs.owmWrappedURL == "" || wrappedURLs.ometeoWrappedURL == "" {
		t.Error("Wrapped URLs should not be empty")
	}

	if wrappedURLs.gmpWrappedURL != "https://weather.googleapis.com/v1/currentConditions:lookup?key="+gmpKey+"&location.latitude=40.712800&location.longitude=-74.006000" {
		t.Errorf("Expected GMP wrapped URL to be %s, got %s", "https://weather.googleapis.com/v1/currentConditions:lookup?key="+gmpKey+"&location.latitude=40.712800&location.longitude=-74.006000", wrappedURLs.gmpWrappedURL)
	}

	if wrappedURLs.owmWrappedURL != "https://api.openweathermap.org/data/3.0/onecall?lat=40.712800&lon=-74.006000&exclude=minutely,hourly,daily,alerts&units=metric&appid="+owmKey {
		t.Errorf("Expected OWM wrapped URL to be %s, got %s", "https://api.openweathermap.org/data/3.0/onecall?lat=40.712800&lon=-74.006000&exclude=minutely,hourly,daily,alerts&units=metric&appid="+owmKey, wrappedURLs.owmWrappedURL)
	}

	if wrappedURLs.ometeoWrappedURL != "https://api.open-meteo.com/v1/forecast?latitude=40.712800&longitude=-74.006000&current=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,rain,showers,snowfall,weather_code&timezone=auto" {
		t.Errorf("Expected Ometeo wrapped URL to be %s, got %s", "https://api.open-meteo.com/v1/forecast?latitude=40.712800&longitude=-74.006000&current=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,rain,showers,snowfall,weather_code&timezone=auto", wrappedURLs.ometeoWrappedURL)
	}
}

func TestWrapForDailyForecast(t *testing.T) {
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

	location := Location{Latitude: 40.7128, Longitude: -74.0060} // Example coordinates for New York City
	wrappedURLs := cfg.WrapForDailyForecast(location)
	if wrappedURLs.gmpWrappedURL == "" || wrappedURLs.owmWrappedURL == "" || wrappedURLs.ometeoWrappedURL == "" {
		t.Error("Wrapped URLs should not be empty")
	}

	if wrappedURLs.gmpWrappedURL != "https://weather.googleapis.com/v1/forecast/days:lookup?key="+gmpKey+"&location.latitude=40.712800&location.longitude=-74.006000" {
		t.Errorf("Expected GMP wrapped URL to be %s, got %s", "https://weather.googleapis.com/v1/forecast/days:lookup?key="+gmpKey+"&location.latitude=40.712800&location.longitude=-74.006000", wrappedURLs.gmpWrappedURL)
	}

	if wrappedURLs.owmWrappedURL != "https://api.openweathermap.org/data/3.0/onecall?lat=40.712800&lon=-74.006000&exclude=current,minutely,hourly,alerts&units=metric&appid="+owmKey {
		t.Errorf("Expected OWM wrapped URL to be %s, got %s", "https://api.openweathermap.org/data/3.0/onecall?lat=40.712800&lon=-74.006000&exclude=current,minutely,hourly,alerts&units=metric&appid="+owmKey, wrappedURLs.owmWrappedURL)
	}

	if wrappedURLs.ometeoWrappedURL != "https://api.open-meteo.com/v1/forecast?latitude=40.712800&longitude=-74.006000&daily=temperature_2m_max,temperature_2m_mean,temperature_2m_min,precipitation_sum,precipitation_probability_max,rain_sum,showers_sum,snowfall_sum,wind_speed_10m_max,weather_code&timezone=auto" {
		t.Errorf("Expected Ometeo wrapped URL to be %s, got %s", "https://api.open-meteo.com/v1/forecast?latitude=40.712800&longitude=-74.006000&daily=temperature_2m_max,temperature_2m_mean,temperature_2m_min,precipitation_sum,precipitation_probability_max,rain_sum,showers_sum,snowfall_sum,wind_speed_10m_max,weather_code&timezone=auto", wrappedURLs.ometeoWrappedURL)
	}
}

func TestWrapForHourlyForecast(t *testing.T) {
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

	location := Location{Latitude: 40.7128, Longitude: -74.0060} // Example coordinates for New York City
	wrappedURLs := cfg.WrapForHourlyForecast(location)
	if wrappedURLs.gmpWrappedURL == "" || wrappedURLs.owmWrappedURL == "" || wrappedURLs.ometeoWrappedURL == "" {
		t.Error("Wrapped URLs should not be empty")
	}

	if wrappedURLs.gmpWrappedURL != "https://weather.googleapis.com/v1/forecast/hours:lookup?key="+gmpKey+"&location.latitude=40.712800&location.longitude=-74.006000" {
		t.Errorf("Expected GMP wrapped URL to be %s, got %s", "https://weather.googleapis.com/v1/forecast/hours:lookup?key="+gmpKey+"&location.latitude=40.712800&location.longitude=-74.006000", wrappedURLs.gmpWrappedURL)
	}

	if wrappedURLs.owmWrappedURL != "https://api.openweathermap.org/data/3.0/onecall?lat=40.712800&lon=-74.006000&exclude=current,minutely,daily,alerts&units=metric&appid="+owmKey {
		t.Errorf("Expected OWM wrapped URL to be %s, got %s", "https://api.openweathermap.org/data/3.0/onecall?lat=40.712800&lon=-74.006000&exclude=current,minutely,daily,alerts&units=metric&appid="+owmKey, wrappedURLs.owmWrappedURL)
	}

	if wrappedURLs.ometeoWrappedURL != "https://api.open-meteo.com/v1/forecast?latitude=40.712800&longitude=-74.006000&hourly=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,precipitation_probability,rain,showers,snowfall,weather_code&forecast_days=2&timezone=auto" {
		t.Errorf("Expected Ometeo wrapped URL to be %s, got %s", "https://api.open-meteo.com/v1/forecast?latitude=40.712800&longitude=-74.006000&hourly=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,precipitation_probability,rain,showers,snowfall,weather_code&forecast_days=2&timezone=auto", wrappedURLs.ometeoWrappedURL)
	}
}
