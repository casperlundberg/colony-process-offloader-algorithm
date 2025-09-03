package algorithm

import (
	"fmt"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/learning"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/policy"
)

// Algorithm is the main orchestrator that integrates all components
type Algorithm struct {
	decisionEngine *decision.DecisionEngine
	learner        *learning.AdaptiveLearner
	policyEngine   *policy.PolicyEngine
	
	// Configuration
	config      Config
	version     string
	initialized bool
	
	// Runtime state
	decisionCount       int
	lastPerformanceEval time.Time
}

// Config contains algorithm configuration
type Config struct {
	InitialWeights      decision.AdaptiveWeights `json:"initial_weights"`
	LearningConfig      learning.LearningConfig  `json:"learning_config"`
	SafetyConstraints   policy.SafetyConstraints `json:"safety_constraints"`
	PerformanceTargets  PerformanceTargets       `json:"performance_targets"`
	MonitoringConfig    MonitoringConfig         `json:"monitoring_config"`
}

// PerformanceTargets defines expected performance levels
type PerformanceTargets struct {
	MaxDecisionLatency     time.Duration `json:"max_decision_latency"`
	MinDecisionAccuracy    float64       `json:"min_decision_accuracy"`
	MaxPolicyViolations    int           `json:"max_policy_violations"`
	MinPerformanceGain     float64       `json:"min_performance_gain"`
	ConvergenceTimeout     int           `json:"convergence_timeout"`
}

// MonitoringConfig defines monitoring and alerting configuration
type MonitoringConfig struct {
	EnableMetrics    bool          `json:"enable_metrics"`
	MetricsInterval  time.Duration `json:"metrics_interval"`
	EnableAuditLogs  bool          `json:"enable_audit_logs"`
	EnableAlerts     bool          `json:"enable_alerts"`
}

// NewAlgorithm creates a new algorithm instance
func NewAlgorithm(config Config) (*Algorithm, error) {
	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize decision engine
	decisionEngine := decision.NewDecisionEngine(config.InitialWeights)

	// Initialize learning component
	learner := learning.NewAdaptiveLearner(config.LearningConfig)

	// Initialize policy engine
	policyEngine := policy.NewPolicyEngine()
	policyEngine.SetSafetyConstraints(config.SafetyConstraints)

	// Add default policy rules
	defaultRules := createDefaultPolicyRules()
	for _, rule := range defaultRules {
		if err := policyEngine.AddRule(rule); err != nil {
			return nil, fmt.Errorf("failed to add default policy rule: %w", err)
		}
	}

	return &Algorithm{
		decisionEngine: decisionEngine,
		learner:        learner,
		policyEngine:   policyEngine,
		config:         config,
		version:        "1.0.0",
		initialized:    true,
	}, nil
}

