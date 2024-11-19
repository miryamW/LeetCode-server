package questionService

import (
	"LeetCode-server/models"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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


func runTestPython(funcCode, input, expectedOutput string) (string, error) {
	  err := os.MkdirAll("my_tests", 0755)
	  if err != nil {
		  return "", fmt.Errorf("failed to create directory: %w", err)
    }
		defer func() {
			err := os.RemoveAll("my_tests")
			if err != nil {
				fmt.Printf("failed to remove directory: %v\n", err)
			}
		}()
		funcName, err := extractFuncName(funcCode)
		if err != nil {
			return "", err
		}

		_, err = createTempFile(funcCode, "my_tests/func", "py")
		if err != nil {
			return "", err
		}
	
		testCode := fmt.Sprintf(`
from func import *
	
def test():
		result = %s(%s)
		assert result == %s, f"Expected %s but got {result}"
	`, funcName, input, expectedOutput, expectedOutput)
	
		_, err = createTempFile(testCode, "my_tests/test_func", "py")
		if err != nil {
			return "", err
		}
	
		kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = "/home/miryam/.minikube/config/config" 
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Failed to build kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}
	podName := "python-test-pod" + uuid.New().String()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "python-test",
					Image: "miryamw/python-test:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"memory": resource.MustParse("512Mi"),
							"cpu":    resource.MustParse("500m"), 
						},
						Limits: corev1.ResourceList{
							"memory": resource.MustParse("1Gi"), 
							"cpu":    resource.MustParse("1"),   
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever, 
		},
	}

	_, err = clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Failed to create pod: %v", err)
	}
	fmt.Println("Pod created successfully.")

	for {
		podStatus, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get pod status: %v", err)
		}
		if podStatus.Status.Phase == corev1.PodRunning {
			fmt.Println("Pod is running.")
			break
		}
		fmt.Println("Waiting for pod to be in 'Running' state...")
		time.Sleep(5 * time.Second) 
	}

	cmd := exec.Command("kubectl", "cp", "/home/miryam/LeetCode-server/my_tests", podName+":/app/my_tests/")
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to copy files: %v", err)
	}
	fmt.Println("Files copied to pod.")

	cmd = exec.Command("kubectl", "exec", "-it", podName, "--", "pytest", "/app/my_tests")
	output, _ := cmd.CombinedOutput()
	fmt.Println("Test execution finished.")

	err = clientset.CoreV1().Pods("default").Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("Failed to delete pod: %v", err)
	}
	fmt.Println("Pod deleted successfully.")
	return string(output), nil
}

func extractFuncName(funcCode string) (string, error) {
	re := regexp.MustCompile(`def\s+(\w+)\s*\(.*\)\s*:`)
	matches := re.FindStringSubmatch(funcCode)
	fmt.Println(matches)
	if len(matches) < 1 {
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
	err := os.MkdirAll("src/main/java", 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	err = os.MkdirAll("src/test/java", 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	defer func() {
		err := os.RemoveAll("src")
		if err != nil {
			fmt.Printf("failed to remove directory: %v\n", err)
		}
	}()

	_, err = createTempFile(funcCode, "src/main/java/Main", "java")
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
	if convertedOutput == expectedOutput {
		assert = "assertEquals"
		print = fmt.Sprintf("System.out.println(main.%s(%s));", funcName, convertedInput)
	} else {
		assert = "assertArrayEquals"
		print = fmt.Sprintf("System.out.println(Arrays.toString(main.%s(%s)));", funcName, convertedInput)
	}

	testCode := fmt.Sprintf(
		`import java.util.Arrays;
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

	_, err = createTempFile(testCode, "src/test/java/MainTest", "java")
	if err != nil {
		return "", err
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = "/home/miryam/.minikube/config/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Failed to build kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}
	podName := "java-test-pod" + uuid.New().String()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "java-test",
					Image: "miryamw/java-test:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"memory": resource.MustParse("512Mi"),
							"cpu":    resource.MustParse("500m"),
						},
						Limits: corev1.ResourceList{
							"memory": resource.MustParse("1Gi"),
							"cpu":    resource.MustParse("1"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	_, err = clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Failed to create pod: %v", err)
	}
	fmt.Println("Pod created successfully.")

	for {
		podStatus, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get pod status: %v", err)
		}
		if podStatus.Status.Phase == corev1.PodRunning {
			fmt.Println("Pod is running.")
			break
		}
		fmt.Println("Waiting for pod to be in 'Running' state...")
		time.Sleep(5 * time.Second)
	}

	cmd := exec.Command("kubectl", "cp", "/home/miryam/LeetCode-server/src", podName+":/app/src/")
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to copy files: %v", err)
	}
	fmt.Println("Files copied to pod.")

	cmd = exec.Command("kubectl", "exec", "-it", podName, "--", "mvn", "test")
	output, _ := cmd.CombinedOutput()
	fmt.Println("Test execution finished.")

	err = clientset.CoreV1().Pods("default").Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("Failed to delete pod: %v", err)
	}
	fmt.Println("Pod deleted successfully.")
	return string(output), nil
}

type TestResult struct {
	TestNumber  int    `json:"test_number"`
	Passed      bool   `json:"passed"`
	Output    string `json:"output"`
	Input    string `json: "input"`
	ExpectedOutput    string `json: "expectedOutput"`
	Comments string `json:"comments"`
}

func findErrorLine(output string) string {
	lines := strings.Split(output, "\n")

	re := regexp.MustCompile(`\w+Error:.*$`)

	for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			if re.MatchString(line) {
					// בודק אם זה לא "AssertionError"
					if !strings.Contains(line, "AssertionError") {
							// מחזיר את השורה כולה
							return line
					}
			}
	}

	return ""
}

func RunTests(funcCode string, questionId string, language string) ([]TestResult, error) {
	question, err := GetQuestionByID(questionId)
	if err != nil {
			return nil, fmt.Errorf("Error fetching question: %v", err)
	}

	var results []TestResult
	failureRegex := regexp.MustCompile(`got (\S.*\S?)`)
	failedKeywords := []string{"failed", "FAILED"}
	compilationErrorRegex := regexp.MustCompile(`/app/src/main/java/Main.java:\[(\d+,\d+)\] (.+)`)

	for i, test := range question.Tests {
			var out string
			var err error
			if language == "java" {
					out, err = runTestJava(funcCode, test.Input, test.ExpectedOutput)
			} else {
					out, err = runTestPython(funcCode, test.Input, test.ExpectedOutput)
			}
			fmt.Println(out)
			passed := true
			var comments string
			output := ""

			if err != nil {
					passed = false
					comments = err.Error()
			} else {
					if language == "java" {
							compilationErrorMatch := compilationErrorRegex.FindStringSubmatch(out)
							fmt.Println(compilationErrorMatch)
							if len(compilationErrorMatch) > 2 {
									passed = false
									comments = fmt.Sprintf("compilation error - [%s] %s", compilationErrorMatch[1], compilationErrorMatch[2])
							}
					}

					if language == "python" {
						errorMessage := findErrorLine(out)
						 if(errorMessage != ""){
						   passed = false
						   comments = "compilation error - " + errorMessage
						}
					}

					if comments == "" {
							for _, keyword := range failedKeywords {
									if strings.Contains(strings.ToLower(out), keyword) {
											passed = false
											break
									}
							}

							allMatches := failureRegex.FindAllStringSubmatch(out, -1)
							fmt.Println(allMatches)
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
			}
			if output == "" && passed {
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
