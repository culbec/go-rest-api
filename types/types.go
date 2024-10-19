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
