package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// DataLocation represents where data is located
type DataLocation string

const (
	DataLocationEdge     DataLocation = "edge"
	DataLocationCloud    DataLocation = "cloud"
	DataLocationHPC      DataLocation = "hpc"
	DataLocationLocal    DataLocation = "local"
	DataLocationFog      DataLocation = "fog"
	DataLocationUnknown  DataLocation = "unknown"
)

// DAGStage represents a stage in a processing pipeline
type DAGStage struct {
	StageID             int          `json:"stage_id"`
	StageName           string       `json:"stage_name"`
	InputLocation       DataLocation `json:"input_location"`
	PreferredLocation   DataLocation `json:"preferred_location"`
	InputSizeGB         float64      `json:"input_size_gb"`
	EstimatedOutputGB   float64      `json:"estimated_output_gb"`
	ComputeRequirement  float64      `json:"compute_requirement"` // CPU cores needed
	MemoryRequirementGB float64      `json:"memory_requirement_gb"`
	UpstreamStages      []int        `json:"upstream_stages"`
	DownstreamStages    []int        `json:"downstream_stages"`
}

// DAGContext provides pipeline context for DAG-aware scheduling  
type DAGContext struct {
	PipelineID       string     `json:"pipeline_id"`
	CurrentStage     int        `json:"current_stage"`
	TotalStages      int        `json:"total_stages"`
	Stages           []DAGStage `json:"stages"`
	SafetyFactor     float64    `json:"safety_factor"`     // Buffer for capacity planning
	PipelineDeadline time.Time  `json:"pipeline_deadline"` // End-to-end deadline
}

// ExtendedMetricsVector represents the complete metrics vector M(t)
type ExtendedMetricsVector struct {
	// Base system state (existing)
	SystemState SystemState `json:"system_state"`
	
	// Data Locality Metrics
	DataLocation      DataLocation `json:"data_location"`        // Where is input data?
	DataSizePendingGB float64      `json:"data_size_pending_gb"` // Size of data waiting (GB)
	TransferCostRate  float64      `json:"transfer_cost_rate"`   // Current $/GB for transfers
	TransferTimeEst   time.Duration `json:"transfer_time_est"`   // Estimated transfer time
	
	// DAG Metrics
	DAGContext DAGContext `json:"dag_context"`
	
	// Additional context
	Timestamp time.Time `json:"timestamp"`
}

// TransferCostMatrix represents costs between different locations
type TransferCostMatrix map[DataLocation]map[DataLocation]float64

// DefaultTransferCosts provides realistic transfer cost estimates ($/GB)
func DefaultTransferCosts() TransferCostMatrix {
	return TransferCostMatrix{
		DataLocationLocal: {
			DataLocationLocal: 0.0,
			DataLocationEdge:  0.001, // Very low cost edge transfer
			DataLocationCloud: 0.02,  // Moderate cloud ingress
			DataLocationHPC:   0.05,  // Higher HPC transfer costs
			DataLocationFog:   0.001, // Low fog transfer
		},
		DataLocationEdge: {
			DataLocationLocal: 0.001,
			DataLocationEdge:  0.0,
			DataLocationCloud: 0.03,  // Edge to cloud
			DataLocationHPC:   0.08,  // Edge to HPC expensive
			DataLocationFog:   0.001,
		},
		DataLocationCloud: {
			DataLocationLocal: 0.05,  // Cloud egress expensive
			DataLocationEdge:  0.03,
			DataLocationCloud: 0.0,
			DataLocationHPC:   0.04,  // Cross-cloud transfer
			DataLocationFog:   0.04,
		},
		DataLocationHPC: {
			DataLocationLocal: 0.08,
			DataLocationEdge:  0.08,
			DataLocationCloud: 0.04,
			DataLocationHPC:   0.0,
			DataLocationFog:   0.09,  // HPC to fog most expensive
		},
		DataLocationFog: {
			DataLocationLocal: 0.001,
			DataLocationEdge:  0.001,
			DataLocationCloud: 0.04,
			DataLocationHPC:   0.09,
			DataLocationFog:   0.0,
		},
	}
}

