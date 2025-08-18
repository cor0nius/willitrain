package main

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"time"
)

func ParseCurrentWeatherGMP(body io.Reader, logger *slog.Logger) (CurrentWeather, error) {
	var response ResponseCurrentWeatherGMP

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return CurrentWeather{SourceAPI: "Google Weather API"}, err
	}
	if response.Timestamp.IsZero() {
		return CurrentWeather{SourceAPI: "Google Weather API"}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.TimeZone.ID)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	weather := CurrentWeather{
		SourceAPI:     "Google Weather API",
		Timestamp:     (response.Timestamp).In(loc),
		Temperature:   response.Temperature.Degrees,
		Humidity:      int(response.Humidity),
		WindSpeed:     response.Wind.Speed.Value,
		Precipitation: response.Precipitation.Qpf.Quantity,
		Condition:     response.Condition.Description.Text,
	}

	return weather, nil
}

func ParseCurrentWeatherOWM(body io.Reader, logger *slog.Logger) (CurrentWeather, error) {
	var response ResponseCurrentWeatherOWM

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return CurrentWeather{SourceAPI: "OpenWeatherMap API"}, err
	}
	if response.CurrentWeather.Dt == 0 {
		return CurrentWeather{SourceAPI: "OpenWeatherMap API"}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.Timezone)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	weather := CurrentWeather{
		SourceAPI:     "OpenWeatherMap API",
		Timestamp:     time.Unix(response.CurrentWeather.Dt, 0).UTC().In(loc),
		Temperature:   response.CurrentWeather.Temp,
		Humidity:      int(response.CurrentWeather.Humidity),
		WindSpeed:     Round(response.CurrentWeather.WindSpeed*3.6, 4),
		Precipitation: response.CurrentWeather.Rain.Quantity + response.CurrentWeather.Snow.Quantity,
		Condition:     response.CurrentWeather.Weather[0].Main,
	}

	return weather, nil
}

func ParseCurrentWeatherOMeteo(body io.Reader, logger *slog.Logger) (CurrentWeather, error) {
	var response ResponseCurrentWeatherOMeteo

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return CurrentWeather{SourceAPI: "Open-Meteo API"}, err
	}
	if response.CurrentWeather.Time == 0 {
		return CurrentWeather{SourceAPI: "Open-Meteo API"}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.Timezone)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	weather := CurrentWeather{
		SourceAPI:     "Open-Meteo API",
		Timestamp:     time.Unix(response.CurrentWeather.Time, 0).UTC().In(loc),
		Temperature:   response.CurrentWeather.Temperature2m,
		Humidity:      int(response.CurrentWeather.RelativeHumidity2m),
		WindSpeed:     response.CurrentWeather.WindSpeed10m,
		Precipitation: response.CurrentWeather.Precipitation,
		Condition:     interpretWeatherCode(response.CurrentWeather.WeatherCode),
	}

	return weather, nil
}

func ParseDailyForecastGMP(body io.Reader, logger *slog.Logger) ([]DailyForecast, error) {
	var response ResponseDailyForecastGMP

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return []DailyForecast{{SourceAPI: "Google Weather API"}}, err
	}
	if len(response.ForecastDays) == 0 {
		return []DailyForecast{{SourceAPI: "Google Weather API"}}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.TimeZone.ID)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	var forecast []DailyForecast
	for i, day := range response.ForecastDays {
		if i >= 5 {
			break
		}
		forecast = append(forecast, DailyForecast{
			SourceAPI:           "Google Weather API",
			ForecastDate:        time.Date((day.Interval.StartTime).Year(), (day.Interval.StartTime).Month(), (day.Interval.StartTime).Day(), 0, 0, 0, 0, loc),
			MinTemp:             day.MinTemperature.Degrees,
			MaxTemp:             day.MaxTemperature.Degrees,
			Precipitation:       day.DaytimeForecast.Precipitation.Qpf.Quantity,
			PrecipitationChance: day.DaytimeForecast.Precipitation.Probability.Percent,
			WindSpeed:           day.DaytimeForecast.Wind.Speed.Value,
			Humidity:            day.DaytimeForecast.RelativeHumidity,
		})
	}

	return forecast, nil
}

