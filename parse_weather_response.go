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
		SourceAPI: "Google Weather API"
		Timestamp: response.Timestamp
		Temperature: response.Temperature.Degrees
		Humidity: response.Humidity
		WindSpeed: response.Wind.Speed.Value
		Precipitation: response.Precipitation.Qpf.Quantity
		Condition: response.Condition.Description.Text
		Error: nil
	}

	return weather
}

type ResponseCurrentWeatherGMP struct {
	Timestamp time.Time `json:"currentTime"`
	Temperature Temperature `json:"temperature"`
	Humidity float64 `json:"relativeHumidity"`
	Wind Wind `json:"wind"`
	Precipitation Precipitation `json:"precipitation"`
	Condition WeatherCondition `json:"weatherCondition"`
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

type WeatherCondition struct {
	Description Description `json:"description"`
}

type Description struct {
	Text string `json:"text"`
}