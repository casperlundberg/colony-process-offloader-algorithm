package algorithm

import (
	"fmt"
	"math"
	"time"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/learning"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/policy"
)

// ConfigurableCAPE implements the complete configurable CAPE algorithm from FURTHER_CONTROL_THEORY.md
type ConfigurableCAPE struct {
	// Configuration
	config *models.DeploymentConfig
	
	// Core Components
	dataGravity        *models.DataGravityModel
	thompsonSampler    *learning.ThompsonSampler
	qLearning          *learning.QLearning
	policyEngine       *policy.PolicyEngine
	
	// Algorithm Components from FURTHER_CONTROL_THEORY.md
	arima              *learning.ARIMA       // [1] ARIMA(3,1,2) predictor
	ewma               *learning.EWMA        // [2] EWMA(0.167) smoother  
	cusum              *learning.CUSUM       // [3] CUSUM(0.5σ, 5σ) anomaly detector
	sgd                *learning.SGD         // [4] SGD(0.001) optimizer
	dtw                *learning.DTW         // [5] DTW pattern matcher
	
	// Decision tracking
	decisionHistory    []CAPEDecision
	performanceHistory []CAPEPerformanceMetrics
	
	// Adaptation parameters
	lastAdaptation     time.Time
	adaptationInterval time.Duration
	
	// Runtime state
	currentStrategy    models.Strategy
	totalDecisions     int
	successfulDecisions int
}

// CAPEDecision represents a complete decision made by the configurable CAPE algorithm
type CAPEDecision struct {
	DecisionID          string                      `json:"decision_id"`
	Timestamp           time.Time                   `json:"timestamp"`
	
	// Input context
	Process             models.Process              `json:"process"`
	Metrics             *models.ExtendedMetricsVector `json:"metrics"`
	AvailableTargets    []models.OffloadTarget      `json:"available_targets"`
	
	// Decision logic
	SelectedStrategy    models.Strategy             `json:"selected_strategy"`
	SelectedTarget      models.OffloadTarget        `json:"selected_target"`
	PlacementScores     map[string]float64          `json:"placement_scores"`
	
	// Reasoning
	ObjectiveScores     map[string]float64          `json:"objective_scores"`
	DataGravityScore    float64                     `json:"data_gravity_score"`
	PolicyEvaluation    policy.PolicyEvaluation    `json:"policy_evaluation"`
	
	// Outcome (filled in later)
	Outcome             *CAPEOutcome                `json:"outcome,omitempty"`
}

// CAPEOutcome represents the outcome of a CAPE decision
type CAPEOutcome struct {
	Success             bool                        `json:"success"`
	LatencyMS           float64                     `json:"latency_ms"`
	CostUSD             float64                     `json:"cost_usd"`
	ThroughputOps       float64                     `json:"throughput_ops"`
	EnergyWh            float64                     `json:"energy_wh"`
	DataTransferGB      float64                     `json:"data_transfer_gb"`
	SLAViolation        bool                        `json:"sla_violation"`
	BudgetOverrun       bool                        `json:"budget_overrun"`
	CompletedAt         time.Time                   `json:"completed_at"`
}

// CAPEPerformanceMetrics tracks system performance over time for CAPE
type CAPEPerformanceMetrics struct {
	Timestamp           time.Time                   `json:"timestamp"`
	AvgLatencyMS        float64                     `json:"avg_latency_ms"`
	AvgCostUSD          float64                     `json:"avg_cost_usd"`
	SLAViolationRate    float64                     `json:"sla_violation_rate"`
	BudgetOverrunRate   float64                     `json:"budget_overrun_rate"`
	SuccessRate         float64                     `json:"success_rate"`
	DataMovementGB      float64                     `json:"data_movement_gb"`
	EnergyEfficiency    float64                     `json:"energy_efficiency"`
}

// NewConfigurableCAPE creates a new configurable CAPE algorithm instance
func NewConfigurableCAPE(config *models.DeploymentConfig) *ConfigurableCAPE {
	cape := &ConfigurableCAPE{
		config:             config,
		dataGravity:        models.NewDataGravityModel(),
		thompsonSampler:    learning.NewThompsonSampler(config.ExplorationFactor),
		qLearning:          learning.NewQLearning(config.LearningRate, 0.9, config.ExplorationFactor, 10),
		policyEngine:       policy.NewPolicyEngine(),
		
		// Initialize all algorithms from FURTHER_CONTROL_THEORY.md
		arima:              learning.NewARIMA(),                    // [1] ARIMA(3,1,2)
		ewma:               learning.NewEWMADefault(),              // [2] EWMA(0.167)
		cusum:              learning.NewCUSUMAdaptive(),            // [3] CUSUM(0.5σ, 5σ)
		sgd:                learning.NewSGD(0.001, 10),            // [4] SGD(0.001)
		dtw:                learning.NewDTW(),                     // [5] DTW pattern matcher
		
		decisionHistory:    make([]CAPEDecision, 0),
		performanceHistory: make([]CAPEPerformanceMetrics, 0),
		lastAdaptation:     time.Now(),
		adaptationInterval: time.Hour, // Adapt every hour
		currentStrategy:    models.StrategyBalanced,
	}
	
	return cape
}

