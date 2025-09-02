package decision

import (
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// AdaptiveWeights represents the adaptive weights for scoring
type AdaptiveWeights struct {
	QueueDepth    float64 `json:"queue_depth"`
	ProcessorLoad float64 `json:"processor_load"`
	NetworkCost   float64 `json:"network_cost"`
	LatencyCost   float64 `json:"latency_cost"`
	EnergyCost    float64 `json:"energy_cost"`
	PolicyCost    float64 `json:"policy_cost"`
}

// Normalize ensures weights sum to 1.0
func (w *AdaptiveWeights) Normalize() {
	sum := w.QueueDepth + w.ProcessorLoad + w.NetworkCost + 
		   w.LatencyCost + w.EnergyCost + w.PolicyCost
	
	if sum == 0 {
		// Set default weights if all are zero
		w.QueueDepth = 0.2
		w.ProcessorLoad = 0.2
		w.NetworkCost = 0.2
		w.LatencyCost = 0.2
		w.EnergyCost = 0.1
		w.PolicyCost = 0.1
		return
	}
	
	// Normalize to sum to 1.0
	w.QueueDepth /= sum
	w.ProcessorLoad /= sum
	w.NetworkCost /= sum
	w.LatencyCost /= sum
	w.EnergyCost /= sum
	w.PolicyCost /= sum
}

// Sum returns the sum of all weights
func (w AdaptiveWeights) Sum() float64 {
	return w.QueueDepth + w.ProcessorLoad + w.NetworkCost + 
		   w.LatencyCost + w.EnergyCost + w.PolicyCost
}

// OffloadDecision represents the algorithm's decision output
type OffloadDecision struct {
	// Core decision
	ShouldOffload bool                   `json:"should_offload"`
	Target        *models.OffloadTarget  `json:"target"`
	Confidence    float64                `json:"confidence"`
	
	// Decision reasoning
	Score           float64              `json:"score"`
	ScoreComponents ScoreBreakdown       `json:"score_components"`
	AppliedPattern  *DiscoveredPattern   `json:"applied_pattern"`
	PolicyViolations []string            `json:"policy_violations"`
	
	// Execution strategy
	Strategy        ExecutionStrategy    `json:"strategy"`
	ExpectedBenefit float64              `json:"expected_benefit"`
	EstimatedCost   float64              `json:"estimated_cost"`
	
	// Metadata
	DecisionTime    time.Time            `json:"decision_time"`
	DecisionLatency time.Duration        `json:"decision_latency"`
	AlgorithmVersion string              `json:"algorithm_version"`
}

// ScoreBreakdown provides transparency into decision factors
type ScoreBreakdown struct {
	QueueImpact   float64         `json:"queue_impact"`
	LoadBalance   float64         `json:"load_balance"`
	NetworkCost   float64         `json:"network_cost"`
	LatencyImpact float64         `json:"latency_impact"`
	EnergyImpact  float64         `json:"energy_impact"`
	PolicyMatch   float64         `json:"policy_match"`
	WeightsUsed   AdaptiveWeights `json:"weights_used"`
}

// ExecutionStrategy defines how to execute the offload
type ExecutionStrategy string

const (
	IMMEDIATE    ExecutionStrategy = "immediate"
	DELAYED      ExecutionStrategy = "delayed"
	BATCHED      ExecutionStrategy = "batched"
	PIPELINED    ExecutionStrategy = "pipelined"
)

// DiscoveredPattern represents learned behavioral patterns
type DiscoveredPattern struct {
	ID                string                     `json:"id"`
	Name              string                     `json:"name"`
	Description       string                     `json:"description"`
	Conditions        []PatternCondition         `json:"conditions"`
	Confidence        float64                    `json:"confidence"`
	RecommendedAction models.ActionType          `json:"recommended_action"`
	PreferredTargets  []string                   `json:"preferred_targets"`
	WeightAdjustments map[string]float64         `json:"weight_adjustments"`
	ApplicationCount  int                        `json:"application_count"`
	SuccessRate       float64                    `json:"success_rate"`
	AvgBenefit        float64                    `json:"avg_benefit"`
	CreatedTime       time.Time                  `json:"created_time"`
	LastUpdated       time.Time                  `json:"last_updated"`
	LastUsed          time.Time                  `json:"last_used"`
	Stability         float64                    `json:"stability"`
	MinSamples        int                        `json:"min_samples"`
	ValidationStatus  PatternStatus              `json:"validation_status"`
}

// PatternCondition defines when a pattern should be applied
type PatternCondition struct {
	Field    string           `json:"field"`
	Operator models.Operator  `json:"operator"`
	Value    interface{}      `json:"value"`
	Weight   float64          `json:"weight"`
}

// PatternStatus represents the validation status of a pattern
type PatternStatus string

const (
	DISCOVERING PatternStatus = "discovering"
	VALIDATED   PatternStatus = "validated"
	DEPRECATED  PatternStatus = "deprecated"
)

// OffloadOutcome tracks the results of an offloading decision
type OffloadOutcome struct {
	DecisionID         string                 `json:"decision_id"`
	ProcessID          string                 `json:"process_id"`
	TargetID           string                 `json:"target_id"`
	ExecutionTime      time.Duration          `json:"execution_time"`
	CompletedOnTime    bool                   `json:"completed_on_time"`
	Success            bool                   `json:"success"`
	ErrorType          string                 `json:"error_type"`
	QueueReduction     float64                `json:"queue_reduction"`
	LoadBalanceBenefit float64                `json:"load_balance_benefit"`
	NetworkCostActual  float64                `json:"network_cost_actual"`
	LatencyActual      time.Duration          `json:"latency_actual"`
	EnergyConsumed     float64                `json:"energy_consumed"`
	LocalWorkDelayed   bool                   `json:"local_work_delayed"`
	NetworkCongestion  bool                   `json:"network_congestion"`
	TargetOverloaded   bool                   `json:"target_overloaded"`
	PolicyViolation    bool                   `json:"policy_violation"`
	ViolationType      []string               `json:"violation_type"`
	CostActual         float64                `json:"cost_actual"`
	CostSavings        float64                `json:"cost_savings"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	MeasurementTime    time.Time              `json:"measurement_time"`
	Reward             float64                `json:"reward"`
	Attribution        map[string]float64     `json:"attribution"`
}