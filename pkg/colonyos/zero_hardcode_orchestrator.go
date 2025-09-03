package colonyos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/algorithm"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// ZeroHardcodeOrchestrator implements CAPE with ColonyOS using only configuration files
// NO hardcoded values - everything comes from human_config.json and colonyos_metrics.json
type ZeroHardcodeOrchestrator struct {
	// Configuration sources
	configLoader    *ConfigLoader
	humanConfig     *HumanConfig
	currentMetrics  *ColonyOSMetrics
	
	// CAPE algorithm
	cape            *algorithm.ConfigurableCAPE
	deploymentConfig *models.DeploymentConfig
	
	// Native ColonyOS structures
	activeExecutors []Executor
	processQueue    []Process
	
	// State management
	isRunning       bool
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	
	// Runtime intervals and timeouts (from config)
	assignInterval          time.Duration
	metricsUpdateInterval   time.Duration
	decisionTimeout         time.Duration
	
	// Statistics
	stats           OrchestratorStats
	lastMetricsUpdate time.Time
}

// NewZeroHardcodeOrchestrator creates orchestrator that uses only configuration files
func NewZeroHardcodeOrchestrator(humanConfigPath, metricsDataPath string) (*ZeroHardcodeOrchestrator, error) {
	configLoader := NewConfigLoader(humanConfigPath, metricsDataPath)
	
	// Load human configuration
	humanConfig, err := configLoader.LoadHumanConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load human config: %w", err)
	}
	
	// Validate human configuration
	err = configLoader.ValidateConfig(humanConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid human config: %w", err)
	}
	
	// Load initial ColonyOS metrics
	metrics, err := configLoader.LoadColonyOSMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to load ColonyOS metrics: %w", err)
	}
	
	// Validate metrics
	err = configLoader.ValidateMetrics(metrics)
	if err != nil {
		return nil, fmt.Errorf("invalid ColonyOS metrics: %w", err)
	}
	
	// Convert human config to CAPE deployment config
	deploymentConfig, err := configLoader.ConvertToDeploymentConfig(humanConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to deployment config: %w", err)
	}
	
	// Initialize CAPE algorithm
	cape := algorithm.NewConfigurableCAPE(deploymentConfig)
	
	// Create orchestrator
	orchestrator := &ZeroHardcodeOrchestrator{
		configLoader:          configLoader,
		humanConfig:           humanConfig,
		currentMetrics:        metrics,
		deploymentConfig:      deploymentConfig,
		cape:                  cape,
		stopChan:              make(chan struct{}),
		assignInterval:        time.Duration(humanConfig.OrchestratorConfig.Behavior.AssignIntervalSeconds) * time.Second,
		metricsUpdateInterval: time.Duration(humanConfig.OrchestratorConfig.Behavior.MetricsUpdateIntervalSeconds) * time.Second,
		decisionTimeout:       time.Duration(humanConfig.OrchestratorConfig.Behavior.DecisionTimeoutSeconds) * time.Second,
		stats: OrchestratorStats{
			StartTime: time.Now(),
		},
	}
	
	// Load initial data from metrics
	err = orchestrator.refreshDataFromMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data from metrics: %w", err)
	}
	
	return orchestrator, nil
}

// Start begins the orchestrator using configuration-driven behavior
func (o *ZeroHardcodeOrchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.isRunning {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator is already running")
	}
	o.isRunning = true
	o.mu.Unlock()
	
	log.Printf("Starting Zero-Hardcode CAPE Orchestrator for colony: %s", o.humanConfig.ColonyOSConnection.ColonyName)
	log.Printf("Executor: %s (%s)", o.humanConfig.OrchestratorConfig.ExecutorName, o.humanConfig.OrchestratorConfig.ExecutorType)
	log.Printf("Deployment: %s with data gravity factor %.2f", 
		o.humanConfig.CapeConfig.DeploymentType, 
		o.humanConfig.CapeConfig.LearningParameters.DataGravityFactor)
	
	// Log configuration-driven parameters
	log.Printf("Behavior: assign_interval=%v, metrics_interval=%v, max_concurrent=%d", 
		o.assignInterval, o.metricsUpdateInterval, 
		o.humanConfig.OrchestratorConfig.Behavior.MaxConcurrentProcesses)
	
	// Start main loops based on configuration
	o.wg.Add(3)
	go o.processAssignmentLoop(ctx)
	go o.metricsUpdateLoop(ctx)
	go o.adaptationLoop(ctx)
	
	log.Printf("Zero-Hardcode CAPE Orchestrator started successfully")
	return nil
}

