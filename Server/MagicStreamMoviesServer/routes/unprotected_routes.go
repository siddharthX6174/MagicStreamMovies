package routes

import (
	"github.com/gin-gonic/gin"
	controller "github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/controllers"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupUnProtectedRoutes(router *gin.Engine, client *mongo.Client) {
	
	// Public routes (no authentication)
	router.GET("/movies", controller.GetMovies())
	router.POST("/register", controller.RegisterUser())
	router.POST("/login", controller.LoginUser(client))
	router.POST("/logout", controller.LogoutHandler(client))
	router.GET("/genres", controller.GetGenres(client))
	router.POST("/refresh", controller.RefreshTokenHandler(client))
}