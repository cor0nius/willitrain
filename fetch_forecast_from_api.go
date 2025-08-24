package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

func fetchForecastFromAPI[T Forecast](
	cfg *apiConfig,
	url string,
	parser func(body io.Reader, logger *slog.Logger) (T, string, error),
	errorVal T,
	wg *sync.WaitGroup,
	results chan<- struct {
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
