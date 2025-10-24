package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	model "github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"github.com/siddharthX6174/MagicStreamMovies/Server/MagicStreamMoviesServer/utils"
)

var userCollection *mongo.Collection = database.OpenCollection("users", client)

func HashPassword(password string) (string, error) {
	HashPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(HashPassword), nil

}

func RegisterUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user model.User

		if err := c.ShouldBind(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input Data"})
			return
		}
		validate := validator.New()
		if err := validate.Struct(user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
			return
		}

		hashedPassword, err := HashPassword(user.Password)
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing user"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}

		user.UserID = primitive.NewObjectID().Hex()
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		user.Password = hashedPassword

		result, err := userCollection.InsertOne(ctx, user)
		
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		c.JSON(http.StatusCreated, result)
	}
}

func LoginUser(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userLogin model.UserLogin

		if err := c.ShouldBindJSON(&userLogin); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalide input data"})
			return
		}

		var ctx, cancel = context.WithTimeout(c, 100*time.Second)
		defer cancel()

		var userCollection *mongo.Collection = database.OpenCollection("users", client)

		var foundUser model.User
		err := userCollection.FindOne(ctx, bson.D{{Key: "email", Value: userLogin.Email}}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(userLogin.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		token, refreshToken, err := utils.GenerateAllTokens(foundUser.Email, foundUser.FirstName, foundUser.LastName, foundUser.Role, foundUser.UserID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
			return
		}

		err = utils.UpdateAllTokens(foundUser.UserID, token, refreshToken, client)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tokens"})
			return
		}
		// http.SetCookie(c.Writer, &http.Cookie{
		// 	Name:  "access_token",
		// 	Value: token,
		// 	Path:  "/",
		// 	// Domain:   "localhost",
		// 	MaxAge:   86400,
		// 	Secure:   true,
		// 	HttpOnly: true,
		// 	SameSite: http.SameSiteNoneMode,
		// })
		// http.SetCookie(c.Writer, &http.Cookie{
		// 	Name:  "refresh_token",
		// 	Value: refreshToken,
		// 	Path:  "/",
		// 	// Domain:   "localhost",
		// 	MaxAge:   604800,
		// 	Secure:   true,
		// 	HttpOnly: true,
		// 	SameSite: http.SameSiteNoneMode,
		// })

		c.JSON(http.StatusOK, model.UserResponse{
			UserId:    foundUser.UserID,
			FirstName: foundUser.FirstName,
			LastName:  foundUser.LastName,
			Email:     foundUser.Email,
			Role:      foundUser.Role,
			Token:           token,
			RefreshToken:    refreshToken,
			FavouriteGenres: foundUser.FavouriteGenres,
		})

	}
}

//--------------------------------------------------------------------------------------------
// Logout user by clearing tokens from database
func LogoutHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Get user ID from the request (could be from token or request body)
		var logoutRequest struct {
			UserID string `json:"user_id" validate:"required"`
		}

		// Try to bind JSON first
		if err := c.ShouldBindJSON(&logoutRequest); err != nil {
			// If JSON binding fails, try to get user_id from query parameter
			userID := c.Query("user_id")
			if userID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
				return
			}
			logoutRequest.UserID = userID
		}

		// Validate user ID
		validate := validator.New()
		if err := validate.Struct(logoutRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID", "details": err.Error()})
			return
		}

		// Get user collection
		var userCollection *mongo.Collection = database.OpenCollection("users", client)

		// Check if user exists
		var existingUser model.User
		err := userCollection.FindOne(ctx, bson.D{{Key: "user_id", Value: logoutRequest.UserID}}).Decode(&existingUser)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Clear tokens by setting them to empty strings
		filter := bson.D{{Key: "user_id", Value: logoutRequest.UserID}}
		update := bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "token", Value: ""},
				{Key: "refresh_token", Value: ""},
				{Key: "update_at", Value: time.Now()},
			}},
		}

		result, err := userCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout user"})
			return
		}

		// Check if any document was modified
		if result.ModifiedCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No changes made during logout"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "User logged out successfully",
			"user_id": logoutRequest.UserID,
		})
	}
}

//--------------------------------------------------------------------------------------------
// Refresh access token using refresh token
func RefreshTokenHandler(client *mongo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Define request structure for refresh token
		var refreshRequest struct {
			RefreshToken string `json:"refresh_token" validate:"required"`
		}

		// Bind JSON request body
		if err := c.ShouldBindJSON(&refreshRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format"})
			return
		}

		// Validate the refresh token
		validate := validator.New()
		if err := validate.Struct(refreshRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token is required", "details": err.Error()})
			return
		}

		// Validate the refresh token
		claims, err := utils.ValidateRefreshToken(refreshRequest.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
			return
		}

		// Get user collection
		var userCollection *mongo.Collection = database.OpenCollection("users", client)

		// Check if user exists and refresh token matches
		var foundUser model.User
		err = userCollection.FindOne(ctx, bson.D{
			{Key: "user_id", Value: claims.UserID},
			{Key: "refresh_token", Value: refreshRequest.RefreshToken},
		}).Decode(&foundUser)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token or user not found"})
			return
		}

		// Generate new tokens
		newToken, newRefreshToken, err := utils.GenerateAllTokens(
			foundUser.Email,
			foundUser.FirstName,
			foundUser.LastName,
			foundUser.Role,
			foundUser.UserID,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new tokens"})
			return
		}

		// Update tokens in database
		err = utils.UpdateAllTokens(foundUser.UserID, newToken, newRefreshToken, client)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tokens"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Tokens refreshed successfully",
			"token":   newToken,
			"refresh_token": newRefreshToken,
		})
	}
}
