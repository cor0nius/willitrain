package main

import (
	"database/sql"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/google/uuid"
)

func databaseLocationToLocation(dbLocation database.Location) Location {
	return Location{
		LocationID:  dbLocation.ID,
		CityName:    dbLocation.CityName,
		Latitude:    dbLocation.Latitude,
		Longitude:   dbLocation.Longitude,
		CountryCode: dbLocation.CountryCode,
	}
}

func locationToCreateLocationParams(location Location) database.CreateLocationParams {
	return database.CreateLocationParams{
		CityName:    location.CityName,
		Latitude:    location.Latitude,
		Longitude:   location.Longitude,
		CountryCode: location.CountryCode,
	}
}

func databaseCurrentWeatherToCurrentWeather(dbWeather database.CurrentWeather, location Location) CurrentWeather {
	return CurrentWeather{
		Location:      location,
		SourceAPI:     dbWeather.SourceApi,
		Timestamp:     dbWeather.UpdatedAt,
		Temperature:   dbWeather.TemperatureC.Float64,
		Humidity:      int(dbWeather.Humidity.Int32),
		WindSpeed:     dbWeather.WindSpeedKmh.Float64,
		Precipitation: dbWeather.PrecipitationMm.Float64,
		Condition:     dbWeather.ConditionText.String,
	}
}

func currentWeatherToCreateCurrentWeatherParams(weather CurrentWeather) database.CreateCurrentWeatherParams {
	return database.CreateCurrentWeatherParams{
		LocationID: weather.Location.LocationID,
		SourceApi:  weather.SourceAPI,
		UpdatedAt:  weather.Timestamp,
		TemperatureC: sql.NullFloat64{
			Float64: weather.Temperature,
			Valid:   true,
		},
		Humidity: sql.NullInt32{
			Int32: int32(weather.Humidity),
			Valid: true,
		},
		WindSpeedKmh: sql.NullFloat64{
			Float64: weather.WindSpeed,
			Valid:   true,
		},
		PrecipitationMm: sql.NullFloat64{
			Float64: weather.Precipitation,
			Valid:   true,
		},
		ConditionText: sql.NullString{
			String: weather.Condition,
			Valid:  true,
		},
	}
}

func currentWeatherToUpdateCurrentWeatherParams(weather CurrentWeather, dbWeatherID uuid.UUID) database.UpdateCurrentWeatherParams {
	return database.UpdateCurrentWeatherParams{
		ID:        dbWeatherID,
		UpdatedAt: weather.Timestamp,
		TemperatureC: sql.NullFloat64{
			Float64: weather.Temperature,
			Valid:   true,
		},
		Humidity: sql.NullInt32{
			Int32: int32(weather.Humidity),
			Valid: true,
		},
		WindSpeedKmh: sql.NullFloat64{
			Float64: weather.WindSpeed,
			Valid:   true,
		},
		PrecipitationMm: sql.NullFloat64{
			Float64: weather.Precipitation,
			Valid:   true,
		},
		ConditionText: sql.NullString{
			String: weather.Condition,
			Valid:  true,
		},
	}
}

func databaseDailyForecastToDailyForecast(dbForecast database.DailyForecast, location Location) DailyForecast {
	return DailyForecast{
		Location:            location,
		SourceAPI:           dbForecast.SourceApi,
		ForecastDate:        dbForecast.ForecastDate,
		MinTemp:             dbForecast.MinTempC.Float64,
		MaxTemp:             dbForecast.MaxTempC.Float64,
		Precipitation:       dbForecast.PrecipitationMm.Float64,
		PrecipitationChance: int(dbForecast.PrecipitationChancePercent.Int32),
		WindSpeed:           dbForecast.WindSpeedKmh.Float64,
		Humidity:            int(dbForecast.Humidity.Int32),
	}
}

func dailyForecastToCreateDailyForecastParams(forecast DailyForecast) database.CreateDailyForecastParams {
	return database.CreateDailyForecastParams{
		LocationID:   forecast.Location.LocationID,
		SourceApi:    forecast.SourceAPI,
		ForecastDate: forecast.ForecastDate,
		MinTempC: sql.NullFloat64{
			Float64: forecast.MinTemp,
			Valid:   true,
		},
		MaxTempC: sql.NullFloat64{
			Float64: forecast.MaxTemp,
			Valid:   true,
		},
		PrecipitationMm: sql.NullFloat64{
			Float64: forecast.Precipitation,
			Valid:   true,
		},
		PrecipitationChancePercent: sql.NullInt32{
			Int32: int32(forecast.PrecipitationChance),
			Valid: true,
		},
		WindSpeedKmh: sql.NullFloat64{
			Float64: forecast.WindSpeed,
			Valid:   true,
		},
		Humidity: sql.NullInt32{
			Int32: int32(forecast.Humidity),
			Valid: true,
		},
	}
}

