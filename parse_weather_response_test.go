package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseCurrentWeatherGMP(t *testing.T) {
	sampleJSON := `{"currentTime":"2025-08-04T09:44:48.736691285Z","timeZone":{"id":"Europe/Warsaw"},"isDaytime":true,"weatherCondition":{"iconBaseUri":"https://maps.gstatic.com/weather/v1/cloudy","description":{"text":"Cloudy","languageCode":"en"},"type":"CLOUDY"},"temperature":{"degrees":18.2,"unit":"CELSIUS"},"feelsLikeTemperature":{"degrees":18.2,"unit":"CELSIUS"},"dewPoint":{"degrees":13.4,"unit":"CELSIUS"},"heatIndex":{"degrees":18.2,"unit":"CELSIUS"},"windChill":{"degrees":18.2,"unit":"CELSIUS"},"relativeHumidity":74,"uvIndex":2,"precipitation":{"probability":{"percent":13,"type":"RAIN"},"snowQpf":{"quantity":0,"unit":"MILLIMETERS"},"qpf":{"quantity":0.1321,"unit":"MILLIMETERS"}},"thunderstormProbability":3,"airPressure":{"meanSeaLevelMillibars":1018.55},"wind":{"direction":{"degrees":225,"cardinal":"SOUTHWEST"},"speed":{"value":6,"unit":"KILOMETERS_PER_HOUR"},"gust":{"value":12,"unit":"KILOMETERS_PER_HOUR"}},"visibility":{"distance":16,"unit":"KILOMETERS"},"cloudCover":100,"currentConditionsHistory":{"temperatureChange":{"degrees":1.1,"unit":"CELSIUS"},"maxTemperature":{"degrees":20.6,"unit":"CELSIUS"},"minTemperature":{"degrees":12.1,"unit":"CELSIUS"},"snowQpf":{"quantity":0,"unit":"MILLIMETERS"},"qpf":{"quantity":2.9008,"unit":"MILLIMETERS"}}}`

	timestamp, _ := time.Parse(time.RFC3339Nano, "2025-08-04T09:44:48.736691285Z")
	weather := CurrentWeather{
		SourceAPI:     "Google Weather API",
		Timestamp:     timestamp,
		Temperature:   18.2,
		Humidity:      74,
		WindSpeed:     6.,
		Precipitation: 0.1321,
		Condition:     "Cloudy",
		Error:         nil,
	}

	parsedWeather := ParseCurrentWeatherGMP(strings.NewReader(sampleJSON))

	if parsedWeather != weather {
		t.Errorf("Expected parsed weather to be %v, got %v", weather, parsedWeather)
	}
}

func TestParseCurrentWeatherOWM(t *testing.T) {
	sampleJSON := `{"lat":51.11,"lon":17.04,"timezone":"Europe/Warsaw","timezone_offset":7200,"current":{"dt":1754300711,"sunrise":1754277695,"sunset":1754332454,"temp":17,"feels_like":16.82,"pressure":1019,"humidity":79,"dew_point":13.34,"uvi":1.32,"clouds":20,"visibility":10000,"wind_speed":2.57,"wind_deg":230,"weather":[{"id":500,"main":"Rain","description":"light rain","icon":"10d"}],"rain":{"1h":0.32}}}`

	timestamp := time.Unix(1754300711, 0)
	weather := CurrentWeather{
		SourceAPI:     "OpenWeatherMap API",
		Timestamp:     timestamp,
		Temperature:   17.,
		Humidity:      79,
		WindSpeed:     Round(2.57/3.6, 4),
		Precipitation: 0.32,
		Condition:     "Rain",
		Error:         nil,
	}

	parsedWeather := ParseCurrentWeatherOWM(strings.NewReader(sampleJSON))

	if parsedWeather != weather {
		t.Errorf("Expected parsed weather to be %v, got %v", weather, parsedWeather)
	}
}

func TestParseCurrentWeatherOMeteo(t *testing.T) {
	sampleJSON := `{"latitude":51.12,"longitude":17.039999,"generationtime_ms":0.0598430633544922,"utc_offset_seconds":7200,"timezone":"Europe/Warsaw","timezone_abbreviation":"GMT+2","elevation":122,"current_units":{"time":"unixtime","interval":"seconds","temperature_2m":"°C","relative_humidity_2m":"%","wind_speed_10m":"km/h","precipitation":"mm","weather_code":"wmo code"},"current":{"time":1754300700,"interval":900,"temperature_2m":18.3,"relative_humidity_2m":71,"wind_speed_10m":9,"precipitation":0.1,"weather_code":61}}`

	timestamp := time.Unix(1754300700, 0)
	weather := CurrentWeather{
		SourceAPI:     "Open-Meteo API",
		Timestamp:     timestamp,
		Temperature:   18.3,
		Humidity:      71,
		WindSpeed:     9.,
		Precipitation: 0.1,
		Condition:     "slight rain",
		Error:         nil,
	}

	parsedWeather := ParseCurrentWeatherOMeteo(strings.NewReader(sampleJSON))

	if parsedWeather != weather {
		t.Errorf("Expected parsed weather to be %v, got %v", weather, parsedWeather)
	}
}
