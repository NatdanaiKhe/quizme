package config

import "os"

type Config struct {
	DatabaseURL   string
	AIAPIKey      string
	InternalToken string
	WebhookURL    string
	Port          string
}

func Load() Config {
	return Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		AIAPIKey:      os.Getenv("AI_API_KEY"),
		InternalToken: os.Getenv("INTERNAL_TOKEN"),
		WebhookURL:    os.Getenv("WEBHOOK_URL"),
		Port:          defaultString(os.Getenv("PORT"), "8080"),
	}
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
