package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cor0nius/willitrain/internal/database"
)

type Scheduler struct {
	cfg           *apiConfig
	currentTicker *time.Ticker
	hourlyTicker  *time.Ticker
	dailyTicker   *time.Ticker
	stop          chan struct{}
}

func NewScheduler(cfg *apiConfig, currentInterval, hourlyInterval, dailyInterval time.Duration) *Scheduler {
	return &Scheduler{
		cfg:           cfg,
		currentTicker: time.NewTicker(currentInterval),
		hourlyTicker:  time.NewTicker(hourlyInterval),
		dailyTicker:   time.NewTicker(dailyInterval),
		stop:          make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-s.currentTicker.C:
				log.Println("Scheduler: Running current weather jobs...")
				s.runCurrentWeatherJobs()
			case <-s.hourlyTicker.C:
				log.Println("Scheduler: Running hourly forecast jobs...")
				s.runHourlyForecastJobs()
			case <-s.dailyTicker.C:
				log.Println("Scheduler: Running daily forecast jobs...")
				s.runDailyForecastJobs()
			case <-s.stop:
				log.Println("Scheduler: Stopping...")
				s.currentTicker.Stop()
				s.hourlyTicker.Stop()
				s.dailyTicker.Stop()
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
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
	s.runUpdateForLocations(func(ctx context.Context, location Location) {
		weather, err := s.cfg.requestCurrentWeather(location)
		if err != nil {
			log.Printf("Scheduler: failed to request current weather for %s: %v", location.CityName, err)
			return
		}
		s.cfg.persistCurrentWeather(ctx, weather)
		log.Printf("Scheduler: Updated current weather for %s", location.CityName)
	})
}

func (s *Scheduler) runHourlyForecastJobs() {
	s.runUpdateForLocations(func(ctx context.Context, location Location) {
		forecast, err := s.cfg.requestHourlyForecast(location)
		if err != nil {
			log.Printf("Scheduler: failed to request hourly forecast for %s: %v", location.CityName, err)
			return
		}
		s.cfg.persistHourlyForecast(ctx, forecast)
		log.Printf("Scheduler: Updated hourly forecast for %s", location.CityName)
	})
}

func (s *Scheduler) runDailyForecastJobs() {
	s.runUpdateForLocations(func(ctx context.Context, location Location) {
		forecast, err := s.cfg.requestDailyForecast(location)
		if err != nil {
			log.Printf("Scheduler: failed to request daily forecast for %s: %v", location.CityName, err)
			return
		}
		s.cfg.persistDailyForecast(ctx, forecast)
		log.Printf("Scheduler: Updated daily forecast for %s", location.CityName)
	})
}
