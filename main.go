package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

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
