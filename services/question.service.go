package questionService

import (
	"LeetCode-server/models"
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
	 metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var questionCollection *mongo.Collection

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


func CreateQuestion(title, description string, level int, tests []question.Test) (*mongo.InsertOneResult, error) {
	question := question.Question{
			Title:       title,
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


func UpdateQuestion(id string, title, description string, level int, tests []question.Test) (*mongo.UpdateResult, error) {
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

func deployToKubernetes(containerImage string) error {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return fmt.Errorf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Error creating Kubernetes client: %v", err)
	}
	podName := "test-pod-" + time.Now().Format("20060102150405")
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:  podName,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "python-test-container",
					Image: containerImage,
					Command: []string{"sh", "-c", "pytest app/test.py"},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	_, err = clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Error creating Kubernetes pod: %v", err)
	}

	fmt.Println("Pod created successfully")

	return nil
}

func runTestPython(funcCode, input, expectedOutput string) (string, error) {
	funcName, err := extractFuncName(funcCode)
	if err != nil {
		return "", err
	}

	funcFile, err := createTempFile(funcCode, "test_func", "py")
	if err != nil {
		return "", err
	}

	testCode := fmt.Sprintf(`
from func import *

def test_func():
    result = %s(%s)
    assert result == %s, f"Expected %s but got {result}"
`, funcName, input, expectedOutput, expectedOutput)

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
		Cmd:   []string{"sh", "-c", "pytest -q app/test.py"},
	}, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("Error creating Docker container: %v", err)
	}

	if err := cli.CopyToContainer(ctx, resp.ID, "/app", funcTar, container.CopyToContainerOptions{}); err != nil {
		return "", fmt.Errorf("Error copying function file to container: %v", err)
	}

	if err := cli.CopyToContainer(ctx, resp.ID, "/app", testTar, container.CopyToContainerOptions{}); err != nil {
		return "", fmt.Errorf("Error copying test file to container: %v", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("Error starting Docker container: %v", err)
	}

	err = deployToKubernetes("my-python-test-image")
	if err != nil {
		return "", fmt.Errorf("Error deploying to Kubernetes: %v", err)
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

func extractFuncName(funcCode string) (string, error) {
	re := regexp.MustCompile(`def (\w+)`)
	matches := re.FindStringSubmatch(funcCode)
	if len(matches) < 2 {
		return "", fmt.Errorf("Could not find function name in the provided code")
	}
	return matches[1], nil
}

func extractFunctionName(funcCode string, returnType string) (string, error) {
	re := regexp.MustCompile(fmt.Sprintf(`%s\s+(\w+)\s*\(`, regexp.QuoteMeta(returnType)))
	matches := re.FindStringSubmatch(funcCode)
	if len(matches) < 1{
		return "", fmt.Errorf("Could not find function name after return type '%s' in the code", returnType)
	}

	return matches[1], nil
}



func extractModifier(funcCode string) (string, error) {
	funcCode = regexp.MustCompile(`\s+`).ReplaceAllString(funcCode, " ")

	reStatic := regexp.MustCompile(`public\s+static\s+([a-zA-Z0-9\[\]]+)\s+\w+\(`)
	matchesStatic := reStatic.FindStringSubmatch(funcCode)
	if len(matchesStatic) > 1 {
		return matchesStatic[1], nil
	}

	rePublic := regexp.MustCompile(`public\s+([a-zA-Z0-9\[\]]+)\s+\w+\(`)
	matchesPublic := rePublic.FindStringSubmatch(funcCode)
	if len(matchesPublic) > 1 {
		return matchesPublic[1], nil
	}

	return "", fmt.Errorf("Could not find return type in the code")
}

func convertInputOutputArray(input string) (string, error) {
	re := regexp.MustCompile(`\[(.*?)\]`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		return input, nil 
	}

	arrayContent := matches[1]
	if regexp.MustCompile(`^\d+(\s*,\s*\d+)*$`).MatchString(arrayContent) {
		return strings.Replace(input, matches[0], "new int[]{" + arrayContent + "}", 1), nil
	} else if regexp.MustCompile(`^\d+\.\d+(\s*,\s*\d+\.\d+)*$`).MatchString(arrayContent) {
		return strings.Replace(input, matches[0], "new double[]{" + arrayContent + "}", 1), nil
	} else if regexp.MustCompile(`^".*?"(\s*,\s*".*?")*$`).MatchString(arrayContent) {
		return strings.Replace(input, matches[0], "new String[]{" + arrayContent + "}", 1), nil
	} else {
		return "", fmt.Errorf("Unsupported array format")
	}
}

func runTestJava(funcCode, input, expectedOutput string) (string, error) {

	funcFile, err := createTempFile(funcCode, "Main", "java")
	if err != nil {
		return "", err
	}

	convertedInput, err := convertInputOutputArray(input)
	if err != nil {
		return "", err
	}
	convertedOutput, err := convertInputOutputArray(expectedOutput)
	if err != nil {
		return "", err
	}

	modifier, err := extractModifier(funcCode)
	if err != nil {
		return "", err
	}
	funcName, err := extractFunctionName(funcCode, modifier)
	if err != nil {
		return "", err
	}
	var assert string
	var print string
	if (convertedOutput == expectedOutput) {
		assert = "assertEquals"
		print = fmt.Sprintf(`System.out.println(main.%s(%s));`, funcName, convertedInput)
	} else {
		assert = "assertArrayEquals"
		print = fmt.Sprintf(`System.out.println(Arrays.toString(main.%s(%s)));`, funcName, convertedInput)
	}	
	testCode := fmt.Sprintf(`
	import java.util.Arrays;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertArrayEquals;


public class MainTest {

	private final Main main = new Main();

	@Test
	public void testFunc() {
			try {
					%s result = main.%s(%s);
					%s(%s, result);
			} catch (AssertionError e) {
					System.out.print("Expected %s but got ");
					%s
					throw e; 
			}
	}
}`, modifier, funcName, convertedInput, assert, convertedOutput, expectedOutput, print)

	testFile, err := createTempFile(testCode, "MainTest", "java")
	if err != nil {
		return "", err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("Error creating Docker client: %v", err)
	}

	ctx := context.Background()

	funcTar, err := createTar(funcFile, "Main.java")
	if err != nil {
		return "", fmt.Errorf("Error creating TAR for function file: %v", err)
	}

	testTar, err := createTar(testFile, "MainTest.java")
	if err != nil {
		return "", fmt.Errorf("Error creating TAR for test file: %v", err)
	}

	containerName := "my-container-" + time.Now().Format("20060102150405")

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "my-java-test-image",
	}, nil, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("Error creating container: %v", err)
	}

	if err := cli.CopyToContainer(ctx, resp.ID, "app/src/main/java/", funcTar, types.CopyToContainerOptions{}); err != nil {
		return "", fmt.Errorf("Error copying function file to container: %v", err)
	}

	if err := cli.CopyToContainer(ctx, resp.ID, "app/src/test/java/", testTar, types.CopyToContainerOptions{}); err != nil {
		return "", fmt.Errorf("Error copying test file to container: %v", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("Error starting container: %v", err)
	}

	err = deployToKubernetes("my-java-test-image")
	if err != nil {
		return "", fmt.Errorf("Error deploying to Kubernetes: %v", err)
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
	Output    string `json:"output"`
	Input    string `json: "input"`
	ExpectedOutput    string `json: "expectedOutput"`
	Comments string `json:"comments"`
}

func RunTests(funcCode string, questionId string, language string) ([]TestResult, error) {
	question, err := GetQuestionByID(questionId)
	if err != nil {
		return nil, fmt.Errorf("Error fetching question: %v", err)
	}

	var results []TestResult
	failureRegex := regexp.MustCompile(`got (\S.*\S)`)
	failedKeywords := []string{"failed", "FAILED"}

	for i, test := range question.Tests {
		var out string
		var err error
		if language == "java" {
			out, err = runTestJava(funcCode, test.Input, test.ExpectedOutput)
		} else {
			out, err = runTestPython(funcCode, test.Input, test.ExpectedOutput)
		}
		passed := true
		var comments string
		output := ""

		if err != nil {
			passed = false
			comments = err.Error()
		} else {
			for _, keyword := range failedKeywords {
				if strings.Contains(strings.ToLower(out), keyword) {
					passed = false
					break
				}
			}

			allMatches := failureRegex.FindAllStringSubmatch(out, -1)
			if len(allMatches) >= 2 {
				match := allMatches[1] 
				if match != nil {
					parts := strings.SplitN(match[0], " ", 2)
					if len(parts) > 1 {
						output = parts[1]
					}
					comments = fmt.Sprintf("Test failed for input %s: output indicates failure: %s", test.Input, match[0])
				} else {
					comments = fmt.Sprintf("Test failed for input %s", test.Input)
				}
			} else if len(allMatches) == 1 {
				match := allMatches[0] 
				if match != nil {
					parts := strings.SplitN(match[0], " ", 2)
					if len(parts) > 1 {
						output = parts[1]
					}
					comments = fmt.Sprintf("Test failed for input %s: output indicates failure: %s", test.Input, match[0])
				} else {
					comments = fmt.Sprintf("Test failed for input %s", test.Input)
				}
			} else {
				comments = "Test passed"
			}
		}
		if output == "" {
			output = test.ExpectedOutput
		}
		results = append(results, TestResult{
			TestNumber:     i + 1,
			Passed:         passed,
			Comments:       comments,
			Input:          test.Input,
			ExpectedOutput: test.ExpectedOutput,
			Output:         output,
		})
	}
	return results, nil
}
