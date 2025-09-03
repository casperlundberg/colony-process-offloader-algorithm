package learning

import (
	"fmt"
	"math"
	"time"
)

// ARIMA implements AutoRegressive Integrated Moving Average (3,1,2) predictor
// Referenced in FURTHER_CONTROL_THEORY.md as [1] for time series prediction
type ARIMA struct {
	p, d, q       int       // ARIMA parameters (3,1,2)
	arCoeffs      []float64 // Autoregressive coefficients
	maCoeffs      []float64 // Moving average coefficients
	observations  []float64 // Historical observations
	residuals     []float64 // Residuals for MA component
	maxHistory    int       // Maximum history to keep
	fitted        bool      // Whether model has been fitted
	lastPrediction float64  // Last prediction made
	lastUpdate    time.Time // Last update timestamp
}

// NewARIMA creates a new ARIMA(3,1,2) predictor
func NewARIMA() *ARIMA {
	return &ARIMA{
		p:            3,  // AR order
		d:            1,  // Differencing order
		q:            2,  // MA order
		arCoeffs:     []float64{0.5, 0.3, 0.2},    // Initial AR coefficients
		maCoeffs:     []float64{0.4, 0.3},         // Initial MA coefficients
		observations: make([]float64, 0),
		residuals:    make([]float64, 0),
		maxHistory:   100,
		fitted:       false,
		lastUpdate:   time.Now(),
	}
}

// AddObservation adds a new observation to the time series
func (a *ARIMA) AddObservation(value float64) {
	a.observations = append(a.observations, value)
	
	// Keep only maxHistory observations
	if len(a.observations) > a.maxHistory {
		a.observations = a.observations[len(a.observations)-a.maxHistory:]
	}
	
	// Update residuals if we have a previous prediction
	if a.fitted && len(a.observations) > 1 {
		actual := value
		predicted := a.lastPrediction
		residual := actual - predicted
		
		a.residuals = append(a.residuals, residual)
		if len(a.residuals) > a.maxHistory {
			a.residuals = a.residuals[len(a.residuals)-a.maxHistory:]
		}
	}
	
	a.lastUpdate = time.Now()
}

// Predict makes a prediction for the next value in the series
func (a *ARIMA) Predict() (float64, error) {
	if len(a.observations) < a.p+a.d {
		return 0, fmt.Errorf("insufficient data: need at least %d observations, have %d", 
			a.p+a.d, len(a.observations))
	}
	
	// Apply differencing (d=1)
	diffSeries := a.difference(a.observations)
	
	if len(diffSeries) < a.p {
		return 0, fmt.Errorf("insufficient differenced data")
	}
	
	// AR component: linear combination of previous p values
	arComponent := 0.0
	for i := 0; i < a.p; i++ {
		if i < len(diffSeries) {
			arComponent += a.arCoeffs[i] * diffSeries[len(diffSeries)-1-i]
		}
	}
	
	// MA component: linear combination of previous q residuals
	maComponent := 0.0
	if len(a.residuals) >= a.q {
		for i := 0; i < a.q; i++ {
			if i < len(a.residuals) {
				maComponent += a.maCoeffs[i] * a.residuals[len(a.residuals)-1-i]
			}
		}
	}
	
	// Combined prediction on differenced series
	diffPrediction := arComponent + maComponent
	
	// Integrate back (reverse differencing)
	if len(a.observations) > 0 {
		prediction := a.observations[len(a.observations)-1] + diffPrediction
		a.lastPrediction = prediction
		a.fitted = true
		return prediction, nil
	}
	
	return diffPrediction, nil
}

// PredictNext predicts the next n values
func (a *ARIMA) PredictNext(n int) ([]float64, error) {
	if n <= 0 {
		return nil, fmt.Errorf("prediction horizon must be positive")
	}
	
	predictions := make([]float64, n)
	
	// Save original state
	originalObs := make([]float64, len(a.observations))
	copy(originalObs, a.observations)
	originalResiduals := make([]float64, len(a.residuals))
	copy(originalResiduals, a.residuals)
	
	// Make iterative predictions
	for i := 0; i < n; i++ {
		pred, err := a.Predict()
		if err != nil {
			return nil, fmt.Errorf("prediction %d failed: %w", i+1, err)
		}
		
		predictions[i] = pred
		
		// Add prediction as observation for next iteration
		a.AddObservation(pred)
	}
	
	// Restore original state
	a.observations = originalObs
	a.residuals = originalResiduals
	
	return predictions, nil
}

// Fit estimates ARIMA coefficients using method of moments (simplified)
func (a *ARIMA) Fit() error {
	if len(a.observations) < a.p+a.d+a.q+10 {
		return fmt.Errorf("insufficient data for fitting: need at least %d observations", 
			a.p+a.d+a.q+10)
	}
	
	// Apply differencing
	diffSeries := a.difference(a.observations)
	
	// Estimate AR coefficients using Yule-Walker equations (simplified)
	a.estimateARCoefficients(diffSeries)
	
	// Estimate MA coefficients using residuals
	a.estimateMACoefficients(diffSeries)
	
	a.fitted = true
	return nil
}

// difference applies differencing to the series
func (a *ARIMA) difference(series []float64) []float64 {
	if len(series) <= a.d {
		return []float64{}
	}
	
	result := make([]float64, len(series))
	copy(result, series)
	
	// Apply differencing d times
	for diff := 0; diff < a.d; diff++ {
		if len(result) <= 1 {
			break
		}
		
		newResult := make([]float64, len(result)-1)
		for i := 1; i < len(result); i++ {
			newResult[i-1] = result[i] - result[i-1]
		}
		result = newResult
	}
	
	return result
}

