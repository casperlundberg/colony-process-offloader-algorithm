package learning

import (
	"fmt"
	"math"
	"time"
)

// SGD implements Stochastic Gradient Descent optimizer
// Referenced in FURTHER_CONTROL_THEORY.md as [4] with learning rate 0.001
type SGD struct {
	learningRate float64   // Learning rate (α)
	momentum     float64   // Momentum parameter for SGD with momentum
	decay        float64   // Weight decay (L2 regularization)
	
	// Momentum state
	velocities   []float64 // Velocity vectors for momentum
	
	// Adaptive learning rate
	adaptive     bool      // Whether to use adaptive learning rate
	gradientSum  []float64 // Sum of squared gradients (for AdaGrad-style adaptation)
	
	// Statistics
	updateCount  int       // Number of parameter updates
	avgGradNorm  float64   // Average gradient norm
	lastUpdate   time.Time // Last update timestamp
	
	// Convergence tracking
	costHistory  []float64 // History of cost values
	converged    bool      // Whether optimizer has converged
	tolerance    float64   // Convergence tolerance
	patience     int       // Patience for early stopping
	
	// Parameters being optimized
	parameters   []float64 // Current parameter values
	bestParams   []float64 // Best parameters found
	bestCost     float64   // Best cost achieved
}

// NewSGD creates a new SGD optimizer
func NewSGD(learningRate float64, numParams int) *SGD {
	if learningRate <= 0 {
		learningRate = 0.001 // Default from FURTHER_CONTROL_THEORY.md
	}
	
	return &SGD{
		learningRate: learningRate,
		momentum:     0.0,
		decay:        0.0,
		velocities:   make([]float64, numParams),
		adaptive:     false,
		gradientSum:  make([]float64, numParams),
		updateCount:  0,
		lastUpdate:   time.Now(),
		costHistory:  make([]float64, 0),
		converged:    false,
		tolerance:    1e-6,
		patience:     10,
		parameters:   make([]float64, numParams),
		bestParams:   make([]float64, numParams),
		bestCost:     math.Inf(1),
	}
}

// NewSGDWithMomentum creates SGD with momentum
func NewSGDWithMomentum(learningRate, momentum float64, numParams int) *SGD {
	sgd := NewSGD(learningRate, numParams)
	sgd.momentum = momentum
	return sgd
}

// NewSGDAdaptive creates SGD with adaptive learning rate
func NewSGDAdaptive(learningRate float64, numParams int) *SGD {
	sgd := NewSGD(learningRate, numParams)
	sgd.adaptive = true
	return sgd
}

// Update performs a single SGD update step
func (sgd *SGD) Update(parameters []float64, gradients []float64, cost float64) error {
	if len(parameters) != len(gradients) || len(parameters) != len(sgd.parameters) {
		return fmt.Errorf("parameter and gradient dimensions mismatch")
	}
	
	sgd.updateCount++
	sgd.lastUpdate = time.Now()
	sgd.costHistory = append(sgd.costHistory, cost)
	
	// Keep cost history bounded
	if len(sgd.costHistory) > 100 {
		sgd.costHistory = sgd.costHistory[len(sgd.costHistory)-100:]
	}
	
	// Update best parameters if cost improved
	if cost < sgd.bestCost {
		sgd.bestCost = cost
		copy(sgd.bestParams, parameters)
	}
	
	// Calculate gradient norm for statistics
	gradNorm := sgd.calculateGradientNorm(gradients)
	sgd.updateGradientNorm(gradNorm)
	
	// Update parameters
	for i := range parameters {
		sgd.updateParameter(i, parameters[i], gradients[i])
	}
	
	// Copy updated parameters back
	copy(parameters, sgd.parameters)
	
	// Check convergence
	sgd.checkConvergence()
	
	return nil
}

// updateParameter updates a single parameter using SGD with optional momentum
func (sgd *SGD) updateParameter(index int, param, gradient float64) {
	// Apply weight decay (L2 regularization)
	if sgd.decay > 0 {
		gradient += sgd.decay * param
	}
	
	// Adaptive learning rate (AdaGrad-style)
	effectiveLR := sgd.learningRate
	if sgd.adaptive {
		sgd.gradientSum[index] += gradient * gradient
		if sgd.gradientSum[index] > 0 {
			effectiveLR = sgd.learningRate / (math.Sqrt(sgd.gradientSum[index]) + 1e-8)
		}
	}
	
	// SGD with momentum
	if sgd.momentum > 0 {
		sgd.velocities[index] = sgd.momentum*sgd.velocities[index] - effectiveLR*gradient
		sgd.parameters[index] = param + sgd.velocities[index]
	} else {
		// Standard SGD
		sgd.parameters[index] = param - effectiveLR*gradient
	}
}

// calculateGradientNorm calculates L2 norm of gradient vector
func (sgd *SGD) calculateGradientNorm(gradients []float64) float64 {
	sum := 0.0
	for _, grad := range gradients {
		sum += grad * grad
	}
	return math.Sqrt(sum)
}

