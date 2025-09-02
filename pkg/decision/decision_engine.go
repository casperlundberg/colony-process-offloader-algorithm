package decision

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// DecisionEngine makes offloading decisions based on system state and process requirements
type DecisionEngine struct {
	weights          AdaptiveWeights
	patterns         []*DiscoveredPattern
	safetyMargins    SafetyMargins
	algorithmVersion string
}

// SafetyMargins defines safety constraints for decision making
type SafetyMargins struct {
	MinLocalCompute       float64
	MinLocalMemory        float64
	MaxConcurrentOffloads int
	MaxLatencyTolerance   time.Duration
	MinReliability        float64
}

// NewDecisionEngine creates a new decision engine with the given weights
func NewDecisionEngine(weights AdaptiveWeights) *DecisionEngine {
	weights.Normalize()
	return &DecisionEngine{
		weights:          weights,
		patterns:         make([]*DiscoveredPattern, 0),
		algorithmVersion: "1.0.0",
		safetyMargins: SafetyMargins{
			MinLocalCompute:       0.2,  // Keep 20% compute local
			MinLocalMemory:        0.2,  // Keep 20% memory local
			MaxConcurrentOffloads: 10,
			MaxLatencyTolerance:   500 * time.Millisecond,
			MinReliability:        0.5,
		},
	}
}

// MakeDecision makes an offloading decision for a process
func (de *DecisionEngine) MakeDecision(
	process models.Process,
	targets []models.OffloadTarget,
	state models.SystemState,
) (OffloadDecision, error) {
	startTime := time.Now()

	// Validate inputs
	if err := process.Validate(); err != nil {
		return OffloadDecision{}, fmt.Errorf("invalid process: %w", err)
	}
	if err := state.Validate(); err != nil {
		return OffloadDecision{}, fmt.Errorf("invalid system state: %w", err)
	}

	// Step 1: Check if we should consider offloading
	shouldOffload, reason := de.shouldConsiderOffloading(state)
	if !shouldOffload {
		return de.createLocalDecision(process, reason, startTime), nil
	}

	// Step 2: Filter targets by safety and policy constraints
	viableTargets := de.filterTargets(process, targets, state)
	if len(viableTargets) == 0 {
		return de.createLocalDecision(process, "no viable targets", startTime), nil
	}

	// Step 3: Check for applicable patterns
	pattern := de.findBestPattern(process, state)

	// Step 4: Score each target
	scores := de.scoreTargets(process, viableTargets, state, pattern)

	// Step 5: Select best target
	bestTarget, bestScore := de.selectBestTarget(scores, viableTargets)
	if bestTarget == nil || bestScore < 0.3 { // Minimum score threshold
		return de.createLocalDecision(process, "scores below threshold", startTime), nil
	}

	// Step 6: Create offload decision
	decision := de.createOffloadDecision(process, bestTarget, bestScore, pattern, startTime)
	
	// Ensure decision latency is within requirement
	if decision.DecisionLatency > 500*time.Millisecond {
		// Log warning but don't fail
		fmt.Printf("Warning: Decision latency %v exceeds 500ms requirement\n", decision.DecisionLatency)
	}

	return decision, nil
}

// shouldConsiderOffloading checks if offloading should be considered
func (de *DecisionEngine) shouldConsiderOffloading(state models.SystemState) (bool, string) {
	// Don't offload if local resources are underutilized
	if float64(state.ComputeUsage) < de.safetyMargins.MinLocalCompute && 
	   float64(state.MemoryUsage) < de.safetyMargins.MinLocalMemory {
		return false, "local resources underutilized"
	}

	// Consider offloading if queue is building up
	if state.QueueDepth > state.QueueThreshold {
		return true, "queue pressure"
	}

	// Consider offloading if system is under high load
	if state.IsHighLoad() {
		return true, "high system load"
	}

	// Consider offloading if load score is above threshold
	if state.GetLoadScore() > 0.6 {
		return true, "load score above threshold"
	}

	return false, "no offload trigger"
}

