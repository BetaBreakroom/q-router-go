package main

import (
	"fmt"
	"log"
	"q-router-go/internal/learner"
	"q-router-go/internal/worker"
	"sync"
	"sync/atomic"
	"time"
)

func taskHandler(payload string, env *worker.Environment, wg *sync.WaitGroup, count *atomic.Int64, selectedWorkers chan<- int) {
	defer func() {
		wg.Done()
		count.Add(-1)
		if err := recover(); err != nil {
			fmt.Println("!!!!!work failed:", err)
		}
	}()

	start := time.Now()

	state := env.GetState(payload)
	//fmt.Printf("State: %s\n", state)

	// Choose worker based on RL agent
	availableWorkers := env.GetAvailableWorkers()
	if len(availableWorkers) == 0 {
		log.Println("No available workers!")
		selectedWorkers <- -1
	} else {
		workerIndex := env.Agent.ChooseWorker(state, availableWorkers)
		if workerIndex == -1 {
			log.Println("RL Agent could not select a worker!")
			selectedWorkers <- -1
		} else {
			log.Printf("Selected worker: %d\n", workerIndex)

			// Enqueue task
			resultChannel := make(chan string)
			isSubmitted := env.EnqueueTask(workerIndex, worker.Task{Payload: payload, Result: resultChannel})

			var reward float64

			if isSubmitted {
				// Wait for result
				<-resultChannel
				//fmt.Printf("Task result: %s\n", result)
				close(resultChannel)

				duration := time.Now().Sub(start)
				log.Printf("Task processed in %d ms.\n", duration.Milliseconds())

				reward = 1.0 / float64(duration.Seconds())
				selectedWorkers <- workerIndex
			} else {
				log.Printf("Worker %d queue is full!\n", workerIndex)
				reward = -50.0
				selectedWorkers <- -1
			}

			// Learn RLAgent based on the observed reward
			nextState := env.GetState(payload)
			env.Agent.Learn(state, nextState, workerIndex, reward)
		}
	}
}

func main() {
	workerCount := 4
	agent := learner.NewRLAgent(0.5, 0.5, 0.1, workerCount)
	env := worker.NewEnvironment(workerCount, agent)

	env.SleepPolicies[0] = worker.CreateSleepPolicy(500, 500, 0, 0.0)
	env.SleepPolicies[1] = worker.CreateSleepPolicy(40, 60, 0, 0.0)
	env.SleepPolicies[2] = worker.CreateSleepPolicy(0, 100, 0, 0.0)
	env.SleepPolicies[3] = worker.CreateSleepPolicy(50, 50, 800, 0.1)

	log.Println("Created agent, worker count:", agent.WorkerCount)

	env.StartWorkers()

	var requestWg sync.WaitGroup
	var tasksSubmitted atomic.Int64
	selectedWorkers := make(chan int, 1000)

	start := time.Now()

	const numTasks = 1000
	for _ = range numTasks {
		requestWg.Add(1)
		tasksSubmitted.Add(1)
		payload := "TASK"
		go taskHandler(payload, env, &requestWg, &tasksSubmitted, selectedWorkers)
		time.Sleep(20 * time.Millisecond)
	}

	log.Println("All tasks submitted, waiting for completion...")
	log.Println("Current tasks in progress:", tasksSubmitted.Load())
	requestWg.Wait()
	close(selectedWorkers)

	log.Println("All tasks completed.")
	env.StopWorkers()

	countWorkers := make(map[int]int)
	for workerId := range selectedWorkers {
		countWorkers[workerId]++
	}

	for i, count := range countWorkers {
		log.Printf("Worker %d was selected %f%%.\n", i, (float64(count)/float64(numTasks))*100)
	}

	elapsed := time.Now().Sub(start)
	log.Printf("Executed %d tasks in %d ms.\n", numTasks, elapsed.Milliseconds())

	log.Println("Final Q-Table:")
	for state, qValues := range agent.Table {
		log.Printf("State: %s, Q-Values: %v\n", state, qValues)
	}
}
