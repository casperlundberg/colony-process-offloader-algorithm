package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// QueueSimulator simulates a ColonyOS process queue with priority-based processes
type QueueSimulator struct {
	mu              sync.RWMutex
	processes       []models.ColonyOSProcess
	processCounter  int
	currentTime     time.Time
	baseProcessRate float64 // Processes per minute
	random          *rand.Rand
	
	// Metrics
	totalGenerated   int
	totalCompleted   int
	totalFailed      int
	queueDepthHistory []QueueSnapshot
	
	// Data locations and their characteristics
	dataLocations   map[string]DataLocation
}

// DataLocation represents a data storage location
type DataLocation struct {
	Name        string
	Longitude   float64
	Latitude    float64
	Provider    string // "edge", "cloud", "on-premise"
	StorageGB   float64
	Datasets    []string
}

// QueueSnapshot captures queue state at a point in time
type QueueSnapshot struct {
	Timestamp        time.Time
	QueueDepth       int
	WaitingByType    map[string]int
	PriorityBreakdown map[int]int
	AverageWaitTime  time.Duration
	OldestWaitTime   time.Duration
}

// NewQueueSimulator creates a new queue simulator
func NewQueueSimulator(baseProcessRate float64) *QueueSimulator {
	return &QueueSimulator{
		processes:       make([]models.ColonyOSProcess, 0),
		processCounter:  0,
		currentTime:     time.Now(),
		baseProcessRate: baseProcessRate,
		random:          rand.New(rand.NewSource(time.Now().UnixNano())),
		queueDepthHistory: make([]QueueSnapshot, 0),
		dataLocations:   initializeDataLocations(),
	}
}

// initializeDataLocations sets up the data location map
func initializeDataLocations() map[string]DataLocation {
	return map[string]DataLocation{
		"iceland": {
			Name:      "Iceland Datacenter",
			Longitude: -21.9426,
			Latitude:  64.1466,
			Provider:  "edge",
			StorageGB: 10000,
			Datasets:  []string{"ml-models", "training-data", "research-data"},
		},
		"stockholm": {
			Name:      "Stockholm Edge Node",
			Longitude: 18.0686,
			Latitude:  59.3293,
			Provider:  "edge",
			StorageGB: 500,
			Datasets:  []string{"sensor-data", "iot-streams", "real-time-events"},
		},
		"aws_east": {
			Name:      "AWS us-east-1",
			Longitude: -77.0369,
			Latitude:  38.9072,
			Provider:  "cloud",
			StorageGB: 50000,
			Datasets:  []string{"batch-data", "archives", "logs", "backups"},
		},
	}
}

// GenerateProcessesFromSpike creates processes based on a spike event
func (qs *QueueSimulator) GenerateProcessesFromSpike(spike SpikeEvent, currentMinute int) []models.ColonyOSProcess {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	
	processes := make([]models.ColonyOSProcess, 0)
	
	// Check if we're within the spike duration
	if currentMinute < 0 || currentMinute >= len(spike.IntensityProfile) {
		return processes
	}
	
	// Get process rate for this minute
	processRate := spike.IntensityProfile[currentMinute]
	numProcesses := qs.poissonSample(processRate)
	
	for i := 0; i < numProcesses; i++ {
		process := qs.generateProcess(spike, i)
		processes = append(processes, process)
		qs.processes = append(qs.processes, process)
		qs.totalGenerated++
	}
	
	return processes
}

// GenerateBaselineProcesses creates normal background processes
func (qs *QueueSimulator) GenerateBaselineProcesses() []models.ColonyOSProcess {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	
	processes := make([]models.ColonyOSProcess, 0)
	numProcesses := qs.poissonSample(qs.baseProcessRate)
	
	for i := 0; i < numProcesses; i++ {
		process := qs.generateBaselineProcess(i)
		processes = append(processes, process)
		qs.processes = append(qs.processes, process)
		qs.totalGenerated++
	}
	
	return processes
}

