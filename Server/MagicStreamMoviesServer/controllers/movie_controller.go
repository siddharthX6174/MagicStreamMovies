package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	database "github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	models "github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var validate = validator.New()

// Create client and collection
var client *mongo.Client = database.Connect()
var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

func GetMovies() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var movies []models.Movie

		cursor, err := movieCollection.Find(ctx, bson.M{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching movies"})
			return
		}
		defer cursor.Close(ctx)

		if err = cursor.All(ctx, &movies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movies"})
			return
		}
		c.JSON(http.StatusOK, movies)
	}
}

//--------------------------------------------------------------------------------------------
func GetMovieByID() gin.HandlerFunc {
	return func(c *gin.Context){
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		movieID := c.Param("imdb_id")

		if movieID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Movie ID is required"})
			return
		}

		var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

		var movie models.Movie

		err := movieCollection.FindOne(ctx, bson.D{{Key: "imdb_id", Value: movieID}}).Decode(&movie)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}

		c.JSON(http.StatusOK, movie)
	}
}
//--------------------------------------------------------------------------------------------
// post request to add movie
func AddMovie(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, 100*time.Second)
		defer cancel()

		var movie models.Movie
		if err := c.ShouldBindJSON(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		if err := validate.Struct(movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
			return
		}
		var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

		result, err := movieCollection.InsertOne(ctx, movie)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add movie"})
			return
		}
		c.JSON(http.StatusCreated, result)
	}
}

//--------------------------------------------------------------------------------------------
// Get recommended movies based on user's favorite genres
func GetRecommendedMovies(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Get user ID from middleware
		userID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
			return
		}

		// Get user collection to fetch user's favorite genres
		var userCollection *mongo.Collection = database.OpenCollection("users", client)
		var user models.User

		err := userCollection.FindOne(ctx, bson.D{{Key: "user_id", Value: userID}}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Extract favorite genre IDs
		var favoriteGenreIDs []int
		for _, genre := range user.FavouriteGenres {
			favoriteGenreIDs = append(favoriteGenreIDs, genre.GenreID)
		}

		if len(favoriteGenreIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No favorite genres found for user"})
			return
		}

		// Build query to find movies with matching genres
		// Using $elemMatch to match any genre in the user's favorite genres
		genreMatch := bson.M{
			"genre": bson.M{
				"$elemMatch": bson.M{
					"genre_id": bson.M{
						"$in": favoriteGenreIDs,
					},
				},
			},
		}

		// Find movies that match user's favorite genres
		cursor, err := movieCollection.Find(ctx, genreMatch)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching recommended movies"})
			return
		}
		defer cursor.Close(ctx)

		var recommendedMovies []models.Movie
		if err = cursor.All(ctx, &recommendedMovies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode recommended movies"})
			return
		}

		// If no movies found, return popular movies as fallback
		if len(recommendedMovies) == 0 {
			// Get movies with high ranking as fallback
			fallbackQuery := bson.M{
				"ranking.ranking_value": bson.M{
					"$gte": 7, // Movies with rating 7 or above
				},
			}
			
			cursor, err = movieCollection.Find(ctx, fallbackQuery)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching fallback movies"})
				return
			}
			defer cursor.Close(ctx)

			if err = cursor.All(ctx, &recommendedMovies); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode fallback movies"})
				return
			}
		}

		// Limit results to 20 movies for better performance
		if len(recommendedMovies) > 20 {
			recommendedMovies = recommendedMovies[:20]
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Recommended movies retrieved successfully",
			"count":   len(recommendedMovies),
			"movies":  recommendedMovies,
		})
	}
}

//--------------------------------------------------------------------------------------------
// Update admin review for a specific movie (Admin only)
func AdminReviewUpdate(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Get user role from middleware to check if user is admin
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			return
		}

		// Check if user is admin
		if userRole != "ADMIN" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. Admin privileges required"})
			return
		}

		// Get movie ID from URL parameter
		movieID := c.Param("imdb_id")
		if movieID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Movie ID is required"})
			return
		}

		// Define request structure for admin review update
		var updateRequest struct {
			AdminReview string `json:"admin_review" validate:"required,min=10,max=1000"`
		}

		// Bind JSON request body
		if err := c.ShouldBindJSON(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format"})
			return
		}

		// Validate the admin review
		if err := validate.Struct(updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
			return
		}

		// Get movie collection
		var movieCollection *mongo.Collection = database.OpenCollection("movies", client)

		// Check if movie exists
		var existingMovie models.Movie
		err := movieCollection.FindOne(ctx, bson.D{{Key: "imdb_id", Value: movieID}}).Decode(&existingMovie)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}

		// Update the admin review
		filter := bson.D{{Key: "imdb_id", Value: movieID}}
		update := bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "admin_review", Value: updateRequest.AdminReview},
			}},
		}

		result, err := movieCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movie review"})
			return
		}

		// Check if any document was modified
		if result.ModifiedCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No changes made to the movie review"})
			return
		}

		// Fetch the updated movie to return
		var updatedMovie models.Movie
		err = movieCollection.FindOne(ctx, filter).Decode(&updatedMovie)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated movie"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Movie review updated successfully",
			"movie":   updatedMovie,
		})
	}
}

//--------------------------------------------------------------------------------------------
// Get all available genres
func GetGenres(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Get genres collection
		var genresCollection *mongo.Collection = database.OpenCollection("genres", client)

		var genres []models.Genre

		cursor, err := genresCollection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching genres"})
			return
		}
		defer cursor.Close(ctx)

		if err = cursor.All(ctx, &genres); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode genres"})
			return
		}

		// If no genres found in database, return default genres
		if len(genres) == 0 {
			defaultGenres := []models.Genre{
				{GenreID: 1, GenreName: "Comedy"},
				{GenreID: 2, GenreName: "Drama"},
				{GenreID: 3, GenreName: "Western"},
				{GenreID: 4, GenreName: "Fantasy"},
				{GenreID: 5, GenreName: "Thriller"},
				{GenreID: 6, GenreName: "Sci-Fi"},
				{GenreID: 7, GenreName: "Action"},
				{GenreID: 8, GenreName: "Mystery"},
				{GenreID: 9, GenreName: "Crime"},
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "Genres retrieved successfully",
				"count":   len(defaultGenres),
				"genres":  defaultGenres,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Genres retrieved successfully",
			"count":   len(genres),
			"genres":  genres,
		})
	}
}
