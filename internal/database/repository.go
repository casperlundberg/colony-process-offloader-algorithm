package database

import (
	"fmt"
	"time"
	
	"gorm.io/gorm"
)

// Repository provides data access methods
type Repository struct {
	db *DB
}

// NewRepository creates a new repository
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// CreateSimulation creates a new simulation record
func (r *Repository) CreateSimulation(sim *Simulation) error {
	return r.db.Create(sim).Error
}

// GetSimulation retrieves a simulation by ID
func (r *Repository) GetSimulation(id string) (*Simulation, error) {
	var sim Simulation
	err := r.db.First(&sim, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &sim, nil
}

// ListSimulations lists all simulations
func (r *Repository) ListSimulations() ([]Simulation, error) {
	var sims []Simulation
	err := r.db.Order("created_at DESC").Find(&sims).Error
	return sims, err
}

// UpdateSimulation updates a simulation record
func (r *Repository) UpdateSimulation(sim *Simulation) error {
	return r.db.Save(sim).Error
}

// EndSimulation marks a simulation as completed
func (r *Repository) EndSimulation(id string, status string) error {
	now := time.Now()
	return r.db.Model(&Simulation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"end_time": now,
			"status":   status,
		}).Error
}

// SaveMetricSnapshot saves a metric snapshot
func (r *Repository) SaveMetricSnapshot(snapshot *MetricSnapshot) error {
	return r.db.Create(snapshot).Error
}

