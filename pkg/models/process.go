package models

import (
	"fmt"
	"strings"
	"time"
)

// Process represents a workload candidate for offloading
type Process struct {
	// Identity
	ID   string `json:"id"`
	Type string `json:"type"`
	Priority int `json:"priority"` // Priority level (1-10, 10=highest)

	// Resource requirements
	CPURequirement    float64 `json:"cpu_requirement"`     // CPU cores needed
	MemoryRequirement int64   `json:"memory_requirement"`  // Memory bytes needed
	DiskRequirement   int64   `json:"disk_requirement"`    // Storage bytes needed
	NetworkRequirement float64 `json:"network_requirement"` // Network bandwidth needed

	// Data characteristics
	InputSize       int64 `json:"input_size"`        // Input data bytes
	OutputSize      int64 `json:"output_size"`       // Expected output data bytes
	DataSensitivity int   `json:"data_sensitivity"`  // Sensitivity level (0-5)

	// Execution characteristics
	EstimatedDuration time.Duration `json:"estimated_duration"` // Expected runtime
	MaxDuration       time.Duration `json:"max_duration"`       // SLA deadline
	RealTime          bool          `json:"real_time"`          // Real-time processing required
	SafetyCritical    bool          `json:"safety_critical"`    // Safety implications

	// Dependencies
	HasDAG       bool     `json:"has_dag"`       // Is part of processing pipeline
	DAG          *DAG     `json:"dag"`           // Pipeline structure if applicable
	Dependencies []string `json:"dependencies"`  // Process dependencies

	// Policy attributes
	LocalityRequired bool `json:"locality_required"` // Must stay in jurisdiction
	SecurityLevel    int  `json:"security_level"`    // Required security level (0-5)

	// State
	SubmissionTime time.Time     `json:"submission_time"` // When submitted
	StartTime      time.Time     `json:"start_time"`      // When started (zero if not started)
	Status         ProcessStatus `json:"status"`          // Current status
}

// DAG represents a Directed Acyclic Graph for pipeline processing
type DAG struct {
	ID     string  `json:"id"`
	Stages []Stage `json:"stages"`
}

// Stage represents a stage in a processing pipeline
type Stage struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	InputSize    int64     `json:"input_size"`
	OutputSize   int64     `json:"output_size"`
	Dependencies []string  `json:"dependencies"`
	Depth        int       `json:"depth"`        // Depth in the DAG
}

