package api

import (
	"net/http"
	"time"

	"github.com/culbec/go-rest-api/db"
	"github.com/culbec/go-rest-api/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Returns a list of all the games
func GetGames(ctx *gin.Context, db *db.Client) {
	var games []types.Game = make([]types.Game, 0)

	// Querying the 'games' collection to retrieve all the documents
	status, err := db.QueryCollection("games", nil, &games)
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	ctx.IndentedJSON(status, games)
}

// Returns a game based on its ID
func GetOneGame(ctx *gin.Context, db *db.Client, id string) {
	var game []types.Game

	// Converting the ID to an ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Querying the 'games' collection to retrieve the document with the provided ID
	status, err := db.QueryCollection("games", &bson.D{{Key: "_id", Value: objectID}}, &game)
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Verifying if the game was found
	if len(game) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	ctx.IndentedJSON(status, game[0])
}

// Adds a new Game to the database
// Error if the game already exists
func AddGame(ctx *gin.Context, db *db.Client) {
	// Retrieving the game from the request body
	var game types.Game
	if err := ctx.ShouldBindJSON(&game); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Declaring the inserting conditions for the document
	insertingConditions := &bson.D{
		{Key: "title", Value: game.Title},
	}

	game.Date = time.Now().Format("2006-01-02 15:04:05")
	game.Version = 1

	id, status, err := db.InsertDocument("games", insertingConditions, game)
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Checking if no ID was provided by the server
	// In this case, the game already exists
	if id == nil {
		ctx.JSON(status, gin.H{"error": "Game already exists"})
		return
	}

	game.ID = id.(primitive.ObjectID)

	ctx.JSON(status, game)
}

// Deletes a game based on its ID
// Error if the game does not exist
func DeleteGame(ctx *gin.Context, db *db.Client, id string) {
	// Converting the ID to an ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := db.DeleteDocument("games", &bson.D{{Key: "_id", Value: objectID}})
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(status, id)
}

// Edits a game based on its ID
// Error if the game does not exist
func EditGame(ctx *gin.Context, db *db.Client) {
	// Retrieving the game from the request body
	var game types.Game
	if err := ctx.ShouldBindJSON(&game); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := db.EditDocument("games", &bson.D{{Key: "_id", Value: game.ID}}, game)

	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	game.Date = time.Now().Format("2006-01-02 15:04:05")
	game.Version++

	ctx.JSON(status, game)
}

func Pong(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
}