func ParseDailyForecastOWM(body io.Reader, logger *slog.Logger) ([]DailyForecast, error) {
	var response ResponseDailyForecastOWM

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return []DailyForecast{{SourceAPI: "OpenWeatherMap API"}}, err
	}
	if len(response.DailyForecast) == 0 {
		return []DailyForecast{{SourceAPI: "OpenWeatherMap API"}}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.Timezone)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	var forecast []DailyForecast
	for i, day := range response.DailyForecast {
		if i >= 5 {
			break
		}
		forecast = append(forecast, DailyForecast{
			SourceAPI:           "OpenWeatherMap API",
			ForecastDate:        time.Date(time.Unix(day.Dt, 0).UTC().Year(), time.Unix(day.Dt, 0).UTC().Month(), time.Unix(day.Dt, 0).UTC().Day(), 0, 0, 0, 0, loc),
			MinTemp:             day.Temp.Min,
			MaxTemp:             day.Temp.Max,
			Precipitation:       day.Rain + day.Snow,
			PrecipitationChance: int(day.Pop * 100),
			WindSpeed:           Round(day.WindSpeed*3.6, 4),
			Humidity:            day.Humidity,
		})
	}

	return forecast, nil
}

func ParseDailyForecastOMeteo(body io.Reader, logger *slog.Logger) ([]DailyForecast, error) {
	var response ResponseDailyForecastOMeteo

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return []DailyForecast{{SourceAPI: "Open-Meteo API"}}, err
	}
	if len(response.DailyForecast.Time) == 0 {
		return []DailyForecast{{SourceAPI: "Open-Meteo API"}}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.Timezone)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	var forecast []DailyForecast
	numDays := len(response.DailyForecast.Time)
	if numDays > 5 {
		numDays = 5
	}

	for i := 0; i < numDays; i++ {
		forecast = append(forecast, DailyForecast{
			SourceAPI:           "Open-Meteo API",
			ForecastDate:        time.Date(time.Unix(response.DailyForecast.Time[i], 0).UTC().Year(), time.Unix(response.DailyForecast.Time[i], 0).UTC().Month(), time.Unix(response.DailyForecast.Time[i], 0).UTC().Day(), 0, 0, 0, 0, loc),
			MinTemp:             response.DailyForecast.Temperature2mMin[i],
			MaxTemp:             response.DailyForecast.Temperature2mMax[i],
			Precipitation:       response.DailyForecast.PrecipitationSum[i],
			PrecipitationChance: response.DailyForecast.PrecipitationProbabilityMax[i],
			WindSpeed:           response.DailyForecast.WindSpeed10mMax[i],
			Humidity:            response.DailyForecast.RelativeHumidity2mMax[i],
		})
	}

	return forecast, nil
}

func ParseHourlyForecastGMP(body io.Reader, logger *slog.Logger) ([]HourlyForecast, error) {
	var response ResponseHourlyForecastGMP

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return []HourlyForecast{{SourceAPI: "Google Weather API"}}, err
	}
	if len(response.ForecastHours) == 0 {
		return []HourlyForecast{{SourceAPI: "Google Weather API"}}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.TimeZone.ID)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	var forecast []HourlyForecast
	for i, hour := range response.ForecastHours {
		if i >= 24 {
			break
		}
		forecast = append(forecast, HourlyForecast{
			SourceAPI:           "Google Weather API",
			ForecastDateTime:    (hour.Interval.StartTime).In(loc),
			Temperature:         hour.Temperature.Degrees,
			Humidity:            hour.Humidity,
			WindSpeed:           hour.Wind.Speed.Value,
			Precipitation:       hour.Precipitation.Qpf.Quantity,
			PrecipitationChance: hour.Precipitation.Probability.Percent,
			Condition:           hour.Condition.Description.Text,
		})
	}

	return forecast, nil
}

func ParseHourlyForecastOWM(body io.Reader, logger *slog.Logger) ([]HourlyForecast, error) {
	var response ResponseHourlyForecastOWM

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return []HourlyForecast{{SourceAPI: "OpenWeatherMap API"}}, err
	}
	if len(response.HourlyForecast) == 0 {
		return []HourlyForecast{{SourceAPI: "OpenWeatherMap API"}}, errors.New("empty or invalid response from API")
	}

	loc, err := time.LoadLocation(response.Timezone)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	var forecast []HourlyForecast
	for i, hour := range response.HourlyForecast {
		if i >= 24 {
			break
		}
		forecast = append(forecast, HourlyForecast{
			SourceAPI:           "OpenWeatherMap API",
			ForecastDateTime:    time.Unix(hour.Dt, 0).UTC().In(loc),
			Temperature:         hour.Temp,
			Humidity:            hour.Humidity,
			WindSpeed:           Round(hour.WindSpeed*3.6, 4),
			Precipitation:       hour.Rain.Quantity + hour.Snow.Quantity,
			PrecipitationChance: int(hour.Pop * 100),
			Condition:           hour.Weather[0].Main,
		})
	}

	return forecast, nil
}

