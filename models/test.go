package models

type Test struct {
	Input          string `bson:"input"`         
	ExpectedOutput string `bson:"expected_output"` 
}