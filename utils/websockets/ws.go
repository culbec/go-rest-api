package websockets

import "fmt"

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	authapi "github.com/culbec/go-rest-api/api/authApi"
	"github.com/culbec/go-rest-api/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSConfig struct {
	auth        *authapi.AuthHandler
	connections map[*websocket.Conn]string // conn: username
	mutex       *sync.Mutex
	cancelFuncs map[*websocket.Conn]context.CancelFunc
}

func NewWSConfig(auth *authapi.AuthHandler) *WSConfig {
	return &WSConfig{
		auth:        auth,
		connections: make(map[*websocket.Conn]string),
		mutex:       &sync.Mutex{},
		cancelFuncs: make(map[*websocket.Conn]context.CancelFunc),
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // all origins are allowed
	},
}

func (wsc *WSConfig) StartSampleNotifications(ctx context.Context, conn *websocket.Conn, username string) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	notificationId := 1

	for {
		select {
		case <-ctx.Done():
			log.Printf("[WS] Stopping notifications for %s", username)
			return
		case <-ticker.C:
			log.Printf("[WS] Sending notification to %s", username)
			message := types.MessageData{
				Type:    "notification",
				Payload: "Hello " + username + ", this is a sample notification no. " + fmt.Sprint(notificationId),
				Sender:  "server",
			}
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("[WS] Error sending notification to %s: %v", username, err)
				conn.Close()
				wsc.mutex.Lock()
				delete(wsc.connections, conn)
				delete(wsc.cancelFuncs, conn)
				wsc.mutex.Unlock()
				return
			}
			notificationId++
		}
	}
}

func (wsc *WSConfig) HandleWS(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("[WS] Error upgrading the connection: %s", err.Error())
		return
	}

	defer func() {
		wsc.mutex.Lock()
		defer wsc.mutex.Unlock()

		conn.Close()
		delete(wsc.connections, conn)
		if cancel, ok := wsc.cancelFuncs[conn]; ok {
			cancel()
			delete(wsc.cancelFuncs, conn)
		}
	}()

	// Wait for the authMessage
	var authMessage types.MessageData

	if err := conn.ReadJSON(&authMessage); err != nil {
		log.Printf("[WS] Error reading the auth message: %s", err.Error())
		conn.WriteJSON(types.MessageData{
			Type:    "error",
			Payload: "Invalid message format",
			Sender:  "server",
		})
		return
	}

	log.Printf("[WS] Auth message: %s", authMessage)

	if authMessage.Type != "authorization" {
		log.Printf("[WS] Invalid message type: %s", authMessage.Type)
		conn.WriteJSON(types.MessageData{
			Type:    "error",
			Payload: "Invalid message type",
			Sender:  "server",
		})
		return
	}

	token := authMessage.Payload.(string)
	// Validate the token and retrieve the username
	username, err := wsc.auth.GetJWTManager().ValidateToken(token)
	if err != nil {
		conn.WriteJSON(types.MessageData{
			Type:    "error",
			Payload: "Invalid token",
			Sender:  "server",
		})
		return
	}

	log.Printf("[WS] User '%s' connected", username)

	// Store the connection
	wsc.mutex.Lock()
	wsc.connections[conn] = username
	wsc.mutex.Unlock()

	// Create a context for the notification goroutine
	wsContext, cancel := context.WithCancel(context.Background())
	wsc.mutex.Lock()
	wsc.cancelFuncs[conn] = cancel
	wsc.mutex.Unlock()

	// Start sending sample notifications
	go wsc.StartSampleNotifications(wsContext, conn, username)

	for {
		var message types.MessageData
		if err := conn.ReadJSON(&message); err != nil {
			log.Printf("[WS] Error reading the message: %s", err.Error())
			break
		}

		log.Printf("[WS] Received message: %s", message)

		if message.Type == "logout" {
			wsc.HandleLogout(username)
			break
		}

		wsc.broadcastMessage(message)
	}
}

func (wsc *WSConfig) HandleLogout(username string) {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()

	for conn, user := range wsc.connections {
		if user == username {
			// Send close message
			closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "User logged out")
			log.Printf("[WS] Sending close message to %s", username)

			if err := conn.WriteMessage(websocket.CloseMessage, closeMessage); err != nil {
				log.Printf("[WS] Error sending close message to %s: %v", username, err)
			}

			if cancel, ok := wsc.cancelFuncs[conn]; ok {
				cancel()
				delete(wsc.cancelFuncs, conn)
			}

			// Time for the client to receive the message
			time.Sleep(1000 * time.Millisecond)

			conn.Close()
			delete(wsc.connections, conn)
			break
		}
	}
}

func (wsc *WSConfig) broadcastMessage(message types.MessageData) {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()

	for conn, username := range wsc.connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("[WS] Error broadcasting to %s: %v", username, err)

			conn.Close()
			delete(wsc.connections, conn)
			if cancel, ok := wsc.cancelFuncs[conn]; ok {
				cancel()
				delete(wsc.cancelFuncs, conn)
			}
		}
	}
}