// MakeDecision implements the complete configurable CAPE decision algorithm
func (cape *ConfigurableCAPE) MakeDecision(
	process models.Process,
	availableTargets []models.OffloadTarget,
	systemState models.SystemState,
) (CAPEDecision, error) {
	
	// Create extended metrics vector
	dagContext := cape.createDAGContext(process)
	metrics := models.NewExtendedMetricsVector(
		systemState,
		cape.inferDataLocation(process),
		cape.estimateDataSize(process),
		*dagContext,
	)
	
	// Step 1: Determine strategy based on learned performance (if enabled)
	if cape.config.StrategyEnabled {
		cape.currentStrategy = cape.thompsonSampler.SelectStrategy()
		cape.config.UpdateStrategy(cape.currentStrategy)
	}
	
	// Step 2: Calculate DAG-aware capacity needed with prediction enhancement
	baseCapacity := dagContext.CalculateDAGAwareCapacity()
	
	// Use ARIMA to predict future capacity needs
	currentLoad := float64(metrics.SystemState.ComputeUsage)
	cape.arima.AddObservation(currentLoad)
	predictedLoad, err := cape.arima.Predict()
	
	var capacityNeeded float64
	if err == nil {
		// Adjust capacity based on predicted load
		capacityNeeded = baseCapacity * (1.0 + (predictedLoad-currentLoad)*0.5)
	} else {
		capacityNeeded = baseCapacity
	}
	
	// Smooth capacity prediction with EWMA
	capacityNeeded = cape.ewma.Update(capacityNeeded)
	
	// Check for capacity anomalies with CUSUM
	cusumResult := cape.cusum.Update(capacityNeeded)
	if cusumResult.IsAnomaly {
		// Apply anomaly penalty to encourage conservative decisions
		capacityNeeded *= 1.2
	}
	
	// Step 3: Evaluate placement options considering ALL factors
	placementScores := make(map[string]float64)
	objectiveScores := make(map[string]float64)
	
	var selectedTarget models.OffloadTarget
	bestScore := math.Inf(-1)
	
	for _, target := range availableTargets {
		// Check policy constraints first
		policyEval := cape.policyEngine.EvaluatePolicy(process, target)
		if policyEval.HasHardViolations() {
			continue // Skip targets that violate hard constraints
		}
		
		// Calculate placement score
		score := cape.calculatePlacementScore(metrics, target, capacityNeeded, policyEval)
		placementScores[target.ID] = score
		
		if score > bestScore {
			bestScore = score
			selectedTarget = target
		}
	}
	
	if selectedTarget.ID == "" {
		return CAPEDecision{}, fmt.Errorf("no suitable target found after policy evaluation")
	}
	
	// Step 4: Create decision record
	decision := CAPEDecision{
		DecisionID:       fmt.Sprintf("cape-%d-%d", time.Now().Unix(), cape.totalDecisions),
		Timestamp:        time.Now(),
		Process:          process,
		Metrics:          metrics,
		AvailableTargets: availableTargets,
		SelectedStrategy: cape.currentStrategy,
		SelectedTarget:   selectedTarget,
		PlacementScores:  placementScores,
		ObjectiveScores:  objectiveScores,
		DataGravityScore: cape.dataGravity.CalculateDataGravityScore(
			metrics.DataLocation, 
			models.DataLocation(selectedTarget.Location),
		),
		PolicyEvaluation: cape.policyEngine.EvaluatePolicy(process, selectedTarget),
	}
	
	// Step 5: Learn from decision (Q-learning update will happen when outcome is reported)
	cape.decisionHistory = append(cape.decisionHistory, decision)
	cape.totalDecisions++
	
	// Step 6: Trigger adaptation if needed
	if time.Since(cape.lastAdaptation) > cape.adaptationInterval {
		cape.adaptOverTime()
		cape.lastAdaptation = time.Now()
	}
	
	return decision, nil
}

