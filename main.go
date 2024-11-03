package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	authapi "github.com/culbec/go-rest-api/api/authApi"
	gameapi "github.com/culbec/go-rest-api/api/gameApi"
	"github.com/culbec/go-rest-api/db"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const SERVER_HOST = "localhost"
const SERVER_PORT = "3000"

type WSConfig struct {
	auth        *authapi.AuthHandler
	connections map[*websocket.Conn]string // conn: username
	mutex       *sync.Mutex
}

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Sender    string      `json:"sender"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewWSConfig(auth *authapi.AuthHandler) *WSConfig {
	return &WSConfig{
		auth:        auth,
		connections: make(map[*websocket.Conn]string),
		mutex:       &sync.Mutex{},
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // all origins are allowed
	},
}

func (wsc *WSConfig) handleWS(ctx *gin.Context) {
	// Validate the token and retrieving the username
	username, err := wsc.auth.ValidateToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("Error upgrading the connection: %s", err.Error())
		return
	}

	// Storing the connection
	wsc.mutex.Lock()
	wsc.connections[conn] = username
	wsc.mutex.Unlock()

	defer func() {
		wsc.mutex.Lock()
		defer wsc.mutex.Unlock()

		conn.Close()
		delete(wsc.connections, conn)
	}()

	for {
		var message Message
		if err := conn.ReadJSON(&message); err != nil {
			log.Printf("Error reading the message: %s", err.Error())
			break
		}

		message.Sender = username
		message.Timestamp = time.Now()

		wsc.broadcastMessage(message)
	}
}

func (wsc *WSConfig) handleLogout(username string) {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()

	for conn, user := range wsc.connections {
		if user == username {
			conn.Close()
			delete(wsc.connections, conn)
			break
		}
	}
}

func (wsc *WSConfig) broadcastMessage(message Message) {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()

	for conn, username := range wsc.connections {
		// Sending the message only to the sender
		if username != message.Sender {
			continue
		}

		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error broadcasting to %s: %v", username, err)
			conn.Close()
			delete(wsc.connections, conn)
		}
	}
}

func prepareAuthHandlers(router *gin.Engine, auth *authapi.AuthHandler, wsc *WSConfig) *gin.RouterGroup {
	router.POST("/gamestop/api/auth/login", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		auth.Login(ctx)
	})

	router.POST("/gamestop/api/auth/register", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		auth.Register(ctx)
	})

	router.POST("/gamestop/api/auth/logout", func(ctx *gin.Context) {
		username, exists := ctx.Get("username")

		if !exists {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
			return
		}

		wsc.handleLogout(username.(string))
		auth.Logout(ctx)
	})

	// Protected routes
	protectedRoutes := router.Group("/gamestop/api/games", auth.AuthMiddleware())
	return protectedRoutes
}

func prepareGameApiHandlers(router *gin.RouterGroup, gameH *gameapi.GamesHandler, wsc *WSConfig) {
	router.GET("/gamestop/api/games", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")
		gameH.GetGames(ctx)
	})
	router.GET("/gamestop/api/games/:id", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")

		// Retrieving the ID from the URL
		id := ctx.Param("id")

		if id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID not provided"})
			return
		}

		gameH.GetOneGame(ctx, id)
	})
	router.POST("/gamestop/api/games", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		game := gameH.AddGame(ctx)
		message := Message{
			Type: "SAVE_GAME",
			Data: game,
		}
		log.Printf("Message: %s", message)
		wsc.broadcastMessage(message)
	})
	router.PUT("/gamestop/api/games", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		ctx.Header("Accept", "application/json")
		game := gameH.EditGame(ctx)
		message := Message{
			Type: "UPDATE_GAME",
			Data: game,
		}
		log.Printf("Message: %s", message)
		wsc.broadcastMessage(message)
	})
	router.DELETE("/gamestop/api/games/:id", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")

		// Retrieving the ID from the URL
		id := ctx.Param("id")

		if id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID not provided"})
			return
		}

		deletedId := gameH.DeleteGame(ctx, id)
		message := Message{
			Type: "DELETE_GAME",
			Data: deletedId,
		}
		log.Printf("Message: %s", message)
		wsc.broadcastMessage(message)
	})
}

func prepareHandlers(router *gin.Engine, db *db.Client) {
	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(secretKey) == 0 {
		log.Fatal("JWT_SECRET_KEY missing!")
	}

	auth := authapi.NewAuthHandler(db, secretKey)
	gameH := gameapi.NewGamesHandler(db)
	wsc := NewWSConfig(auth)

	router.GET("/ws", wsc.handleWS)
	protectedRoutes := prepareAuthHandlers(router, auth, wsc)
	prepareGameApiHandlers(protectedRoutes, gameH, wsc)
}

func main() {
	router := gin.Default()

	// Enabling default CORS configuration
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

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
