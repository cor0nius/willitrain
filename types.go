package main

import (
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

// Location represents the geographic and identifying details of a specific place.
type Location struct {
	LocationID  uuid.UUID `json:"location_id"`
	CityName    string    `json:"city_name"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	CountryCode string    `json:"country_code"`
	Timezone    string    `json:"timezone,omitempty"`
}

// CurrentWeather holds the weather conditions for a location at a specific moment.
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

// DailyForecast represents the predicted weather conditions for a full day.
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

// HourlyForecast represents the predicted weather conditions for a specific hour.
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

// Forecast is a generic type constraint that allows functions to work with any of the forecast types.
type Forecast interface {
	CurrentWeather | []DailyForecast | []HourlyForecast
}

// CurrentWeatherJSON is the structure used to serialize current weather data to JSON for API responses.
type CurrentWeatherJSON struct {
	SourceAPI     string  `json:"source_api"`
	Timestamp     string  `json:"timestamp"`
	Temperature   float64 `json:"temperature_c"`
	Humidity      int32   `json:"humidity"`
	WindSpeed     float64 `json:"wind_speed_kmh"`
	Precipitation float64 `json:"precipitation_mm"`
	Condition     string  `json:"condition_text"`
}

// DailyForecastJSON is the structure used to serialize daily forecast data to JSON.
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

// HourlyForecastJSON is the structure used to serialize hourly forecast data to JSON.
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

// CurrentWeatherResponse is the top-level structure for the /api/currentweather endpoint.
type CurrentWeatherResponse struct {
	Location Location             `json:"location"`
	Weather  []CurrentWeatherJSON `json:"weather"`
}

// DailyForecastsResponse is the top-level structure for the /api/dailyforecast endpoint.
type DailyForecastsResponse struct {
	Location  Location            `json:"location"`
	Forecasts []DailyForecastJSON `json:"forecasts"`
}

// HourlyForecastsResponse is the top-level structure for the /api/hourlyforecast endpoint.
type HourlyForecastsResponse struct {
	Location  Location             `json:"location"`
	Forecasts []HourlyForecastJSON `json:"forecasts"`
}

// apiModel and dbModel are generic type constraints used in the caching and persistence helpers.
type apiModel interface {
	CurrentWeather | DailyForecast | HourlyForecast
}

type dbModel interface {
	database.CurrentWeather | database.DailyForecast | database.HourlyForecast
}
