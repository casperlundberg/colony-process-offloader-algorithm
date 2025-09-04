package learning

import (
	"math"
	"testing"
)

func TestARIMA_NewARIMA(t *testing.T) {
	arima := NewARIMA()
	if arima == nil {
		t.Fatal("NewARIMA() returned nil")
	}
	
	if arima.p != 3 {
		t.Errorf("Expected p=3, got %d", arima.p)
	}
	
	if arima.d != 1 {
		t.Errorf("Expected d=1, got %d", arima.d)
	}
	
	if arima.q != 2 {
		t.Errorf("Expected q=2, got %d", arima.q)
	}
}

func TestARIMA_AddObservation(t *testing.T) {
	arima := NewARIMA()
	
	// Add some observations
	observations := []float64{10.0, 12.0, 14.0, 16.0, 18.0}
	for _, obs := range observations {
		arima.AddObservation(obs)
	}
	
	if len(arima.observations) != len(observations) {
		t.Errorf("Expected %d observations, got %d", 
			len(observations), len(arima.observations))
	}
}

func TestARIMA_Predict(t *testing.T) {
	arima := NewARIMA()
	
	// Add a simple linear trend
	for i := 1; i <= 20; i++ {
		arima.AddObservation(float64(i * 10))
	}
	
	// Make prediction
	prediction, err := arima.Predict()
	if err != nil {
		t.Fatalf("Predict() returned error: %v", err)
	}
	
	// Prediction should be positive for increasing trend
	if prediction <= 0 {
		t.Errorf("Expected positive prediction for increasing trend, got %f", prediction)
	}
	
	// Check prediction is reasonable (not NaN or Inf)
	if math.IsNaN(prediction) || math.IsInf(prediction, 0) {
		t.Errorf("Prediction is not a valid number: %f", prediction)
	}
}

func TestARIMA_InsufficientData(t *testing.T) {
	arima := NewARIMA()
	
	// Try to predict with insufficient data
	_, err := arima.Predict()
	if err == nil {
		t.Error("Expected error for prediction with insufficient data")
	}
	
	// Add minimal observations
	arima.AddObservation(10.0)
	arima.AddObservation(20.0)
	
	// Still insufficient for ARIMA(3,1,2)
	_, err = arima.Predict()
	if err == nil {
		t.Error("Expected error for prediction with insufficient data for ARIMA(3,1,2)")
	}
}

// Benchmark for performance testing
func BenchmarkARIMA_Predict(b *testing.B) {
	arima := NewARIMA()
	
	// Add sample data
	for i := 0; i < 100; i++ {
		arima.AddObservation(float64(i) + math.Sin(float64(i)/10)*5)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = arima.Predict()
	}
}