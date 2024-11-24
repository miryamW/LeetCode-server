package service

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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// createTempFile creates a temporary file with the given content, prefix, and extension.
// It returns the file name and any error encountered during file creation.
func createTempFile(content, prefix, ext string) (string, error) {
	fileName := prefix + "." + ext
	file, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	if _, err := file.Write([]byte(content)); err != nil {
		return "", fmt.Errorf("error writing to file: %v", err)
	}

	return file.Name(), nil
}

// findErrorPython processes the output of a Python test and finds any error messages.
// It searches for Python error messages and returns the first match that is not an "AssertionError".
func findErrorPython(output string) string {
	lines := strings.Split(output, "\n")
	re := regexp.MustCompile(`\w+Error:.*$`)

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if re.MatchString(line) {
			errorRe := regexp.MustCompile(`(\w+Error:.*)`)
			match := errorRe.FindString(line)
			if match != "" && !strings.HasPrefix(match, "AssertionError:") {
				return fmt.Sprint("error - ",match)
			}
		}
	}

	return ""
}

// findErrorJava processes the output of a Java test and finds any error messages.
// It returns the compilation or runtime error message found in the Java output.
func findErrorJava(output string) string {
	compilationErrorRegex := regexp.MustCompile(`/app/src/main/java/Main\.java:\[(\d+),(\d+)\] (.*)`)
	runtimeErrorRegex := regexp.MustCompile(`java\.lang\.\S+: (.+)\n\s+at .*\((.*):(\d+)\)`)
	compilationErrorMatch := compilationErrorRegex.FindStringSubmatch(output)
	runtimeErrorMatch := runtimeErrorRegex.FindStringSubmatch(output)

	if len(compilationErrorMatch) > 1 {
			line := compilationErrorMatch[1]
			column := compilationErrorMatch[2]
			errorMessage := compilationErrorMatch[3]
			return fmt.Sprintf("compilation error - [%s,%s] %s", line, column, errorMessage)
  } else if len(runtimeErrorMatch) > 0 {
			return fmt.Sprintf("run time error - %s", runtimeErrorMatch[1])
  }
	return ""
}

// extractFuncNamePython extracts the function name from a Python function's code.
// It returns the function name or an error if the name cannot be found.
func extractFuncNamePython(funcCode string) (string, error) {
	re := regexp.MustCompile(`def\s+(\w+)\s*\(.*\)\s*:`)
	matches := re.FindStringSubmatch(funcCode)
	if len(matches) < 1 {
		return "", fmt.Errorf("Could not find function name in the provided code")
	}
	return matches[1], nil
}

// extractFuncNameJava extracts the function name from a Java function's code based on the return type.
// It returns the function name or an error if the name cannot be found.
func extractFuncNameJava(funcCode string, returnType string) (string, error) {
	re := regexp.MustCompile(fmt.Sprintf(`%s\s+(\w+)\s*\(`, regexp.QuoteMeta(returnType)))
	matches := re.FindStringSubmatch(funcCode)
	if len(matches) < 1{
		return "", fmt.Errorf("Could not find function name after return type '%s' in the code", returnType)
	}

	return matches[1], nil
}

