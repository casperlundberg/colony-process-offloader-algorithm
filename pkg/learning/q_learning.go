package learning

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"time"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// QLearning implements simplified Q-Learning (Watkins, 1989) for long-term optimization
// Uses discretized state space to keep it tractable
type QLearning struct {
	qTable          map[string]map[models.DataLocation]float64 // Q(state, action) table
	alpha           float64                                     // Learning rate
	gamma           float64                                     // Discount factor
	epsilon         float64                                     // Exploration rate
	buckets         int                                         // State discretization buckets
	lastState       string                                      // Last observed state
	lastAction      models.DataLocation                         // Last action taken
	totalUpdates    int                                         // Total Q-table updates
	convergenceData map[string]float64                          // Track convergence
}

// QState represents a discretized state for Q-Learning
type QState struct {
	LoadLevel       int                `json:"load_level"`        // 0-9 (discretized system load)
	DataSize        int                `json:"data_size"`         // 0-9 (discretized data size)
	DataLocation    models.DataLocation `json:"data_location"`     // Current data location
	DAGStage        int                `json:"dag_stage"`         // Current pipeline stage
	TimeSlot        int                `json:"time_slot"`         // Hour of day
}

// QAction represents possible actions (target locations for offloading)
type QAction = models.DataLocation

// NewQLearning creates a new Q-Learning component
func NewQLearning(alpha, gamma, epsilon float64, buckets int) *QLearning {
	return &QLearning{
		qTable:          make(map[string]map[models.DataLocation]float64),
		alpha:           alpha,
		gamma:           gamma,
		epsilon:         epsilon,
		buckets:         buckets,
		convergenceData: make(map[string]float64),
	}
}

// DiscretizeState converts continuous metrics to discrete state representation
func (ql *QLearning) DiscretizeState(metrics *models.ExtendedMetricsVector) QState {
	// Discretize load level (0-9 based on overall system load)
	loadScore := metrics.SystemState.GetLoadScore()
	loadLevel := int(loadScore * 10.0)
	if loadLevel > 9 {
		loadLevel = 9
	}
	
	// Discretize data size (0-9 based on logarithmic scale)
	dataSizeLevel := 0
	if metrics.DataSizePendingGB > 0 {
		// Use log scale: 0-0.1GB=0, 0.1-1GB=1, 1-10GB=2, etc.
		dataSizeLevel = int(math.Log10(metrics.DataSizePendingGB) + 2.0)
		if dataSizeLevel < 0 {
			dataSizeLevel = 0
		}
		if dataSizeLevel > 9 {
			dataSizeLevel = 9
		}
	}
	
	// Current DAG stage (keep as-is, bounded)
	dagStage := metrics.DAGContext.CurrentStage
	if dagStage > 9 {
		dagStage = 9
	}
	
	return QState{
		LoadLevel:    loadLevel,
		DataSize:     dataSizeLevel,
		DataLocation: metrics.DataLocation,
		DAGStage:     dagStage,
		TimeSlot:     metrics.SystemState.TimeSlot,
	}
}

// StateToKey converts QState to string key for Q-table lookup
func (ql *QLearning) StateToKey(state QState) string {
	// Create a hash-based key to handle large state spaces efficiently
	stateData := fmt.Sprintf("L%d-D%d-%s-S%d-T%d", 
		state.LoadLevel, state.DataSize, state.DataLocation, 
		state.DAGStage, state.TimeSlot)
	
	// Use MD5 hash for consistent, compact keys
	hash := md5.Sum([]byte(stateData))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes of hash
}

// InitializeQValue initializes Q-value for a state-action pair if not exists
func (ql *QLearning) InitializeQValue(stateKey string, action QAction) {
	if ql.qTable[stateKey] == nil {
		ql.qTable[stateKey] = make(map[models.DataLocation]float64)
	}
	
	if _, exists := ql.qTable[stateKey][action]; !exists {
		// Initialize with small random value to break ties
		ql.qTable[stateKey][action] = (math.Mod(float64(ql.totalUpdates), 1000.0) - 500.0) / 10000.0
	}
}

// GetBestAction returns the best action for given state (exploitation)
func (ql *QLearning) GetBestAction(state QState, availableActions []QAction) QAction {
	stateKey := ql.StateToKey(state)
	
	if len(availableActions) == 0 {
		return models.DataLocationLocal // Default fallback
	}
	
	// Initialize Q-values if needed
	for _, action := range availableActions {
		ql.InitializeQValue(stateKey, action)
	}
	
	// Find action with highest Q-value
	bestAction := availableActions[0]
	bestValue := ql.qTable[stateKey][bestAction]
	
	for _, action := range availableActions {
		if ql.qTable[stateKey][action] > bestValue {
			bestValue = ql.qTable[stateKey][action]
			bestAction = action
		}
	}
	
	return bestAction
}

