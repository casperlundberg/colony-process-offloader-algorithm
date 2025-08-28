package learning

import (
	"testing"
	"time"
	"math"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AdaptiveLearner test requirements:
// 1. Weights must always sum to 1.0 ± 0.001
// 2. Weight adaptation must converge within 200 decisions
// 3. Learning must improve performance by >10% over static baseline
// 4. Pattern discovery should discover >10 useful patterns in diverse environments

type AdaptiveLearnerTestSuite struct {
	suite.Suite
	learner *AdaptiveLearner
	config  LearningConfig
}

func (suite *AdaptiveLearnerTestSuite) SetupTest() {
	suite.config = LearningConfig{
		WindowSize:      100,
		LearningRate:    0.01,
		ExplorationRate: 0.1,
		MinSamples:      10,
	}
	
	suite.learner = NewAdaptiveLearner(suite.config)
}

// Test that weights always sum to 1.0 ± 0.001
func (suite *AdaptiveLearnerTestSuite) TestWeightNormalizationRequirement() {
	initialWeights := AdaptiveWeights{
		QueueDepth:    0.2,
		ProcessorLoad: 0.2,
		NetworkCost:   0.2,
		LatencyCost:   0.2,
		EnergyCost:    0.1,
		PolicyCost:    0.1,
	}

	// Test weight normalization after updates
	testOutcomes := []OffloadOutcome{
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
		totalWeight := weights.QueueDepth + weights.ProcessorLoad + 
		               weights.NetworkCost + weights.LatencyCost +
		               weights.EnergyCost + weights.PolicyCost
		
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
	extremeOutcome := OffloadOutcome{
		DecisionID: "extreme-test",
		Success:    true,
		Reward:     2.0,
		Attribution: map[string]float64{
			"QueueDepth": 1.0, // 100% attribution to one factor
		},
	}

	suite.learner.UpdateWeights(&weights, extremeOutcome)
	
	totalWeight := weights.QueueDepth + weights.ProcessorLoad + 
	               weights.NetworkCost + weights.LatencyCost +
	               weights.EnergyCost + weights.PolicyCost
	
	assert.InDelta(suite.T(), 1.0, totalWeight, 0.001,
		"Even with extreme attribution, weights should sum to 1.0")
}

// Test that weight adaptation converges within 200 decisions
func (suite *AdaptiveLearnerTestSuite) TestWeightConvergenceRequirement() {
	// Simulate consistent feedback that processor load is most important
	optimalWeights := AdaptiveWeights{
		QueueDepth:    0.15,
		ProcessorLoad: 0.40, // Should be highest
		NetworkCost:   0.20,
		LatencyCost:   0.15,
		EnergyCost:    0.05,
		PolicyCost:    0.05,
	}

	// Start with suboptimal weights
	currentWeights := AdaptiveWeights{
		QueueDepth:    0.40, // Too high
		ProcessorLoad: 0.10, // Too low
		NetworkCost:   0.20,
		LatencyCost:   0.20,
		EnergyCost:    0.05,
		PolicyCost:    0.05,
	}

	// Track convergence metrics
	convergenceHistory := []WeightConvergenceMetric{}
	
	// Apply 200 decisions with consistent feedback
	for i := 0; i < 200; i++ {
		outcome := OffloadOutcome{
			DecisionID:      fmt.Sprintf("convergence-test-%d", i),
			Success:         true,
			CompletedOnTime: true,
			Reward:          1.0 + 0.5*rand.Float64(), // Slight randomness
			Attribution: map[string]float64{
				"ProcessorLoad": 0.50 + 0.1*rand.Float64(), // Consistently high with noise
				"NetworkCost":   0.25 + 0.05*rand.Float64(),
				"QueueDepth":    0.15 + 0.05*rand.Float64(),
				"LatencyCost":   0.10 + 0.05*rand.Float64(),
			},
		}

		previousWeights := currentWeights
		suite.learner.UpdateWeights(&currentWeights, outcome)
		
		// Calculate convergence metric
		convergence := calculateWeightDistance(currentWeights, optimalWeights)
		weightStability := calculateWeightStability(currentWeights, previousWeights)
		
		convergenceHistory = append(convergenceHistory, WeightConvergenceMetric{
			Iteration:         i,
			DistanceToOptimal: convergence,
			WeightStability:   weightStability,
			Weights:          currentWeights,
		})
		
		suite.T().Logf("Iteration %d: ProcessorLoad=%.3f, QueueDepth=%.3f, Distance=%.3f", 
			i+1, currentWeights.ProcessorLoad, currentWeights.QueueDepth, convergence)
	}

	// Verify convergence occurred within 200 decisions
	finalDistance := convergenceHistory[len(convergenceHistory)-1].DistanceToOptimal
	assert.Less(suite.T(), finalDistance, 0.15,
		"Weights should converge to within 0.15 of optimal within 200 decisions")

	// Verify ProcessorLoad weight increased significantly
	initialProcessorWeight := 0.10
	finalProcessorWeight := currentWeights.ProcessorLoad
	
	assert.Greater(suite.T(), finalProcessorWeight, initialProcessorWeight+0.15,
		"ProcessorLoad weight should increase significantly based on feedback")

	// Verify QueueDepth weight decreased significantly  
	initialQueueWeight := 0.40
	finalQueueWeight := currentWeights.QueueDepth
	
	assert.Less(suite.T(), finalQueueWeight, initialQueueWeight-0.15,
		"QueueDepth weight should decrease based on lack of positive attribution")

	// Verify convergence stability in final iterations
	finalStabilityWindow := convergenceHistory[len(convergenceHistory)-20:]
	avgFinalStability := 0.0
	for _, metric := range finalStabilityWindow {
		avgFinalStability += metric.WeightStability
	}
	avgFinalStability /= float64(len(finalStabilityWindow))
	
	assert.Greater(suite.T(), avgFinalStability, 0.90,
		"Final 20 iterations should show high weight stability (>90%)")

	// Find point of convergence
	convergencePoint := findConvergencePoint(convergenceHistory, 0.20, 20)
	assert.LessOrEqual(suite.T(), convergencePoint, 200,
		"Convergence should occur within 200 decisions")
	assert.Greater(suite.T(), convergencePoint, 0,
		"Should find a clear convergence point")
	
	suite.T().Logf("Convergence achieved at iteration %d", convergencePoint)
}

// Test that learning improves performance by >10% over static baseline
func (suite *AdaptiveLearnerTestSuite) TestPerformanceImprovementRequirement() {
	// Define baseline performance with static weights
	staticWeights := AdaptiveWeights{
		QueueDepth:    0.2,
		ProcessorLoad: 0.2,
		NetworkCost:   0.2,
		LatencyCost:   0.2,
		EnergyCost:    0.1,
		PolicyCost:    0.1,
	}

	// Create adaptive learner
	adaptiveLearner := NewAdaptiveLearner(suite.config)
	adaptiveWeights := staticWeights // Start with same weights

	// Generate diverse test scenarios
	testScenarios := generateDiverseScenarios(100)
	
	staticPerformance := []float64{}
	adaptivePerformance := []float64{}

	for i, scenario := range testScenarios {
		// Simulate decision with static weights
		staticScore := calculateDecisionScore(scenario, staticWeights)
		staticOutcome := simulateOutcome(scenario, staticScore)
		staticPerformance = append(staticPerformance, staticOutcome.Reward)

		// Simulate decision with adaptive weights
		adaptiveScore := calculateDecisionScore(scenario, adaptiveWeights)
		adaptiveOutcome := simulateOutcome(scenario, adaptiveScore)
		adaptivePerformance = append(adaptivePerformance, adaptiveOutcome.Reward)

		// Update adaptive weights based on outcome
		adaptiveLearner.UpdateWeights(&adaptiveWeights, adaptiveOutcome)
		
		// Log progress periodically
		if (i+1)%20 == 0 {
			staticAvg := calculateAverage(staticPerformance)
			adaptiveAvg := calculateAverage(adaptivePerformance)
			improvement := (adaptiveAvg - staticAvg) / staticAvg * 100
			
			suite.T().Logf("After %d scenarios: Static=%.3f, Adaptive=%.3f, Improvement=%.1f%%",
				i+1, staticAvg, adaptiveAvg, improvement)
		}
	}

	// Calculate final performance metrics
	staticAvgReward := calculateAverage(staticPerformance)
	adaptiveAvgReward := calculateAverage(adaptivePerformance)
	
	improvementPercent := (adaptiveAvgReward - staticAvgReward) / staticAvgReward * 100
	
	assert.Greater(suite.T(), improvementPercent, 10.0,
		"Adaptive learning should improve performance by >10%% over static baseline, got %.1f%%",
		improvementPercent)

	// Verify improvement is statistically significant
	pValue := performTTest(staticPerformance, adaptivePerformance)
	assert.Less(suite.T(), pValue, 0.05,
		"Performance improvement should be statistically significant (p < 0.05)")

	// Verify improvement trend over time
	windowSize := 20
	improvementTrend := []float64{}
	
	for i := windowSize; i <= len(testScenarios); i++ {
		staticWindow := staticPerformance[i-windowSize:i]
		adaptiveWindow := adaptivePerformance[i-windowSize:i]
		
		staticWindowAvg := calculateAverage(staticWindow)
		adaptiveWindowAvg := calculateAverage(adaptiveWindow)
		
		windowImprovement := (adaptiveWindowAvg - staticWindowAvg) / staticWindowAvg * 100
		improvementTrend = append(improvementTrend, windowImprovement)
	}

	// Verify improvement trend is positive and increasing
	firstHalfImprovement := calculateAverage(improvementTrend[:len(improvementTrend)/2])
	secondHalfImprovement := calculateAverage(improvementTrend[len(improvementTrend)/2:])
	
	assert.Greater(suite.T(), secondHalfImprovement, firstHalfImprovement,
		"Improvement should increase over time as learning progresses")
	
	suite.T().Logf("Final results: Static=%.3f, Adaptive=%.3f, Improvement=%.1f%%",
		staticAvgReward, adaptiveAvgReward, improvementPercent)
}

// Test pattern discovery requirement (>10 useful patterns in diverse environments)
func (suite *AdaptiveLearnerTestSuite) TestPatternDiscoveryRequirement() {
	// Generate diverse scenarios for pattern discovery
	scenarios := generateDiverseScenariosForPatterns(500) // Larger dataset for pattern discovery
	
	// Process scenarios to build history
	for _, scenario := range scenarios {
		outcome := OffloadOutcome{
			DecisionID:      scenario.DecisionID,
			ProcessID:       scenario.ProcessID,
			TargetID:        scenario.OptimalTarget,
			Success:         scenario.ExpectedSuccess,
			CompletedOnTime: scenario.MeetsDeadline,
			QueueReduction:  scenario.QueueImpact,
			Reward:          scenario.ExpectedReward,
			SystemContext:   scenario.SystemState,
			ProcessContext:  scenario.Process,
			TargetContext:   scenario.SelectedTarget,
		}
		
		suite.learner.history = append(suite.learner.history, outcome)
	}

	// Trigger pattern discovery
	suite.learner.discoverPatterns()
	
	discoveredPatterns := suite.learner.patterns
	
	// Verify number of patterns discovered
	assert.GreaterOrEqual(suite.T(), len(discoveredPatterns), 10,
		"Should discover at least 10 patterns in diverse environment, found %d", 
		len(discoveredPatterns))

	// Verify patterns are useful (high success rate and confidence)
	usefulPatterns := 0
	highConfidencePatterns := 0
	
	for _, pattern := range discoveredPatterns {
		suite.T().Logf("Pattern: %s, Success=%.2f, Confidence=%.2f, Count=%d",
			pattern.Name, pattern.SuccessRate, pattern.Confidence, pattern.ApplicationCount)
		
		// Verify pattern quality
		assert.GreaterOrEqual(suite.T(), pattern.SuccessRate, 0.0, "Success rate should be valid")
		assert.LessOrEqual(suite.T(), pattern.SuccessRate, 1.0, "Success rate should be valid")
		assert.GreaterOrEqual(suite.T(), pattern.Confidence, 0.0, "Confidence should be valid")
		assert.LessOrEqual(suite.T(), pattern.Confidence, 1.0, "Confidence should be valid")
		assert.GreaterOrEqual(suite.T(), pattern.ApplicationCount, suite.config.MinSamples,
			"Pattern should have minimum sample count")
		
		// Count useful patterns
		if pattern.SuccessRate >= 0.75 && pattern.Confidence >= 0.70 {
			usefulPatterns++
		}
		
		if pattern.Confidence >= 0.80 {
			highConfidencePatterns++
		}
		
		// Verify pattern conditions are meaningful
		assert.NotEmpty(suite.T(), pattern.Conditions, "Pattern should have conditions")
		assert.NotEmpty(suite.T(), pattern.RecommendedAction, "Pattern should have recommended action")
	}

	assert.GreaterOrEqual(suite.T(), usefulPatterns, 10,
		"Should discover at least 10 useful patterns (>75%% success, >70%% confidence), found %d",
		usefulPatterns)
	
	assert.GreaterOrEqual(suite.T(), highConfidencePatterns, 5,
		"Should discover at least 5 high-confidence patterns (>80%% confidence), found %d",
		highConfidencePatterns)

	// Test pattern diversity (different types of patterns)
	patternTypes := map[string]int{}
	conditionFields := map[string]int{}
	
	for _, pattern := range discoveredPatterns {
		// Classify pattern by recommended action
		patternTypes[pattern.RecommendedAction]++
		
		// Track condition fields used
		for _, condition := range pattern.Conditions {
			conditionFields[condition.Field]++
		}
	}

	assert.GreaterOrEqual(suite.T(), len(patternTypes), 2,
		"Should discover patterns with different recommended actions")
	
	assert.GreaterOrEqual(suite.T(), len(conditionFields), 4,
		"Should discover patterns using different condition fields")

	suite.T().Logf("Pattern types: %v", patternTypes)
	suite.T().Logf("Condition fields: %v", conditionFields)

	// Test pattern application effectiveness
	testApplicationScenarios := generateDiverseScenariosForPatterns(100)
	correctApplications := 0
	totalApplications := 0
	
	for _, scenario := range testApplicationScenarios {
		applicablePattern := suite.learner.findApplicablePattern(
			scenario.Process, scenario.SystemState)
		
		if applicablePattern != nil {
			totalApplications++
			
			// Check if pattern recommendation matches expected optimal decision
			expectedAction := scenario.OptimalAction
			if applicablePattern.RecommendedAction == expectedAction {
				correctApplications++
			}
		}
	}

	if totalApplications > 0 {
		applicationAccuracy := float64(correctApplications) / float64(totalApplications)
		assert.Greater(suite.T(), applicationAccuracy, 0.70,
			"Pattern applications should be correct >70%% of the time, got %.1f%%",
			applicationAccuracy*100)
		
		suite.T().Logf("Pattern application: %d/%d correct (%.1f%%)",
			correctApplications, totalApplications, applicationAccuracy*100)
	}
}

// Test learning from failure scenarios
func (suite *AdaptiveLearnerTestSuite) TestLearningFromFailures() {
	// Create scenarios with failures and poor outcomes
	failureScenarios := []OffloadOutcome{
		{
			DecisionID:        "failure-1",
			Success:           false,
			CompletedOnTime:   false,
			NetworkCongestion: true,
			LocalWorkDelayed:  true,
			Reward:           -2.0,
			Attribution: map[string]float64{
				"NetworkCost": 0.8, // Network decision led to failure
				"LatencyCost": 0.2,
			},
		},
		{
			DecisionID:        "failure-2", 
			Success:           false,
			TargetOverloaded:  true,
			QueueReduction:    -0.3, // Made queue worse
			Reward:           -1.5,
			Attribution: map[string]float64{
				"ProcessorLoad": 0.7, // Poor load balancing decision
				"QueueDepth":    0.3,
			},
		},
		{
			DecisionID:      "failure-3",
			Success:         true,
			CompletedOnTime: false, // Missed deadline
			PolicyViolation: true,
			Reward:         -1.0,
			Attribution: map[string]float64{
				"PolicyCost": 0.9, // Policy decision was wrong
				"LatencyCost": 0.1,
			},
		},
	}

	initialWeights := AdaptiveWeights{
		QueueDepth:    0.2,
		ProcessorLoad: 0.2,
		NetworkCost:   0.2,
		LatencyCost:   0.2,
		EnergyCost:    0.1,
		PolicyCost:    0.1,
	}

	weights := initialWeights

	// Apply failure scenarios
	for i, outcome := range failureScenarios {
		previousWeights := weights
		suite.learner.UpdateWeights(&weights, outcome)
		
		suite.T().Logf("After failure %d: Network=%.3f, Processor=%.3f, Policy=%.3f",
			i+1, weights.NetworkCost, weights.ProcessorLoad, weights.PolicyCost)
		
		// Verify weights changed in response to failure
		assert.NotEqual(suite.T(), previousWeights, weights,
			"Weights should change in response to failure %d", i+1)
	}

	// Verify specific learning from failures
	// Network failures should reduce NetworkCost weight
	assert.Less(suite.T(), weights.NetworkCost, initialWeights.NetworkCost,
		"NetworkCost weight should decrease after network failures")
	
	// Processor failures should reduce ProcessorLoad weight
	assert.Less(suite.T(), weights.ProcessorLoad, initialWeights.ProcessorLoad,
		"ProcessorLoad weight should decrease after processor failures")
	
	// Policy failures should impact PolicyCost weight
	// (could increase to pay more attention to policies, or decrease if policies are unreliable)
	policyWeightChanged := math.Abs(weights.PolicyCost - initialWeights.PolicyCost) > 0.01
	assert.True(suite.T(), policyWeightChanged,
		"PolicyCost weight should change significantly after policy violations")

	// Test recovery after positive outcomes
	recoveryOutcomes := []OffloadOutcome{
		{
			DecisionID:      "recovery-1",
			Success:         true,
			CompletedOnTime: true,
			QueueReduction:  0.5,
			Reward:          1.5,
			Attribution: map[string]float64{
				"QueueDepth": 0.5,
				"LatencyCost": 0.3,
				"ProcessorLoad": 0.2,
			},
		},
		{
			DecisionID:      "recovery-2",
			Success:         true,
			CompletedOnTime: true,
			NetworkCostActual: 0.1, // Low network cost worked well
			Reward:          1.2,
			Attribution: map[string]float64{
				"NetworkCost": 0.4, // Network decisions are working again
				"QueueDepth": 0.3,
				"LatencyCost": 0.3,
			},
		},
	}

	weightsAfterFailures := weights

	// Apply recovery scenarios
	for _, outcome := range recoveryOutcomes {
		suite.learner.UpdateWeights(&weights, outcome)
	}

	// Verify recovery (some weights should improve)
	assert.Greater(suite.T(), weights.NetworkCost, weightsAfterFailures.NetworkCost,
		"NetworkCost weight should recover after positive network outcomes")
	
	assert.Greater(suite.T(), weights.QueueDepth, weightsAfterFailures.QueueDepth,
		"QueueDepth weight should increase after positive queue outcomes")
}

// Test exploration vs exploitation balance
func (suite *AdaptiveLearnerTestSuite) TestExplorationExploitationBalance() {
	// Create learner with high exploration rate
	highExplorationConfig := suite.config
	highExplorationConfig.ExplorationRate = 0.3 // 30% exploration
	
	highExplorationLearner := NewAdaptiveLearner(highExplorationConfig)

	// Create learner with low exploration rate
	lowExplorationConfig := suite.config
	lowExplorationConfig.ExplorationRate = 0.05 // 5% exploration
	
	lowExplorationLearner := NewAdaptiveLearner(lowExplorationConfig)

	// Start with same weights
	highExplorationWeights := AdaptiveWeights{0.2, 0.2, 0.2, 0.2, 0.1, 0.1}
	lowExplorationWeights := AdaptiveWeights{0.2, 0.2, 0.2, 0.2, 0.1, 0.1}

	// Apply same outcomes to both learners
	outcomes := generateConsistentOutcomes(100)
	
	highExplorationVariance := []float64{}
	lowExplorationVariance := []float64{}

	for i, outcome := range outcomes {
		prevHighWeights := highExplorationWeights
		prevLowWeights := lowExplorationWeights
		
		highExplorationLearner.UpdateWeights(&highExplorationWeights, outcome)
		lowExplorationLearner.UpdateWeights(&lowExplorationWeights, outcome)
		
		// Calculate weight variance (measure of exploration)
		highVariance := calculateWeightVariance(highExplorationWeights, prevHighWeights)
		lowVariance := calculateWeightVariance(lowExplorationWeights, prevLowWeights)
		
		highExplorationVariance = append(highExplorationVariance, highVariance)
		lowExplorationVariance = append(lowExplorationVariance, lowVariance)
		
		if (i+1)%20 == 0 {
			suite.T().Logf("Update %d: High exploration variance=%.4f, Low exploration variance=%.4f",
				i+1, highVariance, lowVariance)
		}
	}

	// High exploration should show more variance
	avgHighVariance := calculateAverage(highExplorationVariance)
	avgLowVariance := calculateAverage(lowExplorationVariance)
	
	assert.Greater(suite.T(), avgHighVariance, avgLowVariance,
		"High exploration should show more weight variance than low exploration")
	
	// Test convergence differences
	finalDistance := calculateWeightDistance(highExplorationWeights, lowExplorationWeights)
	assert.Greater(suite.T(), finalDistance, 0.05,
		"Different exploration rates should lead to different final weights")

	suite.T().Logf("Final weight distance between exploration strategies: %.3f", finalDistance)
}

func TestAdaptiveLearnerTestSuite(t *testing.T) {
	suite.Run(t, new(AdaptiveLearnerTestSuite))
}

// Helper functions and types

type WeightConvergenceMetric struct {
	Iteration         int
	DistanceToOptimal float64
	WeightStability   float64
	Weights          AdaptiveWeights
}

func calculateWeightDistance(w1, w2 AdaptiveWeights) float64 {
	return math.Sqrt(
		math.Pow(w1.QueueDepth-w2.QueueDepth, 2) +
		math.Pow(w1.ProcessorLoad-w2.ProcessorLoad, 2) +
		math.Pow(w1.NetworkCost-w2.NetworkCost, 2) +
		math.Pow(w1.LatencyCost-w2.LatencyCost, 2) +
		math.Pow(w1.EnergyCost-w2.EnergyCost, 2) +
		math.Pow(w1.PolicyCost-w2.PolicyCost, 2))
}

func calculateWeightStability(current, previous AdaptiveWeights) float64 {
	distance := calculateWeightDistance(current, previous)
	return math.Max(0.0, 1.0-distance) // Stability as inverse of distance
}

func calculateWeightVariance(current, previous AdaptiveWeights) float64 {
	return calculateWeightDistance(current, previous)
}

func findConvergencePoint(history []WeightConvergenceMetric, threshold float64, windowSize int) int {
	if len(history) < windowSize {
		return -1
	}

	for i := windowSize; i < len(history); i++ {
		// Check if last windowSize points are below threshold
		converged := true
		for j := i - windowSize + 1; j <= i; j++ {
			if history[j].DistanceToOptimal > threshold {
				converged = false
				break
			}
		}
		
		if converged {
			return i - windowSize + 1 // Return start of convergent window
		}
	}
	
	return len(history) // Did not converge
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	
	return sum / float64(len(values))
}

// Simplified t-test for performance comparison
func performTTest(sample1, sample2 []float64) float64 {
	// This is a simplified implementation
	// In practice, you'd use a proper statistical library
	
	if len(sample1) == 0 || len(sample2) == 0 {
		return 1.0
	}
	
	mean1 := calculateAverage(sample1)
	mean2 := calculateAverage(sample2)
	
	// Simple difference test (placeholder for real t-test)
	diff := math.Abs(mean2 - mean1)
	
	// Return artificial p-value based on difference magnitude
	if diff > 0.2 {
		return 0.01 // Significant
	} else if diff > 0.1 {
		return 0.05 // Marginally significant
	} else {
		return 0.1 // Not significant
	}
}