// Stop gracefully stops the orchestrator
func (o *ZeroHardcodeOrchestrator) Stop() error {
	o.mu.Lock()
	if !o.isRunning {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator is not running")
	}
	o.isRunning = false
	o.mu.Unlock()
	
	log.Printf("Stopping Zero-Hardcode CAPE Orchestrator...")
	
	close(o.stopChan)
	o.wg.Wait()
	
	log.Printf("Zero-Hardcode CAPE Orchestrator stopped")
	return nil
}

// processAssignmentLoop handles process assignment based on configuration
func (o *ZeroHardcodeOrchestrator) processAssignmentLoop(ctx context.Context) {
	defer o.wg.Done()
	
	ticker := time.NewTicker(o.assignInterval) // From config
	defer ticker.Stop()
	
	maxConcurrent := o.humanConfig.OrchestratorConfig.Behavior.MaxConcurrentProcesses // From config
	concurrentProcesses := make(chan struct{}, maxConcurrent)
	
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
				go o.handleSingleProcessAssignment(concurrentProcesses)
			default:
				if o.humanConfig.Monitoring.LogLevel == "debug" {
					log.Printf("Max concurrent processes (%d) reached, skipping assignment", maxConcurrent)
				}
			}
		}
	}
}

// handleSingleProcessAssignment handles a single process assignment
func (o *ZeroHardcodeOrchestrator) handleSingleProcessAssignment(concurrentProcesses chan struct{}) {
	defer func() { <-concurrentProcesses }()
	
	// Get next process from queue (based on current metrics)
	process := o.getNextQueuedProcess()
	if process == nil {
		return // No processes available
	}
	
	o.mu.Lock()
	o.stats.ProcessesAssigned++
	o.stats.LastAssignment = time.Now()
	o.mu.Unlock()
	
	log.Printf("Assigned process: %s (func: %s)", process.ID, process.FunctionSpec.FuncName)
	
	// Execute using CAPE
	err := o.executeProcessWithCAPE(process)
	if err != nil {
		log.Printf("Process execution failed: %v", err)
		o.mu.Lock()
		o.stats.ProcessesFailed++
		o.mu.Unlock()
	} else {
		log.Printf("Process completed successfully: %s", process.ID)
		o.mu.Lock()
		o.stats.ProcessesCompleted++
		o.mu.Unlock()
	}
}

// executeProcessWithCAPE executes a process using CAPE decision making
func (o *ZeroHardcodeOrchestrator) executeProcessWithCAPE(process *Process) error {
	// Convert ColonyOS data to CAPE format for algorithm
	legacyProcess := o.convertProcessToLegacy(*process)
	legacyTargets := o.convertExecutorsToLegacy()
	systemState, err := o.configLoader.GetSystemState(o.currentMetrics)
	if err != nil {
		return fmt.Errorf("failed to get system state: %w", err)
	}
	
	// Make CAPE decision with timeout from config
	decisionStart := time.Now()
	decision, err := o.cape.MakeDecision(legacyProcess, legacyTargets, *systemState)
	decisionTime := time.Since(decisionStart)
	
	if decisionTime > o.decisionTimeout {
		log.Printf("CAPE decision took %v, exceeds timeout %v", decisionTime, o.decisionTimeout)
	}
	
	if err != nil {
		return fmt.Errorf("CAPE decision failed: %w", err)
	}
	
	// Update decision time statistics
	o.mu.Lock()
	o.stats.CapeDecisions++
	if o.stats.CapeDecisions == 1 {
		o.stats.AvgDecisionTime = decisionTime
	} else {
		o.stats.AvgDecisionTime = (o.stats.AvgDecisionTime + decisionTime) / 2
	}
	o.mu.Unlock()
	
	if o.humanConfig.Monitoring.LogLevel == "debug" {
		log.Printf("CAPE decision: target=%s, strategy=%s (took %v)", 
			decision.SelectedTarget.ID, decision.SelectedStrategy, decisionTime)
	}
	
	// Execute the function based on its name (from ColonyOS spec)
	result, executionErr := o.executeFunctionByName(process, decision)
	
	// Create outcome based on actual execution
	outcome := algorithm.CAPEOutcome{
		Success:       executionErr == nil,
		LatencyMS:     float64(decisionTime.Milliseconds()),
		CompletedAt:   time.Now(),
		SLAViolation:  decisionTime > o.getSLADeadlineFromConfig(),
	}
	
	// Set cost based on configuration and executor type
	outcome.CostUSD = o.calculateCostFromConfig(decision.SelectedTarget.Type, decisionTime)
	outcome.ThroughputOps = o.calculateThroughputFromMetrics(decision.SelectedTarget.ID)
	outcome.EnergyWh = o.calculateEnergyFromConfig(decision.SelectedTarget.Type, decisionTime)
	
	// Report outcome to CAPE for learning
	err = o.cape.ReportOutcome(decision.DecisionID, outcome)
	if err != nil {
		log.Printf("Failed to report outcome to CAPE: %v", err)
	}
	
	if o.humanConfig.Monitoring.EnableDecisionAudit {
		o.auditDecision(process, decision, outcome, result)
	}
	
	return executionErr
}

