package learning

import (
	"fmt"
	"math"
	"time"
)

// CUSUM implements Cumulative Sum anomaly detection algorithm
// Referenced in FURTHER_CONTROL_THEORY.md as [3] with parameters (0.5σ, 5σ)
type CUSUM struct {
	// Algorithm parameters
	threshold    float64   // Detection threshold (h) - typically 5σ
	drift        float64   // Expected drift (k) - typically 0.5σ  
	reference    float64   // Reference value (μ0) - target mean
	
	// State variables
	cumulativeSum float64  // Current cumulative sum
	positiveSum   float64  // Positive cumulative sum (C+)
	negativeSum   float64  // Negative cumulative sum (C-)
	
	// Statistics tracking
	observations  []float64 // Recent observations for statistics
	mean          float64   // Sample mean
	stdDev        float64   // Sample standard deviation
	maxHistory    int       // Maximum history to maintain
	
	// Detection state
	lastAnomaly   time.Time // Last anomaly detection time
	anomalyCount  int       // Total anomalies detected
	falsePositives int      // Estimated false positives
	
	// Adaptation
	adaptive      bool      // Whether to use adaptive parameters
	updateCount   int       // Number of updates
}

// NewCUSUM creates a new CUSUM detector with specified parameters
func NewCUSUM(threshold, drift, reference float64) *CUSUM {
	return &CUSUM{
		threshold:     threshold,
		drift:         drift,
		reference:     reference,
		cumulativeSum: 0.0,
		positiveSum:   0.0,
		negativeSum:   0.0,
		observations:  make([]float64, 0),
		maxHistory:    100,
		adaptive:      false,
	}
}

// NewCUSUMFromSigma creates CUSUM with parameters based on sigma
// Uses the standard configuration: k=0.5σ, h=5σ as per FURTHER_CONTROL_THEORY.md
func NewCUSUMFromSigma(sigma float64, reference float64) *CUSUM {
	return NewCUSUM(
		5.0*sigma, // h = 5σ (detection threshold)
		0.5*sigma, // k = 0.5σ (drift parameter)  
		reference, // μ0 (reference mean)
	)
}

// NewCUSUMAdaptive creates an adaptive CUSUM that estimates parameters from data
func NewCUSUMAdaptive() *CUSUM {
	cusum := &CUSUM{
		threshold:     5.0, // Will be updated
		drift:         0.5, // Will be updated
		reference:     0.0, // Will be updated
		cumulativeSum: 0.0,
		positiveSum:   0.0,
		negativeSum:   0.0,
		observations:  make([]float64, 0),
		maxHistory:    100,
		adaptive:      true,
	}
	
	return cusum
}

// Update processes a new observation and returns anomaly detection result
func (c *CUSUM) Update(value float64) CUSUMResult {
	c.observations = append(c.observations, value)
	c.updateCount++
	
	// Maintain history limit
	if len(c.observations) > c.maxHistory {
		c.observations = c.observations[len(c.observations)-c.maxHistory:]
	}
	
	// Update statistics if adaptive
	if c.adaptive {
		c.updateStatistics()
		c.updateParameters()
	}
	
	// Calculate deviation from reference
	deviation := value - c.reference
	
	// Update cumulative sums using standard CUSUM formulas
	// C+ = max(0, C+_{t-1} + (x_t - μ0) - k)
	// C- = max(0, C-_{t-1} - (x_t - μ0) - k)
	
	c.positiveSum = math.Max(0, c.positiveSum+deviation-c.drift)
	c.negativeSum = math.Max(0, c.negativeSum-deviation-c.drift)
	
	// Overall cumulative sum for general monitoring
	c.cumulativeSum += deviation
	
	// Detect anomalies
	anomalyType := AnomalyNone
	anomalySeverity := 0.0
	
	if c.positiveSum > c.threshold {
		anomalyType = AnomalyUpward
		anomalySeverity = c.positiveSum / c.threshold
		c.onAnomalyDetected()
	} else if c.negativeSum > c.threshold {
		anomalyType = AnomalyDownward  
		anomalySeverity = c.negativeSum / c.threshold
		c.onAnomalyDetected()
	}
	
	return CUSUMResult{
		Value:           value,
		CumulativeSum:   c.cumulativeSum,
		PositiveSum:     c.positiveSum,
		NegativeSum:     c.negativeSum,
		AnomalyType:     anomalyType,
		AnomalySeverity: anomalySeverity,
		IsAnomaly:       anomalyType != AnomalyNone,
		Timestamp:       time.Now(),
	}
}