func ParseHourlyForecastOMeteo(body io.Reader, logger *slog.Logger) ([]HourlyForecast, error) {
	var response ResponseHourlyForecastOMeteo

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return []HourlyForecast{{SourceAPI: "Open-Meteo API"}}, err
	}
	if len(response.HourlyForecast.Time) == 0 {
		return []HourlyForecast{{SourceAPI: "Open-Meteo API"}}, errors.New("empty or invalid response from API")
	}

	now := time.Now().UTC()
	startIndex := -1

	for i := 0; i < len(response.HourlyForecast.Time); i++ {
		if time.Unix(response.HourlyForecast.Time[i], 0).After(now.Add(-1 * time.Hour)) {
			startIndex = i
			break
		}
	}

	if startIndex == -1 {
		return []HourlyForecast{{SourceAPI: "Open-Meteo API"}}, errors.New("all forecasts are in the past")
	}

	endIndex := startIndex + 24
	if endIndex > len(response.HourlyForecast.Time) {
		endIndex = len(response.HourlyForecast.Time)
	}

	loc, err := time.LoadLocation(response.Timezone)
	if err != nil {
		logger.Warn("Failed to load timezone, using UTC as fallback", "error", err)
		loc = time.UTC
	}

	var forecast []HourlyForecast

	for i := startIndex; i < endIndex; i++ {
		forecast = append(forecast, HourlyForecast{
			SourceAPI:           "Open-Meteo API",
			ForecastDateTime:    time.Unix(response.HourlyForecast.Time[i], 0).UTC().In(loc),
			Temperature:         response.HourlyForecast.Temperature2m[i],
			Humidity:            response.HourlyForecast.RelativeHumidity2m[i],
			WindSpeed:           response.HourlyForecast.WindSpeed10m[i],
			Precipitation:       response.HourlyForecast.Precipitation[i],
			PrecipitationChance: response.HourlyForecast.PrecipitationProbability[i],
			Condition:           interpretWeatherCode(response.HourlyForecast.WeatherCode[i]),
		})
	}

	return forecast, nil
}

// GMP Structs
type ResponseCurrentWeatherGMP struct {
	Timestamp     time.Time        `json:"currentTime"`
	TimeZone      TimeZone         `json:"timeZone"`
	Temperature   Temperature      `json:"temperature"`
	Humidity      float64          `json:"relativeHumidity"`
	Wind          Wind             `json:"wind"`
	Precipitation Precipitation    `json:"precipitation"`
	Condition     WeatherCondition `json:"weatherCondition"`
}

type ResponseDailyForecastGMP struct {
	ForecastDays []ForecastDay `json:"forecastDays"`
	TimeZone     TimeZone      `json:"timeZone"`
}

type ResponseHourlyForecastGMP struct {
	ForecastHours []ForecastHour `json:"forecastHours"`
	TimeZone      TimeZone       `json:"timeZone"`
}

type ForecastDay struct {
	Interval        Interval        `json:"interval"`
	DaytimeForecast ForecastDayPart `json:"daytimeForecast"`
	MaxTemperature  Temperature     `json:"maxTemperature"`
	MinTemperature  Temperature     `json:"minTemperature"`
}

type ForecastHour struct {
	Interval      Interval         `json:"interval"`
	Condition     WeatherCondition `json:"weatherCondition"`
	Temperature   Temperature      `json:"temperature"`
	Precipitation Precipitation    `json:"precipitation"`
	Wind          Wind             `json:"wind"`
	Humidity      int              `json:"relativeHumidity"`
}

type Interval struct {
	StartTime time.Time `json:"startTime"`
}

type TimeZone struct {
	ID string `json:"id"`
}

type ForecastDayPart struct {
	Condition        WeatherCondition `json:"weatherCondition"`
	Precipitation    Precipitation    `json:"precipitation"`
	Wind             Wind             `json:"wind"`
	RelativeHumidity int              `json:"relativeHumidity"`
}

type Temperature struct {
	Degrees float64 `json:"degrees"`
}

type Wind struct {
	Speed Speed `json:"speed"`
}

type Speed struct {
	Value float64 `json:"value"`
}

type Precipitation struct {
	Qpf         Qpf                      `json:"qpf"`
	Probability PrecipitationProbability `json:"probability"`
}

type Qpf struct {
	Quantity float64 `json:"quantity"`
}

type PrecipitationProbability struct {
	Percent int `json:"percent"`
}

type WeatherCondition struct {
	Description Description `json:"description"`
}

type Description struct {
	Text string `json:"text"`
}

// OWM Structs
type ResponseCurrentWeatherOWM struct {
	CurrentWeather CurrentOWM `json:"current"`
	Timezone       string     `json:"timezone"`
}

type ResponseDailyForecastOWM struct {
	DailyForecast []DailyOWM `json:"daily"`
	Timezone      string     `json:"timezone"`
}

type ResponseHourlyForecastOWM struct {
	HourlyForecast []HourlyOWM `json:"hourly"`
	Timezone       string      `json:"timezone"`
}

