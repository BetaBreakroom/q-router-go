package apihandler

import (
	"log"
	"net/http"
	"q-router-go/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type TaskDefinition struct {
	Type *string `json:"type" binding:"required"`
}

type TaskHandler struct {
	Environment *worker.Environment
	Upgrader    *websocket.Upgrader
	Clients     map[*websocket.Conn]bool
}

type WorkerStatisticsResponse struct {
	TaskThroughput      float64 `json:"taskThroughput"`
	TotalProcessedTasks int64   `json:"totalProcessedTasks"`
	TasksPerWorker      []int64 `json:"tasksPerWorker"`
	DismissedTasks      int64   `json:"dismissedTasks"`
}

func NewTaskHandler(env *worker.Environment, upgrader *websocket.Upgrader) *TaskHandler {
	return &TaskHandler{
		Environment: env,
		Upgrader:    upgrader,
		Clients:     make(map[*websocket.Conn]bool),
	}
}

// Post a task on queue
// @Tags Task management
// @Accept json
// @Produce json
// @Param task body TaskDefinition true "Task to enqueue"
// @Success 200
// @Router /task [post]
func (h *TaskHandler) PostTask(c *gin.Context) {
	var json TaskDefinition
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dispatchedIndex, err := h.Environment.HandleTask(*json.Type)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if dispatchedIndex == -1 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "No available workers or all queues are full"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}

// Get worker statistics
// @Tags Task management
// @Produce json
// @Success 200
// @Router /statistics [get]
func (h *TaskHandler) GetStatistics(c *gin.Context) {
	taskCountPerWorker := make([]int64, h.Environment.WorkerCount)
	for i := range h.Environment.WorkerCount {
		taskCountPerWorker[i] = h.Environment.WorkerStatistics.CountTasksPerWorker[i]
	}

	dismissedTasks := h.Environment.WorkerStatistics.CountTasksPerWorker[-1]

	c.JSON(http.StatusOK, gin.H{
		"taskThroughput":      h.Environment.WorkerStatistics.TaskThroughput,
		"totalProcessedTasks": h.Environment.WorkerStatistics.TotalProcessedTasks,
		"countTasksPerWorker": taskCountPerWorker,
		"dismissedTasks":      dismissedTasks,
	})
}

// Websocket endpoint to stream worker statistics
// @Tags Task management
// @Produce json
// @Success 200
// @Router /ws [get]
func (h *TaskHandler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP to WebSocket
	ws, err := h.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Error(err)
		return
	}
	// Register client
	h.Clients[ws] = true
}

func (h *TaskHandler) HandleBroadcastStatistics() {
	for {
		stats := <-h.Environment.BroadcastStatisticsChannel
		for client := range h.Clients {
			err := client.WriteJSON(WorkerStatisticsResponse{
				TaskThroughput:      stats.TaskThroughput,
				TotalProcessedTasks: stats.TotalProcessedTasks,
				TasksPerWorker: func() []int64 {
					taskCounts := make([]int64, h.Environment.WorkerCount)
					for i := range taskCounts {
						taskCounts[i] = stats.CountTasksPerWorker[i]
					}
					return taskCounts
				}(),
				DismissedTasks: stats.CountTasksPerWorker[-1],
			})

			if err != nil {
				log.Printf("WebSocket error: %v", err)
				client.Close()
				delete(h.Clients, client)
			}
		}
	}
}
