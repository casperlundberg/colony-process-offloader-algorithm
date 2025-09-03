package colonyos

import (
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// ColonyOSClient provides an interface to ColonyOS server
type ColonyOSClient struct {
	serverURL    string
	colonyName   string
	colonyPrvKey string
	executorName string
	executorPrvKey string
	executorID   string
	
	// Client configuration
	assignTimeout time.Duration
	retryInterval time.Duration
}

// NewColonyOSClient creates a new ColonyOS client
func NewColonyOSClient(config ColonyOSClientConfig) *ColonyOSClient {
	return &ColonyOSClient{
		serverURL:     config.ServerURL,
		colonyName:    config.ColonyName,
		colonyPrvKey:  config.ColonyPrivateKey,
		executorName:  config.ExecutorName,
		executorPrvKey: config.ExecutorPrivateKey,
		executorID:    config.ExecutorID,
		assignTimeout: config.AssignTimeout,
		retryInterval: config.RetryInterval,
	}
}

// ColonyOSClientConfig holds configuration for ColonyOS client
type ColonyOSClientConfig struct {
	ServerURL          string        `json:"server_url"`
	ColonyName         string        `json:"colony_name"`
	ColonyPrivateKey   string        `json:"colony_private_key"`
	ExecutorName       string        `json:"executor_name"`
	ExecutorPrivateKey string        `json:"executor_private_key"`
	ExecutorID         string        `json:"executor_id"`
	AssignTimeout      time.Duration `json:"assign_timeout"`
	RetryInterval      time.Duration `json:"retry_interval"`
}

// ExecutorRegistration represents executor registration data
type ExecutorRegistration struct {
	ExecutorName string                    `json:"executorname"`
	ExecutorID   string                    `json:"executorid"`
	ColonyName   string                    `json:"colonyname"`
	ExecutorType string                    `json:"executortype"`
	Location     *models.ColonyOSLocation  `json:"location,omitempty"`
	Capabilities *models.ColonyOSCapabilities `json:"capabilities,omitempty"`
}

// ColonyOSAPI defines the interface for ColonyOS operations needed by CAPE
type ColonyOSAPI interface {
	// Executor management
	RegisterExecutor(registration ExecutorRegistration) error
	UnregisterExecutor() error
	AddFunction(funcName string) error
	
	// Process assignment and execution
	AssignProcess(timeout time.Duration) (*models.ColonyOSProcess, error)
	CloseProcess(processID string, output []interface{}) error
	FailProcess(processID string, errors []string) error
	
	// Monitoring and metrics
	GetQueuedProcesses() ([]models.ColonyOSProcess, error)
	GetActiveExecutors() ([]models.ColonyOSExecutor, error)
	GetSystemStats() (*models.ColonyOSSystemState, error)
	GetProcessHistory(limit int) ([]models.ColonyOSProcess, error)
	
	// Process submission (for testing/coordination)
	SubmitProcessSpec(spec models.ColonyOSProcessSpec) (*models.ColonyOSProcess, error)
	
	// Logging and debugging
	AddLog(processID string, message string) error
}

// RegisterExecutor registers this instance as an executor with ColonyOS
func (c *ColonyOSClient) RegisterExecutor(registration ExecutorRegistration) error {
	// TODO: Implement actual HTTP call to ColonyOS server
	// This is a placeholder that would normally make an HTTP POST request
	fmt.Printf("Registering executor: %s (type: %s) with colony: %s\n", 
		registration.ExecutorName, registration.ExecutorType, registration.ColonyName)
	
	// Simulate registration
	time.Sleep(100 * time.Millisecond)
	return nil
}

// UnregisterExecutor removes this executor from ColonyOS
func (c *ColonyOSClient) UnregisterExecutor() error {
	// TODO: Implement actual HTTP call to ColonyOS server
	fmt.Printf("Unregistering executor: %s\n", c.executorName)
	
	// Simulate unregistration
	time.Sleep(50 * time.Millisecond)
	return nil
}

// AddFunction registers a function that this executor can handle
func (c *ColonyOSClient) AddFunction(funcName string) error {
	// TODO: Implement actual HTTP call to ColonyOS server
	fmt.Printf("Adding function '%s' to executor: %s\n", funcName, c.executorName)
	
	// Simulate function registration
	time.Sleep(50 * time.Millisecond)
	return nil
}

// AssignProcess attempts to assign a process from the queue
func (c *ColonyOSClient) AssignProcess(timeout time.Duration) (*models.ColonyOSProcess, error) {
	// TODO: Implement actual HTTP call to ColonyOS server
	// This would normally make an HTTP POST request to /assign endpoint
	
	// For now, return nil to indicate no process available
	// In a real implementation, this would:
	// 1. Make HTTP POST to /assign with timeout
	// 2. Handle competition with other executors
	// 3. Parse the returned process JSON
	// 4. Return the process or nil if none available
	
	return nil, fmt.Errorf("no process available")
}

// CloseProcess marks a process as successfully completed
func (c *ColonyOSClient) CloseProcess(processID string, output []interface{}) error {
	// TODO: Implement actual HTTP call to ColonyOS server
	fmt.Printf("Closing process %s with output: %v\n", processID, output)
	
	// Simulate process closure
	time.Sleep(50 * time.Millisecond)
	return nil
}

// FailProcess marks a process as failed
func (c *ColonyOSClient) FailProcess(processID string, errors []string) error {
	// TODO: Implement actual HTTP call to ColonyOS server
	fmt.Printf("Failing process %s with errors: %v\n", processID, errors)
	
	// Simulate process failure
	time.Sleep(50 * time.Millisecond)
	return nil
}

// GetQueuedProcesses returns all processes currently in the queue
func (c *ColonyOSClient) GetQueuedProcesses() ([]models.ColonyOSProcess, error) {
	// TODO: Implement actual HTTP call to ColonyOS server
	// This would query the colony's process queue
	
	// Return empty slice for now
	return []models.ColonyOSProcess{}, nil
}

// GetActiveExecutors returns all active executors in the colony
func (c *ColonyOSClient) GetActiveExecutors() ([]models.ColonyOSExecutor, error) {
	// TODO: Implement actual HTTP call to ColonyOS server
	// This would query the colony's executor registry
	
	// Return sample executors for demonstration
	executors := []models.ColonyOSExecutor{
		{
			ExecutorName: "sample-ml-executor",
			ExecutorType: "ml",
			Location: models.ColonyOSLocation{
				Longitude:   65.61,
				Latitude:    22.13,
				Description: "ICE Datacenter",
			},
			Capabilities: models.ColonyOSCapabilities{
				Hardware: models.ColonyOSHardware{
					Model:   "AMD Ryzen 9 5950X",
					CPU:     "4000m",
					Memory:  "16Gi",
					Storage: "100Ti",
					GPU: &models.ColonyOSGPU{
						Name:  "nvidia_3080ti",
						Count: 1,
					},
				},
				Software: models.ColonyOSSoftware{
					Name:    "colonyos/ml:latest",
					Type:    "k8s",
					Version: "latest",
				},
			},
			Status:      models.ExecutorStatusOnline,
			LastSeen:    time.Now(),
			Utilization: models.DetailedUtilization{
				ComputeUsage: 0.3,
				MemoryUsage:  0.4,
				DiskUsage:    0.2,
				NetworkUsage: 0.1,
			},
		},
	}
	
	return executors, nil
}

// GetSystemStats returns system-wide statistics
func (c *ColonyOSClient) GetSystemStats() (*models.ColonyOSSystemState, error) {
	// TODO: Implement actual HTTP call to ColonyOS server
	
	// Return sample system state
	stats := &models.ColonyOSSystemState{
		ColonyName:         c.colonyName,
		Timestamp:          time.Now(),
		PendingProcesses:   5,
		RunningProcesses:   3,
		CompletedProcesses: 1250,
		FailedProcesses:    45,
		ExecutorsByType: map[string]int{
			"ml":    2,
			"edge":  5,
			"cloud": 3,
		},
		TotalCapacity: models.ResourceCapacity{
			TotalCPU:      32.0,
			TotalMemoryGB: 128.0,
			TotalGPUs:     4,
			TotalStorage:  1000.0,
		},
		AvailableCapacity: models.ResourceCapacity{
			TotalCPU:      20.0,
			TotalMemoryGB: 85.0,
			TotalGPUs:     2,
			TotalStorage:  800.0,
		},
		AvgProcessLatency: 2 * time.Second,
		ProcessThroughput: 15.5,
		SuccessRate:       0.96,
	}
	
	return stats, nil
}

// GetProcessHistory returns recent process execution history
func (c *ColonyOSClient) GetProcessHistory(limit int) ([]models.ColonyOSProcess, error) {
	// TODO: Implement actual HTTP call to ColonyOS server
	
	// Return empty slice for now
	return []models.ColonyOSProcess{}, nil
}

// SubmitProcessSpec submits a new process specification to the colony
func (c *ColonyOSClient) SubmitProcessSpec(spec models.ColonyOSProcessSpec) (*models.ColonyOSProcess, error) {
	// TODO: Implement actual HTTP call to ColonyOS server
	
	// Create a mock process
	process := &models.ColonyOSProcess{
		ProcessID:      fmt.Sprintf("proc-%d", time.Now().Unix()),
		Spec:           spec,
		State:          models.ProcessStateWaiting,
		SubmissionTime: time.Now(),
	}
	
	fmt.Printf("Submitted process %s (func: %s)\n", process.ProcessID, spec.FuncName)
	return process, nil
}

// AddLog adds a log entry for a process
func (c *ColonyOSClient) AddLog(processID string, message string) error {
	// TODO: Implement actual HTTP call to ColonyOS server
	fmt.Printf("Log [%s]: %s\n", processID, message)
	
	return nil
}

// MockColonyOSClient provides a mock implementation for testing
type MockColonyOSClient struct {
	*ColonyOSClient
	mockProcesses []models.ColonyOSProcess
	mockExecutors []models.ColonyOSExecutor
}

// NewMockColonyOSClient creates a mock client for testing
func NewMockColonyOSClient() *MockColonyOSClient {
	return &MockColonyOSClient{
		ColonyOSClient: &ColonyOSClient{
			colonyName:   "test-colony",
			executorName: "mock-executor",
			executorID:   "mock-executor-id",
		},
		mockProcesses: []models.ColonyOSProcess{},
		mockExecutors: []models.ColonyOSExecutor{},
	}
}

// AssignProcess returns a mock process for testing
func (m *MockColonyOSClient) AssignProcess(timeout time.Duration) (*models.ColonyOSProcess, error) {
	if len(m.mockProcesses) > 0 {
		process := m.mockProcesses[0]
		m.mockProcesses = m.mockProcesses[1:]
		return &process, nil
	}
	return nil, fmt.Errorf("no process available")
}

// AddMockProcess adds a process to the mock queue
func (m *MockColonyOSClient) AddMockProcess(process models.ColonyOSProcess) {
	m.mockProcesses = append(m.mockProcesses, process)
}

// AddMockExecutor adds an executor to the mock registry
func (m *MockColonyOSClient) AddMockExecutor(executor models.ColonyOSExecutor) {
	m.mockExecutors = append(m.mockExecutors, executor)
}

// GetActiveExecutors returns mock executors
func (m *MockColonyOSClient) GetActiveExecutors() ([]models.ColonyOSExecutor, error) {
	return m.mockExecutors, nil
}

// JSON serialization helpers

// MarshalProcessSpec converts a process spec to JSON
func MarshalProcessSpec(spec models.ColonyOSProcessSpec) ([]byte, error) {
	return json.Marshal(spec)
}

// UnmarshalProcessSpec converts JSON to a process spec
func UnmarshalProcessSpec(data []byte) (*models.ColonyOSProcessSpec, error) {
	var spec models.ColonyOSProcessSpec
	err := json.Unmarshal(data, &spec)
	if err != nil {
		return nil, err
	}
	return &spec, nil
}

// MarshalExecutor converts an executor to JSON
func MarshalExecutor(executor models.ColonyOSExecutor) ([]byte, error) {
	return json.Marshal(executor)
}

// UnmarshalExecutor converts JSON to an executor
func UnmarshalExecutor(data []byte) (*models.ColonyOSExecutor, error) {
	var executor models.ColonyOSExecutor
	err := json.Unmarshal(data, &executor)
	if err != nil {
		return nil, err
	}
	return &executor, nil
}