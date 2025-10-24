package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/routes"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: unable to find .env file")
	}

	client := database.Connect()
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err := client.Ping(ctx, nil); err != nil {
		fmt.Println("Failed to ping MongoDB:", err)
		return
	}

	router := gin.Default()

	// CORS Configuration
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	var origins []string

	if allowedOrigins != "" {
		origins = strings.Split(allowedOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
			log.Println("Allowed Origin:", origins[i])
		}
	} else {
		origins = []string{
			"http://localhost:5173", // Vite
			"http://localhost:5174",
			"http://localhost:3000", // React
			"http://localhost:8081",
		}
		log.Println("Using default allowed origins for development")
	}

	corsConfig := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	router.Use(cors.New(corsConfig))
	router.Use(gin.Logger())

	router.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, MagicStreamMoviesServer!")
	})

	routes.SetupProtectedRoutes(router, client)
	routes.SetupUnProtectedRoutes(router, client)

	fmt.Println("Server starting on :8080")
	if err := router.Run(":8080"); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
