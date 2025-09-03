package simulator

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/colonyos"
)

// ComputeSimulator simulates realistic compute workloads and system dynamics
type ComputeSimulator struct {
	// Configuration
	config        *SimulatorConfig
	orchestrator  *colonyos.ZeroHardcodeOrchestrator
	
	// State management
	isRunning     bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	
	// Workload simulation
	activeWorkloads map[string]*WorkloadExecution
	completedWorkloads []WorkloadExecution
	
	// Resource simulation
	executorStates map[string]*ExecutorState
	
	// Statistics
	stats         SimulationStats
	startTime     time.Time
}

// SimulatorConfig configures the compute simulator
type SimulatorConfig struct {
	// Workload generation
	WorkloadArrivalRate    float64       `json:"workload_arrival_rate"`    // processes per second
	WorkloadTypes         []WorkloadType `json:"workload_types"`          // types of workloads to simulate
	WorkloadDuration      Duration       `json:"workload_duration"`       // how long to run simulation
	
	// Resource dynamics
	ResourceFluctuation   bool          `json:"resource_fluctuation"`    // simulate resource changes
	NetworkLatencyNoise   bool          `json:"network_latency_noise"`   // simulate network variations
	ExecutorFailures      bool          `json:"executor_failures"`       // simulate executor failures
	
	// Data patterns
	DataLocalityPattern   string        `json:"data_locality_pattern"`   // "random", "clustered", "sequential"
	SeasonalPatterns      bool          `json:"seasonal_patterns"`       // simulate time-based patterns
	
	// Realism factors
	CacheEffects         bool          `json:"cache_effects"`           // simulate caching benefits
	WarmupOverhead       bool          `json:"warmup_overhead"`         // cold start penalties
	QueueingEffects      bool          `json:"queueing_effects"`        // realistic queueing behavior
}

// WorkloadType defines different types of compute workloads
type WorkloadType struct {
	Name              string        `json:"name"`
	Weight            float64       `json:"weight"`           // probability weight
	CPUIntensity      float64       `json:"cpu_intensity"`    // 0.0-1.0
	MemoryIntensity   float64       `json:"memory_intensity"` // 0.0-1.0
	IOIntensity       float64       `json:"io_intensity"`     // 0.0-1.0
	Duration          Duration      `json:"duration"`         // execution time range
	DataSize          DataSize      `json:"data_size"`        // input/output data size
	LatencySensitive  bool          `json:"latency_sensitive"`
	Parallelizable    bool          `json:"parallelizable"`
	CacheAffinitive   bool          `json:"cache_affinitive"`
}

// Duration represents a time range
type Duration struct {
	Min time.Duration `json:"min"`
	Max time.Duration `json:"max"`
}

// DataSize represents data size range
type DataSize struct {
	MinMB float64 `json:"min_mb"`
	MaxMB float64 `json:"max_mb"`
}

// WorkloadExecution represents an active workload execution
type WorkloadExecution struct {
	ID               string        `json:"id"`
	Type             WorkloadType  `json:"type"`
	ProcessID        string        `json:"process_id"`
	ExecutorID       string        `json:"executor_id"`
	StartTime        time.Time     `json:"start_time"`
	EstimatedEndTime time.Time     `json:"estimated_end_time"`
	ActualEndTime    *time.Time    `json:"actual_end_time,omitempty"`
	Progress         float64       `json:"progress"`       // 0.0-1.0
	ResourceUsage    ResourceUsage `json:"resource_usage"`
	Status           WorkloadStatus `json:"status"`
}

// ExecutorState simulates real executor resource usage
type ExecutorState struct {
	ExecutorID        string        `json:"executor_id"`
	CPUUsage          float64       `json:"cpu_usage"`          // 0.0-1.0
	MemoryUsage       float64       `json:"memory_usage"`       // 0.0-1.0
	NetworkUsage      float64       `json:"network_usage"`      // 0.0-1.0
	StorageUsage      float64       `json:"storage_usage"`      // 0.0-1.0
	Temperature       float64       `json:"temperature"`        // Celsius (for thermal throttling)
	LastUpdate        time.Time     `json:"last_update"`
	IsHealthy         bool          `json:"is_healthy"`
	ActiveWorkloads   []string      `json:"active_workloads"`
}