// filterTargets filters targets based on constraints
func (de *DecisionEngine) filterTargets(
	process models.Process,
	targets []models.OffloadTarget,
	state models.SystemState,
) []models.OffloadTarget {
	viable := make([]models.OffloadTarget, 0)

	for _, target := range targets {
		// Skip unhealthy targets
		if !target.IsHealthy() {
			continue
		}

		// Skip targets below minimum reliability
		if target.Reliability < de.safetyMargins.MinReliability {
			continue
		}

		// Skip targets that can't accommodate the process
		if !target.CanAccommodate(process) {
			continue
		}

		// Skip targets with excessive latency for real-time processes
		if process.RealTime && target.NetworkLatency > de.safetyMargins.MaxLatencyTolerance {
			continue
		}

		// Skip non-local targets for safety-critical processes
		if process.SafetyCritical && target.Type != models.LOCAL {
			continue
		}

		// Skip targets that don't meet security requirements
		if process.SecurityLevel > target.SecurityLevel {
			continue
		}

		// Check data locality requirements
		if process.LocalityRequired && target.Type != models.LOCAL && target.Type != models.EDGE {
			continue
		}

		viable = append(viable, target)
	}

	return viable
}

// scoreTargets computes scores for each target
func (de *DecisionEngine) scoreTargets(
	process models.Process,
	targets []models.OffloadTarget,
	state models.SystemState,
	pattern *DiscoveredPattern,
) map[string]float64 {
	scores := make(map[string]float64)
	weights := de.weights

	// Apply pattern weight adjustments if applicable
	if pattern != nil && pattern.ValidationStatus == VALIDATED {
		weights = de.applyPatternWeights(weights, pattern)
	}

	for _, target := range targets {
		score := de.computeTargetScore(process, target, state, weights)
		scores[target.ID] = score
	}

	return scores
}

// computeTargetScore computes a single target's score
func (de *DecisionEngine) computeTargetScore(
	process models.Process,
	target models.OffloadTarget,
	state models.SystemState,
	weights AdaptiveWeights,
) float64 {
	components := ScoreBreakdown{
		WeightsUsed: weights,
	}

	// Queue impact: How much this helps reduce queue pressure
	if state.QueueThreshold > 0 {
		queuePressure := float64(state.QueueDepth) / float64(state.QueueThreshold)
		components.QueueImpact = math.Min(1.0, 1.0/(1.0+math.Exp(-2*(queuePressure-0.5))))
	}

	// Load balance: How well this balances the load
	localLoad := state.GetLoadScore()
	targetLoad := target.CurrentLoad
	loadDiff := math.Abs(localLoad - targetLoad)
	components.LoadBalance = 1.0 - loadDiff

	// Network cost: Normalized network transfer cost
	dataSize := float64(process.InputSize + process.OutputSize)
	maxDataSize := float64(100 * 1024 * 1024) // 100MB baseline
	normalizedDataCost := math.Min(1.0, dataSize/maxDataSize)
	latencyFactor := math.Min(1.0, float64(target.NetworkLatency)/(100*float64(time.Millisecond)))
	components.NetworkCost = 1.0 - (0.5*normalizedDataCost + 0.5*latencyFactor)

	// Latency impact: How latency affects the process
	estimatedTime := target.EstimateExecutionTime(process)
	if process.MaxDuration > 0 {
		timeRatio := float64(estimatedTime) / float64(process.MaxDuration)
		components.LatencyImpact = math.Max(0.0, 1.0-timeRatio)
	} else {
		// No deadline, score based on absolute latency
		components.LatencyImpact = 1.0 / (1.0 + float64(estimatedTime)/(30*float64(time.Second)))
	}

	// Energy impact: Favor energy-efficient targets
	energyScore := 1.0 - target.EnergyCost/10.0 // Normalized energy cost
	components.EnergyImpact = math.Max(0.0, math.Min(1.0, energyScore))

	// Policy match: How well target matches policy preferences
	components.PolicyMatch = target.GetCompatibilityScore(process)
	
	// Add historical success bonus
	if target.HistoricalSuccess > 0 {
		components.PolicyMatch = components.PolicyMatch*0.7 + target.HistoricalSuccess*0.3
	}

	// Compute weighted score
	finalScore := weights.QueueDepth*components.QueueImpact +
		weights.ProcessorLoad*components.LoadBalance +
		weights.NetworkCost*components.NetworkCost +
		weights.LatencyCost*components.LatencyImpact +
		weights.EnergyCost*components.EnergyImpact +
		weights.PolicyCost*components.PolicyMatch

	// Ensure score is in [0.0, 1.0] range
	return math.Max(0.0, math.Min(1.0, finalScore))
}

