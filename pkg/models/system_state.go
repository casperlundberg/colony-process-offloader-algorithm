package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// SystemState represents the complete observable state at decision time
type SystemState struct {
	// Queue metrics
	QueueDepth      int           `json:"queue_depth"`
	QueueThreshold  int           `json:"queue_threshold"`
	QueueWaitTime   time.Duration `json:"queue_wait_time"`
	QueueThroughput float64       `json:"queue_throughput"` // Processes per second

	// Resource utilization (0.0-1.0 scale)
	ComputeUsage Utilization `json:"compute_usage"`
	MemoryUsage  Utilization `json:"memory_usage"`
	DiskUsage    Utilization `json:"disk_usage"`
	NetworkUsage Utilization `json:"network_usage"`

	// Management overhead
	MasterUsage       Utilization `json:"master_usage"`
	ActiveConnections int         `json:"active_connections"`

	// Temporal context
	Timestamp time.Time `json:"timestamp"`
	TimeSlot  int       `json:"time_slot"`  // Hour of day (0-23)
	DayOfWeek int       `json:"day_of_week"` // Day of week (0-6)
}

// SystemStateInput represents raw system metrics for creating SystemState
type SystemStateInput struct {
	CPUCores          int
	CPUUsedCores      int
	TotalMemory       int64
	UsedMemory        int64
	TotalDisk         int64
	UsedDisk          int64
	NetworkBandwidth  int64
	NetworkUsed       int64
	QueueLength       int
	ActiveConnections int
}

// SystemStateCollector collects system state metrics
type SystemStateCollector struct {
	// Implementation would interface with actual system metrics
}

// NewSystemStateCollector creates a new system state collector
func NewSystemStateCollector() *SystemStateCollector {
	return &SystemStateCollector{}
}

// CaptureState captures the current system state
func (ssc *SystemStateCollector) CaptureState() (SystemState, error) {
	start := time.Now()
	
	// In a real implementation, this would collect actual metrics
	// For now, we'll simulate realistic values
	state := SystemState{
		QueueDepth:        0,  // Would be read from actual queue
		QueueThreshold:    20,
		QueueWaitTime:     0,
		QueueThroughput:   10.0,
		ComputeUsage:      0.3,
		MemoryUsage:       0.4,
		DiskUsage:         0.2,
		NetworkUsage:      0.1,
		MasterUsage:       0.15,
		ActiveConnections: 25,
		Timestamp:         start,
		TimeSlot:          start.Hour(),
		DayOfWeek:         int(start.Weekday()),
	}
	
	return state, nil
}

// CreateSystemStateFromInput creates a SystemState from raw input metrics
func CreateSystemStateFromInput(input SystemStateInput) SystemState {
	now := time.Now()
	
	// Calculate normalized utilization values
	var computeUsage, memoryUsage, diskUsage, networkUsage Utilization
	
	if input.CPUCores > 0 {
		computeUsage = Utilization(float64(input.CPUUsedCores) / float64(input.CPUCores))
	}
	
	if input.TotalMemory > 0 {
		memoryUsage = Utilization(float64(input.UsedMemory) / float64(input.TotalMemory))
	}
	
	if input.TotalDisk > 0 {
		diskUsage = Utilization(float64(input.UsedDisk) / float64(input.TotalDisk))
	}
	
	if input.NetworkBandwidth > 0 {
		networkUsage = Utilization(float64(input.NetworkUsed) / float64(input.NetworkBandwidth))
	}
	
	// Master usage correlates with overall system load
	masterUsage := Utilization(0.1 + float64(computeUsage)*0.3)
	if masterUsage > 1.0 {
		masterUsage = 1.0
	}
	
	// Queue throughput inversely correlates with compute usage
	throughput := 15.0 * (1.0 - float64(computeUsage))
	if throughput < 1.0 {
		throughput = 1.0
	}
	
	// Queue wait time correlates with queue depth
	waitTime := time.Duration(input.QueueLength*2) * time.Second
	
	return SystemState{
		QueueDepth:        input.QueueLength,
		QueueThreshold:    20, // Default threshold
		QueueWaitTime:     waitTime,
		QueueThroughput:   throughput,
		ComputeUsage:      computeUsage,
		MemoryUsage:       memoryUsage,
		DiskUsage:         diskUsage,
		NetworkUsage:      networkUsage,
		MasterUsage:       masterUsage,
		ActiveConnections: input.ActiveConnections,
		Timestamp:         now,
		TimeSlot:          now.Hour(),
		DayOfWeek:         int(now.Weekday()),
	}
}

