package main

import (
	"time"

	"github.com/google/uuid"
)

type Location struct {
	LocationID  uuid.UUID `json:"location_id"`
	CityName    string    `json:"city_name"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	CountryCode string    `json:"country_code"`
}

type CurrentWeather struct {
	LocationID    uuid.UUID `json:"location_id"`
	SourceAPI     string    `json:"source_api"`
	Timestamp     time.Time `json:"timestamp"`
	Temperature   float64   `json:"temperature_c"`
	Humidity      int       `json:"humidity"`
	WindSpeed     float64   `json:"wind_speed_kmh"`
	Precipitation float64   `json:"precipitation_mm"`
	Condition     string    `json:"condition_text"`
}

type DailyForecast struct {
	LocationID    uuid.UUID `json:"location_id"`
	SourceAPI     string    `json:"source_api"`
	ForecastDate  time.Time `json:"forecast_date"`
	MinTemp       float64   `json:"min_temp_c"`
	MaxTemp       float64   `json:"max_temp_c"`
	AvgTemp       float64   `json:"avg_temp_c"`
	Precipitation float64   `json:"precipitation_mm"`
	ChanceOfRain  int       `json:"chance_of_rain"`
}

type HourlyForecast struct {
	LocationID       uuid.UUID `json:"location_id"`
	SourceAPI        string    `json:"source_api"`
	ForecastDateTime time.Time `json:"forecast_datetime"`
	Temperature      float64   `json:"min_temp_c"`
	Humidity         int       `json:"humidity"`
	WindSpeed        float64   `json:"wind_speed_kmh"`
	Precipitation    float64   `json:"precipitation_mm"`
}