// calculatePlacementScore calculates comprehensive placement score
func (cape *ConfigurableCAPE) calculatePlacementScore(
	metrics *models.ExtendedMetricsVector,
	target models.OffloadTarget,
	capacityNeeded float64,
	policyEval policy.PolicyEvaluation,
) float64 {
	targetLocation := models.DataLocation(target.Location)
	
	// Calculate component costs/scores
	transferCost := cape.calculateTransferCost(metrics, targetLocation)
	computeCost := cape.calculateComputeCost(target, capacityNeeded)
	downstreamPenalty := metrics.DAGContext.EstimateDownstreamPenalty(targetLocation)
	
	// Evaluate against configured objectives
	totalScore := cape.evaluateObjective(
		cape.config.OptimizationGoals,
		transferCost,
		computeCost,
		downstreamPenalty,
		metrics,
		target,
	)
	
	// Apply data gravity factor
	placementScore := cape.dataGravity.CalculatePlacementScore(
		totalScore,
		metrics.DataLocation,
		targetLocation,
		cape.config.DataGravityFactor,
	)
	
	// Apply soft constraint penalties
	softPenalty := 0.0
	for _, violation := range policyEval.SoftViolations {
		// Convert severity to penalty weight
		var penaltyWeight float64
		switch violation.Severity {
		case policy.CRITICAL:
			penaltyWeight = 1.0
		case policy.HIGH:
			penaltyWeight = 0.8
		case policy.MEDIUM:
			penaltyWeight = 0.5
		case policy.LOW:
			penaltyWeight = 0.2
		default:
			penaltyWeight = 0.3
		}
		softPenalty += penaltyWeight * 0.1 // Small penalty for soft violations
	}
	
	finalScore := placementScore - softPenalty
	
	return finalScore
}

// calculateTransferCost estimates cost of transferring data to target location
func (cape *ConfigurableCAPE) calculateTransferCost(
	metrics *models.ExtendedMetricsVector,
	targetLocation models.DataLocation,
) float64 {
	transferCosts := models.DefaultTransferCosts()
	return transferCosts.GetTransferCost(
		metrics.DataLocation,
		targetLocation,
		metrics.DataSizePendingGB,
	)
}

// calculateComputeCost estimates computational cost at target
func (cape *ConfigurableCAPE) calculateComputeCost(
	target models.OffloadTarget,
	capacityNeeded float64,
) float64 {
	// Simple cost model based on target capacity and utilization
	utilizationPenalty := 1.0 + target.Utilization.ComputeUsage
	return capacityNeeded * utilizationPenalty * 0.1 // Base cost factor
}

// evaluateObjective evaluates placement against configured optimization goals
func (cape *ConfigurableCAPE) evaluateObjective(
	goals []models.OptimizationGoal,
	transferCost float64,
	computeCost float64,
	downstreamPenalty float64,
	metrics *models.ExtendedMetricsVector,
	target models.OffloadTarget,
) float64 {
	score := 0.0
	
	for _, goal := range goals {
		var component float64
		
		switch goal.Metric {
		case "data_movement":
			component = transferCost + downstreamPenalty
		case "compute_cost":
			component = computeCost
		case "latency":
			component = cape.estimateLatency(metrics, target)
		case "throughput":
			component = -cape.estimateThroughput(target) // Negative for maximization
		case "energy_efficiency":
			component = -cape.estimateEnergyEfficiency(target) // Negative for maximization
		default:
			continue
		}
		
		// Apply minimize/maximize direction
		if !goal.Minimize {
			component = -component
		}
		
		score += goal.Weight * component
	}
	
	return -score // Convert to maximization problem (higher score = better)
}

// Helper estimation functions
func (cape *ConfigurableCAPE) estimateLatency(metrics *models.ExtendedMetricsVector, target models.OffloadTarget) float64 {
	// Simple latency model
	baseLatency := 10.0 // Base latency in ms
	utilizationPenalty := target.Utilization.ComputeUsage * 100.0
	transferLatency := metrics.TransferTimeEst.Seconds() * 1000.0 // Convert to ms
	
	return baseLatency + utilizationPenalty + transferLatency
}

func (cape *ConfigurableCAPE) estimateThroughput(target models.OffloadTarget) float64 {
	// Simple throughput model (ops/sec)
	baseThroughput := 100.0
	utilizationFactor := 1.0 - target.Utilization.ComputeUsage
	return baseThroughput * utilizationFactor
}

