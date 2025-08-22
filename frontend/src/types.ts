export interface Location {
  city_name: string;
}

export interface CurrentWeather {
  temperature_c: number;
  condition_text: string;
  wind_speed_kmh: number;
  humidity: number;
  precipitation_mm: number;
  source_api: string;
  timestamp: string;
}

export interface DailyForecast {
  min_temp_c: number;
  max_temp_c: number;
  precipitation_mm: number;
  precipitation_chance: number;
  wind_speed_kmh: number;
  humidity: number;
  source_api: string;
  forecast_date: string;
}

export interface HourlyForecast {
  temperature_c: number;
  condition_text: string;
  precipitation_mm: number;
  precipitation_chance: number;
  wind_speed_kmh: number;
  humidity: number;
  source_api: string;
  forecast_datetime: string;
}

export interface CurrentWeatherResponse {
  location: Location;
  weather: CurrentWeather[];
}

export interface DailyForecastsResponse {
  location: Location;
  forecasts: DailyForecast[];
}

export interface HourlyForecastsResponse {
  location: Location;
  forecasts: HourlyForecast[];
}
