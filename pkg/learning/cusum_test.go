package learning

import (
	"math"
	"testing"
	"time"
)

func TestCUSUM_NewCUSUM(t *testing.T) {
	threshold := 5.0
	drift := 0.5
	reference := 10.0
	
	cusum := NewCUSUM(threshold, drift, reference)
	if cusum == nil {
		t.Fatal("NewCUSUM() returned nil")
	}
	
	if cusum.threshold != threshold {
		t.Errorf("Expected threshold=%f, got %f", threshold, cusum.threshold)
	}
	
	if cusum.drift != drift {
		t.Errorf("Expected drift=%f, got %f", drift, cusum.drift)
	}
	
	if cusum.reference != reference {
		t.Errorf("Expected reference=%f, got %f", reference, cusum.reference)
	}
	
	if cusum.positiveSum != 0.0 {
		t.Errorf("Expected initial positiveSum=0.0, got %f", cusum.positiveSum)
	}
	
	if cusum.negativeSum != 0.0 {
		t.Errorf("Expected initial negativeSum=0.0, got %f", cusum.negativeSum)
	}
}

func TestCUSUM_NewCUSUMFromSigma(t *testing.T) {
	sigma := 2.0
	reference := 10.0
	
	cusum := NewCUSUMFromSigma(sigma, reference)
	if cusum == nil {
		t.Fatal("NewCUSUMFromSigma() returned nil")
	}
	
	expectedThreshold := 5.0 * sigma
	expectedDrift := 0.5 * sigma
	
	if cusum.threshold != expectedThreshold {
		t.Errorf("Expected threshold=%f, got %f", expectedThreshold, cusum.threshold)
	}
	
	if cusum.drift != expectedDrift {
		t.Errorf("Expected drift=%f, got %f", expectedDrift, cusum.drift)
	}
	
	if cusum.reference != reference {
		t.Errorf("Expected reference=%f, got %f", reference, cusum.reference)
	}
}

func TestCUSUM_NewCUSUMAdaptive(t *testing.T) {
	cusum := NewCUSUMAdaptive()
	if cusum == nil {
		t.Fatal("NewCUSUMAdaptive() returned nil")
	}
	
	if !cusum.adaptive {
		t.Error("Expected adaptive=true for adaptive CUSUM")
	}
	
	if cusum.maxHistory != 100 {
		t.Errorf("Expected maxHistory=100, got %d", cusum.maxHistory)
	}
}

func TestCUSUM_UpdateNormalData(t *testing.T) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	
	// Add normal observations around reference
	normalValues := []float64{10.1, 9.9, 10.2, 9.8, 10.0}
	
	for _, value := range normalValues {
		result := cusum.Update(value)
		
		if result.IsAnomaly {
			t.Errorf("Expected no anomaly for normal value %f, got anomaly: %s", 
				value, result.AnomalyType.String())
		}
		
		if result.Value != value {
			t.Errorf("Expected result value=%f, got %f", value, result.Value)
		}
		
		if math.IsNaN(result.PositiveSum) || math.IsInf(result.PositiveSum, 0) {
			t.Errorf("PositiveSum is invalid: %f", result.PositiveSum)
		}
		
		if math.IsNaN(result.NegativeSum) || math.IsInf(result.NegativeSum, 0) {
			t.Errorf("NegativeSum is invalid: %f", result.NegativeSum)
		}
	}
}

func TestCUSUM_UpdateUpwardAnomaly(t *testing.T) {
	cusum := NewCUSUM(2.0, 0.5, 10.0) // Lower threshold for easier testing
	
	// Add values that create upward anomaly
	values := []float64{10.0, 12.0, 14.0, 16.0, 18.0, 20.0}
	
	anomalyDetected := false
	for _, value := range values {
		result := cusum.Update(value)
		
		if result.IsAnomaly && result.AnomalyType == AnomalyUpward {
			anomalyDetected = true
			
			if result.AnomalySeverity <= 1.0 {
				t.Errorf("Expected anomaly severity > 1.0, got %f", result.AnomalySeverity)
			}
			break
		}
	}
	
	if !anomalyDetected {
		t.Error("Expected upward anomaly to be detected")
	}
	
	if cusum.anomalyCount == 0 {
		t.Error("Expected anomaly count > 0")
	}
}

func TestCUSUM_UpdateDownwardAnomaly(t *testing.T) {
	cusum := NewCUSUM(2.0, 0.5, 10.0) // Lower threshold for easier testing
	
	// Add values that create downward anomaly
	values := []float64{10.0, 8.0, 6.0, 4.0, 2.0, 0.0}
	
	anomalyDetected := false
	for _, value := range values {
		result := cusum.Update(value)
		
		if result.IsAnomaly && result.AnomalyType == AnomalyDownward {
			anomalyDetected = true
			
			if result.AnomalySeverity <= 1.0 {
				t.Errorf("Expected anomaly severity > 1.0, got %f", result.AnomalySeverity)
			}
			break
		}
	}
	
	if !anomalyDetected {
		t.Error("Expected downward anomaly to be detected")
	}
}

