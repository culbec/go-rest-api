package gameapi

import (
	"net/http"
	"time"

	"github.com/culbec/go-rest-api/db"
	"github.com/culbec/go-rest-api/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GamesHandler struct {
	db *db.Client
}

func NewGamesHandler(db *db.Client) *GamesHandler {
	return &GamesHandler{
		db: db,
	}
}

// Returns a list of all the games
func (h *GamesHandler) GetGames(ctx *gin.Context) {
	var gquery types.GamesQuery

	if err := ctx.ShouldBindQuery(&gquery); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.D{}

	if gquery.Username != "" {
		filter = append(filter, bson.E{Key: "username", Value: gquery.Username})
	}

	if gquery.SearchFilter != "" {
		filter = append(filter, bson.E{
			Key: "title",
			Value: bson.D{{
				Key: "$regex",
				Value: primitive.Regex{
					Pattern: gquery.SearchFilter,
					Options: "i",
				},
			}},
		})
	}

	skip := uint32(gquery.Page-1) * gquery.ItemsPerPage
	options := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(gquery.ItemsPerPage))

	var games []types.Game = make([]types.Game, 0)

	// Querying the 'games' collection to retrieve all the documents
	status, err := h.db.QueryCollection("games", &filter, options, &games)
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	ctx.IndentedJSON(status, games)
}

// Returns a game based on its ID
func (h *GamesHandler) GetOneGame(ctx *gin.Context, id string) {
	var game []types.Game

	// Converting the ID to an ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Querying the 'games' collection to retrieve the document with the provided ID
	status, err := h.db.QueryCollection("games", &bson.D{{Key: "_id", Value: objectID}}, nil, &game)
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
func (h *GamesHandler) AddGame(ctx *gin.Context) *types.Game {
	// Retrieving the game from the request body
	var game types.Game
	if err := ctx.ShouldBindJSON(&game); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}

	// Declaring the inserting conditions for the document
	insertingConditions := &bson.D{
		{Key: "title", Value: game.Title},
	}

	game.Date = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	game.Version = 1

	id, status, err := h.db.InsertDocument("games", insertingConditions, game)
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return nil
	}

	// Checking if no ID was provided by the server
	// In this case, the game already exists
	if id == nil {
		ctx.JSON(status, gin.H{"error": "Game already exists"})
		return nil
	}

	game.ID = id.(primitive.ObjectID)

	return &game
}

// Deletes a game based on its ID
// Error if the game does not exist
func (h *GamesHandler) DeleteGame(ctx *gin.Context, id string) string {
	// Converting the ID to an ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return ""
	}

	status, err := h.db.DeleteDocument("games", &bson.D{{Key: "_id", Value: objectID}})
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return ""
	}

	return id
}

// Edits a game based on its ID
// Error if the game does not exist
func (h *GamesHandler) EditGame(ctx *gin.Context) *types.Game {
	// Retrieving the game from the request body
	var game types.Game
	if err := ctx.ShouldBindJSON(&game); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil
	}

	status, err := h.db.EditDocument("games", &bson.D{{Key: "_id", Value: game.ID}}, game)

	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
		return nil
	}

	game.Date = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	game.Version++

	return &game
}
