package apihandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type TaskDefinition struct {
	N *int `json:"n" binding:"required"`
}

// Post a task on queue
// @Tags Task management
// @Accept json
// @Produce json
// @Param task body TaskDefinition true "Task to enqueue"
// @Success 200
// @Router /task [post]
func PostTask(c *gin.Context) {
	var json TaskDefinition
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"n": *json.N,
	})
}
