package simulation

import (
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/database"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
	"github.com/google/uuid"
)

// ExecutorState represents the state of executors in the simulation
type ExecutorState struct {
	TotalExecutors   int
	PlannedExecutors int
	PendingExecutors int
	FailedExecutors  int
}

// SimulationQueueMetrics represents queue-related metrics for database storage
type SimulationQueueMetrics struct {
	QueueDepth     int
	Urgency        float64
	ProcessingRate float64
}

// DBMetricsCollector collects metrics and stores them in the database
type DBMetricsCollector struct {
	repo         *database.Repository
	simulationID string
	buffer       []database.MetricSnapshot
	bufferSize   int
	lastFlush    time.Time
}

// NewDBMetricsCollector creates a new database metrics collector
func NewDBMetricsCollector(repo *database.Repository, simName, simDescription string) (*DBMetricsCollector, error) {
	// Create simulation record
	sim := &database.Simulation{
		ID:          uuid.New().String(),
		Name:        simName,
		Description: simDescription,
		StartTime:   time.Now(),
		Status:      "running",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if err := repo.CreateSimulation(sim); err != nil {
		return nil, fmt.Errorf("failed to create simulation: %w", err)
	}
	
	return &DBMetricsCollector{
		repo:         repo,
		simulationID: sim.ID,
		buffer:       make([]database.MetricSnapshot, 0, 100),
		bufferSize:   100,
		lastFlush:    time.Now(),
	}, nil
}

// GetSimulationID returns the simulation ID
func (dc *DBMetricsCollector) GetSimulationID() string {
	return dc.simulationID
}

// CollectMetrics collects and stores metrics in the database
func (dc *DBMetricsCollector) CollectMetrics(
	timestamp time.Time,
	systemState models.SystemState,
	executorState ExecutorState,
	queueMetrics SimulationQueueMetrics,
) error {
	snapshot := database.MetricSnapshot{
		SimulationID: dc.simulationID,
		Timestamp:    timestamp,
		
		// Queue metrics
		QueueDepth:        systemState.Queue.Depth,
		QueueVelocity:     systemState.Queue.Velocity,
		QueueAcceleration: systemState.Queue.Acceleration,
		QueueThroughput:   systemState.Queue.Throughput,
		QueueWaitTime:     systemState.Queue.WaitTime.Seconds(),
		
		// Executor metrics
		ActualExecutors:  executorState.TotalExecutors,
		PlannedExecutors: executorState.PlannedExecutors,
		PendingExecutors: executorState.PendingExecutors,
		FailedExecutors:  executorState.FailedExecutors,
		
		// System metrics
		ComputeUsage: float64(systemState.ComputeUsage),
		MemoryUsage:  float64(systemState.MemoryUsage),
		NetworkUsage: float64(systemState.NetworkUsage),
		DiskUsage:    float64(systemState.DiskUsage),
		MasterUsage:  float64(systemState.MasterUsage),
		
		// Load metrics
		SystemLoad:    systemState.GetLoadScore(),
		QueuePressure: systemState.GetQueuePressure(),
		Urgency:       queueMetrics.Urgency,
		
		CreatedAt: time.Now(),
	}
	
	// Add to buffer
	dc.buffer = append(dc.buffer, snapshot)
	
	// Flush if buffer is full or every 5 seconds
	if len(dc.buffer) >= dc.bufferSize || time.Since(dc.lastFlush) > 5*time.Second {
		return dc.flush()
	}
	
	return nil
}

// CollectScalingDecision stores a scaling decision in the database
func (dc *DBMetricsCollector) CollectScalingDecision(
	timestamp time.Time,
	decisionType string,
	fromCount, toCount int,
	systemState models.SystemState,
	reason string,
	algorithm string,
	confidence float64,
) error {
	decision := &database.ScalingDecision{
		SimulationID: dc.simulationID,
		Timestamp:    timestamp,
		
		DecisionType: decisionType,
		FromCount:    fromCount,
		ToCount:      toCount,
		Delta:        toCount - fromCount,
		
		// Decision factors
		QueueDepth:        systemState.Queue.Depth,
		QueueVelocity:     systemState.Queue.Velocity,
		QueueAcceleration: systemState.Queue.Acceleration,
		SystemLoad:        systemState.GetLoadScore(),
		Urgency:           systemState.Queue.GetUrgency(),
		
		// Reasoning
		Reason:     reason,
		Algorithm:  algorithm,
		Confidence: confidence,
		
		Success:   true, // Will be updated later if it fails
		CreatedAt: time.Now(),
	}
	
	return dc.repo.SaveScalingDecision(decision)
}

// CollectEvent stores an event in the database
func (dc *DBMetricsCollector) CollectEvent(
	timestamp time.Time,
	eventType, category, severity, message string,
	details interface{},
) error {
	detailsJSON := ""
	if details != nil {
		data, err := json.Marshal(details)
		if err == nil {
			detailsJSON = string(data)
		}
	}
	
	event := &database.Event{
		SimulationID: dc.simulationID,
		Timestamp:    timestamp,
		EventType:    eventType,
		Category:     category,
		Severity:     severity,
		Message:      message,
		Details:      detailsJSON,
		CreatedAt:    time.Now(),
	}
	
	return dc.repo.SaveEvent(event)
}

// CollectPredictionAccuracy stores prediction accuracy metrics
func (dc *DBMetricsCollector) CollectPredictionAccuracy(
	timestamp time.Time,
	horizon int,
	predictedQueue, actualQueue int,
	predictedLoad, actualLoad float64,
	predictedExecutors, actualExecutors int,
) error {
	accuracy := &database.PredictionAccuracy{
		SimulationID:      dc.simulationID,
		Timestamp:         timestamp,
		PredictionHorizon: horizon,
		
		PredictedQueueDepth: predictedQueue,
		ActualQueueDepth:    actualQueue,
		QueueDepthError:     float64(abs(predictedQueue - actualQueue)),
		
		PredictedLoad: predictedLoad,
		ActualLoad:    actualLoad,
		LoadError:     absFloat(predictedLoad - actualLoad),
		
		PredictedExecutors: predictedExecutors,
		ActualExecutors:    actualExecutors,
		ExecutorError:      float64(abs(predictedExecutors - actualExecutors)),
		
		CreatedAt: time.Now(),
	}
	
	return dc.repo.SavePredictionAccuracy(accuracy)
}

// CollectLearningMetrics stores learning algorithm metrics
func (dc *DBMetricsCollector) CollectLearningMetrics(
	timestamp time.Time,
	algorithm string,
	metrics map[string]interface{},
) error {
	learning := &database.LearningMetrics{
		SimulationID: dc.simulationID,
		Timestamp:    timestamp,
		Algorithm:    algorithm,
		CreatedAt:    time.Now(),
	}
	
	// Extract specific metrics based on algorithm
	if algorithm == "q_learning" {
		if v, ok := metrics["q_table_size"].(int); ok {
			learning.QTableSize = v
		}
		if v, ok := metrics["exploration_rate"].(float64); ok {
			learning.ExplorationRate = v
		}
		if v, ok := metrics["learning_rate"].(float64); ok {
			learning.LearningRate = v
		}
		if v, ok := metrics["convergence_rate"].(float64); ok {
			learning.ConvergenceRate = v
		}
	} else if algorithm == "thompson_sampling" {
		if v, ok := metrics["arm_count"].(int); ok {
			learning.ArmCount = v
		}
		if v, ok := metrics["best_arm_reward"].(float64); ok {
			learning.BestArmReward = v
		}
		if v, ok := metrics["total_reward"].(float64); ok {
			learning.TotalReward = v
		}
	} else if algorithm == "arima" {
		if v, ok := metrics["prediction_error"].(float64); ok {
			learning.ARIMAPredictionError = v
		}
		if v, ok := metrics["model_order"].(string); ok {
			learning.ARIMAModelOrder = v
		}
	}
	
	// General metrics
	if v, ok := metrics["accuracy_score"].(float64); ok {
		learning.AccuracyScore = v
	}
	if v, ok := metrics["recall_score"].(float64); ok {
		learning.RecallScore = v
	}
	if v, ok := metrics["f1_score"].(float64); ok {
		learning.F1Score = v
	}
	
	return dc.repo.SaveLearningMetrics(learning)
}

// flush writes buffered metrics to the database
func (dc *DBMetricsCollector) flush() error {
	if len(dc.buffer) == 0 {
		return nil
	}
	
	if err := dc.repo.BatchSaveMetrics(dc.buffer); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}
	
	dc.buffer = dc.buffer[:0] // Clear buffer
	dc.lastFlush = time.Now()
	return nil
}

// Close flushes remaining data and marks simulation as completed
func (dc *DBMetricsCollector) Close() error {
	// Flush any remaining metrics
	if err := dc.flush(); err != nil {
		return err
	}
	
	// Mark simulation as completed
	return dc.repo.EndSimulation(dc.simulationID, "completed")
}

// Helper functions
func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func absFloat(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}