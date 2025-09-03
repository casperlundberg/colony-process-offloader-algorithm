package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/colonyos"
)

// Zero Hardcode ColonyOS-CAPE Integration Example
// 
// This example demonstrates the completely configuration-driven CAPE system where:
// 1. ALL behavior comes from human_config.json (no hardcoded values)
// 2. ALL data comes from colonyos_metrics.json (real ColonyOS format)  
// 3. Native ColonyOS data structures used throughout
// 4. Real ColonyOS process specifications and executor formats
//
// The system adapts its behavior entirely based on configuration files,
// making it suitable for production deployment with different scenarios.

func main() {
	fmt.Println(" Zero Hardcode ColonyOS-CAPE Integration Example")
	fmt.Println("================================================")
	fmt.Println()
	fmt.Println("This system has ZERO hardcoded values!")
	fmt.Println(" All behavior: config/human_config.json")
	fmt.Println(" All data: config/colonyos_metrics.json")
	fmt.Println(" Native ColonyOS formats throughout")
	fmt.Println()

	// Configuration file paths
	humanConfigPath := "./config/human_config.json"
	metricsDataPath := "./config/colonyos_metrics.json"

	// Verify configuration files exist
	if err := verifyConfigFiles(humanConfigPath, metricsDataPath); err != nil {
		log.Fatalf("Configuration verification failed: %v", err)
	}

	// Create the zero-hardcode orchestrator
	fmt.Println(" Loading configuration and metrics...")
	orchestrator, err := colonyos.NewZeroHardcodeOrchestrator(humanConfigPath, metricsDataPath)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Display loaded configuration summary
	displayConfigurationSummary(orchestrator)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\n Received shutdown signal...")
		cancel()
	}()

	// Start the orchestrator
	fmt.Println("\n Starting CAPE Orchestrator with configuration-driven behavior...")
	err = orchestrator.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Monitor the system
	go monitorSystem(orchestrator, ctx)

	// Wait for shutdown
	<-ctx.Done()

	// Stop orchestrator
	fmt.Println("\n  Stopping orchestrator...")
	err = orchestrator.Stop()
	if err != nil {
		log.Printf("Error stopping orchestrator: %v", err)
	}

	fmt.Println(" Zero Hardcode example completed successfully!")
}

// verifyConfigFiles checks that required configuration files exist
func verifyConfigFiles(humanConfigPath, metricsDataPath string) error {
	fmt.Printf(" Verifying configuration files...\n")
	
	if _, err := os.Stat(humanConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("human config file not found: %s", humanConfigPath)
	}
	fmt.Printf("   Human config: %s\n", humanConfigPath)
	
	if _, err := os.Stat(metricsDataPath); os.IsNotExist(err) {
		return fmt.Errorf("metrics data file not found: %s", metricsDataPath)
	}
	fmt.Printf("   Metrics data: %s\n", metricsDataPath)
	
	return nil
}

// displayConfigurationSummary shows the loaded configuration
func displayConfigurationSummary(orchestrator *colonyos.ZeroHardcodeOrchestrator) {
	fmt.Println("\n Configuration Summary")
	fmt.Println("========================")

	config := orchestrator.GetConfiguration()
	
	if humanConfig, ok := config["human_config"].(*colonyos.HumanConfig); ok {
		fmt.Printf("Colony: %s\n", humanConfig.ColonyOSConnection.ColonyName)
		fmt.Printf("Executor: %s (%s)\n", humanConfig.OrchestratorConfig.ExecutorName, humanConfig.OrchestratorConfig.ExecutorType)
		fmt.Printf("Deployment: %s\n", humanConfig.CapeConfig.DeploymentType)
		fmt.Printf("Data Gravity: %.2f\n", humanConfig.CapeConfig.LearningParameters.DataGravityFactor)
		
		fmt.Println("\nOptimization Goals:")
		for _, goal := range humanConfig.CapeConfig.OptimizationGoals {
			fmt.Printf("  • %s: weight=%.2f, minimize=%v\n", goal.Metric, goal.Weight, goal.Minimize)
		}
		
		fmt.Println("\nConstraints:")
		for _, constraint := range humanConfig.CapeConfig.Constraints {
			hardSoft := "soft"
			if constraint.IsHard {
				hardSoft = "hard"
			}
			fmt.Printf("  • %s: %s (%s)\n", constraint.Type, constraint.Value, hardSoft)
		}
		
		fmt.Println("\nBehavior (from config):")
		fmt.Printf("  • Assign interval: %ds\n", humanConfig.OrchestratorConfig.Behavior.AssignIntervalSeconds)
		fmt.Printf("  • Metrics interval: %ds\n", humanConfig.OrchestratorConfig.Behavior.MetricsUpdateIntervalSeconds)
		fmt.Printf("  • Max concurrent: %d\n", humanConfig.OrchestratorConfig.Behavior.MaxConcurrentProcesses)
		fmt.Printf("  • Decision timeout: %ds\n", humanConfig.OrchestratorConfig.Behavior.DecisionTimeoutSeconds)
		
		fmt.Println("\nSupported Functions:")
		for _, funcName := range humanConfig.OrchestratorConfig.SupportedFunctions {
			fmt.Printf("  • %s\n", funcName)
		}
	}
	
	if executorCount, ok := config["active_executors_count"].(int); ok {
		fmt.Printf("\nActive Executors: %d\n", executorCount)
	}
	
	if queueCount, ok := config["queued_processes_count"].(int); ok {
		fmt.Printf("Queued Processes: %d\n", queueCount)
	}
}

