package main

import (
	"log"

	"github.com/joho/godotenv"

	"business_services/internal/api"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("нет .env, используем переменные окружения")
	}
	api.StartServer()
}
