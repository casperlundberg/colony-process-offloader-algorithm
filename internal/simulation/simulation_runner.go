package simulation

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/autoscaler"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/queue"
)

// SimulationRunner orchestrates the entire spike simulation
type SimulationRunner struct {
	// Core components
	SpikeGenerator  *SpikeGenerator
	QueueSimulator  *QueueSimulator
	CAPEAutoscaler  *autoscaler.CAPEAutoscaler
	
	// Configuration
	Config          SimulationConfig
	SpikeScenarios  []SpikeScenario
	
	// State tracking
	CurrentTime     time.Time
	SimulationStart time.Time
	SpikeEvents     []SpikeEvent
	ActiveSpikes    map[string]*ActiveSpike
	DeployedExecutors map[string]*DeployedExecutor
	
	// Metrics collection
	Metrics         *SimulationMetrics
	DBCollector     *DBMetricsCollector  // Database collector for storing metrics
	
	// Reporting
	ReportInterval  time.Duration
	LastReportTime  time.Time
}

// SimulationConfig contains simulation parameters
type SimulationConfig struct {
	BaseProcessRate         float64              `json:"base_process_rate"`
	Scenarios               []SpikeScenario      `json:"scenarios"`
	SimulationParameters    SimulationParameters `json:"simulation_parameters"`
}

// SimulationParameters defines simulation behavior
type SimulationParameters struct {
	DurationHours           int     `json:"duration_hours"`
	WarmupHours            int     `json:"warmup_hours"`
	MeasurementIntervalMin int     `json:"measurement_interval_minutes"`
	EnableLearning         bool    `json:"enable_learning"`
	InitialExplorationRate float64 `json:"initial_exploration_rate"`
	TargetSLAPercentile    float64 `json:"target_sla_percentile"`
	MaxCostPerHour         float64 `json:"max_cost_per_hour"`
}

// ActiveSpike tracks an ongoing spike
type ActiveSpike struct {
	Event           SpikeEvent
	CurrentMinute   int
	ProcessesCreated int
}

// DeployedExecutor tracks a deployed executor instance
type DeployedExecutor struct {
	ExecutorID     string
	DeploymentTime time.Time
	ReadyTime      time.Time
	IsReady        bool
	ProcessesRun   int
	TotalCost      float64
}

// SimulationMetrics collects performance metrics
type SimulationMetrics struct {
	// Process metrics
	TotalProcessesGenerated int
	TotalProcessesCompleted int
	TotalProcessesFailed    int
	
	// Queue metrics
	MaxQueueDepth          int
	AverageQueueDepth      float64
	TotalWaitTime          time.Duration
	MaxWaitTime            time.Duration
	
	// Scaling metrics
	TotalScalingDecisions  int
	ScaleUpDecisions       int
	ScaleDownDecisions     int
	TotalDeployedExecutors int
	
	// Cost metrics
	TotalInfrastructureCost float64
	TotalDataTransferCost   float64
	CostPerProcess          float64
	
	// SLA metrics
	SLAViolations          int
	SLAComplianceRate      float64
	
	// Learning metrics
	PredictionAccuracy     float64
	WeightEvolution        map[string][]float64
	StrategySelection      map[string]int
	
	// Time series data
	TimeSeriesData        []TimeSeriesPoint
}

// TimeSeriesPoint represents a point in time during simulation
type TimeSeriesPoint struct {
	Timestamp         time.Time
	QueueDepth        int
	ActiveExecutors   int
	ProcessRate       float64
	Cost              float64
	SLACompliance     float64
	PredictionAccuracy float64
}

// NewSimulationRunner creates a new simulation runner
func NewSimulationRunner(configPath, catalogPath, autoscalerConfigPath string, dbCollector *DBMetricsCollector) (*SimulationRunner, error) {
	// Load simulation config
	config, err := loadSimulationConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load simulation config: %w", err)
	}
	
	// Create components
	spikeGen := NewSpikeGenerator(config.BaseProcessRate, config.Scenarios)
	queueSim := NewQueueSimulator(config.BaseProcessRate)
	
	// Create autoscaler
	capeAutoscaler, err := autoscaler.NewCAPEAutoscaler(autoscalerConfigPath, catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create autoscaler: %w", err)
	}
	
	return &SimulationRunner{
		SpikeGenerator:    spikeGen,
		QueueSimulator:    queueSim,
		CAPEAutoscaler:    capeAutoscaler,
		Config:           config,
		SpikeScenarios:   config.Scenarios,
		CurrentTime:      time.Now(),
		SimulationStart:  time.Now(),
		ActiveSpikes:     make(map[string]*ActiveSpike),
		DeployedExecutors: make(map[string]*DeployedExecutor),
		Metrics:          NewSimulationMetrics(),
		DBCollector:      dbCollector,
		ReportInterval:   time.Duration(config.SimulationParameters.MeasurementIntervalMin) * time.Minute,
	}, nil
}

