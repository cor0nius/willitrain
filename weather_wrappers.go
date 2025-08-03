package main

import (
	"fmt"
)

func (cfg *apiConfig) WrapForCurrentWeather(location Location) wrappedURLs {
	gmpBaseURL := "https://weather.googleapis.com/v1/currentConditions:lookup"
	gmpWrappedURL := fmt.Sprintf("%s?key=%s&location.latitude=%f&location.longitude=%f", gmpBaseURL, cfg.gmpKey, location.Latitude, location.Longitude)

	owmBaseURL := "https://api.openweathermap.org/data/3.0/onecall?"
	owmWrappedURL := fmt.Sprintf("%slat=%f&lon=%f&exclude=minutely,hourly,daily,alerts&units=metric&appid=%s", owmBaseURL, location.Latitude, location.Longitude, cfg.owmKey)

	ometeoBaseURL := "https://api.open-meteo.com/v1/forecast?"
	ometeoSuffix := "&current=temperature_2m,precipitation,rain,showers,relative_humidity_2m,weather_code,snowfall,wind_speed_10m"
	ometeoWrappedURL := fmt.Sprintf("%slatitude=%f&longitude=%f%s", ometeoBaseURL, location.Latitude, location.Longitude, ometeoSuffix)

	return wrappedURLs{
		gmpWrappedURL:    gmpWrappedURL,
		owmWrappedURL:    owmWrappedURL,
		ometeoWrappedURL: ometeoWrappedURL,
	}
}

func (cfg *apiConfig) WrapForDailyForecast(location Location) wrappedURLs {
	gmpBaseURL := "https://weather.googleapis.com/v1/forecast/days:lookup"
	gmpWrappedURL := fmt.Sprintf("%s?key=%s&location.latitude=%f&location.longitude=%f", gmpBaseURL, cfg.gmpKey, location.Latitude, location.Longitude)

	owmBaseURL := "https://api.openweathermap.org/data/3.0/onecall?"
	owmWrappedURL := fmt.Sprintf("%slat=%f&lon=%f&exclude=current,minutely,hourly,alerts&units=metric&appid=%s", owmBaseURL, location.Latitude, location.Longitude, cfg.owmKey)

	ometeoBaseURL := "https://api.open-meteo.com/v1/forecast?"
	ometeoSuffix := "&daily=weather_code,temperature_2m_max,temperature_2m_mean,temperature_2m_min,rain_sum,showers_sum,snowfall_sum,precipitation_sum,precipitation_probability_max,wind_speed_10m_max"
	ometeoWrappedURL := fmt.Sprintf("%slatitude=%f&longitude=%f%s", ometeoBaseURL, location.Latitude, location.Longitude, ometeoSuffix)

	return wrappedURLs{
		gmpWrappedURL:    gmpWrappedURL,
		owmWrappedURL:    owmWrappedURL,
		ometeoWrappedURL: ometeoWrappedURL,
	}
}

func (cfg *apiConfig) WrapForHourlyForecast(location Location) wrappedURLs {
	gmpBaseURL := "https://weather.googleapis.com/v1/forecast/hours:lookup"
	gmpWrappedURL := fmt.Sprintf("%s?key=%s&location.latitude=%f&location.longitude=%f", gmpBaseURL, cfg.gmpKey, location.Latitude, location.Longitude)

	owmBaseURL := "https://api.openweathermap.org/data/3.0/onecall?"
	owmWrappedURL := fmt.Sprintf("%slat=%f&lon=%f&exclude=current,minutely,daily,alerts&units=metric&appid=%s", owmBaseURL, location.Latitude, location.Longitude, cfg.owmKey)

	ometeoBaseURL := "https://api.open-meteo.com/v1/forecast?"
	ometeoSuffix := "&hourly=temperature_2m,relative_humidity_2m,showers,rain,snowfall,precipitation_probability,precipitation,weather_code,wind_speed_10m&forecast_days=3"
	ometeoWrappedURL := fmt.Sprintf("%slatitude=%f&longitude=%f%s", ometeoBaseURL, location.Latitude, location.Longitude, ometeoSuffix)

	return wrappedURLs{
		gmpWrappedURL:    gmpWrappedURL,
		owmWrappedURL:    owmWrappedURL,
		ometeoWrappedURL: ometeoWrappedURL,
	}
}

type wrappedURLs struct {
	gmpWrappedURL    string
	owmWrappedURL    string
	ometeoWrappedURL string
}
