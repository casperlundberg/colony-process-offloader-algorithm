package decision

import (
	"testing"
	"time"
	"math"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DecisionEngine test requirements:
// 1. Decision must be deterministic given same inputs
// 2. Decision must complete within 500ms
// 3. All scores must be in [0.0, 1.0] range
// 4. Decision quality must be explainable and auditable

type DecisionEngineTestSuite struct {
	suite.Suite
	engine *DecisionEngine
}

func (suite *DecisionEngineTestSuite) SetupTest() {
	weights := AdaptiveWeights{
		QueueDepth:    0.2,
		ProcessorLoad: 0.2,
		NetworkCost:   0.2,
		LatencyCost:   0.2,
		EnergyCost:    0.1,
		PolicyCost:    0.1,
	}
	
	suite.engine = NewDecisionEngine(weights)
}

// Test that decisions are deterministic given same inputs
func (suite *DecisionEngineTestSuite) TestDecisionDeterminism() {
	process := Process{
		ID:                "test-determinism",
		CPURequirement:    2.0,
		MemoryRequirement: 4 * 1024 * 1024 * 1024, // 4GB
		InputSize:         1024 * 1024,             // 1MB
		OutputSize:        512 * 1024,              // 512KB
		EstimatedDuration: 30 * time.Second,
		Priority:          5,
	}

	state := SystemState{
		QueueDepth:      25,
		QueueThreshold:  20,
		ComputeUsage:    0.75,
		MemoryUsage:     0.60,
		NetworkUsage:    0.40,
		MasterUsage:     0.30,
		Timestamp:       time.Now(),
	}

	targets := []OffloadTarget{
		{
			ID:               "target-1",
			Type:             "edge",
			TotalCapacity:    8.0,
			AvailableCapacity: 6.0,
			NetworkLatency:   15 * time.Millisecond,
			NetworkBandwidth: 100 * 1024 * 1024, // 100MB/s
			Reliability:      0.95,
			ComputeCost:      0.10,
		},
		{
			ID:               "target-2", 
			Type:             "cloud",
			TotalCapacity:    32.0,
			AvailableCapacity: 24.0,
			NetworkLatency:   50 * time.Millisecond,
			NetworkBandwidth: 50 * 1024 * 1024, // 50MB/s
			Reliability:      0.99,
			ComputeCost:      0.05,
		},
	}

	// Make the same decision multiple times
	decisions := []OffloadDecision{}
	for i := 0; i < 10; i++ {
		decision := suite.engine.ComputeOffloadDecision(process, state, targets)
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
	process := Process{
		ID:                "test-latency",
		CPURequirement:    4.0,
		MemoryRequirement: 8 * 1024 * 1024 * 1024,
		InputSize:         10 * 1024 * 1024,
		OutputSize:        5 * 1024 * 1024,
		EstimatedDuration: 60 * time.Second,
		Priority:          7,
	}

	state := SystemState{
		QueueDepth:   50,
		ComputeUsage: 0.85,
		MemoryUsage:  0.70,
		NetworkUsage: 0.50,
	}

	// Create many targets to stress-test decision latency
	targets := []OffloadTarget{}
	for i := 0; i < 20; i++ {
		target := OffloadTarget{
			ID:                fmt.Sprintf("target-%d", i),
			Type:              []string{"edge", "cloud", "fog"}[i%3],
			TotalCapacity:     float64(4 + i*2),
			AvailableCapacity: float64(2 + i),
			NetworkLatency:    time.Duration(10+i*5) * time.Millisecond,
			NetworkBandwidth:  float64((50 + i*10) * 1024 * 1024),
			Reliability:       0.8 + float64(i%20)*0.01,
			ComputeCost:       0.05 + float64(i)*0.01,
		}
		targets = append(targets, target)
	}

	// Test decision latency under normal conditions
	latencies := []time.Duration{}
	timeoutCount := 0
	iterations := 100

	for i := 0; i < iterations; i++ {
		start := time.Now()
		decision := suite.engine.ComputeOffloadDecision(process, state, targets)
		duration := time.Since(start)
		
		latencies = append(latencies, duration)
		
		// Verify decision is valid
		require.NotNil(suite.T(), decision, "Decision should not be nil")
		
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
		process  Process
		state    SystemState
		targets  []OffloadTarget
	}{
		{
			name: "extreme_high_load",
			process: Process{
				ID:                "extreme-process",
				CPURequirement:    16.0,
				MemoryRequirement: 64 * 1024 * 1024 * 1024, // 64GB
				InputSize:         1024 * 1024 * 1024,       // 1GB
				OutputSize:        512 * 1024 * 1024,        // 512MB
				Priority:          10,
			},
			state: SystemState{
				QueueDepth:   100,
				ComputeUsage: 0.99,
				MemoryUsage:  0.95,
				NetworkUsage: 0.90,
				MasterUsage:  0.85,
			},
			targets: []OffloadTarget{
				{
					ID:               "powerful-target",
					TotalCapacity:    64.0,
					AvailableCapacity: 32.0,
					NetworkLatency:   1 * time.Millisecond,
					NetworkBandwidth: 10000 * 1024 * 1024, // 10GB/s
					Reliability:      0.999,
				},
			},
		},
		{
			name: "extreme_low_load",
			process: Process{
				ID:              "tiny-process",
				CPURequirement:  0.1,
				InputSize:       1024, // 1KB
				OutputSize:      512,  // 512B
				Priority:        1,
			},
			state: SystemState{
				QueueDepth:   1,
				ComputeUsage: 0.01,
				MemoryUsage:  0.05,
				NetworkUsage: 0.01,
				MasterUsage:  0.02,
			},
			targets: []OffloadTarget{
				{
					ID:               "weak-target",
					TotalCapacity:    1.0,
					AvailableCapacity: 0.1,
					NetworkLatency:   1000 * time.Millisecond, // Very high latency
					NetworkBandwidth: 1024,                     // 1KB/s
					Reliability:      0.5,
				},
			},
		},
		{
			name: "mixed_scenario",
			process: Process{
				ID:              "mixed-process",
				CPURequirement:  4.0,
				InputSize:       50 * 1024 * 1024, // 50MB
				OutputSize:      25 * 1024 * 1024, // 25MB
				Priority:        5,
			},
			state: SystemState{
				QueueDepth:   25,
				ComputeUsage: 0.50,
				MemoryUsage:  0.60,
				NetworkUsage: 0.70,
				MasterUsage:  0.40,
			},
			targets: []OffloadTarget{
				{
					ID:               "fast-expensive",
					TotalCapacity:    8.0,
					AvailableCapacity: 4.0,
					NetworkLatency:   5 * time.Millisecond,
					NetworkBandwidth: 1000 * 1024 * 1024,
					Reliability:      0.95,
					ComputeCost:      0.50, // Expensive
				},
				{
					ID:               "slow-cheap",
					TotalCapacity:    16.0,
					AvailableCapacity: 12.0,
					NetworkLatency:   200 * time.Millisecond,
					NetworkBandwidth: 10 * 1024 * 1024,
					Reliability:      0.80,
					ComputeCost:      0.02, // Cheap
				},
			},
		},
	}

	for _, scenario := range testScenarios {
		suite.Run(scenario.name, func() {
			decision := suite.engine.ComputeOffloadDecision(
				scenario.process, scenario.state, scenario.targets)
			
			require.NotNil(suite.T(), decision, "Decision should not be nil")
			
			// Verify overall score is in range
			assert.GreaterOrEqual(suite.T(), decision.Score, 0.0,
				"Decision score should be >= 0.0")
			assert.LessOrEqual(suite.T(), decision.Score, 1.0,
				"Decision score should be <= 1.0")
			
			// Verify all score components are in range
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
			
			// Verify confidence is in range
			assert.GreaterOrEqual(suite.T(), decision.Confidence, 0.0)
			assert.LessOrEqual(suite.T(), decision.Confidence, 1.0)
		})
	}
}

// Test decision explainability and auditability
func (suite *DecisionEngineTestSuite) TestDecisionExplainabilityAndAuditability() {
	process := Process{
		ID:                "audit-test",
		CPURequirement:    4.0,
		MemoryRequirement: 8 * 1024 * 1024 * 1024,
		InputSize:         5 * 1024 * 1024,
		OutputSize:        2 * 1024 * 1024,
		Priority:          7,
		RealTime:          true,
		SafetyCritical:    false,
	}

	state := SystemState{
		QueueDepth:      40,
		QueueThreshold:  20,
		ComputeUsage:    0.80,
		MemoryUsage:     0.70,
		NetworkUsage:    0.30,
		MasterUsage:     0.50,
		Timestamp:       time.Now(),
	}

	targets := []OffloadTarget{
		{
			ID:               "fast-edge",
			Type:             "edge",
			TotalCapacity:    8.0,
			AvailableCapacity: 6.0,
			NetworkLatency:   10 * time.Millisecond,
			NetworkBandwidth: 200 * 1024 * 1024,
			Reliability:      0.95,
			ComputeCost:      0.15,
		},
		{
			ID:               "slow-cloud",
			Type:             "cloud",
			TotalCapacity:    32.0,
			AvailableCapacity: 24.0,
			NetworkLatency:   100 * time.Millisecond,
			NetworkBandwidth: 50 * 1024 * 1024,
			Reliability:      0.99,
			ComputeCost:      0.05,
		},
	}

	decision := suite.engine.ComputeOffloadDecision(process, state, targets)
	require.NotNil(suite.T(), decision)

	// Verify decision has all required audit information
	assert.NotEmpty(suite.T(), decision.DecisionID, "Decision should have unique ID")
	assert.NotZero(suite.T(), decision.DecisionTime, "Decision should have timestamp")
	assert.NotZero(suite.T(), decision.DecisionLatency, "Decision should record processing time")
	assert.NotEmpty(suite.T(), decision.AlgorithmVersion, "Decision should record algorithm version")

	// Verify score breakdown provides explainability
	components := decision.ScoreComponents
	assert.NotNil(suite.T(), components, "Score components should be available")
	
	// Verify weights used are recorded
	weights := components.WeightsUsed
	assert.NotZero(suite.T(), weights.QueueDepth)
	assert.NotZero(suite.T(), weights.ProcessorLoad)
	assert.NotZero(suite.T(), weights.NetworkCost)
	assert.NotZero(suite.T(), weights.LatencyCost)
	
	// Weights should sum to approximately 1.0
	totalWeight := weights.QueueDepth + weights.ProcessorLoad + 
	              weights.NetworkCost + weights.LatencyCost +
	              weights.EnergyCost + weights.PolicyCost
	assert.InDelta(suite.T(), 1.0, totalWeight, 0.01, "Weights should sum to 1.0")

	// Test decision reasoning generation
	reasoning := suite.engine.GenerateDecisionReasoning(decision, process, state)
	assert.NotEmpty(suite.T(), reasoning, "Decision should generate reasoning")
	
	// Reasoning should mention key factors
	if decision.ShouldOffload {
		assert.Contains(suite.T(), reasoning, "offload", "Reasoning should mention offloading")
		if decision.Target != nil {
			assert.Contains(suite.T(), reasoning, decision.Target.ID, 
				"Reasoning should mention selected target")
		}
	} else {
		assert.Contains(suite.T(), reasoning, "local", "Reasoning should mention local execution")
	}

	// Reasoning should explain dominant factors
	maxComponentValue := math.Max(components.QueueImpact, 
		math.Max(components.LoadBalance,
			math.Max(components.NetworkCost, components.LatencyImpact)))
	
	if components.QueueImpact == maxComponentValue {
		assert.Contains(suite.T(), reasoning, "queue", "Reasoning should mention queue impact")
	} else if components.NetworkCost == maxComponentValue {
		assert.Contains(suite.T(), reasoning, "network", "Reasoning should mention network cost")
	} else if components.LatencyImpact == maxComponentValue {
		assert.Contains(suite.T(), reasoning, "latency", "Reasoning should mention latency")
	}

	// Test audit trail generation
	auditRecord := suite.engine.GenerateAuditRecord(decision, process, state, targets)
	assert.NotNil(suite.T(), auditRecord, "Should generate audit record")
	
	// Audit record should contain all relevant information
	assert.Equal(suite.T(), process.ID, auditRecord.ProcessID)
	assert.Equal(suite.T(), decision.DecisionID, auditRecord.DecisionID)
	assert.Contains(suite.T(), auditRecord.InputData, "SystemState")
	assert.Contains(suite.T(), auditRecord.InputData, "Process")
	assert.Contains(suite.T(), auditRecord.InputData, "Targets")
	assert.NotEmpty(suite.T(), auditRecord.OutputData)
	assert.NotEmpty(suite.T(), auditRecord.DecisionReasoning)
}

// Test score component calculations
func (suite *DecisionEngineTestSuite) TestScoreComponentCalculations() {
	// Test Queue Impact calculation
	suite.Run("QueueImpactCalculation", func() {
		// High queue pressure scenario
		highQueueState := SystemState{
			QueueDepth:     50,
			QueueThreshold: 20,
			ComputeUsage:   0.85,
		}
		
		highCapacityTarget := OffloadTarget{
			ID:                "high-capacity",
			TotalCapacity:     16.0,
			AvailableCapacity: 14.0,
		}
		
		queueImpact := suite.engine.evaluateQueueImpact(highQueueState, highCapacityTarget)
		
		assert.Greater(suite.T(), queueImpact, 0.7,
			"High queue pressure with high capacity target should have high queue impact")
		assert.LessOrEqual(suite.T(), queueImpact, 1.0)
		
		// Low queue pressure scenario
		lowQueueState := SystemState{
			QueueDepth:     5,
			QueueThreshold: 20,
			ComputeUsage:   0.30,
		}
		
		lowQueueImpact := suite.engine.evaluateQueueImpact(lowQueueState, highCapacityTarget)
		
		assert.Less(suite.T(), lowQueueImpact, 0.3,
			"Low queue pressure should have low queue impact")
		assert.Greater(suite.T(), lowQueueImpact, queueImpact,
			"Low queue should have lower impact than high queue")
	})

	// Test Network Cost calculation
	suite.Run("NetworkCostCalculation", func() {
		dataIntensiveProcess := Process{
			InputSize:  100 * 1024 * 1024, // 100MB
			OutputSize: 50 * 1024 * 1024,  // 50MB
		}
		
		fastTarget := OffloadTarget{
			ID:               "fast-network",
			NetworkLatency:   5 * time.Millisecond,
			NetworkBandwidth: 1000 * 1024 * 1024, // 1GB/s
			NetworkStability: 0.99,
		}
		
		slowTarget := OffloadTarget{
			ID:               "slow-network",
			NetworkLatency:   200 * time.Millisecond,
			NetworkBandwidth: 1 * 1024 * 1024, // 1MB/s
			NetworkStability: 0.70,
		}
		
		fastNetworkCost := suite.engine.evaluateNetworkCost(dataIntensiveProcess, fastTarget)
		slowNetworkCost := suite.engine.evaluateNetworkCost(dataIntensiveProcess, slowTarget)
		
		assert.Greater(suite.T(), fastNetworkCost, slowNetworkCost,
			"Fast network should have better (higher) network cost score than slow network")
		
		assert.Greater(suite.T(), fastNetworkCost, 0.8,
			"Fast network should have high network cost score")
		assert.Less(suite.T(), slowNetworkCost, 0.3,
			"Slow network should have low network cost score")
	})

	// Test Load Balance calculation
	suite.Run("LoadBalanceCalculation", func() {
		highLoadState := SystemState{
			ComputeUsage: 0.90,
			MemoryUsage:  0.85,
		}
		
		lowLoadState := SystemState{
			ComputeUsage: 0.20,
			MemoryUsage:  0.30,
		}
		
		target := OffloadTarget{
			ID:                "balance-target",
			TotalCapacity:     8.0,
			AvailableCapacity: 6.0,
		}
		
		highLoadBalance := suite.engine.evaluateLoadBalance(highLoadState, target)
		lowLoadBalance := suite.engine.evaluateLoadBalance(lowLoadState, target)
		
		assert.Greater(suite.T(), highLoadBalance, lowLoadBalance,
			"High load system should benefit more from load balancing")
		
		assert.Greater(suite.T(), highLoadBalance, 0.7,
			"High load should have high load balance score")
		assert.Less(suite.T(), lowLoadBalance, 0.4,
			"Low load should have low load balance score")
	})

	// Test Latency Impact calculation
	suite.Run("LatencyImpactCalculation", func() {
		realTimeProcess := Process{
			RealTime:         true,
			MaxDuration:      500 * time.Millisecond,
		}
		
		batchProcess := Process{
			RealTime:         false,
			MaxDuration:      30 * time.Minute,
		}
		
		lowLatencyTarget := OffloadTarget{
			NetworkLatency: 2 * time.Millisecond,
		}
		
		highLatencyTarget := OffloadTarget{
			NetworkLatency: 150 * time.Millisecond,
		}
		
		rtLowLatency := suite.engine.evaluateLatency(realTimeProcess, lowLatencyTarget)
		rtHighLatency := suite.engine.evaluateLatency(realTimeProcess, highLatencyTarget)
		batchLowLatency := suite.engine.evaluateLatency(batchProcess, lowLatencyTarget)
		batchHighLatency := suite.engine.evaluateLatency(batchProcess, highLatencyTarget)
		
		// Real-time processes should be more sensitive to latency
		assert.Greater(suite.T(), rtLowLatency, rtHighLatency,
			"Real-time process should prefer low-latency target")
		
		latencySensitivity := (rtLowLatency - rtHighLatency) - (batchLowLatency - batchHighLatency)
		assert.Greater(suite.T(), latencySensitivity, 0.1,
			"Real-time process should be more latency-sensitive than batch process")
	})
}

// Test pattern application
func (suite *DecisionEngineTestSuite) TestPatternApplication() {
	// Create a discovered pattern
	pattern := &DiscoveredPattern{
		ID:   "high-queue-cpu-pattern",
		Name: "High Queue + High CPU -> Offload to Edge",
		Conditions: []PatternCondition{
			{Field: "QueueDepth", Operator: GREATER_THAN, Value: 30},
			{Field: "ComputeUsage", Operator: GREATER_THAN, Value: 0.8},
			{Field: "Process.CPURequirement", Operator: GREATER_THAN, Value: 2.0},
		},
		RecommendedAction: OFFLOAD_TO,
		PreferredTargets:  []string{"edge"},
		SuccessRate:       0.90,
		Confidence:        0.85,
		ApplicationCount:  25,
	}
	
	suite.engine.AddPattern(pattern)

	// Create scenario that matches the pattern
	matchingProcess := Process{
		ID:             "pattern-match-process",
		CPURequirement: 4.0, // > 2.0
		Priority:       5,
	}

	matchingState := SystemState{
		QueueDepth:   40,   // > 30
		ComputeUsage: 0.85, // > 0.8
	}

	targets := []OffloadTarget{
		{
			ID:                "edge-target",
			Type:              "edge",
			TotalCapacity:     8.0,
			AvailableCapacity: 6.0,
		},
		{
			ID:                "cloud-target",
			Type:              "cloud",
			TotalCapacity:     32.0,
			AvailableCapacity: 24.0,
		},
	}

	decision := suite.engine.ComputeOffloadDecision(matchingProcess, matchingState, targets)
	
	// Verify pattern was applied
	assert.True(suite.T(), decision.ShouldOffload, "Pattern should recommend offloading")
	assert.NotNil(suite.T(), decision.AppliedPattern, "Decision should record applied pattern")
	assert.Equal(suite.T(), pattern.ID, decision.AppliedPattern.ID)
	
	// Should prefer edge target as specified by pattern
	if decision.Target != nil {
		assert.Equal(suite.T(), "edge-target", decision.Target.ID,
			"Pattern should prefer edge target")
	}
	
	// Confidence should be influenced by pattern confidence
	assert.Greater(suite.T(), decision.Confidence, 0.8,
		"Pattern-based decision should have high confidence")

	// Test scenario that doesn't match pattern
	nonMatchingState := SystemState{
		QueueDepth:   15,   // < 30 (doesn't match)
		ComputeUsage: 0.85, // > 0.8 (matches)
	}

	decisionNoPattern := suite.engine.ComputeOffloadDecision(
		matchingProcess, nonMatchingState, targets)
	
	assert.Nil(suite.T(), decisionNoPattern.AppliedPattern,
		"Non-matching scenario should not apply pattern")
}

// Test edge cases and error conditions
func (suite *DecisionEngineTestSuite) TestEdgeCasesAndErrorConditions() {
	// Empty targets list
	suite.Run("EmptyTargetsList", func() {
		process := Process{ID: "test", Priority: 5, EstimatedDuration: 30 * time.Second}
		state := SystemState{QueueDepth: 10, ComputeUsage: 0.5}
		
		decision := suite.engine.ComputeOffloadDecision(process, state, []OffloadTarget{})
		
		assert.False(suite.T(), decision.ShouldOffload,
			"Should not offload with no targets available")
		assert.Nil(suite.T(), decision.Target, "Target should be nil with empty list")
		assert.Equal(suite.T(), 0.0, decision.Score, "Score should be 0 with no options")
		assert.NotEmpty(suite.T(), decision.DecisionID, "Should still generate decision ID")
	})

	// Invalid process
	suite.Run("InvalidProcess", func() {
		invalidProcess := Process{
			ID:                "", // Invalid: empty ID
			CPURequirement:    -1, // Invalid: negative
			Priority:          0,  // Invalid: outside [1,10]
			EstimatedDuration: 0,  // Invalid: must be > 0
		}
		
		state := SystemState{QueueDepth: 10, ComputeUsage: 0.5}
		targets := []OffloadTarget{{ID: "valid-target", TotalCapacity: 4.0}}
		
		decision := suite.engine.ComputeOffloadDecision(invalidProcess, state, targets)
		
		assert.False(suite.T(), decision.ShouldOffload,
			"Should not offload invalid process")
		assert.Contains(suite.T(), decision.DecisionReasoning, "invalid",
			"Decision reasoning should mention process is invalid")
	})

	// Unreachable targets
	suite.Run("UnreachableTargets", func() {
		process := Process{ID: "test", Priority: 5, EstimatedDuration: 30 * time.Second}
		state := SystemState{QueueDepth: 30, ComputeUsage: 0.8} // High load
		
		unreachableTargets := []OffloadTarget{
			{
				ID:               "unreachable-1",
				TotalCapacity:    8.0,
				AvailableCapacity: 6.0,
				LastSeen:         time.Now().Add(-300 * time.Second), // 5 minutes ago
			},
			{
				ID:               "unreachable-2",
				TotalCapacity:    16.0,
				AvailableCapacity: 0, // No capacity available
			},
		}
		
		decision := suite.engine.ComputeOffloadDecision(process, state, unreachableTargets)
		
		// Might decide not to offload, or select least bad target
		if decision.ShouldOffload {
			assert.NotNil(suite.T(), decision.Target, "If offloading, should select a target")
			// Should prefer more recently seen target
			assert.NotEqual(suite.T(), "unreachable-1", decision.Target.ID,
				"Should not select very stale target")
		}
		
		assert.Contains(suite.T(), decision.DecisionReasoning, "target",
			"Reasoning should mention target availability issues")
	})

	// Extreme resource requirements
	suite.Run("ExtremeResourceRequirements", func() {
		hugeProcess := Process{
			ID:                "huge-process",
			CPURequirement:    1000,                    // Unrealistic requirement
			MemoryRequirement: 1024 * 1024 * 1024 * 1024, // 1TB
			InputSize:         1024 * 1024 * 1024 * 1024,  // 1TB
			Priority:          10,
			EstimatedDuration: 24 * time.Hour,
		}
		
		state := SystemState{QueueDepth: 50, ComputeUsage: 0.9}
		
		normalTargets := []OffloadTarget{
			{
				ID:                "normal-target",
				TotalCapacity:     16.0,
				AvailableCapacity: 12.0,
				MemoryTotal:       32 * 1024 * 1024 * 1024, // 32GB
				MemoryAvailable:   24 * 1024 * 1024 * 1024,
			},
		}
		
		decision := suite.engine.ComputeOffloadDecision(hugeProcess, state, normalTargets)
		
		// Should recognize resource constraints
		if decision.ShouldOffload {
			// If it decides to offload, score should be low due to resource mismatch
			assert.Less(suite.T(), decision.Score, 0.3,
				"Score should be low for resource-constrained decision")
		}
		
		assert.Contains(suite.T(), decision.DecisionReasoning, "resource",
			"Reasoning should mention resource constraints")
	})
}

func TestDecisionEngineTestSuite(t *testing.T) {
	suite.Run(t, new(DecisionEngineTestSuite))
}

// Helper functions

func calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	
	return total / time.Duration(len(latencies))
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
	
	index := (percentile * len(sorted)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}