package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

// fetchForecastFromAPI is a generic function that concurrently fetches and parses weather data from a given URL.
// It is designed to work with any type of forecast (CurrentWeather, []DailyForecast, []HourlyForecast)
// by accepting a parser function tailored to the specific API response structure.
func fetchForecastFromAPI[T Forecast](
	cfg *apiConfig, // The application's configuration, containing the HTTP client.
	url string, // The specific API endpoint URL to fetch.
	parser func(body io.Reader, logger *slog.Logger) (T, string, error), // A function that takes the HTTP response body and returns the parsed forecast data, a timezone string, and an error.
	errorVal T, // A zero-value instance of the forecast type, used to return a typed nil on error.
	wg *sync.WaitGroup, // A WaitGroup to signal completion of the goroutine.
	results chan<- struct { // A channel to send the parsed data (or an error) back to the caller.
		t   T
		tz  string
		err error
	},
) {
	defer wg.Done()

	resp, err := cfg.httpClient.Get(url)
	if err != nil {
		results <- struct {
			t   T
			tz  string
			err error
		}{t: errorVal, tz: "", err: err}
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		results <- struct {
			t   T
			tz  string
			err error
		}{t: errorVal, tz: "", err: fmt.Errorf("failed to fetch forecast: %s", resp.Status)}
		return
	}

	data, tz, err := parser(resp.Body, cfg.logger)
	if err != nil {
		results <- struct {
			t   T
			tz  string
			err error
		}{t: data, tz: "", err: err}
		return
	}

	results <- struct {
		t   T
		tz  string
		err error
	}{t: data, tz: tz, err: nil}
}
