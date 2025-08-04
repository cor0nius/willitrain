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

	location := Location{Latitude: 51.1093, Longitude: 17.0386} // Example coordinates for Wrocław
	wrappedURLs := cfg.WrapForCurrentWeather(location)

	if wrappedURLs["gmpWrappedURL"] == "" || wrappedURLs["owmWrappedURL"] == "" || wrappedURLs["ometeoWrappedURL"] == "" {
		t.Error("Wrapped URLs should not be empty")
	}

	if wrappedURLs["gmpWrappedURL"] != "https://weather.googleapis.com/v1/currentConditions:lookup?key="+gmpKey+"&location.latitude=51.11&location.longitude=17.04" {
		t.Errorf("Expected GMP wrapped URL to be %s, got %s", "https://weather.googleapis.com/v1/currentConditions:lookup?key="+gmpKey+"&location.latitude=51.11&location.longitude=17.04", wrappedURLs["gmpWrappedURL"])
	}

	if wrappedURLs["owmWrappedURL"] != "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=minutely,hourly,daily,alerts&units=metric&appid="+owmKey {
		t.Errorf("Expected OWM wrapped URL to be %s, got %s", "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=minutely,hourly,daily,alerts&units=metric&appid="+owmKey, wrappedURLs["owmWrappedURL"])
	}

	if wrappedURLs["ometeoWrappedURL"] != "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&current=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,weather_code&timezone=auto&timeformat=unixtime" {
		t.Errorf("Expected Ometeo wrapped URL to be %s, got %s", "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&current=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,weather_code&timezone=auto&timeformat=unixtime", wrappedURLs["ometeoWrappedURL"])
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

	location := Location{Latitude: 51.1093, Longitude: 17.0386} // Example coordinates for Wrocław
	wrappedURLs := cfg.WrapForDailyForecast(location)

	if wrappedURLs["gmpWrappedURL"] == "" || wrappedURLs["owmWrappedURL"] == "" || wrappedURLs["ometeoWrappedURL"] == "" {
		t.Error("Wrapped URLs should not be empty")
	}

	if wrappedURLs["gmpWrappedURL"] != "https://weather.googleapis.com/v1/forecast/days:lookup?key="+gmpKey+"&location.latitude=51.11&location.longitude=17.04" {
		t.Errorf("Expected GMP wrapped URL to be %s, got %s", "https://weather.googleapis.com/v1/forecast/days:lookup?key="+gmpKey+"&location.latitude=51.11&location.longitude=17.04", wrappedURLs["gmpWrappedURL"])
	}

	if wrappedURLs["owmWrappedURL"] != "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=current,minutely,hourly,alerts&units=metric&appid="+owmKey {
		t.Errorf("Expected OWM wrapped URL to be %s, got %s", "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=current,minutely,hourly,alerts&units=metric&appid="+owmKey, wrappedURLs["owmWrappedURL"])
	}

	if wrappedURLs["ometeoWrappedURL"] != "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&daily=temperature_2m_max,temperature_2m_mean,temperature_2m_min,precipitation_sum,precipitation_probability_max,wind_speed_10m_max,weather_code&timezone=auto&timeformat=unixtime" {
		t.Errorf("Expected Ometeo wrapped URL to be %s, got %s", "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&daily=temperature_2m_max,temperature_2m_mean,temperature_2m_min,precipitation_sum,precipitation_probability_max,wind_speed_10m_max,weather_code&timezone=auto&timeformat=unixtime", wrappedURLs["ometeoWrappedURL"])
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

	location := Location{Latitude: 51.1093, Longitude: 17.0386} // Example coordinates for Wrocław
	wrappedURLs := cfg.WrapForHourlyForecast(location)

	if wrappedURLs["gmpWrappedURL"] == "" || wrappedURLs["owmWrappedURL"] == "" || wrappedURLs["ometeoWrappedURL"] == "" {
		t.Error("Wrapped URLs should not be empty")
	}

	if wrappedURLs["gmpWrappedURL"] != "https://weather.googleapis.com/v1/forecast/hours:lookup?key="+gmpKey+"&location.latitude=51.11&location.longitude=17.04" {
		t.Errorf("Expected GMP wrapped URL to be %s, got %s", "https://weather.googleapis.com/v1/forecast/hours:lookup?key="+gmpKey+"&location.latitude=51.11&location.longitude=17.04", wrappedURLs["gmpWrappedURL"])
	}

	if wrappedURLs["owmWrappedURL"] != "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=current,minutely,daily,alerts&units=metric&appid="+owmKey {
		t.Errorf("Expected OWM wrapped URL to be %s, got %s", "https://api.openweathermap.org/data/3.0/onecall?lat=51.11&lon=17.04&exclude=current,minutely,daily,alerts&units=metric&appid="+owmKey, wrappedURLs["owmWrappedURL"])
	}

	if wrappedURLs["ometeoWrappedURL"] != "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&hourly=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,precipitation_probability,weather_code&forecast_days=2&timezone=auto&timeformat=unixtime" {
		t.Errorf("Expected Ometeo wrapped URL to be %s, got %s", "https://api.open-meteo.com/v1/forecast?latitude=51.11&longitude=17.04&hourly=temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,precipitation_probability,weather_code&forecast_days=2&timezone=auto&timeformat=unixtime", wrappedURLs["ometeoWrappedURL"])
	}
}
