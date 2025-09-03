package colonyos

import (
	"fmt"
	"log"
	"time"
	"context"
	"sync"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/algorithm"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// CAPEOrchestrator integrates CAPE algorithm with ColonyOS
type CAPEOrchestrator struct {
	// ColonyOS integration
	client      ColonyOSAPI
	config      CAPEOrchestratorConfig
	
	// CAPE algorithm
	cape        *algorithm.ConfigurableCAPE
	
	// State management
	isRunning   bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
	
	// Metrics and monitoring
	stats       OrchestratorStats
	lastUpdate  time.Time
}

// CAPEOrchestratorConfig configures the orchestrator
type CAPEOrchestratorConfig struct {
	// ColonyOS configuration
	ColonyName     string        `json:"colony_name"`
	ExecutorName   string        `json:"executor_name"`
	ExecutorType   string        `json:"executor_type"`
	
	// CAPE configuration  
	DeploymentConfig *models.DeploymentConfig `json:"deployment_config"`
	
	// Orchestrator behavior
	AssignInterval    time.Duration `json:"assign_interval"`    // How often to try assigning processes
	MetricsInterval   time.Duration `json:"metrics_interval"`   // How often to update system metrics
	DecisionTimeout   time.Duration `json:"decision_timeout"`   // Max time for CAPE decision
	MaxConcurrent     int           `json:"max_concurrent"`     // Max concurrent process handling
	
	// Functions this orchestrator can handle
	SupportedFunctions []string `json:"supported_functions"`
}

// OrchestratorStats tracks orchestrator performance
type OrchestratorStats struct {
	ProcessesAssigned   int64         `json:"processes_assigned"`
	ProcessesCompleted  int64         `json:"processes_completed"`
	ProcessesFailed     int64         `json:"processes_failed"`
	CapeDecisions       int64         `json:"cape_decisions"`
	AvgDecisionTime     time.Duration `json:"avg_decision_time"`
	LastAssignment      time.Time     `json:"last_assignment"`
	Uptime              time.Duration `json:"uptime"`
	StartTime           time.Time     `json:"start_time"`
}

// NewCAPEOrchestrator creates a new CAPE-ColonyOS orchestrator
func NewCAPEOrchestrator(client ColonyOSAPI, config CAPEOrchestratorConfig) *CAPEOrchestrator {
	// Initialize CAPE algorithm with deployment configuration
	cape := algorithm.NewConfigurableCAPE(config.DeploymentConfig)
	
	return &CAPEOrchestrator{
		client:   client,
		config:   config,
		cape:     cape,
		stopChan: make(chan struct{}),
		stats: OrchestratorStats{
			StartTime: time.Now(),
		},
	}
}

// Start begins the orchestrator main loop
func (o *CAPEOrchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.isRunning {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator is already running")
	}
	o.isRunning = true
	o.mu.Unlock()
	
	log.Printf("Starting CAPE Orchestrator for colony: %s", o.config.ColonyName)
	
	// Register as executor
	err := o.registerExecutor()
	if err != nil {
		return fmt.Errorf("failed to register executor: %w", err)
	}
	
	// Register supported functions
	for _, funcName := range o.config.SupportedFunctions {
		err := o.client.AddFunction(funcName)
		if err != nil {
			log.Printf("Failed to register function %s: %v", funcName, err)
		}
	}
	
	// Start main orchestrator goroutines
	o.wg.Add(3)
	go o.processAssignmentLoop(ctx)
	go o.metricsUpdateLoop(ctx)
	go o.adaptationLoop(ctx)
	
	log.Printf("CAPE Orchestrator started successfully")
	return nil
}

// Stop gracefully stops the orchestrator
func (o *CAPEOrchestrator) Stop() error {
	o.mu.Lock()
	if !o.isRunning {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator is not running")
	}
	o.isRunning = false
	o.mu.Unlock()
	
	log.Printf("Stopping CAPE Orchestrator...")
	
	// Signal stop to all goroutines
	close(o.stopChan)
	
	// Wait for goroutines to finish
	o.wg.Wait()
	
	// Unregister executor
	err := o.client.UnregisterExecutor()
	if err != nil {
		log.Printf("Failed to unregister executor: %v", err)
	}
	
	log.Printf("CAPE Orchestrator stopped")
	return nil
}

// registerExecutor registers this instance as a ColonyOS executor
func (o *CAPEOrchestrator) registerExecutor() error {
	registration := ExecutorRegistration{
		ExecutorName: o.config.ExecutorName,
		ColonyName:   o.config.ColonyName,
		ExecutorType: o.config.ExecutorType,
		// Location and Capabilities would be populated from system info
	}
	
	return o.client.RegisterExecutor(registration)
}

// processAssignmentLoop continuously tries to assign and process tasks
func (o *CAPEOrchestrator) processAssignmentLoop(ctx context.Context) {
	defer o.wg.Done()
	
	ticker := time.NewTicker(o.config.AssignInterval)
	defer ticker.Stop()
	
	concurrentProcesses := make(chan struct{}, o.config.MaxConcurrent)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			// Try to assign a process
			select {
			case concurrentProcesses <- struct{}{}:
				go o.handleProcessAssignment(concurrentProcesses)
			default:
				// Already at max concurrent processes
				log.Printf("Max concurrent processes (%d) reached, skipping assignment", o.config.MaxConcurrent)
			}
		}
	}
}

