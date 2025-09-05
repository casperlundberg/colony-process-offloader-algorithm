package queue

import (
	"fmt"
	"time"
)

// Analyzer provides queue analysis and decision support
type Analyzer struct {
	accelerationTracker *AccelerationTracker
	thresholds         Thresholds
	history           []AnalysisResult
	maxHistoryLength   int
}

// Thresholds defines queue behavior thresholds
type Thresholds struct {
	CriticalDepth     int     `json:"critical_depth"`     // Depth at which queue is critical
	WarningDepth      int     `json:"warning_depth"`      // Depth at which to start warning
	VelocityThreshold float64 `json:"velocity_threshold"` // Velocity considered significant
	AccelThreshold    float64 `json:"accel_threshold"`    // Acceleration considered significant
}

// AnalysisResult represents the result of queue analysis
type AnalysisResult struct {
	Timestamp     time.Time `json:"timestamp"`
	QueueMetrics  Metrics   `json:"queue_metrics"`
	Status        Status    `json:"status"`
	Urgency       float64   `json:"urgency"`
	Recommendation Action   `json:"recommendation"`
	Reason        string    `json:"reason"`
}

// Status represents queue status levels
type Status int

const (
	StatusHealthy Status = iota
	StatusWarning
	StatusCritical
	StatusEmergency
)

func (s Status) String() string {
	switch s {
	case StatusWarning:
		return "warning"
	case StatusCritical:
		return "critical"
	case StatusEmergency:
		return "emergency"
	default:
		return "healthy"
	}
}

// Action represents recommended actions
type Action int

const (
	ActionNone Action = iota
	ActionMonitor
	ActionScaleUp
	ActionScaleUpUrgent
	ActionScaleDown
)

func (a Action) String() string {
	switch a {
	case ActionMonitor:
		return "monitor"
	case ActionScaleUp:
		return "scale_up"
	case ActionScaleUpUrgent:
		return "scale_up_urgent"
	case ActionScaleDown:
		return "scale_down"
	default:
		return "none"
	}
}

// NewAnalyzer creates a new queue analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		accelerationTracker: NewAccelerationTracker(),
		thresholds: Thresholds{
			CriticalDepth:     20,
			WarningDepth:      10,
			VelocityThreshold: 0.5,  // 0.5 items/second
			AccelThreshold:    0.1,  // 0.1 items/second²
		},
		history:          make([]AnalysisResult, 0),
		maxHistoryLength: 100,
	}
}

// NewAnalyzerWithThresholds creates an analyzer with custom thresholds
func NewAnalyzerWithThresholds(thresholds Thresholds) *Analyzer {
	analyzer := NewAnalyzer()
	analyzer.thresholds = thresholds
	return analyzer
}

// AnalyzeQueue performs comprehensive queue analysis
func (qa *Analyzer) AnalyzeQueue(queueDepth int, timestamp time.Time) AnalysisResult {
	// Update queue metrics with acceleration tracking
	metrics := qa.accelerationTracker.Update(queueDepth, timestamp)
	
	// Determine queue status
	status := qa.determineStatus(metrics)
	
	// Calculate urgency score
	urgency := qa.calculateUrgency(metrics)
	
	// Determine recommended action
	action := qa.recommendAction(metrics, status, urgency)
	
	// Build reason string
	reason := qa.buildReason(metrics, status, urgency)
	
	result := AnalysisResult{
		Timestamp:      timestamp,
		QueueMetrics:   metrics,
		Status:         status,
		Urgency:        urgency,
		Recommendation: action,
		Reason:         reason,
	}
	
	// Store in history
	qa.addToHistory(result)
	
	return result
}

// determineStatus determines queue status based on depth and dynamics
func (qa *Analyzer) determineStatus(metrics Metrics) Status {
	// Emergency: Very high depth with positive acceleration
	if metrics.Depth >= qa.thresholds.CriticalDepth*2 {
		return StatusEmergency
	}
	
	// Critical: High depth OR fast growth
	if metrics.Depth >= qa.thresholds.CriticalDepth {
		return StatusCritical
	}
	
	// Also critical if fast sustained growth
	if metrics.Velocity > qa.thresholds.VelocityThreshold*2 && metrics.SustainedAcceleration {
		return StatusCritical
	}
	
	// Warning: Moderate depth OR growing velocity
	if metrics.Depth >= qa.thresholds.WarningDepth {
		return StatusWarning
	}
	
	if metrics.Velocity > qa.thresholds.VelocityThreshold {
		return StatusWarning
	}
	
	// Healthy
	return StatusHealthy
}

