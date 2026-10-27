package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type accountInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (a *App) signUp(c *gin.Context) {
	var input accountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Name = strings.TrimSpace(input.Name)
	if !validEmail(input.Email) || len(input.Password) < 10 || len(input.Password) > 72 {
		writeError(c, http.StatusBadRequest, "Enter a valid email and a password of 10–72 characters")
		return
	}
	if len([]rune(input.Name)) < 2 || len([]rune(input.Name)) > 60 {
		writeError(c, http.StatusBadRequest, "Name must be 2–60 characters")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "Could not create account")
		return
	}
	user := User{
		ID:        primitive.NewObjectID(),
		Email:     input.Email,
		Name:      input.Name,
		Password:  string(passwordHash),
		CreatedAt: time.Now().UTC(),
	}
	if _, err := a.db.Collection("users").InsertOne(c.Request.Context(), user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			writeError(c, http.StatusConflict, "An account with that email already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "Could not create account")
		return
	}
	a.writeSession(c, user)
}

func (a *App) logIn(c *gin.Context) {
	var input accountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if !validEmail(input.Email) || len(input.Password) < 1 || len(input.Password) > 72 {
		writeError(c, http.StatusUnauthorized, "Email or password is incorrect")
		return
	}

	var user User
	err := a.db.Collection("users").FindOne(c.Request.Context(), bson.M{"email": input.Email}).Decode(&user)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)) != nil {
		writeError(c, http.StatusUnauthorized, "Email or password is incorrect")
		return
	}
	a.writeSession(c, user)
}

func (a *App) writeSession(c *gin.Context, user User) {
	claims := tokenClaims{
		UserID: user.ID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.jwtSecret)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "Could not create session")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID.Hex(), "email": user.Email, "name": user.Name},
	})
}

func (a *App) requireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(c, http.StatusUnauthorized, "Sign in to continue")
			c.Abort()
			return
		}

		claims := &tokenClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return a.jwtSecret, nil
		})
		if err != nil || token == nil || !token.Valid {
			writeError(c, http.StatusUnauthorized, "Session expired; please sign in again")
			c.Abort()
			return
		}

		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			writeError(c, http.StatusUnauthorized, "Invalid session")
			c.Abort()
			return
		}
		c.Set("userID", userID)
		c.Next()
	}
}

func (a *App) currentUser(c *gin.Context) {
	userID, _ := c.Get("userID")
	var user User
	err := a.db.Collection("users").FindOne(c.Request.Context(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		writeError(c, http.StatusNotFound, "Account not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID.Hex(), "email": user.Email, "name": user.Name})
}

func validEmail(email string) bool {
	if len(email) < 5 || len(email) > 254 {
		return false
	}
	parts := strings.Split(email, "@")
	return len(parts) == 2 && parts[0] != "" && strings.Contains(parts[1], ".")
}
