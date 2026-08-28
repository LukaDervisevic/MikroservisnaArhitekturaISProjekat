package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	env := os.Getenv("ENVIRONMENT")
	if env == "" || env == "local" {
		env = "local"
	}
	envPath := fmt.Sprintf(".env.%s", env)
	if err := godotenv.Load(envPath); err != nil {
		_ = godotenv.Load(fmt.Sprintf("./event-service/%s", envPath))
	}
	return nil
}
