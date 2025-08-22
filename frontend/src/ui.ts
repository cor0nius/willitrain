import type { CurrentWeatherResponse, DailyForecastsResponse, HourlyForecastsResponse, DailyForecast, HourlyForecast } from './types';

// --- Helper Functions ---
function formatDate(date: Date): string {
  const day = String(date.getDate()).padStart(2, '0');
  const month = String(date.getMonth() + 1).padStart(2, '0'); // Months are 0-based
  return `${day}.${month}`;
}

// --- DOM Element References ---
export const dom = {
  locationInput: document.querySelector<HTMLInputElement>('#location-input')!,
  getWeatherBtn: document.querySelector<HTMLButtonElement>('#get-weather-btn')!,
  resetDbBtn: document.querySelector<HTMLButtonElement>('#reset-db-btn')!,
  runSchedulerBtn: document.querySelector<HTMLButtonElement>('#run-scheduler-btn')!,
  tabs: {
    current: document.querySelector<HTMLButtonElement>('#current-tab')!,
    daily: document.querySelector<HTMLButtonElement>('#daily-tab')!,
    hourly: document.querySelector<HTMLButtonElement>('#hourly-tab')!,
  },
  panels: {
    current: document.querySelector<HTMLDivElement>('#current-panel')!,
    daily: document.querySelector<HTMLDivElement>('#daily-panel')!,
    hourly: document.querySelector<HTMLDivElement>('#hourly-panel')!,
  },
  dailyElements: {
    list: document.querySelector<HTMLDivElement>('#daily-list')!,
    details: document.querySelector<HTMLDivElement>('#daily-details')!,
  },
  hourlyElements: {
    list: document.querySelector<HTMLDivElement>('#hourly-list')!,
    details: document.querySelector<HTMLDivElement>('#hourly-details')!,
  },
};

// --- Tab Management ---
export function setActiveTab(tabName: keyof typeof dom.tabs) {
  Object.values(dom.tabs).forEach(tab => tab.classList.remove('active'));
  Object.values(dom.panels).forEach(panel => panel.classList.remove('active'));
  dom.tabs[tabName].classList.add('active');
  dom.panels[tabName].classList.add('active');
}

// --- Rendering Functions ---
export function showLoading(panel: keyof typeof dom.panels) {
  if (panel === 'daily') {
    dom.dailyElements.list.innerHTML = 'Loading...';
    dom.dailyElements.details.innerHTML = '';
  } else if (panel === 'hourly') {
    dom.hourlyElements.list.innerHTML = 'Loading...';
    dom.hourlyElements.details.innerHTML = '';
  } else {
    dom.panels[panel].innerHTML = 'Loading...';
  }
}

export function showError(panel: keyof typeof dom.panels, error: Error) {
  const errorHtml = `<h3>Error</h3><p>${error.message}</p>`;
  if (panel === 'daily') {
    dom.dailyElements.list.innerHTML = errorHtml;
    dom.dailyElements.details.innerHTML = '';
  } else if (panel === 'hourly') {
    dom.hourlyElements.list.innerHTML = errorHtml;
    dom.hourlyElements.details.innerHTML = '';
  } else {
    dom.panels[panel].innerHTML = errorHtml;
  }
}

export function renderCurrentWeather(data: CurrentWeatherResponse) {
  if (!data.weather || data.weather.length === 0) {
    throw new Error('No weather data available for this location.');
  }
  const weatherHtml = data.weather.map(weather => `
    <div class="weather-card">
      <p><strong>Temperature:</strong> ${weather.temperature_c.toFixed(1)} °C</p>
      <p><strong>Condition:</strong> ${weather.condition_text}</p>
      <p><strong>Wind:</strong> ${weather.wind_speed_kmh.toFixed(1)} km/h</p>
      <p><strong>Humidity:</strong> ${weather.humidity} %</p>
      <p><strong>Precipitation:</strong> ${weather.precipitation_mm.toFixed(1)} mm</p>
      <p><em><small>Source: ${weather.source_api} at ${new Date(weather.timestamp).toLocaleTimeString()}</small></em></p>
    </div>
  `).join('');
  dom.panels.current.innerHTML = `
    <h3>Current Weather in ${data.location.city_name}</h3>
    <div class="weather-cards-container">
      ${weatherHtml}
    </div>
  `;
}

