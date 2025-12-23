package learner

import (
	"testing"
)

func TestEnsureStateExists_NewState(t *testing.T) {
	// Arrange
	workerCount := 4
	agent := NewRLAgent(0.5, 0.5, 0.1, workerCount)
	state := "W0:LOW_W1:LOW_W2:LOW_W3:LOW_P:task1"

	// Act
	existed := agent.EnsureStateExists(state)

	// Assert
	if existed {
		t.Error("Expected EnsureStateExists to return false for a new state")
	}

	qValues, exists := agent.Table[state]
	if !exists {
		t.Fatal("Expected state to exist in Q-table after EnsureStateExists")
	}

	if len(qValues) != workerCount {
		t.Errorf("Expected %d Q-values, got %d", workerCount, len(qValues))
	}

	for i, qValue := range qValues {
		if qValue != InitialOptimism {
			t.Errorf("Expected Q-value[%d] to be %.2f, got %.2f", i, InitialOptimism, qValue)
		}
	}
}

func TestEnsureStateExists_ExistingState(t *testing.T) {
	// Arrange
	workerCount := 3
	agent := NewRLAgent(0.5, 0.5, 0.1, workerCount)
	state := "W0:MED_W1:HIGH_W2:LOW_P:task2"

	// Pre-populate the state with custom Q-values
	customQValues := []float64{10.0, 20.0, 30.0}
	agent.Table[state] = customQValues

	// Act
	existed := agent.EnsureStateExists(state)

	// Assert
	if !existed {
		t.Error("Expected EnsureStateExists to return true for an existing state")
	}

	qValues := agent.Table[state]
	if len(qValues) != workerCount {
		t.Errorf("Expected %d Q-values, got %d", workerCount, len(qValues))
	}

	// Verify that Q-values were NOT modified
	for i, qValue := range qValues {
		if qValue != customQValues[i] {
			t.Errorf("Expected Q-value[%d] to remain %.2f, got %.2f", i, customQValues[i], qValue)
		}
	}
}

func TestEnsureStateExists_MultipleStates(t *testing.T) {
	// Arrange
	workerCount := 2
	agent := NewRLAgent(0.5, 0.5, 0.1, workerCount)
	state1 := "W0:LOW_W1:LOW_P:task1"
	state2 := "W0:HIGH_W1:MED_P:task2"

	// Act
	existed1 := agent.EnsureStateExists(state1)
	existed2 := agent.EnsureStateExists(state2)
	existed1Again := agent.EnsureStateExists(state1)

	// Assert
	if existed1 {
		t.Error("Expected first call to return false for state1")
	}
	if existed2 {
		t.Error("Expected first call to return false for state2")
	}
	if !existed1Again {
		t.Error("Expected second call to return true for state1")
	}

	if len(agent.Table) != 2 {
		t.Errorf("Expected 2 states in Q-table, got %d", len(agent.Table))
	}
}

func TestEnsureStateExists_EmptyState(t *testing.T) {
	// Arrange
	workerCount := 3
	agent := NewRLAgent(0.5, 0.5, 0.1, workerCount)
	state := ""

	// Act
	existed := agent.EnsureStateExists(state)

	// Assert
	if existed {
		t.Error("Expected EnsureStateExists to return false for empty state")
	}

	qValues, exists := agent.Table[state]
	if !exists {
		t.Fatal("Expected empty state to exist in Q-table")
	}

	if len(qValues) != workerCount {
		t.Errorf("Expected %d Q-values, got %d", workerCount, len(qValues))
	}

	for i, qValue := range qValues {
		if qValue != InitialOptimism {
			t.Errorf("Expected Q-value[%d] to be %.2f, got %.2f", i, InitialOptimism, qValue)
		}
	}
}

func TestEnsureStateExists_ZeroWorkers(t *testing.T) {
	// Arrange
	workerCount := 0
	agent := NewRLAgent(0.5, 0.5, 0.1, workerCount)
	state := "W_P:task"

	// Act
	existed := agent.EnsureStateExists(state)

	// Assert
	if existed {
		t.Error("Expected EnsureStateExists to return false for new state")
	}

	qValues, exists := agent.Table[state]
	if !exists {
		t.Fatal("Expected state to exist in Q-table")
	}

	if len(qValues) != 0 {
		t.Errorf("Expected 0 Q-values, got %d", len(qValues))
	}
}