type CurrentOWM struct {
	Dt        int64     `json:"dt"`
	Temp      float64   `json:"temp"`
	Humidity  float64   `json:"humidity"`
	WindSpeed float64   `json:"wind_speed"`
	Rain      Rain      `json:"rain"`
	Snow      Snow      `json:"snow"`
	Weather   []Weather `json:"weather"`
}

type DailyOWM struct {
	Dt        int64     `json:"dt"`
	Temp      Temp      `json:"temp"`
	Rain      float64   `json:"rain"`
	Snow      float64   `json:"snow"`
	Weather   []Weather `json:"weather"`
	Pop       float64   `json:"pop"`
	WindSpeed float64   `json:"wind_speed"`
	Humidity  int       `json:"humidity"`
}

type HourlyOWM struct {
	Dt        int64     `json:"dt"`
	Temp      float64   `json:"temp"`
	Humidity  int       `json:"humidity"`
	WindSpeed float64   `json:"wind_speed"`
	Rain      Rain      `json:"rain"`
	Snow      Snow      `json:"snow"`
	Weather   []Weather `json:"weather"`
	Pop       float64   `json:"pop"`
}

type Temp struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Rain struct {
	Quantity float64 `json:"1h"`
}

type Snow struct {
	Quantity float64 `json:"1h"`
}

type Weather struct {
	Main string `json:"main"`
}

// OMeteo Structs
type ResponseCurrentWeatherOMeteo struct {
	CurrentWeather CurrentOMeteo `json:"current"`
	Timezone       string        `json:"timezone"`
}

type ResponseDailyForecastOMeteo struct {
	DailyForecast DailyOMeteo `json:"daily"`
	Timezone      string      `json:"timezone"`
}

type ResponseHourlyForecastOMeteo struct {
	HourlyForecast HourlyOMeteo `json:"hourly"`
	Timezone       string       `json:"timezone"`
}

type CurrentOMeteo struct {
	Time               int64   `json:"time"`
	Temperature2m      float64 `json:"temperature_2m"`
	RelativeHumidity2m float64 `json:"relative_humidity_2m"`
	WindSpeed10m       float64 `json:"wind_speed_10m"`
	Precipitation      float64 `json:"precipitation"`
	WeatherCode        int     `json:"weather_code"`
}

type DailyOMeteo struct {
	Time                        []int64   `json:"time"`
	Temperature2mMax            []float64 `json:"temperature_2m_max"`
	Temperature2mMin            []float64 `json:"temperature_2m_min"`
	PrecipitationSum            []float64 `json:"precipitation_sum"`
	PrecipitationProbabilityMax []int     `json:"precipitation_probability_max"`
	WeatherCode                 []int     `json:"weather_code"`
	WindSpeed10mMax             []float64 `json:"wind_speed_10m_max"`
	RelativeHumidity2mMax       []int     `json:"relative_humidity_2m_max"`
}

type HourlyOMeteo struct {
	Time                     []int64   `json:"time"`
	Temperature2m            []float64 `json:"temperature_2m"`
	RelativeHumidity2m       []int     `json:"relative_humidity_2m"`
	WindSpeed10m             []float64 `json:"wind_speed_10m"`
	Precipitation            []float64 `json:"precipitation"`
	PrecipitationProbability []int     `json:"precipitation_probability"`
	WeatherCode              []int     `json:"weather_code"`
}

// Utility functions

func Round(val float64, precision int) float64 {
	p := math.Pow10(precision)
	return math.Round(val*p) / p
}

func interpretWeatherCode(i int) string {
	switch i {
	case 0:
		return "clear sky"
	case 1:
		return "mainly clear"
	case 2:
		return "partly cloudy"
	case 3:
		return "overcast"
	case 45:
		return "fog"
	case 48:
		return "depositing rime fog"
	case 51:
		return "light drizzle"
	case 53:
		return "moderate drizzle"
	case 55:
		return "dense drizzle"
	case 56:
		return "light freezing drizzle"
	case 57:
		return "dense freezing drizzle"
	case 61:
		return "slight rain"
	case 63:
		return "moderate rain"
	case 65:
		return "heavy rain"
	case 66:
		return "light freezing rain"
	case 67:
		return "heavy freezing rain"
	case 71:
		return "slight snowfall"
	case 73:
		return "moderate snowfall"
	case 75:
		return "heavy snowfall"
	case 77:
		return "snow grains"
	case 80:
		return "slight showers"
	case 81:
		return "moderate showers"
	case 82:
		return "violent showers"
	case 85:
		return "slight snow showers"
	case 86:
		return "heavy snow showers"
	case 95:
		return "thunderstorm"
	case 96:
		return "thunderstorm with slight hail"
	case 99:
		return "thunderstorm with heavy hail"
	default:
		return "unknown code"
	}
}
