package learning

import (
	"math"
	"testing"
)

func TestSGD_NewSGD(t *testing.T) {
	learningRate := 0.01
	numParams := 5

	sgd := NewSGD(learningRate, numParams)
	if sgd == nil {
		t.Fatal("NewSGD() returned nil")
	}

	if sgd.learningRate != learningRate {
		t.Errorf("Expected learningRate=%f, got %f", learningRate, sgd.learningRate)
	}

	if len(sgd.parameters) != numParams {
		t.Errorf("Expected parameters length=%d, got %d", numParams, len(sgd.parameters))
	}

	if len(sgd.velocities) != numParams {
		t.Errorf("Expected velocities length=%d, got %d", numParams, len(sgd.velocities))
	}

	if len(sgd.gradientSum) != numParams {
		t.Errorf("Expected gradientSum length=%d, got %d", numParams, len(sgd.gradientSum))
	}

	if sgd.updateCount != 0 {
		t.Errorf("Expected initial updateCount=0, got %d", sgd.updateCount)
	}

	if !math.IsInf(sgd.bestCost, 1) {
		t.Errorf("Expected initial bestCost=Inf, got %f", sgd.bestCost)
	}

	// Test default learning rate
	sgdDefault := NewSGD(0, numParams)
	if sgdDefault.learningRate != 0.001 {
		t.Errorf("Expected default learningRate=0.001, got %f", sgdDefault.learningRate)
	}
}

func TestSGD_NewSGDWithMomentum(t *testing.T) {
	learningRate := 0.01
	momentum := 0.9
	numParams := 3

	sgd := NewSGDWithMomentum(learningRate, momentum, numParams)
	if sgd == nil {
		t.Fatal("NewSGDWithMomentum() returned nil")
	}

	if sgd.momentum != momentum {
		t.Errorf("Expected momentum=%f, got %f", momentum, sgd.momentum)
	}

	if sgd.learningRate != learningRate {
		t.Errorf("Expected learningRate=%f, got %f", learningRate, sgd.learningRate)
	}
}

func TestSGD_NewSGDAdaptive(t *testing.T) {
	learningRate := 0.01
	numParams := 3

	sgd := NewSGDAdaptive(learningRate, numParams)
	if sgd == nil {
		t.Fatal("NewSGDAdaptive() returned nil")
	}

	if !sgd.adaptive {
		t.Error("Expected adaptive=true")
	}
}

func TestSGD_Update(t *testing.T) {
	sgd := NewSGD(0.1, 3)

	params := []float64{1.0, 2.0, 3.0}
	grads := []float64{0.1, -0.2, 0.3}
	cost := 5.0

	err := sgd.Update(params, grads, cost)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Parameters should be updated: param = param - lr * grad
	expectedParams := []float64{
		1.0 - 0.1*0.1, // 0.99
		2.0 - 0.1*(-0.2), // 2.02
		3.0 - 0.1*0.3, // 2.97
	}

	for i, expected := range expectedParams {
		if math.Abs(params[i]-expected) > 0.0001 {
			t.Errorf("Parameter %d: expected %f, got %f", i, expected, params[i])
		}
	}

	if sgd.updateCount != 1 {
		t.Errorf("Expected updateCount=1, got %d", sgd.updateCount)
	}

	if sgd.bestCost != cost {
		t.Errorf("Expected bestCost=%f, got %f", cost, sgd.bestCost)
	}

	if len(sgd.costHistory) != 1 {
		t.Errorf("Expected costHistory length=1, got %d", len(sgd.costHistory))
	}

	if sgd.costHistory[0] != cost {
		t.Errorf("Expected costHistory[0]=%f, got %f", cost, sgd.costHistory[0])
	}
}

func TestSGD_UpdateWithMomentum(t *testing.T) {
	sgd := NewSGDWithMomentum(0.1, 0.9, 2)

	params := []float64{1.0, 2.0}
	grads := []float64{0.1, -0.2}
	cost := 3.0

	// First update
	err := sgd.Update(params, grads, cost)
	if err != nil {
		t.Fatalf("First update failed: %v", err)
	}

	// Store first update results
	firstParams := make([]float64, len(params))
	copy(firstParams, params)

	// Second update with same gradients
	err = sgd.Update(params, grads, cost)
	if err != nil {
		t.Fatalf("Second update failed: %v", err)
	}

	// With momentum, the second update should have larger changes
	// due to accumulated velocity
	for i := range params {
		if math.Abs(params[i]-firstParams[i]) <= 0.0001 {
			t.Errorf("Parameter %d should change more with momentum", i)
		}
	}
}

