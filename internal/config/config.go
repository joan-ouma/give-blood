package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	MongoURI      string
	JWTSecret     string
	AllowedOrigin string
	Port          string
}

func Load() (*Config, error) {
	mongoURI, ok1 := os.LookupEnv("MONGO_URI")
	jwtSecret, ok2 := os.LookupEnv("JWT_SECRET")
	allowedOrigin, ok3 := os.LookupEnv("ALLOWED_ORIGIN")
	port, ok4 := os.LookupEnv("PORT")

	if !ok1 || strings.TrimSpace(mongoURI) == "" {
		return nil, errors.New("missing MONGO_URI env var")
	}
	if !ok2 || strings.TrimSpace(jwtSecret) == "" {
		return nil, errors.New("missing JWT_SECRET env var")
	}
	if !ok3 || strings.TrimSpace(allowedOrigin) == "" {
		return nil, errors.New("missing ALLOWED_ORIGIN env var")
	}
	if !ok4 || strings.TrimSpace(port) == "" {
		return nil, errors.New("missing PORT env var")
	}

	return &Config{
		MongoURI:      strings.TrimSpace(mongoURI),
		JWTSecret:     strings.TrimSpace(jwtSecret),
		AllowedOrigin: strings.TrimSpace(allowedOrigin),
		Port:          strings.TrimSpace(port),
	}, nil
}
