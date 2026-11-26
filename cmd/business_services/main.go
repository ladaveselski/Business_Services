package main

import (
	"log"

	"github.com/joho/godotenv"

	"business_services/internal/api"
	"business_services/internal/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("нет .env, используем переменные окружения")
	}

	dsn := "app.db"
	db, err := database.New(dsn)
	if err != nil {
		log.Fatalf("не удалось инициализировать БД: %v", err)
	}

	api.StartServer(db)
}
