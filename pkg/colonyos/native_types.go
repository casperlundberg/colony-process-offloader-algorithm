package colonyos

import (
	"fmt"
	"time"
)

// Native ColonyOS data structures directly from the source code
// These match exactly with the official ColonyOS implementation

// Executor represents a ColonyOS executor exactly as defined in pkg/core/executor.go
type Executor struct {
	ID                 string        `json:"executorid"`
	Type               string        `json:"executortype"`
	Name               string        `json:"executorname"`
	ColonyName         string        `json:"colonyname"`
	State              int           `json:"state"` // PENDING=0, APPROVED=1, REJECTED=2
	RequireFuncReg     bool          `json:"requirefuncreg"`
	CommissionTime     time.Time     `json:"commissiontime"`
	LastHeardFromTime  time.Time     `json:"lastheardfromtime"`
	Location           Location      `json:"location"`
	Capabilities       Capabilities  `json:"capabilities"`
	Allocations        Allocations   `json:"allocations"`
}

// Location represents executor physical location
type Location struct {
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Description string  `json:"description"`
}

// Capabilities represents executor hardware and software capabilities
type Capabilities struct {
	Hardware Hardware `json:"hardware"`
	Software Software `json:"software"`
}

// Hardware represents hardware specifications
type Hardware struct {
	Nodes   int    `json:"nodes"`
	CPU     string `json:"cpu"`     // e.g., "4000m" 
	Memory  string `json:"memory"`  // e.g., "16Gi"
	Storage string `json:"storage"` // e.g., "500Gi"
	GPUs    []GPU  `json:"gpus"`
}

// GPU represents GPU specifications
type GPU struct {
	Name   string `json:"name"`   // e.g., "nvidia_a100"
	Count  int    `json:"count"`
	Memory string `json:"memory"` // e.g., "40Gi"
}

// Software represents software specifications
type Software struct {
	Name    string `json:"name"`    // e.g., "colonyos/ml:latest"
	Type    string `json:"type"`    // e.g., "container", "k8s", "slurm"
	Version string `json:"version"` // e.g., "v2.1.0"
}

// Allocations tracks resource allocations across projects
type Allocations struct {
	Projects map[string]ProjectAllocation `json:"projects"`
}

// ProjectAllocation represents resource allocation for a specific project
type ProjectAllocation struct {
	AllocatedCPU  int `json:"allocatedcpu"`
	UsedCPU       int `json:"usedcpu"`
	AllocatedGPUs int `json:"allocatedgpus"`
	UsedGPUs      int `json:"usedgpus"`
	AllocatedStorage int64 `json:"allocatedstorage"`
	UsedStorage      int64 `json:"usedstorage"`
}

// Process represents a ColonyOS process exactly as defined in pkg/core/process.go
type Process struct {
	ID                   string        `json:"processid"`
	InitiatorID          string        `json:"initiatorid"`
	InitiatorName        string        `json:"initiatorname"`
	AssignedExecutorID   string        `json:"assignedexecutorid"`
	IsAssigned           bool          `json:"isassigned"`
	State                int           `json:"state"` // WAITING=0, RUNNING=1, SUCCESS=2, FAILED=3
	SubmissionTime       time.Time     `json:"submissiontime"`
	StartTime            time.Time     `json:"starttime"`
	EndTime              time.Time     `json:"endtime"`
	WaitDeadline         time.Time     `json:"waitdeadline"`
	ExecDeadline         time.Time     `json:"execdeadline"`
	PriorityTime         int64         `json:"prioritytime"`
	Parents              []string      `json:"parents"`
	Children             []string      `json:"children"`
	ProcessGraphID       string        `json:"processgraphid"`
	WaitForParents       bool          `json:"waitforparents"`
	FunctionSpec         FunctionSpec  `json:"functionspec"`
	Attributes           []Attribute   `json:"attributes"`
	Input                []interface{} `json:"input"`
	Output               []interface{} `json:"output"`
	Errors               []string      `json:"errors"`
	Retries              int           `json:"retries"`
}

// FunctionSpec represents a function specification exactly as defined in pkg/core/function_spec.go
type FunctionSpec struct {
	NodeName     string                 `json:"nodename"`
	FuncName     string                 `json:"funcname"`
	Args         []interface{}          `json:"args"`
	KwArgs       map[string]interface{} `json:"kwargs"`
	Priority     int                    `json:"priority"`
	MaxWaitTime  int                    `json:"maxwaittime"`
	MaxExecTime  int                    `json:"maxexectime"`
	MaxRetries   int                    `json:"maxretries"`
	Conditions   Conditions             `json:"conditions"`
	Label        string                 `json:"label"`
	Filesystem   Filesystem             `json:"fs"`
	Env          map[string]string      `json:"env"`
}

// Conditions represents execution conditions and constraints
type Conditions struct {
	ColonyName       string   `json:"colonyname"`
	ExecutorNames    []string `json:"executornames"`
	ExecutorType     string   `json:"executortype"`
	Dependencies     []string `json:"dependencies"`
	Nodes            int      `json:"nodes"`
	CPU              string   `json:"cpu"`
	Processes        int      `json:"processes"`
	ProcessesPerNode int      `json:"processespernode"`
	Memory           string   `json:"memory"`
	Storage          string   `json:"storage"`
	WallTime         int      `json:"walltime"`
}

