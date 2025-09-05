package queue

import (
	"time"
)

// simpleEWMA implements a basic exponentially weighted moving average
type simpleEWMA struct {
	alpha   float64
	current float64
	hasData bool
}

// newSimpleEWMA creates a new simple EWMA with given smoothing factor
func newSimpleEWMA(alpha float64) *simpleEWMA {
	return &simpleEWMA{
		alpha:   alpha,
		current: 0.0,
		hasData: false,
	}
}

// update adds a new value and returns the smoothed result
func (e *simpleEWMA) update(value float64) float64 {
	if !e.hasData {
		e.current = value
		e.hasData = true
	} else {
		e.current = e.alpha*value + (1-e.alpha)*e.current
	}
	return e.current
}

// getCurrent returns the current smoothed value
func (e *simpleEWMA) getCurrent() float64 {
	return e.current
}

// reset resets the EWMA to initial state
func (e *simpleEWMA) reset() {
	e.current = 0.0
	e.hasData = false
}

// AccelerationTracker tracks queue depth changes and calculates velocity and acceleration
type AccelerationTracker struct {
	// Historical data
	previousDepth       int         // Last queue depth measurement
	previousVelocity    float64     // Last velocity measurement  
	previousTimestamp   time.Time   // Last measurement time
	
	// EWMA smoothers for noise reduction
	velocityEWMA        *simpleEWMA // Smoothed velocity
	accelerationEWMA    *simpleEWMA // Smoothed acceleration
	
	// Sustained acceleration detection
	accelerationHistory []float64   // Recent acceleration values
	maxHistory         int          // Maximum history to keep
	sustainedThreshold int          // Minimum measurements for sustained acceleration
	
	// Configuration
	minTimeDelta       time.Duration // Minimum time between measurements
	initialized        bool         // Whether tracker has been initialized
}

// NewAccelerationTracker creates a new queue acceleration tracker
func NewAccelerationTracker() *AccelerationTracker {
	return &AccelerationTracker{
		velocityEWMA:       newSimpleEWMA(0.167),     // Conservative smoothing for velocity
		accelerationEWMA:   newSimpleEWMA(0.1),       // More conservative for acceleration
		accelerationHistory: make([]float64, 0),
		maxHistory:         10,
		sustainedThreshold: 3,                           // Need 3+ measurements
		minTimeDelta:       time.Second,                 // Minimum 1 second between measurements
		initialized:        false,
	}
}

// Update processes a new queue depth measurement and calculates velocity and acceleration
func (at *AccelerationTracker) Update(currentDepth int, timestamp time.Time) Metrics {
	metrics := Metrics{
		Depth:        currentDepth,
		Velocity:     0.0,
		Acceleration: 0.0,
		Timestamp:    timestamp,
	}
	
	// First measurement - initialize
	if !at.initialized {
		at.previousDepth = currentDepth
		at.previousVelocity = 0.0
		at.previousTimestamp = timestamp
		at.initialized = true
		return metrics
	}
	
	// Calculate time delta
	timeDelta := timestamp.Sub(at.previousTimestamp)
	if timeDelta < at.minTimeDelta {
		// Too soon since last measurement, return previous values
		metrics.Velocity = at.velocityEWMA.getCurrent()
		metrics.Acceleration = at.accelerationEWMA.getCurrent()
		return metrics
	}
	
	// Calculate raw velocity (items per second)
	depthDelta := float64(currentDepth - at.previousDepth)
	rawVelocity := depthDelta / timeDelta.Seconds()
	
	// Smooth velocity with EWMA
	smoothedVelocity := at.velocityEWMA.update(rawVelocity)
	
	// Calculate raw acceleration (items per second²)
	velocityDelta := smoothedVelocity - at.previousVelocity
	rawAcceleration := velocityDelta / timeDelta.Seconds()
	
	// Smooth acceleration with EWMA
	smoothedAcceleration := at.accelerationEWMA.update(rawAcceleration)
	
	// Update acceleration history for sustained detection
	at.updateAccelerationHistory(smoothedAcceleration)
	
	// Update metrics
	metrics.Velocity = smoothedVelocity
	metrics.Acceleration = smoothedAcceleration
	metrics.SustainedAcceleration = at.detectSustainedAcceleration()
	
	// Store current values as previous for next calculation
	at.previousDepth = currentDepth
	at.previousVelocity = smoothedVelocity
	at.previousTimestamp = timestamp
	
	return metrics
}

// updateAccelerationHistory maintains a rolling window of acceleration values
func (at *AccelerationTracker) updateAccelerationHistory(acceleration float64) {
	at.accelerationHistory = append(at.accelerationHistory, acceleration)
	
	// Keep only recent history
	if len(at.accelerationHistory) > at.maxHistory {
		at.accelerationHistory = at.accelerationHistory[len(at.accelerationHistory)-at.maxHistory:]
	}
}