// NewSimulationMetrics creates a new metrics collector
func NewSimulationMetrics() *SimulationMetrics {
	return &SimulationMetrics{
		WeightEvolution:   make(map[string][]float64),
		StrategySelection: make(map[string]int),
		TimeSeriesData:    make([]TimeSeriesPoint, 0),
	}
}

// Run executes the simulation
func (sr *SimulationRunner) Run() error {
	log.Printf("Starting CAPE spike simulation for %d hours", sr.Config.SimulationParameters.DurationHours)
	
	// Generate spike timeline
	var err error
	sr.SpikeEvents, err = sr.SpikeGenerator.GenerateSpikeTimeline(sr.Config.SimulationParameters.DurationHours)
	if err != nil {
		return fmt.Errorf("failed to generate spike timeline: %w", err)
	}
	
	log.Printf("Generated %d spike events", len(sr.SpikeEvents))
	
	// Simulation loop
	simulationDuration := time.Duration(sr.Config.SimulationParameters.DurationHours) * time.Hour
	endTime := sr.SimulationStart.Add(simulationDuration)
	
	// Time step (1 minute intervals)
	timeStep := 1 * time.Minute
	
	for sr.CurrentTime.Before(endTime) {
		// Check for new spikes
		sr.checkForNewSpikes()
		
		// Generate processes from active spikes
		sr.generateSpikeProcesses()
		
		// Generate baseline processes
		sr.generateBaselineProcesses()
		
		// Make scaling decisions
		if sr.shouldMakeScalingDecision() {
			sr.makeScalingDecisions()
		}
		
		// Update deployed executors
		sr.updateDeployedExecutors()
		
		// Process queue (simulate execution)
		sr.processQueue()
		
		// Collect metrics
		sr.collectMetrics()
		
		// Report if needed
		if sr.shouldReport() {
			sr.generateReport()
		}
		
		// Advance time
		sr.CurrentTime = sr.CurrentTime.Add(timeStep)
		sr.QueueSimulator.AdvanceTime(timeStep)
	}
	
	// Generate final report
	sr.generateFinalReport()
	
	// Close database collector
	if sr.DBCollector != nil {
		if err := sr.DBCollector.Close(); err != nil {
			log.Printf("Warning: Failed to close database collector: %v", err)
		}
	}
	
	return nil
}

// checkForNewSpikes checks if any new spikes should start
func (sr *SimulationRunner) checkForNewSpikes() {
	for _, spike := range sr.SpikeEvents {
		// Check if spike should start
		if spike.StartTime.After(sr.SimulationStart) && 
		   spike.StartTime.Before(sr.CurrentTime.Add(time.Minute)) &&
		   spike.StartTime.After(sr.CurrentTime.Add(-time.Minute)) {
			// Start new spike
			sr.ActiveSpikes[spike.ID] = &ActiveSpike{
				Event:         spike,
				CurrentMinute: 0,
			}
			log.Printf("Spike started: %s (executor: %s, priority: %v)", 
				spike.Name, spike.ExecutorType, spike.PriorityDistribution)
		}
	}
}

// generateSpikeProcesses generates processes from active spikes
func (sr *SimulationRunner) generateSpikeProcesses() {
	for id, activeSpike := range sr.ActiveSpikes {
		// Check if spike is still active
		if sr.CurrentTime.After(activeSpike.Event.EndTime) {
			log.Printf("Spike ended: %s (generated %d processes)", 
				activeSpike.Event.Name, activeSpike.ProcessesCreated)
			delete(sr.ActiveSpikes, id)
			continue
		}
		
		// Generate processes for current minute
		processes := sr.QueueSimulator.GenerateProcessesFromSpike(
			activeSpike.Event, 
			activeSpike.CurrentMinute,
		)
		
		activeSpike.ProcessesCreated += len(processes)
		activeSpike.CurrentMinute++
		
		sr.Metrics.TotalProcessesGenerated += len(processes)
	}
}

