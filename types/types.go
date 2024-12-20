package types

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Photo struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserID   string             `json:"user_id" bson:"user_id"`
	GameID   string             `json:"game_id" bson:"game_id"`
	Filepath string             `json:"filepath" bson:"filepath"`
	Data     string             `json:"data" bson:"data"`
}

// Location struct
type Location struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
}

// Game Struct
type Game struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Username    string             `json:"username" bson:"username"`
	Title       string             `json:"title" bson:"title"`
	ReleaseDate string             `json:"release_date" bson:"release_date"`
	RentalPrice float64            `json:"rental_price" bson:"rental_price"`
	Rating      int                `json:"rating" bson:"rating"`
	Category    string             `json:"category" bson:"category"`
	Location    Location           `json:"location" bson:"location"`
	Date        string             `json:"date" bson:"date"`
	Version     int                `json:"version" bson:"version"`
}

// GamesQuery Struct
type GamesQuery struct {
	Page  int64  `bson:"page"`
	Limit int64  `bson:"limit"`
	Title string `bson:"title"`
}

// User Struct
type User struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Username string             `json:"username" bson:"username"`
	Password string             `json:"password" bson:"password"`
	Salt     string             `json:"salt" bson:"salt"`
	Date     string             `json:"date" bson:"date"`
	Version  int                `json:"version" bson:"version"`
}

// LoginRequest Struct
type LoginRequest struct {
	Username string `json:"username" bson:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest Struct
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse Struct
type AuthResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// MessageData Struct
type MessageData struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Sender  string      `json:"sender"`
}
