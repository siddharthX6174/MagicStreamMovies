package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	controllers "github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/controllers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// create a context with timeout for MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		fmt.Println("Failed to connect to MongoDB:", err)
		return
	}
	// ensure disconnect when main exits
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	// verify connection
	if err := client.Ping(ctx, nil); err != nil {
		fmt.Println("Failed to ping MongoDB:", err)
		return
	}

	router := gin.Default()

	router.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, MagicStreamMoviesServer!") 
	})


	router.GET("/movies", controllers.GetMovies())
	router.GET("/movie/:imdb_id", controllers.GetMovieByID())
	router.POST("/addmovie", controllers.AddMovie(client))
	router.POST("/register", controllers.RegisterUser())
	router.POST("/login", controllers.LoginUser(client))

	if err := router.Run(":8080"); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