func TestSGD_UpdateDimensionMismatch(t *testing.T) {
	sgd := NewSGD(0.1, 3)

	params := []float64{1.0, 2.0}      // 2 parameters
	grads := []float64{0.1, -0.2, 0.3} // 3 gradients
	cost := 5.0

	err := sgd.Update(params, grads, cost)
	if err == nil {
		t.Error("Expected error for dimension mismatch")
	}
}

func TestSGD_UpdateAdaptive(t *testing.T) {
	sgd := NewSGDAdaptive(0.1, 2)

	params := []float64{1.0, 2.0}
	grads := []float64{0.1, 0.1}
	cost := 3.0

	// Perform multiple updates to see adaptive behavior
	initialLR := sgd.learningRate

	for i := 0; i < 5; i++ {
		err := sgd.Update(params, grads, cost)
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}
	}

	// The effective learning rate should be different due to adaptive behavior
	// (This is verified through the gradient sum accumulation)
	if sgd.gradientSum[0] == 0.0 {
		t.Error("Expected gradient sum to accumulate in adaptive mode")
	}

	// Learning rate itself shouldn't change, but effective LR is calculated differently
	if sgd.learningRate != initialLR {
		t.Errorf("Base learning rate should not change in adaptive mode")
	}
}

func TestSGD_BatchUpdate(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	paramBatch := [][]float64{
		{1.0, 2.0},
		{1.5, 2.5},
		{2.0, 3.0},
	}
	gradBatch := [][]float64{
		{0.1, -0.1},
		{0.2, -0.2},
		{0.3, -0.3},
	}
	costs := []float64{1.0, 2.0, 3.0}

	err := sgd.BatchUpdate(paramBatch, gradBatch, costs)
	if err != nil {
		t.Fatalf("BatchUpdate() failed: %v", err)
	}

	if sgd.updateCount != 3 {
		t.Errorf("Expected updateCount=3, got %d", sgd.updateCount)
	}

	if len(sgd.costHistory) != 3 {
		t.Errorf("Expected costHistory length=3, got %d", len(sgd.costHistory))
	}
}

func TestSGD_BatchUpdateMismatch(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	paramBatch := [][]float64{{1.0, 2.0}}
	gradBatch := [][]float64{{0.1, -0.1}, {0.2, -0.2}} // Different size
	costs := []float64{1.0}

	err := sgd.BatchUpdate(paramBatch, gradBatch, costs)
	if err == nil {
		t.Error("Expected error for batch size mismatch")
	}
}

func TestSGD_MiniBatchUpdate(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	paramBatch := [][]float64{
		{1.0, 2.0},
		{1.1, 2.1},
		{0.9, 1.9},
	}
	gradBatch := [][]float64{
		{0.1, -0.1},
		{0.2, -0.2},
		{0.0, 0.0},
	}
	costs := []float64{1.0, 2.0, 0.5}

	err := sgd.MiniBatchUpdate(paramBatch, gradBatch, costs)
	if err != nil {
		t.Fatalf("MiniBatchUpdate() failed: %v", err)
	}

	// Should perform only one update (averaged)
	if sgd.updateCount != 1 {
		t.Errorf("Expected updateCount=1, got %d", sgd.updateCount)
	}

	// Cost should be average: (1.0 + 2.0 + 0.5) / 3 = 1.167
	expectedAvgCost := (1.0 + 2.0 + 0.5) / 3.0
	if math.Abs(sgd.costHistory[0]-expectedAvgCost) > 0.0001 {
		t.Errorf("Expected average cost=%f, got %f", expectedAvgCost, sgd.costHistory[0])
	}
}

func TestSGD_MiniBatchUpdateEmpty(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	err := sgd.MiniBatchUpdate([][]float64{}, [][]float64{}, []float64{})
	if err == nil {
		t.Error("Expected error for empty batch")
	}
}

func TestSGD_GetCurrentParameters(t *testing.T) {
	sgd := NewSGD(0.1, 3)

	params := []float64{1.0, 2.0, 3.0}
	grads := []float64{0.1, -0.1, 0.2}

	sgd.Update(params, grads, 5.0)

	currentParams := sgd.GetCurrentParameters()
	if len(currentParams) != len(params) {
		t.Errorf("Expected %d parameters, got %d", len(params), len(currentParams))
	}

	// Should be a copy, not the same slice
	currentParams[0] = 999.0
	if sgd.parameters[0] == 999.0 {
		t.Error("GetCurrentParameters() should return a copy")
	}
}

