package learning

import (
	"math"
	"math/rand"
	"time"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// ThompsonSampler implements Thompson Sampling (1933) for strategy selection
// Uses Beta distribution to model success/failure rates for each strategy
type ThompsonSampler struct {
	strategies      []models.Strategy
	successes       map[models.Strategy]int     // Alpha parameter for Beta distribution
	failures        map[models.Strategy]int     // Beta parameter for Beta distribution
	totalTrials     map[models.Strategy]int     // Total attempts per strategy
	random          *rand.Rand
	explorationRate float64                     // Minimum exploration probability
	lastSelected    models.Strategy
	lastOutcome     *StrategyOutcome
}

// StrategyOutcome represents the result of applying a strategy
type StrategyOutcome struct {
	Strategy       models.Strategy `json:"strategy"`
	MetSLA         bool            `json:"met_sla"`
	CostUnderBudget bool           `json:"cost_under_budget"`
	LatencyMS      float64         `json:"latency_ms"`
	CostUSD        float64         `json:"cost_usd"`
	ThroughputOps  float64         `json:"throughput_ops"`
	EnergyWh       float64         `json:"energy_wh"`
	Success        bool            `json:"success"`        // Overall success metric
	Reward         float64         `json:"reward"`         // Reward signal [-1.0, 1.0]
	Timestamp      time.Time       `json:"timestamp"`
}

// NewThompsonSampler creates a new Thompson sampler with default strategies
func NewThompsonSampler(explorationRate float64) *ThompsonSampler {
	strategies := []models.Strategy{
		models.StrategyDataLocal,
		models.StrategyPerformance,
		models.StrategyCostOptimal,
		models.StrategyBalanced,
		models.StrategyLatencyFirst,
		models.StrategyGreenCompute,
	}
	
	return NewThompsonSamplerWithStrategies(strategies, explorationRate)
}

// NewThompsonSamplerWithStrategies creates a Thompson sampler with custom strategies
func NewThompsonSamplerWithStrategies(strategies []models.Strategy, explorationRate float64) *ThompsonSampler {
	ts := &ThompsonSampler{
		strategies:      make([]models.Strategy, len(strategies)),
		successes:       make(map[models.Strategy]int),
		failures:        make(map[models.Strategy]int),
		totalTrials:     make(map[models.Strategy]int),
		random:          rand.New(rand.NewSource(time.Now().UnixNano())),
		explorationRate: explorationRate,
	}
	
	copy(ts.strategies, strategies)
	
	// Initialize with uniform priors (1 success, 1 failure each)
	for _, strategy := range strategies {
		ts.successes[strategy] = 1
		ts.failures[strategy] = 1
		ts.totalTrials[strategy] = 0
	}
	
	return ts
}

// SelectStrategy selects a strategy using Thompson Sampling
func (ts *ThompsonSampler) SelectStrategy() models.Strategy {
	if len(ts.strategies) == 0 {
		return models.StrategyBalanced // Default fallback
	}
	
	// With small probability, do pure exploration
	if ts.random.Float64() < ts.explorationRate {
		selected := ts.strategies[ts.random.Intn(len(ts.strategies))]
		ts.lastSelected = selected
		return selected
	}
	
	// Thompson Sampling: sample from Beta distribution for each strategy
	bestStrategy := ts.strategies[0]
	bestSample := -1.0
	
	for _, strategy := range ts.strategies {
		alpha := float64(ts.successes[strategy])
		beta := float64(ts.failures[strategy])
		
		// Sample from Beta(alpha, beta) distribution
		sample := ts.sampleBeta(alpha, beta)
		
		if sample > bestSample {
			bestSample = sample
			bestStrategy = strategy
		}
	}
	
	ts.lastSelected = bestStrategy
	return bestStrategy
}

// UpdateStrategy updates the success/failure counts based on outcome
func (ts *ThompsonSampler) UpdateStrategy(strategy models.Strategy, outcome StrategyOutcome) {
	ts.totalTrials[strategy]++
	ts.lastOutcome = &outcome
	
	if outcome.Success {
		ts.successes[strategy]++
	} else {
		ts.failures[strategy]++
	}
}

// UpdateLastStrategy updates the last selected strategy with an outcome
func (ts *ThompsonSampler) UpdateLastStrategy(outcome StrategyOutcome) {
	if ts.lastSelected != "" {
		ts.UpdateStrategy(ts.lastSelected, outcome)
	}
}

// GetStrategyStats returns current statistics for all strategies
func (ts *ThompsonSampler) GetStrategyStats() map[models.Strategy]StrategyStats {
	stats := make(map[models.Strategy]StrategyStats)
	
	for _, strategy := range ts.strategies {
		successes := ts.successes[strategy]
		failures := ts.failures[strategy]
		trials := ts.totalTrials[strategy]
		
		successRate := 0.0
		if trials > 0 {
			successRate = float64(successes-1) / float64(trials) // Subtract prior
		}
		
		// Calculate confidence interval using Beta distribution
		alpha := float64(successes)
		beta := float64(failures)
		mean := alpha / (alpha + beta)
		
		// 95% confidence interval approximation
		variance := (alpha * beta) / ((alpha + beta) * (alpha + beta) * (alpha + beta + 1))
		stdDev := math.Sqrt(variance)
		confidence := 1.96 * stdDev
		
		stats[strategy] = StrategyStats{
			Strategy:           strategy,
			Successes:          successes - 1, // Subtract prior
			Failures:           failures - 1,  // Subtract prior
			TotalTrials:        trials,
			SuccessRate:        successRate,
			EstimatedMean:      mean,
			ConfidenceLower:    math.Max(0.0, mean-confidence),
			ConfidenceUpper:    math.Min(1.0, mean+confidence),
			LastUpdated:        time.Now(),
		}
	}
	
	return stats
}

// StrategyStats represents performance statistics for a strategy
type StrategyStats struct {
	Strategy        models.Strategy `json:"strategy"`
	Successes       int             `json:"successes"`
	Failures        int             `json:"failures"`
	TotalTrials     int             `json:"total_trials"`
	SuccessRate     float64         `json:"success_rate"`
	EstimatedMean   float64         `json:"estimated_mean"`
	ConfidenceLower float64         `json:"confidence_lower"`
	ConfidenceUpper float64         `json:"confidence_upper"`
	LastUpdated     time.Time       `json:"last_updated"`
}

// GetBestStrategy returns the strategy with highest estimated success rate
func (ts *ThompsonSampler) GetBestStrategy() models.Strategy {
	bestStrategy := models.StrategyBalanced
	bestRate := -1.0
	
	stats := ts.GetStrategyStats()
	for strategy, stat := range stats {
		if stat.SuccessRate > bestRate {
			bestRate = stat.SuccessRate
			bestStrategy = strategy
		}
	}
	
	return bestStrategy
}

// GetConvergenceMetrics returns metrics about convergence and exploration
func (ts *ThompsonSampler) GetConvergenceMetrics() ConvergenceMetrics {
	stats := ts.GetStrategyStats()
	
	totalTrials := 0
	maxTrials := 0
	minTrials := math.MaxInt32
	strategyCount := len(stats)
	
	for _, stat := range stats {
		totalTrials += stat.TotalTrials
		if stat.TotalTrials > maxTrials {
			maxTrials = stat.TotalTrials
		}
		if stat.TotalTrials < minTrials {
			minTrials = stat.TotalTrials
		}
	}
	
	explorationRatio := 0.0
	if maxTrials > 0 {
		explorationRatio = float64(minTrials) / float64(maxTrials)
	}
	
	// Calculate strategy entropy (measure of exploration diversity)
	entropy := 0.0
	if totalTrials > 0 {
		for _, stat := range stats {
			if stat.TotalTrials > 0 {
				prob := float64(stat.TotalTrials) / float64(totalTrials)
				entropy -= prob * math.Log2(prob)
			}
		}
	}
	
	// Normalized entropy (0 = all exploration on one strategy, 1 = uniform exploration)
	maxEntropy := math.Log2(float64(strategyCount))
	normalizedEntropy := 0.0
	if maxEntropy > 0 {
		normalizedEntropy = entropy / maxEntropy
	}
	
	return ConvergenceMetrics{
		TotalTrials:         totalTrials,
		StrategyCount:       strategyCount,
		ExplorationRatio:    explorationRatio,
		EntropyScore:        normalizedEntropy,
		ConvergenceScore:    1.0 - normalizedEntropy, // Higher = more converged
		LastUpdated:         time.Now(),
	}
}

// ConvergenceMetrics provides insights into learning progress
type ConvergenceMetrics struct {
	TotalTrials      int       `json:"total_trials"`
	StrategyCount    int       `json:"strategy_count"`
	ExplorationRatio float64   `json:"exploration_ratio"`    // Min trials / Max trials
	EntropyScore     float64   `json:"entropy_score"`        // Exploration diversity [0,1]
	ConvergenceScore float64   `json:"convergence_score"`    // How converged [0,1]
	LastUpdated      time.Time `json:"last_updated"`
}

// Reset resets the Thompson sampler to initial state
func (ts *ThompsonSampler) Reset() {
	for _, strategy := range ts.strategies {
		ts.successes[strategy] = 1  // Uniform prior
		ts.failures[strategy] = 1   // Uniform prior
		ts.totalTrials[strategy] = 0
	}
	ts.lastSelected = ""
	ts.lastOutcome = nil
}

// sampleBeta samples from Beta(alpha, beta) distribution using method of moments
func (ts *ThompsonSampler) sampleBeta(alpha, beta float64) float64 {
	if alpha <= 0 || beta <= 0 {
		return ts.random.Float64()
	}
	
	// For efficiency, we'll use a simple approximation for Beta sampling
	// In production, you might want to use a more sophisticated algorithm
	
	// Generate two gamma-distributed variables and use the ratio
	x := ts.sampleGamma(alpha)
	y := ts.sampleGamma(beta)
	
	if x+y == 0 {
		return 0.5 // Fallback
	}
	
	return x / (x + y)
}

// sampleGamma samples from Gamma distribution using simple method
func (ts *ThompsonSampler) sampleGamma(shape float64) float64 {
	if shape < 1.0 {
		// Use transformation for shape < 1
		return ts.sampleGamma(shape+1.0) * math.Pow(ts.random.Float64(), 1.0/shape)
	}
	
	// Marsaglia and Tsang's method for shape >= 1
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	
	for {
		// Generate normal random variable
		x := ts.sampleNormal()
		v := 1.0 + c*x
		
		if v <= 0 {
			continue
		}
		
		v = v * v * v
		u := ts.random.Float64()
		
		if u < 1.0-0.0331*(x*x)*(x*x) {
			return d * v
		}
		
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}

// sampleNormal samples from standard normal distribution using Box-Muller transform
func (ts *ThompsonSampler) sampleNormal() float64 {
	u1 := ts.random.Float64()
	u2 := ts.random.Float64()
	
	// Box-Muller transform
	z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
	return z
}

// EvaluateOutcome creates a StrategyOutcome based on decision results
func EvaluateOutcome(
	strategy models.Strategy,
	latencyMS float64,
	costUSD float64,
	throughputOps float64,
	energyWh float64,
	slaThresholdMS float64,
	budgetUSD float64,
) StrategyOutcome {
	
	metSLA := latencyMS <= slaThresholdMS
	costUnderBudget := costUSD <= budgetUSD
	
	// Calculate reward based on multiple factors
	reward := 0.0
	
	// SLA compliance (40% of reward)
	if metSLA {
		reward += 0.4
	} else {
		// Penalize SLA violations
		slaViolationRatio := (latencyMS - slaThresholdMS) / slaThresholdMS
		reward -= 0.4 * math.Min(slaViolationRatio, 1.0)
	}
	
	// Budget compliance (30% of reward)
	if costUnderBudget {
		// Reward for staying under budget, more reward for bigger savings
		savingsRatio := (budgetUSD - costUSD) / budgetUSD
		reward += 0.3 * (0.5 + 0.5*savingsRatio)
	} else {
		// Penalize budget overruns
		overrunRatio := (costUSD - budgetUSD) / budgetUSD
		reward -= 0.3 * math.Min(overrunRatio, 1.0)
	}
	
	// Throughput performance (20% of reward)
	// Normalize throughput to [0,1] range (assuming max ~1000 ops/sec)
	normalizedThroughput := math.Min(throughputOps/1000.0, 1.0)
	reward += 0.2 * normalizedThroughput
	
	// Energy efficiency (10% of reward)  
	// Lower energy consumption is better (assuming max ~100 Wh)
	energyEfficiency := math.Max(0.0, 1.0-(energyWh/100.0))
	reward += 0.1 * energyEfficiency
	
	// Clamp reward to [-1.0, 1.0]
	reward = math.Max(-1.0, math.Min(1.0, reward))
	
	// Overall success is based on meeting key constraints
	success := metSLA && costUnderBudget && reward > 0.0
	
	return StrategyOutcome{
		Strategy:        strategy,
		MetSLA:          metSLA,
		CostUnderBudget: costUnderBudget,
		LatencyMS:       latencyMS,
		CostUSD:         costUSD,
		ThroughputOps:   throughputOps,
		EnergyWh:        energyWh,
		Success:         success,
		Reward:          reward,
		Timestamp:       time.Now(),
	}
}