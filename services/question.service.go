package service

import (
	"LeetCode-server/models"
	"context"
	"fmt"
	"log"
	"os"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var questionCollection *mongo.Collection

// Init initializes the database connection using environment variables and sets up the questionCollection.
func Init() {
	godotenv.Load()
	dbUrl := os.Getenv("DATABASE_URL")
	dbName := os.Getenv("DATABASE_NAME")
	dbCollection := os.Getenv("COLLECTION_NAME")
	clientOptions := options.Client().ApplyURI(dbUrl)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	questionCollection = client.Database(dbName).Collection(dbCollection)
}

// CreateQuestion inserts a new question into the database. It requires a title, description, level, tests, input types, and output type.
// It returns the result of the insertion and any errors encountered.
func CreateQuestion(title, description string, level int, tests []models.Test, inputTypes string, outputType string) (*mongo.InsertOneResult, error) {
	if(title == "" || description == "" ||level==0 ||len(tests) == 0){
		return nil, fmt.Errorf("Question must contain title & description & level & at least one test")
	}
	question := models.Question{
			Title:       title,
			Description: description,
			Level:       level,
			Tests:       tests,
			InputTypes: inputTypes,
			OutputType: outputType,
	}

	result, err := questionCollection.InsertOne(context.Background(), question)
	if err != nil {
			return nil, err
	}
	return result, nil
}

// GetQuestionByID retrieves a question by its ID. It returns the question object and any errors encountered.
func GetQuestionByID(id string) (*models.Question, error) {
	questionID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
			return nil, err
	}

	var question models.Question
	err = questionCollection.FindOne(context.Background(), bson.M{"_id": questionID}).Decode(&question)
	if err != nil {
			return nil, err
	}
	return &question, nil
}

// GetAllQuestions retrieves all questions from the database and returns them in a slice. It returns any errors encountered during the operation.
func GetAllQuestions() ([]models.Question, error) {
	cursor, err := questionCollection.Find(context.Background(), bson.M{})
	if err != nil {
			return nil, err
	}
	defer cursor.Close(context.Background())

	var questions []models.Question
	for cursor.Next(context.Background()) {
			var question models.Question
			if err := cursor.Decode(&question); err != nil {
					return nil, err
			}
			questions = append(questions, question)
	}

	if err := cursor.Err(); err != nil {
			return nil, err
	}

	return questions, nil
}

// UpdateQuestion updates an existing question based on the provided ID. It updates the question's title, description, level, tests, input types, and output type.
// It returns the result of the update operation and any errors encountered.
func UpdateQuestion(id string, title, description string, level int, tests []models.Test, inputTypes string, outputType string) (*mongo.UpdateResult, error) {
	questionID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
			return nil, err
	}

	update := bson.M{
			"$set": bson.M{
					"title":       title,
					"description": description,
					"level":       level,
					"tests":       tests,
					"inputTypes":  inputTypes,
					"outputType":  outputType,
			},
	}

	result, err := questionCollection.UpdateOne(context.Background(), bson.M{"_id": questionID}, update)
	if err != nil {
			return nil, err
	}

	return result, nil
}

// DeleteQuestion deletes a question by its ID. It returns the result of the deletion and any errors encountered.
func DeleteQuestion(id string) (*mongo.DeleteResult, error) {
	questionID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
			return nil, err
	}

	result, err := questionCollection.DeleteOne(context.Background(), bson.M{"_id": questionID})
	if err != nil {
			return nil, err
	}

	return result, nil
}
