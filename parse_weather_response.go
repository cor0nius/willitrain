package main

import (
	"encoding/json"
	"io"
	"time"
)

func ParseCurrentWeatherGMP(body io.Reader) CurrentWeather {
	data, err := io.ReadAll(body)
	if err != nil {
		return CurrentWeather{SourceAPI: "Google Weather API", Error: err}
	}

	var response ResponseCurrentWeatherGMP
	if err = json.Unmarshal(data, &response); err != nil {
		return CurrentWeather{SourceAPI: "Google Weather API", Error: err}
	}

	weather := CurrentWeather{
		SourceAPI:     "Google Weather API",
		Timestamp:     response.Timestamp,
		Temperature:   response.Temperature.Degrees,
		Humidity:      int(response.Humidity),
		WindSpeed:     response.Wind.Speed.Value,
		Precipitation: response.Precipitation.Qpf.Quantity,
		Condition:     response.Condition.Description.Text,
		Error:         nil,
	}

	return weather
}

func ParseCurrentWeatherOWM(body io.Reader) CurrentWeather {
	data, err := io.ReadAll(body)
	if err != nil {
		return CurrentWeather{SourceAPI: "OpenWeatherMap API", Error: err}
	}

	var response ResponseCurrentWeatherOWM
	if err = json.Unmarshal(data, &response); err != nil {
		return CurrentWeather{SourceAPI: "OpenWeatherMap API", Error: err}
	}

	weather := CurrentWeather{
		SourceAPI:     "OpenWeatherMap API",
		Timestamp:     response.CurrentWeather.Dt,
		Temperature:   response.CurrentWeather.Temp,
		Humidity:      int(response.CurrentWeather.Humidity),
		WindSpeed:     response.CurrentWeather.WindSpeed / 3.6,
		Precipitation: response.CurrentWeather.Rain.Quantity + response.CurrentWeather.Snow.Quantity,
		Condition:     response.CurrentWeather.Weather.Main,
		Error:         nil,
	}

	return weather
}

func ParseCurrentWeatherOMeteo(body io.Reader) CurrentWeather {
	data, err := io.ReadAll(body)
	if err != nil {
		return CurrentWeather{SourceAPI: "Open-Meteo API", Error: err}
	}

	var response ResponseCurrentWeatherOMeteo
	if err = json.Unmarshal(data, &response); err != nil {
		return CurrentWeather{SourceAPI: "Open-Meteo API", Error: err}
	}

	weather := CurrentWeather{
		SourceAPI:     "Open-Meteo API",
		Timestamp:     response.CurrentWeather.Time,
		Temperature:   response.CurrentWeather.Temperature2m,
		Humidity:      int(response.CurrentWeather.RelativeHumidity2m),
		WindSpeed:     response.CurrentWeather.WindSpeed10m,
		Precipitation: response.CurrentWeather.Precipitation,
		Condition:     interpretWeatherCode(response.CurrentWeather.WeatherCode),
		Error:         nil,
	}

	return weather
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

type ResponseCurrentWeatherGMP struct {
	Timestamp     time.Time        `json:"currentTime"`
	Temperature   Temperature      `json:"temperature"`
	Humidity      float64          `json:"relativeHumidity"`
	Wind          Wind             `json:"wind"`
	Precipitation Precipitation    `json:"precipitation"`
	Condition     WeatherCondition `json:"weatherCondition"`
}

type ResponseCurrentWeatherOWM struct {
	CurrentWeather Current `json:"current"`
}

type ResponseCurrentWeatherOMeteo struct {
	CurrentWeather Current `json:"current"`
}

type Current struct {
	Dt                 time.Time        `json:"dt"`
	Time               time.Time        `json:"time"`
	Temp               float64          `json:"temp"`
	Temperature2m      float64          `json:"temperature_2m"`
	Humidity           float64          `json:"humidity"`
	RelativeHumidity2m float64          `json:"relative_humidity_2m"`
	WindSpeed          float64          `json:"wind_speed"`
	WindSpeed10m       float64          `json:"wind_speed_10m"`
	Precipitation      float64          `json:"precipitation"`
	Rain               Rain             `json:"rain"`
	Snow               Snow             `json:"snow"`
	Weather            WeatherCondition `json:"weather"`
	WeatherCode        int              `json:"weather_code"`
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
	Qpf Qpf `json:"qpf"`
}

type Qpf struct {
	Quantity float64 `json:"quantity"`
}

type Rain struct {
	Quantity float64 `json:"1h"`
}

type Snow struct {
	Quantity float64 `json:"1h"`
}

type WeatherCondition struct {
	Description Description `json:"description"`
	Main        string      `json:"main"`
}

type Description struct {
	Text string `json:"text"`
}
