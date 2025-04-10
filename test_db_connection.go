package main

import (
	"fmt"
	"log"
	"url_shortener/database"
)

func main() {
	database.Connect()

	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Fatal("Failed to get database connection:", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Database connection successful!")
}