// updateGradientNorm updates running average of gradient norm
func (sgd *SGD) updateGradientNorm(gradNorm float64) {
	if sgd.updateCount == 1 {
		sgd.avgGradNorm = gradNorm
	} else {
		// Exponential moving average
		alpha := 0.1
		sgd.avgGradNorm = alpha*gradNorm + (1-alpha)*sgd.avgGradNorm
	}
}

// checkConvergence checks if the optimizer has converged
func (sgd *SGD) checkConvergence() {
	if len(sgd.costHistory) < sgd.patience+1 {
		return
	}
	
	// Check if cost has improved in the last 'patience' iterations
	recent := sgd.costHistory[len(sgd.costHistory)-sgd.patience-1:]
	minRecent := recent[0]
	
	improved := false
	for i := 1; i < len(recent); i++ {
		if recent[i] < minRecent-sgd.tolerance {
			improved = true
			minRecent = recent[i]
		}
	}
	
	sgd.converged = !improved
}

// BatchUpdate performs multiple SGD updates on a batch of data
func (sgd *SGD) BatchUpdate(parameterBatch [][]float64, gradientBatch [][]float64, costs []float64) error {
	if len(parameterBatch) != len(gradientBatch) || len(parameterBatch) != len(costs) {
		return fmt.Errorf("batch sizes mismatch")
	}
	
	for i := range parameterBatch {
		err := sgd.Update(parameterBatch[i], gradientBatch[i], costs[i])
		if err != nil {
			return fmt.Errorf("batch update %d failed: %w", i, err)
		}
	}
	
	return nil
}

// MiniBatchUpdate performs mini-batch SGD by averaging gradients
func (sgd *SGD) MiniBatchUpdate(parameterBatch [][]float64, gradientBatch [][]float64, costs []float64) error {
	if len(parameterBatch) == 0 || len(gradientBatch) == 0 {
		return fmt.Errorf("empty batch")
	}
	
	if len(parameterBatch) != len(gradientBatch) || len(parameterBatch) != len(costs) {
		return fmt.Errorf("batch sizes mismatch")
	}
	
	// Average parameters and gradients across the mini-batch
	avgParams := make([]float64, len(parameterBatch[0]))
	avgGrads := make([]float64, len(gradientBatch[0]))
	avgCost := 0.0
	
	batchSize := float64(len(parameterBatch))
	
	for i := range parameterBatch {
		avgCost += costs[i]
		
		for j := range parameterBatch[i] {
			avgParams[j] += parameterBatch[i][j]
			avgGrads[j] += gradientBatch[i][j]
		}
	}
	
	// Normalize by batch size
	avgCost /= batchSize
	for j := range avgParams {
		avgParams[j] /= batchSize
		avgGrads[j] /= batchSize
	}
	
	// Perform single update with averaged values
	return sgd.Update(avgParams, avgGrads, avgCost)
}

// GetCurrentParameters returns current parameter values
func (sgd *SGD) GetCurrentParameters() []float64 {
	result := make([]float64, len(sgd.parameters))
	copy(result, sgd.parameters)
	return result
}

// GetBestParameters returns best parameters found so far
func (sgd *SGD) GetBestParameters() []float64 {
	result := make([]float64, len(sgd.bestParams))
	copy(result, sgd.bestParams)
	return result
}

// GetStats returns optimizer statistics
func (sgd *SGD) GetStats() SGDStats {
	convergenceRate := 0.0
	if len(sgd.costHistory) > 1 {
		initial := sgd.costHistory[0]
		current := sgd.costHistory[len(sgd.costHistory)-1]
		if initial != 0 {
			convergenceRate = (initial - current) / initial
		}
	}
	
	return SGDStats{
		LearningRate:    sgd.learningRate,
		Momentum:        sgd.momentum,
		Decay:           sgd.decay,
		UpdateCount:     sgd.updateCount,
		AvgGradNorm:     sgd.avgGradNorm,
		BestCost:        sgd.bestCost,
		CurrentCost:     sgd.getCurrentCost(),
		ConvergenceRate: convergenceRate,
		Converged:       sgd.converged,
		IsAdaptive:      sgd.adaptive,
		LastUpdate:      sgd.lastUpdate,
	}
}

// getCurrentCost returns the most recent cost value
func (sgd *SGD) getCurrentCost() float64 {
	if len(sgd.costHistory) == 0 {
		return math.Inf(1)
	}
	return sgd.costHistory[len(sgd.costHistory)-1]
}

// SetLearningRate updates the learning rate
func (sgd *SGD) SetLearningRate(lr float64) {
	if lr > 0 {
		sgd.learningRate = lr
	}
}

// SetMomentum updates the momentum parameter
func (sgd *SGD) SetMomentum(momentum float64) {
	if momentum >= 0 && momentum < 1 {
		sgd.momentum = momentum
	}
}

