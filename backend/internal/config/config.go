package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr        string
	DatabaseURL string

	CookieSecure bool
	TrustProxy   bool

	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
}

func Load() (Config, error) {
	c := Config{
		Addr:         env("ADDR", ":8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CookieSecure: os.Getenv("COOKIE_SECURE") != "0",
		TrustProxy:   os.Getenv("TRUST_PROXY") == "1",
		S3Endpoint:   env("S3_ENDPOINT", "http://localhost:8333"),
		S3Region:     env("S3_REGION", "us-east-1"),
		S3Bucket:     env("S3_BUCKET", "itassist-attachments"),
		S3AccessKey:  os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:  os.Getenv("S3_SECRET_KEY"),
	}
	for name, v := range map[string]string{
		"DATABASE_URL": c.DatabaseURL, "S3_ACCESS_KEY": c.S3AccessKey, "S3_SECRET_KEY": c.S3SecretKey,
	} {
		if v == "" {
			return c, fmt.Errorf("missing required environment variable %s", name)
		}
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