// GetTransferCost calculates cost to transfer data between locations
func (tcm TransferCostMatrix) GetTransferCost(from, to DataLocation, sizeGB float64) float64 {
	if from == to {
		return 0.0
	}
	
	costPerGB, exists := tcm[from][to]
	if !exists {
		// Default to expensive transfer for unknown combinations
		costPerGB = 0.10
	}
	
	return costPerGB * sizeGB
}

// EstimateTransferTime estimates transfer time based on size and locations
func EstimateTransferTime(from, to DataLocation, sizeGB float64) time.Duration {
	if from == to {
		return 0
	}
	
	// Transfer speeds (GB/second) between locations
	speeds := map[DataLocation]map[DataLocation]float64{
		DataLocationLocal: {
			DataLocationEdge:  10.0,  // High-speed local to edge
			DataLocationCloud: 1.0,   // Internet speeds
			DataLocationHPC:   2.0,   // Dedicated links
			DataLocationFog:   8.0,   // Local network
		},
		DataLocationEdge: {
			DataLocationLocal: 10.0,
			DataLocationCloud: 0.5,   // Limited edge bandwidth
			DataLocationHPC:   1.0,
			DataLocationFog:   5.0,
		},
		DataLocationCloud: {
			DataLocationLocal: 1.0,
			DataLocationEdge:  0.5,
			DataLocationHPC:   3.0,   // High-speed cloud interconnects
			DataLocationFog:   0.3,   // Slow cloud to fog
		},
		DataLocationHPC: {
			DataLocationLocal: 2.0,
			DataLocationEdge:  1.0,
			DataLocationCloud: 3.0,
			DataLocationFog:   0.5,
		},
		DataLocationFog: {
			DataLocationLocal: 8.0,
			DataLocationEdge:  5.0,
			DataLocationCloud: 0.3,
			DataLocationHPC:   0.5,
		},
	}
	
	speed, exists := speeds[from][to]
	if !exists {
		speed = 0.1 // Default very slow speed
	}
	
	transferSeconds := sizeGB / speed
	return time.Duration(transferSeconds * float64(time.Second))
}

// NewExtendedMetricsVector creates a new extended metrics vector
func NewExtendedMetricsVector(
	systemState SystemState,
	dataLocation DataLocation,
	dataSizeGB float64,
	dagContext DAGContext,
) *ExtendedMetricsVector {
	
	// Calculate transfer cost and time based on current data location and process requirements
	transferCosts := DefaultTransferCosts()
	
	// For now, assume we might need to transfer to edge (common case)
	estimatedCost := transferCosts.GetTransferCost(dataLocation, DataLocationEdge, dataSizeGB)
	estimatedTime := EstimateTransferTime(dataLocation, DataLocationEdge, dataSizeGB)
	
	return &ExtendedMetricsVector{
		SystemState:       systemState,
		DataLocation:      dataLocation,
		DataSizePendingGB: dataSizeGB,
		TransferCostRate:  estimatedCost / maxFloat64(dataSizeGB, 0.001), // $/GB rate
		TransferTimeEst:   estimatedTime,
		DAGContext:        dagContext,
		Timestamp:         time.Now(),
	}
}

// GetDownstreamStages returns stages that depend on the current stage
func (dag *DAGContext) GetDownstreamStages(currentStageID int) []DAGStage {
	var downstream []DAGStage
	
	for _, stage := range dag.Stages {
		for _, upstreamID := range stage.UpstreamStages {
			if upstreamID == currentStageID {
				downstream = append(downstream, stage)
				break
			}
		}
	}
	
	return downstream
}

// GetUpstreamStages returns stages that the current stage depends on
func (dag *DAGContext) GetUpstreamStages(currentStageID int) []DAGStage {
	var upstream []DAGStage
	
	// Find the current stage
	var currentStage *DAGStage
	for _, stage := range dag.Stages {
		if stage.StageID == currentStageID {
			currentStage = &stage
			break
		}
	}
	
	if currentStage == nil {
		return upstream
	}
	
	// Find all upstream stages
	for _, upstreamID := range currentStage.UpstreamStages {
		for _, stage := range dag.Stages {
			if stage.StageID == upstreamID {
				upstream = append(upstream, stage)
				break
			}
		}
	}
	
	return upstream
}

