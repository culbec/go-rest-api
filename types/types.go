package types

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Game Struct
type Game struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title"`
	ReleaseDate string             `json:"release_date" bson:"release_date"`
	RentalPrice float64            `json:"rental_price" bson:"rental_price"`
	Rating      int                `json:"rating" bson:"rating"`
	Category    string             `json:"category" bson:"category"`
	Date        string             `json:"date" bson:"date"`
	Version     int                `json:"version" bson:"version"`
}

// GamesQuery Struct
type GamesQuery struct {
	Username     string `json:"username" bson:"username"`
	Page         uint32 `json:"page" bson:"page"`
	ItemsPerPage uint32 `json:"items_per_page" bson:"items_per_page"`
	SearchFilter string `json:"search_filter" bson:"search_filter"`
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
	Token string `json:"token"`
}
