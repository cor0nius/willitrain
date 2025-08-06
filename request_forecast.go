package main

import (
	"io"
	"log"
	"sync"
)

func (cfg *apiConfig) requestCurrentWeather(location Location) ([]CurrentWeather, error) {
	urls := cfg.WrapForCurrentWeather(location)

	providers := map[string]forecastProvider[CurrentWeather]{
		"gmpWrappedURL": {
			parser:   ParseCurrentWeatherGMP,
			errorVal: CurrentWeather{SourceAPI: "Google Weather API"},
		},
		"owmWrappedURL": {
			parser:   ParseCurrentWeatherOWM,
			errorVal: CurrentWeather{SourceAPI: "OpenWeatherMap API"},
		},
		"ometeoWrappedURL": {
			parser:   ParseCurrentWeatherOMeteo,
			errorVal: CurrentWeather{SourceAPI: "Open-Meteo API"},
		},
	}

	return processForecastRequests(urls, providers)
}

func (cfg *apiConfig) requestDailyForecast(location Location) ([]DailyForecast, error) {
	urls := cfg.WrapForDailyForecast(location)

	providers := map[string]forecastProvider[[]DailyForecast]{
		"gmpWrappedURL": {
			parser:   ParseDailyForecastGMP,
			errorVal: []DailyForecast{{SourceAPI: "Google Weather API"}},
		},
		"owmWrappedURL": {
			parser:   ParseDailyForecastOWM,
			errorVal: []DailyForecast{{SourceAPI: "OpenWeatherMap API"}},
		},
		"ometeoWrappedURL": {
			parser:   ParseDailyForecastOMeteo,
			errorVal: []DailyForecast{{SourceAPI: "Open-Meteo API"}},
		},
	}

	results, err := processForecastRequests(urls, providers)
	if err != nil {
		return nil, err
	}

	var allForecasts []DailyForecast
	for _, forecastSlice := range results {
		allForecasts = append(allForecasts, forecastSlice...)
	}

	return allForecasts, nil
}

func (cfg *apiConfig) requestHourlyForecast(location Location) ([]HourlyForecast, error) {
	urls := cfg.WrapForHourlyForecast(location)

	providers := map[string]forecastProvider[[]HourlyForecast]{
		"gmpWrappedURL": {
			parser:   ParseHourlyForecastGMP,
			errorVal: []HourlyForecast{{SourceAPI: "Google Weather API"}},
		},
		"owmWrappedURL": {
			parser:   ParseHourlyForecastOWM,
			errorVal: []HourlyForecast{{SourceAPI: "OpenWeatherMap API"}},
		},
		"ometeoWrappedURL": {
			parser:   ParseHourlyForecastOMeteo,
			errorVal: []HourlyForecast{{SourceAPI: "Open-Meteo API"}},
		},
	}

	results, err := processForecastRequests(urls, providers)
	if err != nil {
		return nil, err
	}

	var allForecasts []HourlyForecast
	for _, forecastSlice := range results {
		allForecasts = append(allForecasts, forecastSlice...)
	}

	return allForecasts, nil
}

func processForecastRequests[T Forecast](
	urls map[string]string,
	providers map[string]forecastProvider[T],
) ([]T, error) {
	var wg sync.WaitGroup
	results := make(chan struct {
		t   T
		err error
	}, len(urls))

	for key, url := range urls {
		if provider, ok := providers[key]; ok {
			wg.Add(1)
			go fetchForecastFromAPI(url, provider.parser, provider.errorVal, &wg, results)
		} else {
			log.Printf("No provider found for key: %s", key)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []T
	for res := range results {
		if res.err != nil {
			var sourceAPI string
			v := any(res.t)
			switch v := v.(type) {
			case CurrentWeather:
				sourceAPI = v.SourceAPI
			case []DailyForecast:
				if len(v) > 0 {
					sourceAPI = v[0].SourceAPI
				}
			case []HourlyForecast:
				if len(v) > 0 {
					sourceAPI = v[0].SourceAPI
				}
			}
			if sourceAPI != "" {
				log.Printf("Error fetching forecast from %s: %v", sourceAPI, res.err)
			} else {
				log.Printf("Error fetching forecast: %v", res.err)
			}
		} else {
			allResults = append(allResults, res.t)
		}
	}

	return allResults, nil
}

type forecastProvider[T Forecast] struct {
	parser   func(io.Reader) (T, error)
	errorVal T
}
