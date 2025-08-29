# Will It Rain?

## Overview

Will It Rain? is a weather forecast application designed to provide more reliable predictions by aggregating and comparing data from multiple sources. Instead of relying on a single forecast, which can sometimes be misleading, this application fetches weather information from Google Weather, OpenWeatherMap, and Open-Meteo. By presenting a consolidated view, it helps users make a more informed decision—if the forecasts align, the prediction is likely accurate; if they conflict, it's best to be prepared for anything.

This application is built with a Go backend, a lightweight TypeScript frontend, and is fully containerized for easy deployment.

## Live Demo

A version of this application is deployed on Google Cloud Run and is available here:
[https://willitrain-908739103426.europe-west1.run.app/](https://willitrain-908739103426.europe-west1.run.app/)

## Features

-   **Multi-Source Weather Data:** Aggregates current, hourly, and daily forecasts from three different weather APIs.
-   **Forecast Comparison:** (Future goal) A simple UI to visually compare the forecasts and identify consensus or discrepancies.
-   **REST API:** A clean API to access the aggregated weather data.
-   **Metrics:** Exposes application metrics in Prometheus format.
-   **Containerized:** Ships with a `docker-compose.yaml` for easy setup and deployment.

## Getting Started

You can run the application using Docker Compose (recommended) or by setting up the backend and frontend manually.

### Prerequisites

-   [Go](https://go.dev/dl/) (version 1.24.4 or newer)
-   [Node.js](https://nodejs.org/en/download) (for the frontend)
-   [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
-   API keys for [Google Maps Platform](https://developers.google.com/maps/documentation/geocoding/get-api-key) and [OpenWeatherMap](https://openweathermap.org/api).

### Installation & Running with Docker

This is the simplest way to get the entire application stack running.

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/cor0nius/willitrain.git
    cd willitrain
    ```

2.  **Create an environment file:**
    Create a file named `docker.env` in the root directory. This file is used to configure the application, including database connections, API keys, and other settings. Below is a table explaining all the possible variables.

    | Variable               | Description                                                              | Example                                                              |
    | ---------------------- | ------------------------------------------------------------------------ | -------------------------------------------------------------------- |
    | `POSTGRES_USER`        | **Required.** The username for the PostgreSQL database.                  | `user`                                                               |
    | `POSTGRES_PASSWORD`    | **Required.** The password for the PostgreSQL database.                  | `password`                                                           |
    | `POSTGRES_DB`          | **Required.** The name of the PostgreSQL database.                       | `willitrain`                                                         |
    | `DB_URL`               | **Required.** The connection string for the PostgreSQL database.         | `postgres://user:password@postgres:5432/willitrain?sslmode=disable`  |
    | `REDIS_URL`            | **Required.** The connection string for the Redis instance.              | `redis://redis:6379/`                                                |
    | `GMP_KEY`              | **Required.** Your API key for the Google Maps Platform.                 | `your_google_maps_platform_api_key`                                  |
    | `OWM_KEY`              | **Required.** Your API key for OpenWeatherMap.                           | `your_openweathermap_api_key`                                        |
    | `GMP_GEOCODE_URL`      | **Required.** The base URL for the Google Geocoding API.                 | `https://maps.googleapis.com/maps/api/geocode/`                      |
    | `GMP_WEATHER_URL`      | **Required.** The base URL for the Google Weather API.                   | `https://weather.googleapis.com/v1/`                                 |
    | `OWM_WEATHER_URL`      | **Required.** The base URL for the OpenWeatherMap API.                   | `https://api.openweathermap.org/data/3.0/onecall?`                   |
    | `OMETEO_WEATHER_URL`   | **Required.** The base URL for the Open-Meteo API.                       | `https://api.open-meteo.com/v1/forecast?`                            |
    | `CURRENT_INTERVAL_MIN` | The interval (in minutes) for fetching current weather data.             | `10`                                                                 |
    | `HOURLY_INTERVAL_MIN`  | The interval (in minutes) for fetching hourly forecast data.             | `60`                                                                 |
    | `DAILY_INTERVAL_MIN`   | The interval (in minutes) for fetching daily forecast data.              | `720`                                                                |
    | `DEV_MODE`             | Set to `1` to enable development-only endpoints.                         | `1`                                                                  |

    *Note: Open-Meteo does not require an API key for the free tier.*

3.  **Run with Docker Compose:**
    ```sh
    docker-compose up --build
    ```

The application will be available at `http://localhost:8080`.

### Manual Installation

Follow these steps if you prefer to run the backend and frontend services separately.

1.  **Backend (Go):**
    -   Navigate to the root directory.
    -   Set the required environment variables:
        ```sh
        export GMP_KEY="your_google_maps_platform_api_key"
        export OWM_KEY="your_openweathermap_api_key"
        export DB_URL="postgres://user:password@localhost:5432/willitrain?sslmode=disable"
        ```
    -   Run the database and Redis using Docker:
        ```sh
        docker-compose up -d postgres redis
        ```
    -   Run the database migrations:
        ```sh
        go install github.com/pressly/goose/v3/cmd/goose@latest
        goose -dir ./sql/schema postgres $DB_URL up
        ```
    -   Start the backend server:
        ```sh
        go run main.go
        ```

2.  **Frontend (TypeScript):**
    -   Navigate to the `frontend` directory:
        ```sh
        cd frontend
        ```
    -   Install dependencies:
        ```sh
        npm install
        ```
    -   Start the development server:
        ```sh
        npm run dev
        ```

## API Endpoints

The backend exposes the following REST API endpoints:

| Method | Endpoint                 | Description                                                            |
|--------|--------------------------|------------------------------------------------------------------------|
| `GET`  | `/api/config`            | Returns the client-side configuration.                                 |
| `GET`  | `/api/currentweather`    | Returns aggregated current weather data.                               |
| `GET`  | `/api/dailyforecast`     | Returns aggregated daily forecast data for 7 days.                     |
| `GET`  | `/api/hourlyforecast`    | Returns aggregated hourly forecast data for 24 hours.                  |
| `GET`  | `/metrics`               | Exposes application metrics for Prometheus.                            |
| `POST` | `/dev/reset-db`          | **(Dev Only)** Resets the database to its initial state.               |
| `POST` | `/dev/runschedulerjobs`  | **(Dev Only)** Manually triggers the scheduler to run all update jobs. |

**Example Usage:**
```sh
curl "http://localhost:8080/api/currentweather?location=London"
```

## Monitoring

The application is designed for robust monitoring in a cloud environment. This is handled by a separate, dedicated scraper service located in the `internal/scraper` directory.

### Scraper Service

-   **Purpose:** The scraper is a small, standalone Go service whose sole responsibility is to periodically fetch metrics from the main application's `/metrics` endpoint.
-   **Architecture:** It is designed to be deployed as a separate, serverless container on Google Cloud Run. It is a cloud-only utility and is **not** part of the local `docker-compose` setup.
-   **Execution:** The scraper is triggered on a schedule by Google Cloud Scheduler.
-   **Functionality:** After scraping the metrics, it converts them into the appropriate format and ingests them into Google Cloud's Managed Service for Prometheus, where they can be queried and visualized (e.g., with Grafana).
-   **CI/CD:** The scraper has its own independent deployment pipeline defined in `.github/workflows/scraper-cd.yaml`, which is triggered only when changes are made to the scraper's code.

## Running Tests

To run the test suite for the Go backend, execute the following command from the root directory:

```sh
go test ./...
```

## Built With

-   **Backend:** [Go](https://go.dev/), [PostgreSQL](https://www.postgresql.org/), [Redis](https://redis.io/)
-   **Frontend:** [TypeScript](https://www.typescriptlang.org/), [Vite](https://vitejs.dev/)
-   **DevOps:** [Docker](https://www.docker.com/), [Prometheus](https://prometheus.io/)
-   **Database Migrations:** [Goose](https://github.com/pressly/goose)