// extractReturnType extracts the return type (e.g., int, string) from the function code.
// It returns the return type or an error if no valid return type is found.
func extractReturnType(funcCode string) (string, error) {
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

// Function to capitalize "true" or "false" in a comma-separated string
func capitalizeBooleans(input string) string {
	// Split the input string by commas
	parts := strings.Split(input, ",")

	// Iterate through each part
	for i, part := range parts {
		// Trim any whitespace around the part
		part = strings.TrimSpace(part)

		// Check if the part is "true" or "false"
		if part == "true" {
			parts[i] = "True"
		} else if part == "false" {
			parts[i] = "False"
		} else {
			parts[i] = part
		}
	}

	// Join the parts back together with commas
	return strings.Join(parts, ", ")
}


// convertInputOutputArray converts the input and output array or matrix string representations into Java array syntax.
// It returns the converted string or an error if the conversion fails.
func convertInputOutputArray(input string) (string, error) {
	arrayPattern := regexp.MustCompile(`\[\s*(\d+(\.\d+)?|"[^"]*")(,\s*(\d+(\.\d+)?|"[^"]*"))*\s*\]`)
	matrixPattern := regexp.MustCompile(`\[\s*\[\s*(\d+(\.\d+)?|"[^"]*")(,\s*(\d+(\.\d+)?|"[^"]*"))*\s*\](,\s*\[\s*(\d+(\.\d+)?|"[^"]*")(,\s*(\d+(\.\d+)?|"[^"]*"))*\s*\])*\s*\]`)

	elementTypeMap := map[string]string{
		"int":    `^\d+$`,
		"double": `^\d+\.\d+$`,
		"String": `^".*"$`,
	}

	getElementType := func(element string) string {
		for key, pattern := range elementTypeMap {
			match, _ := regexp.MatchString(pattern, element)
			if match {
				return key
			}
		}
		return "Unsupported"
	}

	convertArray := func(array string) string {
		innerContent := array[1 : len(array)-1]
		elements := strings.Split(innerContent, ",")
		elementType := getElementType(strings.TrimSpace(elements[0]))
		return fmt.Sprintf("new %s[]{%s}", elementType, strings.Join(elements, ","))
	}

	convertMatrix := func(matrix string) string {
		innerContent := matrix[2 : len(matrix)-2]
		rows := strings.Split(innerContent, "],[")
		elementType := getElementType(strings.TrimSpace(strings.Split(rows[0], ",")[0]))
		for i, row := range rows {
			rows[i] = fmt.Sprintf("{%s}", row)
		}
		return fmt.Sprintf("new %s[][]{%s}", elementType, strings.Join(rows, ", "))
	}

	result := input

	matrixMatches := matrixPattern.FindAllStringIndex(input, -1)
	for i := len(matrixMatches) - 1; i >= 0; i-- {
		match := input[matrixMatches[i][0]:matrixMatches[i][1]]
		replacement := convertMatrix(match)
		result = result[:matrixMatches[i][0]] + replacement + result[matrixMatches[i][1]:]
	}

	arrayMatches := arrayPattern.FindAllStringIndex(result, -1)
	for i := len(arrayMatches) - 1; i >= 0; i-- {
		match := result[arrayMatches[i][0]:arrayMatches[i][1]]
		if !matrixPattern.MatchString(match) {
			replacement := convertArray(match)
			result = result[:arrayMatches[i][0]] + replacement + result[arrayMatches[i][1]:]
		}
	}

	return result, nil
}

// runTestJava runs a Java test based on the provided function code, input, and expected output.
// It prepares the environment, creates necessary files, and runs the test in a Kubernetes pod.
func runTestJava(funcCode, input, expectedOutput string) (string, error) {
	dirName := "src" +  uuid.New().String()
	err := os.MkdirAll(dirName + "/main/java", 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	err = os.MkdirAll(dirName + "/test/java", 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	defer func() {
		err := os.RemoveAll(dirName)
		if err != nil {
			fmt.Printf("failed to remove directory: %v\n", err)
		}
	}()

	_, err = createTempFile(funcCode, dirName + "/main/java/Main", "java")
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

	modifier, err := extractReturnType(funcCode)
	if err != nil {
		return "", err
	}

	funcName, err := extractFuncNameJava(funcCode, modifier)
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
					System.out.print("Expected but got ");
					%s
					throw e; 
			}
	}
}`, modifier, funcName, convertedInput, assert, convertedOutput, print)

	_, err = createTempFile(testCode, dirName + "/test/java/MainTest", "java")
	if err != nil {
		return "", err
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		return "", fmt.Errorf("error","cannot connect to k8s KUBECONFIG is not exist")
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

	for {
		podStatus, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get pod status: %v", err)
		}
		if podStatus.Status.Phase == corev1.PodRunning {
			break
		}
		time.Sleep(5 * time.Second)
	}

	cmd := exec.Command("kubectl", "cp", dirName, podName+":/app/src/")
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to copy files: %v", err)
	}

	cmd = exec.Command("kubectl", "exec", "-it", podName, "--", "mvn", "test")
	output, _ := cmd.CombinedOutput()

	err = clientset.CoreV1().Pods("default").Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("Failed to delete pod: %v", err)
	}

	return string(output), nil
}

// runTestPython runs a Python test based on the provided function code, input, and expected output.
// It prepares the environment, creates necessary files, and runs the test in a Kubernetes pod
func runTestPython(funcCode, input, expectedOutput string) (string, error) {
	dirName := "my_tests" + uuid.New().String()
	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	defer func() {
		err := os.RemoveAll(dirName)
		if err != nil {
			fmt.Printf("failed to remove directory: %v\n", err)
		}
	}()

	funcName, err := extractFuncNamePython(funcCode)
	if err != nil {
		return "", err
	}

	_, err = createTempFile(funcCode, dirName + "/func", "py")
	if err != nil {
		return "", err
	}
	formatedInput:= capitalizeBooleans(input)
	formatedOutput:= capitalizeBooleans(expectedOutput)

	testCode := fmt.Sprintf(`
from func import *

def test():
	result = %s(%s)
	assert result == %s, f"Expected but got {result}"
`, funcName, formatedInput, formatedOutput)

	_, err = createTempFile(testCode, dirName + "/test_func", "py")
	if err != nil {
		return "", err
	}

	kubeconfig := os.Getenv("KUBECONFIG")
  if kubeconfig == "" {
		return "", fmt.Errorf("error","cannot connect to k8s KUBECONFIG is not exist")
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

  for {
	  podStatus, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
	  if err != nil {
		  log.Fatalf("Failed to get pod status: %v", err)
	  }
	  if podStatus.Status.Phase == corev1.PodRunning {
		  break
	  }
	  time.Sleep(5 * time.Second) 
  }

  cmd := exec.Command("kubectl", "cp", dirName, podName+":/app/my_tests/")
  err = cmd.Run()
  if err != nil {
	  log.Fatalf("Failed to copy files: %v", err)
  }

  cmd = exec.Command("kubectl", "exec", "-it", podName, "--", "pytest", "/app/my_tests")
  output, _ := cmd.CombinedOutput()

  err = clientset.CoreV1().Pods("default").Delete(context.TODO(), podName, metav1.DeleteOptions{})
  if err != nil {
	  log.Fatalf("Failed to delete pod: %v", err)
  }

  return string(output), nil
}

type runTest func(string, string, string) (string, error)
type findError func(output string) string 

// The RunTests function executes a series of tests for a given function code in a specified programming language,
// comparing the actual output with the expected output. 
//It returns the results, including success/failure status, error messages, and any discrepancies found during the tests.
func RunTests(funcCode string, questionId string, language string) ([]models.TestResult, error) {
	runTestMap := map[string]runTest{
		"java":     runTestJava,
		"python": runTestPython,
	}

	findErrorMap := map[string]findError{
		"java":   findErrorJava,
		"python": findErrorPython,
	}
	
	question, err := GetQuestionByID(questionId)
	if err != nil {
			return nil, fmt.Errorf("error fetching question: %v", err)
	}

	var results []models.TestResult
	failureRegex := regexp.MustCompile(`got (\S.*\S?)`)
	failedKeywords := []string{"failed", "FAILED"}

	//runAllTests
	for i, test := range question.Tests {
			out, err := runTestMap[language] (funcCode, test.Input, test.ExpectedOutput)
			var errors []models.ErrorLine
			var comments string
			passed := true
			output := ""
			if err != nil {
				passed = false
				comments = err.Error()
			} else {
				//find compilation / run time errors
				errorMessage := findErrorMap[language](out)
				if errorMessage != "" {
					passed = false
					comments = errorMessage
				}
			}
			if comments == "" {
				//find another failures
				for _, keyword := range failedKeywords {
					if strings.Contains(strings.ToLower(out), keyword) {
						passed = false
						break
				  }
				}
				var match []string
				//find the wronע output
				allMatches := failureRegex.FindAllStringSubmatch(out, -1)
				if len(allMatches) >= 2 {
					match = allMatches[1]
				} else if len(allMatches) == 1 {
					match = allMatches[0]
				}
				if match != nil {
					parts := strings.SplitN(match[0], " ", 2)
					if len(parts) > 1 {
						output = parts[1]
					}
					comments = fmt.Sprintf("Test failed for input %s: output indicates failure: %s", test.Input, match[0])
				} else if passed == false{
					comments = fmt.Sprintf("Test failed for input %s", test.Input)
			  }
		  }

			//put the correct output
	    if output == "" && passed {
			  output = test.ExpectedOutput
	    }

			//append to results array
	    results = append(results, models.TestResult{
			  TestNumber:     i + 1,
			  Passed:         passed,
			  Comments:       comments,
			  Input:          test.Input,
			  ExpectedOutput: test.ExpectedOutput,
			  Output:         output,
			  Errors:         errors,
		  })
	}

	return results, nil
}