export function renderDailyForecast(data: DailyForecastsResponse) {
  if (!data.forecasts || data.forecasts.length === 0) {
    throw new Error('No daily forecast data available for this location.');
  }
  const forecastsByDate = data.forecasts.reduce((acc: Record<string, DailyForecast[]>, forecast) => {
    const date = new Date(forecast.forecast_date).toISOString().split('T')[0]; // Use YYYY-MM-DD for reliable key
    if (!acc[date]) acc[date] = [];
    acc[date].push(forecast);
    return acc;
  }, {});

  const sortedDates = Object.keys(forecastsByDate).sort();

  dom.dailyElements.list.innerHTML = sortedDates.map((dateKey, index) => {
    const displayDate = formatDate(new Date(dateKey));
    return `<div class="forecast-list-item ${index === 0 ? 'active' : ''}" data-date-key="${dateKey}">${displayDate}</div>`;
  }).join('');

  const renderDetails = (dateKey: string) => {
    const forecasts = forecastsByDate[dateKey];
    const displayDate = formatDate(new Date(dateKey));
    dom.dailyElements.details.innerHTML = `
      <h3>Forecast for ${displayDate} in ${data.location.city_name}</h3>
      <div class="weather-cards-container">
        ${forecasts.map(f => `
          <div class="weather-card">
            <p><strong>Temp:</strong> ${f.min_temp_c.toFixed(1)} - ${f.max_temp_c.toFixed(1)} °C</p>
            <p><strong>Precipitation:</strong> ${f.precipitation_mm.toFixed(1)} mm (${f.precipitation_chance}%)</p>
            <p><strong>Wind:</strong> ${f.wind_speed_kmh.toFixed(1)} km/h</p>
            <p><strong>Humidity:</strong> ${f.humidity} %</p>
            <p><em><small>Source: ${f.source_api}</small></em></p>
          </div>
        `).join('')}
      </div>
    `;
  };

  dom.dailyElements.list.querySelectorAll('.forecast-list-item').forEach(item => {
    item.addEventListener('click', () => {
      dom.dailyElements.list.querySelectorAll('.forecast-list-item').forEach(i => i.classList.remove('active'));
      item.classList.add('active');
      renderDetails(item.getAttribute('data-date-key')!);
    });
  });

  renderDetails(sortedDates[0]);
}

export function renderHourlyForecast(data: HourlyForecastsResponse) {
  if (!data.forecasts || data.forecasts.length === 0) {
    throw new Error('No hourly forecast data available for this location.');
  }
  const forecastsByHour = data.forecasts.reduce((acc: Record<string, HourlyForecast[]>, forecast) => {
    const key = new Date(forecast.forecast_datetime).toISOString();
    if (!acc[key]) acc[key] = [];
    acc[key].push(forecast);
    return acc;
  }, {});

  const sortedHours = Object.keys(forecastsByHour).sort();

  dom.hourlyElements.list.innerHTML = sortedHours.map((hourKey, index) => {
    const date = new Date(hourKey);
    const displayTime = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    const displayDate = formatDate(date);
    return `
      <div class="forecast-list-item ${index === 0 ? 'active' : ''}" data-hour-key="${hourKey}">
        <span class="time">${displayTime}</span>
        <span class="date">${displayDate}</span>
      </div>
    `;
  }).join('');

  const renderDetails = (hourKey: string) => {
    const forecasts = forecastsByHour[hourKey];
    const date = new Date(hourKey);
    const displayHour = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    const displayDate = formatDate(date);
    dom.hourlyElements.details.innerHTML = `
      <h3>Forecast for ${displayDate} at ${displayHour} in ${data.location.city_name}</h3>
      <div class="weather-cards-container">
        ${forecasts.map(f => `
          <div class="weather-card">
            <p><strong>Temp:</strong> ${f.temperature_c.toFixed(1)} °C</p>
            <p><strong>Condition:</strong> ${f.condition_text}</p>
            <p><strong>Precipitation:</strong> ${f.precipitation_mm.toFixed(1)} mm (${f.precipitation_chance}%)</p>
            <p><strong>Wind:</strong> ${f.wind_speed_kmh.toFixed(1)} km/h</p>
            <p><strong>Humidity:</strong> ${f.humidity} %</p>
            <p><em><small>Source: ${f.source_api}</small></em></p>
          </div>
        `).join('')}
      </div>
    `;
  };

  dom.hourlyElements.list.querySelectorAll('.forecast-list-item').forEach(item => {
    item.addEventListener('click', () => {
      dom.hourlyElements.list.querySelectorAll('.forecast-list-item').forEach(i => i.classList.remove('active'));
      item.classList.add('active');
      renderDetails(item.getAttribute('data-hour-key')!);
    });
  });

  renderDetails(sortedHours[0]);
}
