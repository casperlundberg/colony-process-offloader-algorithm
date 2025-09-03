package learning

import (
	"math"
	"time"
)

// EWMA implements Exponentially Weighted Moving Average smoother
// Referenced in FURTHER_CONTROL_THEORY.md as [2] with alpha=0.167
type EWMA struct {
	alpha       float64   // Smoothing parameter (0 < alpha <= 1)
	currentEWMA float64   // Current EWMA value
	initialized bool      // Whether EWMA has been initialized
	lastUpdate  time.Time // Last update timestamp
	valueCount  int       // Number of values processed
}

// NewEWMA creates a new EWMA smoother with specified alpha
func NewEWMA(alpha float64) *EWMA {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.167 // Default from FURTHER_CONTROL_THEORY.md
	}
	
	return &EWMA{
		alpha:       alpha,
		initialized: false,
		lastUpdate:  time.Now(),
		valueCount:  0,
	}
}

// NewEWMADefault creates a new EWMA smoother with default alpha=0.167
func NewEWMADefault() *EWMA {
	return NewEWMA(0.167)
}

// Update updates the EWMA with a new observation
func (e *EWMA) Update(value float64) float64 {
	e.lastUpdate = time.Now()
	e.valueCount++
	
	if !e.initialized {
		// Initialize with first value
		e.currentEWMA = value
		e.initialized = true
	} else {
		// EWMA formula: EWMA_t = alpha * X_t + (1 - alpha) * EWMA_{t-1}
		e.currentEWMA = e.alpha*value + (1-e.alpha)*e.currentEWMA
	}
	
	return e.currentEWMA
}

// GetCurrent returns the current EWMA value
func (e *EWMA) GetCurrent() float64 {
	if !e.initialized {
		return 0.0
	}
	return e.currentEWMA
}

// Predict predicts the next value (which is the current EWMA)
func (e *EWMA) Predict() float64 {
	return e.GetCurrent()
}

// UpdateBatch processes multiple values at once
func (e *EWMA) UpdateBatch(values []float64) []float64 {
	results := make([]float64, len(values))
	
	for i, value := range values {
		results[i] = e.Update(value)
	}
	
	return results
}

// CalculateOptimalAlpha estimates optimal alpha based on historical data
func (e *EWMA) CalculateOptimalAlpha(values []float64) float64 {
	if len(values) < 3 {
		return e.alpha
	}
	
	bestAlpha := e.alpha
	bestMSE := math.Inf(1)
	
	// Test different alpha values
	alphas := []float64{0.1, 0.167, 0.2, 0.25, 0.3, 0.4, 0.5}
	
	for _, alpha := range alphas {
		mse := e.evaluateAlpha(alpha, values)
		if mse < bestMSE {
			bestMSE = mse
			bestAlpha = alpha
		}
	}
	
	return bestAlpha
}

// evaluateAlpha calculates MSE for a given alpha on historical data
func (e *EWMA) evaluateAlpha(alpha float64, values []float64) float64 {
	if len(values) < 2 {
		return math.Inf(1)
	}
	
	ewma := values[0] // Initialize with first value
	mse := 0.0
	
	for i := 1; i < len(values); i++ {
		prediction := ewma
		actual := values[i]
		error := actual - prediction
		mse += error * error
		
		// Update EWMA
		ewma = alpha*actual + (1-alpha)*ewma
	}
	
	return mse / float64(len(values)-1)
}

// GetVariance estimates the variance of the smoothed series
func (e *EWMA) GetVariance(values []float64) float64 {
	if len(values) < 2 || !e.initialized {
		return 0.0
	}
	
	// Calculate variance of residuals from EWMA
	variance := 0.0
	ewma := values[0]
	count := 0
	
	for i := 1; i < len(values); i++ {
		residual := values[i] - ewma
		variance += residual * residual
		ewma = e.alpha*values[i] + (1-e.alpha)*ewma
		count++
	}
	
	if count > 1 {
		variance /= float64(count - 1)
	}
	
	return variance
}

// GetConfidenceInterval calculates confidence interval for predictions
func (e *EWMA) GetConfidenceInterval(values []float64, confidence float64) (float64, float64) {
	if len(values) < 3 || !e.initialized {
		return e.currentEWMA, e.currentEWMA
	}
	
	// Calculate standard error
	variance := e.GetVariance(values)
	stdError := math.Sqrt(variance)
	
	// Z-score for confidence level (approximation)
	var zScore float64
	switch {
	case confidence >= 0.99:
		zScore = 2.576
	case confidence >= 0.95:
		zScore = 1.96
	case confidence >= 0.90:
		zScore = 1.645
	default:
		zScore = 1.96 // Default to 95%
	}
	
	margin := zScore * stdError
	return e.currentEWMA - margin, e.currentEWMA + margin
}