// selectBestTarget selects the target with the highest score
func (de *DecisionEngine) selectBestTarget(
	scores map[string]float64,
	targets []models.OffloadTarget,
) (*models.OffloadTarget, float64) {
	if len(scores) == 0 {
		return nil, 0.0
	}

	// Find target with highest score
	var bestTarget *models.OffloadTarget
	bestScore := 0.0

	for _, target := range targets {
		score := scores[target.ID]
		if score > bestScore {
			bestScore = score
			bestTarget = &target
		}
	}

	return bestTarget, bestScore
}

// findBestPattern finds the best matching pattern for the current situation
func (de *DecisionEngine) findBestPattern(process models.Process, state models.SystemState) *DiscoveredPattern {
	var bestPattern *DiscoveredPattern
	bestMatch := 0.0

	for _, pattern := range de.patterns {
		if pattern.ValidationStatus != VALIDATED {
			continue
		}

		matchScore := de.evaluatePatternMatch(pattern, process, state)
		if matchScore > bestMatch && matchScore > 0.7 { // Minimum match threshold
			bestMatch = matchScore
			bestPattern = pattern
		}
	}

	return bestPattern
}

// evaluatePatternMatch evaluates how well a pattern matches current conditions
func (de *DecisionEngine) evaluatePatternMatch(
	pattern *DiscoveredPattern,
	process models.Process,
	state models.SystemState,
) float64 {
	if len(pattern.Conditions) == 0 {
		return 0.0
	}

	totalWeight := 0.0
	matchedWeight := 0.0

	for _, condition := range pattern.Conditions {
		totalWeight += condition.Weight
		if de.evaluateCondition(condition, process, state) {
			matchedWeight += condition.Weight
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return matchedWeight / totalWeight
}

// evaluateCondition evaluates a single pattern condition
func (de *DecisionEngine) evaluateCondition(
	condition PatternCondition,
	process models.Process,
	state models.SystemState,
) bool {
	// This is a simplified implementation
	// In production, this would use reflection or a more sophisticated approach
	
	var value interface{}
	
	switch condition.Field {
	case "QueueDepth":
		value = state.QueueDepth
	case "ComputeUsage":
		value = state.ComputeUsage
	case "ProcessPriority":
		value = process.Priority
	case "ProcessType":
		value = process.Type
	default:
		return false
	}

	return de.compareValues(value, condition.Operator, condition.Value)
}

// compareValues compares values based on operator
func (de *DecisionEngine) compareValues(actual interface{}, op models.Operator, expected interface{}) bool {
	// Simplified comparison logic
	switch op {
	case models.EQUAL_TO:
		return actual == expected
	case models.NOT_EQUAL_TO:
		return actual != expected
	case models.GREATER_THAN:
		return de.compareNumeric(actual, expected) > 0
	case models.LESS_THAN:
		return de.compareNumeric(actual, expected) < 0
	case models.GREATER_EQUAL:
		return de.compareNumeric(actual, expected) >= 0
	case models.LESS_EQUAL:
		return de.compareNumeric(actual, expected) <= 0
	default:
		return false
	}
}

// compareNumeric compares numeric values
func (de *DecisionEngine) compareNumeric(a, b interface{}) float64 {
	aFloat := de.toFloat64(a)
	bFloat := de.toFloat64(b)
	return aFloat - bFloat
}

// toFloat64 converts interface to float64
func (de *DecisionEngine) toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case models.Utilization:
		return float64(val)
	default:
		return 0.0
	}
}

