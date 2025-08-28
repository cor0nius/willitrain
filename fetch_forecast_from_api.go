package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// fetchForecastFromAPI provides a generic and concurrent mechanism for fetching and
// parsing weather data from an external API.
//
// This function is central to the application's data aggregation strategy. Its key features are:
//   - Generics (`[T Forecast]`): It can be used to fetch any type of forecast
//     (current, daily, hourly) without code duplication.
//   - Parser Function: It accepts a parser function as an argument, decoupling the
//     core fetching logic from the specific data format of each external API.
//   - Concurrency: It is designed to be run in a separate goroutine and uses a
//     `sync.WaitGroup` and a channel to manage concurrent operations and return
//     results safely.
//
// This design allows the application to efficiently query multiple weather APIs in
// parallel, improving performance and resilience.
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

	// Instrument the parser duration.
	start := time.Now()
	data, tz, err := parser(resp.Body, cfg.logger)
	duration := time.Since(start).Seconds()

	// Determine provider and forecast type for metric labels.
	var provider, forecastType string
	v := any(errorVal)
	switch val := v.(type) {
	case CurrentWeather:
		provider = val.SourceAPI
		forecastType = "current"
	case []DailyForecast:
		if len(val) > 0 {
			provider = val[0].SourceAPI
		}
		forecastType = "daily"
	case []HourlyForecast:
		if len(val) > 0 {
			provider = val[0].SourceAPI
		}
		forecastType = "hourly"
	}
	if provider != "" {
		parserDuration.WithLabelValues(provider, forecastType).Observe(duration)
	}

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
