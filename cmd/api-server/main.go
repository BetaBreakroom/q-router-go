package main

import (
	"log"
	"net/http"

	"q-router-go/docs"
	"q-router-go/internal/apihandler"
	"q-router-go/internal/learner"
	"q-router-go/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

func initWorkerEnvironment() *worker.Environment {
	// Initialize RL agent
	workerCount := 4
	agent := learner.NewRLAgent(0.5, 0.5, 0.1, workerCount)

	// Initialize worker environment
	env := worker.NewEnvironment(workerCount, agent)
	env.SleepPolicies[0] = worker.CreateSleepPolicy(500, 500, 0, 0.0)
	env.SleepPolicies[1] = worker.CreateSleepPolicy(40, 60, 0, 0.0)
	env.SleepPolicies[2] = worker.CreateSleepPolicy(0, 100, 0, 0.0)
	env.SleepPolicies[3] = worker.CreateSleepPolicy(50, 50, 800, 0.1)

	log.Println("Created agent, worker count:", agent.WorkerCount)

	// Start workers and environment monitoring
	env.StartWorkers()
	env.Start()

	return env
}

func main() {
	r := gin.Default()

	docs.SwaggerInfo.BasePath = "/api/v1"

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	env := initWorkerEnvironment()
	taskHandler := apihandler.NewTaskHandler(env, &upgrader)

	go taskHandler.HandleBroadcastStatistics()

	r.Static("/public", "./static/")

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", GetPing)
		v1.POST("/task", taskHandler.PostTask)
		v1.GET("/statistics", taskHandler.GetStatistics)
		v1.GET("/ws", taskHandler.HandleWebSocket)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run()
}
