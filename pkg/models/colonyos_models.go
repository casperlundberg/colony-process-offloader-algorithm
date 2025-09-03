package models

import (
	"time"
)

// ColonyOSExecutor represents a ColonyOS executor with full capabilities specification
type ColonyOSExecutor struct {
	// Core executor identity
	ExecutorName string `json:"executorname"`
	ExecutorType string `json:"executortype"`
	ExecutorID   string `json:"executorid,omitempty"`
	ColonyName   string `json:"colonyname,omitempty"`
	
	// Physical location information
	Location ColonyOSLocation `json:"location"`
	
	// Hardware and software capabilities
	Capabilities ColonyOSCapabilities `json:"capabilities"`
	
	// Runtime status (for CAPE algorithm use)
	Status       ExecutorStatus       `json:"status,omitempty"`
	LastSeen     time.Time           `json:"last_seen,omitempty"`
	Utilization  DetailedUtilization `json:"utilization,omitempty"`
}

// ColonyOSLocation represents physical location of an executor
type ColonyOSLocation struct {
	Longitude   float64 `json:"long"`
	Latitude    float64 `json:"lat"`
	Description string  `json:"desc"`
}

// ColonyOSCapabilities represents hardware and software capabilities
type ColonyOSCapabilities struct {
	Hardware ColonyOSHardware `json:"hardware"`
	Software ColonyOSSoftware `json:"software"`
}

// ColonyOSHardware represents hardware specifications
type ColonyOSHardware struct {
	Model   string        `json:"model"`
	CPU     string        `json:"cpu"`     // e.g., "4000m"
	Memory  string        `json:"mem"`     // e.g., "16Gi"
	Storage string        `json:"storage"` // e.g., "100Ti"
	GPU     *ColonyOSGPU  `json:"gpu,omitempty"`
}

// ColonyOSGPU represents GPU specifications
type ColonyOSGPU struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ColonyOSSoftware represents software specifications
type ColonyOSSoftware struct {
	Name    string `json:"name"`    // e.g., "colonyos/ml:latest"
	Type    string `json:"type"`    // e.g., "k8s"
	Version string `json:"version"`
}

// ExecutorStatus represents current executor status
type ExecutorStatus string

const (
	ExecutorStatusOnline  ExecutorStatus = "online"
	ExecutorStatusOffline ExecutorStatus = "offline"
	ExecutorStatusBusy    ExecutorStatus = "busy"
	ExecutorStatusIdle    ExecutorStatus = "idle"
)

// ColonyOSProcessSpec represents a ColonyOS process specification
type ColonyOSProcessSpec struct {
	// Core specification
	Conditions ColonyOSConditions     `json:"conditions"`
	FuncName   string                 `json:"funcname"`
	Args       []string               `json:"args,omitempty"`
	Kwargs     map[string]interface{} `json:"kwargs,omitempty"`
	Env        map[string]string      `json:"env,omitempty"`
	
	// Process metadata
	Label       string `json:"label,omitempty"`
	NodeName    string `json:"nodename,omitempty"` // For workflows
	
	// Timing constraints
	MaxWaitTime int `json:"maxwaittime,omitempty"` // seconds
	MaxExecTime int `json:"maxexectime,omitempty"` // seconds
	MaxRetries  int `json:"maxretries,omitempty"`
	
	// CAPE-specific extensions (not part of standard ColonyOS)
	Priority          int               `json:"priority,omitempty"`
	EstimatedDuration time.Duration     `json:"estimated_duration,omitempty"`
	DataRequirements  *DataRequirements `json:"data_requirements,omitempty"`
	ResourceHints     *ResourceHints    `json:"resource_hints,omitempty"`
}

// ColonyOSConditions specifies executor selection conditions
type ColonyOSConditions struct {
	ExecutorType  string   `json:"executortype"`
	ExecutorNames []string `json:"executornames,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"` // For workflow dependencies
	
	// CAPE-specific conditions
	MinCPU         string   `json:"min_cpu,omitempty"`
	MinMemory      string   `json:"min_memory,omitempty"`
	RequiredGPU    bool     `json:"required_gpu,omitempty"`
	LocationHints  []string `json:"location_hints,omitempty"`
	SecurityLevel  int      `json:"security_level,omitempty"`
}

