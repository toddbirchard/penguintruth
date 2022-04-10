package main_test

import (
	"github.com/joho/godotenv"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	// Attempt to load `.env` file.
	err := godotenv.Load()
	if err != nil {
		t.Errorf("Failed to load environment variables from .env file.")
	}

	// Attempt to read config value for `WEBSERVER_PORT`.
	addressPort := os.Getenv("WEBSERVER_PORT")
	if addressPort == "" {
		t.Errorf("Could not fetch `WEBSERVER_PORT` envvar")
	}
}