// ResourceUsage tracks actual resource consumption
type ResourceUsage struct {
	CPUPercent        float64 `json:"cpu_percent"`
	MemoryMB          float64 `json:"memory_mb"`
	NetworkMBps       float64 `json:"network_mbps"`
	StorageMBps       float64 `json:"storage_mbps"`
	GPUPercent        float64 `json:"gpu_percent"`
	CacheHitRate      float64 `json:"cache_hit_rate"`
}

// WorkloadStatus represents workload execution status
type WorkloadStatus string

const (
	WorkloadStatusQueued     WorkloadStatus = "queued"
	WorkloadStatusStarting   WorkloadStatus = "starting"
	WorkloadStatusRunning    WorkloadStatus = "running"
	WorkloadStatusCompleted  WorkloadStatus = "completed"
	WorkloadStatusFailed     WorkloadStatus = "failed"
	WorkloadStatusThrottled  WorkloadStatus = "throttled"
)

// SimulationStats tracks simulation performance
type SimulationStats struct {
	TotalWorkloads        int           `json:"total_workloads"`
	CompletedWorkloads    int           `json:"completed_workloads"`
	FailedWorkloads       int           `json:"failed_workloads"`
	AvgExecutionTime      time.Duration `json:"avg_execution_time"`
	AvgQueueTime          time.Duration `json:"avg_queue_time"`
	ResourceUtilization   map[string]float64 `json:"resource_utilization"`
	ThroughputPerSecond   float64       `json:"throughput_per_second"`
	CapeDecisionAccuracy  float64       `json:"cape_decision_accuracy"`
	DataLocalityScore     float64       `json:"data_locality_score"`
	CostEfficiency        float64       `json:"cost_efficiency"`
	EnergyEfficiency      float64       `json:"energy_efficiency"`
}

// NewComputeSimulator creates a new compute workload simulator
func NewComputeSimulator(config *SimulatorConfig, orchestrator *colonyos.ZeroHardcodeOrchestrator) *ComputeSimulator {
	return &ComputeSimulator{
		config:             config,
		orchestrator:       orchestrator,
		stopChan:           make(chan struct{}),
		activeWorkloads:    make(map[string]*WorkloadExecution),
		executorStates:     make(map[string]*ExecutorState),
		stats:              SimulationStats{ResourceUtilization: make(map[string]float64)},
		startTime:          time.Now(),
	}
}

// Start begins the compute simulation
func (cs *ComputeSimulator) Start(ctx context.Context) error {
	cs.mu.Lock()
	if cs.isRunning {
		cs.mu.Unlock()
		return fmt.Errorf("simulator is already running")
	}
	cs.isRunning = true
	cs.mu.Unlock()

	log.Printf("Starting Compute Workload Simulator")
	log.Printf("Workload arrival rate: %.2f processes/second", cs.config.WorkloadArrivalRate)
	log.Printf("Simulation duration: %v", cs.config.WorkloadDuration.Max)

	// Initialize executor states
	cs.initializeExecutorStates()

	// Start simulation loops
	cs.wg.Add(4)
	go cs.workloadGenerationLoop(ctx)
	go cs.workloadExecutionLoop(ctx)
	go cs.resourceSimulationLoop(ctx)
	go cs.monitoringLoop(ctx)

	return nil
}

// Stop stops the compute simulation
func (cs *ComputeSimulator) Stop() error {
	cs.mu.Lock()
	if !cs.isRunning {
		cs.mu.Unlock()
		return fmt.Errorf("simulator is not running")
	}
	cs.isRunning = false
	cs.mu.Unlock()

	log.Printf("Stopping Compute Workload Simulator...")
	close(cs.stopChan)
	cs.wg.Wait()
	
	// Print final statistics
	cs.printFinalStats()
	
	return nil
}

// workloadGenerationLoop generates new workloads based on arrival rate
func (cs *ComputeSimulator) workloadGenerationLoop(ctx context.Context) {
	defer cs.wg.Done()

	// Calculate interval between workload arrivals
	interval := time.Duration(float64(time.Second) / cs.config.WorkloadArrivalRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	workloadCounter := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-cs.stopChan:
			return
		case <-ticker.C:
			// Generate new workload
			workload := cs.generateWorkload(workloadCounter)
			workloadCounter++

			// Create ColonyOS process specification
			process := cs.createColonyOSProcess(workload)

			// Submit to orchestrator (simulated)
			cs.submitWorkloadToOrchestrator(workload, process)

			cs.mu.Lock()
			cs.stats.TotalWorkloads++
			cs.mu.Unlock()

			if workloadCounter%10 == 0 {
				log.Printf("Generated %d workloads...", workloadCounter)
			}
		}
	}
}

