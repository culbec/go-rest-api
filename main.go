package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/culbec/go-rest-api/utils/websockets"
	"github.com/joho/godotenv"

	authapi "github.com/culbec/go-rest-api/api/authApi"
	gameapi "github.com/culbec/go-rest-api/api/gameApi"
	photoapi "github.com/culbec/go-rest-api/api/photoApi"
	"github.com/culbec/go-rest-api/db"
	"github.com/gin-gonic/gin"
)

const ServerHost = "127.0.0.1"
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

func prepareAuthHandlers(router *gin.Engine, auth *authapi.AuthHandler) []*gin.RouterGroup {
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

	router.POST("/gamestop/api/auth/validate", func(ctx *gin.Context) {
		_, err := auth.ValidateToken(ctx)

		if err != nil {
			log.Println(err)
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "Valid token"})
	})

	// Protected routes
	protectedRoutes := []*gin.RouterGroup{
		router.Group("/gamestop/api/games", CORSMiddleware(), auth.AuthMiddleware()),
		router.Group("/gamestop/api/photos", CORSMiddleware(), auth.AuthMiddleware()),
	}
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

func preparePhotoApiHandlers(router *gin.RouterGroup, photoH *photoapi.PhotoHandler) {
	router.GET("/:user_id", func(ctx *gin.Context) {
		ctx.Header("Content-type", "application/json")
		photoH.GetPhotosOfUser(ctx)
	})
	router.POST("", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		photoH.AddPhoto(ctx)
	})
	router.DELETE("/:filepath", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")
		photoH.DeletePhoto(ctx)
	})
}

func prepareHandlers(router *gin.Engine, db *db.Client) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(secretKey) == 0 {
		log.Fatal("JWT_SECRET_KEY missing!")
	}

	auth := authapi.NewAuthHandler(db, secretKey)
	gameH := gameapi.NewGamesHandler(db)
	photoH := photoapi.NewPhotoHandler(db)
	wsc := websockets.NewWSConfig(auth)

	router.GET("/ws", wsc.HandleWS)
	protectedRoutes := prepareAuthHandlers(router, auth)
	prepareGameApiHandlers(protectedRoutes[0], gameH)
	preparePhotoApiHandlers(protectedRoutes[1], photoH)
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	serverHost := os.Getenv("SERVER_HOST")
	serverPort := os.Getenv("SERVER_PORT")

	if serverHost != "" {
		log.Printf("Server host: %s", serverHost)
	} else {
		log.Printf("Server host not set, using default: %s", ServerHost)
	}

	if serverPort != "" {
		log.Printf("Server port: %s", serverPort)
	} else {
		log.Printf("Server port not set, using default: %s", ServerPort)
	}

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
	server := fmt.Sprintf("%s:%s", serverHost, serverPort)
	err = router.Run(server)

	if err != nil {
		log.Panicf("Error running the server: %s", err.Error())
	}
}
