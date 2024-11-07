package question

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Test struct {
	Input          interface{} `bson:"input"`         
	ExpectedOutput interface{} `bson:"expected_output"` 
}

type Question struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Description string             `bson:"description"`
	Level       int                `bson:"level"`
	Tests       []Test             `bson:"tests"` 
}