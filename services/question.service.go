package questionService

import (
	"bytes"
	"context"
	"regexp" 
	"fmt"
	"io"
	"archive/tar"   
	"os"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"LeetCode-server/models"
	"log"
)

var questionCollection *mongo.Collection

func Init() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	questionCollection = client.Database("LeetCode").Collection("questions")
}

func CreateQuestion(description string, level int, tests []question.Test) (*mongo.InsertOneResult, error) {
	question := question.Question{
		Description: description,
		Level:       level,
		Tests:       tests,
	}

	result, err := questionCollection.InsertOne(context.Background(), question)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetQuestionByID(id string) (*question.Question, error) {
	questionID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var question question.Question
	err = questionCollection.FindOne(context.Background(), bson.M{"_id": questionID}).Decode(&question)
	if err != nil {
		return nil, err
	}
	return &question, nil
}

func GetAllQuestions() ([]question.Question, error) {
	cursor, err := questionCollection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var questions []question.Question
	for cursor.Next(context.Background()) {
		var question question.Question
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

func UpdateQuestion(id string, description string, level int, tests []question.Test) (*mongo.UpdateResult, error) {
	questionID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	update := bson.M{
		"$set": bson.M{
			"description": description,
			"level":       level,
			"tests":       tests,
		},
	}

	result, err := questionCollection.UpdateOne(context.Background(), bson.M{"_id": questionID}, update)
	if err != nil {
		return nil, err
	}

	return result, nil
}

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

func createTempFile(content, prefix, ext string) (string, error) {
	fileName := prefix + "." + ext
	file, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("Error creating file: %v", err)
	}
	defer file.Close()

	if _, err := file.Write([]byte(content)); err != nil {
		return "", fmt.Errorf("Error writing to file: %v", err)
	}

	return file.Name(), nil
}

func createTar(filePath string, destPath string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	header := &tar.Header{
		Name: destPath, 
		Mode: 0600,     
		Size: stat.Size(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}

	if _, err := io.Copy(tw, file); err != nil {
		return nil, err
	}

	return buf, nil
}

func runTest(funcCode, input, expectedOutput string) (string, error) {
	funcFile, err := createTempFile(funcCode, "test_func", "py")
	if err != nil {
		return "", err
	}

	testCode := fmt.Sprintf(`
from func import *

def test_func():
    result = main(%s)
    assert result == %s, f"Expected %s but got {result}"
`, input, expectedOutput, expectedOutput)

	testFile, err := createTempFile(testCode, "test", "py")
	if err != nil {
		return "", err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("Error creating Docker client: %v", err)
	}

	ctx := context.Background()

	funcTar, err := createTar(funcFile, "app/func.py")
	if err != nil {
		return "", fmt.Errorf("Error creating TAR for function file: %v", err)
	}

	testTar, err := createTar(testFile, "app/test.py")
	if err != nil {
		return "", fmt.Errorf("Error creating TAR for test file: %v", err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "my-python-test-image", 
		Cmd:   []string{"sh", "-c", "pytest app/test.py"},
	}, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("Error creating Docker container: %v", err)
	}

	if err := cli.CopyToContainer(ctx, resp.ID, "/app", funcTar, types.CopyToContainerOptions{}); err != nil {
		return "", fmt.Errorf("Error copying function file to container: %v", err)
	}

	if err := cli.CopyToContainer(ctx, resp.ID, "/app", testTar, types.CopyToContainerOptions{}); err != nil {
		return "", fmt.Errorf("Error copying test file to container: %v", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("Error starting Docker container: %v", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNextExit)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("Error waiting for container: %v", err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", fmt.Errorf("Error getting container logs: %v", err)
	}
	defer out.Close()

	var buf bytes.Buffer
	_, err = stdcopy.StdCopy(&buf, os.Stderr, out)
	if err != nil {
		return "", fmt.Errorf("Error copying logs: %v", err)
	}

	defer os.Remove(testFile)
	defer os.Remove(funcFile)

	return buf.String(), nil
}

type TestResult struct {
	TestNumber  int    `json:"test_number"`
	Passed      bool   `json:"passed"`
	Comments    string `json:"comments"`
}

func RunTests(funcCode string, questionId string) ([]TestResult, error) {
	question, err := GetQuestionByID(questionId)
	if err != nil {
		return nil, fmt.Errorf("Error fetching question: %v", err)
	}

	var results []TestResult 

	failureRegex := regexp.MustCompile(`Expected (\d+) but got ([\d\.]+)`)

	for i, test := range question.Tests {
		out, err := runTest(funcCode, test.Input, test.ExpectedOutput)

		passed := true
		var comments string
		if err != nil {
			passed = false
			comments = fmt.Sprintf("Test failed for input %s: %v", test.Input, err)
		} else {
			if failureRegex.MatchString(out) {
				match := failureRegex.FindStringSubmatch(out)
				if match != nil {
				passed = false
				comments = fmt.Sprintf("Test failed for input %s: output indicates failure: %s", test.Input, match[0])
				} else{
					comments = fmt.Sprintf("Test failed for input %s", test.Input)
				}} else {
				comments = "Test passed"
			}
		
	}
		results = append(results, TestResult{
			TestNumber: i + 1,   
			Passed:     passed,  
			Comments:   comments, 
		})
	}

	return results, nil
}