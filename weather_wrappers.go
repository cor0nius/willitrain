package main

import (
	"fmt"
)

func (cfg *apiConfig) WrapForCurrentWeather(location Location) wrappedURLs {

	gmpWrappedURL := fmt.Sprintf("%scurrentConditions:lookup?key=%s&location.latitude=%f&location.longitude=%f", cfg.gmpWeatherURL, cfg.gmpKey, location.Latitude, location.Longitude)

	owmWrappedURL := fmt.Sprintf("%slat=%f&lon=%f&exclude=minutely,hourly,daily,alerts&units=metric&appid=%s", cfg.owmWeatherURL, location.Latitude, location.Longitude, cfg.owmKey)

	ometeoParameters := "temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,rain,showers,snowfall,weather_code"
	ometeoWrappedURL := fmt.Sprintf("%slatitude=%f&longitude=%f&current=%s&timezone=auto", cfg.ometeoWeatherURL, location.Latitude, location.Longitude, ometeoParameters)

	return wrappedURLs{
		gmpWrappedURL:    gmpWrappedURL,
		owmWrappedURL:    owmWrappedURL,
		ometeoWrappedURL: ometeoWrappedURL,
	}
}

func (cfg *apiConfig) WrapForDailyForecast(location Location) wrappedURLs {

	gmpWrappedURL := fmt.Sprintf("%sforecast/days:lookup?key=%s&location.latitude=%f&location.longitude=%f", cfg.gmpWeatherURL, cfg.gmpKey, location.Latitude, location.Longitude)

	owmWrappedURL := fmt.Sprintf("%slat=%f&lon=%f&exclude=current,minutely,hourly,alerts&units=metric&appid=%s", cfg.owmWeatherURL, location.Latitude, location.Longitude, cfg.owmKey)

	ometeoParameters := "temperature_2m_max,temperature_2m_mean,temperature_2m_min,precipitation_sum,precipitation_probability_max,rain_sum,showers_sum,snowfall_sum,wind_speed_10m_max,weather_code"
	ometeoWrappedURL := fmt.Sprintf("%slatitude=%f&longitude=%f&daily=%s&timezone=auto", cfg.ometeoWeatherURL, location.Latitude, location.Longitude, ometeoParameters)

	return wrappedURLs{
		gmpWrappedURL:    gmpWrappedURL,
		owmWrappedURL:    owmWrappedURL,
		ometeoWrappedURL: ometeoWrappedURL,
	}
}

func (cfg *apiConfig) WrapForHourlyForecast(location Location) wrappedURLs {

	gmpWrappedURL := fmt.Sprintf("%sforecast/hours:lookup?key=%s&location.latitude=%f&location.longitude=%f", cfg.gmpWeatherURL, cfg.gmpKey, location.Latitude, location.Longitude)

	owmWrappedURL := fmt.Sprintf("%slat=%f&lon=%f&exclude=current,minutely,daily,alerts&units=metric&appid=%s", cfg.owmWeatherURL, location.Latitude, location.Longitude, cfg.owmKey)

	ometeoParameters := "temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,precipitation_probability,rain,showers,snowfall,weather_code&forecast_days=2"
	ometeoWrappedURL := fmt.Sprintf("%slatitude=%f&longitude=%f&hourly=%s&timezone=auto", cfg.ometeoWeatherURL, location.Latitude, location.Longitude, ometeoParameters)

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
