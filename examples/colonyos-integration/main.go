package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/colonyos"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

func main() {
	fmt.Println("ColonyOS-CAPE Integration Example")
	fmt.Println("=================================")

	// Create deployment configuration for hybrid cloud scenario
	deploymentConfig := models.NewDefaultDeploymentConfig(models.DeploymentTypeHybrid)
	
	// Customize for ColonyOS environment
	deploymentConfig.OptimizationGoals = []models.OptimizationGoal{
		{Metric: "latency", Weight: 0.3, Minimize: true},
		{Metric: "compute_cost", Weight: 0.2, Minimize: true},
		{Metric: "data_movement", Weight: 0.3, Minimize: true},
		{Metric: "throughput", Weight: 0.2, Minimize: false},
	}
	deploymentConfig.DataGravityFactor = 0.7 // High importance of data locality

	// Print deployment configuration
	fmt.Printf("Deployment Configuration:\n")
	fmt.Printf("  Type: %s\n", deploymentConfig.DeploymentType)
	fmt.Printf("  Data Gravity Factor: %.2f\n", deploymentConfig.DataGravityFactor)
	fmt.Printf("  Optimization Goals:\n")
	for _, goal := range deploymentConfig.OptimizationGoals {
		fmt.Printf("    - %s: weight=%.1f, minimize=%v\n", 
			goal.Metric, goal.Weight, goal.Minimize)
	}

	// Create ColonyOS client (mock for demonstration)
	client := colonyos.NewMockColonyOSClient()

	// Add some mock executors to simulate ColonyOS environment
	setupMockExecutors(client)

	// Create CAPE orchestrator configuration
	orchestratorConfig := colonyos.CAPEOrchestratorConfig{
		ColonyName:         "hybrid-computing-colony",
		ExecutorName:       "cape-orchestrator-001",
		ExecutorType:       "cape-optimizer",
		DeploymentConfig:   deploymentConfig,
		AssignInterval:     2 * time.Second,
		MetricsInterval:    10 * time.Second,
		DecisionTimeout:    5 * time.Second,
		MaxConcurrent:      3,
		SupportedFunctions: []string{"echo", "compute", "ml-inference", "data-process"},
	}

	// Create CAPE orchestrator
	orchestrator := colonyos.NewCAPEOrchestrator(client, orchestratorConfig)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal, stopping orchestrator...")
		cancel()
	}()

	// Add some mock processes to demonstrate the system
	addMockProcesses(client)

	// Start the orchestrator
	fmt.Println("\nStarting CAPE Orchestrator...")
	err := orchestrator.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Run status monitoring
	go monitorStatus(orchestrator, ctx)

	// Simulate submitting additional processes
	go simulateProcessSubmission(client, ctx)

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop orchestrator
	err = orchestrator.Stop()
	if err != nil {
		log.Printf("Error stopping orchestrator: %v", err)
	}

	fmt.Println("CAPE-ColonyOS integration example completed")
}