// handleProcessAssignment handles a single process assignment
func (o *CAPEOrchestrator) handleProcessAssignment(concurrentProcesses chan struct{}) {
	defer func() { <-concurrentProcesses }()
	
	// Try to assign a process from ColonyOS
	process, err := o.client.AssignProcess(10 * time.Second)
	if err != nil {
		// No process available or assignment failed
		return
	}
	
	if process == nil {
		return
	}
	
	o.mu.Lock()
	o.stats.ProcessesAssigned++
	o.stats.LastAssignment = time.Now()
	o.mu.Unlock()
	
	log.Printf("Assigned process: %s (func: %s)", process.ProcessID, process.Spec.FuncName)
	
	// Add log entry
	o.client.AddLog(process.ProcessID, "Process assigned to CAPE orchestrator")
	
	// Handle the process
	err = o.executeProcess(process)
	if err != nil {
		log.Printf("Process execution failed: %v", err)
		o.client.FailProcess(process.ProcessID, []string{err.Error()})
		
		o.mu.Lock()
		o.stats.ProcessesFailed++
		o.mu.Unlock()
	} else {
		log.Printf("Process completed successfully: %s", process.ProcessID)
		
		o.mu.Lock()
		o.stats.ProcessesCompleted++
		o.mu.Unlock()
	}
}

// executeProcess executes a single process using CAPE decision-making
func (o *CAPEOrchestrator) executeProcess(process *models.ColonyOSProcess) error {
	o.client.AddLog(process.ProcessID, "Starting CAPE decision process")
	
	// Get current system state from ColonyOS
	systemState, err := o.getSystemStateForCAPE()
	if err != nil {
		return fmt.Errorf("failed to get system state: %w", err)
	}
	
	// Get available executors from ColonyOS
	colonyOSExecutors, err := o.client.GetActiveExecutors()
	if err != nil {
		return fmt.Errorf("failed to get active executors: %w", err)
	}
	
	// Convert ColonyOS data to CAPE format
	legacyProcess := process.ToProcess()
	legacyTargets := make([]models.OffloadTarget, len(colonyOSExecutors))
	for i, executor := range colonyOSExecutors {
		legacyTargets[i] = executor.ToOffloadTarget()
	}
	
	// Make CAPE decision
	startTime := time.Now()
	decision, err := o.cape.MakeDecision(legacyProcess, legacyTargets, *systemState)
	decisionTime := time.Since(startTime)
	
	if err != nil {
		return fmt.Errorf("CAPE decision failed: %w", err)
	}
	
	// Update stats
	o.mu.Lock()
	o.stats.CapeDecisions++
	if o.stats.CapeDecisions == 1 {
		o.stats.AvgDecisionTime = decisionTime
	} else {
		// Running average
		o.stats.AvgDecisionTime = (o.stats.AvgDecisionTime + decisionTime) / 2
	}
	o.mu.Unlock()
	
	o.client.AddLog(process.ProcessID, fmt.Sprintf("CAPE decision: target=%s, strategy=%s (took %v)", 
		decision.SelectedTarget.ID, decision.SelectedStrategy, decisionTime))
	
	// Execute the actual function
	result, err := o.executeFunction(process, decision)
	if err != nil {
		return err
	}
	
	// Report successful completion to ColonyOS
	o.client.CloseProcess(process.ProcessID, []interface{}{result})
	
	// Report outcome to CAPE for learning
	outcome := algorithm.CAPEOutcome{
		Success:       true,
		LatencyMS:     float64(decisionTime.Milliseconds()),
		CostUSD:       0.10, // Simplified cost
		ThroughputOps: 100.0, // Simplified throughput
		CompletedAt:   time.Now(),
	}
	
	err = o.cape.ReportOutcome(decision.DecisionID, outcome)
	if err != nil {
		log.Printf("Failed to report outcome to CAPE: %v", err)
	}
	
	return nil
}