// generateProcess creates a single process from spike characteristics
func (qs *QueueSimulator) generateProcess(spike SpikeEvent, index int) models.ColonyOSProcess {
	qs.processCounter++
	processID := fmt.Sprintf("proc-%s-%d", spike.ID, qs.processCounter)
	
	// Select priority from distribution
	priority := spike.PriorityDistribution[qs.random.Intn(len(spike.PriorityDistribution))]
	
	// Generate process characteristics based on executor type
	var funcName string
	var estimatedDuration time.Duration
	var estimatedMemoryGB float64
	var requiresGPU bool
	
	switch spike.ExecutorType {
	case "ml":
		funcName = qs.selectMLFunction()
		estimatedDuration = time.Duration(300+qs.random.Intn(600)) * time.Second
		estimatedMemoryGB = 8.0 + qs.random.Float64()*56.0
		requiresGPU = true
	case "edge":
		funcName = qs.selectEdgeFunction()
		estimatedDuration = time.Duration(10+qs.random.Intn(120)) * time.Second
		estimatedMemoryGB = 1.0 + qs.random.Float64()*8.0
		requiresGPU = false
	case "cloud":
		funcName = qs.selectCloudFunction()
		estimatedDuration = time.Duration(60+qs.random.Intn(300)) * time.Second
		estimatedMemoryGB = 2.0 + qs.random.Float64()*16.0
		requiresGPU = false
	default:
		funcName = "generic-compute"
		estimatedDuration = time.Duration(30+qs.random.Intn(180)) * time.Second
		estimatedMemoryGB = 2.0 + qs.random.Float64()*8.0
		requiresGPU = false
	}
	
	// Calculate data movement cost
	dataSizeGB := spike.DataSizeGB / float64(spike.ProcessesGenerated) // Per-process data
	
	process := models.ColonyOSProcess{
		ProcessID:      processID,
		SubmissionTime: qs.currentTime,
		State:          models.ProcessStateWaiting,
		Spec: models.ColonyOSProcessSpec{
			FuncName: funcName,
			Priority: priority,
			Args:     []string{fmt.Sprintf("--data-location=%s", spike.DataLocation)},
			MaxExecTime: int(estimatedDuration.Seconds()),
			Conditions: models.ColonyOSConditions{
				ExecutorType: spike.ExecutorType,
				Dependencies: []string{},
				MinCPU:       fmt.Sprintf("%dm", 500+qs.random.Intn(3500)),
				MinMemory:    fmt.Sprintf("%dGi", int(estimatedMemoryGB)),
				RequiredGPU:  requiresGPU,
			},
			EstimatedDuration: estimatedDuration,
			DataRequirements: &models.DataRequirements{
				DataSizeGB:     dataSizeGB,
				DataSensitive:  false,
			},
			ResourceHints: &models.ResourceHints{
				CPUIntensive:      true,
				MemoryIntensive:   estimatedMemoryGB > 16,
				GPURequired:       requiresGPU,
				LatencySensitive:  spike.ExecutorType == "edge",
				CostSensitive:     priority < 7,
			},
		},
	}
	
	return process
}

// generateBaselineProcess creates a normal background process
func (qs *QueueSimulator) generateBaselineProcess(index int) models.ColonyOSProcess {
	qs.processCounter++
	processID := fmt.Sprintf("proc-baseline-%d", qs.processCounter)
	
	// Random executor type for baseline
	executorTypes := []string{"edge", "cloud", "ml"}
	executorType := executorTypes[qs.random.Intn(len(executorTypes))]
	
	// Lower priority for baseline processes
	priority := 3 + qs.random.Intn(4) // 3-6
	
	// Random data location
	locations := []string{"iceland", "stockholm", "aws_east"}
	_ = locations[qs.random.Intn(len(locations))] // Not used for now
	
	process := models.ColonyOSProcess{
		ProcessID:      processID,
		SubmissionTime: qs.currentTime,
		State:          models.ProcessStateWaiting,
		Spec: models.ColonyOSProcessSpec{
			FuncName: "baseline-compute",
			Priority: priority,
			MaxExecTime: 300,
			Conditions: models.ColonyOSConditions{
				ExecutorType: executorType,
				Dependencies: []string{},
				MinCPU:       fmt.Sprintf("%dm", 500+qs.random.Intn(2000)),
				MinMemory:    fmt.Sprintf("%dGi", 2+qs.random.Intn(8)),
			},
			DataRequirements: &models.DataRequirements{
				DataSizeGB:     qs.random.Float64() * 10,
				DataSensitive:  false,
			},
		},
	}
	
	return process
}

// selectMLFunction returns a random ML function name
func (qs *QueueSimulator) selectMLFunction() string {
	functions := []string{
		"train-model",
		"inference-batch",
		"hyperparameter-tuning",
		"feature-extraction",
		"model-evaluation",
		"data-preprocessing",
	}
	return functions[qs.random.Intn(len(functions))]
}

// selectEdgeFunction returns a random edge function name
func (qs *QueueSimulator) selectEdgeFunction() string {
	functions := []string{
		"sensor-processing",
		"real-time-analytics",
		"event-detection",
		"stream-aggregation",
		"edge-inference",
		"data-filtering",
	}
	return functions[qs.random.Intn(len(functions))]
}

