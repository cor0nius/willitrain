package main

import (
	"time"

	"github.com/google/uuid"
)

type Location struct {
	LocationID  uuid.UUID `json:"location_id"`
	CityName    string    `json:"city_name"`
	Latitude    float32   `json:"latitude"`
	Longitude   float32   `json:"longitude"`
	CountryCode string    `json:"country_code"`
}

type CurrentWeather struct {
	LocationID    uuid.UUID `json:"location_id"`
	SourceAPI     string    `json:"source_api"`
	Timestamp     time.Time `json:"timestamp"`
	Temperature   float32   `json:"temperature_c"`
	Humidity      int       `json:"humidity"`
	WindSpeed     float32   `json:"wind_speed_kmh"`
	Precipitation float32   `json:"precipitation_mm"`
	Condition     string    `json:"condition_text"`
}

type DailyForecast struct {
	LocationID    uuid.UUID `json:"location_id"`
	SourceAPI     string    `json:"source_api"`
	ForecastDate  time.Time `json:"forecast_date"`
	MinTemp       float32   `json:"min_temp_c"`
	MaxTemp       float32   `json:"max_temp_c"`
	AvgTemp       float32   `json:"avg_temp_c"`
	Precipitation float32   `json:"precipitation_mm"`
	ChanceOfRain  int       `json:"chance_of_rain"`
}

type HourlyForecast struct {
	LocationID       uuid.UUID `json:"location_id"`
	SourceAPI        string    `json:"source_api"`
	ForecastDateTime time.Time `json:"forecast_datetime"`
	Temperature      float32   `json:"min_temp_c"`
	Humidity         int       `json:"humidity"`
	WindSpeed        float32   `json:"wind_speed_kmh"`
	Precipitation    float32   `json:"precipitation_mm"`
}