func dailyForecastToUpdateDailyForecastParams(forecast DailyForecast, dbForecastID uuid.UUID) database.UpdateDailyForecastParams {
	return database.UpdateDailyForecastParams{
		ID:           dbForecastID,
		ForecastDate: forecast.ForecastDate,
		MinTempC: sql.NullFloat64{
			Float64: forecast.MinTemp,
			Valid:   true,
		},
		MaxTempC: sql.NullFloat64{
			Float64: forecast.MaxTemp,
			Valid:   true,
		},
		PrecipitationMm: sql.NullFloat64{
			Float64: forecast.Precipitation,
			Valid:   true,
		},
		PrecipitationChancePercent: sql.NullInt32{
			Int32: int32(forecast.PrecipitationChance),
			Valid: true,
		},
		WindSpeedKmh: sql.NullFloat64{
			Float64: forecast.WindSpeed,
			Valid:   true,
		},
		Humidity: sql.NullInt32{
			Int32: int32(forecast.Humidity),
			Valid: true,
		},
	}
}

func databaseHourlyForecastToHourlyForecast(dbForecast database.HourlyForecast, location Location) HourlyForecast {
	return HourlyForecast{
		Location:            location,
		SourceAPI:           dbForecast.SourceApi,
		ForecastDateTime:    dbForecast.ForecastDatetimeUtc,
		Temperature:         dbForecast.TemperatureC.Float64,
		Humidity:            int(dbForecast.Humidity.Int32),
		WindSpeed:           dbForecast.WindSpeedKmh.Float64,
		Precipitation:       dbForecast.PrecipitationMm.Float64,
		PrecipitationChance: int(dbForecast.PrecipitationChancePercent.Int32),
		Condition:           dbForecast.ConditionText.String,
	}
}

func hourlyForecastToCreateHourlyForecastParams(forecast HourlyForecast) database.CreateHourlyForecastParams {
	return database.CreateHourlyForecastParams{
		LocationID:          forecast.Location.LocationID,
		SourceApi:           forecast.SourceAPI,
		ForecastDatetimeUtc: forecast.ForecastDateTime,
		TemperatureC: sql.NullFloat64{
			Float64: forecast.Temperature,
			Valid:   true,
		},
		Humidity: sql.NullInt32{
			Int32: int32(forecast.Humidity),
			Valid: true,
		},
		WindSpeedKmh: sql.NullFloat64{
			Float64: forecast.WindSpeed,
			Valid:   true,
		},
		PrecipitationMm: sql.NullFloat64{
			Float64: forecast.Precipitation,
			Valid:   true,
		},
		PrecipitationChancePercent: sql.NullInt32{
			Int32: int32(forecast.PrecipitationChance),
			Valid: true,
		},
		ConditionText: sql.NullString{
			String: forecast.Condition,
			Valid:  true,
		},
	}
}

func hourlyForecastToUpdateHourlyForecastParams(forecast HourlyForecast, dbForecastID uuid.UUID) database.UpdateHourlyForecastParams {
	return database.UpdateHourlyForecastParams{
		ID:                  dbForecastID,
		ForecastDatetimeUtc: forecast.ForecastDateTime,
		TemperatureC: sql.NullFloat64{
			Float64: forecast.Temperature,
			Valid:   true,
		},
		Humidity: sql.NullInt32{
			Int32: int32(forecast.Humidity),
			Valid: true,
		},
		WindSpeedKmh: sql.NullFloat64{
			Float64: forecast.WindSpeed,
			Valid:   true,
		},
		PrecipitationMm: sql.NullFloat64{
			Float64: forecast.Precipitation,
			Valid:   true,
		},
		PrecipitationChancePercent: sql.NullInt32{
			Int32: int32(forecast.PrecipitationChance),
			Valid: true,
		},
		ConditionText: sql.NullString{
			String: forecast.Condition,
			Valid:  true,
		},
	}
}
