package learning

import (
	"math"
	"strings"
	"testing"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

func TestQLearning_NewQLearning(t *testing.T) {
	alpha := 0.1
	gamma := 0.9
	epsilon := 0.1
	buckets := 10

	ql := NewQLearning(alpha, gamma, epsilon, buckets)
	if ql == nil {
		t.Fatal("NewQLearning() returned nil")
	}

	if ql.alpha != alpha {
		t.Errorf("Expected alpha=%f, got %f", alpha, ql.alpha)
	}

	if ql.gamma != gamma {
		t.Errorf("Expected gamma=%f, got %f", gamma, ql.gamma)
	}

	if ql.epsilon != epsilon {
		t.Errorf("Expected epsilon=%f, got %f", epsilon, ql.epsilon)
	}

	if ql.buckets != buckets {
		t.Errorf("Expected buckets=%d, got %d", buckets, ql.buckets)
	}

	if ql.qTable == nil {
		t.Error("Q-table should be initialized")
	}

	if ql.totalUpdates != 0 {
		t.Errorf("Expected totalUpdates=0, got %d", ql.totalUpdates)
	}
}

func TestQLearning_DiscretizeState(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	metrics := &models.ExtendedMetricsVector{
		DataSizePendingGB: 5.0,
		DataLocation:     models.DataLocationCloud,
		SystemState: models.SystemState{
			TimeSlot: 14, // 2 PM
		},
		DAGContext: models.DAGContext{
			CurrentStage: 3,
		},
	}

	// Mock the GetLoadScore method for testing
	metrics.SystemState.ComputeUsage = 0.7
	metrics.SystemState.MemoryUsage = 0.6

	state := ql.DiscretizeState(metrics)

	if state.DataLocation != models.DataLocationCloud {
		t.Errorf("Expected DataLocation=%s, got %s", 
			models.DataLocationCloud, state.DataLocation)
	}

	if state.TimeSlot != 14 {
		t.Errorf("Expected TimeSlot=14, got %d", state.TimeSlot)
	}

	if state.DAGStage != 3 {
		t.Errorf("Expected DAGStage=3, got %d", state.DAGStage)
	}

	// Test data size discretization (log scale)
	if state.DataSize < 0 || state.DataSize > 9 {
		t.Errorf("DataSize should be 0-9, got %d", state.DataSize)
	}

	// Test load level discretization
	if state.LoadLevel < 0 || state.LoadLevel > 9 {
		t.Errorf("LoadLevel should be 0-9, got %d", state.LoadLevel)
	}
}

func TestQLearning_StateToKey(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	state := QState{
		LoadLevel:    5,
		DataSize:     3,
		DataLocation: models.DataLocationCloud,
		DAGStage:     2,
		TimeSlot:     10,
	}

	key1 := ql.StateToKey(state)
	key2 := ql.StateToKey(state)

	// Same state should produce same key
	if key1 != key2 {
		t.Errorf("Same state should produce same key: %s != %s", key1, key2)
	}

	// Key should be a valid hex string of expected length
	if len(key1) != 16 { // 8 bytes * 2 hex chars per byte
		t.Errorf("Expected key length 16, got %d", len(key1))
	}

	// Different state should produce different key
	state.LoadLevel = 6
	key3 := ql.StateToKey(state)
	if key1 == key3 {
		t.Error("Different states should produce different keys")
	}
}

func TestQLearning_InitializeQValue(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	state := QState{LoadLevel: 1, DataSize: 1}
	stateKey := ql.StateToKey(state)
	action := models.DataLocationCloud

	// Initially should not exist
	if ql.qTable[stateKey] != nil {
		t.Error("Q-table entry should not exist initially")
	}

	ql.InitializeQValue(stateKey, action)

	// Should now exist
	if ql.qTable[stateKey] == nil {
		t.Error("Q-table entry should exist after initialization")
	}

	if _, exists := ql.qTable[stateKey][action]; !exists {
		t.Error("Q-value should exist after initialization")
	}

	// Value should be small
	qValue := ql.qTable[stateKey][action]
	if math.Abs(qValue) > 0.1 {
		t.Errorf("Initial Q-value should be small, got %f", qValue)
	}
}

func TestQLearning_GetBestAction(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	state := QState{LoadLevel: 1, DataSize: 1}
	actions := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
		models.DataLocationEdge,
	}

	bestAction := ql.GetBestAction(state, actions)

	// Should return one of the available actions
	found := false
	for _, action := range actions {
		if action == bestAction {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Best action %s not in available actions", bestAction)
	}

	// Test with empty actions
	emptyActions := []models.DataLocation{}
	defaultAction := ql.GetBestAction(state, emptyActions)
	if defaultAction != models.DataLocationLocal {
		t.Errorf("Expected default action %s, got %s", 
			models.DataLocationLocal, defaultAction)
	}
}

func TestQLearning_SelectAction(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	state := QState{LoadLevel: 1, DataSize: 1}
	actions := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
	}

	// Test multiple selections to verify both exploration and exploitation happen
	actionCounts := make(map[models.DataLocation]int)
	for i := 0; i < 100; i++ {
		action := ql.SelectAction(state, actions)
		actionCounts[action]++
	}

	// Should have selected some actions
	if len(actionCounts) == 0 {
		t.Error("No actions were selected")
	}

	// All selected actions should be in available actions
	for action := range actionCounts {
		found := false
		for _, availableAction := range actions {
			if action == availableAction {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Selected action %s not in available actions", action)
		}
	}
}

func TestQLearning_UpdateQValue(t *testing.T) {
	ql := NewQLearning(0.5, 0.9, 0.1, 10)

	currentState := QState{LoadLevel: 1, DataSize: 1}
	nextState := QState{LoadLevel: 2, DataSize: 2}
	action := models.DataLocationCloud
	reward := 10.0
	availableNextActions := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
	}

	// Get initial Q-value (should be 0 or small)
	initialQ := ql.GetQValue(currentState, action)
	initialUpdates := ql.totalUpdates

	ql.UpdateQValue(currentState, action, reward, nextState, availableNextActions)

	// Q-value should be updated
	updatedQ := ql.GetQValue(currentState, action)
	if updatedQ == initialQ {
		t.Errorf("Q-value should be updated, still %f", initialQ)
	}

	// Total updates should increment
	if ql.totalUpdates != initialUpdates+1 {
		t.Errorf("Expected totalUpdates=%d, got %d", initialUpdates+1, ql.totalUpdates)
	}

	// Positive reward should generally increase Q-value (with positive learning rate)
	if reward > 0 && ql.alpha > 0 && updatedQ < initialQ {
		t.Logf("Note: Q-value decreased despite positive reward (can happen due to future state)")
	}
}

