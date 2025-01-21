package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

var WebsiteAccess string = "http://localhost:3000"

func Load() (*Config, error) {
	// Charger le fichier .env, si présent
	godotenv.Load()

	cfg := &Config{
		DBUser: os.Getenv("DB_USER"),
		DBPass: os.Getenv("DB_PASS"),
		DBName: os.Getenv("DB_NAME"),
		DBHost: os.Getenv("DB_HOST"),
		DBPort: os.Getenv("DB_PORT"),
		DBSSL:  "disable",

		YouTubeAPIKey: os.Getenv("GOOGLE_API_KEY"),
		WebsiteAccess: WebsiteAccess,
	}

	if cfg.DBUser == "" || cfg.DBPass == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("missing DB config (DB_USER, DB_PASS, DB_NAME)")
	}
	if cfg.YouTubeAPIKey == "" {
		return nil, fmt.Errorf("missing YouTube API key (GOOGLE_API_KEY)")
	}

	return cfg, nil
}
