import './style.css'

const locationInput = document.querySelector<HTMLInputElement>('#location-input')!;
const getCurrentWeatherBtn = document.querySelector<HTMLButtonElement>('#get-current-weather-btn')!;
const currentWeatherDiv = document.querySelector<HTMLDivElement>('#current-weather-div')!;

getCurrentWeatherBtn.addEventListener('click', async () => {
  const location = locationInput.value.trim();
  if (!location) {
    currentWeatherDiv.innerHTML = 'Please enter a location.';
    return;
  }

  currentWeatherDiv.innerHTML = 'Loading...';

  try {
    const response = await fetch(`https://willitrain-908739103426.europe-west1.run.app/api/currentweather?city=${encodeURIComponent(location)}`);
    
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
    }

    const data = await response.json();
    
    // Assuming the first weather entry is the most relevant
    const weather = data.weather[0]; 
    if (!weather) {
      throw new Error('No weather data available for this location.');
    }

    currentWeatherDiv.innerHTML = `
      <h3>Current Weather in ${data.location.city_name}</h3>
      <p><strong>Temperature:</strong> ${weather.temperature_c.toFixed(1)} °C</p>
      <p><strong>Condition:</strong> ${weather.condition_text}</p>
      <p><strong>Wind:</strong> ${weather.wind_speed_kmh.toFixed(1)} km/h</p>
      <p><em><small>Source: ${weather.source_api} at ${new Date(weather.timestamp).toLocaleTimeString()}</small></em></p>
    `;

  } catch (error) {
    currentWeatherDiv.innerHTML = `
      <h3>Error</h3>
      <p>${error instanceof Error ? error.message : 'An unknown error occurred.'}</p>
    `;
  }
});