// Validate validates the process
func (p Process) Validate() error {
	var errors ValidationErrors

	// Validate identity
	errors.AddIf(p.ID == "", "ID", p.ID, "ID cannot be empty")

	// Validate priority range [1,10]
	errors.AddIf(p.Priority < 1 || p.Priority > 10, "Priority", p.Priority, 
		"Priority must be in range [1,10]")

	// Validate resource requirements are non-negative
	errors.AddIf(p.CPURequirement < 0, "CPURequirement", p.CPURequirement, 
		"CPURequirement must be non-negative")
	errors.AddIf(p.MemoryRequirement < 0, "MemoryRequirement", p.MemoryRequirement, 
		"MemoryRequirement must be non-negative")
	errors.AddIf(p.DiskRequirement < 0, "DiskRequirement", p.DiskRequirement, 
		"DiskRequirement must be non-negative")
	errors.AddIf(p.NetworkRequirement < 0, "NetworkRequirement", p.NetworkRequirement, 
		"NetworkRequirement must be non-negative")

	// Validate data sizes are non-negative
	errors.AddIf(p.InputSize < 0, "InputSize", p.InputSize, 
		"InputSize must be non-negative")
	errors.AddIf(p.OutputSize < 0, "OutputSize", p.OutputSize, 
		"OutputSize must be non-negative")

	// Validate EstimatedDuration > 0
	errors.AddIf(p.EstimatedDuration <= 0, "EstimatedDuration", p.EstimatedDuration, 
		"EstimatedDuration must be > 0 for valid processes")

	// Validate MaxDuration is non-negative
	errors.AddIf(p.MaxDuration < 0, "MaxDuration", p.MaxDuration, 
		"MaxDuration must be non-negative")

	// Validate security and sensitivity levels
	errors.AddIf(p.DataSensitivity < 0 || p.DataSensitivity > 5, "DataSensitivity", p.DataSensitivity, 
		"DataSensitivity must be in range [0,5]")
	errors.AddIf(p.SecurityLevel < 0 || p.SecurityLevel > 5, "SecurityLevel", p.SecurityLevel, 
		"SecurityLevel must be in range [0,5]")

	// Validate DAG consistency
	if p.HasDAG && p.DAG == nil {
		errors.Add("DAG", p.DAG, "Process marked as having DAG but DAG is nil")
	}
	if !p.HasDAG && p.DAG != nil {
		errors.Add("DAG", p.DAG, "Process marked as not having DAG but DAG is provided")
	}

	// Validate dependencies don't include self
	for _, dep := range p.Dependencies {
		if dep == p.ID {
			errors.Add("Dependencies", p.Dependencies, "Process cannot depend on itself (self-dependency)")
			break
		}
	}

	// Check for duplicate dependencies
	depSet := make(map[string]bool)
	for _, dep := range p.Dependencies {
		if depSet[dep] {
			errors.Add("Dependencies", p.Dependencies, "Duplicate dependencies found")
			break
		}
		depSet[dep] = true
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// GetWarnings returns validation warnings (non-fatal issues)
func (p Process) GetWarnings() []string {
	var warnings []string

	// Warn if deadline is before estimated duration
	if p.MaxDuration > 0 && p.MaxDuration < p.EstimatedDuration {
		warnings = append(warnings, 
			fmt.Sprintf("deadline (%v) is before estimated duration (%v)", 
				p.MaxDuration, p.EstimatedDuration))
	}

	// Warn if real-time process has low priority
	if p.RealTime && p.Priority < 7 {
		warnings = append(warnings, 
			"real-time process should have high priority (≥7)")
	}

	// Warn if safety-critical process has low priority
	if p.SafetyCritical && p.Priority < 9 {
		warnings = append(warnings, 
			"safety-critical process should have maximum priority (≥9)")
	}

	return warnings
}

// CanTransitionTo checks if the process can transition to the target status
func (p Process) CanTransitionTo(target ProcessStatus) bool {
	return p.Status.CanTransitionTo(target)
}

// TransitionTo transitions the process to a new status
func (p *Process) TransitionTo(target ProcessStatus) error {
	if !p.CanTransitionTo(target) {
		return fmt.Errorf("cannot transition from %s to %s", p.Status, target)
	}

	p.Status = target

	// Set timestamps for certain transitions
	if target == EXECUTING && p.StartTime.IsZero() {
		p.StartTime = time.Now()
	}

	return nil
}

// GetDataSize returns the total data size (input + output)
func (p Process) GetDataSize() int64 {
	return p.InputSize + p.OutputSize
}

// IsDataIntensive returns true if the process has large data requirements
func (p Process) IsDataIntensive() bool {
	const dataThreshold = 10 * 1024 * 1024 // 10MB
	return p.GetDataSize() > dataThreshold
}

// IsCPUIntensive returns true if the process has high CPU requirements
func (p Process) IsCPUIntensive() bool {
	return p.CPURequirement > 4.0
}

// IsMemoryIntensive returns true if the process has high memory requirements
func (p Process) IsMemoryIntensive() bool {
	const memoryThreshold = 8 * 1024 * 1024 * 1024 // 8GB
	return p.MemoryRequirement > memoryThreshold
}

// GetResourceProfile returns a string describing the resource profile
func (p Process) GetResourceProfile() string {
	var profile []string

	if p.IsCPUIntensive() {
		profile = append(profile, "cpu_intensive")
	}
	if p.IsMemoryIntensive() {
		profile = append(profile, "memory_intensive")
	}
	if p.IsDataIntensive() {
		profile = append(profile, "data_intensive")
	}
	if p.RealTime {
		profile = append(profile, "real_time")
	}
	if p.SafetyCritical {
		profile = append(profile, "safety_critical")
	}

	if len(profile) == 0 {
		return "standard"
	}

	return strings.Join(profile, ",")
}

// GetSLABuffer returns the time buffer between estimated and max duration
func (p Process) GetSLABuffer() time.Duration {
	if p.MaxDuration <= 0 {
		return 0 // No SLA deadline
	}
	
	if p.MaxDuration > p.EstimatedDuration {
		return p.MaxDuration - p.EstimatedDuration
	}
	
	return 0
}

// GetBufferRatio returns the SLA buffer as a ratio of estimated duration
func (p Process) GetBufferRatio() float64 {
	if p.EstimatedDuration <= 0 {
		return 0
	}
	
	buffer := p.GetSLABuffer()
	return float64(buffer) / float64(p.EstimatedDuration)
}

// ToProcess converts a Stage to a Process (for DAG processing)
func (s Stage) ToProcess() Process {
	return Process{
		ID:                s.ID,
		Type:              "dag_stage",
		InputSize:         s.InputSize,
		OutputSize:        s.OutputSize,
		Dependencies:      s.Dependencies,
		EstimatedDuration: 30 * time.Second, // Default for stages
		Priority:          5,                 // Default priority
		Status:            QUEUED,
	}
}

// TopologicalSort returns stages in topological order
func (dag *DAG) TopologicalSort() []Stage {
	if dag == nil {
		return nil
	}

	// Build dependency map
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize in-degrees
	for _, stage := range dag.Stages {
		inDegree[stage.ID] = 0
	}

	// Build adjacency list and calculate in-degrees
	for _, stage := range dag.Stages {
		for _, dep := range stage.Dependencies {
			adjList[dep] = append(adjList[dep], stage.ID)
			inDegree[stage.ID]++
		}
	}

	// Topological sort using Kahn's algorithm
	queue := []string{}
	result := []Stage{}

	// Find nodes with no incoming edges
	for stageID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, stageID)
		}
	}

	// Create stage lookup map
	stageMap := make(map[string]Stage)
	for _, stage := range dag.Stages {
		stageMap[stage.ID] = stage
	}

	// Process queue
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if stage, exists := stageMap[current]; exists {
			result = append(result, stage)
		}

		// Reduce in-degree for neighbors
		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return result
}

// GetDepth calculates the depth of each stage in the DAG
func (dag *DAG) GetDepth() map[string]int {
	if dag == nil {
		return nil
	}

	depths := make(map[string]int)
	visited := make(map[string]bool)

	var calculateDepth func(stageID string) int
	calculateDepth = func(stageID string) int {
		if visited[stageID] {
			return depths[stageID]
		}

		visited[stageID] = true
		maxDepth := 0

		// Find the stage
		var currentStage *Stage
		for _, stage := range dag.Stages {
			if stage.ID == stageID {
				currentStage = &stage
				break
			}
		}

		if currentStage == nil {
			return 0
		}

		// Calculate depth based on dependencies
		for _, dep := range currentStage.Dependencies {
			depDepth := calculateDepth(dep)
			if depDepth > maxDepth {
				maxDepth = depDepth
			}
		}

		depths[stageID] = maxDepth + 1
		return depths[stageID]
	}

	// Calculate depths for all stages
	for _, stage := range dag.Stages {
		calculateDepth(stage.ID)
	}

	return depths
}