// MakeOffloadDecision makes an intelligent offloading decision
func (a *Algorithm) MakeOffloadDecision(
	process models.Process,
	availableTargets []models.OffloadTarget,
	systemState models.SystemState,
) (decision.OffloadDecision, error) {
	if !a.initialized {
		return decision.OffloadDecision{}, fmt.Errorf("algorithm not initialized")
	}

	startTime := time.Now()
	a.decisionCount++

	// Step 1: Validate inputs
	if err := process.Validate(); err != nil {
		return decision.OffloadDecision{}, fmt.Errorf("invalid process: %w", err)
	}
	if err := systemState.Validate(); err != nil {
		return decision.OffloadDecision{}, fmt.Errorf("invalid system state: %w", err)
	}

	// Step 2: Check safety constraints
	if !a.policyEngine.CheckSafetyConstraints(systemState, a.getCurrentOffloadCount()) {
		return a.createSafetyBlockedDecision(process, "safety constraints not met", startTime), nil
	}

	// Step 3: Filter targets by hard policy constraints
	viableTargets := a.policyEngine.FilterTargetsByPolicy(process, availableTargets)
	if len(viableTargets) == 0 {
		return a.createLocalDecision(process, "no policy-compliant targets", startTime), nil
	}

	// Step 4: Apply discovered patterns to the decision engine
	patterns := a.learner.GetPatterns()
	for _, pattern := range patterns {
		if pattern.ValidationStatus == decision.VALIDATED {
			a.decisionEngine.AddPattern(pattern)
		}
	}

	// Step 5: Make the core decision (uses current adaptive weights)
	coreDecision, err := a.decisionEngine.MakeDecision(process, viableTargets, systemState)
	if err != nil {
		return decision.OffloadDecision{}, fmt.Errorf("decision engine error: %w", err)
	}

	// Step 7: Apply policy score adjustments
	if coreDecision.ShouldOffload && coreDecision.Target != nil {
		policyEval := a.policyEngine.EvaluatePolicy(process, *coreDecision.Target)
		if !policyEval.Allowed {
			// Hard constraint violation - should not happen after filtering
			return a.createLocalDecision(process, "policy violation detected", startTime), nil
		}
		
		// Apply soft policy score adjustment
		coreDecision.Score += policyEval.ScoreAdjustment
		if len(policyEval.ViolatedRules) > 0 {
			for _, rule := range policyEval.ViolatedRules {
				coreDecision.PolicyViolations = append(coreDecision.PolicyViolations, rule.Description)
			}
		}
	}

	// Step 8: Final validation
	if coreDecision.DecisionLatency > a.config.PerformanceTargets.MaxDecisionLatency {
		// Log performance issue but don't fail
		fmt.Printf("Warning: Decision latency %v exceeds target %v\n", 
			coreDecision.DecisionLatency, a.config.PerformanceTargets.MaxDecisionLatency)
	}

	return coreDecision, nil
}

// ProcessOutcome processes the outcome of an offloading decision for learning
func (a *Algorithm) ProcessOutcome(outcome decision.OffloadOutcome) error {
	if !a.initialized {
		return fmt.Errorf("algorithm not initialized")
	}

	// Step 1: Update adaptive weights based on outcome
	currentWeights := a.decisionEngine.GetWeights()
	a.learner.UpdateWeights(&currentWeights, outcome)
	a.decisionEngine.UpdateWeights(currentWeights)

	// Step 2: Pattern discovery - create dummy state and process for pattern learning
	// In a real system, these would be stored from the original decision
	dummyState := models.SystemState{
		QueueDepth: 10,
		Timestamp: time.Now(),
		TimeSlot: time.Now().Hour(),
		DayOfWeek: int(time.Now().Weekday()),
	}
	
	dummyProcess := models.Process{
		ID: outcome.ProcessID,
		EstimatedDuration: 30 * time.Second,
		Priority: 5,
	}

	patterns := a.learner.DiscoverPatterns(dummyState, dummyProcess, outcome)
	
	// Step 3: Update decision engine with new patterns
	for _, pattern := range patterns {
		a.decisionEngine.AddPattern(pattern)
	}

	return nil
}

// GetPerformanceMetrics returns current performance metrics
func (a *Algorithm) GetPerformanceMetrics() PerformanceMetrics {
	learningProgress := a.learner.GetProgress()
	policyStats := a.policyEngine.GetStats()
	
	return PerformanceMetrics{
		DecisionCount:        a.decisionCount,
		LearningProgress:     *learningProgress,
		PolicyStats:         policyStats,
		CurrentWeights:      a.decisionEngine.GetWeights(),
		DiscoveredPatterns:  len(a.learner.GetPatterns()),
		ValidatedPatterns:   learningProgress.PatternsValidated,
		PerformanceGain:     a.learner.GetPerformanceImprovement(),
		IsConverged:         a.learner.IsConverged(),
		Version:             a.version,
	}
}

// GetConfiguration returns the current algorithm configuration
func (a *Algorithm) GetConfiguration() Config {
	return a.config
}

// IsHealthy returns the health status of the algorithm
func (a *Algorithm) IsHealthy() bool {
	if !a.initialized {
		return false
	}
	
	// Check if weights are valid
	weights := a.decisionEngine.GetWeights()
	if weights.Sum() < 0.99 || weights.Sum() > 1.01 {
		return false
	}
	
	// Check policy engine
	stats := a.policyEngine.GetStats()
	if stats.TotalEvaluations > 0 {
		errorRate := float64(stats.HardViolations) / float64(stats.TotalEvaluations)
		if errorRate > 0.05 { // More than 5% hard violations indicates a problem
			return false
		}
	}
	
	return true
}

