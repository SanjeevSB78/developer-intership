package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type createPollInput struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

type voteInput struct {
	OptionID string `json:"optionId"`
	VoterID  string `json:"voterId"`
}

type voteEvent struct {
	OptionID string `json:"optionId"`
	Count    int64  `json:"count"`
	Total    int64  `json:"total"`
}

func (a *App) createPoll(c *gin.Context) {
	var input createPollInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	input.Question = strings.TrimSpace(input.Question)
	if len([]rune(input.Question)) < 8 || len([]rune(input.Question)) > 180 {
		writeError(c, http.StatusBadRequest, "Question must be 8–180 characters")
		return
	}
	if len(input.Options) < 2 || len(input.Options) > 6 {
		writeError(c, http.StatusBadRequest, "A poll must have 2–6 options")
		return
	}

	options := make([]PollOption, 0, len(input.Options))
	seen := make(map[string]bool)
	for _, text := range input.Options {
		text = strings.TrimSpace(text)
		if len([]rune(text)) == 0 || len([]rune(text)) > 80 {
			writeError(c, http.StatusBadRequest, "Each option must be 1–80 characters")
			return
		}
		key := strings.ToLower(text)
		if seen[key] {
			writeError(c, http.StatusBadRequest, "Poll options must be unique")
			return
		}
		seen[key] = true
		options = append(options, PollOption{ID: primitive.NewObjectID().Hex(), Text: text})
	}

	userID, _ := c.Get("userID")
	poll := Poll{
		ID:        primitive.NewObjectID(),
		Question:  input.Question,
		Options:   options,
		OwnerID:   userID.(primitive.ObjectID),
		CreatedAt: time.Now().UTC(),
	}
	if _, err := a.db.Collection("polls").InsertOne(c.Request.Context(), poll); err != nil {
		writeError(c, http.StatusInternalServerError, "Could not create poll")
		return
	}
	a.writePublicPoll(c, poll)
}

func (a *App) getPoll(c *gin.Context) {
	poll, err := a.findPoll(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "Poll not found")
		return
	}
	a.writePublicPoll(c, poll)
}

func (a *App) findPoll(ctx context.Context, id string) (Poll, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return Poll{}, mongo.ErrNoDocuments
	}
	var poll Poll
	err = a.db.Collection("polls").FindOne(ctx, bson.M{"_id": objectID}).Decode(&poll)
	return poll, err
}

func (a *App) writePublicPoll(c *gin.Context, poll Poll) {
	counts, err := a.pollCounts(c.Request.Context(), poll)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "Could not load results")
		return
	}
	total := int64(0)
	for _, count := range counts {
		total += count
	}
	c.JSON(http.StatusOK, gin.H{
		"id": poll.ID.Hex(), "question": poll.Question, "options": poll.Options,
		"counts": counts, "total": total, "createdAt": poll.CreatedAt,
	})
}

// pollCounts reads Redis first because Redis serves the live view. When a poll
// has no cached counts yet, MongoDB's saved votes are used to seed the cache.
func (a *App) pollCounts(ctx context.Context, poll Poll) (map[string]int64, error) {
	key := "poll:" + poll.ID.Hex() + ":counts"
	values, err := a.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		values, err = a.seedCounts(ctx, key, poll)
		if err != nil {
			return nil, err
		}
	}

	counts := make(map[string]int64, len(poll.Options))
	for _, option := range poll.Options {
		count := int64(0)
		if value := values[option.ID]; value != "" {
			if _, err := fmt.Sscan(value, &count); err != nil {
				return nil, err
			}
		}
		counts[option.ID] = count
	}
	return counts, nil
}

