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
	questions, err := service.GetAllQuestions()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, questions)
}

// HandleGetByID handles GET requests for retrieving a question by ID
func (c *QuestionController) HandleGetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing question ID"})
		return
	}

	question, err := service.GetQuestionByID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if question == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	ctx.JSON(http.StatusOK, question)
}

// HandlePost handles POST requests for creating a new question
func (c *QuestionController) HandlePost(ctx *gin.Context) {
	var newQuestion models.Question
	if err := ctx.ShouldBindJSON(&newQuestion); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	createdQuestion, err := service.CreateQuestion(newQuestion.Title, newQuestion.Description, newQuestion.Level, newQuestion.Tests, newQuestion.InputTypes, newQuestion.OutputType)
	if err != nil {
		if err.Error() == "Question must contain title & description & level & at least one test" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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

	var updatedQuestion models.Question
	if err := ctx.ShouldBindJSON(&updatedQuestion); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updatedQuestionResult, err := service.UpdateQuestion(id, updatedQuestion.Title, updatedQuestion.Description, updatedQuestion.Level, updatedQuestion.Tests, updatedQuestion.InputTypes, updatedQuestion.OutputType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedQuestionResult)
}

// HandleDelete handles DELETE requests for deleting a question
func (c *QuestionController) HandleDelete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing question ID"})
		return
	}

	deleteResult, err := service.DeleteQuestion(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error":"Question not exist"})
		return
	}

	ctx.JSON(http.StatusOK, deleteResult)
}

// HandleRunTests handles POST requests to run tests on a solution
func (c *QuestionController) HandleRunTests(ctx *gin.Context) {
	var solution struct {
		Id string `json:"id"`
		Solution string `json:"solution"`
		Language string `json:"language"`
	}

	if err := ctx.ShouldBindJSON(&solution); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	out, err := service.RunTests(solution.Solution, solution.Id, solution.Language)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{"message": out})
}

// RegisterHandlers registers all routes for the question controller
func (c *QuestionController) RegisterHandlers(router *gin.Engine) {
	router.GET("/questions", c.HandleGet)
	router.GET("/questions/:id", c.HandleGetByID)
	router.POST("/questions", c.HandlePost)
	router.PUT("/questions", c.HandlePut)
	router.DELETE("/questions/:id", c.HandleDelete)
	router.POST("/questions/runTests", c.HandleRunTests)
}