// selectCloudFunction returns a random cloud function name
func (qs *QueueSimulator) selectCloudFunction() string {
	functions := []string{
		"batch-processing",
		"data-transformation",
		"report-generation",
		"backup-sync",
		"log-analysis",
		"archive-compression",
	}
	return functions[qs.random.Intn(len(functions))]
}

// poissonSample generates a Poisson-distributed random number
func (qs *QueueSimulator) poissonSample(lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	
	// Using Knuth's algorithm for small lambda
	if lambda < 30 {
		L := math.Exp(-lambda)
		k := 0
		p := 1.0
		
		for p > L {
			k++
			p *= qs.random.Float64()
		}
		return k - 1
	}
	
	// For larger lambda, use normal approximation
	return int(qs.random.NormFloat64()*math.Sqrt(lambda) + lambda)
}

// GetQueueState returns current queue state
func (qs *QueueSimulator) GetQueueState() QueueSnapshot {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	
	snapshot := QueueSnapshot{
		Timestamp:         qs.currentTime,
		QueueDepth:        0,
		WaitingByType:     make(map[string]int),
		PriorityBreakdown: make(map[int]int),
	}
	
	var totalWait time.Duration
	var oldestWait time.Duration
	waitingCount := 0
	
	for _, process := range qs.processes {
		if process.State == models.ProcessStateWaiting {
			snapshot.QueueDepth++
			snapshot.WaitingByType[process.Spec.Conditions.ExecutorType]++
			snapshot.PriorityBreakdown[process.Spec.Priority]++
			
			waitTime := qs.currentTime.Sub(process.SubmissionTime)
			totalWait += waitTime
			waitingCount++
			
			if waitTime > oldestWait {
				oldestWait = waitTime
			}
		}
	}
	
	if waitingCount > 0 {
		snapshot.AverageWaitTime = totalWait / time.Duration(waitingCount)
	}
	snapshot.OldestWaitTime = oldestWait
	
	return snapshot
}

// GetWaitingProcesses returns all processes in waiting state
func (qs *QueueSimulator) GetWaitingProcesses() []models.ColonyOSProcess {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	
	waiting := make([]models.ColonyOSProcess, 0)
	for _, process := range qs.processes {
		if process.State == models.ProcessStateWaiting {
			waiting = append(waiting, process)
		}
	}
	return waiting
}

// CompleteProcess marks a process as completed
func (qs *QueueSimulator) CompleteProcess(processID string) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	
	for i, process := range qs.processes {
		if process.ProcessID == processID {
			qs.processes[i].State = models.ProcessStateSuccessful
			endTime := qs.currentTime
			qs.processes[i].EndTime = &endTime
			qs.totalCompleted++
			return nil
		}
	}
	return fmt.Errorf("process %s not found", processID)
}

// FailProcess marks a process as failed
func (qs *QueueSimulator) FailProcess(processID string) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	
	for i, process := range qs.processes {
		if process.ProcessID == processID {
			qs.processes[i].State = models.ProcessStateFailed
			endTime := qs.currentTime
			qs.processes[i].EndTime = &endTime
			qs.totalFailed++
			return nil
		}
	}
	return fmt.Errorf("process %s not found", processID)
}

// AdvanceTime moves the simulation time forward
func (qs *QueueSimulator) AdvanceTime(duration time.Duration) {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	qs.currentTime = qs.currentTime.Add(duration)
}

// GetMetrics returns simulation metrics
func (qs *QueueSimulator) GetMetrics() QueueMetrics {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	
	return QueueMetrics{
		TotalGenerated:  qs.totalGenerated,
		TotalCompleted:  qs.totalCompleted,
		TotalFailed:     qs.totalFailed,
		CurrentQueueDepth: qs.GetQueueState().QueueDepth,
		SuccessRate:     float64(qs.totalCompleted) / float64(qs.totalGenerated+1) * 100,
	}
}

// QueueMetrics contains queue performance metrics
type QueueMetrics struct {
	TotalGenerated    int
	TotalCompleted    int
	TotalFailed       int
	CurrentQueueDepth int
	SuccessRate       float64
}

// CleanupOldProcesses removes completed/failed processes older than retention period
func (qs *QueueSimulator) CleanupOldProcesses(retentionPeriod time.Duration) int {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	
	cutoffTime := qs.currentTime.Add(-retentionPeriod)
	newProcesses := make([]models.ColonyOSProcess, 0)
	removed := 0
	
	for _, process := range qs.processes {
		// Keep waiting/running processes and recent completed/failed ones
		if process.State == models.ProcessStateWaiting || 
		   process.State == models.ProcessStateRunning ||
		   process.EndTime.After(cutoffTime) {
			newProcesses = append(newProcesses, process)
		} else {
			removed++
		}
	}
	
	qs.processes = newProcesses
	return removed
}