func TestCUSUM_CumulativeSumCalculation(t *testing.T) {
	cusum := NewCUSUM(10.0, 0.5, 5.0)
	
	// Test positive deviation
	result := cusum.Update(7.0)
	expectedDeviation := 7.0 - 5.0 // 2.0
	expectedPositive := math.Max(0, 0+expectedDeviation-0.5) // 1.5
	
	if math.Abs(result.PositiveSum-expectedPositive) > 0.0001 {
		t.Errorf("Expected positiveSum=%f, got %f", expectedPositive, result.PositiveSum)
	}
	
	// Test negative deviation
	result = cusum.Update(3.0)
	expectedDeviation = 3.0 - 5.0 // -2.0
	expectedNegative := math.Max(0, 0-expectedDeviation-0.5) // 1.5
	
	if math.Abs(result.NegativeSum-expectedNegative) > 0.0001 {
		t.Errorf("Expected negativeSum=%f, got %f", expectedNegative, result.NegativeSum)
	}
}

func TestCUSUM_BatchDetect(t *testing.T) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	
	values := []float64{10.0, 11.0, 9.0, 12.0, 8.0}
	results := cusum.BatchDetect(values)
	
	if len(results) != len(values) {
		t.Errorf("Expected %d results, got %d", len(values), len(results))
	}
	
	for i, result := range results {
		if result.Value != values[i] {
			t.Errorf("Result %d: expected value=%f, got %f", i, values[i], result.Value)
		}
		
		if result.Timestamp.IsZero() {
			t.Errorf("Result %d: timestamp should not be zero", i)
		}
	}
}

func TestCUSUM_Reset(t *testing.T) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	
	// Add some data
	cusum.Update(15.0)
	cusum.Update(20.0)
	cusum.Update(25.0)
	
	// Verify state before reset (note: positiveSum might be 0 depending on data and drift)
	if len(cusum.observations) == 0 {
		t.Error("Expected observations > 0 before reset")
	}
	
	// Reset
	cusum.Reset()
	
	// Verify reset state
	if cusum.positiveSum != 0.0 {
		t.Errorf("Expected positiveSum=0.0 after reset, got %f", cusum.positiveSum)
	}
	
	if cusum.negativeSum != 0.0 {
		t.Errorf("Expected negativeSum=0.0 after reset, got %f", cusum.negativeSum)
	}
	
	if cusum.cumulativeSum != 0.0 {
		t.Errorf("Expected cumulativeSum=0.0 after reset, got %f", cusum.cumulativeSum)
	}
	
	if len(cusum.observations) != 0 {
		t.Errorf("Expected 0 observations after reset, got %d", len(cusum.observations))
	}
	
	if cusum.anomalyCount != 0 {
		t.Errorf("Expected anomalyCount=0 after reset, got %d", cusum.anomalyCount)
	}
}

func TestCUSUM_SetParameters(t *testing.T) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	
	newThreshold := 8.0
	newDrift := 1.0
	newReference := 15.0
	
	cusum.SetParameters(newThreshold, newDrift, newReference)
	
	if cusum.threshold != newThreshold {
		t.Errorf("Expected threshold=%f, got %f", newThreshold, cusum.threshold)
	}
	
	if cusum.drift != newDrift {
		t.Errorf("Expected drift=%f, got %f", newDrift, cusum.drift)
	}
	
	if cusum.reference != newReference {
		t.Errorf("Expected reference=%f, got %f", newReference, cusum.reference)
	}
}

func TestCUSUM_GetStats(t *testing.T) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	
	// Add some observations
	cusum.Update(12.0)
	cusum.Update(15.0)
	
	stats := cusum.GetStats()
	
	if stats.Threshold != cusum.threshold {
		t.Errorf("Stats threshold mismatch: expected %f, got %f", cusum.threshold, stats.Threshold)
	}
	
	if stats.Drift != cusum.drift {
		t.Errorf("Stats drift mismatch: expected %f, got %f", cusum.drift, stats.Drift)
	}
	
	if stats.Reference != cusum.reference {
		t.Errorf("Stats reference mismatch: expected %f, got %f", cusum.reference, stats.Reference)
	}
	
	if stats.UpdateCount != cusum.updateCount {
		t.Errorf("Stats update count mismatch: expected %d, got %d", cusum.updateCount, stats.UpdateCount)
	}
	
	if stats.Observations != len(cusum.observations) {
		t.Errorf("Stats observations mismatch: expected %d, got %d", len(cusum.observations), stats.Observations)
	}
}

