import './style.css';
import { fetchCurrentWeather, fetchDailyForecast, fetchHourlyForecast, API_BASE_URL } from './api';
import { dom, setActiveTab, renderCurrentWeather, renderDailyForecast, renderHourlyForecast, showError, showLoading } from './ui';

// --- Tab Event Listeners ---
dom.tabs.current.addEventListener('click', () => setActiveTab('current'));
dom.tabs.daily.addEventListener('click', () => setActiveTab('daily'));
dom.tabs.hourly.addEventListener('click', () => setActiveTab('hourly'));

// --- Dev Button Event Listeners ---
const DEV_API_URL = API_BASE_URL.replace('api', 'dev');

dom.resetDbBtn.addEventListener('click', async () => {
  if (confirm('Are you sure you want to purge the database and cache?'))
{
  try {
    const response = await fetch(`${DEV_API_URL}/reset-db`, { method: 'POST'});
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const data = await response.json();
    alert(`Success: ${data.status}`);
  } catch (error) {
    alert(`Error: ${(error instanceof Error ? error.message : 'Unknown error')}`);
    }
  }
});

dom.runSchedulerBtn.addEventListener('click', async () => {
  if (confirm('Are you sure you want to manually trigger the scheduler jobs?'))
{
  try {
    const response = await fetch(`${DEV_API_URL}/runschedulerjobs`, { method: 'POST'});
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const data = await response.json();
    alert(`Success: ${data.status}`);
  } catch (error) {
    alert(`Error: ${(error instanceof Error ? error.message : 'Unknown error')}`);
    }
  }
});

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