package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	docs "github.com/guillaume/q-router-go/docs"
	apihandler "github.com/guillaume/q-router-go/internal/apihandler"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @BasePath /api/v1

// PingExample godoc
// @Tags HealthCheck
// @Produce json
// @Success 200
// @Router /ping [get]
func GetPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func main() {
	r := gin.Default()

	docs.SwaggerInfo.BasePath = "/api/v1"

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", GetPing)
		v1.POST("/task", apihandler.PostTask)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run()
}