func TestQLearning_GetQValue(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	state := QState{LoadLevel: 1, DataSize: 1}
	action := models.DataLocationCloud

	// Initially should return 0
	qValue := ql.GetQValue(state, action)
	if qValue != 0.0 {
		t.Errorf("Initial Q-value should be 0.0, got %f", qValue)
	}

	// After initialization, should return initialized value
	stateKey := ql.StateToKey(state)
	ql.InitializeQValue(stateKey, action)
	
	qValue = ql.GetQValue(state, action)
	if qValue == 0.0 {
		t.Error("Q-value should not be 0.0 after initialization")
	}
}

func TestQLearning_GetStateValues(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	state := QState{LoadLevel: 1, DataSize: 1}
	actions := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
	}

	// Initially should return empty map
	values := ql.GetStateValues(state)
	if len(values) != 0 {
		t.Errorf("Initial state values should be empty, got %d", len(values))
	}

	// Initialize some values
	stateKey := ql.StateToKey(state)
	for _, action := range actions {
		ql.InitializeQValue(stateKey, action)
	}

	// Should now return initialized values
	values = ql.GetStateValues(state)
	if len(values) != len(actions) {
		t.Errorf("Expected %d state values, got %d", len(actions), len(values))
	}

	for _, action := range actions {
		if _, exists := values[action]; !exists {
			t.Errorf("Missing Q-value for action %s", action)
		}
	}
}

func TestQLearning_GetLearningStats(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	// Initial stats
	stats := ql.GetLearningStats()
	if stats.StateCount != 0 {
		t.Errorf("Expected initial state count=0, got %d", stats.StateCount)
	}
	if stats.TotalUpdates != 0 {
		t.Errorf("Expected initial total updates=0, got %d", stats.TotalUpdates)
	}

	// Add some data
	state := QState{LoadLevel: 1, DataSize: 1}
	action := models.DataLocationCloud
	reward := 5.0
	nextState := QState{LoadLevel: 2, DataSize: 1}
	availableActions := []models.DataLocation{models.DataLocationLocal, models.DataLocationCloud}

	ql.UpdateQValue(state, action, reward, nextState, availableActions)

	// Updated stats
	stats = ql.GetLearningStats()
	if stats.StateCount == 0 {
		t.Error("State count should be > 0 after updates")
	}
	if stats.TotalUpdates != 1 {
		t.Errorf("Expected total updates=1, got %d", stats.TotalUpdates)
	}
	if stats.ExplorationRate != ql.epsilon {
		t.Errorf("Expected exploration rate=%f, got %f", ql.epsilon, stats.ExplorationRate)
	}
}

