package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// During local development, read settings from the project-root .env file.
	// Deployed services provide these settings directly in their environment.
	_ = godotenv.Load("../.env")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(env("MONGODB_URI", "mongodb://localhost:27017")))
	if err != nil {
		log.Fatal("connect to MongoDB: ", err)
	}

	database := mongoClient.Database(env("MONGODB_DATABASE", "pulse_polls"))
	redisOptions, err := redisOptionsFromEnv()
	if err != nil {
		log.Fatal("configure Redis: ", err)
	}
	redisClient := redis.NewClient(redisOptions)
	app := &App{
		db:        database,
		redis:     redisClient,
		jwtSecret: []byte(env("JWT_SECRET", "local-development-secret-change-me")),
	}
	if len(app.jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must contain at least 32 characters")
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB is unavailable: ", err)
	}
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis is unavailable: ", err)
	}
	if err := app.createIndexes(ctx); err != nil {
		log.Fatal("create database indexes: ", err)
	}

	server := app.routes()
	address := env("API_ADDR", ":"+env("PORT", "8080"))
	log.Printf("API listening on %s", address)
	if err := server.Run(address); err != nil && err != http.ErrServerClosed {
		log.Fatal("run API server: ", err)
	}
}

func (a *App) createIndexes(ctx context.Context) error {
	_, err := a.db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	_, err = a.db.Collection("votes").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "pollId", Value: 1}, {Key: "voterId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func (a *App) routes() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{env("FRONTEND_ORIGIN", "http://localhost:5173")}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	router.Use(cors.New(corsConfig))

	router.GET("/health", a.health)

	api := router.Group("/api")
	api.POST("/auth/signup", a.signUp)
	api.POST("/auth/login", a.logIn)
	api.GET("/auth/me", a.requireUser(), a.currentUser)
	api.POST("/polls", a.requireUser(), a.createPoll)
	api.GET("/polls/:id", a.getPoll)
	api.POST("/polls/:id/votes", a.createVote)
	api.GET("/polls/:id/events", a.pollEvents)

	return router
}

func redisOptionsFromEnv() (*redis.Options, error) {
	if address := os.Getenv("REDIS_URL"); address != "" {
		return redis.ParseURL(address)
	}
	return &redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
	}, nil
}

func (a *App) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := a.db.Client().Ping(ctx, nil); err != nil {
		writeError(c, http.StatusServiceUnavailable, "Database unavailable")
		return
	}
	if err := a.redis.Ping(ctx).Err(); err != nil {
		writeError(c, http.StatusServiceUnavailable, "Realtime service unavailable")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
