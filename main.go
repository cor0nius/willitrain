package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/cor0nius/willitrain/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	dbQueries *database.Queries
	gmpKey    string
	owmKey    string
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

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		log.Fatal("Missing API Key for Google Maps Platform")
	}

	owmKey := os.Getenv("OWM_KEY")
	if owmKey == "" {
		log.Fatal("Missing API Key for OpenWeatherMap")
	}

	cfg := apiConfig{
		dbQueries: dbQueries,
		gmpKey:    gmpKey,
		owmKey:    owmKey,
	}

	log.Printf("Starting WillItRain API with config: %+v\n", cfg)
}