// onAnomalyDetected handles anomaly detection event
func (c *CUSUM) onAnomalyDetected() {
	c.lastAnomaly = time.Now()
	c.anomalyCount++
	
	// Reset cumulative sums after detection (standard practice)
	c.positiveSum = 0.0
	c.negativeSum = 0.0
}

// updateStatistics updates running statistics for adaptive behavior
func (c *CUSUM) updateStatistics() {
	if len(c.observations) < 2 {
		return
	}
	
	// Calculate sample mean
	sum := 0.0
	for _, obs := range c.observations {
		sum += obs
	}
	c.mean = sum / float64(len(c.observations))
	
	// Calculate sample standard deviation
	variance := 0.0
	for _, obs := range c.observations {
		variance += (obs - c.mean) * (obs - c.mean)
	}
	variance /= float64(len(c.observations) - 1)
	c.stdDev = math.Sqrt(variance)
}

// updateParameters adapts CUSUM parameters based on observed data
func (c *CUSUM) updateParameters() {
	if len(c.observations) < 10 || c.stdDev == 0 {
		return
	}
	
	// Update reference to current mean (for change detection)
	learningRate := 0.1
	c.reference = (1-learningRate)*c.reference + learningRate*c.mean
	
	// Update drift and threshold based on current standard deviation
	c.drift = 0.5 * c.stdDev
	c.threshold = 5.0 * c.stdDev
}

// BatchDetect processes multiple values and returns anomaly results
func (c *CUSUM) BatchDetect(values []float64) []CUSUMResult {
	results := make([]CUSUMResult, len(values))
	
	for i, value := range values {
		results[i] = c.Update(value)
	}
	
	return results
}

// GetChangePoint detects the most likely change point in recent data
func (c *CUSUM) GetChangePoint(lookbackWindow int) (int, float64) {
	if len(c.observations) < lookbackWindow || lookbackWindow < 3 {
		return -1, 0.0
	}
	
	// Use retrospective CUSUM to find change point
	recent := c.observations[len(c.observations)-lookbackWindow:]
	
	maxLikelihood := 0.0
	changePoint := -1
	
	// Test each potential change point
	for k := 2; k < len(recent)-2; k++ {
		// Calculate likelihood ratio for change at position k
		likelihood := c.calculateLikelihoodRatio(recent, k)
		
		if likelihood > maxLikelihood {
			maxLikelihood = likelihood
			changePoint = len(c.observations) - lookbackWindow + k
		}
	}
	
	return changePoint, maxLikelihood
}

// calculateLikelihoodRatio calculates likelihood ratio for change point detection
func (c *CUSUM) calculateLikelihoodRatio(data []float64, changePoint int) float64 {
	if changePoint <= 0 || changePoint >= len(data) {
		return 0.0
	}
	
	// Split data at potential change point
	before := data[:changePoint]
	after := data[changePoint:]
	
	if len(before) < 2 || len(after) < 2 {
		return 0.0
	}
	
	// Calculate means and variances for each segment
	meanBefore, varBefore := c.calculateMeanVar(before)
	meanAfter, varAfter := c.calculateMeanVar(after)
	
	// Avoid division by zero
	if varBefore <= 0 || varAfter <= 0 {
		return 0.0
	}
	
	// Simplified likelihood ratio (log-likelihood difference)
	n1, n2 := float64(len(before)), float64(len(after))
	
	logLikelihood := -0.5 * (n1*math.Log(varBefore) + n2*math.Log(varAfter))
	logLikelihood += 0.5 * (n1+n2) * math.Log((n1*varBefore+n2*varAfter)/(n1+n2))
	
	// Add mean difference component
	meanDiff := meanAfter - meanBefore
	logLikelihood += 0.5 * n1 * n2 / (n1 + n2) * meanDiff * meanDiff / ((n1*varBefore+n2*varAfter)/(n1+n2))
	
	return logLikelihood
}

// calculateMeanVar calculates mean and variance of a slice
func (c *CUSUM) calculateMeanVar(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0.0, 0.0
	}
	
	// Calculate mean
	sum := 0.0
	for _, val := range data {
		sum += val
	}
	mean := sum / float64(len(data))
	
	// Calculate variance
	if len(data) == 1 {
		return mean, 1.0 // Small default variance for single point
	}
	
	variance := 0.0
	for _, val := range data {
		variance += (val - mean) * (val - mean)
	}
	variance /= float64(len(data) - 1)
	
	return mean, math.Max(variance, 1e-10) // Avoid zero variance
}