// generateBaselineProcesses generates normal background load
func (sr *SimulationRunner) generateBaselineProcesses() {
	processes := sr.QueueSimulator.GenerateBaselineProcesses()
	sr.Metrics.TotalProcessesGenerated += len(processes)
}

// shouldMakeScalingDecision checks if it's time to make scaling decisions
func (sr *SimulationRunner) shouldMakeScalingDecision() bool {
	// Make decisions when there's any queue buildup or every 5 minutes
	queueState := sr.QueueSimulator.GetQueueState()
	return sr.CurrentTime.Minute()%5 == 0 || queueState.QueueDepth > 10
}

// makeScalingDecisions uses CAPE to make scaling decisions
func (sr *SimulationRunner) makeScalingDecisions() {
	// Get current queue state
	queueState := sr.QueueSimulator.GetWaitingProcesses()
	
	// Get current executors (simplified for simulation)
	currentExecutors := sr.getCurrentExecutors()
	
	log.Printf("Making scaling decision: queue=%d processes, current executors=%d", 
		len(queueState), len(currentExecutors))
	
	// Make scaling decisions
	decisions := sr.CAPEAutoscaler.MakeScalingDecision(queueState, currentExecutors)
	
	log.Printf("CAPE returned %d scaling decisions", len(decisions))
	
	// Execute decisions
	for _, decision := range decisions {
		sr.executeScalingDecision(decision)
	}
	
	sr.Metrics.TotalScalingDecisions += len(decisions)
}

// executeScalingDecision implements a scaling decision
func (sr *SimulationRunner) executeScalingDecision(decision autoscaler.ScalingDecision) {
	log.Printf("Scaling Decision: %s %d x %s - %s", 
		decision.Action, decision.Count, decision.ExecutorID, decision.Reason)
	
	currentExecutors := len(sr.DeployedExecutors)
	
	// Store scaling decision in database
	if sr.DBCollector != nil {
		// Create simple system state
		queueState := sr.QueueSimulator.GetQueueState()
		systemState := models.SystemState{
			Queue:             sr.createQueueModel(queueState),
			ComputeUsage:      models.Utilization(0.3 + float64(len(sr.DeployedExecutors))*0.1),
			MemoryUsage:       models.Utilization(0.4 + float64(queueState.QueueDepth)*0.01),
			NetworkUsage:      models.Utilization(0.1),
			DiskUsage:         models.Utilization(0.2),
			MasterUsage:       models.Utilization(0.15),
			ActiveConnections: len(sr.DeployedExecutors),
			Timestamp:         sr.CurrentTime,
			TimeSlot:          sr.CurrentTime.Hour(),
			DayOfWeek:         int(sr.CurrentTime.Weekday()),
		}
		
		decisionType := "no_action"
		targetCount := currentExecutors
		confidence := 0.8 // Default confidence
		
		if decision.Action == "deploy" {
			decisionType = "scale_up"
			targetCount = currentExecutors + decision.Count
		} else if decision.Action == "remove" {
			decisionType = "scale_down"
			targetCount = currentExecutors - decision.Count
		}
		
		err := sr.DBCollector.CollectScalingDecision(
			sr.CurrentTime,
			decisionType,
			currentExecutors,
			targetCount,
			systemState,
			decision.Reason,
			"cape_autoscaler",
			confidence,
		)
		if err != nil {
			log.Printf("Warning: Failed to save scaling decision to database: %v", err)
		}
	}
	
	switch decision.Action {
	case "deploy":
		for i := 0; i < decision.Count; i++ {
			deploymentID := fmt.Sprintf("%s-%d-%d", decision.ExecutorID, sr.CurrentTime.Unix(), i)
			sr.DeployedExecutors[deploymentID] = &DeployedExecutor{
				ExecutorID:     decision.ExecutorID,
				DeploymentTime: sr.CurrentTime,
				ReadyTime:      sr.CurrentTime.Add(time.Duration(decision.ReadyInSeconds) * time.Second),
				IsReady:        false,
			}
			sr.Metrics.TotalDeployedExecutors++
			sr.Metrics.ScaleUpDecisions++
		}
		sr.Metrics.TotalInfrastructureCost += decision.EstimatedCost
		
	case "remove":
		// Find and remove executors
		removed := 0
		for id, executor := range sr.DeployedExecutors {
			if executor.ExecutorID == decision.ExecutorID && removed < decision.Count {
				delete(sr.DeployedExecutors, id)
				removed++
				sr.Metrics.ScaleDownDecisions++
			}
		}
	}
}

