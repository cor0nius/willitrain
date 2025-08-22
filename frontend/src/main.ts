import './style.css';
import { fetchCurrentWeather, fetchDailyForecast, fetchHourlyForecast } from './api';
import { dom, setActiveTab, renderCurrentWeather, renderDailyForecast, renderHourlyForecast, showError, showLoading } from './ui';

// --- Tab Event Listeners ---
dom.tabs.current.addEventListener('click', () => setActiveTab('current'));
dom.tabs.daily.addEventListener('click', () => setActiveTab('daily'));
dom.tabs.hourly.addEventListener('click', () => setActiveTab('hourly'));

// --- Main Application Logic ---
dom.getWeatherBtn.addEventListener('click', async () => {
  const location = dom.locationInput.value.trim();
  if (!location) {
    showError('current', new Error('Please enter a location.'));
    return;
  }

  // --- Current Weather ---
  showLoading('current');
  try {
    const currentData = await fetchCurrentWeather(location);
    renderCurrentWeather(currentData);
  } catch (error) {
    showError('current', error as Error);
  }

  // --- Daily Forecast ---
  showLoading('daily');
  try {
    const dailyData = await fetchDailyForecast(location);
    renderDailyForecast(dailyData);
  } catch (error) {
    showError('daily', error as Error);
  }

  // --- Hourly Forecast ---
  showLoading('hourly');
  try {
    const hourlyData = await fetchHourlyForecast(location);
    renderHourlyForecast(hourlyData);
  } catch (error) {
    showError('hourly', error as Error);
  }
});