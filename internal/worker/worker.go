package worker

import (
	"fmt"
	"log"
	"math/rand/v2"
	"q-router-go/internal/learner"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Environment struct {
	WorkerCount                int
	Queues                     []chan Task
	Agent                      *learner.RLAgent
	Sync                       sync.WaitGroup
	SleepPolicies              []func()
	SelectedWorkers            chan int
	ProcessedTaskCount         atomic.Int64
	WorkerStatistics           WorkerStatistics
	BroadcastStatisticsChannel chan WorkerStatistics
}

const MediumTrafficQueueLengthThreshold = 1
const HighTrafficQueueLengthFraction = 0.5

type Task struct {
	Payload string
	Result  chan string
}

type WorkerStatistics struct {
	TotalProcessedTasks int64
	TaskThroughput      float64
	CountTasksPerWorker map[int]int64
}

func NewEnvironment(workerCount int, agent *learner.RLAgent) *Environment {
	return &Environment{
		WorkerCount:        workerCount,
		Queues:             make([]chan Task, workerCount),
		Agent:              agent,
		Sync:               sync.WaitGroup{},
		SleepPolicies:      make([]func(), workerCount),
		SelectedWorkers:    make(chan int, 1000),
		ProcessedTaskCount: atomic.Int64{},
		WorkerStatistics: WorkerStatistics{
			TotalProcessedTasks: 0,
			TaskThroughput:      0.0,
			CountTasksPerWorker: make(map[int]int64),
		},
		BroadcastStatisticsChannel: make(chan WorkerStatistics),
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

func (env *Environment) BuildStatisticsInBackground(selectedWorkers <-chan int, broadcastStatisticsChannel chan<- WorkerStatistics) {
	lastTimestamp := time.Now()
	lastTaskCount := 0
	for selectedWorker := range selectedWorkers {
		log.Printf("Task assigned to worker %d\n", selectedWorker)
		env.WorkerStatistics.CountTasksPerWorker[selectedWorker]++
		env.WorkerStatistics.TotalProcessedTasks = env.ProcessedTaskCount.Load()

		// Update throughput every second
		if elapsed := time.Now().Sub(lastTimestamp); elapsed >= 1*time.Second {
			currentTaskCount := int(env.ProcessedTaskCount.Load())
			env.WorkerStatistics.TaskThroughput = float64(currentTaskCount-lastTaskCount) / elapsed.Seconds()
			lastTaskCount = currentTaskCount
			lastTimestamp = time.Now()
			log.Printf("Updated statistics: %+v\n", env.WorkerStatistics)

			select {
			case broadcastStatisticsChannel <- env.WorkerStatistics:
			default:
				log.Println("Broadcast channel is full, skipping statistics update.")
			}
		}
	}
}

func (env *Environment) Start() {
	go env.BuildStatisticsInBackground(env.SelectedWorkers, env.BroadcastStatisticsChannel)
}

func (env *Environment) Stop() {
	close(env.SelectedWorkers)
}

func (env *Environment) EnqueueTask(workerIndex int, task Task) bool {
	select {
	case env.Queues[workerIndex] <- task:
		return true // Channel not full
	default:
		return false // Channel full
	}
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

		status := "LOW"

		if lenQ >= capQ {
			status = "FULL"
		} else if float64(lenQ) > float64(capQ)*HighTrafficQueueLengthFraction {
			status = "HIGH" // >50% capacity
		} else if lenQ > MediumTrafficQueueLengthThreshold {
			status = "MED" // >1 item, medium traffic
		}

		sb.WriteString(fmt.Sprintf("W%d:%s_", i, status))
	}

	sb.WriteString(fmt.Sprintf("P:%s", payload))

	return sb.String()
}

func (env *Environment) GetAvailableWorkers() map[int]bool {
	availableWorkers := make(map[int]bool)
	for i := 0; i < env.WorkerCount; i++ {
		if len(env.Queues[i]) < cap(env.Queues[i]) {
			availableWorkers[i] = true
		} else {
			// Don't set value in availableWorkers map so that length is 0 if all are full
			log.Printf("Worker %d is full.\n", i)
		}
	}
	return availableWorkers
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
		log.Printf("Worker %d start processing task: %s\n", workerId, task.Payload)
		env.SleepPolicies[workerId]()
		elapsed := time.Now().Sub(start)
		task.Result <- fmt.Sprintf("Processed by worker %d: %s", workerId, task.Payload)
		log.Printf("Worker %d finished processing task: %s in %d ms.\n", workerId, task.Payload, elapsed.Milliseconds())
	}
}

func (env *Environment) HandleTask(payload string) (int, error) {
	defer func() {
		env.ProcessedTaskCount.Add(1)
		if err := recover(); err != nil {
			log.Println("Task failed:", err)
		}
	}()

	start := time.Now()

	state := env.GetState(payload)
	//log.Printf("State: %s\n", state)

	// Make sure there are available workers
	availableWorkers := env.GetAvailableWorkers()
	if len(availableWorkers) == 0 {
		log.Println("No available workers!")
		env.SelectedWorkers <- -1
		return -1, nil
	}

	// Choose worker based on RL agent
	workerIndex := env.Agent.ChooseWorker(state, availableWorkers)
	if workerIndex == -1 {
		log.Println("RL Agent could not select a worker!")
		env.SelectedWorkers <- -1
		return -1, nil
	}

	log.Printf("Selected worker: %d\n", workerIndex)

	// Enqueue task
	resultChannel := make(chan string)
	isSubmitted := env.EnqueueTask(workerIndex, Task{Payload: payload, Result: resultChannel})

	var reward float64

	learnFunc := func(workerIndex int, reward float64) {
		// Learn RLAgent based on the observed reward
		nextState := env.GetState(payload)
		env.Agent.Learn(state, nextState, workerIndex, reward)
	}

	if isSubmitted {
		// Wait for result
		<-resultChannel
		//log.Printf("Task result: %s\n", result)
		close(resultChannel)

		duration := time.Now().Sub(start)
		log.Printf("Task processed in %d ms.\n", duration.Milliseconds())
		reward = 1.0 / float64(duration.Seconds())
		env.SelectedWorkers <- workerIndex
		defer learnFunc(workerIndex, reward)
		return workerIndex, nil
	} else {
		log.Printf("Worker %d queue is full, add reward penalty.\n", workerIndex)
		reward = -50.0
		env.SelectedWorkers <- -1
		defer learnFunc(workerIndex, reward)
		return -1, nil
	}
}