// DetectTrend analyzes if there's a trend in recent EWMA values
func (e *EWMA) DetectTrend(values []float64, windowSize int) TrendDirection {
	if len(values) < windowSize || windowSize < 3 {
		return TrendNone
	}
	
	// Calculate EWMA for recent values
	recent := values[len(values)-windowSize:]
	ewmaValues := make([]float64, len(recent))
	
	tempEWMA := NewEWMA(e.alpha)
	for i, val := range recent {
		ewmaValues[i] = tempEWMA.Update(val)
	}
	
	// Simple trend detection using linear regression slope
	n := float64(len(ewmaValues))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0
	
	for i, y := range ewmaValues {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	
	// Calculate slope
	denominator := n*sumXX - sumX*sumX
	if denominator == 0 {
		return TrendNone
	}
	
	slope := (n*sumXY - sumX*sumY) / denominator
	
	// Classify trend based on slope
	threshold := 0.01 // Minimum slope to consider as trend
	
	if slope > threshold {
		return TrendUpward
	} else if slope < -threshold {
		return TrendDownward
	}
	
	return TrendNone
}

// GetStats returns statistics about the EWMA smoother
func (e *EWMA) GetStats() EWMAStats {
	return EWMAStats{
		Alpha:       e.alpha,
		CurrentEWMA: e.currentEWMA,
		Initialized: e.initialized,
		ValueCount:  e.valueCount,
		LastUpdate:  e.lastUpdate,
	}
}

// Reset resets the EWMA to uninitialized state
func (e *EWMA) Reset() {
	e.currentEWMA = 0.0
	e.initialized = false
	e.valueCount = 0
	e.lastUpdate = time.Now()
}

// SetAlpha updates the smoothing parameter
func (e *EWMA) SetAlpha(alpha float64) {
	if alpha > 0 && alpha <= 1 {
		e.alpha = alpha
	}
}

// TrendDirection represents the direction of a trend
type TrendDirection int

const (
	TrendNone TrendDirection = iota
	TrendUpward
	TrendDownward
)

// String returns string representation of trend direction
func (td TrendDirection) String() string {
	switch td {
	case TrendUpward:
		return "upward"
	case TrendDownward:
		return "downward"
	default:
		return "none"
	}
}

// EWMAStats represents statistics about the EWMA smoother
type EWMAStats struct {
	Alpha       float64   `json:"alpha"`        // Smoothing parameter
	CurrentEWMA float64   `json:"current_ewma"` // Current EWMA value
	Initialized bool      `json:"initialized"`  // Whether initialized
	ValueCount  int       `json:"value_count"`  // Number of values processed
	LastUpdate  time.Time `json:"last_update"`  // Last update time
}

// AdaptiveEWMA implements an EWMA with adaptive alpha
type AdaptiveEWMA struct {
	*EWMA
	baseAlpha    float64   // Base alpha value
	adaptionRate float64   // How quickly to adapt alpha
	recentErrors []float64 // Recent prediction errors
	maxErrors    int       // Maximum errors to track
}

// NewAdaptiveEWMA creates an EWMA with adaptive smoothing parameter
func NewAdaptiveEWMA(baseAlpha, adaptionRate float64) *AdaptiveEWMA {
	return &AdaptiveEWMA{
		EWMA:         NewEWMA(baseAlpha),
		baseAlpha:    baseAlpha,
		adaptionRate: adaptionRate,
		recentErrors: make([]float64, 0),
		maxErrors:    10,
	}
}

// UpdateAdaptive updates EWMA and adapts alpha based on recent performance
func (ae *AdaptiveEWMA) UpdateAdaptive(value float64) float64 {
	// Calculate prediction error
	if ae.initialized {
		prediction := ae.GetCurrent()
		error := math.Abs(value - prediction)
		
		ae.recentErrors = append(ae.recentErrors, error)
		if len(ae.recentErrors) > ae.maxErrors {
			ae.recentErrors = ae.recentErrors[1:]
		}
		
		// Adapt alpha based on recent error variance
		if len(ae.recentErrors) >= 3 {
			errorVariance := ae.calculateErrorVariance()
			
			// Increase alpha when errors are high (more responsive)
			// Decrease alpha when errors are low (more stable)
			adjustment := errorVariance * ae.adaptionRate
			newAlpha := ae.baseAlpha + adjustment
			
			// Keep alpha in valid range
			if newAlpha < 0.01 {
				newAlpha = 0.01
			} else if newAlpha > 1.0 {
				newAlpha = 1.0
			}
			
			ae.SetAlpha(newAlpha)
		}
	}
	
	return ae.Update(value)
}

// calculateErrorVariance calculates variance of recent errors
func (ae *AdaptiveEWMA) calculateErrorVariance() float64 {
	if len(ae.recentErrors) < 2 {
		return 0.0
	}
	
	// Calculate mean
	mean := 0.0
	for _, err := range ae.recentErrors {
		mean += err
	}
	mean /= float64(len(ae.recentErrors))
	
	// Calculate variance
	variance := 0.0
	for _, err := range ae.recentErrors {
		variance += (err - mean) * (err - mean)
	}
	variance /= float64(len(ae.recentErrors) - 1)
	
	return variance
}