// detectSustainedAcceleration determines if acceleration is sustained over multiple measurements
func (at *AccelerationTracker) detectSustainedAcceleration() bool {
	if len(at.accelerationHistory) < at.sustainedThreshold {
		return false
	}
	
	// Check if recent acceleration values are consistently positive
	recentAccelerations := at.accelerationHistory[len(at.accelerationHistory)-at.sustainedThreshold:]
	
	positiveCount := 0
	for _, accel := range recentAccelerations {
		if accel > 0.1 { // Small threshold to avoid noise
			positiveCount++
		}
	}
	
	// Sustained if majority of recent measurements show positive acceleration
	return positiveCount >= (at.sustainedThreshold+1)/2
}

// GetStats returns statistics about the acceleration tracker
func (at *AccelerationTracker) GetStats() Stats {
	avgAcceleration := 0.0
	maxAcceleration := 0.0
	minAcceleration := 0.0
	
	if len(at.accelerationHistory) > 0 {
		sum := 0.0
		maxAcceleration = at.accelerationHistory[0]
		minAcceleration = at.accelerationHistory[0]
		
		for _, accel := range at.accelerationHistory {
			sum += accel
			if accel > maxAcceleration {
				maxAcceleration = accel
			}
			if accel < minAcceleration {
				minAcceleration = accel
			}
		}
		avgAcceleration = sum / float64(len(at.accelerationHistory))
	}
	
	return Stats{
		CurrentVelocity:       at.velocityEWMA.getCurrent(),
		CurrentAcceleration:   at.accelerationEWMA.getCurrent(),
		AvgAcceleration:       avgAcceleration,
		MaxAcceleration:       maxAcceleration,
		MinAcceleration:       minAcceleration,
		HistoryLength:         len(at.accelerationHistory),
		SustainedAcceleration: at.detectSustainedAcceleration(),
		Initialized:           at.initialized,
		LastUpdate:            at.previousTimestamp,
	}
}

// Reset resets the acceleration tracker to initial state
func (at *AccelerationTracker) Reset() {
	at.velocityEWMA.reset()
	at.accelerationEWMA.reset()
	at.accelerationHistory = make([]float64, 0)
	at.initialized = false
	at.previousDepth = 0
	at.previousVelocity = 0.0
	at.previousTimestamp = time.Time{}
}

// Metrics represents calculated queue dynamics
type Metrics struct {
	Depth                 int       `json:"depth"`                   // Current queue depth
	Velocity              float64   `json:"velocity"`                // Rate of change (items/sec)
	Acceleration          float64   `json:"acceleration"`            // Rate of velocity change (items/sec²)
	SustainedAcceleration bool      `json:"sustained_acceleration"`  // Whether acceleration is sustained
	Timestamp             time.Time `json:"timestamp"`               // Measurement timestamp
}

// Stats provides insights into queue acceleration tracking
type Stats struct {
	CurrentVelocity       float64   `json:"current_velocity"`       // Current smoothed velocity
	CurrentAcceleration   float64   `json:"current_acceleration"`   // Current smoothed acceleration
	AvgAcceleration       float64   `json:"avg_acceleration"`       // Average recent acceleration
	MaxAcceleration       float64   `json:"max_acceleration"`       // Maximum recent acceleration
	MinAcceleration       float64   `json:"min_acceleration"`       // Minimum recent acceleration
	HistoryLength         int       `json:"history_length"`         // Number of acceleration measurements
	SustainedAcceleration bool      `json:"sustained_acceleration"` // Whether acceleration is sustained
	Initialized           bool      `json:"initialized"`            // Whether tracker is initialized
	LastUpdate            time.Time `json:"last_update"`            // Last measurement time
}

// IsAccelerating returns true if queue is accelerating (growing faster)
func (m Metrics) IsAccelerating() bool {
	return m.Acceleration > 0.1 && m.Velocity > 0
}

// IsDecelerating returns true if queue is decelerating (growth slowing)
func (m Metrics) IsDecelerating() bool {
	return m.Acceleration < -0.1 && m.Velocity > 0
}

// GetUrgencyScore calculates urgency based on depth, velocity, and acceleration
func (m Metrics) GetUrgencyScore(threshold int) float64 {
	// Base urgency from current depth
	baseUrgency := float64(m.Depth) / float64(threshold)
	
	// Velocity component (predictive) - 30% weight
	velocityFactor := 0.3
	velocityUrgency := 0.0
	if m.Velocity > 0 {
		velocityUrgency = m.Velocity * velocityFactor
	}
	
	// Acceleration component (early warning) - 10% weight to avoid over-reaction
	accelFactor := 0.1
	accelUrgency := 0.0
	if m.Acceleration > 0 {
		accelUrgency = m.Acceleration * accelFactor
		
		// Bonus for sustained acceleration
		if m.SustainedAcceleration {
			accelUrgency *= 1.5
		}
	}
	
	totalUrgency := baseUrgency + velocityUrgency + accelUrgency
	
	// Cap to avoid runaway scaling
	if totalUrgency > 2.0 {
		return 2.0
	}
	return totalUrgency
}