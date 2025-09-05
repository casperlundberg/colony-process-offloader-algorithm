package queue

import (
	"encoding/json"
	"fmt"
	"time"
)

// Queue represents the core queue model with dynamics tracking
type Queue struct {
	// Basic queue properties
	Depth      int           `json:"depth"`       // Current number of items in queue
	Threshold  int           `json:"threshold"`   // Critical depth threshold
	WaitTime   time.Duration `json:"wait_time"`   // Average wait time for items
	Throughput float64       `json:"throughput"`  // Items processed per second

	// Queue dynamics (calculated)
	Velocity     float64 `json:"velocity"`      // Rate of change (items/sec)
	Acceleration float64 `json:"acceleration"`  // Rate of velocity change (items/sec²)

	// Metadata
	LastUpdated time.Time `json:"last_updated"` // When queue was last updated
}

// State represents different queue states
type State int

const (
	StateEmpty State = iota
	StateLight
	StateModerate  
	StateHeavy
	StateCritical
	StateOverflow
)

func (s State) String() string {
	switch s {
	case StateLight:
		return "light"
	case StateModerate:
		return "moderate"
	case StateHeavy:
		return "heavy"
	case StateCritical:
		return "critical"
	case StateOverflow:
		return "overflow"
	default:
		return "empty"
	}
}

// NewQueue creates a new queue with default settings
func NewQueue(threshold int) *Queue {
	return &Queue{
		Depth:       0,
		Threshold:   threshold,
		WaitTime:    0,
		Throughput:  0,
		Velocity:    0,
		Acceleration: 0,
		LastUpdated: time.Now(),
	}
}

// UpdateDepth updates the queue depth and timestamp
func (q *Queue) UpdateDepth(newDepth int) {
	q.Depth = newDepth
	q.LastUpdated = time.Now()
}

// UpdateMetrics updates all queue metrics
func (q *Queue) UpdateMetrics(depth int, waitTime time.Duration, throughput float64) {
	q.Depth = depth
	q.WaitTime = waitTime
	q.Throughput = throughput
	q.LastUpdated = time.Now()
}

// UpdateDynamics updates velocity and acceleration from external tracker
func (q *Queue) UpdateDynamics(velocity, acceleration float64) {
	q.Velocity = velocity
	q.Acceleration = acceleration
	q.LastUpdated = time.Now()
}

// GetState returns the current queue state based on depth and threshold
func (q *Queue) GetState() State {
	if q.Depth == 0 {
		return StateEmpty
	}
	
	ratio := float64(q.Depth) / float64(q.Threshold)
	
	switch {
	case ratio >= 2.0:
		return StateOverflow
	case ratio >= 1.5:
		return StateCritical
	case ratio >= 1.0:
		return StateHeavy
	case ratio >= 0.5:
		return StateModerate
	default:
		return StateLight
	}
}

// GetPressure returns the queue pressure as a ratio of depth to threshold
func (q *Queue) GetPressure() float64 {
	if q.Threshold <= 0 {
		return 0.0
	}
	return float64(q.Depth) / float64(q.Threshold)
}

// IsHealthy returns true if queue is in a healthy state
func (q *Queue) IsHealthy() bool {
	state := q.GetState()
	return state == StateEmpty || state == StateLight || state == StateModerate
}

// IsCritical returns true if queue needs immediate attention
func (q *Queue) IsCritical() bool {
	state := q.GetState()
	return state == StateCritical || state == StateOverflow
}

// IsGrowing returns true if queue is growing (positive velocity)
func (q *Queue) IsGrowing() bool {
	return q.Velocity > 0.1
}

// IsShrinking returns true if queue is shrinking (negative velocity) 
func (q *Queue) IsShrinking() bool {
	return q.Velocity < -0.1
}

// IsAccelerating returns true if queue growth is accelerating
func (q *Queue) IsAccelerating() bool {
	return q.Acceleration > 0.1 && q.Velocity > 0
}

// IsDecelerating returns true if queue growth is slowing down
func (q *Queue) IsDecelerating() bool {
	return q.Acceleration < -0.1 && q.Velocity > 0
}

// GetUrgency returns an urgency score based on multiple factors
func (q *Queue) GetUrgency() float64 {
	// Base urgency from pressure
	pressure := q.GetPressure()
	urgency := pressure
	
	// Add velocity component (30% weight)
	if q.Velocity > 0 {
		urgency += q.Velocity * 0.3
	}
	
	// Add acceleration component (10% weight)
	if q.Acceleration > 0 {
		urgency += q.Acceleration * 0.1
	}
	
	// Cap at 2.0 to prevent runaway scaling
	if urgency > 2.0 {
		urgency = 2.0
	}
	
	return urgency
}

// EstimateTimeToThreshold estimates time to reach threshold at current velocity
func (q *Queue) EstimateTimeToThreshold() time.Duration {
	if q.Velocity <= 0 {
		return time.Duration(0) // Not growing
	}
	
	remaining := q.Threshold - q.Depth
	if remaining <= 0 {
		return time.Duration(0) // Already at or above threshold
	}
	
	secondsToThreshold := float64(remaining) / q.Velocity
	return time.Duration(secondsToThreshold * float64(time.Second))
}

// EstimateTimeToEmpty estimates time to empty queue at current throughput
func (q *Queue) EstimateTimeToEmpty() time.Duration {
	if q.Throughput <= 0 || q.Depth <= 0 {
		return time.Duration(0)
	}
	
	secondsToEmpty := float64(q.Depth) / q.Throughput
	return time.Duration(secondsToEmpty * float64(time.Second))
}

// Serialize converts the queue to JSON
func (q *Queue) Serialize() (string, error) {
	data, err := json.Marshal(q)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Deserialize creates a queue from JSON
func Deserialize(data string) (*Queue, error) {
	var queue Queue
	err := json.Unmarshal([]byte(data), &queue)
	if err != nil {
		return nil, err
	}
	return &queue, nil
}

// Clone creates a deep copy of the queue
func (q *Queue) Clone() *Queue {
	return &Queue{
		Depth:       q.Depth,
		Threshold:   q.Threshold,
		WaitTime:    q.WaitTime,
		Throughput:  q.Throughput,
		Velocity:    q.Velocity,
		Acceleration: q.Acceleration,
		LastUpdated: q.LastUpdated,
	}
}

// Summary provides a human-readable summary of the queue
func (q *Queue) Summary() string {
	state := q.GetState()
	pressure := q.GetPressure()
	
	summary := fmt.Sprintf("Queue: %d items, %s (%.1f%% of threshold)", 
		q.Depth, state.String(), pressure*100)
	
	if q.Velocity != 0 {
		direction := "growing"
		if q.Velocity < 0 {
			direction = "shrinking"
		}
		summary += fmt.Sprintf(", %s at %.2f/s", direction, q.Velocity)
	}
	
	if abs(q.Acceleration) > 0.1 {
		accelDir := "accelerating"
		if q.Acceleration < 0 {
			accelDir = "decelerating"
		}
		summary += fmt.Sprintf(", %s", accelDir)
	}
	
	return summary
}