// SetDecay updates the weight decay parameter
func (sgd *SGD) SetDecay(decay float64) {
	if decay >= 0 {
		sgd.decay = decay
	}
}

// Reset resets the optimizer to initial state
func (sgd *SGD) Reset() {
	for i := range sgd.velocities {
		sgd.velocities[i] = 0.0
		sgd.gradientSum[i] = 0.0
		sgd.parameters[i] = 0.0
		sgd.bestParams[i] = 0.0
	}
	
	sgd.updateCount = 0
	sgd.avgGradNorm = 0.0
	sgd.costHistory = make([]float64, 0)
	sgd.converged = false
	sgd.bestCost = math.Inf(1)
	sgd.lastUpdate = time.Now()
}

// GetCostHistory returns the cost history
func (sgd *SGD) GetCostHistory() []float64 {
	result := make([]float64, len(sgd.costHistory))
	copy(result, sgd.costHistory)
	return result
}

// IsConverged returns whether the optimizer has converged
func (sgd *SGD) IsConverged() bool {
	return sgd.converged
}

// EstimateLearningRate estimates optimal learning rate using line search
func (sgd *SGD) EstimateLearningRate(parameters []float64, gradients []float64, 
	costFunc func([]float64) float64) (float64, error) {
	
	if len(parameters) != len(gradients) {
		return 0, fmt.Errorf("parameter and gradient dimensions mismatch")
	}
	
	// Try different learning rates
	learningRates := []float64{0.1, 0.01, 0.001, 0.0001}
	bestLR := sgd.learningRate
	bestCost := math.Inf(1)
	
	// Current cost
	currentCost := costFunc(parameters)
	
	for _, lr := range learningRates {
		// Make a test step
		testParams := make([]float64, len(parameters))
		for i := range parameters {
			testParams[i] = parameters[i] - lr*gradients[i]
		}
		
		// Evaluate cost at test point
		testCost := costFunc(testParams)
		
		// Check if this learning rate gives improvement and is stable
		if testCost < currentCost && testCost < bestCost {
			bestCost = testCost
			bestLR = lr
		}
	}
	
	return bestLR, nil
}

// ScheduleLearningRate applies learning rate scheduling
func (sgd *SGD) ScheduleLearningRate(schedule LRScheduleType, epoch int) {
	switch schedule {
	case LRScheduleStep:
		// Step decay: reduce by factor of 10 every 100 epochs
		if epoch > 0 && epoch%100 == 0 {
			sgd.learningRate *= 0.1
		}
		
	case LRScheduleExponential:
		// Exponential decay: lr = lr0 * exp(-decay * epoch)
		decay := 0.01
		sgd.learningRate = sgd.learningRate * math.Exp(-decay*float64(epoch))
		
	case LRScheduleLinear:
		// Linear decay
		if epoch > 0 {
			sgd.learningRate = sgd.learningRate * (1.0 - 0.001*float64(epoch))
			if sgd.learningRate < 1e-6 {
				sgd.learningRate = 1e-6
			}
		}
		
	case LRScheduleCosine:
		// Cosine annealing
		minLR := sgd.learningRate * 0.01
		maxLR := sgd.learningRate
		sgd.learningRate = minLR + 0.5*(maxLR-minLR)*(1+math.Cos(math.Pi*float64(epoch)/100.0))
	}
}

// SGDStats represents statistics about the SGD optimizer
type SGDStats struct {
	LearningRate    float64   `json:"learning_rate"`     // Current learning rate
	Momentum        float64   `json:"momentum"`          // Momentum parameter
	Decay           float64   `json:"decay"`             // Weight decay
	UpdateCount     int       `json:"update_count"`      // Number of updates
	AvgGradNorm     float64   `json:"avg_grad_norm"`     // Average gradient norm
	BestCost        float64   `json:"best_cost"`         // Best cost achieved
	CurrentCost     float64   `json:"current_cost"`      // Current cost
	ConvergenceRate float64   `json:"convergence_rate"`  // Rate of convergence
	Converged       bool      `json:"converged"`         // Whether converged
	IsAdaptive      bool      `json:"is_adaptive"`       // Whether using adaptive LR
	LastUpdate      time.Time `json:"last_update"`       // Last update time
}

// LRScheduleType represents different learning rate schedules
type LRScheduleType int

const (
	LRScheduleConstant LRScheduleType = iota
	LRScheduleStep
	LRScheduleExponential
	LRScheduleLinear
	LRScheduleCosine
)

// String returns string representation of learning rate schedule
func (lrs LRScheduleType) String() string {
	switch lrs {
	case LRScheduleStep:
		return "step"
	case LRScheduleExponential:
		return "exponential"
	case LRScheduleLinear:
		return "linear"
	case LRScheduleCosine:
		return "cosine"
	default:
		return "constant"
	}
}