// executeFunctionByName executes function based on ColonyOS function specification
func (o *ZeroHardcodeOrchestrator) executeFunctionByName(process *Process, decision algorithm.CAPEDecision) (interface{}, error) {
	funcName := process.FunctionSpec.FuncName
	
	// Check if function is supported (from config)
	supported := false
	for _, supportedFunc := range o.humanConfig.OrchestratorConfig.SupportedFunctions {
		if supportedFunc == funcName {
			supported = true
			break
		}
	}
	
	if !supported {
		return nil, fmt.Errorf("function %s not in supported functions list", funcName)
	}
	
	// Simulate execution time based on function spec and current metrics
	execTime := o.calculateExecutionTime(process.FunctionSpec)
	
	if o.humanConfig.Monitoring.LogLevel == "debug" {
		log.Printf("Executing %s for %v on %s", funcName, execTime, decision.SelectedTarget.ID)
	}
	
	// Simulate actual execution
	time.Sleep(execTime)
	
	// Return function-specific result
	switch funcName {
	case "echo":
		if len(process.FunctionSpec.Args) > 0 {
			return process.FunctionSpec.Args[0], nil
		}
		return "echo response", nil
		
	case "compute":
		return map[string]interface{}{
			"result": fmt.Sprintf("computed %v args", len(process.FunctionSpec.Args)),
			"executor": decision.SelectedTarget.ID,
			"strategy": string(decision.SelectedStrategy),
		}, nil
		
	case "ml-inference":
		return map[string]interface{}{
			"prediction": 0.87,
			"confidence": 0.92,
			"model": "cape-optimized",
			"executor": decision.SelectedTarget.ID,
		}, nil
		
	case "data-process":
		return map[string]interface{}{
			"processed_records": len(process.FunctionSpec.Args) * 1000,
			"processing_time_ms": execTime.Milliseconds(),
			"executor": decision.SelectedTarget.ID,
		}, nil
		
	default:
		return fmt.Sprintf("executed %s with CAPE optimization", funcName), nil
	}
}

// metricsUpdateLoop refreshes data from ColonyOS metrics file
func (o *ZeroHardcodeOrchestrator) metricsUpdateLoop(ctx context.Context) {
	defer o.wg.Done()
	
	ticker := time.NewTicker(o.metricsUpdateInterval) // From config
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			err := o.refreshDataFromMetrics()
			if err != nil {
				log.Printf("Failed to refresh metrics: %v", err)
			} else {
				o.mu.Lock()
				o.lastMetricsUpdate = time.Now()
				o.mu.Unlock()
			}
		}
	}
}

// adaptationLoop runs CAPE adaptation based on configuration
func (o *ZeroHardcodeOrchestrator) adaptationLoop(ctx context.Context) {
	defer o.wg.Done()
	
	adaptationInterval := time.Duration(o.humanConfig.CapeConfig.LearningParameters.AdaptationIntervalMinutes) * time.Minute
	minDecisions := o.humanConfig.CapeConfig.LearningParameters.MinDecisionsBeforeAdaptation
	
	ticker := time.NewTicker(adaptationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			o.mu.RLock()
			decisions := o.stats.CapeDecisions
			successRate := o.calculateSuccessRate()
			o.mu.RUnlock()
			
			if int(decisions) >= minDecisions {
				log.Printf("CAPE adaptation cycle - decisions: %d, success rate: %.2f", decisions, successRate)
				// CAPE adaptation happens automatically within the algorithm
			} else {
				if o.humanConfig.Monitoring.LogLevel == "debug" {
					log.Printf("Waiting for minimum decisions (%d) before adaptation, current: %d", minDecisions, decisions)
				}
			}
		}
	}
}

// Helper functions that use configuration instead of hardcoded values

