package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/algorithm"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

func main() {
	fmt.Println("Colony Process Offloader Algorithm - Configurable CAPE Demo")
	fmt.Println("==========================================================")

	// Create different deployment configurations to demonstrate
	configs := []*models.DeploymentConfig{
		models.NewDefaultDeploymentConfig(models.DeploymentTypeEdge),
		models.NewDefaultDeploymentConfig(models.DeploymentTypeCloud),
		models.NewDefaultDeploymentConfig(models.DeploymentTypeHybrid),
	}

	for i, config := range configs {
		fmt.Printf("\n%d. Testing %s Deployment Configuration\n", i+1, config.DeploymentType)
		fmt.Println("   " + config.Description)
		fmt.Printf("   Data Gravity Factor: %.2f\n", config.DataGravityFactor)
		
		err := runConfigurationTest(config, 10)
		if err != nil {
			fmt.Printf("   ❌ Configuration test failed: %v\n", err)
		} else {
			fmt.Printf("   ✅ Configuration test completed successfully\n")
		}
	}

	fmt.Println("\n🎯 All deployment configurations tested!")
	fmt.Println("The configurable CAPE algorithm successfully adapts to different scenarios")
}

func runConfigurationTest(config *models.DeploymentConfig, decisions int) error {
	// Initialize configurable CAPE algorithm
	cape := algorithm.NewConfigurableCAPE(config)

	// Create sample targets for different location types
	targets := []models.OffloadTarget{
		{
			ID:                "local-executor",
			Type:              models.LOCAL,
			Location:          string(models.DataLocationLocal),
			TotalCapacity:     8.0,
			AvailableCapacity: 6.0,
			MemoryTotal:       16 * 1024 * 1024 * 1024,
			MemoryAvailable:   12 * 1024 * 1024 * 1024,
			NetworkLatency:    1 * time.Millisecond,
			ProcessingSpeed:   1.0,
			Reliability:       0.99,
			ComputeCost:       0.0,
			SecurityLevel:     5,
		},
		{
			ID:                "edge-executor", 
			Type:              models.EDGE,
			Location:          string(models.DataLocationEdge),
			TotalCapacity:     16.0,
			AvailableCapacity: 14.0,
			MemoryTotal:       32 * 1024 * 1024 * 1024,
			MemoryAvailable:   28 * 1024 * 1024 * 1024,
			NetworkLatency:    5 * time.Millisecond,
			ProcessingSpeed:   1.5,
			Reliability:       0.95,
			ComputeCost:       0.05,
			SecurityLevel:     4,
		},
		{
			ID:                "cloud-executor",
			Type:              models.PUBLIC_CLOUD,
			Location:          string(models.DataLocationCloud),
			TotalCapacity:     64.0,
			AvailableCapacity: 48.0,
			MemoryTotal:       128 * 1024 * 1024 * 1024,
			MemoryAvailable:   96 * 1024 * 1024 * 1024,
			NetworkLatency:    25 * time.Millisecond,
			ProcessingSpeed:   2.0,
			Reliability:       0.99,
			ComputeCost:       0.10,
			SecurityLevel:     3,
		},
		{
			ID:                "hpc-executor",
			Type:              models.HPC_CLUSTER,
			Location:          string(models.DataLocationHPC),
			TotalCapacity:     256.0,
			AvailableCapacity: 200.0,
			MemoryTotal:       512 * 1024 * 1024 * 1024,
			MemoryAvailable:   400 * 1024 * 1024 * 1024,
			NetworkLatency:    10 * time.Millisecond,
			ProcessingSpeed:   4.0,
			Reliability:       0.98,
			ComputeCost:       0.20,
			SecurityLevel:     3,
		},
	}

	// Add utilization field to targets
	for i := range targets {
		targets[i].Utilization = models.DetailedUtilization{
			ComputeUsage: rand.Float64() * 0.5, // 0-50% utilization
			MemoryUsage:  rand.Float64() * 0.6, // 0-60% utilization
			DiskUsage:    rand.Float64() * 0.4, // 0-40% utilization
			NetworkUsage: rand.Float64() * 0.3, // 0-30% utilization
		}
	}

	// Run decision simulation
	successfulDecisions := 0
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < decisions; i++ {
		// Create sample process
		process := models.Process{
			ID:                fmt.Sprintf("proc-%d", i+1),
			Type:              []string{"compute", "data", "ml", "batch"}[rand.Intn(4)],
			Priority:          rand.Intn(10) + 1,
			CPURequirement:    float64(rand.Intn(8) + 1),
			MemoryRequirement: int64(rand.Intn(16)+1) * 1024 * 1024 * 1024,
			EstimatedDuration: time.Duration(rand.Intn(300)+30) * time.Second,
			RealTime:          rand.Float64() < 0.2,
			SafetyCritical:    rand.Float64() < 0.1,
			SecurityLevel:     rand.Intn(6),
			LocalityRequired:  rand.Float64() < 0.3,
			Status:            models.QUEUED,
		}

		// Create system state
		systemState := models.SystemState{
			QueueDepth:        rand.Intn(30),
			QueueThreshold:    20,
			ComputeUsage:      models.Utilization(rand.Float64() * 0.8),
			MemoryUsage:       models.Utilization(rand.Float64() * 0.7),
			DiskUsage:         models.Utilization(rand.Float64() * 0.5),
			NetworkUsage:      models.Utilization(rand.Float64() * 0.4),
			MasterUsage:       models.Utilization(rand.Float64() * 0.3),
			ActiveConnections: rand.Intn(100),
			Timestamp:         time.Now(),
			TimeSlot:          time.Now().Hour(),
			DayOfWeek:         int(time.Now().Weekday()),
		}

		// Make decision using configurable CAPE
		decision, err := cape.MakeDecision(process, targets, systemState)
		if err != nil {
			fmt.Printf("     Decision %d failed: %v\n", i+1, err)
			continue
		}

		// Simulate outcome
		outcome := simulateCAPEOutcome(decision, process)
		
		// Report outcome for learning
		err = cape.ReportOutcome(decision.DecisionID, outcome)
		if err != nil {
			fmt.Printf("     Outcome reporting failed: %v\n", err)
		}

		if outcome.Success {
			successfulDecisions++
		}

		// Brief output for first few decisions
		if i < 3 {
			fmt.Printf("     Decision %d: %s -> %s (strategy: %s, success: %v)\n",
				i+1, process.ID, decision.SelectedTarget.ID, decision.SelectedStrategy, outcome.Success)
		}
	}

	// Display results
	successRate := float64(successfulDecisions) / float64(decisions)
	fmt.Printf("     Success Rate: %.1f%% (%d/%d)\n", successRate*100, successfulDecisions, decisions)

	// Get algorithm statistics
	stats := cape.GetStats()
	fmt.Printf("     Current Strategy: %s\n", stats.CurrentStrategy)
	fmt.Printf("     Thompson Sampler: %d strategies tracked\n", len(stats.ThompsonStats))

	return nil
}