// setupMockExecutors creates mock executors to simulate ColonyOS environment
func setupMockExecutors(client *colonyos.MockColonyOSClient) {
	// Edge executor for low-latency tasks
	edgeExecutor := models.ColonyOSExecutor{
		ExecutorName: "edge-executor-01",
		ExecutorType: "edge",
		Location: models.ColonyOSLocation{
			Longitude:   59.3293,
			Latitude:    18.0686,
			Description: "Stockholm Edge Node",
		},
		Capabilities: models.ColonyOSCapabilities{
			Hardware: models.ColonyOSHardware{
				Model:   "Intel Xeon E-2124",
				CPU:     "2000m",
				Memory:  "8Gi",
				Storage: "1Ti",
			},
			Software: models.ColonyOSSoftware{
				Name:    "colonyos/edge:latest",
				Type:    "container",
				Version: "v1.2.0",
			},
		},
		Status:   models.ExecutorStatusOnline,
		LastSeen: time.Now(),
		Utilization: models.DetailedUtilization{
			ComputeUsage: 0.2,
			MemoryUsage:  0.3,
			DiskUsage:    0.1,
			NetworkUsage: 0.15,
		},
	}
	client.AddMockExecutor(edgeExecutor)

	// Cloud executor for high-throughput tasks
	cloudExecutor := models.ColonyOSExecutor{
		ExecutorName: "cloud-executor-01",
		ExecutorType: "cloud",
		Location: models.ColonyOSLocation{
			Longitude:   -77.0369,
			Latitude:    38.9072,
			Description: "AWS us-east-1",
		},
		Capabilities: models.ColonyOSCapabilities{
			Hardware: models.ColonyOSHardware{
				Model:   "AWS c5.4xlarge",
				CPU:     "8000m",
				Memory:  "32Gi",
				Storage: "500Gi",
			},
			Software: models.ColonyOSSoftware{
				Name:    "colonyos/cloud:latest",
				Type:    "k8s",
				Version: "v2.1.0",
			},
		},
		Status:   models.ExecutorStatusOnline,
		LastSeen: time.Now(),
		Utilization: models.DetailedUtilization{
			ComputeUsage: 0.4,
			MemoryUsage:  0.5,
			DiskUsage:    0.3,
			NetworkUsage: 0.2,
		},
	}
	client.AddMockExecutor(cloudExecutor)

	// ML executor for AI/ML workloads
	mlExecutor := models.ColonyOSExecutor{
		ExecutorName: "ml-executor-01",
		ExecutorType: "ml",
		Location: models.ColonyOSLocation{
			Longitude:   65.6120,
			Latitude:    22.1322,
			Description: "Iceland GPU Farm",
		},
		Capabilities: models.ColonyOSCapabilities{
			Hardware: models.ColonyOSHardware{
				Model:   "AMD Ryzen 9 5950X",
				CPU:     "4000m",
				Memory:  "64Gi",
				Storage: "2Ti",
				GPU: &models.ColonyOSGPU{
					Name:  "nvidia_rtx_4090",
					Count: 2,
				},
			},
			Software: models.ColonyOSSoftware{
				Name:    "colonyos/ml:latest",
				Type:    "container",
				Version: "v3.0.0",
			},
		},
		Status:   models.ExecutorStatusOnline,
		LastSeen: time.Now(),
		Utilization: models.DetailedUtilization{
			ComputeUsage: 0.6,
			MemoryUsage:  0.7,
			DiskUsage:    0.4,
			NetworkUsage: 0.1,
		},
	}
	client.AddMockExecutor(mlExecutor)

	fmt.Printf("Set up %d mock executors\n", 3)
}

// addMockProcesses adds initial processes to demonstrate the system
func addMockProcesses(client *colonyos.MockColonyOSClient) {
	processes := []models.ColonyOSProcess{
		{
			ProcessID: "proc-echo-001",
			Spec: models.ColonyOSProcessSpec{
				Conditions: models.ColonyOSConditions{
					ExecutorType: "edge",
				},
				FuncName: "echo",
				Args:     []string{"Hello from CAPE!"},
				Label:    "demo-echo",
				Priority: 5,
				ResourceHints: &models.ResourceHints{
					LatencySensitive: true,
					CPUIntensive:     false,
				},
			},
			State:          models.ProcessStateWaiting,
			SubmissionTime: time.Now(),
		},
		{
			ProcessID: "proc-compute-001",
			Spec: models.ColonyOSProcessSpec{
				Conditions: models.ColonyOSConditions{
					ExecutorType: "cloud",
				},
				FuncName: "compute",
				Args:     []string{"matrix", "1000x1000"},
				Label:    "demo-compute",
				Priority: 7,
				ResourceHints: &models.ResourceHints{
					CPUIntensive:  true,
					CostSensitive: true,
				},
			},
			State:          models.ProcessStateWaiting,
			SubmissionTime: time.Now(),
		},
		{
			ProcessID: "proc-ml-001",
			Spec: models.ColonyOSProcessSpec{
				Conditions: models.ColonyOSConditions{
					ExecutorType: "ml",
					RequiredGPU:  true,
				},
				FuncName: "ml-inference",
				Args:     []string{"image-classification", "batch-size-32"},
				Label:    "demo-ml",
				Priority: 8,
				ResourceHints: &models.ResourceHints{
					GPURequired:      true,
					MemoryIntensive:  true,
					LatencySensitive: false,
				},
			},
			State:          models.ProcessStateWaiting,
			SubmissionTime: time.Now(),
		},
	}

	for _, process := range processes {
		client.AddMockProcess(process)
		fmt.Printf("Added mock process: %s (%s)\n", process.ProcessID, process.Spec.FuncName)
	}
}

// simulateProcessSubmission periodically submits new processes
func simulateProcessSubmission(client *colonyos.MockColonyOSClient, ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	processCount := 4
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create a random process spec
			spec := models.ColonyOSProcessSpec{
				Conditions: models.ColonyOSConditions{
					ExecutorType: []string{"edge", "cloud", "ml"}[processCount%3],
				},
				FuncName: []string{"echo", "compute", "data-process"}[processCount%3],
				Args:     []string{fmt.Sprintf("arg-%d", processCount)},
				Label:    fmt.Sprintf("auto-generated-%d", processCount),
				Priority: 5 + (processCount % 3),
			}

			process := models.ColonyOSProcess{
				ProcessID:      fmt.Sprintf("proc-auto-%03d", processCount),
				Spec:           spec,
				State:          models.ProcessStateWaiting,
				SubmissionTime: time.Now(),
			}

			client.AddMockProcess(process)
			fmt.Printf("Submitted new process: %s\n", process.ProcessID)
			processCount++
		}
	}
}