func (cape *ConfigurableCAPE) estimateEnergyEfficiency(target models.OffloadTarget) float64 {
	// Simple energy efficiency model (higher is better)
	baseEfficiency := 0.8
	utilizationBonus := (1.0 - target.Utilization.ComputeUsage) * 0.2
	return baseEfficiency + utilizationBonus
}

// ReportOutcome reports the outcome of a decision for learning
func (cape *ConfigurableCAPE) ReportOutcome(decisionID string, outcome CAPEOutcome) error {
	// Find the decision
	var decision *CAPEDecision
	for i := range cape.decisionHistory {
		if cape.decisionHistory[i].DecisionID == decisionID {
			decision = &cape.decisionHistory[i]
			break
		}
	}
	
	if decision == nil {
		return fmt.Errorf("decision not found: %s", decisionID)
	}
	
	decision.Outcome = &outcome
	
	if outcome.Success {
		cape.successfulDecisions++
	}
	
	// Update Thompson sampler
	if cape.config.StrategyEnabled {
		strategyOutcome := learning.EvaluateOutcome(
			decision.SelectedStrategy,
			outcome.LatencyMS,
			outcome.CostUSD,
			outcome.ThroughputOps,
			outcome.EnergyWh,
			1000.0, // SLA threshold (configurable)
			100.0,  // Budget threshold (configurable)
		)
		cape.thompsonSampler.UpdateStrategy(decision.SelectedStrategy, strategyOutcome)
	}
	
	// Update Q-learning (would need next state - simplified for now)
	// In a real implementation, this would be updated when the next decision is made
	
	return nil
}

// adaptOverTime implements the adaptive learning from FURTHER_CONTROL_THEORY.md
func (cape *ConfigurableCAPE) adaptOverTime() {
	if len(cape.performanceHistory) == 0 {
		return
	}
	
	// Get recent performance
	recentPerformance := cape.getRecentPerformance()
	
	// Use SGD to optimize configuration weights based on performance
	cape.optimizeWeightsWithSGD(recentPerformance)
	
	// Pattern discovery using DTW
	cape.discoverDecisionPatterns()
	
	// Learn data gravity factor for this specific workload
	if recentPerformance.DataMovementGB > 100.0 { // High data movement
		cape.config.DataGravityFactor = math.Min(0.95, cape.config.DataGravityFactor*1.05)
	}
	
	cape.config.UpdatedAt = time.Now()
}

// optimizeWeightsWithSGD uses SGD to optimize objective weights
func (cape *ConfigurableCAPE) optimizeWeightsWithSGD(performance CAPEPerformanceMetrics) {
	// Convert optimization goals to parameter vector
	parameters := make([]float64, len(cape.config.OptimizationGoals))
	for i, goal := range cape.config.OptimizationGoals {
		parameters[i] = goal.Weight
	}
	
	// Calculate gradients based on performance gaps
	gradients := make([]float64, len(parameters))
	
	// Calculate cost based on performance violations
	cost := 0.0
	
	// SLA violation penalty
	if performance.SLAViolationRate > 0.1 {
		cost += performance.SLAViolationRate * 10.0
		// Increase gradient for latency-related objectives
		for i, goal := range cape.config.OptimizationGoals {
			if goal.Metric == "latency" {
				gradients[i] = -0.1 // Increase weight
			} else if goal.Metric == "compute_cost" {
				gradients[i] = 0.05 // Decrease weight
			}
		}
	}
	
	// Budget overrun penalty  
	if performance.BudgetOverrunRate > 0.1 {
		cost += performance.BudgetOverrunRate * 10.0
		// Increase gradient for cost-related objectives
		for i, goal := range cape.config.OptimizationGoals {
			if goal.Metric == "compute_cost" {
				gradients[i] = -0.1 // Increase weight
			} else if goal.Metric == "latency" {
				gradients[i] = 0.05 // Decrease weight
			}
		}
	}
	
	// Energy efficiency reward
	if performance.EnergyEfficiency > 0.8 {
		cost -= performance.EnergyEfficiency * 2.0
		for i, goal := range cape.config.OptimizationGoals {
			if goal.Metric == "energy_efficiency" {
				gradients[i] = -0.05 // Increase weight
			}
		}
	}
	
	// Update weights using SGD
	err := cape.sgd.Update(parameters, gradients, cost)
	if err == nil {
		// Apply updated weights (with normalization)
		updatedParams := cape.sgd.GetCurrentParameters()
		
		// Normalize weights to sum to 1.0
		weightSum := 0.0
		for _, weight := range updatedParams {
			weightSum += math.Max(0.01, weight) // Ensure positive weights
		}
		
		if weightSum > 0 {
			for i := range cape.config.OptimizationGoals {
				cape.config.OptimizationGoals[i].Weight = math.Max(0.01, updatedParams[i]) / weightSum
			}
		}
	}
}