// calculateUrgency calculates overall urgency score
func (qa *Analyzer) calculateUrgency(metrics Metrics) float64 {
	return metrics.GetUrgencyScore(qa.thresholds.CriticalDepth)
}

// recommendAction recommends scaling action based on analysis
func (qa *Analyzer) recommendAction(metrics Metrics, status Status, urgency float64) Action {
	// Emergency scaling
	if status == StatusEmergency {
		return ActionScaleUpUrgent
	}
	
	// Critical scaling
	if status == StatusCritical {
		return ActionScaleUpUrgent
	}
	
	// High urgency scaling
	if urgency > 1.5 {
		return ActionScaleUpUrgent
	}
	
	// Moderate scaling needs
	if status == StatusWarning || urgency > 1.0 {
		return ActionScaleUp
	}
	
	// Scale down if queue is empty and decelerating
	if metrics.Depth == 0 && metrics.Velocity < -qa.thresholds.VelocityThreshold {
		return ActionScaleDown
	}
	
	// Monitor if there's any activity
	if metrics.Velocity > qa.thresholds.VelocityThreshold/2 || metrics.Depth > 0 {
		return ActionMonitor
	}
	
	return ActionNone
}

// buildReason creates a human-readable reason for the analysis
func (qa *Analyzer) buildReason(metrics Metrics, status Status, urgency float64) string {
	reason := fmt.Sprintf("Queue depth: %d, status: %s", metrics.Depth, status.String())
	
	// Add velocity info
	if metrics.Velocity != 0 {
		direction := "growing"
		if metrics.Velocity < 0 {
			direction = "shrinking"
		}
		reason += fmt.Sprintf(", velocity: %.2f/s (%s)", metrics.Velocity, direction)
	}
	
	// Add acceleration info
	if abs(metrics.Acceleration) > qa.thresholds.AccelThreshold {
		direction := "accelerating"
		if metrics.Acceleration < 0 {
			direction = "decelerating"
		}
		reason += fmt.Sprintf(", %s at %.2f/s²", direction, metrics.Acceleration)
		
		if metrics.SustainedAcceleration {
			reason += " (sustained)"
		}
	}
	
	// Add urgency if significant
	if urgency > 1.0 {
		reason += fmt.Sprintf(", urgency: %.2f", urgency)
	}
	
	return reason
}

// addToHistory adds result to analysis history
func (qa *Analyzer) addToHistory(result AnalysisResult) {
	qa.history = append(qa.history, result)
	
	// Maintain history limit
	if len(qa.history) > qa.maxHistoryLength {
		qa.history = qa.history[len(qa.history)-qa.maxHistoryLength:]
	}
}

// GetHistory returns recent analysis history
func (qa *Analyzer) GetHistory(limit int) []AnalysisResult {
	if limit <= 0 || limit >= len(qa.history) {
		return qa.history
	}
	
	return qa.history[len(qa.history)-limit:]
}

// GetStats returns analyzer statistics
func (qa *Analyzer) GetStats() AnalyzerStats {
	return AnalyzerStats{
		Thresholds:           qa.thresholds,
		AccelerationStats:    qa.accelerationTracker.GetStats(),
		HistoryLength:        len(qa.history),
		LastAnalysis:         qa.getLastAnalysis(),
	}
}

// getLastAnalysis returns the most recent analysis result
func (qa *Analyzer) getLastAnalysis() *AnalysisResult {
	if len(qa.history) == 0 {
		return nil
	}
	return &qa.history[len(qa.history)-1]
}

// Reset resets the analyzer state
func (qa *Analyzer) Reset() {
	qa.accelerationTracker.Reset()
	qa.history = make([]AnalysisResult, 0)
}

// UpdateThresholds updates the analyzer thresholds
func (qa *Analyzer) UpdateThresholds(thresholds Thresholds) {
	qa.thresholds = thresholds
}

// AnalyzerStats provides comprehensive analyzer statistics
type AnalyzerStats struct {
	Thresholds        Thresholds  `json:"thresholds"`
	AccelerationStats Stats       `json:"acceleration_stats"`
	HistoryLength     int         `json:"history_length"`
	LastAnalysis      *AnalysisResult `json:"last_analysis,omitempty"`
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}