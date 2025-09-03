package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// DeploymentType represents the type of deployment scenario
type DeploymentType string

const (
	DeploymentTypeEdge   DeploymentType = "edge"
	DeploymentTypeCloud  DeploymentType = "cloud"
	DeploymentTypeHPC    DeploymentType = "hpc"
	DeploymentTypeHybrid DeploymentType = "hybrid"
	DeploymentTypeFog    DeploymentType = "fog"
)

// OptimizationGoal represents a single optimization objective
type OptimizationGoal struct {
	Metric   string  `json:"metric"`   // "data_movement", "compute_cost", "latency", "throughput"
	Weight   float64 `json:"weight"`   // Weight in objective function
	Minimize bool    `json:"minimize"` // true to minimize, false to maximize
}

// ConstraintType represents the type of constraint
type ConstraintType string

const (
	ConstraintTypeSLADeadline     ConstraintType = "sla_deadline"
	ConstraintTypeBudgetHourly    ConstraintType = "budget_hourly"
	ConstraintTypeDataSovereignty ConstraintType = "data_sovereignty"
	ConstraintTypeSecurity        ConstraintType = "security_level"
	ConstraintTypeLatencyMax      ConstraintType = "latency_max"
	ConstraintTypeMemoryMax       ConstraintType = "memory_max"
	ConstraintTypeCPUMax          ConstraintType = "cpu_max"
)

// DeploymentConstraint represents a hard or soft constraint
type DeploymentConstraint struct {
	Type     ConstraintType `json:"type"`
	Value    interface{}    `json:"value"`
	IsHard   bool           `json:"is_hard"`   // true for hard constraints, false for soft
	Priority int            `json:"priority"`  // Priority for soft constraints (1-10)
}