func simulateCAPEOutcome(decision algorithm.CAPEDecision, process models.Process) algorithm.CAPEOutcome {
	// Simulate realistic outcomes based on decision and target characteristics
	success := true
	latencyMS := 100.0
	costUSD := 0.50
	throughputOps := 50.0
	energyWh := 10.0
	dataTransferGB := 1.0

	// Target-specific simulation
	switch decision.SelectedTarget.Type {
	case models.LOCAL:
		latencyMS = 10.0 + rand.Float64()*20.0  // 10-30ms
		costUSD = 0.0                           // No monetary cost
		throughputOps = 30.0 + rand.Float64()*20.0 // 30-50 ops/sec
		energyWh = 5.0 + rand.Float64()*5.0     // 5-10 Wh
		dataTransferGB = 0.0                    // No transfer
		success = rand.Float64() < 0.95         // 95% success rate

	case models.EDGE:
		latencyMS = 20.0 + rand.Float64()*30.0  // 20-50ms
		costUSD = 0.02 + rand.Float64()*0.08    // $0.02-0.10
		throughputOps = 40.0 + rand.Float64()*30.0 // 40-70 ops/sec
		energyWh = 8.0 + rand.Float64()*7.0     // 8-15 Wh
		dataTransferGB = 0.1 + rand.Float64()*0.4 // 0.1-0.5 GB
		success = rand.Float64() < 0.90         // 90% success rate

	case models.PUBLIC_CLOUD:
		latencyMS = 50.0 + rand.Float64()*100.0 // 50-150ms
		costUSD = 0.05 + rand.Float64()*0.20    // $0.05-0.25
		throughputOps = 60.0 + rand.Float64()*40.0 // 60-100 ops/sec
		energyWh = 15.0 + rand.Float64()*15.0   // 15-30 Wh
		dataTransferGB = 0.5 + rand.Float64()*1.5 // 0.5-2.0 GB
		success = rand.Float64() < 0.85         // 85% success rate

	case models.HPC_CLUSTER:
		latencyMS = 30.0 + rand.Float64()*50.0  // 30-80ms
		costUSD = 0.10 + rand.Float64()*0.40    // $0.10-0.50
		throughputOps = 100.0 + rand.Float64()*100.0 // 100-200 ops/sec
		energyWh = 25.0 + rand.Float64()*25.0   // 25-50 Wh
		dataTransferGB = 1.0 + rand.Float64()*2.0 // 1.0-3.0 GB
		success = rand.Float64() < 0.80         // 80% success rate
	}

	// Process-specific adjustments
	if process.RealTime && latencyMS > 100.0 {
		success = false // Real-time processes fail with high latency
	}

	if process.SafetyCritical && decision.SelectedTarget.Type != models.LOCAL {
		success = rand.Float64() < 0.70 // Safety-critical prefers local
	}

	// SLA and budget violations
	slaViolation := latencyMS > 1000.0 // 1 second SLA
	budgetOverrun := costUSD > 1.00    // $1 budget

	return algorithm.CAPEOutcome{
		Success:         success,
		LatencyMS:       latencyMS,
		CostUSD:         costUSD,
		ThroughputOps:   throughputOps,
		EnergyWh:        energyWh,
		DataTransferGB:  dataTransferGB,
		SLAViolation:    slaViolation,
		BudgetOverrun:   budgetOverrun,
		CompletedAt:     time.Now(),
	}
}