func TestSGD_GetBestParameters(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	params1 := []float64{1.0, 2.0}
	grads1 := []float64{0.1, -0.1}
	cost1 := 10.0

	params2 := []float64{1.5, 2.5}
	grads2 := []float64{0.2, -0.2}
	cost2 := 5.0 // Better cost

	// First update
	sgd.Update(params1, grads1, cost1)
	
	// Second update with better cost
	sgd.Update(params2, grads2, cost2)

	bestParams := sgd.GetBestParameters()
	
	// Should be the parameters from the better cost update
	for i, expected := range params2 {
		if math.Abs(bestParams[i]-expected) > 0.0001 {
			t.Errorf("Best parameter %d: expected %f, got %f", i, expected, bestParams[i])
		}
	}

	if sgd.bestCost != cost2 {
		t.Errorf("Expected bestCost=%f, got %f", cost2, sgd.bestCost)
	}
}

func TestSGD_GetStats(t *testing.T) {
	sgd := NewSGD(0.05, 2)
	sgd.momentum = 0.9
	sgd.decay = 0.01

	params := []float64{1.0, 2.0}
	grads := []float64{0.1, -0.1}
	cost := 5.0

	sgd.Update(params, grads, cost)

	stats := sgd.GetStats()

	if stats.LearningRate != 0.05 {
		t.Errorf("Expected learningRate=%f, got %f", 0.05, stats.LearningRate)
	}

	if stats.Momentum != 0.9 {
		t.Errorf("Expected momentum=%f, got %f", 0.9, stats.Momentum)
	}

	if stats.Decay != 0.01 {
		t.Errorf("Expected decay=%f, got %f", 0.01, stats.Decay)
	}

	if stats.UpdateCount != 1 {
		t.Errorf("Expected updateCount=1, got %d", stats.UpdateCount)
	}

	if stats.BestCost != cost {
		t.Errorf("Expected bestCost=%f, got %f", cost, stats.BestCost)
	}

	if stats.CurrentCost != cost {
		t.Errorf("Expected currentCost=%f, got %f", cost, stats.CurrentCost)
	}

	if stats.AvgGradNorm <= 0 {
		t.Errorf("Expected positive avgGradNorm, got %f", stats.AvgGradNorm)
	}

	if stats.IsAdaptive != sgd.adaptive {
		t.Errorf("Expected isAdaptive=%t, got %t", sgd.adaptive, stats.IsAdaptive)
	}
}

func TestSGD_SetLearningRate(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	newLR := 0.01
	sgd.SetLearningRate(newLR)

	if sgd.learningRate != newLR {
		t.Errorf("Expected learningRate=%f, got %f", newLR, sgd.learningRate)
	}

	// Test invalid learning rate
	sgd.SetLearningRate(-0.1)
	if sgd.learningRate != newLR {
		t.Error("Learning rate should not change for invalid value")
	}
}

func TestSGD_SetMomentum(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	newMomentum := 0.95
	sgd.SetMomentum(newMomentum)

	if sgd.momentum != newMomentum {
		t.Errorf("Expected momentum=%f, got %f", newMomentum, sgd.momentum)
	}

	// Test invalid momentum
	sgd.SetMomentum(-0.1)
	if sgd.momentum != newMomentum {
		t.Error("Momentum should not change for invalid value")
	}

	sgd.SetMomentum(1.5)
	if sgd.momentum != newMomentum {
		t.Error("Momentum should not change for invalid value")
	}
}

func TestSGD_SetDecay(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	newDecay := 0.001
	sgd.SetDecay(newDecay)

	if sgd.decay != newDecay {
		t.Errorf("Expected decay=%f, got %f", newDecay, sgd.decay)
	}

	// Test invalid decay
	sgd.SetDecay(-0.1)
	if sgd.decay != newDecay {
		t.Error("Decay should not change for invalid value")
	}
}