// monitorStatus periodically prints status information
func monitorStatus(orchestrator *colonyos.CAPEOrchestrator, ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := orchestrator.GetStats()
			capeStats := orchestrator.GetCAPEStats()

			fmt.Printf("\n--- CAPE Orchestrator Status ---\n")
			fmt.Printf("Uptime: %v\n", stats.Uptime.Round(time.Second))
			fmt.Printf("Processes: assigned=%d, completed=%d, failed=%d\n", 
				stats.ProcessesAssigned, stats.ProcessesCompleted, stats.ProcessesFailed)
			fmt.Printf("CAPE Decisions: %d (avg time: %v)\n", 
				stats.CapeDecisions, stats.AvgDecisionTime.Round(time.Millisecond))
			
			if stats.ProcessesCompleted > 0 || stats.ProcessesFailed > 0 {
				successRate := float64(stats.ProcessesCompleted) / float64(stats.ProcessesCompleted + stats.ProcessesFailed) * 100
				fmt.Printf("Success Rate: %.1f%%\n", successRate)
			}
			
			fmt.Printf("Current CAPE Strategy: %s\n", capeStats.CurrentStrategy)
			fmt.Printf("Active Thompson Sampler Strategies: %d\n", len(capeStats.ThompsonStats))
			
			// Print detailed CAPE stats
			if len(capeStats.ThompsonStats) > 0 {
				fmt.Printf("Strategy Performance:\n")
				for strategy, performance := range capeStats.ThompsonStats {
					fmt.Printf("  %s: success=%.1f%%, trials=%d\n", 
						strategy, performance.SuccessRate*100, performance.TotalTrials)
				}
			}
			fmt.Printf("------------------------------\n")
		}
	}
}

// Example showing how to create a process specification programmatically
func createSampleProcessSpec() models.ColonyOSProcessSpec {
	return models.ColonyOSProcessSpec{
		Conditions: models.ColonyOSConditions{
			ExecutorType:    "ml",
			ExecutorNames:   []string{"ml-executor-01", "ml-executor-02"},
			MinCPU:          "2000m",
			MinMemory:       "8Gi",
			RequiredGPU:     true,
			LocationHints:   []string{"iceland", "norway"},
			SecurityLevel:   3,
		},
		FuncName: "ml-training",
		Args:     []string{"model=resnet50", "dataset=imagenet", "epochs=10"},
		Kwargs: map[string]interface{}{
			"learning_rate": 0.001,
			"batch_size":    32,
			"optimizer":     "adam",
		},
		Env: map[string]string{
			"CUDA_VISIBLE_DEVICES": "0,1",
			"OMP_NUM_THREADS":      "8",
		},
		Label:       "ml-training-job",
		MaxWaitTime: 300,  // 5 minutes
		MaxExecTime: 3600, // 1 hour
		MaxRetries:  2,
		Priority:    9,
		EstimatedDuration: 30 * time.Minute,
		DataRequirements: &models.DataRequirements{
			InputDataLocation:  models.DataLocationCloud,
			OutputDataLocation: models.DataLocationCloud,
			DataSizeGB:         50.0,
			DataSensitive:      false,
		},
		ResourceHints: &models.ResourceHints{
			PreferredExecutorType: "ml",
			CPUIntensive:          true,
			MemoryIntensive:       true,
			GPURequired:           true,
			NetworkIntensive:      false,
			LatencySensitive:      false,
			CostSensitive:         true,
		},
	}
}

// Example showing JSON serialization
func demonstrateJSONSerialization() {
	spec := createSampleProcessSpec()
	
	// Convert to JSON
	jsonData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}
	
	fmt.Println("Sample ColonyOS Process Specification (JSON):")
	fmt.Println(string(jsonData))
	
	// Convert back from JSON
	var parsedSpec models.ColonyOSProcessSpec
	err = json.Unmarshal(jsonData, &parsedSpec)
	if err != nil {
		log.Printf("Failed to unmarshal JSON: %v", err)
		return
	}
	
	fmt.Printf("Successfully parsed specification for function: %s\n", parsedSpec.FuncName)
}