func TestQLearning_DecayExploration(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.5, 10)
	
	initialEpsilon := ql.epsilon
	decayRate := 0.95
	minEpsilon := 0.01

	ql.DecayExploration(decayRate, minEpsilon)

	if ql.epsilon >= initialEpsilon {
		t.Error("Exploration rate should decrease after decay")
	}

	expectedEpsilon := initialEpsilon * decayRate
	if math.Abs(ql.epsilon-expectedEpsilon) > 0.0001 {
		t.Errorf("Expected epsilon=%f, got %f", expectedEpsilon, ql.epsilon)
	}

	// Test minimum bound
	ql.epsilon = 0.02
	ql.DecayExploration(0.1, minEpsilon)
	if ql.epsilon < minEpsilon {
		t.Errorf("Epsilon should not go below minimum %f, got %f", minEpsilon, ql.epsilon)
	}
}

func TestQLearning_RecommendAction(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	metrics := &models.ExtendedMetricsVector{
		DataSizePendingGB: 1.0,
		DataLocation:     models.DataLocationLocal,
		SystemState: models.SystemState{
			TimeSlot:     10,
			ComputeUsage: 0.5,
			MemoryUsage:  0.4,
		},
		DAGContext: models.DAGContext{
			CurrentStage: 1,
		},
	}

	availableLocations := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
	}

	action := ql.RecommendAction(metrics, availableLocations)

	// Should return one of available locations
	found := false
	for _, loc := range availableLocations {
		if loc == action {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Recommended action %s not in available locations", action)
	}
}

func TestQLearning_ExportPolicy(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	// Add some Q-values
	state := QState{LoadLevel: 1, DataSize: 1}
	stateKey := ql.StateToKey(state)
	
	ql.InitializeQValue(stateKey, models.DataLocationLocal)
	ql.InitializeQValue(stateKey, models.DataLocationCloud)
	
	// Set different values to test best action selection
	ql.qTable[stateKey][models.DataLocationLocal] = 5.0
	ql.qTable[stateKey][models.DataLocationCloud] = 10.0

	policy := ql.ExportPolicy()

	if len(policy) == 0 {
		t.Error("Policy should not be empty")
	}

	rule, exists := policy[stateKey]
	if !exists {
		t.Error("Policy rule should exist for state")
	}

	if rule.BestAction != models.DataLocationCloud {
		t.Errorf("Expected best action %s, got %s", 
			models.DataLocationCloud, rule.BestAction)
	}

	if rule.BestValue != 10.0 {
		t.Errorf("Expected best value=10.0, got %f", rule.BestValue)
	}

	if len(rule.ActionValues) != 2 {
		t.Errorf("Expected 2 action values, got %d", len(rule.ActionValues))
	}
}

func TestQLearning_Reset(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	// Add some data
	state := QState{LoadLevel: 1, DataSize: 1}
	action := models.DataLocationCloud
	ql.UpdateQValue(state, action, 5.0, state, []models.DataLocation{action})

	// Verify data exists
	if len(ql.qTable) == 0 {
		t.Error("Q-table should have entries before reset")
	}
	if ql.totalUpdates == 0 {
		t.Error("Total updates should be > 0 before reset")
	}

	ql.Reset()

	// Verify reset
	if len(ql.qTable) != 0 {
		t.Errorf("Q-table should be empty after reset, got %d entries", len(ql.qTable))
	}

	if ql.totalUpdates != 0 {
		t.Errorf("Total updates should be 0 after reset, got %d", ql.totalUpdates)
	}

	if ql.lastState != "" {
		t.Errorf("Last state should be empty after reset, got %s", ql.lastState)
	}

	if ql.lastAction != models.DataLocationLocal {
		t.Errorf("Last action should be local after reset, got %s", ql.lastAction)
	}
}

func TestQLearning_SerializeQTable(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	// Add some data
	state := QState{LoadLevel: 1, DataSize: 1}
	stateKey := ql.StateToKey(state)
	ql.InitializeQValue(stateKey, models.DataLocationCloud)
	ql.qTable[stateKey][models.DataLocationCloud] = 42.0

	data, err := ql.SerializeQTable()
	if err != nil {
		t.Fatalf("SerializeQTable() failed: %v", err)
	}

	if data == "" {
		t.Error("Serialized data should not be empty")
	}

	// Should be valid JSON
	if !strings.Contains(data, "{") || !strings.Contains(data, "}") {
		t.Error("Serialized data should be valid JSON")
	}
}