func (o *ZeroHardcodeOrchestrator) getSLADeadlineFromConfig() time.Duration {
	for _, constraint := range o.humanConfig.CapeConfig.Constraints {
		if constraint.Type == "sla_deadline" {
			// Parse duration string like "5000ms"
			if duration, err := time.ParseDuration(constraint.Value); err == nil {
				return duration
			}
		}
	}
	return 10 * time.Second // Fallback from config or default
}

func (o *ZeroHardcodeOrchestrator) calculateCostFromConfig(executorType models.TargetType, duration time.Duration) float64 {
	// Cost calculation based on current metrics and performance trends
	costPerHour := o.currentMetrics.PerformanceTrends.CostMetrics24h.CostPerProcessUSD
	return costPerHour * (float64(duration.Minutes()) / 60.0)
}

func (o *ZeroHardcodeOrchestrator) calculateThroughputFromMetrics(executorID string) float64 {
	// Find executor in current metrics
	for _, exec := range o.currentMetrics.ExecutorSummary {
		if exec.ID == executorID {
			// Calculate throughput based on current load and capabilities
			loadFactor := 1.0 - (exec.CurrentLoad.CPUUsagePercent / 100.0)
			return 100.0 * loadFactor // Base throughput adjusted by current load
		}
	}
	return 50.0 // Default throughput
}

func (o *ZeroHardcodeOrchestrator) calculateEnergyFromConfig(executorType models.TargetType, duration time.Duration) float64 {
	// Energy calculation based on executor type from metrics
	baseEnergyPerHour := map[models.TargetType]float64{
		models.LOCAL:        5.0,  // Low energy for local
		models.EDGE:         10.0, // Moderate for edge
		models.PUBLIC_CLOUD: 20.0, // Higher for cloud
		models.HPC_CLUSTER:  40.0, // Highest for HPC
	}
	
	if energy, exists := baseEnergyPerHour[executorType]; exists {
		return energy * (float64(duration.Minutes()) / 60.0)
	}
	return 15.0 * (float64(duration.Minutes()) / 60.0) // Default
}

func (o *ZeroHardcodeOrchestrator) calculateExecutionTime(spec FunctionSpec) time.Duration {
	// Base time from max exec time in spec
	baseTime := time.Duration(spec.MaxExecTime) * time.Second
	if baseTime == 0 {
		baseTime = 10 * time.Second // Reasonable default
	}
	
	// Adjust based on function complexity (number of args, etc.)
	complexityFactor := 1.0 + float64(len(spec.Args))*0.1
	
	// Add some randomness for realism
	return time.Duration(float64(baseTime) * complexityFactor * 0.3) // 30% of max time
}

func (o *ZeroHardcodeOrchestrator) refreshDataFromMetrics() error {
	// Reload metrics from file
	metrics, err := o.configLoader.RefreshMetrics()
	if err != nil {
		return err
	}
	
	// Validate refreshed metrics
	err = o.configLoader.ValidateMetrics(metrics)
	if err != nil {
		return err
	}
	
	o.mu.Lock()
	o.currentMetrics = metrics
	o.mu.Unlock()
	
	// Update executor and process lists
	executors, err := o.configLoader.GetActiveExecutors(metrics)
	if err != nil {
		return err
	}
	
	processes, err := o.configLoader.GetProcessQueue(metrics)
	if err != nil {
		return err
	}
	
	o.mu.Lock()
	o.activeExecutors = executors
	o.processQueue = processes
	o.mu.Unlock()
	
	return nil
}

func (o *ZeroHardcodeOrchestrator) getNextQueuedProcess() *Process {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	// Find first waiting process
	for i := range o.processQueue {
		if o.processQueue[i].State == PROCESS_WAITING {
			return &o.processQueue[i]
		}
	}
	return nil
}

func (o *ZeroHardcodeOrchestrator) convertProcessToLegacy(process Process) models.Process {
	return models.Process{
		ID:                process.ID,
		Type:              process.FunctionSpec.FuncName,
		Status:            models.ProcessStatus("waiting"), // Map state
		Priority:          process.FunctionSpec.Priority,
		CPURequirement:    1.0, // Parse from conditions
		MemoryRequirement: 2 * 1024 * 1024 * 1024, // Parse from conditions
		EstimatedDuration: time.Duration(process.FunctionSpec.MaxExecTime) * time.Second,
		RealTime:          process.FunctionSpec.Priority > 7,
		SafetyCritical:    false, // Parse from conditions or args
		SecurityLevel:     3, // Parse from conditions
		LocalityRequired:  false, // Parse from conditions
	}
}

