package main

import (
	"github.com/gin-gonic/gin"
	"LeetCode-server/controllers"
	"LeetCode-server/services"
)

func main() {
	r := gin.Default()

	controller := &questioncontroller.QuestionController{}
	questionService.Init()
	controller.RegisterHandlers(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
