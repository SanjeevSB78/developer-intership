package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// User is an account stored in MongoDB. Password contains a bcrypt hash, never
// the original password, and its JSON tag keeps it out of API responses.
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string             `bson:"email" json:"email"`
	Name      string             `bson:"name" json:"name"`
	Password  string             `bson:"password" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type PollOption struct {
	ID   string `bson:"id" json:"id"`
	Text string `bson:"text" json:"text"`
}

type Poll struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Question  string             `bson:"question" json:"question"`
	Options   []PollOption       `bson:"options" json:"options"`
	OwnerID   primitive.ObjectID `bson:"ownerId" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type Vote struct {
	PollID    primitive.ObjectID `bson:"pollId"`
	OptionID  string             `bson:"optionId"`
	VoterID   string             `bson:"voterId"`
	CreatedAt time.Time          `bson:"createdAt"`
}

// App holds shared connections used by the HTTP handlers.
type App struct {
	db        *mongo.Database
	redis     *redis.Client
	jwtSecret []byte
}

type tokenClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
