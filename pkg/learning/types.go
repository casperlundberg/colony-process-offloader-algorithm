package learning

import (
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
)

// LearningConfig contains configuration for the adaptive learner
type LearningConfig struct {
	WindowSize       int     `json:"window_size"`       // Size of sliding window for outcomes
	LearningRate     float64 `json:"learning_rate"`     // Rate of weight adjustment
	ExplorationRate  float64 `json:"exploration_rate"`  // Rate of exploration vs exploitation
	MinSamples       int     `json:"min_samples"`       // Minimum samples for pattern detection
	ConvergenceThreshold float64 `json:"convergence_threshold"` // Threshold for weight convergence
	MaxPatterns      int     `json:"max_patterns"`      // Maximum patterns to maintain
}

// LearningObjective defines what the algorithm learns to optimize
type LearningObjective struct {
	Name         string         `json:"name"`
	Weight       float64        `json:"weight"`
	MinWeight    float64        `json:"min_weight"`
	MaxWeight    float64        `json:"max_weight"`
	TargetValue  float64        `json:"target_value"`
	CurrentValue float64        `json:"current_value"`
	Trend        TrendDirection `json:"trend"`
}

// TrendDirection represents the trend of a metric
type TrendDirection string

const (
	IMPROVING TrendDirection = "improving"
	DECLINING TrendDirection = "declining"
	STABLE    TrendDirection = "stable"
	UNKNOWN   TrendDirection = "unknown"
)

// LearningProgress tracks the progress of the learning algorithm
type LearningProgress struct {
	DecisionCount      int                      `json:"decision_count"`
	WeightUpdates      int                      `json:"weight_updates"`
	PatternsDiscovered int                      `json:"patterns_discovered"`
	PatternsValidated  int                      `json:"patterns_validated"`
	PerformanceGain    float64                  `json:"performance_gain"`
	WeightHistory      []decision.AdaptiveWeights `json:"weight_history"`
	IsConverged        bool                     `json:"is_converged"`
	ConvergenceTime    int                      `json:"convergence_time"`
}

// PatternRecognizer identifies patterns in decision outcomes
type PatternRecognizer struct {
	patterns       []*decision.DiscoveredPattern
	outcomeHistory []decision.OffloadOutcome
	minSamples     int
	maxPatterns    int
}

// WeightAdapter adapts weights based on feedback
type WeightAdapter struct {
	learningRate    float64
	explorationRate float64
	weightHistory   []decision.AdaptiveWeights
	convergenceTime int
}

// PerformanceBaseline tracks baseline performance for comparison
type PerformanceBaseline struct {
	StaticWeights     decision.AdaptiveWeights `json:"static_weights"`
	BaselineScore     float64                  `json:"baseline_score"`
	CurrentScore      float64                  `json:"current_score"`
	ImprovementRatio  float64                  `json:"improvement_ratio"`
	SampleCount       int                      `json:"sample_count"`
	MeasurementWindow time.Duration            `json:"measurement_window"`
}

// OutcomeWindow maintains a sliding window of recent outcomes
type OutcomeWindow struct {
	outcomes   []decision.OffloadOutcome
	maxSize    int
	totalCount int
}

// Add adds an outcome to the window
func (ow *OutcomeWindow) Add(outcome decision.OffloadOutcome) {
	if len(ow.outcomes) >= ow.maxSize {
		// Remove oldest outcome
		ow.outcomes = ow.outcomes[1:]
	}
	ow.outcomes = append(ow.outcomes, outcome)
	ow.totalCount++
}

// GetOutcomes returns all outcomes in the window
func (ow *OutcomeWindow) GetOutcomes() []decision.OffloadOutcome {
	return ow.outcomes
}

// GetAverageReward returns the average reward in the window
func (ow *OutcomeWindow) GetAverageReward() float64 {
	if len(ow.outcomes) == 0 {
		return 0.0
	}
	
	totalReward := 0.0
	for _, outcome := range ow.outcomes {
		totalReward += outcome.Reward
	}
	return totalReward / float64(len(ow.outcomes))
}

// GetSuccessRate returns the success rate in the window
func (ow *OutcomeWindow) GetSuccessRate() float64 {
	if len(ow.outcomes) == 0 {
		return 0.0
	}
	
	successCount := 0
	for _, outcome := range ow.outcomes {
		if outcome.Success {
			successCount++
		}
	}
	return float64(successCount) / float64(len(ow.outcomes))
}