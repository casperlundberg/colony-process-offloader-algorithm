package models

import "time"

// ExecutorTemplate defines a deployable executor type with its characteristics
type ExecutorTemplate struct {
	// Identity
	TemplateID   string `json:"template_id"`
	ExecutorType string `json:"executor_type"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	
	// Hardware specifications (from ColonyOS format)
	Capabilities ColonyOSCapabilities `json:"capabilities"`
	
	// Deployment characteristics
	DeploymentCost   float64       `json:"deployment_cost"`    // Cost to deploy one instance
	OperationalCost  float64       `json:"operational_cost"`   // Cost per hour to run
	DeploymentTime   time.Duration `json:"deployment_time"`    // Time to spin up
	ShutdownTime     time.Duration `json:"shutdown_time"`      // Time to gracefully stop
	MinInstances     int           `json:"min_instances"`      // Minimum instances to keep running
	MaxInstances     int           `json:"max_instances"`      // Maximum instances allowed
	
	// CAPE optimization weights (specific to this executor type)
	OptimizationWeights ExecutorOptimizationWeights `json:"optimization_weights"`
	
	// Geographic and networking
	PreferredLocations []GeographicLocation `json:"preferred_locations"` // Where this executor performs best
	DataAffinities     []string            `json:"data_affinities"`     // Data sources this executor is close to
	NetworkRequirements NetworkRequirements `json:"network_requirements"`
}

// ExecutorOptimizationWeights defines how this executor type should be optimized
type ExecutorOptimizationWeights struct {
	// Core objectives
	LatencyWeight     float64 `json:"latency_weight"`      // How much latency matters for this executor
	CostWeight        float64 `json:"cost_weight"`         // How much cost optimization matters
	ThroughputWeight  float64 `json:"throughput_weight"`   // How much throughput matters
	EnergyWeight      float64 `json:"energy_weight"`       // Energy efficiency importance
	
	// Data and network
	DataGravityWeight float64 `json:"data_gravity_weight"` // How strongly tied to data location
	NetworkWeight     float64 `json:"network_weight"`      // Network performance importance
	
	// Resource characteristics
	CPUPreference     float64 `json:"cpu_preference"`      // CPU vs other resources priority
	MemoryPreference  float64 `json:"memory_preference"`   // Memory importance
	GPUPreference     float64 `json:"gpu_preference"`      // GPU importance (if applicable)
	StoragePreference float64 `json:"storage_preference"`  // Local storage importance
	
	// Operational
	ScalabilityWeight float64 `json:"scalability_weight"`  // How easily this executor scales
	ReliabilityWeight float64 `json:"reliability_weight"`  // Uptime/reliability importance
}

// GeographicLocation represents a preferred deployment location
type GeographicLocation struct {
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Description string  `json:"description"`
	Region      string  `json:"region"`      // e.g., "us-east-1", "eu-north-1"
	Provider    string  `json:"provider"`    // e.g., "aws", "gcp", "azure", "edge"
}

// NetworkRequirements defines network characteristics needed
type NetworkRequirements struct {
	MinBandwidthMbps float64 `json:"min_bandwidth_mbps"` // Minimum network bandwidth needed
	MaxLatencyMs     float64 `json:"max_latency_ms"`     // Maximum acceptable network latency
	RequiresLowJitter bool    `json:"requires_low_jitter"` // Needs stable network
}

// ExecutorCatalog manages available executor templates
type ExecutorCatalog struct {
	Templates map[string]*ExecutorTemplate `json:"templates"`
	Version   string                       `json:"version"`
	UpdatedAt time.Time                    `json:"updated_at"`
}

// NewExecutorCatalog creates a new executor catalog
func NewExecutorCatalog() *ExecutorCatalog {
	return &ExecutorCatalog{
		Templates: make(map[string]*ExecutorTemplate),
		Version:   "1.0.0",
		UpdatedAt: time.Now(),
	}
}

// AddTemplate adds a new executor template to the catalog
func (ec *ExecutorCatalog) AddTemplate(template *ExecutorTemplate) {
	ec.Templates[template.TemplateID] = template
	ec.UpdatedAt = time.Now()
}

// GetTemplate retrieves an executor template by ID
func (ec *ExecutorCatalog) GetTemplate(templateID string) (*ExecutorTemplate, bool) {
	template, exists := ec.Templates[templateID]
	return template, exists
}

// GetTemplatesByType returns all templates of a specific executor type
func (ec *ExecutorCatalog) GetTemplatesByType(executorType string) []*ExecutorTemplate {
	var templates []*ExecutorTemplate
	for _, template := range ec.Templates {
		if template.ExecutorType == executorType {
			templates = append(templates, template)
		}
	}
	return templates
}

// GetAllTemplates returns all available templates
func (ec *ExecutorCatalog) GetAllTemplates() []*ExecutorTemplate {
	var templates []*ExecutorTemplate
	for _, template := range ec.Templates {
		templates = append(templates, template)
	}
	return templates
}

// ScalingDecision represents a CAPE decision about executor scaling
type ScalingDecision struct {
	// Decision metadata
	DecisionID   string    `json:"decision_id"`
	Timestamp    time.Time `json:"timestamp"`
	DecisionType string    `json:"decision_type"` // "scale_up", "scale_down", "maintain"
	
	// Target executor
	TemplateID       string             `json:"template_id"`
	ExecutorType     string             `json:"executor_type"`
	TargetLocation   GeographicLocation `json:"target_location"`
	InstanceCount    int                `json:"instance_count"`    // How many instances to deploy/remove
	
	// Decision rationale
	TriggerReason    string  `json:"trigger_reason"`    // Why this decision was made
	PriorityDriven   bool    `json:"priority_driven"`   // Was this triggered by high-priority processes
	HighestPriority  int     `json:"highest_priority"`  // Highest priority in queue that triggered this
	PredictedDemand  float64 `json:"predicted_demand"`  // ARIMA prediction that influenced decision
	ConfidenceScore  float64 `json:"confidence_score"`  // How confident CAPE is in this decision
	
	// Cost-benefit analysis
	EstimatedCost    float64       `json:"estimated_cost"`    // Expected cost of this scaling action
	ExpectedBenefit  float64       `json:"expected_benefit"`  // Expected performance improvement
	PaybackTime      time.Duration `json:"payback_time"`      // Expected time to justify the cost
	
	// Algorithms used
	ARIMAPrediction  float64 `json:"arima_prediction"`  // ARIMA forecast value
	EWMASmoothed     float64 `json:"ewma_smoothed"`     // EWMA smoothed value  
	CUSUMAnomaly     bool    `json:"cusum_anomaly"`     // CUSUM detected anomaly
	ThompsonStrategy string  `json:"thompson_strategy"` // Thompson sampling strategy chosen
	
	// Execution details
	DeploymentRegion string `json:"deployment_region"` // Where to deploy
	DataAffinity     string `json:"data_affinity"`     // Data sources this relates to
}

// PriorityWeightedDemand represents demand analysis with priority consideration
type PriorityWeightedDemand struct {
	ExecutorType        string             `json:"executor_type"`
	TotalProcesses      int                `json:"total_processes"`
	WeightedDemand      float64            `json:"weighted_demand"`      // Priority-adjusted demand
	HighPriorityCount   int                `json:"high_priority_count"`  // Processes with priority >= 7
	AverageWaitTime     time.Duration      `json:"average_wait_time"`
	PredictedGrowth     float64            `json:"predicted_growth"`     // ARIMA prediction
	RecommendedAction   string             `json:"recommended_action"`   // scale_up/scale_down/maintain
	UrgencyScore        float64            `json:"urgency_score"`        // How urgent this scaling is (0-1)
	DataLocalityFactors []DataLocalityHint `json:"data_locality_factors"`
}

// DataLocalityHint provides information about data placement affecting scaling
type DataLocalityHint struct {
	DataSource   string             `json:"data_source"`    // Name/ID of data source
	Location     GeographicLocation `json:"location"`       // Where data is located
	SizeGB       float64            `json:"size_gb"`        // Data size
	AccessCount  int                `json:"access_count"`   // How many processes need this data
	MovementCost float64            `json:"movement_cost"`  // Cost to move data (if applicable)
}