// executeFunction executes the actual function logic
func (o *CAPEOrchestrator) executeFunction(process *models.ColonyOSProcess, decision algorithm.CAPEDecision) (interface{}, error) {
	o.client.AddLog(process.ProcessID, fmt.Sprintf("Executing function: %s", process.Spec.FuncName))
	
	// This is where the actual function execution would happen
	// For now, we'll simulate different function types
	
	switch process.Spec.FuncName {
	case "echo":
		// Simple echo function
		if len(process.Spec.Args) > 0 {
			return process.Spec.Args[0], nil
		}
		return "hello from CAPE orchestrator", nil
		
	case "compute":
		// Simulate compute-intensive task
		time.Sleep(time.Duration(1+len(process.Spec.Args)) * time.Second)
		return fmt.Sprintf("computed result for %d args", len(process.Spec.Args)), nil
		
	case "ml-inference":
		// Simulate ML inference
		time.Sleep(2 * time.Second)
		return map[string]interface{}{
			"prediction": 0.85,
			"confidence": 0.92,
			"model": "cape-optimized-model",
		}, nil
		
	default:
		return fmt.Sprintf("executed %s with CAPE optimization", process.Spec.FuncName), nil
	}
}

// getSystemStateForCAPE converts ColonyOS system state to CAPE format
func (o *CAPEOrchestrator) getSystemStateForCAPE() (*models.SystemState, error) {
	colonyStats, err := o.client.GetSystemStats()
	if err != nil {
		return nil, err
	}
	
	// Convert ColonyOS system state to legacy CAPE format
	systemState := &models.SystemState{
		QueueDepth:        colonyStats.PendingProcesses,
		QueueThreshold:    20, // Configurable threshold
		ComputeUsage:      models.Utilization(1.0 - colonyStats.AvailableCapacity.TotalCPU/colonyStats.TotalCapacity.TotalCPU),
		MemoryUsage:       models.Utilization(1.0 - colonyStats.AvailableCapacity.TotalMemoryGB/colonyStats.TotalCapacity.TotalMemoryGB),
		ActiveConnections: colonyStats.RunningProcesses,
		Timestamp:         time.Now(),
		TimeSlot:          time.Now().Hour(),
		DayOfWeek:         int(time.Now().Weekday()),
	}
	
	return systemState, nil
}

// metricsUpdateLoop periodically updates system metrics
func (o *CAPEOrchestrator) metricsUpdateLoop(ctx context.Context) {
	defer o.wg.Done()
	
	ticker := time.NewTicker(o.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			o.updateMetrics()
		}
	}
}

// updateMetrics updates internal metrics
func (o *CAPEOrchestrator) updateMetrics() {
	o.mu.Lock()
	o.stats.Uptime = time.Since(o.stats.StartTime)
	o.lastUpdate = time.Now()
	o.mu.Unlock()
}

// adaptationLoop runs CAPE adaptation periodically
func (o *CAPEOrchestrator) adaptationLoop(ctx context.Context) {
	defer o.wg.Done()
	
	// Run adaptation less frequently than metrics updates
	ticker := time.NewTicker(o.config.MetricsInterval * 10)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			// CAPE adaptation happens automatically within the algorithm
			// This loop could trigger additional adaptation logic if needed
			log.Printf("CAPE adaptation cycle - decisions: %d, success rate: %.2f", 
				o.stats.CapeDecisions, o.calculateSuccessRate())
		}
	}
}

// calculateSuccessRate calculates the current success rate
func (o *CAPEOrchestrator) calculateSuccessRate() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	total := o.stats.ProcessesCompleted + o.stats.ProcessesFailed
	if total == 0 {
		return 0.0
	}
	return float64(o.stats.ProcessesCompleted) / float64(total)
}

// GetStats returns current orchestrator statistics
func (o *CAPEOrchestrator) GetStats() OrchestratorStats {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	stats := o.stats
	stats.Uptime = time.Since(o.stats.StartTime)
	return stats
}

// GetCAPEStats returns CAPE algorithm statistics
func (o *CAPEOrchestrator) GetCAPEStats() algorithm.CAPEStats {
	return o.cape.GetStats()
}

// IsRunning returns whether the orchestrator is currently running
func (o *CAPEOrchestrator) IsRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.isRunning
}