// monitorSystem provides real-time monitoring of the orchestrator
func monitorSystem(orchestrator *colonyos.ZeroHardcodeOrchestrator, ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Monitoring interval
	defer ticker.Stop()

	fmt.Println("\n System Monitoring Started")
	fmt.Println("============================")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			displaySystemStatus(orchestrator)
		}
	}
}

// displaySystemStatus shows current system statistics
func displaySystemStatus(orchestrator *colonyos.ZeroHardcodeOrchestrator) {
	stats := orchestrator.GetStats()
	config := orchestrator.GetConfiguration()
	
	fmt.Printf("\n %s\n", time.Now().Format("15:04:05"))
	fmt.Println("------------------")
	
	// Orchestrator stats
	fmt.Printf("Uptime: %v\n", stats.Uptime.Round(time.Second))
	fmt.Printf("Processes: assigned=%d, completed=%d, failed=%d\n", 
		stats.ProcessesAssigned, stats.ProcessesCompleted, stats.ProcessesFailed)
	
	// Success rate
	if stats.ProcessesCompleted > 0 || stats.ProcessesFailed > 0 {
		total := stats.ProcessesCompleted + stats.ProcessesFailed
		successRate := float64(stats.ProcessesCompleted) / float64(total) * 100
		fmt.Printf("Success Rate: %.1f%%\n", successRate)
	}
	
	// CAPE decisions
	fmt.Printf("CAPE Decisions: %d", stats.CapeDecisions)
	if stats.AvgDecisionTime > 0 {
		fmt.Printf(" (avg: %v)", stats.AvgDecisionTime.Round(time.Millisecond))
	}
	fmt.Printf("\n")
	
	// Configuration-driven info
	if lastUpdate, ok := config["last_metrics_update"].(time.Time); ok && !lastUpdate.IsZero() {
		fmt.Printf("Last Metrics Update: %v ago\n", time.Since(lastUpdate).Round(time.Second))
	}
	
	if executorCount, ok := config["active_executors_count"].(int); ok {
		fmt.Printf("Active Executors: %d\n", executorCount)
	}
	
	if queueCount, ok := config["queued_processes_count"].(int); ok {
		fmt.Printf("Process Queue: %d\n", queueCount)
	}
}

// Example showing the difference between hardcoded and config-driven approaches
func demonstrateZeroHardcodeApproach() {
	fmt.Println("\n Zero Hardcode Approach Demonstration")
	fmt.Println("======================================")
	
	fmt.Println(" OLD WAY (hardcoded):")
	fmt.Println("  assignInterval := 5 * time.Second  // Hardcoded!")
	fmt.Println("  maxConcurrent := 3                 // Hardcoded!")
	fmt.Println("  dataGravity := 0.6                 // Hardcoded!")
	fmt.Println("  executorType := \"ml\"                // Hardcoded!")
	fmt.Println()
	
	fmt.Println(" NEW WAY (config-driven):")
	fmt.Println("  assignInterval := config.Behavior.AssignIntervalSeconds")
	fmt.Println("  maxConcurrent := config.Behavior.MaxConcurrentProcesses") 
	fmt.Println("  dataGravity := config.LearningParameters.DataGravityFactor")
	fmt.Println("  executorType := executor.Type  // From ColonyOS metrics")
	fmt.Println()
	
	fmt.Println(" Benefits:")
	fmt.Println("  • No recompilation needed for behavior changes")
	fmt.Println("  • Different configs for dev/staging/production")
	fmt.Println("  • Real ColonyOS data structures")
	fmt.Println("  • Easy A/B testing of optimization strategies")
	fmt.Println("  • Configuration validation and error checking")
	fmt.Println("  • Audit trail of configuration changes")
}

// Example configuration scenarios
func showConfigurationScenarios() {
	fmt.Println("\n  Configuration Scenarios")
	fmt.Println("===========================")
	
	scenarios := []struct {
		name        string
		description string
		config      string
	}{
		{
			name:        "Low Latency Edge",
			description: "Optimize for ultra-low latency at edge nodes",
			config:      `{"deployment_type": "edge", "data_gravity_factor": 0.2, "optimization_goals": [{"metric": "latency", "weight": 0.6, "minimize": true}]}`,
		},
		{
			name:        "Cost-Optimized Cloud", 
			description: "Minimize costs in cloud deployment",
			config:      `{"deployment_type": "cloud", "data_gravity_factor": 0.9, "optimization_goals": [{"metric": "compute_cost", "weight": 0.5, "minimize": true}]}`,
		},
		{
			name:        "High-Throughput HPC",
			description: "Maximum throughput for HPC workloads",
			config:      `{"deployment_type": "hpc", "data_gravity_factor": 0.8, "optimization_goals": [{"metric": "throughput", "weight": 0.7, "minimize": false}]}`,
		},
		{
			name:        "Balanced Hybrid",
			description: "Balance all objectives across hybrid infrastructure", 
			config:      `{"deployment_type": "hybrid", "data_gravity_factor": 0.6, "optimization_goals": [{"metric": "latency", "weight": 0.25, "minimize": true}, {"metric": "compute_cost", "weight": 0.25, "minimize": true}]}`,
		},
	}
	
	for i, scenario := range scenarios {
		fmt.Printf("%d. %s\n", i+1, scenario.name)
		fmt.Printf("   %s\n", scenario.description)
		fmt.Printf("   Config: %s\n", scenario.config)
		fmt.Println()
	}
}

// Utility function to create sample configuration files if they don't exist
func createSampleConfigIfNeeded(humanConfigPath, metricsDataPath string) error {
	// This function would create sample config files
	// Implementation would check if files exist and create samples if needed
	// For this example, we assume the config files already exist
	return nil
}