package learning

import (
	"math"
	"testing"
)

func TestEWMA_NewEWMA(t *testing.T) {
	// Test with valid alpha
	ewma := NewEWMA(0.2)
	if ewma == nil {
		t.Fatal("NewEWMA() returned nil")
	}
	
	if ewma.alpha != 0.2 {
		t.Errorf("Expected alpha=0.2, got %f", ewma.alpha)
	}
	
	if ewma.initialized {
		t.Error("EWMA should not be initialized on creation")
	}
}

func TestEWMA_NewEWMADefault(t *testing.T) {
	ewma := NewEWMADefault()
	if ewma == nil {
		t.Fatal("NewEWMADefault() returned nil")
	}
	
	if ewma.alpha != 0.167 {
		t.Errorf("Expected default alpha=0.167, got %f", ewma.alpha)
	}
}

func TestEWMA_InvalidAlpha(t *testing.T) {
	testCases := []float64{-0.1, 0.0, 1.5, 2.0}
	
	for _, alpha := range testCases {
		ewma := NewEWMA(alpha)
		if ewma.alpha != 0.167 {
			t.Errorf("Invalid alpha %f should default to 0.167, got %f", 
				alpha, ewma.alpha)
		}
	}
}

func TestEWMA_Update(t *testing.T) {
	ewma := NewEWMA(0.5) // Use 0.5 for easy calculation
	
	// First update should set the value directly
	result := ewma.Update(10.0)
	if result != 10.0 {
		t.Errorf("First update should return input value, got %f", result)
	}
	
	if !ewma.initialized {
		t.Error("EWMA should be initialized after first update")
	}
	
	// Second update should apply EWMA formula
	result = ewma.Update(20.0)
	expected := 0.5*20.0 + (1-0.5)*10.0 // 15.0
	if math.Abs(result-expected) > 0.0001 {
		t.Errorf("Expected EWMA value %f, got %f", expected, result)
	}
}

func TestEWMA_UpdateSequence(t *testing.T) {
	ewma := NewEWMA(0.2)
	values := []float64{10, 15, 12, 18, 20}
	
	var lastEWMA float64
	for i, val := range values {
		result := ewma.Update(val)
		
		if i == 0 {
			// First value should be returned as-is
			if result != val {
				t.Errorf("First value should be %f, got %f", val, result)
			}
		} else {
			// Subsequent values should be smoothed
			expected := 0.2*val + 0.8*lastEWMA
			if math.Abs(result-expected) > 0.0001 {
				t.Errorf("Step %d: expected %f, got %f", i, expected, result)
			}
		}
		lastEWMA = result
	}
}

func TestEWMA_GetCurrent(t *testing.T) {
	ewma := NewEWMA(0.3)
	
	// Before any updates
	current := ewma.GetCurrent()
	if current != 0.0 {
		t.Errorf("Initial current value should be 0.0, got %f", current)
	}
	
	// After updates
	ewma.Update(100.0)
	current = ewma.GetCurrent()
	if current != 100.0 {
		t.Errorf("After first update, current should be 100.0, got %f", current)
	}
	
	ewma.Update(50.0)
	current = ewma.GetCurrent()
	expected := 0.3*50.0 + 0.7*100.0 // 85.0
	if math.Abs(current-expected) > 0.0001 {
		t.Errorf("Expected current value %f, got %f", expected, current)
	}
}

func TestEWMA_Reset(t *testing.T) {
	ewma := NewEWMA(0.4)
	
	// Add some values
	ewma.Update(10.0)
	ewma.Update(20.0)
	ewma.Update(30.0)
	
	if !ewma.initialized {
		t.Error("EWMA should be initialized after updates")
	}
	
	// Reset
	ewma.Reset()
	
	if ewma.initialized {
		t.Error("EWMA should not be initialized after reset")
	}
	
	if ewma.currentEWMA != 0.0 {
		t.Errorf("Current EWMA should be 0.0 after reset, got %f", ewma.currentEWMA)
	}
	
	if ewma.valueCount != 0 {
		t.Errorf("Value count should be 0 after reset, got %d", ewma.valueCount)
	}
}

func TestEWMA_SmoothingBehavior(t *testing.T) {
	ewma := NewEWMA(0.1) // Low alpha for heavy smoothing
	
	// Start with a value
	ewma.Update(100.0)
	
	// Add a spike
	result := ewma.Update(1000.0)
	
	// With alpha=0.1, the spike should be heavily damped
	// Expected: 0.1 * 1000 + 0.9 * 100 = 190
	if result >= 500.0 { // Should be much less than halfway
		t.Errorf("EWMA with low alpha should heavily smooth spikes, got %f", result)
	}
}

func TestEWMA_HighAlphaBehavior(t *testing.T) {
	ewma := NewEWMA(0.9) // High alpha for light smoothing
	
	// Start with a value
	ewma.Update(100.0)
	
	// Add a different value
	result := ewma.Update(200.0)
	
	// With alpha=0.9, should follow the new value closely
	// Expected: 0.9 * 200 + 0.1 * 100 = 190
	expected := 0.9*200.0 + 0.1*100.0
	if math.Abs(result-expected) > 0.0001 {
		t.Errorf("Expected %f with high alpha, got %f", expected, result)
	}
}

// Benchmark EWMA performance
func BenchmarkEWMA_Update(b *testing.B) {
	ewma := NewEWMA(0.167)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ewma.Update(float64(i))
	}
}

func BenchmarkEWMA_UpdateAndGet(b *testing.B) {
	ewma := NewEWMA(0.167)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ewma.Update(float64(i))
		_ = ewma.GetCurrent()
	}
}