// applyPatternWeights applies pattern weight adjustments
func (de *DecisionEngine) applyPatternWeights(
	baseWeights AdaptiveWeights,
	pattern *DiscoveredPattern,
) AdaptiveWeights {
	adjusted := baseWeights

	for field, adjustment := range pattern.WeightAdjustments {
		switch field {
		case "QueueDepth":
			adjusted.QueueDepth *= (1.0 + adjustment)
		case "ProcessorLoad":
			adjusted.ProcessorLoad *= (1.0 + adjustment)
		case "NetworkCost":
			adjusted.NetworkCost *= (1.0 + adjustment)
		case "LatencyCost":
			adjusted.LatencyCost *= (1.0 + adjustment)
		case "EnergyCost":
			adjusted.EnergyCost *= (1.0 + adjustment)
		case "PolicyCost":
			adjusted.PolicyCost *= (1.0 + adjustment)
		}
	}

	adjusted.Normalize()
	return adjusted
}

// createLocalDecision creates a decision to execute locally
func (de *DecisionEngine) createLocalDecision(
	process models.Process,
	reason string,
	startTime time.Time,
) OffloadDecision {
	return OffloadDecision{
		ShouldOffload:    false,
		Target:           nil,
		Confidence:       0.9,
		Score:            1.0,
		PolicyViolations: []string{},
		Strategy:         IMMEDIATE,
		ExpectedBenefit:  0.0,
		EstimatedCost:    0.0,
		DecisionTime:     startTime,
		DecisionLatency:  time.Since(startTime),
		AlgorithmVersion: de.algorithmVersion,
		ScoreComponents: ScoreBreakdown{
			QueueImpact:   0.0,
			LoadBalance:   1.0,
			NetworkCost:   1.0,
			LatencyImpact: 1.0,
			EnergyImpact:  1.0,
			PolicyMatch:   1.0,
			WeightsUsed:   de.weights,
		},
	}
}

// createOffloadDecision creates an offload decision
func (de *DecisionEngine) createOffloadDecision(
	process models.Process,
	target *models.OffloadTarget,
	score float64,
	pattern *DiscoveredPattern,
	startTime time.Time,
) OffloadDecision {
	// Calculate expected benefit
	localExecutionTime := process.EstimatedDuration
	targetExecutionTime := target.EstimateExecutionTime(process)
	timeSavings := float64(localExecutionTime - targetExecutionTime)
	expectedBenefit := math.Max(0, timeSavings/float64(localExecutionTime))

	// Calculate estimated cost
	estimatedCost := target.GetTotalCost(process)

	// Determine execution strategy
	strategy := IMMEDIATE
	if process.HasDAG {
		strategy = PIPELINED
	} else if target.EstimatedWaitTime > 10*time.Second {
		strategy = DELAYED
	}

	// Calculate confidence based on score and pattern match
	confidence := score
	if pattern != nil {
		confidence = score*0.7 + pattern.Confidence*0.3
	}

	return OffloadDecision{
		ShouldOffload:    true,
		Target:           target,
		Confidence:       confidence,
		Score:            score,
		AppliedPattern:   pattern,
		PolicyViolations: []string{},
		Strategy:         strategy,
		ExpectedBenefit:  expectedBenefit,
		EstimatedCost:    estimatedCost,
		DecisionTime:     startTime,
		DecisionLatency:  time.Since(startTime),
		AlgorithmVersion: de.algorithmVersion,
		ScoreComponents:  ScoreBreakdown{WeightsUsed: de.weights},
	}
}

// UpdateWeights updates the adaptive weights
func (de *DecisionEngine) UpdateWeights(weights AdaptiveWeights) {
	weights.Normalize()
	de.weights = weights
}

// AddPattern adds a discovered pattern
func (de *DecisionEngine) AddPattern(pattern *DiscoveredPattern) {
	de.patterns = append(de.patterns, pattern)
	
	// Keep only the most recent patterns (max 50)
	if len(de.patterns) > 50 {
		// Sort by last used time and keep most recent
		sort.Slice(de.patterns, func(i, j int) bool {
			return de.patterns[i].LastUsed.After(de.patterns[j].LastUsed)
		})
		de.patterns = de.patterns[:50]
	}
}

// SetSafetyMargins updates safety margins
func (de *DecisionEngine) SetSafetyMargins(margins SafetyMargins) {
	de.safetyMargins = margins
}

// GetWeights returns current weights
func (de *DecisionEngine) GetWeights() AdaptiveWeights {
	return de.weights
}

// GetPatterns returns current patterns
func (de *DecisionEngine) GetPatterns() []*DiscoveredPattern {
	return de.patterns
}