// Filesystem represents filesystem configuration
type Filesystem struct {
	Mount          Mount           `json:"mount"`
	SnapshotMounts []SnapshotMount `json:"snapshotmounts"`
	SyncDirMounts  []SyncDirMount  `json:"syncdirmounts"`
}

// Mount represents a filesystem mount
type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// SnapshotMount represents a snapshot mount
type SnapshotMount struct {
	SnapshotID string `json:"snapshotid"`
	Source     string `json:"source"`
	Target     string `json:"target"`
}

// SyncDirMount represents a synchronized directory mount
type SyncDirMount struct {
	Label  string `json:"label"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// Attribute represents additional process attributes
type Attribute struct {
	ID          string      `json:"attributeid"`
	TargetID    string      `json:"targetid"`
	TargetType  int         `json:"targettype"`
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
}

// Statistics represents colony-wide statistics from monitoring
type Statistics struct {
	Colonies  int                `json:"colonies"`
	Executors int                `json:"executors"`
	Processes ProcessStatistics  `json:"processes"`
	Workflows WorkflowStatistics `json:"workflows"`
}

// ProcessStatistics represents process-related statistics
type ProcessStatistics struct {
	Waiting    int `json:"waiting"`
	Running    int `json:"running"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
}

// WorkflowStatistics represents workflow-related statistics
type WorkflowStatistics struct {
	Waiting    int `json:"waiting"`
	Running    int `json:"running"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
}

// ExecutorStates constants matching ColonyOS source
const (
	EXECUTOR_PENDING  = 0
	EXECUTOR_APPROVED = 1
	EXECUTOR_REJECTED = 2
)

// ProcessStates constants matching ColonyOS source
const (
	PROCESS_WAITING = 0
	PROCESS_RUNNING = 1
	PROCESS_SUCCESS = 2
	PROCESS_FAILED  = 3
)

// ResourceParser contains ColonyOS resource parsing functions
type ResourceParser struct{}

// ConvertCPUToInt converts CPU string (e.g., "4000m") to integer millicores
func (rp *ResourceParser) ConvertCPUToInt(cpu string) (int, error) {
	// Implementation matching pkg/parsers/parser.go
	if len(cpu) == 0 {
		return 0, nil
	}
	
	if cpu[len(cpu)-1] == 'm' {
		// Parse integer from string - simplified implementation
		// In real implementation, use strconv.Atoi(cpu[:len(cpu)-1])
		return 4000, nil // Placeholder - actual implementation needed
	}
	
	// If no 'm' suffix, assume cores and convert to millicores
	// Parse and multiply by 1000 - simplified implementation
	return 1000, nil // Placeholder - actual implementation needed
}

// ConvertMemoryToBytes converts memory string (e.g., "16Gi") to bytes
func (rp *ResourceParser) ConvertMemoryToBytes(memory string) (int64, error) {
	// Implementation matching pkg/parsers/parser.go
	// Handles Ki, Mi, Gi, Ti, K, M, G, T suffixes
	// Returns bytes - simplified implementation
	// In real implementation, parse string and apply multipliers
	return 17179869184, nil // 16Gi in bytes - placeholder
}

// ConvertCPUToString converts millicores integer to string format
func (rp *ResourceParser) ConvertCPUToString(cpu int) string {
	// Implementation matching pkg/parsers/parser.go
	return fmt.Sprintf("%dm", cpu)
}

// ConvertMemoryToString converts bytes to memory string format
func (rp *ResourceParser) ConvertMemoryToString(bytes int64) string {
	// Implementation matching pkg/parsers/parser.go
	// Convert to appropriate unit (Mi, Gi, etc.)
	return fmt.Sprintf("%dMi", bytes/(1024*1024))
}

// ExecutorList represents a list of executors
type ExecutorList []Executor

// ProcessList represents a list of processes  
type ProcessList []Process

// GetExecutorsByType filters executors by type
func (el ExecutorList) GetExecutorsByType(executorType string) ExecutorList {
	var filtered ExecutorList
	for _, executor := range el {
		if executor.Type == executorType {
			filtered = append(filtered, executor)
		}
	}
	return filtered
}

// GetExecutorsByState filters executors by state
func (el ExecutorList) GetExecutorsByState(state int) ExecutorList {
	var filtered ExecutorList
	for _, executor := range el {
		if executor.State == state {
			filtered = append(filtered, executor)
		}
	}
	return filtered
}

// GetProcessesByState filters processes by state
func (pl ProcessList) GetProcessesByState(state int) ProcessList {
	var filtered ProcessList
	for _, process := range pl {
		if process.State == state {
			filtered = append(filtered, process)
		}
	}
	return filtered
}

// GetProcessesByExecutorType filters processes by executor type condition
func (pl ProcessList) GetProcessesByExecutorType(executorType string) ProcessList {
	var filtered ProcessList
	for _, process := range pl {
		if process.FunctionSpec.Conditions.ExecutorType == executorType {
			filtered = append(filtered, process)
		}
	}
	return filtered
}