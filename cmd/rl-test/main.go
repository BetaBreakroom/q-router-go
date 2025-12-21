package main

import (
	"fmt"
	learner "q-router-go/internal/learner"
	worker "q-router-go/internal/worker"
	"sync"
	"time"
)

func taskHandler(payload string, env *worker.Environment, wg *sync.WaitGroup, selectedWorkers chan<- int) {
	defer wg.Done()

	start := time.Now()

	state := env.GetState(payload)
	//fmt.Printf("State: %s\n", state)

	// Choose worker based on RL agent
	workerIndex := env.Agent.ChooseWorker(state)
	fmt.Printf("Selected worker: %d\n", workerIndex)

	// Enqueue task
	resultChannel := make(chan string)
	env.EnqueueTask(workerIndex, worker.Task{Payload: payload, Result: resultChannel})

	// Wait for result
	<-resultChannel
	//fmt.Printf("Task result: %s\n", result)
	close(resultChannel)

	duration := time.Now().Sub(start)
	fmt.Printf("Task processed in %d ms.\n", duration.Milliseconds())

	// Learn RLAgent based on the observed reward
	nextState := env.GetState(payload)
	reward := 1.0 / float64(duration.Seconds())
	env.Agent.Learn(state, nextState, workerIndex, reward)

	selectedWorkers <- workerIndex
}

func main() {
	workerCount := 4
	agent := learner.NewRLAgent(0.1, 0.5, 0.1, workerCount)
	env := worker.NewEnvironment(workerCount, agent)

	env.SleepPolicies[0] = worker.CreateSleepPolicy(500, 500, 0, 0.0)
	env.SleepPolicies[1] = worker.CreateSleepPolicy(50, 50, 0, 0.0)
	env.SleepPolicies[2] = worker.CreateSleepPolicy(200, 200, 0, 0.0)
	env.SleepPolicies[3] = worker.CreateSleepPolicy(50, 50, 200, 0.5)

	fmt.Println("Created agent, worker count:", agent.WorkerCount)

	env.StartWorkers()

	var requestWg sync.WaitGroup
	selectedWorkers := make(chan int, 1000)

	start := time.Now()

	const numTasks = 100
	for _ = range numTasks {
		requestWg.Add(1)
		payload := "TASK"
		go taskHandler(payload, env, &requestWg, selectedWorkers)
		time.Sleep(100 * time.Millisecond)
	}

	requestWg.Wait()
	close(selectedWorkers)

	env.StopWorkers()

	countWorkers := make([]int, workerCount)
	for workerId := range selectedWorkers {
		countWorkers[workerId]++
	}

	for i, count := range countWorkers {
		fmt.Printf("Worker %d was selected %f%%.\n", i, (float64(count)/float64(numTasks))*100)
	}

	elapsed := time.Now().Sub(start)
	fmt.Printf("Executed %d tasks in %d ms.\n", numTasks, elapsed.Milliseconds())

	fmt.Println("Final Q-Table:")
	for state, qValues := range agent.Table {
		fmt.Printf("State: %s, Q-Values: %v\n", state, qValues)
	}
}
