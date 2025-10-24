package routes

import (
	"github.com/gin-gonic/gin"
	controllers "github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/controllers"
	"github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/middleware"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupProtectedRoutes(router *gin.Engine, client *mongo.Client) {
	// Apply auth middleware to protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleWare())
	{
		protected.GET("/movie/:imdb_id", controllers.GetMovieByID())
		protected.POST("/addmovie", controllers.AddMovie(client))
		protected.GET("/recommendedmovies", controllers.GetRecommendedMovies(client))
		protected.PATCH("/updatereview/:imdb_id", controllers.AdminReviewUpdate(client))
	}
}