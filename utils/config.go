package utils

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	DomainName          string
	OpenAIAPIKey        string
	GHUsername          string
	GHRepo              string
	GHPAT               string
	StripeSecretKey     string
	StripeWebhookSecret string
	StripePriceID       string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	return &Config{
		DomainName:          os.Getenv("DOMAIN_NAME"),
		OpenAIAPIKey:        os.Getenv("OPENAI_API_KEY"),
		GHUsername:          os.Getenv("GH_USERNAME"),
		GHRepo:              os.Getenv("GH_REPO"),
		GHPAT:               os.Getenv("GH_PAT"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePriceID:       os.Getenv("STRIPE_PRICE_ID"),
	}
}
