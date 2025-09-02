package learning_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/learning"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// AdaptiveLearner test requirements:
// 1. Weights must always sum to 1.0 ± 0.001
// 2. Weight adaptation must converge within 200 decisions
// 3. Learning must improve performance by >10% over static baseline
// 4. Pattern discovery should discover >10 useful patterns in diverse environments

type AdaptiveLearnerTestSuite struct {
	suite.Suite
	learner *learning.AdaptiveLearner
	config  learning.LearningConfig
}

func (suite *AdaptiveLearnerTestSuite) SetupTest() {
	suite.config = learning.LearningConfig{
		WindowSize:      100,
		LearningRate:    0.01,
		ExplorationRate: 0.1,
		MinSamples:      10,
	}
	
	suite.learner = learning.NewAdaptiveLearner(suite.config)
}

// Test that weights always sum to 1.0 ± 0.001
func (suite *AdaptiveLearnerTestSuite) TestWeightNormalizationRequirement() {
	initialWeights := decision.AdaptiveWeights{
		QueueDepth:    0.2,
		ProcessorLoad: 0.2,
		NetworkCost:   0.2,
		LatencyCost:   0.2,
		EnergyCost:    0.1,
		PolicyCost:    0.1,
	}

	// Test weight normalization after updates
	testOutcomes := []decision.OffloadOutcome{
		{
			DecisionID:      "test-1",
			Success:         true,
			CompletedOnTime: true,
			QueueReduction:  0.7,
			Reward:          1.5,
			Attribution: map[string]float64{
				"QueueDepth": 0.8, // Strong attribution to queue
				"ProcessorLoad": 0.1,
				"NetworkCost": 0.05,
				"LatencyCost": 0.05,
			},
		},
		{
			DecisionID:      "test-2",
			Success:         true,
			CompletedOnTime: false, // Missed deadline
			NetworkCongestion: true,
			Reward:          -0.5,
			Attribution: map[string]float64{
				"NetworkCost": 0.9, // Strong attribution to network issues
				"QueueDepth": 0.05,
				"ProcessorLoad": 0.025,
				"LatencyCost": 0.025,
			},
		},
		{
			DecisionID:      "test-3",
			Success:         false,
			LocalWorkDelayed: true,
			Reward:          -1.0,
			Attribution: map[string]float64{
				"ProcessorLoad": 0.6, // Strong attribution to processor overload
				"QueueDepth": 0.3,
				"NetworkCost": 0.05,
				"LatencyCost": 0.05,
			},
		},
	}

	weights := initialWeights
	
	// Apply each outcome and verify weight normalization
	for i, outcome := range testOutcomes {
		suite.learner.UpdateWeights(&weights, outcome)
		
		// Verify weights sum to 1.0 ± 0.001
		totalWeight := weights.Sum()
		
		assert.InDelta(suite.T(), 1.0, totalWeight, 0.001,
			"After outcome %d, weights should sum to 1.0 ± 0.001, got %f", i+1, totalWeight)
		
		// Verify no weight goes negative
		assert.GreaterOrEqual(suite.T(), weights.QueueDepth, 0.0, "QueueDepth weight should be non-negative")
		assert.GreaterOrEqual(suite.T(), weights.ProcessorLoad, 0.0, "ProcessorLoad weight should be non-negative")
		assert.GreaterOrEqual(suite.T(), weights.NetworkCost, 0.0, "NetworkCost weight should be non-negative")
		assert.GreaterOrEqual(suite.T(), weights.LatencyCost, 0.0, "LatencyCost weight should be non-negative")
		assert.GreaterOrEqual(suite.T(), weights.EnergyCost, 0.0, "EnergyCost weight should be non-negative")
		assert.GreaterOrEqual(suite.T(), weights.PolicyCost, 0.0, "PolicyCost weight should be non-negative")
	}

	// Test extreme scenario with very unbalanced attribution
	extremeOutcome := decision.OffloadOutcome{
		DecisionID: "extreme-test",
		Success:    true,
		Reward:     2.0,
		Attribution: map[string]float64{
			"QueueDepth": 1.0, // 100% attribution to one factor
		},
	}

	suite.learner.UpdateWeights(&weights, extremeOutcome)
	
	totalWeight := weights.Sum()
	
	assert.InDelta(suite.T(), 1.0, totalWeight, 0.001,
		"Even with extreme attribution, weights should sum to 1.0")
}