// Helper methods

func (a *Algorithm) createLocalDecision(process models.Process, reason string, startTime time.Time) decision.OffloadDecision {
	return decision.OffloadDecision{
		ShouldOffload:     false,
		Target:            nil,
		Confidence:        0.9,
		Score:             1.0,
		PolicyViolations:  []string{reason},
		Strategy:          decision.IMMEDIATE,
		ExpectedBenefit:   0.0,
		EstimatedCost:     0.0,
		DecisionTime:      startTime,
		DecisionLatency:   time.Since(startTime),
		AlgorithmVersion:  a.version,
		ScoreComponents:   decision.ScoreBreakdown{WeightsUsed: a.decisionEngine.GetWeights()},
	}
}

func (a *Algorithm) createSafetyBlockedDecision(process models.Process, reason string, startTime time.Time) decision.OffloadDecision {
	return decision.OffloadDecision{
		ShouldOffload:     false,
		Target:            nil,
		Confidence:        1.0,
		Score:             0.0,
		PolicyViolations:  []string{reason},
		Strategy:          decision.IMMEDIATE,
		ExpectedBenefit:   0.0,
		EstimatedCost:     0.0,
		DecisionTime:      startTime,
		DecisionLatency:   time.Since(startTime),
		AlgorithmVersion:  a.version,
		ScoreComponents:   decision.ScoreBreakdown{WeightsUsed: a.decisionEngine.GetWeights()},
	}
}

func (a *Algorithm) getCurrentOffloadCount() int {
	// In a real implementation, this would track active offloads
	return 0
}

// createDefaultPolicyRules creates default safety and compliance rules
func createDefaultPolicyRules() []policy.PolicyRule {
	return []policy.PolicyRule{
		{
			Type:     models.HARD,
			Priority: 1,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Safety-critical processes must stay local
				return !(p.SafetyCritical && t.Type != models.LOCAL)
			},
			Description: "Safety-critical processes must execute locally",
		},
		{
			Type:     models.HARD,
			Priority: 1,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Security level compliance
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Target must meet process security requirements",
		},
		{
			Type:     models.HARD,
			Priority: 1,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Data locality for sensitive data
				if p.LocalityRequired {
					return t.Type == models.LOCAL || t.Type == models.EDGE
				}
				return true
			},
			Description: "Processes requiring data locality must use local or edge targets",
		},
		{
			Type:     models.SOFT,
			Priority: 2,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Prefer reliable targets
				return t.Reliability > 0.8
			},
			Description: "Prefer targets with high reliability",
		},
		{
			Type:     models.SOFT,
			Priority: 3,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Prefer cost-effective targets
				return t.ComputeCost < 0.20
			},
			Description: "Prefer cost-effective targets",
		},
	}
}

// validate validates algorithm configuration
func (c *Config) validate() error {
	// Validate weights sum to approximately 1.0
	sum := c.InitialWeights.Sum()
	if sum < 0.99 || sum > 1.01 {
		return fmt.Errorf("initial weights must sum to 1.0, got %f", sum)
	}
	
	// Validate learning config
	if c.LearningConfig.LearningRate <= 0 || c.LearningConfig.LearningRate > 1 {
		return fmt.Errorf("learning rate must be between 0 and 1")
	}
	
	// Validate performance targets
	if c.PerformanceTargets.MaxDecisionLatency <= 0 {
		return fmt.Errorf("max decision latency must be positive")
	}
	
	return nil
}

// PerformanceMetrics aggregates performance data from all components
type PerformanceMetrics struct {
	DecisionCount       int                        `json:"decision_count"`
	LearningProgress    learning.LearningProgress  `json:"learning_progress"`
	PolicyStats        policy.PolicyStats         `json:"policy_stats"`
	CurrentWeights     decision.AdaptiveWeights   `json:"current_weights"`
	DiscoveredPatterns int                        `json:"discovered_patterns"`
	ValidatedPatterns  int                        `json:"validated_patterns"`
	PerformanceGain    float64                    `json:"performance_gain"`
	IsConverged        bool                       `json:"is_converged"`
	Version            string                     `json:"version"`
}