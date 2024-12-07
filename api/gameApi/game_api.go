package gameapi

import (
	"log"
	"net/http"
	"time"

	"github.com/culbec/go-rest-api/db"
	"github.com/culbec/go-rest-api/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GamesHandler struct {
	db *db.Client
}

func NewGamesHandler(db *db.Client) *GamesHandler {
	return &GamesHandler{
		db: db,
	}
}

// GetGames Returns a list of all the games
func (h *GamesHandler) GetGames(ctx *gin.Context) {
	var games = make([]types.Game, 0)

	filter := &bson.D{{Key: "username", Value: ctx.GetString("username")}}

	// Querying the 'games' collection to retrieve all the documents
	status, err := h.db.QueryCollection("games", filter, nil, &games)
	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	ctx.IndentedJSON(status, games)
}

// GetOneGame Returns a game based on its ID
func (h *GamesHandler) GetOneGame(ctx *gin.Context, id string) {
	var game []types.Game

	// Converting the ID to an ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Querying the 'games' collection to retrieve the document with the provided ID
	status, err := h.db.QueryCollection("games", &bson.D{{Key: "_id", Value: objectID}}, nil, &game)
	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	// Verifying if the game was found
	if len(game) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Game not found"})
		return
	}

	ctx.IndentedJSON(status, game[0])
}

// AddGame Adds a new Game to the database
// Error if the game already exists
func (h *GamesHandler) AddGame(ctx *gin.Context) {
	// Retrieving the game from the request body
	var game types.Game
	if err := ctx.ShouldBindJSON(&game); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Declaring the inserting conditions for the document
	insertingConditions := &bson.D{
		{Key: "title", Value: game.Title},
	}

	game.Username = ctx.GetString("username")
	game.Date = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	game.Version = 1

	id, status, err := h.db.InsertDocument("games", insertingConditions, game)
	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	// Checking if no ID was provided by the server
	// In this case, the game already exists
	if id == nil {
		ctx.JSON(status, gin.H{"message": "Game already exists"})
		return
	}

	game.ID = id.(primitive.ObjectID)

	ctx.JSON(status, game)
}

// DeleteGame Deletes a game based on its ID
// Error if the game does not exist
func (h *GamesHandler) DeleteGame(ctx *gin.Context) {
	// Checking if an ID was provided
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "ID not provided"})
		return
	}

	// Converting the ID to an ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	status, err := h.db.DeleteDocument("games", &bson.D{{Key: "_id", Value: objectID}})
	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	ctx.JSON(status, gin.H{"message": "Game deleted"})
}

// EditGame Edits a game based on its ID
// Error if the game does not exist
func (h *GamesHandler) EditGame(ctx *gin.Context) {
	// Retrieving the game from the request body
	var game types.Game
	if err := ctx.ShouldBindJSON(&game); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	log.Printf("Received game: %v\n", game)

	status, err := h.db.EditDocument("games", &bson.D{{Key: "_id", Value: game.ID}}, game)

	if err != nil {
		ctx.JSON(status, gin.H{"message": err.Error()})
		return
	}

	game.Date = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	game.Version++

	ctx.JSON(status, game)
}
