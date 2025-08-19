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
	Location      Location  `json:"location"`
	SourceAPI     string    `json:"source_api"`
	Timestamp     time.Time `json:"timestamp"`
	Temperature   float64   `json:"temperature_c"`
	Humidity      int32     `json:"humidity"`
	WindSpeed     float64   `json:"wind_speed_kmh"`
	Precipitation float64   `json:"precipitation_mm"`
	Condition     string    `json:"condition_text"`
}

type DailyForecast struct {
	Location            Location  `json:"location"`
	SourceAPI           string    `json:"source_api"`
	Timestamp           time.Time `json:"timestamp"`
	ForecastDate        time.Time `json:"forecast_date"`
	MinTemp             float64   `json:"min_temp_c"`
	MaxTemp             float64   `json:"max_temp_c"`
	Precipitation       float64   `json:"precipitation_mm"`
	PrecipitationChance int32     `json:"precipitation_chance"`
	WindSpeed           float64   `json:"wind_speed_kmh"`
	Humidity            int32     `json:"humidity"`
}

type HourlyForecast struct {
	Location            Location  `json:"location"`
	SourceAPI           string    `json:"source_api"`
	Timestamp           time.Time `json:"timestamp"`
	ForecastDateTime    time.Time `json:"forecast_datetime"`
	Temperature         float64   `json:"temperature_c"`
	Humidity            int32     `json:"humidity"`
	WindSpeed           float64   `json:"wind_speed_kmh"`
	Precipitation       float64   `json:"precipitation_mm"`
	PrecipitationChance int32     `json:"precipitation_chance"`
	Condition           string    `json:"condition_text"`
}

type Forecast interface {
	CurrentWeather | []DailyForecast | []HourlyForecast
}
