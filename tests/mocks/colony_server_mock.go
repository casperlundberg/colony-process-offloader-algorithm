package mocks

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// MockColonyServer provides a test implementation of ColonyOS server functionality
type MockColonyServer struct {
	mu sync.RWMutex
	
	// Server state
	running       bool
	port          int
	
	// Data storage
	colonies      map[string]*Colony
	executors     map[string]*Executor
	processes     map[string]*Process
	processQueue  []string // Process IDs in queue order
	
	// Execution simulation
	executingProcesses map[string]*ExecutionContext
	completedProcesses map[string]*ExecutionResult
	
	// Configuration
	simulateLatency    time.Duration
	simulateFailures   bool
	failureRate       float64
	
	// Event handlers
	eventHandlers map[EventType][]EventHandler
}

type Colony struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
}

type Executor struct {
	ID           string
	Name         string
	Type         string
	ColonyName   string
	State        ExecutorState
	Capabilities []string
	LastSeen     time.Time
	
	// Mock-specific fields
	CurrentLoad    float64
	MaxCapacity    float64
	ProcessingSpeed float64
}

type Process struct {
	ID                string
	FuncName          string
	Args              []string
	ColonyName        string
	State             ProcessState
	Priority          int
	MaxWaitTime       int
	MaxExecTime       int
	MaxRetries        int
	
	// Runtime fields
	SubmissionTime    time.Time
	AssignedExecutorID string
	StartTime         time.Time
	EndTime           time.Time
	RetryCount        int
	
	// Mock-specific fields
	RequiredCapabilities []string
	ResourceRequirements ResourceRequirements
	EstimatedDuration    time.Duration
}

type ResourceRequirements struct {
	CPUCores   float64
	Memory     int64
	Storage    int64
	Bandwidth  float64
}

type ExecutionContext struct {
	ProcessID    string
	ExecutorID   string
	StartTime    time.Time
	ExpectedEnd  time.Time
	Progress     float64
}

type ExecutionResult struct {
	ProcessID   string
	ExecutorID  string
	Success     bool
	Error       string
	Duration    time.Duration
	CompletedAt time.Time
}

type ExecutorState int
type ProcessState int
type EventType string

const (
	EXECUTOR_APPROVED ExecutorState = iota
	EXECUTOR_OFFLINE
	EXECUTOR_FAILED
)

const (
	PROCESS_WAITING ProcessState = iota
	PROCESS_ASSIGNED
	PROCESS_EXECUTING  
	PROCESS_SUCCESSFUL
	PROCESS_FAILED
)

const (
	EVENT_PROCESS_SUBMITTED EventType = "process_submitted"
	EVENT_PROCESS_ASSIGNED  EventType = "process_assigned"
	EVENT_PROCESS_STARTED   EventType = "process_started"
	EVENT_PROCESS_COMPLETED EventType = "process_completed"
	EVENT_EXECUTOR_STATE    EventType = "executor_state_changed"
)

type EventHandler func(event Event)

type Event struct {
	Type      EventType
	Data      map[string]interface{}
	Timestamp time.Time
}

func NewMockColonyServer() *MockColonyServer {
	return &MockColonyServer{
		running:            false,
		colonies:           make(map[string]*Colony),
		executors:          make(map[string]*Executor),
		processes:          make(map[string]*Process),
		processQueue:       []string{},
		executingProcesses: make(map[string]*ExecutionContext),
		completedProcesses: make(map[string]*ExecutionResult),
		simulateLatency:    50 * time.Millisecond,
		simulateFailures:   false,
		failureRate:        0.05,
		eventHandlers:      make(map[EventType][]EventHandler),
	}
}

// Server management
func (m *MockColonyServer) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.running {
		return fmt.Errorf("server already running")
	}
	
	m.running = true
	m.port = 8080 + len(m.colonies) // Simple port assignment
	
	// Start background process executor
	go m.executeProcesses()
	
	return nil
}

func (m *MockColonyServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.running = false
	return nil
}

func (m *MockColonyServer) GetPort() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.port
}