// generateWorkload creates a new workload based on configured patterns
func (cs *ComputeSimulator) generateWorkload(id int) *WorkloadExecution {
	// Select workload type based on weights
	workloadType := cs.selectWorkloadType()
	
	// Generate realistic duration
	duration := cs.randomDuration(workloadType.Duration)
	
	// Generate data size
	dataSizeMB := cs.randomFloat(workloadType.DataSize.MinMB, workloadType.DataSize.MaxMB)
	
	// Add seasonal patterns if enabled
	if cs.config.SeasonalPatterns {
		duration = cs.applySeasonalPattern(duration)
	}

	workload := &WorkloadExecution{
		ID:               fmt.Sprintf("workload-%d", id),
		Type:             workloadType,
		ProcessID:        fmt.Sprintf("proc-%d-%d", id, time.Now().Unix()),
		StartTime:        time.Now(),
		EstimatedEndTime: time.Now().Add(duration),
		Progress:         0.0,
		Status:           WorkloadStatusQueued,
		ResourceUsage: ResourceUsage{
			CPUPercent:   workloadType.CPUIntensity * 100,
			MemoryMB:     dataSizeMB * workloadType.MemoryIntensity,
			NetworkMBps:  dataSizeMB / float64(duration.Seconds()),
		},
	}

	return workload
}

// workloadExecutionLoop simulates actual workload execution
func (cs *ComputeSimulator) workloadExecutionLoop(ctx context.Context) {
	defer cs.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // Update every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.updateActiveWorkloads()
		}
	}
}

// updateActiveWorkloads simulates progress of running workloads
func (cs *ComputeSimulator) updateActiveWorkloads() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()
	
	for id, workload := range cs.activeWorkloads {
		if workload.Status != WorkloadStatusRunning {
			continue
		}

		// Calculate progress
		totalDuration := workload.EstimatedEndTime.Sub(workload.StartTime)
		elapsed := now.Sub(workload.StartTime)
		progress := float64(elapsed) / float64(totalDuration)

		// Add some randomness to execution time
		if cs.config.ResourceFluctuation {
			progress += (rand.Float64() - 0.5) * 0.1 // ±5% randomness
		}

		workload.Progress = progress

		// Check if workload is complete
		if progress >= 1.0 {
			workload.Progress = 1.0
			workload.Status = WorkloadStatusCompleted
			actualEndTime := now
			workload.ActualEndTime = &actualEndTime

			// Move to completed
			cs.completedWorkloads = append(cs.completedWorkloads, *workload)
			delete(cs.activeWorkloads, id)

			cs.stats.CompletedWorkloads++

			log.Printf("Workload %s completed on %s (took %v)", 
				workload.ID, workload.ExecutorID, actualEndTime.Sub(workload.StartTime).Round(time.Millisecond))

			// Update executor state
			cs.updateExecutorAfterWorkload(workload.ExecutorID, workload)
		}
	}
}

// resourceSimulationLoop simulates dynamic resource changes
func (cs *ComputeSimulator) resourceSimulationLoop(ctx context.Context) {
	defer cs.wg.Done()

	ticker := time.NewTicker(5 * time.Second) // Update every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.updateExecutorStates()
			cs.simulateResourceDynamics()
		}
	}
}

// updateExecutorStates simulates realistic executor resource usage
func (cs *ComputeSimulator) updateExecutorStates() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()
	
	for executorID, state := range cs.executorStates {
		// Calculate resource usage based on active workloads
		totalCPU := 0.0
		totalMemory := 0.0
		totalNetwork := 0.0

		activeCount := 0
		for _, workload := range cs.activeWorkloads {
			if workload.ExecutorID == executorID && workload.Status == WorkloadStatusRunning {
				totalCPU += workload.ResourceUsage.CPUPercent / 100.0
				totalMemory += workload.ResourceUsage.MemoryMB / 1024.0 // Convert to GB
				totalNetwork += workload.ResourceUsage.NetworkMBps
				activeCount++
			}
		}

		// Apply resource limits and contention
		state.CPUUsage = cs.applyResourceContention(totalCPU, 1.0)
		state.MemoryUsage = cs.applyResourceContention(totalMemory/16.0, 1.0) // Assume 16GB total
		state.NetworkUsage = cs.applyResourceContention(totalNetwork/100.0, 1.0) // Assume 100 Mbps

		// Simulate thermal effects
		if cs.config.ResourceFluctuation {
			state.Temperature = 40.0 + state.CPUUsage*40.0 + rand.Float64()*10.0
			
			// Thermal throttling
			if state.Temperature > 80.0 {
				state.CPUUsage *= 0.8 // Reduce performance due to heat
			}
		}

		state.LastUpdate = now
		state.IsHealthy = activeCount < 10 // Arbitrary health check
	}
}

