package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/algorithm"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/learning"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/policy"
)

func main() {
	fmt.Println("Colony Process Offloader Algorithm - Demo")
	fmt.Println("=========================================")

	// Create algorithm configuration
	config := algorithm.Config{
		InitialWeights: decision.AdaptiveWeights{
			QueueDepth:    0.2,
			ProcessorLoad: 0.2,
			NetworkCost:   0.2,
			LatencyCost:   0.2,
			EnergyCost:    0.1,
			PolicyCost:    0.1,
		},
		LearningConfig: learning.LearningConfig{
			WindowSize:      100,
			LearningRate:    0.01,
			ExplorationRate: 0.1,
			MinSamples:      10,
		},
		SafetyConstraints: policy.SafetyConstraints{
			MinLocalCompute:       0.2,
			MinLocalMemory:        0.2,
			MaxConcurrentOffloads: 10,
			DataSovereignty:       true,
			SecurityClearance:     true,
			MaxLatencyTolerance:   500 * time.Millisecond,
			MinReliability:        0.5,
		},
		PerformanceTargets: algorithm.PerformanceTargets{
			MaxDecisionLatency:  500 * time.Millisecond,
			MinDecisionAccuracy: 0.85,
			MaxPolicyViolations: 5,
			MinPerformanceGain:  0.1,
			ConvergenceTimeout:  200,
		},
	}

	// Initialize the algorithm
	alg, err := algorithm.NewAlgorithm(config)
	if err != nil {
		fmt.Printf("Failed to initialize algorithm: %v\n", err)
		return
	}

	fmt.Println("✓ Algorithm initialized successfully")
	fmt.Printf("✓ Health status: %v\n", alg.IsHealthy())

	// Create sample system state
	systemState := models.SystemState{
		QueueDepth:        25,
		QueueThreshold:    20,
		ComputeUsage:      0.75,
		MemoryUsage:       0.60,
		DiskUsage:         0.40,
		NetworkUsage:      0.30,
		MasterUsage:       0.25,
		ActiveConnections: 50,
		Timestamp:         time.Now(),
		TimeSlot:          time.Now().Hour(),
		DayOfWeek:         int(time.Now().Weekday()),
	}

	// Create sample targets
	targets := []models.OffloadTarget{
		{
			ID:                "local-1",
			Type:              models.LOCAL,
			TotalCapacity:     8.0,
			AvailableCapacity: 4.0,
			MemoryTotal:       16 * 1024 * 1024 * 1024,
			MemoryAvailable:   8 * 1024 * 1024 * 1024,
			NetworkLatency:    1 * time.Millisecond,
			NetworkBandwidth:  1000 * 1024 * 1024,
			NetworkStability:  1.0,
			ProcessingSpeed:   1.0,
			Reliability:       0.99,
			ComputeCost:       0.0,
			SecurityLevel:     5,
			DataJurisdiction:  "domestic",
			LastSeen:          time.Now(),
		},
		{
			ID:                "edge-1",
			Type:              models.EDGE,
			TotalCapacity:     16.0,
			AvailableCapacity: 12.0,
			MemoryTotal:       32 * 1024 * 1024 * 1024,
			MemoryAvailable:   24 * 1024 * 1024 * 1024,
			NetworkLatency:    5 * time.Millisecond,
			NetworkBandwidth:  500 * 1024 * 1024,
			NetworkStability:  0.98,
			ProcessingSpeed:   1.5,
			Reliability:       0.95,
			ComputeCost:       0.05,
			SecurityLevel:     4,
			DataJurisdiction:  "domestic",
			LastSeen:          time.Now(),
		},
		{
			ID:                "cloud-1",
			Type:              models.PUBLIC_CLOUD,
			TotalCapacity:     64.0,
			AvailableCapacity: 48.0,
			MemoryTotal:       128 * 1024 * 1024 * 1024,
			MemoryAvailable:   96 * 1024 * 1024 * 1024,
			NetworkLatency:    25 * time.Millisecond,
			NetworkBandwidth:  200 * 1024 * 1024,
			NetworkStability:  0.99,
			ProcessingSpeed:   2.0,
			Reliability:       0.99,
			ComputeCost:       0.10,
			SecurityLevel:     3,
			DataJurisdiction:  "international",
			LastSeen:          time.Now(),
		},
	}

	fmt.Printf("✓ Created %d offload targets\n", len(targets))

	// Simulate decision-making and learning over time
	rand.Seed(time.Now().UnixNano())
	
	fmt.Println("\nRunning decision simulation...")
	fmt.Println("==============================")

	for i := 0; i < 20; i++ {
		// Create a sample process
		process := models.Process{
			ID:                fmt.Sprintf("process-%d", i+1),
			Type:              []string{"compute", "data", "ml", "batch"}[rand.Intn(4)],
			Priority:          rand.Intn(10) + 1,
			CPURequirement:    float64(rand.Intn(8) + 1),
			MemoryRequirement: int64(rand.Intn(16)+1) * 1024 * 1024 * 1024,
			InputSize:         int64(rand.Intn(100)+1) * 1024 * 1024,
			OutputSize:        int64(rand.Intn(50)+1) * 1024 * 1024,
			EstimatedDuration: time.Duration(rand.Intn(300)+30) * time.Second,
			RealTime:          rand.Float64() < 0.2, // 20% real-time
			SafetyCritical:    rand.Float64() < 0.1, // 10% safety-critical
			SecurityLevel:     rand.Intn(6),
			DataSensitivity:   rand.Intn(6),
			LocalityRequired:  rand.Float64() < 0.3, // 30% require locality
			Status:            models.QUEUED,
		}

		// Make decision
		decision, err := alg.MakeOffloadDecision(process, targets, systemState)
		if err != nil {
			fmt.Printf("Error making decision for %s: %v\n", process.ID, err)
			continue
		}

		// Display decision
		action := "KEEP LOCAL"
		targetID := "local"
		if decision.ShouldOffload && decision.Target != nil {
			action = "OFFLOAD"
			targetID = decision.Target.ID
		}

		fmt.Printf("Process %s: %s -> %s (score: %.3f, confidence: %.3f)\n",
			process.ID, action, targetID, decision.Score, decision.Confidence)

		// Simulate outcome
		outcome := simulateOutcome(decision, process)
		
		// Process outcome for learning
		err = alg.ProcessOutcome(outcome)
		if err != nil {
			fmt.Printf("Error processing outcome: %v\n", err)
		}

		// Update system state slightly for next iteration
		systemState.QueueDepth = maxInt(0, systemState.QueueDepth + rand.Intn(5) - 2)
		systemState.ComputeUsage = models.Utilization(min(1.0, maxFloat(0.0, float64(systemState.ComputeUsage) + (rand.Float64()-0.5)*0.1)))
	}

	// Display final performance metrics
	fmt.Println("\nFinal Performance Metrics:")
	fmt.Println("==========================")
	
	metrics := alg.GetPerformanceMetrics()
	fmt.Printf("Decision Count: %d\n", metrics.DecisionCount)
	fmt.Printf("Performance Gain: %.2f%%\n", metrics.PerformanceGain*100)
	fmt.Printf("Convergence Status: %v\n", metrics.IsConverged)
	fmt.Printf("Discovered Patterns: %d\n", metrics.DiscoveredPatterns)
	fmt.Printf("Validated Patterns: %d\n", metrics.ValidatedPatterns)
	
	fmt.Printf("\nCurrent Adaptive Weights:\n")
	fmt.Printf("  Queue Depth: %.3f\n", metrics.CurrentWeights.QueueDepth)
	fmt.Printf("  Processor Load: %.3f\n", metrics.CurrentWeights.ProcessorLoad)
	fmt.Printf("  Network Cost: %.3f\n", metrics.CurrentWeights.NetworkCost)
	fmt.Printf("  Latency Cost: %.3f\n", metrics.CurrentWeights.LatencyCost)
	fmt.Printf("  Energy Cost: %.3f\n", metrics.CurrentWeights.EnergyCost)
	fmt.Printf("  Policy Cost: %.3f\n", metrics.CurrentWeights.PolicyCost)
	
	fmt.Printf("\nPolicy Enforcement Stats:\n")
	fmt.Printf("  Total Evaluations: %d\n", metrics.PolicyStats.TotalEvaluations)
	fmt.Printf("  Hard Violations: %d\n", metrics.PolicyStats.HardViolations)
	fmt.Printf("  Soft Violations: %d\n", metrics.PolicyStats.SoftViolations)
	fmt.Printf("  Blocked Decisions: %d\n", metrics.PolicyStats.BlockedDecisions)
	fmt.Printf("  Allowed Decisions: %d\n", metrics.PolicyStats.AllowedDecisions)

	fmt.Printf("\n✓ Algorithm health: %v\n", alg.IsHealthy())
	fmt.Println("\nDemo completed successfully!")
}