// GetStats returns current CUSUM statistics
func (c *CUSUM) GetStats() CUSUMStats {
	return CUSUMStats{
		Threshold:       c.threshold,
		Drift:           c.drift,
		Reference:       c.reference,
		CurrentMean:     c.mean,
		CurrentStdDev:   c.stdDev,
		PositiveSum:     c.positiveSum,
		NegativeSum:     c.negativeSum,
		AnomalyCount:    c.anomalyCount,
		FalsePositives:  c.falsePositives,
		Observations:    len(c.observations),
		UpdateCount:     c.updateCount,
		LastAnomaly:     c.lastAnomaly,
		IsAdaptive:      c.adaptive,
	}
}

// Reset resets the CUSUM detector to initial state
func (c *CUSUM) Reset() {
	c.cumulativeSum = 0.0
	c.positiveSum = 0.0
	c.negativeSum = 0.0
	c.observations = make([]float64, 0)
	c.anomalyCount = 0
	c.falsePositives = 0
	c.updateCount = 0
	c.lastAnomaly = time.Time{}
}

// SetParameters updates CUSUM parameters
func (c *CUSUM) SetParameters(threshold, drift, reference float64) {
	c.threshold = threshold
	c.drift = drift
	c.reference = reference
}

// EstimateParameters estimates optimal parameters from historical data
func (c *CUSUM) EstimateParameters(data []float64, targetARL float64) error {
	if len(data) < 10 {
		return fmt.Errorf("insufficient data for parameter estimation")
	}
	
	// Calculate sample statistics
	mean, variance := c.calculateMeanVar(data)
	stdDev := math.Sqrt(variance)
	
	c.reference = mean
	c.drift = 0.5 * stdDev
	
	// Estimate threshold for target ARL (Average Run Length)
	// This is a simplified estimation - more sophisticated methods exist
	c.threshold = c.drift * (targetARL - 1.0)
	
	if c.threshold < 2.0*stdDev {
		c.threshold = 2.0 * stdDev
	}
	
	return nil
}

// AnomalyType represents the type of anomaly detected
type AnomalyType int

const (
	AnomalyNone AnomalyType = iota
	AnomalyUpward
	AnomalyDownward
)

// String returns string representation of anomaly type
func (at AnomalyType) String() string {
	switch at {
	case AnomalyUpward:
		return "upward"
	case AnomalyDownward:
		return "downward"
	default:
		return "none"
	}
}

// CUSUMResult represents the result of CUSUM anomaly detection
type CUSUMResult struct {
	Value           float64     `json:"value"`             // Input value
	CumulativeSum   float64     `json:"cumulative_sum"`    // Overall cumulative sum
	PositiveSum     float64     `json:"positive_sum"`      // Positive cumulative sum (C+)
	NegativeSum     float64     `json:"negative_sum"`      // Negative cumulative sum (C-)
	AnomalyType     AnomalyType `json:"anomaly_type"`      // Type of anomaly detected
	AnomalySeverity float64     `json:"anomaly_severity"`  // Severity (ratio to threshold)
	IsAnomaly       bool        `json:"is_anomaly"`        // Whether anomaly detected
	Timestamp       time.Time   `json:"timestamp"`         // Detection timestamp
}

// CUSUMStats represents statistics about the CUSUM detector
type CUSUMStats struct {
	Threshold       float64   `json:"threshold"`         // Detection threshold (h)
	Drift           float64   `json:"drift"`             // Drift parameter (k)
	Reference       float64   `json:"reference"`         // Reference value (μ0)
	CurrentMean     float64   `json:"current_mean"`      // Sample mean
	CurrentStdDev   float64   `json:"current_std_dev"`   // Sample standard deviation
	PositiveSum     float64   `json:"positive_sum"`      // Current C+
	NegativeSum     float64   `json:"negative_sum"`      // Current C-
	AnomalyCount    int       `json:"anomaly_count"`     // Total anomalies detected
	FalsePositives  int       `json:"false_positives"`   // Estimated false positives
	Observations    int       `json:"observations"`      // Number of observations
	UpdateCount     int       `json:"update_count"`      // Total updates
	LastAnomaly     time.Time `json:"last_anomaly"`      // Last anomaly time
	IsAdaptive      bool      `json:"is_adaptive"`       // Whether using adaptive parameters
}