package worker

import (
	"fmt"
	"math/rand/v2"
	learner "q-router-go/internal/learner"
	"strings"
	"sync"
	"time"
)

type Environment struct {
	WorkerCount   int
	Queues        []chan Task
	Agent         *learner.RLAgent
	Sync          sync.WaitGroup
	SleepPolicies []func()
}

type Task struct {
	Payload string
	Result  chan string
}

func NewEnvironment(workerCount int, agent *learner.RLAgent) *Environment {
	return &Environment{
		WorkerCount:   workerCount,
		Queues:        make([]chan Task, workerCount),
		Agent:         agent,
		Sync:          sync.WaitGroup{},
		SleepPolicies: make([]func(), workerCount),
	}
}

func (env *Environment) StartWorkers() {
	for i := range env.WorkerCount {
		env.Queues[i] = make(chan Task, 10)
		env.Sync.Add(1)
		go env.WorkerFunc(i, env.Queues[i])
	}
}

func (env *Environment) StopWorkers() {
	for i := range env.WorkerCount {
		close(env.Queues[i])
	}
	env.Sync.Wait()
}

func (env *Environment) EnqueueTask(workerIndex int, task Task) {
	env.Queues[workerIndex] <- task
}

func (env *Environment) GetQueueLengths() []int {
	lengths := make([]int, env.WorkerCount)
	for i := 0; i < env.WorkerCount; i++ {
		lengths[i] = len(env.Queues[i])
	}
	return lengths
}

// Returns the state string based on the environment's queue lengths and the task payload
func (env *Environment) GetState(payload string) string {
	sb := strings.Builder{}

	for i, q := range env.Queues {
		lenQ := len(q) // Current items in channel
		capQ := cap(q) // Max capacity of channel

		status := "BUSY"

		if lenQ == 0 {
			status = "IDLE"
		} else if lenQ >= capQ {
			status = "FULL"
		} else if float64(lenQ) > float64(capQ)*0.7 {
			status = "HEAVY" // >70% capacity
		}

		sb.WriteString(fmt.Sprintf("W%d:%s_", i, status))
	}

	sb.WriteString(fmt.Sprintf("P:%s", payload))

	return sb.String()
}

func CreateSleepPolicy(sleepMinMs int, sleepMaxMs int, lockMs int, lockProba float64) func() {
	return func() {
		jitter := rand.IntN(10)
		if rand.Float64() < lockProba {
			time.Sleep(time.Duration(lockMs+jitter) * time.Millisecond)
		} else if sleepMaxMs-sleepMinMs > 0 {
			time.Sleep(time.Duration(rand.IntN(sleepMaxMs-sleepMinMs)+sleepMinMs+jitter) * time.Millisecond)
		} else {
			time.Sleep(time.Duration(sleepMinMs+jitter) * time.Millisecond)
		}
	}
}

func (env *Environment) WorkerFunc(workerId int, queue <-chan Task) {
	defer env.Sync.Done()
	for task := range queue {
		start := time.Now()
		fmt.Printf("Worker %d start processing task: %s\n", workerId, task.Payload)
		env.SleepPolicies[workerId]()
		elapsed := time.Now().Sub(start)
		task.Result <- fmt.Sprintf("Processed by worker %d: %s", workerId, task.Payload)
		fmt.Printf("Worker %d finished processing task: %s in %d ms.\n", workerId, task.Payload, elapsed.Milliseconds())
	}
}
