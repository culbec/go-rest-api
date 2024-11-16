package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/culbec/go-rest-api/utils/websockets"

	authapi "github.com/culbec/go-rest-api/api/authApi"
	gameapi "github.com/culbec/go-rest-api/api/gameApi"
	"github.com/culbec/go-rest-api/db"
	"github.com/gin-gonic/gin"
)

const ServerHost = "localhost"
const ServerPort = "3000"

func CORSMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins temporarily
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Authorization, Access-Control-Allow-Origin")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	}
}

func prepareAuthHandlers(router *gin.Engine, auth *authapi.AuthHandler) *gin.RouterGroup {
	router.POST("/gamestop/api/auth/login", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		auth.Login(ctx)
	})

	router.POST("/gamestop/api/auth/register", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		auth.Register(ctx)
	})

	router.POST("/gamestop/api/auth/logout", func(ctx *gin.Context) {
		_, err := auth.ValidateToken(ctx)

		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			return
		}

		auth.Logout(ctx)
	})

	// Protected routes
	protectedRoutes := router.Group("/gamestop/api/games", CORSMiddleware(), auth.AuthMiddleware())
	return protectedRoutes
}

func prepareGameApiHandlers(router *gin.RouterGroup, gameH *gameapi.GamesHandler) {
	router.GET("", func(ctx *gin.Context) {
		ctx.Header("Content-type", "application/json")
		gameH.GetGames(ctx)
	})
	router.GET(":id", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")

		// Retrieving the ID from the URL
		id := ctx.Param("id")

		if id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "ID not provided"})
			return
		}

		gameH.GetOneGame(ctx, id)
	})
	router.POST("", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		gameH.AddGame(ctx)

	})
	router.PUT("", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		ctx.Header("Accept", "application/json")
		gameH.EditGame(ctx)
	})
	router.DELETE(":id", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")
		gameH.DeleteGame(ctx)
	})
}

func prepareHandlers(router *gin.Engine, db *db.Client) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(secretKey) == 0 {
		log.Fatal("JWT_SECRET_KEY missing!")
	}

	auth := authapi.NewAuthHandler(db, secretKey)
	gameH := gameapi.NewGamesHandler(db)
	wsc := websockets.NewWSConfig(auth)

	router.GET("/ws", wsc.HandleWS)
	protectedRoutes := prepareAuthHandlers(router, auth)
	prepareGameApiHandlers(protectedRoutes, gameH)
}

func main() {
	router := gin.Default()

	// Enabling default CORS configuration
	// CORS middleware
	router.Use(CORSMiddleware())

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
	server := fmt.Sprintf("%s:%s", ServerHost, ServerPort)
	err = router.Run(server)

	if err != nil {
		log.Panicf("Error running the server: %s", err.Error())
	}
}