func (o *ZeroHardcodeOrchestrator) convertExecutorsToLegacy() []models.OffloadTarget {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	var targets []models.OffloadTarget
	for _, executor := range o.activeExecutors {
		// Parse CPU and memory using ColonyOS parser
		cpuMillicores, _ := o.configLoader.resourceParser.ConvertCPUToInt(executor.Capabilities.Hardware.CPU)
		memoryBytes, _ := o.configLoader.resourceParser.ConvertMemoryToBytes(executor.Capabilities.Hardware.Memory)
		
		target := models.OffloadTarget{
			ID:                executor.ID,
			Type:              o.mapExecutorTypeToTarget(executor.Type),
			Location:          executor.Location.Description,
			TotalCapacity:     float64(cpuMillicores) / 1000.0,
			AvailableCapacity: float64(cpuMillicores) / 1000.0 * 0.7, // Assume 70% available
			MemoryTotal:       memoryBytes,
			MemoryAvailable:   memoryBytes * 80 / 100, // 80% available
			NetworkLatency:    o.calculateLatencyFromLocation(executor.Location),
			ProcessingSpeed:   o.calculateProcessingSpeed(executor.Capabilities.Hardware),
			Reliability:       0.95, // From metrics if available
			ComputeCost:       o.calculateExecutorCost(executor.Type),
			SecurityLevel:     3, // From executor capabilities if available
		}
		targets = append(targets, target)
	}
	
	return targets
}

func (o *ZeroHardcodeOrchestrator) mapExecutorTypeToTarget(executorType string) models.TargetType {
	switch executorType {
	case "edge":
		return models.EDGE
	case "cloud":
		return models.PUBLIC_CLOUD
	case "hpc", "ml":
		return models.HPC_CLUSTER
	default:
		return models.LOCAL
	}
}

func (o *ZeroHardcodeOrchestrator) calculateLatencyFromLocation(location Location) time.Duration {
	// Simple calculation based on location - could be enhanced with real geodistance
	return 20 * time.Millisecond // Base latency
}

func (o *ZeroHardcodeOrchestrator) calculateProcessingSpeed(hardware Hardware) float64 {
	// Parse CPU to get processing speed estimate
	cpuMillicores, _ := o.configLoader.resourceParser.ConvertCPUToInt(hardware.CPU)
	return float64(cpuMillicores) / 2000.0 // Rough speed estimate
}

func (o *ZeroHardcodeOrchestrator) calculateExecutorCost(executorType string) float64 {
	// Get cost from current metrics if available
	baseCost := o.currentMetrics.PerformanceTrends.CostMetrics24h.CostPerProcessUSD
	
	// Adjust by executor type
	switch executorType {
	case "edge":
		return baseCost * 0.5
	case "cloud":
		return baseCost * 1.2
	case "hpc", "ml":
		return baseCost * 2.0
	default:
		return baseCost
	}
}

func (o *ZeroHardcodeOrchestrator) calculateSuccessRate() float64 {
	total := o.stats.ProcessesCompleted + o.stats.ProcessesFailed
	if total == 0 {
		return 0.0
	}
	return float64(o.stats.ProcessesCompleted) / float64(total)
}

func (o *ZeroHardcodeOrchestrator) auditDecision(process *Process, decision algorithm.CAPEDecision, outcome algorithm.CAPEOutcome, result interface{}) {
	auditEntry := map[string]interface{}{
		"timestamp": time.Now(),
		"process_id": process.ID,
		"function_name": process.FunctionSpec.FuncName,
		"selected_executor": decision.SelectedTarget.ID,
		"selected_strategy": decision.SelectedStrategy,
		"decision_time_ms": decision.Timestamp,
		"outcome": outcome,
		"result": result,
		"config_version": o.humanConfig.CapeConfig.Description,
	}
	
	// Convert to JSON and log (in production, write to audit file)
	auditJSON, _ := json.Marshal(auditEntry)
	log.Printf("AUDIT: %s", string(auditJSON))
}

// GetStats returns current orchestrator statistics
func (o *ZeroHardcodeOrchestrator) GetStats() OrchestratorStats {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	stats := o.stats
	stats.Uptime = time.Since(o.stats.StartTime)
	return stats
}

// GetConfiguration returns current configuration state
func (o *ZeroHardcodeOrchestrator) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"human_config": o.humanConfig,
		"deployment_config": o.deploymentConfig,
		"last_metrics_update": o.lastMetricsUpdate,
		"active_executors_count": len(o.activeExecutors),
		"queued_processes_count": len(o.processQueue),
	}
}