// DeploymentConfig represents the complete configuration for a deployment scenario
type DeploymentConfig struct {
	DeploymentType      DeploymentType         `json:"deployment_type"`
	OptimizationGoals   []OptimizationGoal     `json:"optimization_goals"`
	Constraints         []DeploymentConstraint `json:"constraints"`
	DataGravityFactor   float64                `json:"data_gravity_factor"`   // How much data location matters [0,1]
	LearningRate        float64                `json:"learning_rate"`         // Adaptation speed
	ExplorationFactor   float64                `json:"exploration_factor"`    // Strategy exploration rate
	AdaptationEnabled   bool                   `json:"adaptation_enabled"`    // Enable weight adaptation
	StrategyEnabled     bool                   `json:"strategy_enabled"`      // Enable strategy selection
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// Strategy represents a high-level decision strategy
type Strategy string

const (
	StrategyDataLocal    Strategy = "data_local"    // Keep compute where data is
	StrategyPerformance  Strategy = "performance"   // Max performance regardless of location
	StrategyCostOptimal  Strategy = "cost_optimal"  // Minimize total cost
	StrategyBalanced     Strategy = "balanced"      // Balance all factors
	StrategyLatencyFirst Strategy = "latency_first" // Minimize latency above all
	StrategyGreenCompute Strategy = "green_compute" // Optimize for energy efficiency
)

// NewDefaultDeploymentConfig creates a default configuration for a given deployment type
func NewDefaultDeploymentConfig(deploymentType DeploymentType) *DeploymentConfig {
	now := time.Now()
	
	config := &DeploymentConfig{
		DeploymentType:    deploymentType,
		LearningRate:      0.1,
		ExplorationFactor: 0.2,
		AdaptationEnabled: true,
		StrategyEnabled:   true,
		Name:              fmt.Sprintf("Default %s Configuration", deploymentType),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	
	switch deploymentType {
	case DeploymentTypeEdge:
		config.DataGravityFactor = 0.3 // Can move compute more freely
		config.OptimizationGoals = []OptimizationGoal{
			{Metric: "latency", Weight: 0.4, Minimize: true},
			{Metric: "data_movement", Weight: 0.2, Minimize: true},
			{Metric: "energy_efficiency", Weight: 0.2, Minimize: false},
			{Metric: "compute_cost", Weight: 0.2, Minimize: true},
		}
		config.Constraints = []DeploymentConstraint{
			{Type: ConstraintTypeSLADeadline, Value: "100ms", IsHard: true, Priority: 10},
			{Type: ConstraintTypeMemoryMax, Value: "4GB", IsHard: false, Priority: 8},
		}
		config.Description = "Edge-optimized for ultra-low latency and resource efficiency"
		
	case DeploymentTypeCloud:
		config.DataGravityFactor = 0.9 // Keep compute near data
		config.OptimizationGoals = []OptimizationGoal{
			{Metric: "compute_cost", Weight: 0.3, Minimize: true},
			{Metric: "throughput", Weight: 0.3, Minimize: false},
			{Metric: "data_movement", Weight: 0.25, Minimize: true},
			{Metric: "latency", Weight: 0.15, Minimize: true},
		}
		config.Constraints = []DeploymentConstraint{
			{Type: ConstraintTypeBudgetHourly, Value: "$100", IsHard: false, Priority: 9},
			{Type: ConstraintTypeSLADeadline, Value: "5s", IsHard: true, Priority: 8},
		}
		config.Description = "Cloud-optimized for scalability and cost efficiency"
		
	case DeploymentTypeHPC:
		config.DataGravityFactor = 0.8 // Moderate data gravity
		config.OptimizationGoals = []OptimizationGoal{
			{Metric: "throughput", Weight: 0.4, Minimize: false},
			{Metric: "compute_cost", Weight: 0.25, Minimize: true},
			{Metric: "data_movement", Weight: 0.2, Minimize: true},
			{Metric: "latency", Weight: 0.15, Minimize: true},
		}
		config.Constraints = []DeploymentConstraint{
			{Type: ConstraintTypeCPUMax, Value: "1000", IsHard: false, Priority: 7},
			{Type: ConstraintTypeMemoryMax, Value: "1TB", IsHard: false, Priority: 6},
		}
		config.Description = "HPC-optimized for maximum computational throughput"
		
	case DeploymentTypeHybrid:
		config.DataGravityFactor = 0.6 // Balanced
		config.OptimizationGoals = []OptimizationGoal{
			{Metric: "data_movement", Weight: 0.25, Minimize: true},
			{Metric: "compute_cost", Weight: 0.25, Minimize: true},
			{Metric: "latency", Weight: 0.25, Minimize: true},
			{Metric: "throughput", Weight: 0.25, Minimize: false},
		}
		config.Constraints = []DeploymentConstraint{
			{Type: ConstraintTypeBudgetHourly, Value: "$200", IsHard: false, Priority: 8},
			{Type: ConstraintTypeSLADeadline, Value: "2s", IsHard: true, Priority: 9},
			{Type: ConstraintTypeDataSovereignty, Value: "prefer_edge", IsHard: false, Priority: 7},
		}
		config.Description = "Hybrid deployment balancing multiple objectives"
		
	case DeploymentTypeFog:
		config.DataGravityFactor = 0.2 // High compute mobility
		config.OptimizationGoals = []OptimizationGoal{
			{Metric: "latency", Weight: 0.5, Minimize: true},
			{Metric: "energy_efficiency", Weight: 0.3, Minimize: false},
			{Metric: "data_movement", Weight: 0.1, Minimize: true},
			{Metric: "compute_cost", Weight: 0.1, Minimize: true},
		}
		config.Constraints = []DeploymentConstraint{
			{Type: ConstraintTypeSLADeadline, Value: "50ms", IsHard: true, Priority: 10},
			{Type: ConstraintTypeMemoryMax, Value: "2GB", IsHard: false, Priority: 8},
		}
		config.Description = "Fog computing optimized for mobile edge scenarios"
	}
	
	return config
}

// ValidateWeights ensures optimization goal weights sum to 1.0
func (dc *DeploymentConfig) ValidateWeights() error {
	sum := 0.0
	for _, goal := range dc.OptimizationGoals {
		sum += goal.Weight
	}
	
	if sum < 0.999 || sum > 1.001 { // Allow small floating point error
		return fmt.Errorf("optimization goal weights sum to %.6f, must sum to 1.0", sum)
	}
	
	return nil
}

// NormalizeWeights adjusts weights to sum to 1.0
func (dc *DeploymentConfig) NormalizeWeights() {
	sum := 0.0
	for _, goal := range dc.OptimizationGoals {
		sum += goal.Weight
	}
	
	if sum <= 0.0 {
		// Equal weights if sum is zero
		equalWeight := 1.0 / float64(len(dc.OptimizationGoals))
		for i := range dc.OptimizationGoals {
			dc.OptimizationGoals[i].Weight = equalWeight
		}
	} else {
		// Normalize to sum to 1.0
		for i := range dc.OptimizationGoals {
			dc.OptimizationGoals[i].Weight /= sum
		}
	}
	
	dc.UpdatedAt = time.Now()
}

// GetGoalWeight returns the weight for a specific metric
func (dc *DeploymentConfig) GetGoalWeight(metric string) float64 {
	for _, goal := range dc.OptimizationGoals {
		if goal.Metric == metric {
			return goal.Weight
		}
	}
	return 0.0
}

// IsMetricMinimized returns true if the metric should be minimized
func (dc *DeploymentConfig) IsMetricMinimized(metric string) bool {
	for _, goal := range dc.OptimizationGoals {
		if goal.Metric == metric {
			return goal.Minimize
		}
	}
	return true // Default to minimize
}

// GetHardConstraints returns only hard constraints
func (dc *DeploymentConfig) GetHardConstraints() []DeploymentConstraint {
	var hard []DeploymentConstraint
	for _, constraint := range dc.Constraints {
		if constraint.IsHard {
			hard = append(hard, constraint)
		}
	}
	return hard
}

// GetSoftConstraints returns only soft constraints
func (dc *DeploymentConfig) GetSoftConstraints() []DeploymentConstraint {
	var soft []DeploymentConstraint
	for _, constraint := range dc.Constraints {
		if !constraint.IsHard {
			soft = append(soft, constraint)
		}
	}
	return soft
}

// UpdateStrategy adjusts configuration based on chosen strategy
func (dc *DeploymentConfig) UpdateStrategy(strategy Strategy) {
	dc.UpdatedAt = time.Now()
	
	switch strategy {
	case StrategyDataLocal:
		dc.DataGravityFactor = 0.95
		dc.SetGoalWeight("data_movement", 0.5)
		
	case StrategyPerformance:
		dc.DataGravityFactor = 0.3
		dc.SetGoalWeight("throughput", 0.4)
		dc.SetGoalWeight("latency", 0.4)
		
	case StrategyCostOptimal:
		dc.SetGoalWeight("compute_cost", 0.6)
		dc.SetGoalWeight("data_movement", 0.3)
		
	case StrategyLatencyFirst:
		dc.SetGoalWeight("latency", 0.7)
		dc.DataGravityFactor = 0.2
		
	case StrategyGreenCompute:
		dc.SetGoalWeight("energy_efficiency", 0.5)
		dc.SetGoalWeight("compute_cost", 0.3)
		
	case StrategyBalanced:
		// Reset to balanced weights
		numGoals := len(dc.OptimizationGoals)
		if numGoals > 0 {
			equalWeight := 1.0 / float64(numGoals)
			for i := range dc.OptimizationGoals {
				dc.OptimizationGoals[i].Weight = equalWeight
			}
		}
		dc.DataGravityFactor = 0.6
	}
}

// SetGoalWeight sets the weight for a specific goal, normalizing all weights afterward
func (dc *DeploymentConfig) SetGoalWeight(metric string, weight float64) {
	for i := range dc.OptimizationGoals {
		if dc.OptimizationGoals[i].Metric == metric {
			dc.OptimizationGoals[i].Weight = weight
			break
		}
	}
	dc.NormalizeWeights()
}

// Clone creates a deep copy of the deployment configuration
func (dc *DeploymentConfig) Clone() *DeploymentConfig {
	clone := &DeploymentConfig{
		DeploymentType:    dc.DeploymentType,
		DataGravityFactor: dc.DataGravityFactor,
		LearningRate:      dc.LearningRate,
		ExplorationFactor: dc.ExplorationFactor,
		AdaptationEnabled: dc.AdaptationEnabled,
		StrategyEnabled:   dc.StrategyEnabled,
		Name:              dc.Name,
		Description:       dc.Description,
		CreatedAt:         dc.CreatedAt,
		UpdatedAt:         time.Now(),
	}
	
	// Deep copy goals
	clone.OptimizationGoals = make([]OptimizationGoal, len(dc.OptimizationGoals))
	copy(clone.OptimizationGoals, dc.OptimizationGoals)
	
	// Deep copy constraints
	clone.Constraints = make([]DeploymentConstraint, len(dc.Constraints))
	copy(clone.Constraints, dc.Constraints)
	
	return clone
}

// Validate validates the deployment configuration
func (dc *DeploymentConfig) Validate() error {
	var errors ValidationErrors
	
	// Validate deployment type
	validTypes := []DeploymentType{
		DeploymentTypeEdge, DeploymentTypeCloud, DeploymentTypeHPC, 
		DeploymentTypeHybrid, DeploymentTypeFog,
	}
	
	valid := false
	for _, validType := range validTypes {
		if dc.DeploymentType == validType {
			valid = true
			break
		}
	}
	errors.AddIf(!valid, "DeploymentType", dc.DeploymentType, "Invalid deployment type")
	
	// Validate data gravity factor
	errors.AddIf(dc.DataGravityFactor < 0.0 || dc.DataGravityFactor > 1.0, 
		"DataGravityFactor", dc.DataGravityFactor, "DataGravityFactor must be in range [0.0, 1.0]")
	
	// Validate learning parameters
	errors.AddIf(dc.LearningRate < 0.0 || dc.LearningRate > 1.0,
		"LearningRate", dc.LearningRate, "LearningRate must be in range [0.0, 1.0]")
	errors.AddIf(dc.ExplorationFactor < 0.0 || dc.ExplorationFactor > 1.0,
		"ExplorationFactor", dc.ExplorationFactor, "ExplorationFactor must be in range [0.0, 1.0]")
	
	// Validate optimization goals
	if len(dc.OptimizationGoals) == 0 {
		errors.Add("OptimizationGoals", dc.OptimizationGoals, "At least one optimization goal is required")
	} else {
		for i, goal := range dc.OptimizationGoals {
			errors.AddIf(goal.Weight < 0.0, fmt.Sprintf("OptimizationGoals[%d].Weight", i), 
				goal.Weight, "Goal weight must be non-negative")
			errors.AddIf(goal.Metric == "", fmt.Sprintf("OptimizationGoals[%d].Metric", i), 
				goal.Metric, "Goal metric cannot be empty")
		}
		
		// Validate weights sum to 1.0
		if err := dc.ValidateWeights(); err != nil {
			errors.Add("OptimizationGoals.Weights", "sum", err.Error())
		}
	}
	
	// Validate constraints
	for i, constraint := range dc.Constraints {
		errors.AddIf(constraint.Priority < 1 || constraint.Priority > 10,
			fmt.Sprintf("Constraints[%d].Priority", i), constraint.Priority,
			"Constraint priority must be in range [1, 10]")
	}
	
	if errors.HasErrors() {
		return errors
	}
	
	return nil
}

// Serialize serializes the deployment configuration to JSON
func (dc *DeploymentConfig) Serialize() string {
	data, err := json.Marshal(dc)
	if err != nil {
		return ""
	}
	return string(data)
}

// DeserializeDeploymentConfig deserializes a deployment configuration from JSON
func DeserializeDeploymentConfig(data string) (*DeploymentConfig, error) {
	var config DeploymentConfig
	err := json.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize DeploymentConfig: %w", err)
	}
	return &config, nil
}