func (m *MockColonyServer) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// Colony management
func (m *MockColonyServer) AddColony(colony *Colony) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if colony.ID == "" {
		colony.ID = generateID()
	}
	
	if colony.CreatedAt.IsZero() {
		colony.CreatedAt = time.Now()
	}
	
	m.colonies[colony.ID] = colony
	return nil
}

func (m *MockColonyServer) GetColony(id string) (*Colony, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	colony, exists := m.colonies[id]
	if !exists {
		return nil, fmt.Errorf("colony not found: %s", id)
	}
	
	return colony, nil
}

func (m *MockColonyServer) ListColonies() ([]*Colony, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	colonies := make([]*Colony, 0, len(m.colonies))
	for _, colony := range m.colonies {
		colonies = append(colonies, colony)
	}
	
	return colonies, nil
}

// Executor management
func (m *MockColonyServer) AddExecutor(executor *Executor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if executor.ID == "" {
		executor.ID = generateID()
	}
	
	executor.LastSeen = time.Now()
	executor.State = EXECUTOR_APPROVED
	
	if executor.MaxCapacity == 0 {
		executor.MaxCapacity = 8.0 // Default capacity
	}
	
	if executor.ProcessingSpeed == 0 {
		executor.ProcessingSpeed = 1.0 // Default speed
	}
	
	m.executors[executor.ID] = executor
	
	m.emitEvent(EVENT_EXECUTOR_STATE, map[string]interface{}{
		"executor_id": executor.ID,
		"state":       executor.State,
	})
	
	return nil
}

func (m *MockColonyServer) GetExecutor(id string) (*Executor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	executor, exists := m.executors[id]
	if !exists {
		return nil, fmt.Errorf("executor not found: %s", id)
	}
	
	return executor, nil
}

func (m *MockColonyServer) ListExecutors(colonyName string) ([]*Executor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	executors := []*Executor{}
	for _, executor := range m.executors {
		if colonyName == "" || executor.ColonyName == colonyName {
			executors = append(executors, executor)
		}
	}
	
	return executors, nil
}

func (m *MockColonyServer) SetExecutorState(id string, state ExecutorState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	executor, exists := m.executors[id]
	if !exists {
		return fmt.Errorf("executor not found: %s", id)
	}
	
	executor.State = state
	executor.LastSeen = time.Now()
	
	m.emitEvent(EVENT_EXECUTOR_STATE, map[string]interface{}{
		"executor_id": id,
		"state":       state,
	})
	
	return nil
}

// Process management
func (m *MockColonyServer) SubmitProcessSpec(spec ProcessSpec) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	process := &Process{
		ID:                   generateID(),
		FuncName:             spec.FuncName,
		Args:                 spec.Args,
		ColonyName:           spec.ColonyName,
		State:                PROCESS_WAITING,
		Priority:             spec.Priority,
		MaxWaitTime:          spec.MaxWaitTime,
		MaxExecTime:          spec.MaxExecTime,
		MaxRetries:           spec.MaxRetries,
		SubmissionTime:       time.Now(),
		RequiredCapabilities: spec.RequiredCapabilities,
		ResourceRequirements: spec.ResourceRequirements,
		EstimatedDuration:    spec.EstimatedDuration,
	}
	
	m.processes[process.ID] = process
	m.processQueue = append(m.processQueue, process.ID)
	
	m.emitEvent(EVENT_PROCESS_SUBMITTED, map[string]interface{}{
		"process_id":  process.ID,
		"colony_name": process.ColonyName,
		"func_name":   process.FuncName,
	})
	
	return process, nil
}

func (m *MockColonyServer) GetProcess(id string) (*Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	process, exists := m.processes[id]
	if !exists {
		return nil, fmt.Errorf("process not found: %s", id)
	}
	
	return process, nil
}

func (m *MockColonyServer) GetWaitingProcesses(colonyName string) ([]*Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	waiting := []*Process{}
	
	for _, processID := range m.processQueue {
		process, exists := m.processes[processID]
		if !exists {
			continue
		}
		
		if process.State == PROCESS_WAITING &&
		   (colonyName == "" || process.ColonyName == colonyName) {
			waiting = append(waiting, process)
		}
	}
	
	return waiting, nil
}

