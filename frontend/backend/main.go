package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Poll struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Question  string        `json:"question" bson:"question"`
	Options   []string      `json:"options" bson:"options"`
	Votes     []int         `json:"votes" bson:"votes"`
	CreatedAt time.Time     `json:"createdAt" bson:"createdAt"`
}

type VoteRequest struct {
	Option int `json:"option"`
}

var (
	polls   = make(map[string]*Poll)
	pollsMu sync.RWMutex
)

func main() {

	// Load .env
	err := godotenv.Load()
	if err != nil {
		println("Warning: .env file not found")
	}

	// MongoDB
	mongoURI := os.Getenv("MONGODB_URI")

	var collection *mongo.Collection

	if mongoURI != "" {

		client, err := mongo.Connect(
			options.Client().ApplyURI(mongoURI),
		)

		if err != nil {
			println("MongoDB connection failed.")
			println("Running in temporary memory mode.")
		} else {

			ctx, cancel := context.WithTimeout(
				context.Background(),
				10*time.Second,
			)
			defer cancel()

			err = client.Ping(ctx, nil)

			if err != nil {
				println("MongoDB authentication failed.")
				println("Running in temporary memory mode.")
			} else {

				println("MongoDB connected successfully!")

				collection = client.
					Database("livepoll").
					Collection("polls")
			}
		}
	}

	router := gin.Default()

	// -----------------------------------------
	// CORS
	// -----------------------------------------

	router.Use(func(c *gin.Context) {

		c.Writer.Header().Set(
			"Access-Control-Allow-Origin",
			"*",
		)

		c.Writer.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, DELETE, OPTIONS",
		)

		c.Writer.Header().Set(
			"Access-Control-Allow-Headers",
			"Origin, Content-Type, Accept",
		)

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// -----------------------------------------
	// HEALTH CHECK
	// -----------------------------------------

	router.GET("/api/health", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{
			"message": "LivePoll backend is running!",
		})
	})

	// -----------------------------------------
	// CREATE POLL
	// -----------------------------------------

	router.POST("/api/polls", func(c *gin.Context) {

		var request struct {
			Question string   `json:"question"`
			Options  []string `json:"options"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid poll data",
			})

			return
		}

		if request.Question == "" || len(request.Options) < 2 {

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Question and at least two options are required",
			})

			return
		}

		votes := make([]int, len(request.Options))

		poll := &Poll{
			ID:        bson.NewObjectID(),
			Question:  request.Question,
			Options:   request.Options,
			Votes:     votes,
			CreatedAt: time.Now(),
		}

		// Save in memory
		pollsMu.Lock()
		polls[poll.ID.Hex()] = poll
		pollsMu.Unlock()

		// Save in MongoDB if connected
		if collection != nil {

			ctx, cancel := context.WithTimeout(
				context.Background(),
				10*time.Second,
			)
			defer cancel()

			_, err := collection.InsertOne(ctx, poll)

			if err != nil {
				println("MongoDB save failed")
			}
		}

		// Create share URL
		origin := c.GetHeader("Origin")

		if origin == "" {
			origin = os.Getenv("FRONTEND_URL")
		}

		if origin == "" {
			origin = "http://localhost:5173"
		}

		shareURL := origin + "/poll/" + poll.ID.Hex()

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Poll created successfully!",
			"poll":     poll,
			"shareUrl": shareURL,
		})
	})

	// -----------------------------------------
	// GET POLL
	// -----------------------------------------

	router.GET("/api/polls/:id", func(c *gin.Context) {

		id := c.Param("id")

		pollsMu.RLock()
		poll, exists := polls[id]
		pollsMu.RUnlock()

		if !exists {

			c.JSON(http.StatusNotFound, gin.H{
				"error": "Poll not found",
			})

			return
		}

		c.JSON(http.StatusOK, gin.H{
			"poll": poll,
		})
	})

	// -----------------------------------------
	// VOTE
	// -----------------------------------------

	router.POST("/api/polls/:id/vote", func(c *gin.Context) {

		id := c.Param("id")

		var request VoteRequest

		if err := c.ShouldBindJSON(&request); err != nil {

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid vote",
			})

			return
		}

		pollsMu.Lock()

		poll, exists := polls[id]

		if !exists {

			pollsMu.Unlock()

			c.JSON(http.StatusNotFound, gin.H{
				"error": "Poll not found",
			})

			return
		}

		if request.Option < 0 ||
			request.Option >= len(poll.Options) {

			pollsMu.Unlock()

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid option",
			})

			return
		}

		poll.Votes[request.Option]++

		updatedPoll := *poll

		pollsMu.Unlock()

		// Update MongoDB
		if collection != nil {

			objectID, err := bson.ObjectIDFromHex(id)

			if err == nil {

				ctx, cancel := context.WithTimeout(
					context.Background(),
					10*time.Second,
				)
				defer cancel()

				_, err = collection.UpdateOne(
					ctx,
					bson.M{"_id": objectID},
					bson.M{
						"$set": bson.M{
							"votes": updatedPoll.Votes,
						},
					},
				)

				if err != nil {
					println("MongoDB vote update failed")
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Vote recorded successfully!",
			"poll":    updatedPoll,
		})
	})

	// -----------------------------------------
	// RESULTS
	// -----------------------------------------

	router.GET("/api/polls/:id/results", func(c *gin.Context) {

		id := c.Param("id")

		pollsMu.RLock()
		poll, exists := polls[id]
		pollsMu.RUnlock()

		if !exists {

			c.JSON(http.StatusNotFound, gin.H{
				"error": "Poll not found",
			})

			return
		}

		totalVotes := 0

		for _, vote := range poll.Votes {
			totalVotes += vote
		}

		results := make([]gin.H, len(poll.Options))

		for i := range poll.Options {

			percentage := 0.0

			if totalVotes > 0 {
				percentage =
					float64(poll.Votes[i]) /
						float64(totalVotes) *
						100
			}

			results[i] = gin.H{
				"option": poll.Options[i],
				"votes":  poll.Votes[i],
				"percentage": strconv.FormatFloat(
					percentage,
					'f',
					2,
					64,
				),
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"pollId":     id,
			"totalVotes": totalVotes,
			"results":    results,
		})
	})

	// -----------------------------------------
	// START SERVER
	// -----------------------------------------

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	println("LivePoll backend starting on port " + port + "...")

	err = router.Run(":" + port)

	if err != nil {
		panic(err)
	}
}
