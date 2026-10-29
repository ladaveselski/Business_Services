package database

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=admin password=admin dbname=business_services port=5432 sslmode=disable"
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