// monitoringLoop provides real-time monitoring output
func (cs *ComputeSimulator) monitoringLoop(ctx context.Context) {
	defer cs.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Report every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cs.stopChan:
			return
		case <-ticker.C:
			cs.printCurrentStats()
		}
	}
}

// Helper functions

func (cs *ComputeSimulator) initializeExecutorStates() {
	// Initialize with some mock executors
	executors := []string{
		"executor-edge-stockholm-01",
		"executor-cloud-aws-us-east-1-01", 
		"executor-hpc-iceland-01",
	}

	for _, executorID := range executors {
		cs.executorStates[executorID] = &ExecutorState{
			ExecutorID:      executorID,
			CPUUsage:        0.0,
			MemoryUsage:     0.0,
			NetworkUsage:    0.0,
			StorageUsage:    0.0,
			Temperature:     45.0,
			LastUpdate:      time.Now(),
			IsHealthy:       true,
			ActiveWorkloads: []string{},
		}
	}
}

func (cs *ComputeSimulator) selectWorkloadType() WorkloadType {
	totalWeight := 0.0
	for _, wt := range cs.config.WorkloadTypes {
		totalWeight += wt.Weight
	}

	r := rand.Float64() * totalWeight
	cumWeight := 0.0
	
	for _, wt := range cs.config.WorkloadTypes {
		cumWeight += wt.Weight
		if r <= cumWeight {
			return wt
		}
	}
	
	return cs.config.WorkloadTypes[0] // Fallback
}

func (cs *ComputeSimulator) randomDuration(d Duration) time.Duration {
	if d.Min == d.Max {
		return d.Min
	}
	diff := d.Max - d.Min
	return d.Min + time.Duration(rand.Float64()*float64(diff))
}

func (cs *ComputeSimulator) randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (cs *ComputeSimulator) applySeasonalPattern(duration time.Duration) time.Duration {
	hour := time.Now().Hour()
	// Simulate daily patterns - slower at night, faster during day
	if hour >= 22 || hour <= 6 {
		return time.Duration(float64(duration) * 1.3) // 30% slower at night
	} else if hour >= 9 && hour <= 17 {
		return time.Duration(float64(duration) * 0.8) // 20% faster during business hours
	}
	return duration
}

func (cs *ComputeSimulator) applyResourceContention(requested, limit float64) float64 {
	if requested <= limit {
		return requested
	}
	// Apply contention penalty
	return limit * (1.0 - (requested-limit)*0.1)
}

func (cs *ComputeSimulator) createColonyOSProcess(workload *WorkloadExecution) colonyos.Process {
	return colonyos.Process{
		ID:    workload.ProcessID,
		State: colonyos.PROCESS_WAITING,
		FunctionSpec: colonyos.FunctionSpec{
			FuncName:    workload.Type.Name,
			Priority:    cs.calculatePriority(workload),
			MaxExecTime: int(workload.EstimatedEndTime.Sub(workload.StartTime).Seconds()),
			Args:        []interface{}{fmt.Sprintf("data-size-%.1fMB", workload.ResourceUsage.MemoryMB)},
			Conditions: colonyos.Conditions{
				ExecutorType: cs.selectOptimalExecutorType(workload),
			},
		},
		SubmissionTime: workload.StartTime,
	}
}

func (cs *ComputeSimulator) calculatePriority(workload *WorkloadExecution) int {
	priority := 5 // Default
	if workload.Type.LatencySensitive {
		priority += 3
	}
	if workload.Type.CPUIntensity > 0.8 {
		priority += 1
	}
	return priority
}

func (cs *ComputeSimulator) selectOptimalExecutorType(workload *WorkloadExecution) string {
	if workload.Type.LatencySensitive {
		return "edge"
	} else if workload.Type.CPUIntensity > 0.7 {
		return "hpc"
	} else {
		return "cloud"
	}
}

