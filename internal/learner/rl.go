package learner

import (
	"math"
	"math/rand/v2"
	"strings"
	"sync"
)

type QTable map[string][]float64

type RLAgent struct {
	Table       QTable
	Alpha       float64
	Gamma       float64
	Epsilon     float64
	WorkerCount int
	Lock        sync.RWMutex
	NextWorker  int
}

func NewRLAgent(alpha float64, gamma float64, epsilon float64, workerCount int) *RLAgent {
	return &RLAgent{
		Table:       make(QTable),
		Alpha:       alpha,
		Gamma:       gamma,
		Epsilon:     epsilon,
		WorkerCount: workerCount,
		NextWorker:  0,
	}
}

const InitialOptimism = 25.0

// Ensure the state exists in the Q-table, initializing with default Q-values if not
// Returns true if state already existed, false if it was created
func (agent *RLAgent) EnsureStateExists(state string) bool {
	if _, exists := agent.Table[state]; !exists {
		qValues := make([]float64, agent.WorkerCount)
		for i := range qValues {
			qValues[i] = InitialOptimism
		}
		agent.Table[state] = qValues
		return false
	}
	return true
}

// Chose the worker with the highest Q-value for the given state, or explore randomly based on epsilon
// If the state is not in the table, initialize it with default Q-values
func (agent *RLAgent) ChooseWorker(state string, availableWorkers map[int]bool) int {
	agent.Lock.RLock()
	_, stateExists := agent.Table[state]
	agent.Lock.RUnlock()

	if !stateExists {
		// Initialize state in Q-table
		agent.Lock.Lock()
		agent.EnsureStateExists(state)
		agent.Lock.Unlock()
	}

	agent.Lock.RLock()
	defer agent.Lock.RUnlock()

	// Epsilon-greedy action selection
	// Use NextWorker for round-robin exploration
	if rand.Float64() < agent.Epsilon || !stateExists {
		updateNextWorkerFunc := func() { agent.NextWorker = (agent.NextWorker + 1) % agent.WorkerCount }
		for !availableWorkers[agent.NextWorker] {
			updateNextWorkerFunc()
		}
		defer updateNextWorkerFunc()
		return agent.NextWorker
	}

	// Exploit: choose the worker with the highest Q-value
	var bestIndex []int
	maxVal := -math.MaxFloat64

	const tolerance = 1e-9
	for i, qValue := range agent.Table[state] {
		// Only consider available workers
		if availableWorkers[i] {
			if qValue > maxVal+tolerance {
				maxVal = qValue
				bestIndex = []int{i}
			} else if math.Abs(qValue-maxVal) <= tolerance {
				bestIndex = append(bestIndex, i)
			}
		}
	}

	if len(bestIndex) == 0 {
		// No available workers, should not happen due to prior checks
		return -1
	} else {
		// If multiple best workers, choose randomly among them
		choice := bestIndex[rand.IntN(len(bestIndex))]
		return choice
	}
}

func (agent *RLAgent) Learn(state string, nextState string, workerIndex int, reward float64) {
	agent.Lock.Lock()
	defer agent.Lock.Unlock()

	agent.EnsureStateExists(state)
	agent.EnsureStateExists(nextState)

	currentQ := agent.Table[state][workerIndex]

	var maxNextQ float64
	if strings.Count(nextState, "FULL") == agent.WorkerCount {
		// All workers are full, add pessimism
		maxNextQ = -100.0
	} else {
		maxNextQ = agent.Table[nextState][0]
		for i := 1; i < len(agent.Table[nextState]); i++ {
			if agent.Table[nextState][i] > maxNextQ {
				maxNextQ = agent.Table[nextState][i]
			}
		}
	}

	// Q-learning update rule
	newQ := (1-agent.Alpha)*currentQ + agent.Alpha*(reward+agent.Gamma*maxNextQ)
	//fmt.Printf("Updating Q-value for state %s, worker %d: old Q=%.4f, reward=%.4f, new Q=%.4f\n", state, workerIndex, currentQ, reward, newQ)
	agent.Table[state][workerIndex] = newQ
}