// CalculateDAGAwareCapacity implements the DAG-aware capacity planning algorithm
func (dag *DAGContext) CalculateDAGAwareCapacity() float64 {
	// Look at entire pipeline, not just current stage
	stagesAhead := dag.GetDownstreamStages(dag.CurrentStage)
	
	capacityRequirements := []float64{}
	
	for _, stage := range stagesAhead {
		// Consider data movement between stages
		transferOverhead := 0.0
		if stage.InputLocation != stage.PreferredLocation {
			// Estimate transfer time as additional compute requirement
			transferTime := EstimateTransferTime(
				stage.InputLocation, 
				stage.PreferredLocation, 
				stage.InputSizeGB,
			)
			transferOverhead = transferTime.Seconds() / 3600.0 // Convert to "compute hours"
		}
		
		stageCapacity := stage.ComputeRequirement + transferOverhead
		capacityRequirements = append(capacityRequirements, stageCapacity)
	}
	
	// Plan for the most demanding upcoming stage
	maxCapacity := 0.0
	for _, capacity := range capacityRequirements {
		if capacity > maxCapacity {
			maxCapacity = capacity
		}
	}
	
	// Apply safety factor
	return maxCapacity * dag.SafetyFactor
}

// EstimateDownstreamPenalty calculates penalty for suboptimal placement considering downstream stages
func (dag *DAGContext) EstimateDownstreamPenalty(proposedLocation DataLocation) float64 {
	penalty := 0.0
	downstreamStages := dag.GetDownstreamStages(dag.CurrentStage)
	
	transferCosts := DefaultTransferCosts()
	
	for _, stage := range downstreamStages {
		if stage.PreferredLocation != proposedLocation {
			// Calculate transfer penalty based on estimated output size
			transferCost := transferCosts.GetTransferCost(
				proposedLocation,
				stage.PreferredLocation,
				stage.InputSizeGB, // Assuming input size approximates output from previous stage
			)
			penalty += transferCost
		}
	}
	
	return penalty
}

// Validate validates the extended metrics vector
func (emv *ExtendedMetricsVector) Validate() error {
	var errors ValidationErrors
	
	// Validate base system state
	if err := emv.SystemState.Validate(); err != nil {
		errors.Add("SystemState", emv.SystemState, err.Error())
	}
	
	// Validate data metrics
	errors.AddIf(emv.DataSizePendingGB < 0, "DataSizePendingGB", emv.DataSizePendingGB,
		"DataSizePendingGB must be non-negative")
	errors.AddIf(emv.TransferCostRate < 0, "TransferCostRate", emv.TransferCostRate,
		"TransferCostRate must be non-negative")
	
	// Validate DAG context
	errors.AddIf(emv.DAGContext.CurrentStage < 0, "CurrentStage", emv.DAGContext.CurrentStage,
		"CurrentStage must be non-negative")
	errors.AddIf(emv.DAGContext.TotalStages < 1, "TotalStages", emv.DAGContext.TotalStages,
		"TotalStages must be at least 1")
	errors.AddIf(emv.DAGContext.SafetyFactor < 1.0, "SafetyFactor", emv.DAGContext.SafetyFactor,
		"SafetyFactor must be at least 1.0")
	
	// Validate stages
	for i, stage := range emv.DAGContext.Stages {
		errors.AddIf(stage.StageID < 0, fmt.Sprintf("Stage[%d].StageID", i), stage.StageID,
			"StageID must be non-negative")
		errors.AddIf(stage.InputSizeGB < 0, fmt.Sprintf("Stage[%d].InputSizeGB", i), stage.InputSizeGB,
			"InputSizeGB must be non-negative")
		errors.AddIf(stage.ComputeRequirement < 0, fmt.Sprintf("Stage[%d].ComputeRequirement", i), 
			stage.ComputeRequirement, "ComputeRequirement must be non-negative")
	}
	
	if errors.HasErrors() {
		return errors
	}
	
	return nil
}

// Serialize serializes the extended metrics vector to JSON
func (emv *ExtendedMetricsVector) Serialize() string {
	data, err := json.Marshal(emv)
	if err != nil {
		return ""
	}
	return string(data)
}

// Helper function 
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}