// discoverDecisionPatterns uses DTW to discover patterns in decision history
func (cape *ConfigurableCAPE) discoverDecisionPatterns() {
	if len(cape.decisionHistory) < 10 {
		return
	}
	
	// Extract decision sequence for pattern analysis
	decisionSequence := make([]float64, len(cape.decisionHistory))
	for i, decision := range cape.decisionHistory {
		// Convert decision to numerical representation
		score := 0.5 // Default
		if len(decision.PlacementScores) > 0 {
			for _, s := range decision.PlacementScores {
				score = s
				break // Use first score
			}
		}
		decisionSequence[i] = score
	}
	
	// Discover patterns in decision sequence
	patterns, err := cape.dtw.DiscoverPatterns(decisionSequence, 3, 8)
	if err == nil && len(patterns) > 0 {
		// Add discovered patterns to DTW library
		for _, pattern := range patterns {
			cape.dtw.AddPattern(pattern.Data, pattern.Label, "decision_sequence")
		}
	}
}

// Helper functions
func (cape *ConfigurableCAPE) createDAGContext(process models.Process) *models.DAGContext {
	// Simplified DAG context creation
	// In a real implementation, this would analyze the process specification
	return &models.DAGContext{
		PipelineID:       fmt.Sprintf("pipeline-%s", process.ID),
		CurrentStage:     0,
		TotalStages:      3,
		SafetyFactor:     1.2,
		PipelineDeadline: time.Now().Add(time.Minute * 10),
		Stages: []models.DAGStage{
			{
				StageID:             0,
				StageName:           "input_processing",
				InputLocation:       models.DataLocationEdge,
				PreferredLocation:   models.DataLocationEdge,
				InputSizeGB:         1.0,
				EstimatedOutputGB:   0.5,
				ComputeRequirement:  2.0,
				MemoryRequirementGB: 4.0,
				UpstreamStages:      []int{},
				DownstreamStages:    []int{1},
			},
		},
	}
}

func (cape *ConfigurableCAPE) inferDataLocation(process models.Process) models.DataLocation {
	// Simple data location inference based on process spec
	// In a real implementation, this would analyze process input sources
	return models.DataLocationEdge
}

func (cape *ConfigurableCAPE) estimateDataSize(process models.Process) float64 {
	// Simple data size estimation
	// In a real implementation, this would analyze process requirements
	return 1.0 // 1 GB default
}

func (cape *ConfigurableCAPE) getRecentPerformance() CAPEPerformanceMetrics {
	if len(cape.performanceHistory) == 0 {
		return CAPEPerformanceMetrics{}
	}
	
	// Return most recent performance metrics
	return cape.performanceHistory[len(cape.performanceHistory)-1]
}

// GetStats returns comprehensive statistics about the CAPE algorithm
func (cape *ConfigurableCAPE) GetStats() CAPEStats {
	thompsonStats := cape.thompsonSampler.GetStrategyStats()
	qStats := cape.qLearning.GetLearningStats()
	
	successRate := 0.0
	if cape.totalDecisions > 0 {
		successRate = float64(cape.successfulDecisions) / float64(cape.totalDecisions)
	}
	
	return CAPEStats{
		TotalDecisions:      cape.totalDecisions,
		SuccessfulDecisions: cape.successfulDecisions,
		SuccessRate:         successRate,
		CurrentStrategy:     cape.currentStrategy,
		ThompsonStats:       thompsonStats,
		QLearningStats:      qStats,
		LastAdaptation:      cape.lastAdaptation,
		ConfigUpdatedAt:     cape.config.UpdatedAt,
	}
}

// CAPEStats provides comprehensive statistics about the algorithm
type CAPEStats struct {
	TotalDecisions      int                                   `json:"total_decisions"`
	SuccessfulDecisions int                                   `json:"successful_decisions"`
	SuccessRate         float64                               `json:"success_rate"`
	CurrentStrategy     models.Strategy                       `json:"current_strategy"`
	ThompsonStats       map[models.Strategy]learning.StrategyStats `json:"thompson_stats"`
	QLearningStats      learning.QLearningStats               `json:"q_learning_stats"`
	LastAdaptation      time.Time                             `json:"last_adaptation"`
	ConfigUpdatedAt     time.Time                             `json:"config_updated_at"`
}