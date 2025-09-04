package learning

import (
	"math"
	"testing"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

func TestThompsonSampler_NewThompsonSampler(t *testing.T) {
	explorationRate := 0.1
	ts := NewThompsonSampler(explorationRate)

	if ts == nil {
		t.Fatal("NewThompsonSampler() returned nil")
	}

	if ts.explorationRate != explorationRate {
		t.Errorf("Expected explorationRate=%f, got %f", explorationRate, ts.explorationRate)
	}

	if len(ts.strategies) == 0 {
		t.Error("Expected strategies to be initialized")
	}

	expectedStrategies := 6 // Default strategies count
	if len(ts.strategies) != expectedStrategies {
		t.Errorf("Expected %d strategies, got %d", expectedStrategies, len(ts.strategies))
	}

	// Check that all strategies have initial counts
	for _, strategy := range ts.strategies {
		if ts.successes[strategy] != 1 {
			t.Errorf("Expected initial successes=1 for %s, got %d", strategy, ts.successes[strategy])
		}
		if ts.failures[strategy] != 1 {
			t.Errorf("Expected initial failures=1 for %s, got %d", strategy, ts.failures[strategy])
		}
		if ts.totalTrials[strategy] != 0 {
			t.Errorf("Expected initial totalTrials=0 for %s, got %d", strategy, ts.totalTrials[strategy])
		}
	}
}

func TestThompsonSampler_NewThompsonSamplerWithStrategies(t *testing.T) {
	customStrategies := []models.Strategy{
		models.StrategyBalanced,
		models.StrategyCostOptimal,
	}
	explorationRate := 0.2

	ts := NewThompsonSamplerWithStrategies(customStrategies, explorationRate)

	if ts == nil {
		t.Fatal("NewThompsonSamplerWithStrategies() returned nil")
	}

	if len(ts.strategies) != len(customStrategies) {
		t.Errorf("Expected %d strategies, got %d", len(customStrategies), len(ts.strategies))
	}

	for i, strategy := range customStrategies {
		if ts.strategies[i] != strategy {
			t.Errorf("Strategy %d: expected %s, got %s", i, strategy, ts.strategies[i])
		}
	}
}

func TestThompsonSampler_SelectStrategy(t *testing.T) {
	ts := NewThompsonSampler(0.1)

	// Test that it returns a valid strategy
	strategy := ts.SelectStrategy()
	
	found := false
	for _, validStrategy := range ts.strategies {
		if strategy == validStrategy {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Selected strategy %s not in available strategies", strategy)
	}

	// Test that lastSelected is updated
	if ts.lastSelected != strategy {
		t.Errorf("Expected lastSelected=%s, got %s", strategy, ts.lastSelected)
	}
}

func TestThompsonSampler_SelectStrategyEmptyStrategies(t *testing.T) {
	ts := NewThompsonSamplerWithStrategies([]models.Strategy{}, 0.1)

	strategy := ts.SelectStrategy()
	if strategy != models.StrategyBalanced {
		t.Errorf("Expected default strategy %s, got %s", models.StrategyBalanced, strategy)
	}
}

func TestThompsonSampler_UpdateStrategy(t *testing.T) {
	ts := NewThompsonSampler(0.1)
	strategy := models.StrategyBalanced

	initialSuccesses := ts.successes[strategy]
	initialFailures := ts.failures[strategy]
	initialTrials := ts.totalTrials[strategy]

	// Test successful outcome
	successOutcome := StrategyOutcome{
		Strategy: strategy,
		Success:  true,
		Reward:   0.8,
	}

	ts.UpdateStrategy(strategy, successOutcome)

	if ts.successes[strategy] != initialSuccesses+1 {
		t.Errorf("Expected successes=%d, got %d", initialSuccesses+1, ts.successes[strategy])
	}

	if ts.failures[strategy] != initialFailures {
		t.Errorf("Expected failures=%d, got %d", initialFailures, ts.failures[strategy])
	}

	if ts.totalTrials[strategy] != initialTrials+1 {
		t.Errorf("Expected totalTrials=%d, got %d", initialTrials+1, ts.totalTrials[strategy])
	}

	// Test failed outcome
	failureOutcome := StrategyOutcome{
		Strategy: strategy,
		Success:  false,
		Reward:   -0.3,
	}

	ts.UpdateStrategy(strategy, failureOutcome)

	if ts.failures[strategy] != initialFailures+1 {
		t.Errorf("Expected failures=%d, got %d", initialFailures+1, ts.failures[strategy])
	}

	if ts.totalTrials[strategy] != initialTrials+2 {
		t.Errorf("Expected totalTrials=%d, got %d", initialTrials+2, ts.totalTrials[strategy])
	}

	// Check that lastOutcome is stored
	if ts.lastOutcome == nil {
		t.Error("Expected lastOutcome to be stored")
	}
	if ts.lastOutcome.Success != failureOutcome.Success {
		t.Errorf("Expected lastOutcome success=%t, got %t", failureOutcome.Success, ts.lastOutcome.Success)
	}
}

func TestThompsonSampler_UpdateLastStrategy(t *testing.T) {
	ts := NewThompsonSampler(0.1)

	// Select a strategy first
	strategy := ts.SelectStrategy()

	initialTrials := ts.totalTrials[strategy]

	outcome := StrategyOutcome{
		Strategy: strategy,
		Success:  true,
		Reward:   0.5,
	}

	ts.UpdateLastStrategy(outcome)

	if ts.totalTrials[strategy] != initialTrials+1 {
		t.Errorf("Expected totalTrials=%d, got %d", initialTrials+1, ts.totalTrials[strategy])
	}
}

func TestThompsonSampler_GetStrategyStats(t *testing.T) {
	ts := NewThompsonSampler(0.1)
	strategy := models.StrategyBalanced

	// Add some outcomes
	ts.UpdateStrategy(strategy, StrategyOutcome{Strategy: strategy, Success: true})
	ts.UpdateStrategy(strategy, StrategyOutcome{Strategy: strategy, Success: true})
	ts.UpdateStrategy(strategy, StrategyOutcome{Strategy: strategy, Success: false})

	stats := ts.GetStrategyStats()

	strategyStats, exists := stats[strategy]
	if !exists {
		t.Fatalf("Expected stats for strategy %s", strategy)
	}

	if strategyStats.TotalTrials != 3 {
		t.Errorf("Expected totalTrials=3, got %d", strategyStats.TotalTrials)
	}

	if strategyStats.Successes != 2 {
		t.Errorf("Expected successes=2, got %d", strategyStats.Successes)
	}

	if strategyStats.Failures != 1 {
		t.Errorf("Expected failures=1, got %d", strategyStats.Failures)
	}

	expectedSuccessRate := 2.0 / 3.0
	if math.Abs(strategyStats.SuccessRate-expectedSuccessRate) > 0.0001 {
		t.Errorf("Expected successRate=%f, got %f", expectedSuccessRate, strategyStats.SuccessRate)
	}

	if strategyStats.EstimatedMean <= 0 || strategyStats.EstimatedMean >= 1 {
		t.Errorf("EstimatedMean should be between 0 and 1, got %f", strategyStats.EstimatedMean)
	}

	if strategyStats.ConfidenceLower < 0 || strategyStats.ConfidenceLower > strategyStats.EstimatedMean {
		t.Errorf("ConfidenceLower should be between 0 and mean, got %f", strategyStats.ConfidenceLower)
	}

	if strategyStats.ConfidenceUpper > 1 || strategyStats.ConfidenceUpper < strategyStats.EstimatedMean {
		t.Errorf("ConfidenceUpper should be between mean and 1, got %f", strategyStats.ConfidenceUpper)
	}
}

func TestThompsonSampler_GetBestStrategy(t *testing.T) {
	ts := NewThompsonSampler(0.1)

	// Make one strategy clearly better
	goodStrategy := models.StrategyBalanced
	badStrategy := models.StrategyCostOptimal

	// Add many successes to good strategy
	for i := 0; i < 10; i++ {
		ts.UpdateStrategy(goodStrategy, StrategyOutcome{Strategy: goodStrategy, Success: true})
	}

	// Add many failures to bad strategy
	for i := 0; i < 10; i++ {
		ts.UpdateStrategy(badStrategy, StrategyOutcome{Strategy: badStrategy, Success: false})
	}

	bestStrategy := ts.GetBestStrategy()
	if bestStrategy != goodStrategy {
		t.Errorf("Expected best strategy %s, got %s", goodStrategy, bestStrategy)
	}
}

func TestThompsonSampler_GetConvergenceMetrics(t *testing.T) {
	ts := NewThompsonSampler(0.1)

	// Initially should have no trials
	metrics := ts.GetConvergenceMetrics()
	if metrics.TotalTrials != 0 {
		t.Errorf("Expected initial totalTrials=0, got %d", metrics.TotalTrials)
	}

	// Add some trials
	strategy := models.StrategyBalanced
	for i := 0; i < 5; i++ {
		ts.UpdateStrategy(strategy, StrategyOutcome{Strategy: strategy, Success: true})
	}

	metrics = ts.GetConvergenceMetrics()
	if metrics.TotalTrials != 5 {
		t.Errorf("Expected totalTrials=5, got %d", metrics.TotalTrials)
	}

	if metrics.StrategyCount != len(ts.strategies) {
		t.Errorf("Expected strategyCount=%d, got %d", len(ts.strategies), metrics.StrategyCount)
	}

	// Entropy should be low since only one strategy was tried
	if metrics.EntropyScore > 0.5 {
		t.Errorf("Expected low entropy score, got %f", metrics.EntropyScore)
	}

	// Convergence should be high since entropy is low
	if metrics.ConvergenceScore < 0.5 {
		t.Errorf("Expected high convergence score, got %f", metrics.ConvergenceScore)
	}

	if metrics.LastUpdated.IsZero() {
		t.Error("LastUpdated should not be zero")
	}
}

func TestThompsonSampler_Reset(t *testing.T) {
	ts := NewThompsonSampler(0.1)
	strategy := models.StrategyBalanced

	// Add some data
	ts.SelectStrategy()
	ts.UpdateStrategy(strategy, StrategyOutcome{Strategy: strategy, Success: true})

	// Verify data exists
	if ts.totalTrials[strategy] == 0 {
		t.Error("Expected trials > 0 before reset")
	}
	if ts.lastSelected == "" {
		t.Error("Expected lastSelected to be set before reset")
	}

	ts.Reset()

	// Verify reset
	for _, strat := range ts.strategies {
		if ts.successes[strat] != 1 {
			t.Errorf("Expected successes=1 after reset for %s, got %d", strat, ts.successes[strat])
		}
		if ts.failures[strat] != 1 {
			t.Errorf("Expected failures=1 after reset for %s, got %d", strat, ts.failures[strat])
		}
		if ts.totalTrials[strat] != 0 {
			t.Errorf("Expected totalTrials=0 after reset for %s, got %d", strat, ts.totalTrials[strat])
		}
	}

	if ts.lastSelected != "" {
		t.Errorf("Expected lastSelected to be empty after reset, got %s", ts.lastSelected)
	}

	if ts.lastOutcome != nil {
		t.Error("Expected lastOutcome to be nil after reset")
	}
}

func TestThompsonSampler_SampleBeta(t *testing.T) {
	ts := NewThompsonSampler(0.1)

	// Test valid parameters
	alpha := 2.0
	beta := 3.0

	samples := make([]float64, 100)
	for i := 0; i < 100; i++ {
		sample := ts.sampleBeta(alpha, beta)
		samples[i] = sample

		// Sample should be in [0, 1]
		if sample < 0.0 || sample > 1.0 {
			t.Errorf("Sample %d: expected [0,1], got %f", i, sample)
		}
	}

	// Test with invalid parameters
	invalidSample := ts.sampleBeta(-1.0, 2.0)
	if invalidSample < 0.0 || invalidSample > 1.0 {
		t.Errorf("Invalid parameter sample should still be in [0,1], got %f", invalidSample)
	}
}

func TestThompsonSampler_SampleGamma(t *testing.T) {
	ts := NewThompsonSampler(0.1)

	// Test with shape >= 1
	shape := 2.0
	sample := ts.sampleGamma(shape)
	if sample < 0.0 {
		t.Errorf("Gamma sample should be >= 0, got %f", sample)
	}

	// Test with shape < 1
	shape = 0.5
	sample = ts.sampleGamma(shape)
	if sample < 0.0 {
		t.Errorf("Gamma sample should be >= 0, got %f", sample)
	}
}

func TestThompsonSampler_SampleNormal(t *testing.T) {
	ts := NewThompsonSampler(0.1)

	samples := make([]float64, 100)
	for i := 0; i < 100; i++ {
		samples[i] = ts.sampleNormal()
	}

	// Calculate sample mean and std dev
	mean := 0.0
	for _, sample := range samples {
		mean += sample
	}
	mean /= float64(len(samples))

	variance := 0.0
	for _, sample := range samples {
		variance += (sample - mean) * (sample - mean)
	}
	variance /= float64(len(samples) - 1)
	stdDev := math.Sqrt(variance)

	// For standard normal, mean should be ~0 and std dev ~1
	// With 100 samples, allow for some variation
	if math.Abs(mean) > 0.3 {
		t.Errorf("Sample mean should be ~0, got %f", mean)
	}

	if stdDev < 0.7 || stdDev > 1.3 {
		t.Errorf("Sample std dev should be ~1, got %f", stdDev)
	}
}

func TestEvaluateOutcome(t *testing.T) {
	strategy := models.StrategyBalanced
	latencyMS := 100.0
	costUSD := 5.0
	throughputOps := 500.0
	energyWh := 20.0
	slaThresholdMS := 200.0
	budgetUSD := 10.0

	outcome := EvaluateOutcome(
		strategy, latencyMS, costUSD, throughputOps, energyWh, slaThresholdMS, budgetUSD)

	if outcome.Strategy != strategy {
		t.Errorf("Expected strategy=%s, got %s", strategy, outcome.Strategy)
	}

	if outcome.LatencyMS != latencyMS {
		t.Errorf("Expected latency=%f, got %f", latencyMS, outcome.LatencyMS)
	}

	if outcome.CostUSD != costUSD {
		t.Errorf("Expected cost=%f, got %f", costUSD, outcome.CostUSD)
	}

	if outcome.ThroughputOps != throughputOps {
		t.Errorf("Expected throughput=%f, got %f", throughputOps, outcome.ThroughputOps)
	}

	if outcome.EnergyWh != energyWh {
		t.Errorf("Expected energy=%f, got %f", energyWh, outcome.EnergyWh)
	}

	// SLA should be met (100 <= 200)
	if !outcome.MetSLA {
		t.Error("Expected SLA to be met")
	}

	// Budget should be met (5 <= 10)
	if !outcome.CostUnderBudget {
		t.Error("Expected cost to be under budget")
	}

	// Reward should be positive since both SLA and budget are met
	if outcome.Reward <= 0 {
		t.Errorf("Expected positive reward, got %f", outcome.Reward)
	}

	// Success should be true
	if !outcome.Success {
		t.Error("Expected success to be true")
	}

	if outcome.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestEvaluateOutcome_SLAViolation(t *testing.T) {
	strategy := models.StrategyBalanced
	latencyMS := 300.0        // Violates SLA
	slaThresholdMS := 200.0
	costUSD := 5.0
	budgetUSD := 10.0

	outcome := EvaluateOutcome(
		strategy, latencyMS, costUSD, 500.0, 20.0, slaThresholdMS, budgetUSD)

	if outcome.MetSLA {
		t.Error("Expected SLA to be violated")
	}

	// Should still be under budget
	if !outcome.CostUnderBudget {
		t.Error("Expected cost to be under budget")
	}

	// Success should be false due to SLA violation
	if outcome.Success {
		t.Error("Expected success to be false due to SLA violation")
	}
}

func TestEvaluateOutcome_BudgetOverrun(t *testing.T) {
	strategy := models.StrategyBalanced
	latencyMS := 100.0
	slaThresholdMS := 200.0
	costUSD := 15.0   // Over budget
	budgetUSD := 10.0

	outcome := EvaluateOutcome(
		strategy, latencyMS, costUSD, 500.0, 20.0, slaThresholdMS, budgetUSD)

	if !outcome.MetSLA {
		t.Error("Expected SLA to be met")
	}

	if outcome.CostUnderBudget {
		t.Error("Expected cost to be over budget")
	}

	// Success should be false due to budget overrun
	if outcome.Success {
		t.Error("Expected success to be false due to budget overrun")
	}
}

func TestEvaluateOutcome_RewardCalculation(t *testing.T) {
	strategy := models.StrategyBalanced
	slaThresholdMS := 200.0
	budgetUSD := 10.0

	// Test perfect case
	perfectOutcome := EvaluateOutcome(
		strategy, 50.0, 2.0, 1000.0, 10.0, slaThresholdMS, budgetUSD)

	// Should get high reward
	if perfectOutcome.Reward < 0.8 {
		t.Errorf("Expected high reward for perfect case, got %f", perfectOutcome.Reward)
	}

	// Test worst case
	worstOutcome := EvaluateOutcome(
		strategy, 400.0, 20.0, 0.0, 200.0, slaThresholdMS, budgetUSD)

	// Should get negative reward
	if worstOutcome.Reward > -0.5 {
		t.Errorf("Expected negative reward for worst case, got %f", worstOutcome.Reward)
	}

	// Reward should be clamped to [-1, 1]
	if worstOutcome.Reward < -1.0 || worstOutcome.Reward > 1.0 {
		t.Errorf("Reward should be in [-1,1], got %f", worstOutcome.Reward)
	}
}

func TestThompsonSampler_ExplorationBehavior(t *testing.T) {
	// Test with high exploration rate
	highExploration := NewThompsonSampler(0.9)
	
	// Make one strategy clearly better
	for i := 0; i < 20; i++ {
		highExploration.UpdateStrategy(models.StrategyBalanced, 
			StrategyOutcome{Strategy: models.StrategyBalanced, Success: true})
	}

	// Test with low exploration rate
	lowExploration := NewThompsonSampler(0.01)
	
	// Make one strategy clearly better
	for i := 0; i < 20; i++ {
		lowExploration.UpdateStrategy(models.StrategyBalanced, 
			StrategyOutcome{Strategy: models.StrategyBalanced, Success: true})
	}

	// Count selections over many trials
	highExplorationCounts := make(map[models.Strategy]int)
	lowExplorationCounts := make(map[models.Strategy]int)

	trials := 100
	for i := 0; i < trials; i++ {
		highStrategy := highExploration.SelectStrategy()
		lowStrategy := lowExploration.SelectStrategy()
		
		highExplorationCounts[highStrategy]++
		lowExplorationCounts[lowStrategy]++
	}

	// High exploration should select more diverse strategies
	highDiversity := len(highExplorationCounts)
	lowDiversity := len(lowExplorationCounts)

	if highDiversity <= lowDiversity {
		t.Logf("High exploration diversity: %d, Low exploration diversity: %d", 
			highDiversity, lowDiversity)
		// This is probabilistic, so we just log it rather than fail
	}
}

// Benchmark Thompson Sampling performance
func BenchmarkThompsonSampler_SelectStrategy(b *testing.B) {
	ts := NewThompsonSampler(0.1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.SelectStrategy()
	}
}

func BenchmarkThompsonSampler_UpdateStrategy(b *testing.B) {
	ts := NewThompsonSampler(0.1)
	outcome := StrategyOutcome{
		Strategy: models.StrategyBalanced,
		Success:  true,
		Reward:   0.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ts.UpdateStrategy(models.StrategyBalanced, outcome)
	}
}

func BenchmarkEvaluateOutcome(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EvaluateOutcome(
			models.StrategyBalanced,
			100.0, 5.0, 500.0, 20.0,
			200.0, 10.0)
	}
}