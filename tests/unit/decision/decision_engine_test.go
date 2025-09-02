package decision_test

import (
	"fmt"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// DecisionEngine test requirements:
// 1. Decision must be deterministic given same inputs
// 2. Decision must complete within 500ms
// 3. All scores must be in [0.0, 1.0] range
// 4. Decision quality must be explainable and auditable

type DecisionEngineTestSuite struct {
	suite.Suite
	engine *decision.DecisionEngine
}

func (suite *DecisionEngineTestSuite) SetupTest() {
	weights := decision.AdaptiveWeights{
		QueueDepth:    0.2,
		ProcessorLoad: 0.2,
		NetworkCost:   0.2,
		LatencyCost:   0.2,
		EnergyCost:    0.1,
		PolicyCost:    0.1,
	}
	
	suite.engine = decision.NewDecisionEngine(weights)
}

// Test that decisions are deterministic given same inputs
func (suite *DecisionEngineTestSuite) TestDecisionDeterminism() {
	process := models.Process{
		ID:                "test-determinism",
		CPURequirement:    2.0,
		MemoryRequirement: 4 * 1024 * 1024 * 1024, // 4GB
		InputSize:         1024 * 1024,             // 1MB
		OutputSize:        512 * 1024,              // 512KB
		EstimatedDuration: 30 * time.Second,
		Priority:          5,
		Status:            models.QUEUED,
	}

	state := models.SystemState{
		QueueDepth:      25,
		QueueThreshold:  20,
		ComputeUsage:    0.75,
		MemoryUsage:     0.60,
		NetworkUsage:    0.40,
		MasterUsage:     0.30,
		Timestamp:       time.Now(),
		TimeSlot:        12,
		DayOfWeek:       3,
	}

	targets := []models.OffloadTarget{
		{
			ID:                "target-1",
			Type:              models.EDGE,
			TotalCapacity:     8.0,
			AvailableCapacity: 6.0,
			MemoryTotal:       16 * 1024 * 1024 * 1024,
			MemoryAvailable:   10 * 1024 * 1024 * 1024,
			NetworkLatency:    15 * time.Millisecond,
			NetworkBandwidth:  100 * 1024 * 1024, // 100MB/s
			NetworkStability:  0.95,
			ProcessingSpeed:   1.0,
			Reliability:       0.95,
			ComputeCost:       0.10,
			SecurityLevel:     3,
			LastSeen:          time.Now(),
		},
		{
			ID:                "target-2",
			Type:              models.PUBLIC_CLOUD,
			TotalCapacity:     32.0,
			AvailableCapacity: 24.0,
			MemoryTotal:       64 * 1024 * 1024 * 1024,
			MemoryAvailable:   48 * 1024 * 1024 * 1024,
			NetworkLatency:    50 * time.Millisecond,
			NetworkBandwidth:  50 * 1024 * 1024, // 50MB/s
			NetworkStability:  0.99,
			ProcessingSpeed:   2.0,
			Reliability:       0.99,
			ComputeCost:       0.05,
			SecurityLevel:     4,
			LastSeen:          time.Now(),
		},
	}

	// Make the same decision multiple times
	decisions := []decision.OffloadDecision{}
	for i := 0; i < 10; i++ {
		decision, err := suite.engine.MakeDecision(process, targets, state)
		require.NoError(suite.T(), err)
		decisions = append(decisions, decision)
	}

	// All decisions should be identical
	firstDecision := decisions[0]
	for i := 1; i < len(decisions); i++ {
		assert.Equal(suite.T(), firstDecision.ShouldOffload, decisions[i].ShouldOffload,
			"ShouldOffload decision should be deterministic")
		
		if firstDecision.Target != nil && decisions[i].Target != nil {
			assert.Equal(suite.T(), firstDecision.Target.ID, decisions[i].Target.ID,
				"Target selection should be deterministic")
		}
		
		assert.InDelta(suite.T(), firstDecision.Score, decisions[i].Score, 0.001,
			"Decision score should be deterministic")
		
		// Verify score components are identical
		assert.Equal(suite.T(), firstDecision.ScoreComponents, decisions[i].ScoreComponents,
			"Score components should be deterministic")
	}
}

// Test that decisions complete within 500ms
func (suite *DecisionEngineTestSuite) TestDecisionLatencyRequirement() {
	process := models.Process{
		ID:                "test-latency",
		CPURequirement:    4.0,
		MemoryRequirement: 8 * 1024 * 1024 * 1024,
		InputSize:         10 * 1024 * 1024,
		OutputSize:        5 * 1024 * 1024,
		EstimatedDuration: 60 * time.Second,
		Priority:          7,
		Status:            models.QUEUED,
	}

	state := models.SystemState{
		QueueDepth:      50,
		QueueThreshold:  20,
		ComputeUsage:    0.85,
		MemoryUsage:     0.70,
		NetworkUsage:    0.50,
		MasterUsage:     0.40,
		Timestamp:       time.Now(),
		TimeSlot:        14,
		DayOfWeek:       2,
	}

	// Create many targets to stress-test decision latency
	targets := []models.OffloadTarget{}
	targetTypes := []models.TargetType{models.EDGE, models.PUBLIC_CLOUD, models.FOG}
	
	for i := 0; i < 20; i++ {
		target := models.OffloadTarget{
			ID:                fmt.Sprintf("target-%d", i),
			Type:              targetTypes[i%3],
			TotalCapacity:     float64(4 + i*2),
			AvailableCapacity: float64(2 + i),
			MemoryTotal:       int64((8 + i*4)) * 1024 * 1024 * 1024,
			MemoryAvailable:   int64((4 + i*2)) * 1024 * 1024 * 1024,
			NetworkLatency:    time.Duration(10+i*5) * time.Millisecond,
			NetworkBandwidth:  float64((50 + i*10) * 1024 * 1024),
			NetworkStability:  0.8 + float64(i%20)*0.01,
			ProcessingSpeed:   1.0 + float64(i)*0.1,
			Reliability:       0.8 + float64(i%20)*0.01,
			ComputeCost:       0.05 + float64(i)*0.01,
			SecurityLevel:     3,
			LastSeen:          time.Now(),
		}
		targets = append(targets, target)
	}

	// Test decision latency under normal conditions
	latencies := []time.Duration{}
	timeoutCount := 0
	iterations := 100

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := suite.engine.MakeDecision(process, targets, state)
		duration := time.Since(start)
		
		require.NoError(suite.T(), err)
		latencies = append(latencies, duration)
		
		// Check if decision exceeded 500ms timeout
		if duration > 500*time.Millisecond {
			timeoutCount++
			suite.T().Logf("Decision %d exceeded 500ms: %v", i, duration)
		}
	}

	// Calculate statistics
	avgLatency := calculateAverageLatency(latencies)
	p95Latency := calculatePercentileLatency(latencies, 95)
	p99Latency := calculatePercentileLatency(latencies, 99)

	// Verify performance requirements
	assert.Less(suite.T(), p95Latency, 500*time.Millisecond,
		"95th percentile latency should be < 500ms")
	assert.Less(suite.T(), avgLatency, 100*time.Millisecond,
		"Average latency should be < 100ms")

	// Allow up to 5% of decisions to exceed timeout
	assert.LessOrEqual(suite.T(), timeoutCount, iterations/20,
		"More than 5%% of decisions exceeded 500ms timeout")

	suite.T().Logf("Decision latency stats - Avg: %v, P95: %v, P99: %v", 
		avgLatency, p95Latency, p99Latency)
}

