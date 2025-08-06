package main

import (
	"log"
	"sync"
)

func (cfg *apiConfig) requestCurrentWeather(location Location) ([]CurrentWeather, error) {
	urls := cfg.WrapForCurrentWeather(location)

	var wg sync.WaitGroup
	results := make(chan struct {
		t   CurrentWeather
		err error
	}, len(urls))

	for key, url := range urls {
		wg.Add(1)
		go func(key, url string) {
			switch key {
			case "gmpWrappedURL":
				fetchForecastFromAPI(url, ParseCurrentWeatherGMP, CurrentWeather{SourceAPI: "Google Weather API"}, &wg, results)
			case "owmWrappedURL":
				fetchForecastFromAPI(url, ParseCurrentWeatherOWM, CurrentWeather{SourceAPI: "OpenWeatherMap API"}, &wg, results)
			case "ometeoWrappedURL":
				fetchForecastFromAPI(url, ParseCurrentWeatherOMeteo, CurrentWeather{SourceAPI: "Open-Meteo API"}, &wg, results)
			default:
				log.Printf("Unknown weather API key for current weather: %s", key)
				wg.Done()
			}
		}(key, url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allWeather []CurrentWeather
	for res := range results {
		if res.err != nil {
			log.Printf("Error fetching current weather from %s: %v", res.t.SourceAPI, res.err)
		}
		allWeather = append(allWeather, res.t)
	}

	return allWeather, nil
}

func requestDailyForecast(location Location) ([]DailyForecast, error) {
	// Implementation for requesting daily forecast data
	// This is a placeholder function and should be replaced with actual logic
	return []DailyForecast{}, nil
}

func requestHourlyForecast(location Location) ([]HourlyForecast, error) {
	// Implementation for requesting hourly forecast data
	// This is a placeholder function and should be replaced with actual logic
	return []HourlyForecast{}, nil
}
