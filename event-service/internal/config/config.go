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
	err := godotenv.Load(envPath)
	if err != nil {
		err = godotenv.Load(fmt.Sprintf("./event-service/%s", envPath))
		if err != nil {
			return err
		}
	}
	return nil
}
