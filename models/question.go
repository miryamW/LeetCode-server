package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Question struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title"`
	Description string             `bson:"description"`
	Level       int                `bson:"level"`
	Tests       []Test             `bson:"tests"` 
}