func TestSGD_Reset(t *testing.T) {
	sgd := NewSGD(0.1, 3)

	// Add some data
	params := []float64{1.0, 2.0, 3.0}
	grads := []float64{0.1, -0.1, 0.2}
	sgd.Update(params, grads, 5.0)

	// Verify data exists
	if sgd.updateCount == 0 {
		t.Error("Expected updateCount > 0 before reset")
	}
	if len(sgd.costHistory) == 0 {
		t.Error("Expected cost history before reset")
	}
	if sgd.bestCost == math.Inf(1) {
		t.Error("Expected bestCost to be set before reset")
	}

	sgd.Reset()

	// Verify reset
	if sgd.updateCount != 0 {
		t.Errorf("Expected updateCount=0 after reset, got %d", sgd.updateCount)
	}

	if len(sgd.costHistory) != 0 {
		t.Errorf("Expected empty cost history after reset, got %d", len(sgd.costHistory))
	}

	if !math.IsInf(sgd.bestCost, 1) {
		t.Errorf("Expected bestCost=Inf after reset, got %f", sgd.bestCost)
	}

	if !sgd.converged {
		t.Error("Expected converged=false after reset")
	}

	for i := range sgd.parameters {
		if sgd.parameters[i] != 0.0 {
			t.Errorf("Parameter %d should be 0 after reset, got %f", i, sgd.parameters[i])
		}
		if sgd.velocities[i] != 0.0 {
			t.Errorf("Velocity %d should be 0 after reset, got %f", i, sgd.velocities[i])
		}
		if sgd.gradientSum[i] != 0.0 {
			t.Errorf("GradientSum %d should be 0 after reset, got %f", i, sgd.gradientSum[i])
		}
	}
}

func TestSGD_GetCostHistory(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	costs := []float64{10.0, 8.0, 6.0, 4.0}
	params := []float64{1.0, 2.0}
	grads := []float64{0.1, -0.1}

	for _, cost := range costs {
		sgd.Update(params, grads, cost)
	}

	history := sgd.GetCostHistory()
	if len(history) != len(costs) {
		t.Errorf("Expected history length=%d, got %d", len(costs), len(history))
	}

	for i, expected := range costs {
		if history[i] != expected {
			t.Errorf("History[%d]: expected %f, got %f", i, expected, history[i])
		}
	}

	// Should be a copy
	history[0] = 999.0
	if sgd.costHistory[0] == 999.0 {
		t.Error("GetCostHistory() should return a copy")
	}
}

func TestSGD_IsConverged(t *testing.T) {
	sgd := NewSGD(0.1, 2)
	sgd.patience = 2
	sgd.tolerance = 1e-3

	if sgd.IsConverged() {
		t.Error("Should not be converged initially")
	}

	params := []float64{1.0, 2.0}
	grads := []float64{0.1, -0.1}

	// Add costs that don't improve (should converge)
	costs := []float64{5.0, 5.001, 5.002, 5.003}
	for _, cost := range costs {
		sgd.Update(params, grads, cost)
	}

	if !sgd.IsConverged() {
		t.Error("Should be converged after no improvement")
	}
}

func TestSGD_EstimateLearningRate(t *testing.T) {
	sgd := NewSGD(0.1, 2)

	params := []float64{1.0, 2.0}
	grads := []float64{0.5, -0.5}

	// Simple quadratic cost function: cost = sum(params^2)
	costFunc := func(p []float64) float64 {
		cost := 0.0
		for _, param := range p {
			cost += param * param
		}
		return cost
	}

	bestLR, err := sgd.EstimateLearningRate(params, grads, costFunc)
	if err != nil {
		t.Fatalf("EstimateLearningRate() failed: %v", err)
	}

	if bestLR <= 0 {
		t.Errorf("Expected positive learning rate, got %f", bestLR)
	}

	// Test dimension mismatch
	wrongGrads := []float64{0.1} // Wrong size
	_, err = sgd.EstimateLearningRate(params, wrongGrads, costFunc)
	if err == nil {
		t.Error("Expected error for dimension mismatch")
	}
}

func TestSGD_ScheduleLearningRate(t *testing.T) {
	sgd := NewSGD(0.1, 2)
	initialLR := sgd.learningRate

	// Test step schedule
	sgd.ScheduleLearningRate(LRScheduleStep, 100)
	if sgd.learningRate >= initialLR {
		t.Error("Step schedule should decrease learning rate at epoch 100")
	}

	// Reset and test exponential schedule
	sgd.learningRate = initialLR
	sgd.ScheduleLearningRate(LRScheduleExponential, 10)
	if sgd.learningRate >= initialLR {
		t.Error("Exponential schedule should decrease learning rate")
	}

	// Reset and test linear schedule
	sgd.learningRate = initialLR
	sgd.ScheduleLearningRate(LRScheduleLinear, 10)
	if sgd.learningRate >= initialLR {
		t.Error("Linear schedule should decrease learning rate")
	}

	// Reset and test cosine schedule
	sgd.learningRate = initialLR
	sgd.ScheduleLearningRate(LRScheduleCosine, 25)
	// Cosine schedule might increase or decrease depending on phase
	if sgd.learningRate < 0 {
		t.Error("Cosine schedule should keep learning rate positive")
	}

	// Test constant schedule (should not change)
	sgd.learningRate = initialLR
	sgd.ScheduleLearningRate(LRScheduleConstant, 50)
	if sgd.learningRate != initialLR {
		t.Error("Constant schedule should not change learning rate")
	}
}

