package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func init() {
	// Connect to Redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerUserHandler(c *gin.Context) {
	// Parse request body
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	// Check if user already exists
	exists, err := rdb.Exists(ctx, user.Username).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if exists != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	// Initialize user data including the score
	userData := map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
		"password": user.Password,
		"score":    0, // Initialize score to 0
	}

	// Serialize user data to JSON
	userJSON, err := json.Marshal(userData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Store user data in Redis
	if err := rdb.Set(ctx, user.Username, userJSON, 0).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Send success response
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func leaderboard(c *gin.Context) {
    // Get the username of the logged-in user from the request or session
    username := c.GetString("username") // Assuming you store the username in the context

    // Retrieve the score for the logged-in user from Redis
    score, err := rdb.Get(ctx, username).Int()
    if err != nil {
        // Handle missing key or score
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve score for user: " + username})
        return
    }

    // Create a leaderboard map with the logged-in user's score
    leaderboard := map[string]int{username: score}

    // Log leaderboard data for debugging
    fmt.Println("Leaderboard:", leaderboard)

    // Return leaderboard data
    c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
}

func gameWon(c *gin.Context) {
    // Get the username from the request
    username := c.PostForm("username")

    // Check if the key exists
    exists, err := rdb.Exists(ctx, username).Result()
    if err != nil {
        // Handle error
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check key existence in Redis"})
        return
    }

    // If the key doesn't exist, set initial value to 0
    if exists == 0 {
        err := rdb.Set(ctx, username, 0, 0).Err()
        if err != nil {
            // Handle error
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set initial score in Redis"})
            return
        }
    }

    // Increment the score by one
    err = rdb.Incr(ctx, username).Err()
    if err != nil {
        // Handle error
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment score in Redis"})
        return
    }

    // Get the updated score
    score, err := rdb.Get(ctx, username).Int()
    if err != nil {
        // Handle error
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated score from Redis"})
        return
    }

    // Return success response with the updated score
    fmt.Println("Game won successfully", username, "Score:", score)
    c.JSON(http.StatusOK, gin.H{"message": "Game won successfully", "username": username, "score": score})    
}


func main() {
	fmt.Println("Starting server...")

	r := gin.Default()
	r.Use(cors.Default())

	r.POST("/register", registerUserHandler)
	r.GET("/leaderboard/:username", leaderboard)
	r.POST("/game/won", gameWon)

	if err := r.Run(":8080"); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