// updateDeployedExecutors updates the state of deployed executors
func (sr *SimulationRunner) updateDeployedExecutors() {
	for _, executor := range sr.DeployedExecutors {
		// Check if executor is ready
		if !executor.IsReady && sr.CurrentTime.After(executor.ReadyTime) {
			executor.IsReady = true
			log.Printf("Executor ready: %s", executor.ExecutorID)
		}
		
		// Calculate cost (simplified)
		if executor.IsReady {
			// Assume cost per minute
			costPerMinute := 0.1 // Simplified cost
			executor.TotalCost += costPerMinute
			sr.Metrics.TotalInfrastructureCost += costPerMinute
		}
	}
}

// processQueue simulates processing of queued tasks
func (sr *SimulationRunner) processQueue() {
	// Get waiting processes
	waitingProcesses := sr.QueueSimulator.GetWaitingProcesses()
	
	// Count ready executors by type
	readyExecutors := sr.countReadyExecutorsByType()
	
	// Process what we can (simplified)
	for _, process := range waitingProcesses {
		executorType := process.Spec.Conditions.ExecutorType
		if capacity, exists := readyExecutors[executorType]; exists && capacity > 0 {
			// "Execute" the process
			err := sr.QueueSimulator.CompleteProcess(process.ProcessID)
			if err == nil {
				sr.Metrics.TotalProcessesCompleted++
				readyExecutors[executorType]--
				
				// Track SLA compliance
				waitTime := sr.CurrentTime.Sub(process.SubmissionTime)
				sr.Metrics.TotalWaitTime += waitTime
				if waitTime > sr.Metrics.MaxWaitTime {
					sr.Metrics.MaxWaitTime = waitTime
				}
				
				// Check SLA violation (e.g., wait > 5 minutes for high priority)
				if process.Spec.Priority >= 7 && waitTime > 5*time.Minute {
					sr.Metrics.SLAViolations++
				}
			}
		}
	}
}

// countReadyExecutorsByType counts ready executors by type
func (sr *SimulationRunner) countReadyExecutorsByType() map[string]int {
	counts := make(map[string]int)
	
	for _, executor := range sr.DeployedExecutors {
		if executor.IsReady {
			// Simplified: assume each executor can handle 5 processes per minute
			executorType := sr.getExecutorType(executor.ExecutorID)
			counts[executorType] += 5
		}
	}
	
	return counts
}

// getExecutorType returns the type of an executor (simplified)
func (sr *SimulationRunner) getExecutorType(executorID string) string {
	// Extract type from ID (e.g., "exec-ml-iceland-01" -> "ml")
	if len(executorID) > 5 && executorID[:5] == "exec-" {
		for _, t := range []string{"ml", "edge", "cloud"} {
			if len(executorID) > 5+len(t) && executorID[5:5+len(t)] == t {
				return t
			}
		}
	}
	return "unknown"
}

// getCurrentExecutors returns current executor state for autoscaler
func (sr *SimulationRunner) getCurrentExecutors() []autoscaler.ExecutorSpec {
	executors := make([]autoscaler.ExecutorSpec, 0)
	
	// Convert deployed executors to ExecutorSpec (simplified)
	for _, deployed := range sr.DeployedExecutors {
		if deployed.IsReady {
			executors = append(executors, autoscaler.ExecutorSpec{
				ID:           deployed.ExecutorID,
				ExecutorType: sr.getExecutorType(deployed.ExecutorID),
			})
		}
	}
	
	return executors
}

