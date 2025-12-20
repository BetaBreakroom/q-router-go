package apihandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guillaume/q-router-go/internal/fibo"
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
	if json.N == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "n is required"})
		return
	}
	if *json.N < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "n must be non-negative"})
		return
	}
	if *json.N > 40 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "n must be less than or equal to 40"})
		return
	}
	res, err := fibo.Fibo(*json.N)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"fibo(n)": res,
	})
}
