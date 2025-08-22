import type { CurrentWeatherResponse, DailyForecastsResponse, HourlyForecastsResponse, DailyForecast, HourlyForecast } from './types';

// --- DOM Element References ---
export const dom = {
  locationInput: document.querySelector<HTMLInputElement>('#location-input')!,
  getWeatherBtn: document.querySelector<HTMLButtonElement>('#get-weather-btn')!,
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
  dom.panels[panel].innerHTML = 'Loading...';
}

export function showError(panel: keyof typeof dom.panels, error: Error) {
  dom.panels[panel].innerHTML = `<h3>Error</h3><p>${error.message}</p>`;
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
    const date = new Date(forecast.forecast_date).toLocaleDateString();
    if (!acc[date]) acc[date] = [];
    acc[date].push(forecast);
    return acc;
  }, {});

  dom.dailyElements.list.innerHTML = Object.keys(forecastsByDate).map((date, index) => `
    <div class="forecast-list-item ${index === 0 ? 'active' : ''}" data-date="${date}">${date}</div>
  `).join('');

  const renderDetails = (date: string) => {
    const forecasts = forecastsByDate[date];
    dom.dailyElements.details.innerHTML = `
      <h3>Forecast for ${date}</h3>
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
      renderDetails(item.getAttribute('data-date')!);
    });
  });

  renderDetails(Object.keys(forecastsByDate)[0]);
}

export function renderHourlyForecast(data: HourlyForecastsResponse) {
  if (!data.forecasts || data.forecasts.length === 0) {
    throw new Error('No hourly forecast data available for this location.');
  }
  const forecastsByHour = data.forecasts.reduce((acc: Record<string, HourlyForecast[]>, forecast) => {
    const hour = new Date(forecast.forecast_datetime).toLocaleString();
    if (!acc[hour]) acc[hour] = [];
    acc[hour].push(forecast);
    return acc;
  }, {});

  dom.hourlyElements.list.innerHTML = Object.keys(forecastsByHour).map((hour, index) => `
    <div class="forecast-list-item ${index === 0 ? 'active' : ''}" data-hour="${hour}">${hour}</div>
  `).join('');

  const renderDetails = (hour: string) => {
    const forecasts = forecastsByHour[hour];
    dom.hourlyElements.details.innerHTML = `
      <h3>Forecast for ${hour}</h3>
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
      renderDetails(item.getAttribute('data-hour')!);
    });
  });

  renderDetails(Object.keys(forecastsByHour)[0]);
}
