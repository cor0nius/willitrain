package main

import (
	"time"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

// This file defines the core data structures for the WillItRain application.
// It establishes a clear separation between the internal business logic models and
// the data transfer objects (DTOs) used for API JSON responses.
//
// - Business Logic Structs (e.g., Location, CurrentWeather): These are the primary,
//   rich models used throughout the application's core logic. They are decoupled
//   from both the database schema and the API response format.
// - JSON Structs (e.g., CurrentWeatherJSON): These structs define the precise
//   shape of the JSON data sent to the client. This separation allows the API
//   contract to evolve independently of the internal data models.

// --- Business Logic Models ---

// Location represents the core geographic and identifying details of a place.
type Location struct {
	LocationID  uuid.UUID `json:"location_id"`
	CityName    string    `json:"city_name"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	CountryCode string    `json:"country_code"`
	Timezone    string    `json:"timezone,omitempty"`
}

// CurrentWeather is the internal model for weather conditions at a specific moment.
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

// DailyForecast is the internal model for predicted weather conditions for a full day.
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

// HourlyForecast is the internal model for predicted weather conditions for a specific hour.
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

// --- API Response DTOs (JSON Models) ---

// CurrentWeatherJSON defines the JSON structure for current weather data in API responses.
type CurrentWeatherJSON struct {
	SourceAPI     string  `json:"source_api"`
	Timestamp     string  `json:"timestamp"`
	Temperature   float64 `json:"temperature_c"`
	Humidity      int32   `json:"humidity"`
	WindSpeed     float64 `json:"wind_speed_kmh"`
	Precipitation float64 `json:"precipitation_mm"`
	Condition     string  `json:"condition_text"`
}

// DailyForecastJSON defines the JSON structure for daily forecast data in API responses.
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

// HourlyForecastJSON defines the JSON structure for hourly forecast data in API responses.
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

// CurrentWeatherResponse is the top-level JSON structure for the /api/currentweather endpoint.
type CurrentWeatherResponse struct {
	Location Location             `json:"location"`
	Weather  []CurrentWeatherJSON `json:"weather"`
}

// DailyForecastsResponse is the top-level JSON structure for the /api/dailyforecast endpoint.
type DailyForecastsResponse struct {
	Location  Location            `json:"location"`
	Forecasts []DailyForecastJSON `json:"forecasts"`
}

// HourlyForecastsResponse is the top-level JSON structure for the /api/hourlyforecast endpoint.
type HourlyForecastsResponse struct {
	Location  Location             `json:"location"`
	Forecasts []HourlyForecastJSON `json:"forecasts"`
}

// ErrorResponse standardizes the JSON structure for error messages returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ConfigResponse defines the JSON structure for the /api/config endpoint.
type ConfigResponse struct {
	DevMode         bool   `json:"dev_mode"`
	CurrentInterval string `json:"current_interval"`
	HourlyInterval  string `json:"hourly_interval"`
	DailyInterval   string `json:"daily_interval"`
}

// --- Generic Type Constraints ---

// Forecast is a generic type constraint that allows functions to work with any of the forecast types.
type Forecast interface {
	CurrentWeather | []DailyForecast | []HourlyForecast
}

// apiModel and dbModel are generic type constraints used in the caching and persistence helpers.
type apiModel interface {
	CurrentWeather | DailyForecast | HourlyForecast
}

type dbModel interface {
	database.CurrentWeather | database.DailyForecast | database.HourlyForecast
}
