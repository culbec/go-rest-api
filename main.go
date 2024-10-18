package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/culbec/go-rest-api/api"
	"github.com/culbec/go-rest-api/db"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const SERVER_HOST = "localhost"
const SERVER_PORT = "8080"

func prepareHandlers(router *gin.Engine, db *db.Client) {
	router.GET("/gamestop/api/ping", api.Pong)
	router.GET("/gamestop/api/games", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")
		api.GetGames(ctx, db)
	})
	router.GET("/gamestop/api/games/:id", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")

		// Retrieving the ID from the URL
		id := ctx.Param("id")

		if id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID not provided"})
			return
		}

		api.GetOneGame(ctx, db, id)
	})
	router.POST("/gamestop/api/games", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		api.AddGame(ctx, db)
	})
	router.PUT("/gamestop/api/games", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		ctx.Header("Accept", "application/json")
		api.EditGame(ctx, db)
	})
	router.DELETE("/gamestop/api/games/:id", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")

		// Retrieving the ID from the URL
		id := ctx.Param("id")

		if id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID not provided"})
			return
		}

		api.DeleteGame(ctx, db, id)
	})
}

func main() {
	router := gin.Default()

	// Enabling default CORS configuration
	router.Use(cors.Default())

	// Retrieving a new client connection
	client, err := db.PrepareClient()

	if err != nil {
		log.Panicf("Error connecting to the database: %s", err.Error())
	}
	defer db.Cleanup(client)

	log.Println("Connected to the database!")

	// Preparing the handlers
	prepareHandlers(router, client)

	// Running the server
	server := fmt.Sprintf("%s:%s", SERVER_HOST, SERVER_PORT)
	router.Run(server)
}