// estimateARCoefficients estimates AR coefficients using autocorrelation
func (a *ARIMA) estimateARCoefficients(series []float64) {
	if len(series) < a.p+1 {
		return
	}
	
	// Calculate sample autocorrelations
	autocorrs := make([]float64, a.p)
	n := len(series)
	
	// Calculate mean
	mean := 0.0
	for _, val := range series {
		mean += val
	}
	mean /= float64(n)
	
	// Calculate variance
	variance := 0.0
	for _, val := range series {
		variance += (val - mean) * (val - mean)
	}
	variance /= float64(n)
	
	if variance == 0 {
		return
	}
	
	// Calculate autocorrelations
	for k := 0; k < a.p; k++ {
		lag := k + 1
		covariance := 0.0
		
		for i := 0; i < n-lag; i++ {
			covariance += (series[i] - mean) * (series[i+lag] - mean)
		}
		covariance /= float64(n - lag)
		
		autocorrs[k] = covariance / variance
	}
	
	// Solve Yule-Walker equations (simplified - use autocorrelations directly)
	for i := 0; i < a.p && i < len(autocorrs); i++ {
		a.arCoeffs[i] = autocorrs[i] * 0.8 // Damping factor for stability
	}
}

// estimateMACoefficients estimates MA coefficients (simplified approach)
func (a *ARIMA) estimateMACoefficients(series []float64) {
	// For simplicity, use method of moments approximation
	// In practice, would use maximum likelihood estimation
	
	if len(series) < a.q+1 {
		return
	}
	
	// Calculate residuals from AR fit
	residuals := make([]float64, 0)
	
	for i := a.p; i < len(series); i++ {
		predicted := 0.0
		for j := 0; j < a.p; j++ {
			predicted += a.arCoeffs[j] * series[i-1-j]
		}
		
		residual := series[i] - predicted
		residuals = append(residuals, residual)
	}
	
	// Use autocorrelations of residuals for MA coefficients
	if len(residuals) >= a.q {
		mean := 0.0
		for _, r := range residuals {
			mean += r
		}
		mean /= float64(len(residuals))
		
		for k := 0; k < a.q && k < len(residuals)-1; k++ {
			correlation := 0.0
			count := 0
			
			for i := 0; i < len(residuals)-k-1; i++ {
				correlation += (residuals[i] - mean) * (residuals[i+k+1] - mean)
				count++
			}
			
			if count > 0 {
				a.maCoeffs[k] = (correlation / float64(count)) * 0.6 // Damping
			}
		}
	}
}

// GetModel returns the current model parameters
func (a *ARIMA) GetModel() ARIMAModel {
	return ARIMAModel{
		P:            a.p,
		D:            a.d,
		Q:            a.q,
		ARCoeffs:     append([]float64(nil), a.arCoeffs...),
		MACoeffs:     append([]float64(nil), a.maCoeffs...),
		Fitted:       a.fitted,
		Observations: len(a.observations),
		LastUpdate:   a.lastUpdate,
	}
}

// GetForecastAccuracy calculates forecast accuracy metrics
func (a *ARIMA) GetForecastAccuracy(actual []float64, predictions []float64) ForecastAccuracy {
	if len(actual) != len(predictions) || len(actual) == 0 {
		return ForecastAccuracy{}
	}
	
	n := float64(len(actual))
	mae := 0.0  // Mean Absolute Error
	mse := 0.0  // Mean Squared Error
	mape := 0.0 // Mean Absolute Percentage Error
	
	for i := 0; i < len(actual); i++ {
		error := actual[i] - predictions[i]
		mae += math.Abs(error)
		mse += error * error
		
		if actual[i] != 0 {
			mape += math.Abs(error / actual[i])
		}
	}
	
	mae /= n
	mse /= n
	mape = (mape / n) * 100
	rmse := math.Sqrt(mse)
	
	return ForecastAccuracy{
		MAE:  mae,
		MSE:  mse,
		RMSE: rmse,
		MAPE: mape,
	}
}

// ARIMAModel represents the ARIMA model parameters
type ARIMAModel struct {
	P            int       `json:"p"`             // AR order
	D            int       `json:"d"`             // Differencing order  
	Q            int       `json:"q"`             // MA order
	ARCoeffs     []float64 `json:"ar_coeffs"`     // AR coefficients
	MACoeffs     []float64 `json:"ma_coeffs"`     // MA coefficients
	Fitted       bool      `json:"fitted"`        // Whether model is fitted
	Observations int       `json:"observations"`  // Number of observations
	LastUpdate   time.Time `json:"last_update"`   // Last update time
}

// ForecastAccuracy represents forecast accuracy metrics
type ForecastAccuracy struct {
	MAE  float64 `json:"mae"`  // Mean Absolute Error
	MSE  float64 `json:"mse"`  // Mean Squared Error
	RMSE float64 `json:"rmse"` // Root Mean Squared Error
	MAPE float64 `json:"mape"` // Mean Absolute Percentage Error (%)
}

// Reset resets the ARIMA model to initial state
func (a *ARIMA) Reset() {
	a.observations = make([]float64, 0)
	a.residuals = make([]float64, 0)
	a.fitted = false
	a.lastPrediction = 0.0
	a.lastUpdate = time.Now()
	
	// Reset to default coefficients
	a.arCoeffs = []float64{0.5, 0.3, 0.2}
	a.maCoeffs = []float64{0.4, 0.3}
}