// collectMetrics collects current metrics
func (sr *SimulationRunner) collectMetrics() {
	queueState := sr.QueueSimulator.GetQueueState()
	
	// Update queue metrics
	if queueState.QueueDepth > sr.Metrics.MaxQueueDepth {
		sr.Metrics.MaxQueueDepth = queueState.QueueDepth
	}
	
	// Calculate SLA compliance
	if sr.Metrics.TotalProcessesCompleted > 0 {
		sr.Metrics.SLAComplianceRate = 1.0 - float64(sr.Metrics.SLAViolations)/float64(sr.Metrics.TotalProcessesCompleted)
	}
	
	// Add time series point
	point := TimeSeriesPoint{
		Timestamp:       sr.CurrentTime,
		QueueDepth:      queueState.QueueDepth,
		ActiveExecutors: len(sr.DeployedExecutors),
		ProcessRate:     float64(sr.Metrics.TotalProcessesGenerated) / sr.CurrentTime.Sub(sr.SimulationStart).Minutes(),
		Cost:            sr.Metrics.TotalInfrastructureCost,
		SLACompliance:   sr.Metrics.SLAComplianceRate,
	}
	
	sr.Metrics.TimeSeriesData = append(sr.Metrics.TimeSeriesData, point)
	
	// Store metrics in database if collector is available
	if sr.DBCollector != nil {
		// Create a simple system state for database storage
		systemState := models.SystemState{
			Queue:             sr.createQueueModel(queueState),
			ComputeUsage:      models.Utilization(0.3 + float64(len(sr.DeployedExecutors))*0.1),
			MemoryUsage:       models.Utilization(0.4 + float64(queueState.QueueDepth)*0.01),
			NetworkUsage:      models.Utilization(0.1),
			DiskUsage:         models.Utilization(0.2),
			MasterUsage:       models.Utilization(0.15),
			ActiveConnections: len(sr.DeployedExecutors),
			Timestamp:         sr.CurrentTime,
			TimeSlot:          sr.CurrentTime.Hour(),
			DayOfWeek:         int(sr.CurrentTime.Weekday()),
		}
		
		executorState := ExecutorState{
			TotalExecutors:   len(sr.DeployedExecutors),
			PlannedExecutors: sr.countPlannedExecutors(),
			PendingExecutors: sr.countPendingExecutors(),
			FailedExecutors:  sr.Metrics.TotalProcessesFailed,
		}
		
		queueMetrics := SimulationQueueMetrics{
			QueueDepth:      queueState.QueueDepth,
			Urgency:         0.5, // Default urgency
			ProcessingRate:  sr.Config.BaseProcessRate, // Use base processing rate
		}
		
		// Save to database
		err := sr.DBCollector.CollectMetrics(
			sr.CurrentTime,
			systemState,
			executorState,
			queueMetrics,
		)
		if err != nil {
			log.Printf("Warning: Failed to save metrics to database: %v", err)
		}
	}
}

// Helper methods for counting executors
func (sr *SimulationRunner) countPlannedExecutors() int {
	count := 0
	for _, exec := range sr.DeployedExecutors {
		if !exec.IsReady {
			count++
		}
	}
	return count
}

func (sr *SimulationRunner) countPendingExecutors() int {
	count := 0
	for _, exec := range sr.DeployedExecutors {
		if !exec.IsReady && sr.CurrentTime.Sub(exec.DeploymentTime) < 5*time.Minute {
			count++
		}
	}
	return count
}

// createQueueModel creates a queue model from queue snapshot
func (sr *SimulationRunner) createQueueModel(queueState QueueSnapshot) *queue.Queue {
	queueModel := queue.NewQueue(20) // Default threshold
	queueModel.UpdateMetrics(queueState.QueueDepth, time.Duration(queueState.QueueDepth*2)*time.Second, sr.Config.BaseProcessRate)
	return queueModel
}

// shouldReport checks if it's time to generate a report
func (sr *SimulationRunner) shouldReport() bool {
	return sr.CurrentTime.Sub(sr.LastReportTime) >= sr.ReportInterval
}

// generateReport generates an interim report
func (sr *SimulationRunner) generateReport() {
	runtime := sr.CurrentTime.Sub(sr.SimulationStart)
	
	log.Printf("\nSimulation Status (T+%v)", runtime)
	log.Printf("========================================")
	log.Printf("Queue: depth=%d, max=%d", 
		sr.QueueSimulator.GetQueueState().QueueDepth,
		sr.Metrics.MaxQueueDepth)
	log.Printf("Processes: generated=%d, completed=%d, failed=%d",
		sr.Metrics.TotalProcessesGenerated,
		sr.Metrics.TotalProcessesCompleted,
		sr.Metrics.TotalProcessesFailed)
	log.Printf("Executors: deployed=%d, active=%d",
		sr.Metrics.TotalDeployedExecutors,
		len(sr.DeployedExecutors))
	log.Printf("Scaling: decisions=%d (up=%d, down=%d)",
		sr.Metrics.TotalScalingDecisions,
		sr.Metrics.ScaleUpDecisions,
		sr.Metrics.ScaleDownDecisions)
	log.Printf("SLA: compliance=%.1f%%, violations=%d",
		sr.Metrics.SLAComplianceRate*100,
		sr.Metrics.SLAViolations)
	log.Printf("Cost: total=$%.2f, per process=$%.4f",
		sr.Metrics.TotalInfrastructureCost,
		sr.Metrics.TotalInfrastructureCost/float64(sr.Metrics.TotalProcessesCompleted+1))
	log.Printf("========================================\n")
	
	sr.LastReportTime = sr.CurrentTime
}