// ColonyOSProcess represents an active or completed process
type ColonyOSProcess struct {
	ProcessID  string              `json:"processid"`
	Spec       ColonyOSProcessSpec `json:"spec"`
	
	// Execution details
	AssignedExecutor string                 `json:"assigned_executor,omitempty"`
	State            ProcessState           `json:"state"`
	Output           []interface{}          `json:"output,omitempty"`
	Errors           []string               `json:"errors,omitempty"`
	
	// Timing information
	SubmissionTime time.Time  `json:"submission_time"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	
	// Resource usage (for CAPE learning)
	ActualResources *ResourceUsage `json:"actual_resources,omitempty"`
}

// ProcessState represents the state of a process
type ProcessState string

const (
	ProcessStateWaiting    ProcessState = "waiting"
	ProcessStateRunning    ProcessState = "running"
	ProcessStateSuccessful ProcessState = "successful"
	ProcessStateFailed     ProcessState = "failed"
)

// ColonyOSWorkflow represents a workflow with dependencies
type ColonyOSWorkflow []ColonyOSWorkflowNode

// ColonyOSWorkflowNode represents a node in a workflow
type ColonyOSWorkflowNode struct {
	NodeName   string             `json:"nodename"`
	FuncName   string             `json:"funcname"`
	Args       []string           `json:"args,omitempty"`
	Conditions ColonyOSConditions `json:"conditions"`
}

// DataRequirements represents data locality requirements for CAPE
type DataRequirements struct {
	InputDataLocation  DataLocation `json:"input_data_location"`
	OutputDataLocation DataLocation `json:"output_data_location"`
	DataSizeGB         float64      `json:"data_size_gb"`
	DataSensitive      bool         `json:"data_sensitive"`
}

// ResourceHints provides hints for CAPE algorithm
type ResourceHints struct {
	PreferredExecutorType string  `json:"preferred_executor_type,omitempty"`
	CPUIntensive          bool    `json:"cpu_intensive"`
	MemoryIntensive       bool    `json:"memory_intensive"`
	GPURequired           bool    `json:"gpu_required"`
	NetworkIntensive      bool    `json:"network_intensive"`
	LatencySensitive      bool    `json:"latency_sensitive"`
	CostSensitive         bool    `json:"cost_sensitive"`
}

// ResourceUsage represents actual resource consumption
type ResourceUsage struct {
	CPUUsed       float64       `json:"cpu_used"`
	MemoryUsedGB  float64       `json:"memory_used_gb"`
	DurationUsed  time.Duration `json:"duration_used"`
	NetworkUsedGB float64       `json:"network_used_gb"`
	StorageUsedGB float64       `json:"storage_used_gb"`
	GPUUsed       bool          `json:"gpu_used"`
}

// ColonyOSSystemState represents system-wide state for CAPE
type ColonyOSSystemState struct {
	ColonyName      string                       `json:"colony_name"`
	Timestamp       time.Time                    `json:"timestamp"`
	
	// Queue metrics
	PendingProcesses   int                      `json:"pending_processes"`
	RunningProcesses   int                      `json:"running_processes"`
	CompletedProcesses int                      `json:"completed_processes"`
	FailedProcesses    int                      `json:"failed_processes"`
	
	// Executor metrics
	ActiveExecutors    []ColonyOSExecutor       `json:"active_executors"`
	ExecutorsByType    map[string]int           `json:"executors_by_type"`
	
	// Resource metrics
	TotalCapacity      ResourceCapacity         `json:"total_capacity"`
	AvailableCapacity  ResourceCapacity         `json:"available_capacity"`
	
	// Performance metrics
	AvgProcessLatency  time.Duration            `json:"avg_process_latency"`
	ProcessThroughput  float64                  `json:"process_throughput"` // processes/sec
	SuccessRate        float64                  `json:"success_rate"`
}

// ResourceCapacity represents total system capacity
type ResourceCapacity struct {
	TotalCPU     float64 `json:"total_cpu"`
	TotalMemoryGB float64 `json:"total_memory_gb"`
	TotalGPUs    int     `json:"total_gpus"`
	TotalStorage float64 `json:"total_storage_gb"`
}

// Helper functions for CAPE integration

// ToOffloadTarget converts ColonyOSExecutor to legacy OffloadTarget for CAPE compatibility
func (e ColonyOSExecutor) ToOffloadTarget() OffloadTarget {
	// Parse CPU (e.g., "4000m" -> 4.0 cores)
	cpuCapacity := parseCPUString(e.Capabilities.Hardware.CPU)
	
	// Parse Memory (e.g., "16Gi" -> 16 GB)
	memoryBytes := parseMemoryString(e.Capabilities.Hardware.Memory)
	
	// Determine executor type mapping
	executorType := mapColonyOSTypeToLegacy(e.ExecutorType)
	
	return OffloadTarget{
		ID:                e.ExecutorName,
		Type:              executorType,
		Location:          e.Location.Description,
		TotalCapacity:     cpuCapacity,
		AvailableCapacity: cpuCapacity * (1.0 - e.Utilization.ComputeUsage),
		MemoryTotal:       memoryBytes,
		MemoryAvailable:   int64(float64(memoryBytes) * (1.0 - e.Utilization.MemoryUsage)),
		NetworkLatency:    time.Duration(calculateLatencyFromLocation(e.Location)),
		ProcessingSpeed:   calculateProcessingSpeed(e.Capabilities.Hardware),
		Reliability:       0.95, // Default reliability
		ComputeCost:       calculateComputeCost(e.ExecutorType),
		SecurityLevel:     3, // Default security level
		Utilization:       e.Utilization,
	}
}

// ToProcess converts ColonyOSProcess to legacy Process for CAPE compatibility
func (p ColonyOSProcess) ToProcess() Process {
	process := Process{
		ID:                p.ProcessID,
		Type:              p.Spec.FuncName,
		Status:            mapColonyOSStateToLegacy(p.State),
		Priority:          p.Spec.Priority,
		EstimatedDuration: p.Spec.EstimatedDuration,
	}
	
	// Extract resource requirements from hints
	if p.Spec.ResourceHints != nil {
		process.CPURequirement = parseCPUFromHints(*p.Spec.ResourceHints)
		process.MemoryRequirement = parseMemoryFromHints(*p.Spec.ResourceHints)
		process.RealTime = p.Spec.ResourceHints.LatencySensitive
		process.SecurityLevel = getSecurityLevelFromConditions(p.Spec.Conditions)
		process.LocalityRequired = p.Spec.DataRequirements != nil && p.Spec.DataRequirements.DataSensitive
	}
	
	return process
}

// Helper functions (simplified implementations)
func parseCPUString(cpu string) float64 {
	// Simple parser for "4000m" -> 4.0
	if len(cpu) > 1 && cpu[len(cpu)-1] == 'm' {
		// Parse millicores
		// This is a simplified version - real implementation would use strconv
		return 4.0 // Placeholder
	}
	return 1.0 // Default
}

func parseMemoryString(mem string) int64 {
	// Simple parser for "16Gi" -> bytes
	// This is a simplified version - real implementation would parse units
	return 16 * 1024 * 1024 * 1024 // 16GB in bytes
}

func mapColonyOSTypeToLegacy(executorType string) TargetType {
	switch executorType {
	case "ml", "hpc":
		return HPC_CLUSTER
	case "edge":
		return EDGE
	case "cloud":
		return PUBLIC_CLOUD
	default:
		return LOCAL
	}
}

func mapColonyOSStateToLegacy(state ProcessState) ProcessStatus {
	switch state {
	case ProcessStateWaiting:
		return QUEUED
	case ProcessStateRunning:
		return EXECUTING
	case ProcessStateSuccessful:
		return COMPLETED
	case ProcessStateFailed:
		return FAILED
	default:
		return QUEUED
	}
}

func calculateLatencyFromLocation(location ColonyOSLocation) time.Duration {
	// Simplified latency calculation based on location
	// Real implementation would use geographic distance
	return 50 * time.Millisecond // Default latency
}

func calculateProcessingSpeed(hardware ColonyOSHardware) float64 {
	// Simplified processing speed calculation
	// Real implementation would consider CPU model, GPU, etc.
	return 1.5 // Default processing speed
}

func calculateComputeCost(executorType string) float64 {
	// Simplified cost calculation based on executor type
	switch executorType {
	case "ml", "hpc":
		return 0.20
	case "edge":
		return 0.05
	case "cloud":
		return 0.10
	default:
		return 0.0
	}
}

func parseCPUFromHints(hints ResourceHints) float64 {
	if hints.CPUIntensive {
		return 4.0
	}
	return 1.0
}

func parseMemoryFromHints(hints ResourceHints) int64 {
	if hints.MemoryIntensive {
		return 8 * 1024 * 1024 * 1024 // 8GB
	}
	return 2 * 1024 * 1024 * 1024 // 2GB
}

func getSecurityLevelFromConditions(conditions ColonyOSConditions) int {
	return conditions.SecurityLevel
}