func simulateOutcome(dec decision.OffloadDecision, process models.Process) decision.OffloadOutcome {
	// Simulate realistic outcome based on decision
	success := true
	completedOnTime := true
	reward := 0.5

	if dec.ShouldOffload {
		// Offload outcomes vary based on target and process characteristics
		if process.RealTime && dec.Target != nil && dec.Target.NetworkLatency > 50*time.Millisecond {
			// Real-time process with high latency - likely to have issues
			success = rand.Float64() < 0.7
			completedOnTime = rand.Float64() < 0.6
			reward = -0.5
		} else {
			// Normal offload
			success = rand.Float64() < 0.9
			completedOnTime = rand.Float64() < 0.85
			reward = 1.0
		}
	} else {
		// Local execution is usually reliable but may be slower under high load
		success = rand.Float64() < 0.95
		completedOnTime = rand.Float64() < 0.8
		reward = 0.3
	}

	targetID := "local"
	if dec.ShouldOffload && dec.Target != nil {
		targetID = dec.Target.ID
	}

	return decision.OffloadOutcome{
		DecisionID:      fmt.Sprintf("decision-%s", process.ID),
		ProcessID:       process.ID,
		TargetID:        targetID,
		Success:         success,
		CompletedOnTime: completedOnTime,
		Reward:          reward,
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(process.EstimatedDuration),
		MeasurementTime: time.Now(),
		Attribution: map[string]float64{
			"QueueDepth":    rand.Float64() * 0.3,
			"ProcessorLoad": rand.Float64() * 0.3,
			"NetworkCost":   rand.Float64() * 0.2,
			"LatencyCost":   rand.Float64() * 0.2,
		},
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}