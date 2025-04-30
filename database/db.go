package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	host := getEnv("DB_HOST", "127.0.0.1")
	user := getEnv("DB_USER", "suricat")
	password := getEnv("DB_PASSWORD", "111222333")
	dbname := getEnv("DB_NAME", "urlshortener")
	port := getEnv("DB_PORT", "5454")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port,
	)

	var err error
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Failed to connect (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	log.Println("Database connection established")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
