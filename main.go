package main

import (
	"github.com/gin-gonic/gin"
	"LeetCode-server/controllers"
	"LeetCode-server/services"
	"time"
	 "github.com/gin-contrib/cors"
)

func main() {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"}, 
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
   }))
	controller := &questioncontroller.QuestionController{}
	service.Init()
	controller.RegisterHandlers(r)

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}