func (a *App) seedCounts(ctx context.Context, key string, poll Poll) (map[string]string, error) {
	pipeline, err := a.db.Collection("votes").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"pollId": poll.ID}}},
		{{Key: "$group", Value: bson.M{"_id": "$optionId", "count": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, err
	}
	defer pipeline.Close(ctx)

	countsFromMongo := make(map[string]int64)
	for pipeline.Next(ctx) {
		var row struct {
			OptionID string `bson:"_id"`
			Count    int64  `bson:"count"`
		}
		if err := pipeline.Decode(&row); err != nil {
			return nil, err
		}
		countsFromMongo[row.OptionID] = row.Count
	}
	if err := pipeline.Err(); err != nil {
		return nil, err
	}

	commands := a.redis.Pipeline()
	for _, option := range poll.Options {
		commands.HSetNX(ctx, key, option.ID, countsFromMongo[option.ID])
	}
	if _, err := commands.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}
	return a.redis.HGetAll(ctx, key).Result()
}

func (a *App) createVote(c *gin.Context) {
	ctx := c.Request.Context()
	poll, err := a.findPoll(ctx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "Poll not found")
		return
	}
	var input voteInput
	if err := c.ShouldBindJSON(&input); err != nil || len(input.VoterID) < 16 || len(input.VoterID) > 80 {
		writeError(c, http.StatusBadRequest, "Invalid vote")
		return
	}
	if !hasOption(poll.Options, input.OptionID) {
		writeError(c, http.StatusBadRequest, "Choose a valid option")
		return
	}

	vote := Vote{PollID: poll.ID, OptionID: input.OptionID, VoterID: input.VoterID, CreatedAt: time.Now().UTC()}
	if _, err := a.db.Collection("votes").InsertOne(ctx, vote); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			writeError(c, http.StatusConflict, "You have already voted in this poll")
			return
		}
		writeError(c, http.StatusInternalServerError, "Could not save vote")
		return
	}

	key := "poll:" + poll.ID.Hex() + ":counts"
	count, err := a.redis.HIncrBy(ctx, key, input.OptionID, 1).Result()
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "Vote saved, but live results are catching up; reload results shortly")
		return
	}
	counts, err := a.pollCounts(ctx, poll)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "Vote saved, but live results are catching up")
		return
	}
	total := int64(0)
	for _, optionCount := range counts {
		total += optionCount
	}

	event := voteEvent{OptionID: input.OptionID, Count: count, Total: total}
	encodedEvent, err := json.Marshal(event)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "Could not publish vote")
		return
	}
	if err := a.redis.Publish(ctx, "poll:"+poll.ID.Hex()+":events", encodedEvent).Err(); err != nil {
		writeError(c, http.StatusServiceUnavailable, "Vote saved, but live updates are temporarily unavailable")
		return
	}
	c.JSON(http.StatusCreated, event)
}

func hasOption(options []PollOption, id string) bool {
	for _, option := range options {
		if option.ID == id {
			return true
		}
	}
	return false
}

func (a *App) pollEvents(c *gin.Context) {
	ctx := c.Request.Context()
	poll, err := a.findPoll(ctx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "Poll not found")
		return
	}

	subscription := a.redis.Subscribe(ctx, "poll:"+poll.ID.Hex()+":events")
	defer subscription.Close()
	if _, err := subscription.Receive(ctx); err != nil {
		writeError(c, http.StatusServiceUnavailable, "Could not connect to live results")
		return
	}

	// Subscribe before reading the counts so votes that arrive during setup
	// are either included in this snapshot or queued as an event (or both).
	counts, err := a.pollCounts(ctx, poll)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "Could not initialize live results")
		return
	}
	total := int64(0)
	for _, count := range counts {
		total += count
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.SSEvent("snapshot", gin.H{"counts": counts, "total": total})
	c.Writer.Flush()

	c.Stream(func(writer io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		case message, open := <-subscription.Channel():
			if !open {
				return false
			}
			_, _ = fmt.Fprintf(writer, "event: vote\ndata: %s\n\n", message.Payload)
			return true
		case <-time.After(20 * time.Second):
			_, _ = fmt.Fprint(writer, ": keep-alive\n\n")
			return true
		}
	})
}
