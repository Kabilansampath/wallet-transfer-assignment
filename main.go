package main

import (
	"fmt"
	"log"
	"os"
	"wallet-transfer/internal/models"
	"wallet-transfer/migrations"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {

	godotenv.Load()

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", os.Getenv("DB_HOST"), os.Getenv("DB_USERNAME"), os.Getenv("DB_NAME"), os.Getenv("DB_PASSWORD")) //Build connection string

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	db.AutoMigrate(
		&models.Wallet{},
		&models.Transfer{},
		&models.LedgerEntry{},
		&models.IdempotencyRecord{},
	)

	migrations.SeedWallets(db)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