// Test that all scores are in [0.0, 1.0] range
func (suite *DecisionEngineTestSuite) TestScoreRangeCompliance() {
	// Test across various scenarios with extreme values
	testScenarios := []struct {
		name     string
		process  models.Process
		state    models.SystemState
		targets  []models.OffloadTarget
	}{
		{
			name: "extreme_high_load",
			process: models.Process{
				ID:                "extreme-process",
				CPURequirement:    16.0,
				MemoryRequirement: 64 * 1024 * 1024 * 1024, // 64GB
				InputSize:         1024 * 1024 * 1024,       // 1GB
				OutputSize:        512 * 1024 * 1024,        // 512MB
				EstimatedDuration: 120 * time.Second,
				Priority:          10,
				Status:            models.QUEUED,
			},
			state: models.SystemState{
				QueueDepth:      100,
				QueueThreshold:  20,
				ComputeUsage:    0.99,
				MemoryUsage:     0.95,
				NetworkUsage:    0.90,
				MasterUsage:     0.85,
				Timestamp:       time.Now(),
				TimeSlot:        10,
				DayOfWeek:       1,
			},
			targets: []models.OffloadTarget{
				{
					ID:                "high-capacity",
					Type:              models.PUBLIC_CLOUD,
					TotalCapacity:     64.0,
					AvailableCapacity: 60.0,
					MemoryTotal:       256 * 1024 * 1024 * 1024,
					MemoryAvailable:   200 * 1024 * 1024 * 1024,
					NetworkLatency:    100 * time.Millisecond,
					NetworkBandwidth:  1024 * 1024 * 1024, // 1GB/s
					NetworkStability:  0.999,
					ProcessingSpeed:   3.0,
					Reliability:       0.999,
					ComputeCost:       1.0,
					SecurityLevel:     5,
					LastSeen:          time.Now(),
				},
			},
		},
		{
			name: "minimal_load",
			process: models.Process{
				ID:                "minimal-process",
				CPURequirement:    0.1,
				MemoryRequirement: 128 * 1024 * 1024, // 128MB
				InputSize:         1024,              // 1KB
				OutputSize:        512,               // 512B
				EstimatedDuration: 100 * time.Millisecond,
				Priority:          1,
				Status:            models.QUEUED,
			},
			state: models.SystemState{
				QueueDepth:      0,
				QueueThreshold:  20,
				ComputeUsage:    0.01,
				MemoryUsage:     0.05,
				NetworkUsage:    0.01,
				MasterUsage:     0.01,
				Timestamp:       time.Now(),
				TimeSlot:        3,
				DayOfWeek:       0,
			},
			targets: []models.OffloadTarget{
				{
					ID:                "minimal-target",
					Type:              models.EDGE,
					TotalCapacity:     1.0,
					AvailableCapacity: 0.9,
					MemoryTotal:       1 * 1024 * 1024 * 1024,
					MemoryAvailable:   900 * 1024 * 1024,
					NetworkLatency:    1 * time.Millisecond,
					NetworkBandwidth:  10 * 1024 * 1024, // 10MB/s
					NetworkStability:  0.5,
					ProcessingSpeed:   0.5,
					Reliability:       0.5,
					ComputeCost:       0.01,
					SecurityLevel:     1,
					LastSeen:          time.Now(),
				},
			},
		},
	}

	for _, scenario := range testScenarios {
		suite.Run(scenario.name, func() {
			decision, err := suite.engine.MakeDecision(
				scenario.process,
				scenario.targets,
				scenario.state,
			)
			
			require.NoError(suite.T(), err)
			
			// Verify score is in valid range
			assert.GreaterOrEqual(suite.T(), decision.Score, 0.0,
				"Score should be >= 0.0")
			assert.LessOrEqual(suite.T(), decision.Score, 1.0,
				"Score should be <= 1.0")
			
			// Verify all score components are in valid range
			components := decision.ScoreComponents
			assert.GreaterOrEqual(suite.T(), components.QueueImpact, 0.0)
			assert.LessOrEqual(suite.T(), components.QueueImpact, 1.0)
			
			assert.GreaterOrEqual(suite.T(), components.LoadBalance, 0.0)
			assert.LessOrEqual(suite.T(), components.LoadBalance, 1.0)
			
			assert.GreaterOrEqual(suite.T(), components.NetworkCost, 0.0)
			assert.LessOrEqual(suite.T(), components.NetworkCost, 1.0)
			
			assert.GreaterOrEqual(suite.T(), components.LatencyImpact, 0.0)
			assert.LessOrEqual(suite.T(), components.LatencyImpact, 1.0)
			
			assert.GreaterOrEqual(suite.T(), components.EnergyImpact, 0.0)
			assert.LessOrEqual(suite.T(), components.EnergyImpact, 1.0)
			
			assert.GreaterOrEqual(suite.T(), components.PolicyMatch, 0.0)
			assert.LessOrEqual(suite.T(), components.PolicyMatch, 1.0)
			
			// Verify confidence is in valid range
			assert.GreaterOrEqual(suite.T(), decision.Confidence, 0.0)
			assert.LessOrEqual(suite.T(), decision.Confidence, 1.0)
		})
	}
}

// Helper functions
func calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	return sum / time.Duration(len(latencies))
}

func calculatePercentileLatency(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	// Calculate percentile index
	index := int(math.Ceil(float64(percentile)/100.0*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}

// Run the test suite
func TestDecisionEngineSuite(t *testing.T) {
	suite.Run(t, new(DecisionEngineTestSuite))
}