// SelectAction selects action using epsilon-greedy policy
func (ql *QLearning) SelectAction(state QState, availableActions []QAction) QAction {
	if len(availableActions) == 0 {
		return models.DataLocationLocal // Default fallback
	}
	
	// Epsilon-greedy exploration
	if math.Mod(float64(ql.totalUpdates), 1000.0)/1000.0 < ql.epsilon {
		// Random exploration
		randomIndex := int(math.Mod(float64(ql.totalUpdates), float64(len(availableActions))))
		return availableActions[randomIndex]
	}
	
	// Greedy exploitation
	return ql.GetBestAction(state, availableActions)
}

// UpdateQValue updates Q-value using Q-learning update rule
func (ql *QLearning) UpdateQValue(
	currentState QState,
	action QAction,
	reward float64,
	nextState QState,
	availableNextActions []QAction,
) {
	currentStateKey := ql.StateToKey(currentState)
	nextStateKey := ql.StateToKey(nextState)
	
	// Initialize current state-action if needed
	ql.InitializeQValue(currentStateKey, action)
	
	// Get current Q-value
	currentQ := ql.qTable[currentStateKey][action]
	
	// Calculate max Q-value for next state
	maxNextQ := 0.0
	if len(availableNextActions) > 0 {
		for _, nextAction := range availableNextActions {
			ql.InitializeQValue(nextStateKey, nextAction)
		}
		
		maxNextQ = ql.qTable[nextStateKey][availableNextActions[0]]
		for _, nextAction := range availableNextActions {
			if ql.qTable[nextStateKey][nextAction] > maxNextQ {
				maxNextQ = ql.qTable[nextStateKey][nextAction]
			}
		}
	}
	
	// Q-learning update rule: Q(s,a) = Q(s,a) + α * (reward + γ * max(Q(s',a')) - Q(s,a))
	newQ := currentQ + ql.alpha*(reward+ql.gamma*maxNextQ-currentQ)
	ql.qTable[currentStateKey][action] = newQ
	
	ql.totalUpdates++
	
	// Track convergence
	ql.convergenceData[currentStateKey] = math.Abs(newQ - currentQ)
}

// UpdateFromOutcome updates Q-value from a strategy outcome
func (ql *QLearning) UpdateFromOutcome(outcome QLearningOutcome) {
	ql.UpdateQValue(
		outcome.CurrentState,
		outcome.Action,
		outcome.Reward,
		outcome.NextState,
		outcome.AvailableNextActions,
	)
	
	// Store for next update
	ql.lastState = ql.StateToKey(outcome.NextState)
	ql.lastAction = outcome.Action
}

// QLearningOutcome represents the complete outcome for Q-learning update
type QLearningOutcome struct {
	CurrentState          QState             `json:"current_state"`
	Action               QAction            `json:"action"`
	Reward               float64            `json:"reward"`
	NextState            QState             `json:"next_state"`
	AvailableNextActions []QAction          `json:"available_next_actions"`
	Timestamp            time.Time          `json:"timestamp"`
}

// GetQValue returns Q-value for state-action pair
func (ql *QLearning) GetQValue(state QState, action QAction) float64 {
	stateKey := ql.StateToKey(state)
	if ql.qTable[stateKey] == nil {
		return 0.0
	}
	
	if qValue, exists := ql.qTable[stateKey][action]; exists {
		return qValue
	}
	
	return 0.0
}

// GetStateValues returns all Q-values for a given state
func (ql *QLearning) GetStateValues(state QState) map[QAction]float64 {
	stateKey := ql.StateToKey(state)
	result := make(map[QAction]float64)
	
	if ql.qTable[stateKey] != nil {
		for action, value := range ql.qTable[stateKey] {
			result[action] = value
		}
	}
	
	return result
}

// GetLearningStats returns statistics about the Q-learning process
func (ql *QLearning) GetLearningStats() QLearningStats {
	stateCount := len(ql.qTable)
	totalStateActionPairs := 0
	avgQValue := 0.0
	maxQValue := math.Inf(-1)
	minQValue := math.Inf(1)
	
	for _, actions := range ql.qTable {
		for _, qValue := range actions {
			totalStateActionPairs++
			avgQValue += qValue
			if qValue > maxQValue {
				maxQValue = qValue
			}
			if qValue < minQValue {
				minQValue = qValue
			}
		}
	}
	
	if totalStateActionPairs > 0 {
		avgQValue /= float64(totalStateActionPairs)
	}
	
	// Calculate convergence metric (average of recent updates)
	convergence := 0.0
	convergenceCount := 0
	for _, delta := range ql.convergenceData {
		convergence += delta
		convergenceCount++
		if convergenceCount >= 100 { // Limit to recent updates
			break
		}
	}
	if convergenceCount > 0 {
		convergence /= float64(convergenceCount)
	}
	
	return QLearningStats{
		StateCount:            stateCount,
		StateActionPairs:      totalStateActionPairs,
		TotalUpdates:          ql.totalUpdates,
		AverageQValue:         avgQValue,
		MaxQValue:             maxQValue,
		MinQValue:             minQValue,
		ConvergenceRate:       convergence,
		ExplorationRate:       ql.epsilon,
		LearningRate:          ql.alpha,
		DiscountFactor:        ql.gamma,
		LastUpdated:           time.Now(),
	}
}