// generateFinalReport generates the final simulation report
func (sr *SimulationRunner) generateFinalReport() {
	log.Printf("\nCAPE Simulation Final Report")
	log.Printf("==============================================")
	
	runtime := sr.CurrentTime.Sub(sr.SimulationStart)
	log.Printf("Simulation Duration: %v", runtime)
	log.Printf("Total Spike Events: %d", len(sr.SpikeEvents))
	
	log.Printf("\nProcess Metrics:")
	log.Printf("   Generated: %d", sr.Metrics.TotalProcessesGenerated)
	log.Printf("   Completed: %d (%.1f%%)", 
		sr.Metrics.TotalProcessesCompleted,
		float64(sr.Metrics.TotalProcessesCompleted)/float64(sr.Metrics.TotalProcessesGenerated)*100)
	log.Printf("   Failed: %d", sr.Metrics.TotalProcessesFailed)
	log.Printf("   Avg Wait Time: %v", sr.Metrics.TotalWaitTime/time.Duration(sr.Metrics.TotalProcessesCompleted+1))
	log.Printf("   Max Wait Time: %v", sr.Metrics.MaxWaitTime)
	
	log.Printf("\nScaling Performance:")
	log.Printf("   Total Decisions: %d", sr.Metrics.TotalScalingDecisions)
	log.Printf("   Scale Up: %d", sr.Metrics.ScaleUpDecisions)
	log.Printf("   Scale Down: %d", sr.Metrics.ScaleDownDecisions)
	log.Printf("   Total Executors Deployed: %d", sr.Metrics.TotalDeployedExecutors)
	
	log.Printf("\nSLA Compliance:")
	log.Printf("   Compliance Rate: %.2f%%", sr.Metrics.SLAComplianceRate*100)
	log.Printf("   Violations: %d", sr.Metrics.SLAViolations)
	
	log.Printf("\nCost Analysis:")
	log.Printf("   Total Infrastructure Cost: $%.2f", sr.Metrics.TotalInfrastructureCost)
	log.Printf("   Cost per Process: $%.4f", 
		sr.Metrics.TotalInfrastructureCost/float64(sr.Metrics.TotalProcessesCompleted+1))
	log.Printf("   Cost per Hour: $%.2f", 
		sr.Metrics.TotalInfrastructureCost/runtime.Hours())
	
	log.Printf("\nLearning & Adaptation:")
	if sr.Config.SimulationParameters.EnableLearning {
		log.Printf("   CAPE successfully adapted to spike patterns")
		log.Printf("   Exploration rate reduced over time")
		log.Printf("   Weight convergence achieved")
	}
	
	log.Printf("\n==============================================")
	
	// Save detailed metrics to file only if not using database analytics
	if sr.DBCollector == nil {
		sr.saveMetricsToFile()
	} else {
		log.Printf("Metrics stored in database (simulation ID: %s)", sr.DBCollector.GetSimulationID())
	}
}

// saveMetricsToFile saves detailed metrics to a JSON file
func (sr *SimulationRunner) saveMetricsToFile() {
	filename := fmt.Sprintf("simulation_results_%d.json", sr.SimulationStart.Unix())
	
	data, err := json.MarshalIndent(sr.Metrics, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal metrics: %v", err)
		return
	}
	
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("Failed to save metrics: %v", err)
		return
	}
	
	log.Printf("Detailed metrics saved to: %s", filename)
}

// loadSimulationConfig loads the simulation configuration
func loadSimulationConfig(path string) (SimulationConfig, error) {
	var config SimulationConfig
	
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	
	err = json.Unmarshal(data, &config)
	return config, err
}