// Validate validates the system state
func (ss SystemState) Validate() error {
	var errors ValidationErrors

	// Validate utilization metrics are in range [0.0, 1.0]
	errors.AddIf(!ss.ComputeUsage.IsValid(), "ComputeUsage", ss.ComputeUsage, 
		"ComputeUsage must be in range [0.0, 1.0]")
	errors.AddIf(!ss.MemoryUsage.IsValid(), "MemoryUsage", ss.MemoryUsage, 
		"MemoryUsage must be in range [0.0, 1.0]")
	errors.AddIf(!ss.DiskUsage.IsValid(), "DiskUsage", ss.DiskUsage, 
		"DiskUsage must be in range [0.0, 1.0]")
	errors.AddIf(!ss.NetworkUsage.IsValid(), "NetworkUsage", ss.NetworkUsage, 
		"NetworkUsage must be in range [0.0, 1.0]")
	errors.AddIf(!ss.MasterUsage.IsValid(), "MasterUsage", ss.MasterUsage, 
		"MasterUsage must be in range [0.0, 1.0]")

	// Validate queue metrics
	errors.AddIf(ss.QueueDepth < 0, "QueueDepth", ss.QueueDepth, 
		"QueueDepth must be non-negative")
	errors.AddIf(ss.QueueThreshold < 0, "QueueThreshold", ss.QueueThreshold, 
		"QueueThreshold must be non-negative")
	errors.AddIf(ss.QueueThroughput < 0, "QueueThroughput", ss.QueueThroughput, 
		"QueueThroughput must be non-negative")

	// Validate connection count
	errors.AddIf(ss.ActiveConnections < 0, "ActiveConnections", ss.ActiveConnections, 
		"ActiveConnections must be non-negative")

	// Validate temporal context
	errors.AddIf(ss.TimeSlot < 0 || ss.TimeSlot > 23, "TimeSlot", ss.TimeSlot, 
		"TimeSlot must be in range [0, 23]")
	errors.AddIf(ss.DayOfWeek < 0 || ss.DayOfWeek > 6, "DayOfWeek", ss.DayOfWeek, 
		"DayOfWeek must be in range [0, 6]")

	if errors.HasErrors() {
		if len(errors) > 1 {
			return ValidationErrors{ValidationError{
				Field:   "multiple",
				Value:   "SystemState",
				Message: "Multiple utilization metrics out of range",
			}}
		}
		return errors
	}

	return nil
}

// ValidateTemporalContext validates just the temporal fields
func (ss SystemState) ValidateTemporalContext() error {
	var errors ValidationErrors

	errors.AddIf(ss.TimeSlot < 0 || ss.TimeSlot > 23, "TimeSlot", ss.TimeSlot, 
		"TimeSlot must be in range [0, 23]")
	errors.AddIf(ss.DayOfWeek < 0 || ss.DayOfWeek > 6, "DayOfWeek", ss.DayOfWeek, 
		"DayOfWeek must be in range [0, 6]")

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// Serialize serializes the system state to JSON
func (ss SystemState) Serialize() string {
	data, err := json.Marshal(ss)
	if err != nil {
		return ""
	}
	return string(data)
}

// DeserializeSystemState deserializes a system state from JSON
func DeserializeSystemState(data string) (SystemState, error) {
	var state SystemState
	err := json.Unmarshal([]byte(data), &state)
	if err != nil {
		return SystemState{}, fmt.Errorf("failed to deserialize SystemState: %w", err)
	}
	return state, nil
}

// IsHighLoad returns true if the system is under high load
func (ss SystemState) IsHighLoad() bool {
	return ss.ComputeUsage > 0.8 || ss.MemoryUsage > 0.85 || 
		   ss.QueueDepth > ss.QueueThreshold
}

// IsLowLoad returns true if the system is under low load
func (ss SystemState) IsLowLoad() bool {
	return ss.ComputeUsage < 0.3 && ss.MemoryUsage < 0.4 && 
		   ss.QueueDepth < ss.QueueThreshold/2
}

// GetLoadScore returns an overall load score (0.0-1.0)
func (ss SystemState) GetLoadScore() float64 {
	// Weighted average of different load metrics
	return (float64(ss.ComputeUsage)*0.4 + 
			float64(ss.MemoryUsage)*0.3 + 
			float64(ss.NetworkUsage)*0.1 + 
			float64(ss.MasterUsage)*0.1 + 
			min(float64(ss.QueueDepth)/float64(max(ss.QueueThreshold, 1)), 1.0)*0.1)
}

// GetQueuePressure returns the queue pressure level (0.0-1.0+)
func (ss SystemState) GetQueuePressure() float64 {
	if ss.QueueThreshold <= 0 {
		return 0.0
	}
	return float64(ss.QueueDepth) / float64(ss.QueueThreshold)
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}