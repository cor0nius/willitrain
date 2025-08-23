import type { CurrentWeatherResponse, DailyForecastsResponse, HourlyForecastsResponse, ConfigResponse } from './types';

export const API_BASE_URL = '/api';

async function fetchFromApi<T>(endpoint: string, location?: string): Promise<T> {
  const url = location ? `${API_BASE_URL}/${endpoint}?city=${encodeURIComponent(location)}` : `${API_BASE_URL}/${endpoint}`;
  const response = await fetch(url);
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

export function fetchConfig(): Promise<ConfigResponse> {
  return fetchFromApi<ConfigResponse>('config');
}
