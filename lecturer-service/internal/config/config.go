package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	// TODO: think of a alternative solution
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "local"
	}
	envPath := fmt.Sprintf(".env.%s", env)
	err := godotenv.Load(envPath)
	if err != nil {
		err = godotenv.Load(fmt.Sprintf("./lecturer-service/%s", envPath))
		return nil
	}
	return nil
}
