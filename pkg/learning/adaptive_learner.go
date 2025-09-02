package learning

import (
	"math"
	"sort"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// AdaptiveLearner implements the adaptive learning component of the algorithm
type AdaptiveLearner struct {
	config            LearningConfig
	weightAdapter     *WeightAdapter
	patternRecognizer *PatternRecognizer
	outcomeWindow     *OutcomeWindow
	baseline          *PerformanceBaseline
	progress          *LearningProgress
	objectives        []LearningObjective
}

// NewAdaptiveLearner creates a new adaptive learner
func NewAdaptiveLearner(config LearningConfig) *AdaptiveLearner {
	return &AdaptiveLearner{
		config: config,
		weightAdapter: &WeightAdapter{
			learningRate:    config.LearningRate,
			explorationRate: config.ExplorationRate,
			weightHistory:   make([]decision.AdaptiveWeights, 0),
		},
		patternRecognizer: &PatternRecognizer{
			patterns:       make([]*decision.DiscoveredPattern, 0),
			outcomeHistory: make([]decision.OffloadOutcome, 0),
			minSamples:     config.MinSamples,
			maxPatterns:    50,
		},
		outcomeWindow: &OutcomeWindow{
			outcomes: make([]decision.OffloadOutcome, 0),
			maxSize:  config.WindowSize,
		},
		baseline: &PerformanceBaseline{
			StaticWeights: decision.AdaptiveWeights{
				QueueDepth:    0.2,
				ProcessorLoad: 0.2,
				NetworkCost:   0.2,
				LatencyCost:   0.2,
				EnergyCost:    0.1,
				PolicyCost:    0.1,
			},
		},
		progress: &LearningProgress{
			WeightHistory: make([]decision.AdaptiveWeights, 0),
		},
		objectives: initializeLearningObjectives(),
	}
}

// initializeLearningObjectives creates the default learning objectives
func initializeLearningObjectives() []LearningObjective {
	return []LearningObjective{
		{Name: "QueueReduction", Weight: 0.20, MinWeight: 0.05, MaxWeight: 0.50, TargetValue: 0.95, Trend: UNKNOWN},
		{Name: "LoadBalancing", Weight: 0.20, MinWeight: 0.05, MaxWeight: 0.50, TargetValue: 0.85, Trend: UNKNOWN},
		{Name: "NetworkOptimization", Weight: 0.20, MinWeight: 0.05, MaxWeight: 0.40, TargetValue: 0.90, Trend: UNKNOWN},
		{Name: "LatencyMinimization", Weight: 0.20, MinWeight: 0.10, MaxWeight: 0.40, TargetValue: 0.95, Trend: UNKNOWN},
		{Name: "EnergyEfficiency", Weight: 0.10, MinWeight: 0.00, MaxWeight: 0.30, TargetValue: 0.80, Trend: UNKNOWN},
		{Name: "PolicyCompliance", Weight: 0.10, MinWeight: 0.05, MaxWeight: 0.25, TargetValue: 1.00, Trend: UNKNOWN},
	}
}

// UpdateWeights updates the weights based on an outcome
func (al *AdaptiveLearner) UpdateWeights(weights *decision.AdaptiveWeights, outcome decision.OffloadOutcome) {
	// Add outcome to window
	al.outcomeWindow.Add(outcome)
	al.progress.DecisionCount++
	
	// Calculate weight adjustments based on attribution and reward
	adjustments := al.calculateWeightAdjustments(weights, outcome)
	
	// Apply adjustments with learning rate
	al.applyWeightAdjustments(weights, adjustments)
	
	// Ensure weights are normalized
	weights.Normalize()
	
	// Track weight history
	al.weightAdapter.weightHistory = append(al.weightAdapter.weightHistory, *weights)
	al.progress.WeightHistory = append(al.progress.WeightHistory, *weights)
	al.progress.WeightUpdates++
	
	// Check for convergence
	al.checkConvergence()
}

// calculateWeightAdjustments calculates how weights should be adjusted
func (al *AdaptiveLearner) calculateWeightAdjustments(
	weights *decision.AdaptiveWeights,
	outcome decision.OffloadOutcome,
) map[string]float64 {
	adjustments := make(map[string]float64)
	
	// Base adjustment on reward and attribution
	learningRate := al.config.LearningRate
	
	// If outcome has attribution, use it to guide weight updates
	if outcome.Attribution != nil && len(outcome.Attribution) > 0 {
		// Positive reward: increase weights for highly attributed factors
		// Negative reward: decrease weights for highly attributed factors
		for factor, attribution := range outcome.Attribution {
			adjustment := learningRate * outcome.Reward * attribution
			adjustments[factor] = adjustment
		}
	} else {
		// No attribution, distribute adjustment evenly
		evenAdjustment := learningRate * outcome.Reward / 6.0
		adjustments["QueueDepth"] = evenAdjustment
		adjustments["ProcessorLoad"] = evenAdjustment
		adjustments["NetworkCost"] = evenAdjustment
		adjustments["LatencyCost"] = evenAdjustment
		adjustments["EnergyCost"] = evenAdjustment
		adjustments["PolicyCost"] = evenAdjustment
	}
	
	// Add exploration noise
	if al.config.ExplorationRate > 0 {
		for factor := range adjustments {
			noise := (math.Sin(float64(al.progress.DecisionCount)) * 0.5 + 0.5) * al.config.ExplorationRate * 0.01
			adjustments[factor] += noise - al.config.ExplorationRate*0.005
		}
	}
	
	return adjustments
}

// applyWeightAdjustments applies calculated adjustments to weights
func (al *AdaptiveLearner) applyWeightAdjustments(
	weights *decision.AdaptiveWeights,
	adjustments map[string]float64,
) {
	// Apply adjustments
	if adj, ok := adjustments["QueueDepth"]; ok {
		weights.QueueDepth = math.Max(0.0, math.Min(1.0, weights.QueueDepth+adj))
	}
	if adj, ok := adjustments["ProcessorLoad"]; ok {
		weights.ProcessorLoad = math.Max(0.0, math.Min(1.0, weights.ProcessorLoad+adj))
	}
	if adj, ok := adjustments["NetworkCost"]; ok {
		weights.NetworkCost = math.Max(0.0, math.Min(1.0, weights.NetworkCost+adj))
	}
	if adj, ok := adjustments["LatencyCost"]; ok {
		weights.LatencyCost = math.Max(0.0, math.Min(1.0, weights.LatencyCost+adj))
	}
	if adj, ok := adjustments["EnergyCost"]; ok {
		weights.EnergyCost = math.Max(0.0, math.Min(1.0, weights.EnergyCost+adj))
	}
	if adj, ok := adjustments["PolicyCost"]; ok {
		weights.PolicyCost = math.Max(0.0, math.Min(1.0, weights.PolicyCost+adj))
	}
}

// checkConvergence checks if weights have converged
func (al *AdaptiveLearner) checkConvergence() {
	history := al.weightAdapter.weightHistory
	if len(history) < 20 {
		return
	}
	
	// Check if recent weights are stable
	recent := history[len(history)-20:]
	variance := al.calculateWeightVariance(recent)
	
	if variance < 0.01 { // Low variance indicates convergence
		if !al.progress.IsConverged {
			al.progress.IsConverged = true
			al.progress.ConvergenceTime = al.progress.DecisionCount
			al.weightAdapter.convergenceTime = al.progress.DecisionCount
		}
	}
}

// calculateWeightVariance calculates variance in weight history
func (al *AdaptiveLearner) calculateWeightVariance(weights []decision.AdaptiveWeights) float64 {
	if len(weights) == 0 {
		return 1.0
	}
	
	// Calculate mean weights
	mean := decision.AdaptiveWeights{}
	for _, w := range weights {
		mean.QueueDepth += w.QueueDepth
		mean.ProcessorLoad += w.ProcessorLoad
		mean.NetworkCost += w.NetworkCost
		mean.LatencyCost += w.LatencyCost
		mean.EnergyCost += w.EnergyCost
		mean.PolicyCost += w.PolicyCost
	}
	
	n := float64(len(weights))
	mean.QueueDepth /= n
	mean.ProcessorLoad /= n
	mean.NetworkCost /= n
	mean.LatencyCost /= n
	mean.EnergyCost /= n
	mean.PolicyCost /= n
	
	// Calculate variance
	variance := 0.0
	for _, w := range weights {
		variance += math.Pow(w.QueueDepth-mean.QueueDepth, 2)
		variance += math.Pow(w.ProcessorLoad-mean.ProcessorLoad, 2)
		variance += math.Pow(w.NetworkCost-mean.NetworkCost, 2)
		variance += math.Pow(w.LatencyCost-mean.LatencyCost, 2)
		variance += math.Pow(w.EnergyCost-mean.EnergyCost, 2)
		variance += math.Pow(w.PolicyCost-mean.PolicyCost, 2)
	}
	
	return variance / (n * 6.0) // 6 weight components
}

// DiscoverPatterns discovers patterns from recent outcomes
func (al *AdaptiveLearner) DiscoverPatterns(
	state models.SystemState,
	process models.Process,
	outcome decision.OffloadOutcome,
) []*decision.DiscoveredPattern {
	al.patternRecognizer.outcomeHistory = append(al.patternRecognizer.outcomeHistory, outcome)
	
	// Need minimum samples before pattern discovery
	if len(al.patternRecognizer.outcomeHistory) < al.config.MinSamples {
		return al.patternRecognizer.patterns
	}
	
	// Look for patterns in successful outcomes
	newPatterns := al.findPatternsInOutcomes()
	
	// Add new patterns to the collection
	for _, pattern := range newPatterns {
		al.addPattern(pattern)
	}
	
	// Validate existing patterns
	al.validatePatterns()
	
	// Prune old or ineffective patterns
	al.prunePatterns()
	
	al.progress.PatternsDiscovered = len(al.patternRecognizer.patterns)
	
	// Count validated patterns
	validated := 0
	for _, p := range al.patternRecognizer.patterns {
		if p.ValidationStatus == decision.VALIDATED {
			validated++
		}
	}
	al.progress.PatternsValidated = validated
	
	return al.patternRecognizer.patterns
}

// findPatternsInOutcomes looks for patterns in recent outcomes
func (al *AdaptiveLearner) findPatternsInOutcomes() []*decision.DiscoveredPattern {
	patterns := make([]*decision.DiscoveredPattern, 0)
	
	// Group outcomes by success/failure
	successful := make([]decision.OffloadOutcome, 0)
	failed := make([]decision.OffloadOutcome, 0)
	
	for _, outcome := range al.patternRecognizer.outcomeHistory {
		if outcome.Success {
			successful = append(successful, outcome)
		} else {
			failed = append(failed, outcome)
		}
	}
	
	// Look for patterns in successful outcomes
	if len(successful) >= al.config.MinSamples {
		// Example pattern: High queue depth -> offload beneficial
		highQueuePattern := &decision.DiscoveredPattern{
			ID:          "high_queue_offload",
			Name:        "High Queue Offload",
			Description: "Offloading is beneficial when queue depth is high",
			Conditions: []decision.PatternCondition{
				{
					Field:    "QueueDepth",
					Operator: models.GREATER_THAN,
					Value:    20,
					Weight:   1.0,
				},
			},
			Confidence:        float64(len(successful)) / float64(len(al.patternRecognizer.outcomeHistory)),
			RecommendedAction: models.OFFLOAD_TO,
			ApplicationCount:  0,
			SuccessRate:       float64(len(successful)) / float64(len(successful)+len(failed)),
			CreatedTime:       time.Now(),
			LastUpdated:       time.Now(),
			MinSamples:        al.config.MinSamples,
			ValidationStatus:  decision.DISCOVERING,
		}
		
		if highQueuePattern.SuccessRate > 0.7 {
			patterns = append(patterns, highQueuePattern)
		}
	}
	
	// Look for patterns in failed outcomes
	if len(failed) >= al.config.MinSamples/2 {
		// Example pattern: Network congestion -> keep local
		networkCongestionPattern := &decision.DiscoveredPattern{
			ID:          "network_congestion_local",
			Name:        "Network Congestion Keep Local",
			Description: "Keep processing local when network is congested",
			Conditions: []decision.PatternCondition{
				{
					Field:    "NetworkUsage",
					Operator: models.GREATER_THAN,
					Value:    0.8,
					Weight:   1.0,
				},
			},
			Confidence:        0.8,
			RecommendedAction: models.KEEP_LOCAL,
			ApplicationCount:  0,
			SuccessRate:       0.9,
			CreatedTime:       time.Now(),
			LastUpdated:       time.Now(),
			MinSamples:        al.config.MinSamples,
			ValidationStatus:  decision.DISCOVERING,
		}
		patterns = append(patterns, networkCongestionPattern)
	}
	
	return patterns
}

// addPattern adds a new pattern to the collection
func (al *AdaptiveLearner) addPattern(pattern *decision.DiscoveredPattern) {
	// Check if pattern already exists
	for _, existing := range al.patternRecognizer.patterns {
		if existing.ID == pattern.ID {
			// Update existing pattern
			existing.ApplicationCount++
			existing.LastUpdated = time.Now()
			return
		}
	}
	
	// Add new pattern
	al.patternRecognizer.patterns = append(al.patternRecognizer.patterns, pattern)
}

// validatePatterns validates patterns based on recent performance
func (al *AdaptiveLearner) validatePatterns() {
	for _, pattern := range al.patternRecognizer.patterns {
		if pattern.ApplicationCount >= al.config.MinSamples && pattern.SuccessRate > 0.8 {
			pattern.ValidationStatus = decision.VALIDATED
		} else if pattern.SuccessRate < 0.5 && pattern.ApplicationCount > 5 {
			pattern.ValidationStatus = decision.DEPRECATED
		}
	}
}

// prunePatterns removes old or ineffective patterns
func (al *AdaptiveLearner) prunePatterns() {
	if len(al.patternRecognizer.patterns) <= al.patternRecognizer.maxPatterns {
		return
	}
	
	// Sort by success rate and last used time
	sort.Slice(al.patternRecognizer.patterns, func(i, j int) bool {
		pi := al.patternRecognizer.patterns[i]
		pj := al.patternRecognizer.patterns[j]
		
		// Prioritize validated patterns
		if pi.ValidationStatus != pj.ValidationStatus {
			return pi.ValidationStatus == decision.VALIDATED
		}
		
		// Then by success rate
		if math.Abs(pi.SuccessRate-pj.SuccessRate) > 0.1 {
			return pi.SuccessRate > pj.SuccessRate
		}
		
		// Then by recency
		return pi.LastUsed.After(pj.LastUsed)
	})
	
	// Keep only top patterns
	al.patternRecognizer.patterns = al.patternRecognizer.patterns[:al.patternRecognizer.maxPatterns]
}

// GetPerformanceImprovement calculates performance improvement over baseline
func (al *AdaptiveLearner) GetPerformanceImprovement() float64 {
	if al.outcomeWindow.totalCount < al.config.MinSamples {
		return 0.0
	}
	
	// Calculate current performance
	currentPerf := al.outcomeWindow.GetAverageReward()
	
	// Estimate baseline performance (simplified)
	baselinePerf := 0.5 // Assume baseline average reward is 0.5
	
	if baselinePerf == 0 {
		return 0.0
	}
	
	improvement := (currentPerf - baselinePerf) / math.Abs(baselinePerf)
	al.progress.PerformanceGain = improvement
	
	return improvement
}

// GetProgress returns the current learning progress
func (al *AdaptiveLearner) GetProgress() *LearningProgress {
	return al.progress
}

// GetPatterns returns discovered patterns
func (al *AdaptiveLearner) GetPatterns() []*decision.DiscoveredPattern {
	return al.patternRecognizer.patterns
}

// IsConverged returns whether the weights have converged
func (al *AdaptiveLearner) IsConverged() bool {
	return al.progress.IsConverged
}

// GetConvergenceTime returns the number of decisions until convergence
func (al *AdaptiveLearner) GetConvergenceTime() int {
	return al.weightAdapter.convergenceTime
}