func (m *MockColonyServer) DeleteProcess(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.processes, id)
	
	// Remove from queue
	for i, processID := range m.processQueue {
		if processID == id {
			m.processQueue = append(m.processQueue[:i], m.processQueue[i+1:]...)
			break
		}
	}
	
	return nil
}

// Process execution simulation
func (m *MockColonyServer) executeProcesses() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		m.mu.Lock()
		running := m.running
		m.mu.Unlock()
		
		if !running {
			break
		}
		
		m.processQueuedProcesses()
		m.updateExecutingProcesses()
	}
}

func (m *MockColonyServer) processQueuedProcesses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Try to assign queued processes to available executors
	for i, processID := range m.processQueue {
		_ = processID // Mark as used to avoid compiler warning
		process, exists := m.processes[processID]
		if !exists || process.State != PROCESS_WAITING {
			continue
		}
		
		// Find suitable executor
		executor := m.findSuitableExecutor(process)
		if executor == nil {
			continue
		}
		
		// Assign process to executor
		process.AssignedExecutorID = executor.ID
		process.State = PROCESS_ASSIGNED
		
		// Remove from queue
		m.processQueue = append(m.processQueue[:i], m.processQueue[i+1:]...)
		
		m.emitEvent(EVENT_PROCESS_ASSIGNED, map[string]interface{}{
			"process_id":  process.ID,
			"executor_id": executor.ID,
		})
		
		// Start execution after assignment delay
		go m.startProcessExecution(process, executor)
		break // Process one at a time for simplicity
	}
}

func (m *MockColonyServer) findSuitableExecutor(process *Process) *Executor {
	for _, executor := range m.executors {
		if executor.State != EXECUTOR_APPROVED {
			continue
		}
		
		if executor.ColonyName != process.ColonyName {
			continue
		}
		
		// Check capacity
		if executor.CurrentLoad >= executor.MaxCapacity {
			continue
		}
		
		// Check capabilities
		if m.hasRequiredCapabilities(executor, process.RequiredCapabilities) {
			return executor
		}
	}
	
	return nil
}