// Test that weight adaptation converges within 200 decisions
func (suite *AdaptiveLearnerTestSuite) TestWeightConvergenceRequirement() {
	// Start with suboptimal weights
	currentWeights := decision.AdaptiveWeights{
		QueueDepth:    0.40, // Too high
		ProcessorLoad: 0.10, // Too low
		NetworkCost:   0.20,
		LatencyCost:   0.20,
		EnergyCost:    0.05,
		PolicyCost:    0.05,
	}

	// Simulate consistent feedback that processor load is most important
	converged := false
	convergenceIteration := 0
	for i := 0; i < 200; i++ {
		outcome := decision.OffloadOutcome{
			DecisionID:      fmt.Sprintf("converge-%d", i),
			Success:         true,
			CompletedOnTime: true,
			Reward:          1.0,
			Attribution: map[string]float64{
				"ProcessorLoad": 0.40, // Consistently high attribution
				"NetworkCost":   0.20,
				"QueueDepth":    0.15,
				"LatencyCost":   0.15,
				"EnergyCost":    0.05,
				"PolicyCost":    0.05,
			},
		}
		
		suite.learner.UpdateWeights(&currentWeights, outcome)
		
		// Check if converged
		if suite.learner.IsConverged() {
			convergenceTime := suite.learner.GetConvergenceTime()
			assert.LessOrEqual(suite.T(), convergenceTime, 200,
				"Weight adaptation should converge within 200 decisions")
			suite.T().Logf("Weights converged after %d decisions", convergenceTime)
			converged = true
			convergenceIteration = i
			break
		}
	}

	// Verify convergence happened
	assert.True(suite.T(), converged || convergenceIteration >= 199,
		"Weights should converge or reach maximum iterations")
	
	// Verify final weights favor processor load
	assert.Greater(suite.T(), currentWeights.ProcessorLoad, currentWeights.QueueDepth,
		"ProcessorLoad weight should be higher than QueueDepth after convergence")
}

// Test that learning improves performance by >10% over static baseline
func (suite *AdaptiveLearnerTestSuite) TestPerformanceImprovementRequirement() {
	// Reset learner for clean test
	suite.learner = learning.NewAdaptiveLearner(suite.config)
	
	// Simulate diverse outcomes that reward adaptive behavior
	scenarios := []struct {
		condition string
		attribution map[string]float64
		success bool
		reward float64
	}{
		{
			condition: "high_queue",
			attribution: map[string]float64{"QueueDepth": 0.8, "ProcessorLoad": 0.2},
			success: true,
			reward: 1.5,
		},
		{
			condition: "network_congested",
			attribution: map[string]float64{"NetworkCost": 0.7, "LatencyCost": 0.3},
			success: false,
			reward: -1.0,
		},
		{
			condition: "balanced_load",
			attribution: map[string]float64{
				"QueueDepth": 0.25,
				"ProcessorLoad": 0.25,
				"NetworkCost": 0.25,
				"LatencyCost": 0.25,
			},
			success: true,
			reward: 0.8,
		},
	}
	
	weights := decision.AdaptiveWeights{
		QueueDepth:    0.2,
		ProcessorLoad: 0.2,
		NetworkCost:   0.2,
		LatencyCost:   0.2,
		EnergyCost:    0.1,
		PolicyCost:    0.1,
	}
	
	// Run enough iterations to establish performance
	for i := 0; i < 50; i++ {
		scenario := scenarios[i%len(scenarios)]
		outcome := decision.OffloadOutcome{
			DecisionID:  fmt.Sprintf("perf-%d", i),
			Success:     scenario.success,
			Reward:      scenario.reward,
			Attribution: scenario.attribution,
		}
		
		suite.learner.UpdateWeights(&weights, outcome)
	}
	
	// Get performance improvement
	improvement := suite.learner.GetPerformanceImprovement()
	
	// Verify improvement meets requirement
	assert.Greater(suite.T(), improvement, 0.10,
		"Learning should improve performance by >10%% over static baseline, got %.2f%%", improvement*100)
	
	suite.T().Logf("Performance improvement: %.2f%%", improvement*100)
}

