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
	// A missing dotenv file is not fatal: in Docker, env vars are already
	// injected via env_file rather than present as a file on disk.
	if err := godotenv.Load(envPath); err != nil {
		_ = godotenv.Load(fmt.Sprintf("./lecture-service/%s", envPath))
	}
	return nil
}