// QLearningStats provides insights into the Q-learning process
type QLearningStats struct {
	StateCount         int       `json:"state_count"`
	StateActionPairs   int       `json:"state_action_pairs"`
	TotalUpdates       int       `json:"total_updates"`
	AverageQValue      float64   `json:"average_q_value"`
	MaxQValue          float64   `json:"max_q_value"`
	MinQValue          float64   `json:"min_q_value"`
	ConvergenceRate    float64   `json:"convergence_rate"`    // Lower = more converged
	ExplorationRate    float64   `json:"exploration_rate"`
	LearningRate       float64   `json:"learning_rate"`
	DiscountFactor     float64   `json:"discount_factor"`
	LastUpdated        time.Time `json:"last_updated"`
}

// DecayExploration reduces exploration rate over time
func (ql *QLearning) DecayExploration(decayRate float64, minEpsilon float64) {
	ql.epsilon = math.Max(minEpsilon, ql.epsilon*decayRate)
}

// RecommendAction recommends best action for given state based on learned policy
func (ql *QLearning) RecommendAction(
	metrics *models.ExtendedMetricsVector,
	availableLocations []models.DataLocation,
) models.DataLocation {
	state := ql.DiscretizeState(metrics)
	return ql.GetBestAction(state, availableLocations)
}

// ExportPolicy exports the learned policy as a readable format
func (ql *QLearning) ExportPolicy() map[string]PolicyRule {
	policy := make(map[string]PolicyRule)
	
	for stateKey, actions := range ql.qTable {
		if len(actions) == 0 {
			continue
		}
		
		// Find best action for this state
		bestAction := models.DataLocationLocal
		bestValue := math.Inf(-1)
		
		actionValues := make(map[models.DataLocation]float64)
		for action, value := range actions {
			actionValues[action] = value
			if value > bestValue {
				bestValue = value
				bestAction = action
			}
		}
		
		policy[stateKey] = PolicyRule{
			BestAction:   bestAction,
			BestValue:    bestValue,
			ActionValues: actionValues,
		}
	}
	
	return policy
}

// PolicyRule represents a learned policy rule for a specific state
type PolicyRule struct {
	BestAction   models.DataLocation                  `json:"best_action"`
	BestValue    float64                             `json:"best_value"`
	ActionValues map[models.DataLocation]float64     `json:"action_values"`
}

// Reset resets the Q-learning component to initial state
func (ql *QLearning) Reset() {
	ql.qTable = make(map[string]map[models.DataLocation]float64)
	ql.totalUpdates = 0
	ql.convergenceData = make(map[string]float64)
	ql.lastState = ""
	ql.lastAction = models.DataLocationLocal
}

// SerializeQTable serializes the Q-table to JSON for persistence
func (ql *QLearning) SerializeQTable() (string, error) {
	data, err := json.Marshal(ql.qTable)
	if err != nil {
		return "", fmt.Errorf("failed to serialize Q-table: %w", err)
	}
	return string(data), nil
}

// LoadQTable loads Q-table from JSON string
func (ql *QLearning) LoadQTable(data string) error {
	var qTable map[string]map[models.DataLocation]float64
	err := json.Unmarshal([]byte(data), &qTable)
	if err != nil {
		return fmt.Errorf("failed to deserialize Q-table: %w", err)
	}
	
	ql.qTable = qTable
	return nil
}

// CreateQLearningOutcome creates a Q-learning outcome from decision results
func CreateQLearningOutcome(
	currentMetrics, nextMetrics *models.ExtendedMetricsVector,
	selectedLocation models.DataLocation,
	availableLocations []models.DataLocation,
	reward float64,
	ql *QLearning,
) QLearningOutcome {
	
	currentState := ql.DiscretizeState(currentMetrics)
	nextState := ql.DiscretizeState(nextMetrics)
	
	return QLearningOutcome{
		CurrentState:          currentState,
		Action:               selectedLocation,
		Reward:               reward,
		NextState:            nextState,
		AvailableNextActions: availableLocations,
		Timestamp:            time.Now(),
	}
}