func TestCUSUM_AdaptiveBehavior(t *testing.T) {
	cusum := NewCUSUMAdaptive()
	
	// Add enough data to trigger adaptation
	for i := 0; i < 15; i++ {
		value := 10.0 + float64(i)*0.1 // Gradual increase
		cusum.Update(value)
	}
	
	// Verify that parameters have been adapted
	if cusum.mean == 0.0 {
		t.Error("Expected mean to be calculated in adaptive mode")
	}
	
	if cusum.stdDev == 0.0 {
		t.Error("Expected stdDev to be calculated in adaptive mode")
	}
	
	if cusum.reference == 0.0 {
		t.Error("Expected reference to be updated in adaptive mode")
	}
}

func TestCUSUM_EstimateParameters(t *testing.T) {
	cusum := NewCUSUM(1.0, 0.1, 0.0)
	
	// Generate sample data with known statistics
	data := make([]float64, 100)
	for i := range data {
		data[i] = 10.0 + float64(i%10)*0.5 // Pattern with mean ~12.25
	}
	
	err := cusum.EstimateParameters(data, 10.0)
	if err != nil {
		t.Fatalf("EstimateParameters() failed: %v", err)
	}
	
	// Check that parameters were updated
	if cusum.reference == 0.0 {
		t.Error("Expected reference to be updated by parameter estimation")
	}
	
	if cusum.drift <= 0.0 {
		t.Error("Expected drift > 0 after parameter estimation")
	}
	
	if cusum.threshold <= 0.0 {
		t.Error("Expected threshold > 0 after parameter estimation")
	}
}

func TestCUSUM_EstimateParametersInsufficientData(t *testing.T) {
	cusum := NewCUSUM(1.0, 0.1, 0.0)
	
	// Try with insufficient data
	data := []float64{1.0, 2.0} // Only 2 points
	
	err := cusum.EstimateParameters(data, 10.0)
	if err == nil {
		t.Error("Expected error for insufficient data")
	}
}

func TestCUSUM_GetChangePoint(t *testing.T) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	
	// Create data with a change point
	for i := 0; i < 20; i++ {
		var value float64
		if i < 10 {
			value = 10.0 + float64(i%3)*0.1 // First regime
		} else {
			value = 15.0 + float64(i%3)*0.1 // Second regime (shift up)
		}
		cusum.Update(value)
	}
	
	changePoint, likelihood := cusum.GetChangePoint(15)
	
	// Should detect a change point (may not be exactly at position 10 due to algorithm behavior)
	if changePoint == -1 {
		t.Error("Expected to detect a change point")
	}
	
	if likelihood <= 0.0 {
		t.Errorf("Expected likelihood > 0, got %f", likelihood)
	}
}

func TestCUSUM_AnomalyTypeString(t *testing.T) {
	testCases := []struct {
		anomalyType AnomalyType
		expected    string
	}{
		{AnomalyNone, "none"},
		{AnomalyUpward, "upward"},
		{AnomalyDownward, "downward"},
	}
	
	for _, tc := range testCases {
		if tc.anomalyType.String() != tc.expected {
			t.Errorf("Expected %s.String()=%s, got %s", 
				tc.anomalyType, tc.expected, tc.anomalyType.String())
		}
	}
}

func TestCUSUM_HistoryLimit(t *testing.T) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	maxHistory := 100
	
	// Add more observations than max history
	for i := 0; i < maxHistory+20; i++ {
		cusum.Update(float64(10 + i%5))
	}
	
	if len(cusum.observations) > maxHistory {
		t.Errorf("Expected observations <= %d, got %d", maxHistory, len(cusum.observations))
	}
}

func TestCUSUM_AnomalyReset(t *testing.T) {
	cusum := NewCUSUM(2.0, 0.5, 10.0)
	
	// Build up positive sum to trigger anomaly
	cusum.Update(15.0)
	cusum.Update(20.0)
	result := cusum.Update(25.0)
	
	if result.IsAnomaly {
		// After anomaly detection, sums should be reset
		if cusum.positiveSum != 0.0 {
			t.Errorf("Expected positiveSum=0.0 after anomaly detection, got %f", cusum.positiveSum)
		}
		
		if cusum.negativeSum != 0.0 {
			t.Errorf("Expected negativeSum=0.0 after anomaly detection, got %f", cusum.negativeSum)
		}
		
		if !cusum.lastAnomaly.After(time.Now().Add(-time.Second)) {
			t.Error("Expected lastAnomaly timestamp to be recent")
		}
	}
}

// Benchmark CUSUM performance
func BenchmarkCUSUM_Update(b *testing.B) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cusum.Update(10.0 + float64(i%100)*0.1)
	}
}

func BenchmarkCUSUM_UpdateAdaptive(b *testing.B) {
	cusum := NewCUSUMAdaptive()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cusum.Update(10.0 + float64(i%100)*0.1)
	}
}

func BenchmarkCUSUM_BatchDetect(b *testing.B) {
	cusum := NewCUSUM(5.0, 0.5, 10.0)
	values := make([]float64, 1000)
	for i := range values {
		values[i] = 10.0 + float64(i%100)*0.1
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cusum.BatchDetect(values)
	}
}