package db

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func Connect() *gorm.DB {

	username := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dbname := os.Getenv("POSTGRES_DB")

	var sslMode string
	var err error
	if os.Getenv("ENVIRONMENT") == "local" {
		sslMode = "sslmode=disable"
	} else {
		sslMode = "sslmode=enable"
	}

	dbUrl := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(username, password),
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     "/" + dbname,
		RawQuery: sslMode,
	}

	var m *migrate.Migrate
	for i := 0; i < 10; i++ {
		m, err = migrate.New("file://internal/migrations", dbUrl.String())
		if err == nil {
			break
		}
		log.Info().Err(err).Msgf("Attempt %d: DB not ready, retrying...", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize migration engine")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("Migrations failed.")
	}

	db, err := gorm.Open(postgres.Open(dbUrl.String()), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "lecturer_service.",
			SingularTable: false,
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	} else {
		log.Info().Msg("successfully connected to the postgres database")
	}

	return db
}
