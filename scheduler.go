package main

import (
	"context"
	"sync"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
)

// This file implements a scheduler that periodically fetches and updates weather data.
// It manages separate tickers for current, hourly, and daily forecasts and runs the
// update jobs for all stored locations concurrently.

// Scheduler manages the periodic execution of weather data updates.
type Scheduler struct {
	cfg                *apiConfig
	currentChan        <-chan time.Time
	hourlyChan         <-chan time.Time
	dailyChan          <-chan time.Time
	stop               chan struct{}
	tickers            []*time.Ticker
	currentWeatherJobs func()
	hourlyForecastJobs func()
	dailyForecastJobs  func()
	jobWG              sync.WaitGroup
}

// NewScheduler creates and initializes a new Scheduler instance.
func NewScheduler(cfg *apiConfig, currentInterval, hourlyInterval, dailyInterval time.Duration) *Scheduler {
	currentTicker := time.NewTicker(currentInterval)
	hourlyTicker := time.NewTicker(hourlyInterval)
	dailyTicker := time.NewTicker(dailyInterval)
	s := &Scheduler{
		cfg:         cfg,
		currentChan: currentTicker.C,
		hourlyChan:  hourlyTicker.C,
		dailyChan:   dailyTicker.C,
		stop:        make(chan struct{}),
		tickers:     []*time.Ticker{currentTicker, hourlyTicker, dailyTicker},
	}
	s.currentWeatherJobs = s.runCurrentWeatherJobs
	s.hourlyForecastJobs = s.runHourlyForecastJobs
	s.dailyForecastJobs = s.runDailyForecastJobs
	return s
}

// Start begins the scheduler's main loop in a new goroutine.
// It listens on the ticker channels and triggers the corresponding job functions.
func (s *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-s.currentChan:
				s.cfg.logger.Info("running scheduler jobs", "type", "current weather")
				s.jobWG.Add(1)
				s.currentWeatherJobs()
				s.jobWG.Done()
			case <-s.hourlyChan:
				s.cfg.logger.Info("running scheduler jobs", "type", "hourly forecast")
				s.jobWG.Add(1)
				s.hourlyForecastJobs()
				s.jobWG.Done()
			case <-s.dailyChan:
				s.cfg.logger.Info("running scheduler jobs", "type", "daily forecast")
				s.jobWG.Add(1)
				s.dailyForecastJobs()
				s.jobWG.Done()
			case <-s.stop:
				s.cfg.logger.Info("stopping scheduler")
				for _, ticker := range s.tickers {
					ticker.Stop()
				}
				return
			}
		}
	}()
}

// Stop gracefully shuts down the scheduler.
// It stops all tickers and waits for any running jobs to complete.
func (s *Scheduler) Stop() {
	close(s.stop)
	s.jobWG.Wait()
	s.cfg.logger.Info("scheduler stopped")
}

// runUpdateForLocations retrieves all locations from the database and runs a given update
// function for each one concurrently.
func (s *Scheduler) runUpdateForLocations(jobType string, updateFunc func(context.Context, Location)) {
	ctx := context.Background()
	locations, err := s.cfg.dbQueries.ListLocations(ctx)
	if err != nil {
		s.cfg.logger.Error("scheduler failed to get locations", "error", err)
		return
	}

	var wg sync.WaitGroup
	for _, dbLocation := range locations {
		wg.Add(1)
		go func(loc database.Location) {
			defer wg.Done()
			location := databaseLocationToLocation(loc)
			updateFunc(ctx, location)
		}(dbLocation)
	}
	wg.Wait()
	s.cfg.logger.Info("scheduler jobs for this cycle completed", "type", jobType)
}

// The run...Jobs functions define the specific update logic for each forecast type.
// They fetch all locations from the database and then, for each location, they delete
// the old data and request new data from the external APIs.
func (s *Scheduler) runCurrentWeatherJobs() {
	updateFunc := func(ctx context.Context, location Location) {
		if err := s.cfg.dbQueries.DeleteCurrentWeatherAtLocation(ctx, location.LocationID); err != nil {
			s.cfg.logger.Error("failed to delete current weather", "location", location.CityName, "error", err)
			return
		}
		weather, err := s.cfg.requestCurrentWeather(location)
		if err != nil {
			s.cfg.logger.Error("failed to request current weather", "location", location.CityName, "error", err)
			return
		}
		s.cfg.persistCurrentWeather(ctx, weather)
		s.cfg.logger.Debug("updated current weather", "location", location.CityName)
	}
	s.runUpdateForLocations("current weather", updateFunc)
}

func (s *Scheduler) runHourlyForecastJobs() {
	updateFunc := func(ctx context.Context, location Location) {
		if err := s.cfg.dbQueries.DeleteHourlyForecastsAtLocation(ctx, location.LocationID); err != nil {
			s.cfg.logger.Error("failed to delete hourly forecasts", "location", location.CityName, "error", err)
			return
		}
		forecast, err := s.cfg.requestHourlyForecast(location)
		if err != nil {
			s.cfg.logger.Error("failed to request hourly forecast", "location", location.CityName, "error", err)
			return
		}
		s.cfg.persistHourlyForecast(ctx, forecast)
		s.cfg.logger.Debug("updated hourly forecast", "location", location.CityName)
	}
	s.runUpdateForLocations("hourly forecast", updateFunc)
}

func (s *Scheduler) runDailyForecastJobs() {
	updateFunc := func(ctx context.Context, location Location) {
		if err := s.cfg.dbQueries.DeleteDailyForecastsAtLocation(ctx, location.LocationID); err != nil {
			s.cfg.logger.Error("failed to delete daily forecasts", "location", location.CityName, "error", err)
			return
		}
		forecast, err := s.cfg.requestDailyForecast(location)
		if err != nil {
			s.cfg.logger.Error("failed to request daily forecast", "location", location.CityName, "error", err)
			return
		}
		s.cfg.persistDailyForecast(ctx, forecast)
		s.cfg.logger.Debug("updated daily forecast", "location", location.CityName)
	}
	s.runUpdateForLocations("daily forecast", updateFunc)
}