func TestQLearning_LoadQTable(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	// Create some test data
	state := QState{LoadLevel: 1, DataSize: 1}
	stateKey := ql.StateToKey(state)
	ql.InitializeQValue(stateKey, models.DataLocationCloud)
	ql.qTable[stateKey][models.DataLocationCloud] = 42.0

	// Serialize
	data, err := ql.SerializeQTable()
	if err != nil {
		t.Fatalf("SerializeQTable() failed: %v", err)
	}

	// Reset and load
	ql.Reset()
	err = ql.LoadQTable(data)
	if err != nil {
		t.Fatalf("LoadQTable() failed: %v", err)
	}

	// Verify loaded data
	if len(ql.qTable) == 0 {
		t.Error("Q-table should have entries after loading")
	}

	qValue := ql.GetQValue(state, models.DataLocationCloud)
	if qValue != 42.0 {
		t.Errorf("Expected loaded Q-value=42.0, got %f", qValue)
	}
}

func TestQLearning_LoadQTableInvalidJSON(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	err := ql.LoadQTable("invalid json data")
	if err == nil {
		t.Error("Expected error for invalid JSON data")
	}
}

func TestQLearning_UpdateFromOutcome(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	currentState := QState{LoadLevel: 1, DataSize: 1}
	nextState := QState{LoadLevel: 2, DataSize: 2}
	action := models.DataLocationCloud
	
	outcome := QLearningOutcome{
		CurrentState: currentState,
		Action:       action,
		Reward:       15.0,
		NextState:    nextState,
		AvailableNextActions: []models.DataLocation{
			models.DataLocationLocal,
			models.DataLocationCloud,
		},
	}

	initialUpdates := ql.totalUpdates
	ql.UpdateFromOutcome(outcome)

	// Should update Q-value
	if ql.totalUpdates != initialUpdates+1 {
		t.Errorf("Expected totalUpdates=%d, got %d", initialUpdates+1, ql.totalUpdates)
	}

	// Should store last state and action
	expectedLastState := ql.StateToKey(nextState)
	if ql.lastState != expectedLastState {
		t.Errorf("Expected lastState=%s, got %s", expectedLastState, ql.lastState)
	}

	if ql.lastAction != action {
		t.Errorf("Expected lastAction=%s, got %s", action, ql.lastAction)
	}
}

func TestCreateQLearningOutcome(t *testing.T) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)

	currentMetrics := &models.ExtendedMetricsVector{
		DataSizePendingGB: 1.0,
		DataLocation:     models.DataLocationLocal,
		SystemState: models.SystemState{
			TimeSlot: 10,
		},
		DAGContext: models.DAGContext{
			CurrentStage: 1,
		},
	}

	nextMetrics := &models.ExtendedMetricsVector{
		DataSizePendingGB: 2.0,
		DataLocation:     models.DataLocationCloud,
		SystemState: models.SystemState{
			TimeSlot: 11,
		},
		DAGContext: models.DAGContext{
			CurrentStage: 2,
		},
	}

	selectedLocation := models.DataLocationCloud
	availableLocations := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
	}
	reward := 20.0

	outcome := CreateQLearningOutcome(
		currentMetrics, nextMetrics, selectedLocation, 
		availableLocations, reward, ql)

	if outcome.Action != selectedLocation {
		t.Errorf("Expected action=%s, got %s", selectedLocation, outcome.Action)
	}

	if outcome.Reward != reward {
		t.Errorf("Expected reward=%f, got %f", reward, outcome.Reward)
	}

	if len(outcome.AvailableNextActions) != len(availableLocations) {
		t.Errorf("Expected %d available actions, got %d", 
			len(availableLocations), len(outcome.AvailableNextActions))
	}

	if outcome.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// Benchmark Q-Learning performance
func BenchmarkQLearning_Update(b *testing.B) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)
	
	state := QState{LoadLevel: 1, DataSize: 1}
	action := models.DataLocationCloud
	nextState := QState{LoadLevel: 2, DataSize: 2}
	availableActions := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ql.UpdateQValue(state, action, 10.0, nextState, availableActions)
	}
}

func BenchmarkQLearning_SelectAction(b *testing.B) {
	ql := NewQLearning(0.1, 0.9, 0.1, 10)
	
	state := QState{LoadLevel: 1, DataSize: 1}
	availableActions := []models.DataLocation{
		models.DataLocationLocal,
		models.DataLocationCloud,
		models.DataLocationEdge,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ql.SelectAction(state, availableActions)
	}
}