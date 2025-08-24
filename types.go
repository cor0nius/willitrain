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
	Timezone    string    `json:"timezone,omitempty"`
}

type CurrentWeather struct {
	Location      Location
	SourceAPI     string
	Timestamp     time.Time
	Temperature   float64
	Humidity      int32
	WindSpeed     float64
	Precipitation float64
	Condition     string
}

type DailyForecast struct {
	Location            Location
	SourceAPI           string
	Timestamp           time.Time
	ForecastDate        time.Time
	MinTemp             float64
	MaxTemp             float64
	Precipitation       float64
	PrecipitationChance int32
	WindSpeed           float64
	Humidity            int32
}

type HourlyForecast struct {
	Location            Location
	SourceAPI           string
	Timestamp           time.Time
	ForecastDateTime    time.Time
	Temperature         float64
	Humidity            int32
	WindSpeed           float64
	Precipitation       float64
	PrecipitationChance int32
	Condition           string
}

type Forecast interface {
	CurrentWeather | []DailyForecast | []HourlyForecast
}

type CurrentWeatherJSON struct {
	SourceAPI     string  `json:"source_api"`
	Timestamp     string  `json:"timestamp"`
	Temperature   float64 `json:"temperature_c"`
	Humidity      int32   `json:"humidity"`
	WindSpeed     float64 `json:"wind_speed_kmh"`
	Precipitation float64 `json:"precipitation_mm"`
	Condition     string  `json:"condition_text"`
}

type DailyForecastJSON struct {
	SourceAPI           string  `json:"source_api"`
	ForecastDate        string  `json:"forecast_date"`
	MinTemp             float64 `json:"min_temp_c"`
	MaxTemp             float64 `json:"max_temp_c"`
	Precipitation       float64 `json:"precipitation_mm"`
	PrecipitationChance int32   `json:"precipitation_chance"`
	WindSpeed           float64 `json:"wind_speed_kmh"`
	Humidity            int32   `json:"humidity"`
}

type HourlyForecastJSON struct {
	SourceAPI           string  `json:"source_api"`
	ForecastDateTime    string  `json:"forecast_datetime"`
	Temperature         float64 `json:"temperature_c"`
	Humidity            int32   `json:"humidity"`
	WindSpeed           float64 `json:"wind_speed_kmh"`
	Precipitation       float64 `json:"precipitation_mm"`
	PrecipitationChance int32   `json:"precipitation_chance"`
	Condition           string  `json:"condition_text"`
}

type CurrentWeatherResponse struct {
	Location Location             `json:"location"`
	Weather  []CurrentWeatherJSON `json:"weather"`
}

type DailyForecastsResponse struct {
	Location  Location            `json:"location"`
	Forecasts []DailyForecastJSON `json:"forecasts"`
}

type HourlyForecastsResponse struct {
	Location  Location             `json:"location"`
	Forecasts []HourlyForecastJSON `json:"forecasts"`
}
