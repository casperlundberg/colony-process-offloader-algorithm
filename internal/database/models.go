package database

import (
	"time"
)

// Simulation represents a single simulation run
type Simulation struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	Status      string    `json:"status"` // running, completed, failed
	Config      string    `json:"config"` // JSON configuration
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MetricSnapshot represents a point-in-time metric collection
type MetricSnapshot struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	SimulationID string    `json:"simulation_id" gorm:"index"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	
	// Queue metrics
	QueueDepth       int     `json:"queue_depth"`
	QueueVelocity    float64 `json:"queue_velocity"`
	QueueAcceleration float64 `json:"queue_acceleration"`
	QueueThroughput  float64 `json:"queue_throughput"`
	QueueWaitTime    float64 `json:"queue_wait_time"` // in seconds
	
	// Executor metrics
	ActualExecutors  int `json:"actual_executors"`
	PlannedExecutors int `json:"planned_executors"`
	PendingExecutors int `json:"pending_executors"`
	FailedExecutors  int `json:"failed_executors"`
	
	// System metrics
	ComputeUsage float64 `json:"compute_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	NetworkUsage float64 `json:"network_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	MasterUsage  float64 `json:"master_usage"`
	
	// Load metrics
	SystemLoad    float64 `json:"system_load"`
	QueuePressure float64 `json:"queue_pressure"`
	Urgency       float64 `json:"urgency"`
	
	CreatedAt time.Time `json:"created_at"`
}

// ScalingDecision represents a scaling decision made by the algorithm
type ScalingDecision struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	SimulationID string    `json:"simulation_id" gorm:"index"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	
	DecisionType string `json:"decision_type"` // scale_up, scale_down, no_action
	FromCount    int    `json:"from_count"`
	ToCount      int    `json:"to_count"`
	Delta        int    `json:"delta"`
	
	// Decision factors
	QueueDepth        int     `json:"queue_depth"`
	QueueVelocity     float64 `json:"queue_velocity"`
	QueueAcceleration float64 `json:"queue_acceleration"`
	SystemLoad        float64 `json:"system_load"`
	Urgency           float64 `json:"urgency"`
	
	// Reasoning
	Reason      string `json:"reason"`
	Algorithm   string `json:"algorithm"`
	Confidence  float64 `json:"confidence"`
	
	// Outcome
	Success     bool   `json:"success"`
	ErrorMsg    string `json:"error_msg"`
	ExecutionTime float64 `json:"execution_time"` // milliseconds
	
	CreatedAt time.Time `json:"created_at"`
}

// Event represents simulation events (executor lifecycle, errors, etc)
type Event struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	SimulationID string    `json:"simulation_id" gorm:"index"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	
	EventType string `json:"event_type"` // executor_started, executor_stopped, error, warning
	Category  string `json:"category"`   // executor, queue, system, algorithm
	Severity  string `json:"severity"`   // info, warning, error, critical
	
	Message string `json:"message"`
	Details string `json:"details"` // JSON for additional data
	
	CreatedAt time.Time `json:"created_at"`
}

// PredictionAccuracy tracks how well the algorithm predicted future states
type PredictionAccuracy struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	SimulationID string    `json:"simulation_id" gorm:"index"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	
	PredictionHorizon int `json:"prediction_horizon"` // seconds into future
	
	// Queue predictions
	PredictedQueueDepth int     `json:"predicted_queue_depth"`
	ActualQueueDepth    int     `json:"actual_queue_depth"`
	QueueDepthError     float64 `json:"queue_depth_error"`
	
	// Load predictions
	PredictedLoad float64 `json:"predicted_load"`
	ActualLoad    float64 `json:"actual_load"`
	LoadError     float64 `json:"load_error"`
	
	// Executor predictions
	PredictedExecutors int     `json:"predicted_executors"`
	ActualExecutors    int     `json:"actual_executors"`
	ExecutorError      float64 `json:"executor_error"`
	
	CreatedAt time.Time `json:"created_at"`
}

// LearningMetrics tracks ML algorithm performance
type LearningMetrics struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	SimulationID string    `json:"simulation_id" gorm:"index"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	
	Algorithm string `json:"algorithm"` // q_learning, thompson_sampling, etc
	
	// Q-Learning specific
	QTableSize      int     `json:"q_table_size"`
	ExplorationRate float64 `json:"exploration_rate"`
	LearningRate    float64 `json:"learning_rate"`
	ConvergenceRate float64 `json:"convergence_rate"`
	
	// Thompson Sampling specific
	ArmCount       int     `json:"arm_count"`
	BestArmReward  float64 `json:"best_arm_reward"`
	TotalReward    float64 `json:"total_reward"`
	
	// ARIMA specific
	ARIMAPredictionError float64 `json:"arima_prediction_error"`
	ARIMAModelOrder      string  `json:"arima_model_order"` // p,d,q values
	
	// General metrics
	AccuracyScore float64 `json:"accuracy_score"`
	RecallScore   float64 `json:"recall_score"`
	F1Score       float64 `json:"f1_score"`
	
	CreatedAt time.Time `json:"created_at"`
}