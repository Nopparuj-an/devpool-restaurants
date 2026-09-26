// Package config reads settings from environment variables.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr    string // HTTP_ADDR, default ":8080"
	DatabaseURL string // DATABASE_URL, required
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