func (cs *ComputeSimulator) submitWorkloadToOrchestrator(workload *WorkloadExecution, process colonyos.Process) {
	// Simulate assignment to executor (simplified)
	availableExecutors := []string{"executor-edge-stockholm-01", "executor-cloud-aws-us-east-1-01", "executor-hpc-iceland-01"}
	selectedExecutor := availableExecutors[rand.Intn(len(availableExecutors))]
	
	workload.ExecutorID = selectedExecutor
	workload.Status = WorkloadStatusRunning
	workload.StartTime = time.Now()
	
	cs.mu.Lock()
	cs.activeWorkloads[workload.ID] = workload
	cs.mu.Unlock()
}

func (cs *ComputeSimulator) updateExecutorAfterWorkload(executorID string, workload *WorkloadExecution) {
	if state, exists := cs.executorStates[executorID]; exists {
		// Reduce resource usage
		state.CPUUsage = math.Max(0, state.CPUUsage-workload.ResourceUsage.CPUPercent/100.0)
		state.MemoryUsage = math.Max(0, state.MemoryUsage-workload.ResourceUsage.MemoryMB/1024.0)
	}
}

func (cs *ComputeSimulator) simulateResourceDynamics() {
	// Add some dynamic behavior if configured
	if cs.config.ResourceFluctuation {
		// Simulate random resource spikes
		if rand.Float64() < 0.1 { // 10% chance
			// Pick random executor and add resource spike
			executors := make([]string, 0, len(cs.executorStates))
			for id := range cs.executorStates {
				executors = append(executors, id)
			}
			if len(executors) > 0 {
				randomExecutor := executors[rand.Intn(len(executors))]
				cs.executorStates[randomExecutor].CPUUsage = math.Min(1.0, cs.executorStates[randomExecutor].CPUUsage + 0.2)
			}
		}
	}
}

func (cs *ComputeSimulator) printCurrentStats() {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	uptime := time.Since(cs.startTime)
	
	log.Printf("\nSimulation Stats (Uptime: %v)", uptime.Round(time.Second))
	log.Printf("=================================")
	log.Printf("Workloads: %d total, %d active, %d completed, %d failed", 
		cs.stats.TotalWorkloads, len(cs.activeWorkloads), cs.stats.CompletedWorkloads, cs.stats.FailedWorkloads)
	
	if cs.stats.CompletedWorkloads > 0 {
		successRate := float64(cs.stats.CompletedWorkloads) / float64(cs.stats.TotalWorkloads) * 100
		log.Printf("Success Rate: %.1f%%", successRate)
		
		throughput := float64(cs.stats.CompletedWorkloads) / uptime.Seconds()
		log.Printf("Throughput: %.2f workloads/second", throughput)
	}
	
	log.Printf("\nExecutor States:")
	for id, state := range cs.executorStates {
		log.Printf("  %s: CPU=%.1f%%, Mem=%.1f%%, Net=%.1f%%, Temp=%.1f°C, Healthy=%v", 
			id, state.CPUUsage*100, state.MemoryUsage*100, state.NetworkUsage*100, 
			state.Temperature, state.IsHealthy)
	}
}

func (cs *ComputeSimulator) printFinalStats() {
	uptime := time.Since(cs.startTime)
	
	log.Printf("\nFinal Simulation Results")
	log.Printf("========================")
	log.Printf("Total Runtime: %v", uptime.Round(time.Second))
	log.Printf("Workloads Generated: %d", cs.stats.TotalWorkloads)
	log.Printf("Workloads Completed: %d", cs.stats.CompletedWorkloads)
	log.Printf("Workloads Failed: %d", cs.stats.FailedWorkloads)
	
	if cs.stats.TotalWorkloads > 0 {
		successRate := float64(cs.stats.CompletedWorkloads) / float64(cs.stats.TotalWorkloads) * 100
		log.Printf("Overall Success Rate: %.1f%%", successRate)
		
		avgThroughput := float64(cs.stats.CompletedWorkloads) / uptime.Seconds()
		log.Printf("Average Throughput: %.2f workloads/second", avgThroughput)
	}
}

// GetStats returns current simulation statistics
func (cs *ComputeSimulator) GetStats() SimulationStats {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.stats
}

// GetExecutorStates returns current executor states
func (cs *ComputeSimulator) GetExecutorStates() map[string]*ExecutorState {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	states := make(map[string]*ExecutorState)
	for k, v := range cs.executorStates {
		stateCopy := *v
		states[k] = &stateCopy
	}
	return states
}