package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

func fetchForecastFromAPI[T Forecast](
	url string,
	parser func(body io.Reader) (T, error),
	errorVal T,
	wg *sync.WaitGroup,
	results chan<- struct {
		t   T
		err error
	},
) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		results <- struct {
			t   T
			err error
		}{t: errorVal, err: err}
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		results <- struct {
			t   T
			err error
		}{t: errorVal, err: fmt.Errorf("failed to fetch forecast: %s", resp.Status)}
		return
	}

	data, err := parser(resp.Body)
	if err != nil {
		results <- struct {
			t   T
			err error
		}{t: data, err: err}
		return
	}

	results <- struct {
		t   T
		err error
	}{t: data, err: nil}
}