// GetMetricSnapshots retrieves metric snapshots for a simulation
func (r *Repository) GetMetricSnapshots(simulationID string, limit int) ([]MetricSnapshot, error) {
	var snapshots []MetricSnapshot
	query := r.db.Where("simulation_id = ?", simulationID).Order("timestamp DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&snapshots).Error
	return snapshots, err
}

// GetMetricSnapshotsInRange retrieves metrics within a time range
func (r *Repository) GetMetricSnapshotsInRange(simulationID string, start, end time.Time) ([]MetricSnapshot, error) {
	var snapshots []MetricSnapshot
	err := r.db.Where("simulation_id = ? AND timestamp BETWEEN ? AND ?", simulationID, start, end).
		Order("timestamp ASC").
		Find(&snapshots).Error
	return snapshots, err
}

// SaveScalingDecision saves a scaling decision
func (r *Repository) SaveScalingDecision(decision *ScalingDecision) error {
	return r.db.Create(decision).Error
}

// GetScalingDecisions retrieves scaling decisions for a simulation
func (r *Repository) GetScalingDecisions(simulationID string) ([]ScalingDecision, error) {
	var decisions []ScalingDecision
	err := r.db.Where("simulation_id = ?", simulationID).
		Order("timestamp DESC").
		Find(&decisions).Error
	return decisions, err
}

// SaveEvent saves an event
func (r *Repository) SaveEvent(event *Event) error {
	return r.db.Create(event).Error
}

// GetEvents retrieves events for a simulation
func (r *Repository) GetEvents(simulationID string, eventType string) ([]Event, error) {
	var events []Event
	query := r.db.Where("simulation_id = ?", simulationID)
	
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	
	err := query.Order("timestamp DESC").Find(&events).Error
	return events, err
}

// SavePredictionAccuracy saves prediction accuracy metrics
func (r *Repository) SavePredictionAccuracy(accuracy *PredictionAccuracy) error {
	return r.db.Create(accuracy).Error
}

// GetPredictionAccuracy retrieves prediction accuracy metrics
func (r *Repository) GetPredictionAccuracy(simulationID string) ([]PredictionAccuracy, error) {
	var accuracies []PredictionAccuracy
	err := r.db.Where("simulation_id = ?", simulationID).
		Order("timestamp DESC").
		Find(&accuracies).Error
	return accuracies, err
}

// SaveLearningMetrics saves learning algorithm metrics
func (r *Repository) SaveLearningMetrics(metrics *LearningMetrics) error {
	return r.db.Create(metrics).Error
}

// GetLearningMetrics retrieves learning metrics for a simulation
func (r *Repository) GetLearningMetrics(simulationID string, algorithm string) ([]LearningMetrics, error) {
	var metrics []LearningMetrics
	query := r.db.Where("simulation_id = ?", simulationID)
	
	if algorithm != "" {
		query = query.Where("algorithm = ?", algorithm)
	}
	
	err := query.Order("timestamp DESC").Find(&metrics).Error
	return metrics, err
}

// GetSimulationSummary gets aggregated stats for a simulation
func (r *Repository) GetSimulationSummary(simulationID string) (map[string]interface{}, error) {
	summary := make(map[string]interface{})
	
	// Get simulation details
	sim, err := r.GetSimulation(simulationID)
	if err != nil {
		return nil, err
	}
	summary["simulation"] = sim
	
	// Get metric statistics
	var stats struct {
		AvgQueueDepth     float64
		MaxQueueDepth     int
		AvgExecutors      float64
		MaxExecutors      int
		TotalDecisions    int64
		ScaleUpCount      int64
		ScaleDownCount    int64
	}
	
	r.db.Model(&MetricSnapshot{}).
		Where("simulation_id = ?", simulationID).
		Select("AVG(queue_depth) as avg_queue_depth, MAX(queue_depth) as max_queue_depth, " +
			"AVG(actual_executors) as avg_executors, MAX(actual_executors) as max_executors").
		Scan(&stats)
	
	r.db.Model(&ScalingDecision{}).
		Where("simulation_id = ?", simulationID).
		Count(&stats.TotalDecisions)
	
	r.db.Model(&ScalingDecision{}).
		Where("simulation_id = ? AND decision_type = ?", simulationID, "scale_up").
		Count(&stats.ScaleUpCount)
	
	r.db.Model(&ScalingDecision{}).
		Where("simulation_id = ? AND decision_type = ?", simulationID, "scale_down").
		Count(&stats.ScaleDownCount)
	
	summary["statistics"] = stats
	
	return summary, nil
}

// BatchSaveMetrics saves multiple metric snapshots efficiently
func (r *Repository) BatchSaveMetrics(metrics []MetricSnapshot) error {
	if len(metrics) == 0 {
		return nil
	}
	
	// Use batch insert for efficiency
	return r.db.CreateInBatches(metrics, 100).Error
}

// DeleteSimulation deletes a simulation and all related data
func (r *Repository) DeleteSimulation(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete related data first
		if err := tx.Where("simulation_id = ?", id).Delete(&MetricSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("simulation_id = ?", id).Delete(&ScalingDecision{}).Error; err != nil {
			return err
		}
		if err := tx.Where("simulation_id = ?", id).Delete(&Event{}).Error; err != nil {
			return err
		}
		if err := tx.Where("simulation_id = ?", id).Delete(&PredictionAccuracy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("simulation_id = ?", id).Delete(&LearningMetrics{}).Error; err != nil {
			return err
		}
		
		// Delete simulation
		return tx.Where("id = ?", id).Delete(&Simulation{}).Error
	})
}

// GetLatestMetricSnapshot gets the most recent metric snapshot for a simulation
func (r *Repository) GetLatestMetricSnapshot(simulationID string) (*MetricSnapshot, error) {
	var snapshot MetricSnapshot
	err := r.db.Where("simulation_id = ?", simulationID).
		Order("timestamp DESC").
		First(&snapshot).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metric snapshot: %w", err)
	}
	
	return &snapshot, nil
}

// CRUD Operations for Simulation Management

// UpdateSimulationMetadata updates simulation name and description
func (r *Repository) UpdateSimulationMetadata(simulationID, name, description string) error {
	return r.db.Model(&Simulation{}).
		Where("id = ?", simulationID).
		Updates(map[string]interface{}{
			"name":        name,
			"description": description,
		}).Error
}

// CloneSimulation creates a copy of a simulation with new metadata
func (r *Repository) CloneSimulation(originalID, newName, newDescription string) (string, error) {
	// Create new simulation with different metadata (simple clone without copying data)
	newSim := &Simulation{
		ID:          generateSimulationID(),
		Name:        newName,
		Description: newDescription,
		Status:      "cloned",
		StartTime:   time.Now(),
		CreatedAt:   time.Now(),
	}
	
	if err := r.CreateSimulation(newSim); err != nil {
		return "", fmt.Errorf("failed to create cloned simulation: %w", err)
	}
	
	return newSim.ID, nil
}

// Helper function to generate simulation ID
func generateSimulationID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}