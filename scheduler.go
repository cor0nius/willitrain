package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
)

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
}

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

func (s *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-s.currentChan:
				log.Println("Scheduler: Running current weather jobs...")
				s.currentWeatherJobs()
			case <-s.hourlyChan:
				log.Println("Scheduler: Running hourly forecast jobs...")
				s.hourlyForecastJobs()
			case <-s.dailyChan:
				log.Println("Scheduler: Running daily forecast jobs...")
				s.dailyForecastJobs()
			case <-s.stop:
				log.Println("Scheduler: Stopping...")
				for _, ticker := range s.tickers {
					ticker.Stop()
				}
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	// TODO: Implement a more graceful shutdown.
	// The current implementation signals the scheduler to stop, but doesn't wait
	// for the currently running jobs to complete. A sync.WaitGroup could be
	// added to the Scheduler struct and used in runUpdateForLocations to
	// ensure that the Stop() method blocks until all active jobs are finished.
	close(s.stop)
}

func (s *Scheduler) runUpdateForLocations(updateFunc func(context.Context, Location)) {
	ctx := context.Background()
	locations, err := s.cfg.dbQueries.ListLocations(ctx)
	if err != nil {
		log.Printf("Scheduler: failed to get locations: %v", err)
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
	log.Println("Scheduler: All jobs for this cycle completed.")
}

func (s *Scheduler) runCurrentWeatherJobs() {
	updateFunc := func(ctx context.Context, location Location) {
		weather, err := s.cfg.requestCurrentWeather(location)
		if err != nil {
			log.Printf("Scheduler: failed to request current weather for %s: %v", location.CityName, err)
			return
		}
		s.cfg.persistCurrentWeather(ctx, weather)
		log.Printf("Scheduler: Updated current weather for %s", location.CityName)
	}
	s.runUpdateForLocations(updateFunc)
}

func (s *Scheduler) runHourlyForecastJobs() {
	updateFunc := func(ctx context.Context, location Location) {
		forecast, err := s.cfg.requestHourlyForecast(location)
		if err != nil {
			log.Printf("Scheduler: failed to request hourly forecast for %s: %v", location.CityName, err)
			return
		}
		s.cfg.persistHourlyForecast(ctx, forecast)
		log.Printf("Scheduler: Updated hourly forecast for %s", location.CityName)
	}
	s.runUpdateForLocations(updateFunc)
}

func (s *Scheduler) runDailyForecastJobs() {
	updateFunc := func(ctx context.Context, location Location) {
		forecast, err := s.cfg.requestDailyForecast(location)
		if err != nil {
			log.Printf("Scheduler: failed to request daily forecast for %s: %v", location.CityName, err)
			return
		}
		s.cfg.persistDailyForecast(ctx, forecast)
		log.Printf("Scheduler: Updated daily forecast for %s", location.CityName)
	}
	s.runUpdateForLocations(updateFunc)
}
