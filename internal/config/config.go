package config

import (
	"os"

	"github.com/joho/godotenv"
)

// struct name capital so it is now public
type Config struct {
	// PORT string
	ENV          string
	DATABASE_URL string
}

func MustLoad() Config {
	godotenv.Load()
	// port := os.Getenv("PORT")
	// if port == "" {
	// 	panic("PORT is required")
	// }

	env := os.Getenv("ENV")
	if env == "" {
		panic("ENV is required")
	}

	database_url := os.Getenv("DATABASE_URL")
	if database_url == "" {
		panic("DATABASE URL is required")
	}

	return Config{
		// PORT: port,
		ENV:          env,
		DATABASE_URL: database_url,
	}
}

// Must is default use in go means if error is generated then it will panic here and application crashes
