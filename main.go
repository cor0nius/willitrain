package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	dbQueries        *database.Queries
	gmpGeocodeURL    string
	gmpWeatherURL    string
	owmWeatherURL    string
	ometeoWeatherURL string
	gmpKey           string
	owmKey           string
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Couldn't connect to database: %v", err)
	}
	dbQueries := database.New(db)

	gmpGeocodeURL := os.Getenv("GMP_GEOCODE_URL")
	if gmpGeocodeURL == "" {
		log.Fatal("GMP_GEOCODE_URL must be set")
	}

	gmpWeatherURL := os.Getenv("GMP_WEATHER_URL")
	if gmpWeatherURL == "" {
		log.Fatal("GMP_WEATHER_URL must be set")
	}

	owmWeatherURL := os.Getenv("OWM_WEATHER_URL")
	if owmWeatherURL == "" {
		log.Fatal("OWM_WEATHER_URL must be set")
	}

	ometeoWeatherURL := os.Getenv("OMETEO_WEATHER_URL")
	if ometeoWeatherURL == "" {
		log.Fatal("OMETEO_WEATHER_URL must be set")
	}

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		log.Fatal("Missing API Key for Google Maps Platform")
	}

	owmKey := os.Getenv("OWM_KEY")
	if owmKey == "" {
		log.Fatal("Missing API Key for OpenWeatherMap")
	}

	cfg := apiConfig{
		dbQueries:        dbQueries,
		gmpGeocodeURL:    gmpGeocodeURL,
		gmpWeatherURL:    gmpWeatherURL,
		owmWeatherURL:    owmWeatherURL,
		ometeoWeatherURL: ometeoWeatherURL,
		gmpKey:           gmpKey,
		owmKey:           owmKey,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/currentweather", cfg.handlerCurrentWeather)
	mux.HandleFunc("/dailyforecast", cfg.handlerDailyForecast)
	mux.HandleFunc("/hourlyforecast", cfg.handlerHourlyForecast)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
