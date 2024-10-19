package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/culbec/go-rest-api/api"
	"github.com/culbec/go-rest-api/db"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const SERVER_HOST = "localhost"
const SERVER_PORT = "3000"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	}, // Allow all connections
}

var connections = make(map[*websocket.Conn]bool)
var mutex sync.Mutex

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func broadcastMessage(message Message) {
	mutex.Lock()
	defer mutex.Unlock()

	for conn := range connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error writing the message: %s", err.Error())
			conn.Close()
			delete(connections, conn)
		}
	}
}

func prepareHandlers(router *gin.Engine, db *db.Client) {
	router.GET("/gamestop/api/ping", api.Pong)
	router.GET("/ws", func(ctx *gin.Context) {
		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			log.Printf("Error upgrading the connection: %s", err.Error())
			return
		}
		mutex.Lock()
		connections[conn] = true
		mutex.Unlock()

		defer func() {
			mutex.Lock()
			defer mutex.Unlock()

			conn.Close()
			delete(connections, conn)
		}()

		for {
			// Reading the message from the client
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading the message: %s", err.Error())
				break
			}

			// Echoing the message back to the client
			if err := conn.WriteMessage(messageType, p); err != nil {
				log.Printf("Error writing the message: %s", err.Error())
				break
			}

		}
	})
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
		game := api.AddGame(ctx, db)
		message := Message{
			Type: "SAVE_GAME",
			Data: game,
		}
		log.Printf("Message: %s", message)
		broadcastMessage(message)
	})
	router.PUT("/gamestop/api/games", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		ctx.Header("Accept", "application/json")
		game := api.EditGame(ctx, db)
		message := Message{
			Type: "UPDATE_GAME",
			Data: game,
		}
		log.Printf("Message: %s", message)
		broadcastMessage(message)
	})
	router.DELETE("/gamestop/api/games/:id", func(ctx *gin.Context) {
		ctx.Header("Accept", "application/json")

		// Retrieving the ID from the URL
		id := ctx.Param("id")

		if id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID not provided"})
			return
		}

		deletedId := api.DeleteGame(ctx, db, id)
		message := Message{
			Type: "DELETE_GAME",
			Data: deletedId,
		}
		log.Printf("Message: %s", message)
		broadcastMessage(message)
	})
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
