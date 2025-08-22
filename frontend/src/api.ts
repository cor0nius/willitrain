import type { CurrentWeatherResponse, DailyForecastsResponse, HourlyForecastsResponse } from './types';

export const API_BASE_URL = '/api';

async function fetchFromApi<T>(endpoint: string, location: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}/${endpoint}?city=${encodeURIComponent(location)}`);
  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
  }
  return response.json();
}

export function fetchCurrentWeather(location: string): Promise<CurrentWeatherResponse> {
  return fetchFromApi<CurrentWeatherResponse>('currentweather', location);
}

export function fetchDailyForecast(location: string): Promise<DailyForecastsResponse> {
  return fetchFromApi<DailyForecastsResponse>('dailyforecast', location);
}

export function fetchHourlyForecast(location: string): Promise<HourlyForecastsResponse> {
  return fetchFromApi<HourlyForecastsResponse>('hourlyforecast', location);
}