func (m *MockColonyServer) hasRequiredCapabilities(executor *Executor, required []string) bool {
	for _, req := range required {
		found := false
		for _, cap := range executor.Capabilities {
			if cap == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (m *MockColonyServer) startProcessExecution(process *Process, executor *Executor) {
	time.Sleep(m.simulateLatency)
	
	m.mu.Lock()
	
	process.State = PROCESS_EXECUTING
	process.StartTime = time.Now()
	
	duration := process.EstimatedDuration
	if duration == 0 {
		duration = 30 * time.Second // Default duration
	}
	
	// Add some randomness
	duration = duration + time.Duration(float64(duration)*0.2*(2.0*rand.Float64()-1.0))
	
	context := &ExecutionContext{
		ProcessID:   process.ID,
		ExecutorID:  executor.ID,
		StartTime:   process.StartTime,
		ExpectedEnd: process.StartTime.Add(duration),
		Progress:    0.0,
	}
	
	m.executingProcesses[process.ID] = context
	executor.CurrentLoad += 1.0 // Simplified load tracking
	
	m.mu.Unlock()
	
	m.emitEvent(EVENT_PROCESS_STARTED, map[string]interface{}{
		"process_id":  process.ID,
		"executor_id": executor.ID,
		"start_time":  process.StartTime,
	})
	
	// Simulate execution
	go m.simulateExecution(context)
}

func (m *MockColonyServer) simulateExecution(context *ExecutionContext) {
	duration := context.ExpectedEnd.Sub(context.StartTime)
	
	// Update progress periodically
	progressTicker := time.NewTicker(duration / 10)
	defer progressTicker.Stop()
	
	startTime := time.Now()
	
	for range progressTicker.C {
		elapsed := time.Since(startTime)
		progress := float64(elapsed) / float64(duration)
		
		m.mu.Lock()
		if ctx, exists := m.executingProcesses[context.ProcessID]; exists {
			ctx.Progress = math.Min(progress, 1.0)
		}
		m.mu.Unlock()
		
		if progress >= 1.0 {
			break
		}
	}
	
	// Complete execution
	m.completeExecution(context)
}

func (m *MockColonyServer) completeExecution(context *ExecutionContext) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	process, exists := m.processes[context.ProcessID]
	if !exists {
		return
	}
	
	executor, exists := m.executors[context.ExecutorID]
	if exists {
		executor.CurrentLoad -= 1.0 // Release capacity
		executor.CurrentLoad = math.Max(0.0, executor.CurrentLoad)
	}
	
	// Simulate failures
	success := true
	errorMsg := ""
	
	if m.simulateFailures && rand.Float64() < m.failureRate {
		success = false
		errorMsg = "Simulated execution failure"
	}
	
	// Update process state
	process.EndTime = time.Now()
	if success {
		process.State = PROCESS_SUCCESSFUL
	} else {
		process.State = PROCESS_FAILED
	}
	
	duration := process.EndTime.Sub(process.StartTime)
	
	result := &ExecutionResult{
		ProcessID:   context.ProcessID,
		ExecutorID:  context.ExecutorID,
		Success:     success,
		Error:       errorMsg,
		Duration:    duration,
		CompletedAt: process.EndTime,
	}
	
	m.completedProcesses[context.ProcessID] = result
	delete(m.executingProcesses, context.ProcessID)
	
	m.emitEvent(EVENT_PROCESS_COMPLETED, map[string]interface{}{
		"process_id":  process.ID,
		"executor_id": context.ExecutorID,
		"success":     success,
		"duration":    duration,
		"error":       errorMsg,
	})
}

func (m *MockColonyServer) updateExecutingProcesses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, context := range m.executingProcesses {
		if time.Now().After(context.ExpectedEnd) {
			// Force completion for stuck processes
			go m.completeExecution(context)
		}
	}
}

// Event system
func (m *MockColonyServer) RegisterEventHandler(eventType EventType, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.eventHandlers[eventType] == nil {
		m.eventHandlers[eventType] = []EventHandler{}
	}
	
	m.eventHandlers[eventType] = append(m.eventHandlers[eventType], handler)
	return nil
}

func (m *MockColonyServer) emitEvent(eventType EventType, data map[string]interface{}) {
	event := Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}
	
	if handlers, exists := m.eventHandlers[eventType]; exists {
		for _, handler := range handlers {
			go handler(event) // Async event handling
		}
	}
}

// Test utilities
func (m *MockColonyServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.processes = make(map[string]*Process)
	m.processQueue = []string{}
	m.executingProcesses = make(map[string]*ExecutionContext)
	m.completedProcesses = make(map[string]*ExecutionResult)
	
	// Reset executor loads
	for _, executor := range m.executors {
		executor.CurrentLoad = 0.0
	}
}

func (m *MockColonyServer) SetSimulationConfig(latency time.Duration, simulateFailures bool, failureRate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.simulateLatency = latency
	m.simulateFailures = simulateFailures
	m.failureRate = failureRate
}

func (m *MockColonyServer) GetExecutionResult(processID string) (*ExecutionResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result, exists := m.completedProcesses[processID]
	if !exists {
		return nil, fmt.Errorf("execution result not found for process: %s", processID)
	}
	
	return result, nil
}

func (m *MockColonyServer) GetQueueLength() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.processQueue)
}

func (m *MockColonyServer) GetExecutingProcessCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return len(m.executingProcesses)
}

// ProcessSpec for submitting processes
type ProcessSpec struct {
	FuncName             string
	Args                 []string
	ColonyName           string
	Priority             int
	MaxWaitTime          int
	MaxExecTime          int
	MaxRetries           int
	RequiredCapabilities []string
	ResourceRequirements ResourceRequirements
	EstimatedDuration    time.Duration
}

// Helper functions
func generateID() string {
	return fmt.Sprintf("mock_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}