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
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Print("Couldn't connect to database: %v", err)
	}
	dbQueries := database.New(db)

	gmpKey := os.Getenv("GMP_KEY")
	if gmpKey == "" {
		log.Fatal("Missing API Key for Google Maps Platform")
	}

	cfg := apiConfig{
		dbQueries: dbQueries,
		gmpKey:    gmpKey,
	}
}
