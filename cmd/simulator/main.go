package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

type TaskDefinition struct {
	Type string `json:"type"`
}

func main() {
	// Parse command-line flags
	apiURL := flag.String("url", "http://localhost:8080/api/v1/task", "API endpoint URL")
	interval := flag.Duration("interval", 50*time.Millisecond, "Interval between tasks")
	taskType := flag.String("type", "default", "Task type to send")
	count := flag.Int("count", 0, "Number of tasks to send (0 = infinite)")
	flag.Parse()

	log.Printf("Starting task simulator...")
	log.Printf("API URL: %s", *apiURL)
	log.Printf("Interval: %s", *interval)
	log.Printf("Task type: %s", *taskType)
	if *count == 0 {
		log.Printf("Sending tasks indefinitely (press Ctrl+C to stop)")
	} else {
		log.Printf("Sending %d tasks", *count)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	tasksSent := 0
	tasksSuccessful := 0
	tasksFailed := 0
	startTime := time.Now()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for range ticker.C {
		tasksSent++

		task := TaskDefinition{
			Type: *taskType,
		}

		jsonData, err := json.Marshal(task)
		if err != nil {
			log.Printf("Error marshaling task: %v", err)
			tasksFailed++
			continue
		}

		resp, err := client.Post(*apiURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Error sending task %d: %v", tasksSent, err)
			tasksFailed++
			continue
		}

		if resp.StatusCode == http.StatusOK {
			tasksSuccessful++
			if tasksSent%10 == 0 {
				elapsed := time.Since(startTime)
				rate := float64(tasksSuccessful) / elapsed.Seconds()
				log.Printf("Sent %d tasks (%d successful, %d failed) - Rate: %.2f tasks/sec",
					tasksSent, tasksSuccessful, tasksFailed, rate)
			}
		} else {
			tasksFailed++
			body := make([]byte, 1024)
			resp.Body.Read(body)
			log.Printf("Task %d failed with status %d: %s", tasksSent, resp.StatusCode, string(body))
		}
		resp.Body.Close()

		if *count > 0 && tasksSent >= *count {
			break
		}
	}

	elapsed := time.Since(startTime)
	rate := float64(tasksSuccessful) / elapsed.Seconds()
	fmt.Printf("\n=== Simulation Complete ===\n")
	fmt.Printf("Total tasks sent: %d\n", tasksSent)
	fmt.Printf("Successful: %d\n", tasksSuccessful)
	fmt.Printf("Failed: %d\n", tasksFailed)
	fmt.Printf("Duration: %s\n", elapsed)
	fmt.Printf("Average rate: %.2f tasks/sec\n", rate)
}
