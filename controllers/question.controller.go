package questioncontroller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"LeetCode-server/services"
	"LeetCode-server/models"
)

type QuestionController struct{}

// HandleGet handles GET requests for retrieving questions
func (c *QuestionController) HandleGet(ctx *gin.Context) {
	questions, err := questionService.GetAllQuestions()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, questions)
}

// HandlePost handles POST requests for creating a new question
func (c *QuestionController) HandlePost(ctx *gin.Context) {
	var newQuestion question.Question
	if err := ctx.ShouldBindJSON(&newQuestion); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	createdQuestion, err := questionService.CreateQuestion(newQuestion.Description, newQuestion.Level, newQuestion.Tests)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, createdQuestion)
}

// HandlePut handles PUT requests for updating an existing question
func (c *QuestionController) HandlePut(ctx *gin.Context) {
	id := ctx.DefaultQuery("id", "")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing question ID"})
		return
	}

	var updatedQuestion question.Question
	if err := ctx.ShouldBindJSON(&updatedQuestion); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updatedQuestionResult, err := questionService.UpdateQuestion(id, updatedQuestion.Description, updatedQuestion.Level, updatedQuestion.Tests)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedQuestionResult)
}

// HandleDelete handles DELETE requests for deleting a question
func (c *QuestionController) HandleDelete(ctx *gin.Context) {
	id := ctx.DefaultQuery("id", "")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing question ID"})
		return
	}

	deleteResult, err := questionService.DeleteQuestion(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, deleteResult)
}

// HandleRunTests handles POST requests to run tests on a solution
func (c *QuestionController) HandleRunTests(ctx *gin.Context) {
	var solution struct {
		Solution string `json:"solution"`
	}

	if err := ctx.ShouldBindJSON(&solution); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	questionService.RunTests(solution.Solution, "5", "10")

	ctx.JSON(http.StatusOK, gin.H{"message": "Tests run successfully"})
}

// RegisterHandlers registers all routes for the question controller
func (c *QuestionController) RegisterHandlers(router *gin.Engine) {
	router.GET("/questions", c.HandleGet)
	router.POST("/questions", c.HandlePost)
	router.PUT("/questions", c.HandlePut)
	router.DELETE("/questions", c.HandleDelete)
	router.POST("/questions/runTests", c.HandleRunTests)
}
