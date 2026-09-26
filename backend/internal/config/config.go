// Package config reads settings from environment variables.
package config

import (
	"fmt"
	"os"

	"restaurants/internal/auth"
	"restaurants/internal/storage"
)

type Config struct {
	HTTPAddr     string // HTTP_ADDR, default ":8080"
	DatabaseURL  string // DATABASE_URL, required
	CookieSecure bool   // COOKIE_SECURE, default true; set "false" for plain-http local dev
	S3           storage.S3Config
	ImageBaseURL string // IMAGE_BASE_URL, default "/images" (served by Garage's web endpoint, ADR-0005)
	// GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET / GOOGLE_REDIRECT_URL; Google login is off without an ID.
	Google auth.GoogleConfig
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:     getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CookieSecure: getenv("COOKIE_SECURE", "true") != "false",
		S3: storage.S3Config{
			Endpoint:        getenv("S3_ENDPOINT", "http://localhost:3900"),
			Region:          getenv("S3_REGION", "garage"),
			Bucket:          getenv("S3_BUCKET", "restaurant-images"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		},
		ImageBaseURL: getenv("IMAGE_BASE_URL", "/images"),
		Google: auth.GoogleConfig{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  getenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/auth/google/callback"),
		},
	}
	for name, v := range map[string]string{
		"DATABASE_URL":         c.DatabaseURL,
		"S3_ACCESS_KEY_ID":     c.S3.AccessKeyID,
		"S3_SECRET_ACCESS_KEY": c.S3.SecretAccessKey,
	} {
		if v == "" {
			return c, fmt.Errorf("%s is required", name)
		}
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