func TestSGD_CalculateGradientNorm(t *testing.T) {
	sgd := NewSGD(0.1, 3)

	grads := []float64{3.0, 4.0, 0.0}
	expectedNorm := 5.0 // sqrt(3^2 + 4^2 + 0^2) = sqrt(25) = 5

	norm := sgd.calculateGradientNorm(grads)
	if math.Abs(norm-expectedNorm) > 0.0001 {
		t.Errorf("Expected gradient norm=%f, got %f", expectedNorm, norm)
	}

	// Test with zero gradients
	zeroGrads := []float64{0.0, 0.0, 0.0}
	zeroNorm := sgd.calculateGradientNorm(zeroGrads)
	if zeroNorm != 0.0 {
		t.Errorf("Expected zero gradient norm=0.0, got %f", zeroNorm)
	}
}

func TestSGD_WeightDecay(t *testing.T) {
	sgd := NewSGD(0.1, 2)
	sgd.decay = 0.01

	params := []float64{1.0, 2.0}
	grads := []float64{0.0, 0.0} // Zero gradients to isolate decay effect
	cost := 1.0

	sgd.Update(params, grads, cost)

	// With weight decay, parameters should move towards zero even with zero gradients
	// Expected update: param = param - lr * (grad + decay * param)
	// = param - lr * decay * param = param * (1 - lr * decay)
	expectedParams := []float64{
		1.0 * (1 - 0.1*0.01), // 0.999
		2.0 * (1 - 0.1*0.01), // 1.998
	}

	for i, expected := range expectedParams {
		if math.Abs(params[i]-expected) > 0.0001 {
			t.Errorf("Parameter %d with decay: expected %f, got %f", i, expected, params[i])
		}
	}
}

func TestLRScheduleType_String(t *testing.T) {
	testCases := []struct {
		schedule LRScheduleType
		expected string
	}{
		{LRScheduleConstant, "constant"},
		{LRScheduleStep, "step"},
		{LRScheduleExponential, "exponential"},
		{LRScheduleLinear, "linear"},
		{LRScheduleCosine, "cosine"},
	}

	for _, tc := range testCases {
		if tc.schedule.String() != tc.expected {
			t.Errorf("Expected %s.String()=%s, got %s",
				tc.schedule, tc.expected, tc.schedule.String())
		}
	}
}

// Benchmark SGD performance
func BenchmarkSGD_Update(b *testing.B) {
	sgd := NewSGD(0.01, 10)
	params := make([]float64, 10)
	grads := make([]float64, 10)
	
	for i := range params {
		params[i] = float64(i)
		grads[i] = 0.1 * float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sgd.Update(params, grads, float64(i))
	}
}

func BenchmarkSGD_UpdateWithMomentum(b *testing.B) {
	sgd := NewSGDWithMomentum(0.01, 0.9, 10)
	params := make([]float64, 10)
	grads := make([]float64, 10)
	
	for i := range params {
		params[i] = float64(i)
		grads[i] = 0.1 * float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sgd.Update(params, grads, float64(i))
	}
}

func BenchmarkSGD_MiniBatchUpdate(b *testing.B) {
	sgd := NewSGD(0.01, 5)
	
	paramBatch := make([][]float64, 32) // Batch size 32
	gradBatch := make([][]float64, 32)
	costs := make([]float64, 32)
	
	for i := range paramBatch {
		paramBatch[i] = make([]float64, 5)
		gradBatch[i] = make([]float64, 5)
		costs[i] = float64(i)
		
		for j := range paramBatch[i] {
			paramBatch[i][j] = float64(j)
			gradBatch[i][j] = 0.1 * float64(j)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sgd.MiniBatchUpdate(paramBatch, gradBatch, costs)
	}
}