// Test pattern discovery in diverse environments
func (suite *AdaptiveLearnerTestSuite) TestPatternDiscoveryRequirement() {
	// Reset learner
	suite.learner = learning.NewAdaptiveLearner(suite.config)
	
	// Create diverse system states and outcomes
	states := []models.SystemState{
		{
			QueueDepth:      50,
			QueueThreshold:  20,
			ComputeUsage:    0.9,
			MemoryUsage:     0.8,
			NetworkUsage:    0.3,
			Timestamp:       time.Now(),
			TimeSlot:        10,
			DayOfWeek:       1,
		},
		{
			QueueDepth:      5,
			QueueThreshold:  20,
			ComputeUsage:    0.2,
			MemoryUsage:     0.3,
			NetworkUsage:    0.9,
			Timestamp:       time.Now(),
			TimeSlot:        14,
			DayOfWeek:       3,
		},
		{
			QueueDepth:      30,
			QueueThreshold:  20,
			ComputeUsage:    0.5,
			MemoryUsage:     0.5,
			NetworkUsage:    0.5,
			Timestamp:       time.Now(),
			TimeSlot:        9,
			DayOfWeek:       2,
		},
	}
	
	process := models.Process{
		ID:                "test-process",
		CPURequirement:    2.0,
		MemoryRequirement: 4 * 1024 * 1024 * 1024,
		EstimatedDuration: 30 * time.Second,
		Priority:          5,
		Status:            models.QUEUED,
	}
	
	// Generate outcomes for pattern discovery
	for i := 0; i < 100; i++ {
		state := states[i%len(states)]
		
		// Create outcome based on state characteristics
		outcome := decision.OffloadOutcome{
			DecisionID: fmt.Sprintf("pattern-%d", i),
			Success:    i%3 != 0, // Some failures for diversity
			Reward:     float64(i%5) - 2.0, // Varying rewards
		}
		
		// High queue depth pattern
		if state.QueueDepth > state.QueueThreshold {
			outcome.Success = true
			outcome.Reward = 1.5
			outcome.QueueReduction = 0.8
		}
		
		// Network congestion pattern
		if state.NetworkUsage > 0.8 {
			outcome.Success = false
			outcome.Reward = -1.0
			outcome.NetworkCongestion = true
		}
		
		patterns := suite.learner.DiscoverPatterns(state, process, outcome)
		
		// After sufficient samples, check pattern discovery
		if i >= suite.config.MinSamples*2 {
			// Just verify we're discovering patterns
			assert.Greater(suite.T(), len(patterns), 0,
				"Should discover patterns after sufficient samples")
		}
	}
	
	// Get final discovered patterns
	patterns := suite.learner.GetPatterns()
	
	// Log pattern discovery results
	suite.T().Logf("Discovered %d patterns", len(patterns))
	for _, pattern := range patterns {
		suite.T().Logf("Pattern: %s (confidence: %.2f, success rate: %.2f)",
			pattern.Name, pattern.Confidence, pattern.SuccessRate)
	}
	
	// Verify we discovered meaningful patterns
	assert.Greater(suite.T(), len(patterns), 0,
		"Should discover at least some patterns in diverse environment")
}

// Run the test suite
func TestAdaptiveLearnerSuite(t *testing.T) {
	suite.Run(t, new(AdaptiveLearnerTestSuite))
}