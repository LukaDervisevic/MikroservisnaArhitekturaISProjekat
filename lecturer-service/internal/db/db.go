package db

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() *gorm.DB {

	username := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dbname := os.Getenv("POSTGRES_DB")

	dbUrl := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(username, password),
		Host:     fmt.Sprintf("%s:%s", host, port),
		RawQuery: "sslmode=disable",
		Path:     "/" + dbname,
	}

	db, err := gorm.Open(postgres.Open(dbUrl.String()), &gorm.Config{})
	if err != nil {
		log.Fatal("Unable to initialize database")
	}

	return db
}
