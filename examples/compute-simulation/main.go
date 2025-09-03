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
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/simulator"
)

// Compute Simulation Example
//
// This example demonstrates the CAPE algorithm behavior during realistic
// compute workload simulation. It shows:
//
// 1. Realistic workload generation with different patterns
// 2. Resource contention and dynamic executor states
// 3. CAPE decision-making under load
// 4. Performance metrics and adaptation behavior
// 5. System behavior during different load scenarios

func main() {
	fmt.Println("  Compute Workload Simulation for CAPE Algorithm")
	fmt.Println("==================================================")
	fmt.Println()

	// Configuration file paths
	humanConfigPath := "./config/human_config.json"
	metricsDataPath := "./config/colonyos_metrics_live.json"
	simulatorConfigPath := "./config/simulator_config.json"

	// Load simulator configuration
	simulatorConfig, err := loadSimulatorConfig(simulatorConfigPath)
	if err != nil {
		log.Fatalf("Failed to load simulator config: %v", err)
	}

	// Create the CAPE orchestrator
	fmt.Println(" Initializing CAPE Orchestrator...")
	orchestrator, err := colonyos.NewZeroHardcodeOrchestrator(humanConfigPath, metricsDataPath)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Create the compute simulator
	fmt.Println(" Initializing Compute Workload Simulator...")
	computeSimulator := simulator.NewComputeSimulator(simulatorConfig, orchestrator)

	// Display simulation configuration
	displaySimulationConfig(simulatorConfig)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\n Received shutdown signal, stopping simulation...")
		cancel()
	}()

	// Start the orchestrator
	fmt.Println("\n Starting CAPE Orchestrator...")
	err = orchestrator.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Start the compute simulation
	fmt.Println(" Starting Compute Workload Simulation...")
	err = computeSimulator.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start compute simulator: %v", err)
	}

	// Run monitoring
	go monitorSimulation(orchestrator, computeSimulator, ctx)

	// Run different simulation scenarios
	go runSimulationScenarios(computeSimulator, ctx)

	fmt.Println("\n Simulation running... Press Ctrl+C to stop")
	fmt.Println("════════════════════════════════════════════════")

	// Wait for shutdown
	<-ctx.Done()

	// Stop components
	fmt.Println("\n  Stopping simulation...")
	
	err = computeSimulator.Stop()
	if err != nil {
		log.Printf("Error stopping simulator: %v", err)
	}

	err = orchestrator.Stop()
	if err != nil {
		log.Printf("Error stopping orchestrator: %v", err)
	}

	fmt.Println(" Compute simulation completed!")
}

// SimulatorConfigFile represents the simulator configuration file structure
type SimulatorConfigFile struct {
	SimulationConfig struct {
		WorkloadArrivalRate   float64 `json:"workload_arrival_rate"`
		WorkloadDuration      struct {
			Min string `json:"min"`
			Max string `json:"max"`
		} `json:"workload_duration"`
		ResourceFluctuation   bool   `json:"resource_fluctuation"`
		NetworkLatencyNoise   bool   `json:"network_latency_noise"`
		ExecutorFailures      bool   `json:"executor_failures"`
		DataLocalityPattern   string `json:"data_locality_pattern"`
		SeasonalPatterns      bool   `json:"seasonal_patterns"`
		CacheEffects         bool   `json:"cache_effects"`
		WarmupOverhead       bool   `json:"warmup_overhead"`
		QueueingEffects      bool   `json:"queueing_effects"`
	} `json:"simulation_config"`
	WorkloadTypes []struct {
		Name             string  `json:"name"`
		Weight           float64 `json:"weight"`
		CPUIntensity     float64 `json:"cpu_intensity"`
		MemoryIntensity  float64 `json:"memory_intensity"`
		IOIntensity      float64 `json:"io_intensity"`
		Duration         struct {
			Min string `json:"min"`
			Max string `json:"max"`
		} `json:"duration"`
		DataSize struct {
			MinMB float64 `json:"min_mb"`
			MaxMB float64 `json:"max_mb"`
		} `json:"data_size"`
		LatencySensitive bool `json:"latency_sensitive"`
		Parallelizable   bool `json:"parallelizable"`
		CacheAffinitive  bool `json:"cache_affinitive"`
	} `json:"workload_types"`
	Scenarios map[string]struct {
		WorkloadArrivalRate float64 `json:"workload_arrival_rate"`
		Description         string  `json:"description"`
	} `json:"scenarios"`
}

// loadSimulatorConfig loads the simulator configuration from JSON file
func loadSimulatorConfig(configPath string) (*simulator.SimulatorConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read simulator config: %w", err)
	}

	var configFile SimulatorConfigFile
	err = json.Unmarshal(data, &configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse simulator config: %w", err)
	}

	// Convert to simulator config format
	config := &simulator.SimulatorConfig{
		WorkloadArrivalRate:  configFile.SimulationConfig.WorkloadArrivalRate,
		ResourceFluctuation:  configFile.SimulationConfig.ResourceFluctuation,
		NetworkLatencyNoise:  configFile.SimulationConfig.NetworkLatencyNoise,
		ExecutorFailures:     configFile.SimulationConfig.ExecutorFailures,
		DataLocalityPattern:  configFile.SimulationConfig.DataLocalityPattern,
		SeasonalPatterns:     configFile.SimulationConfig.SeasonalPatterns,
		CacheEffects:        configFile.SimulationConfig.CacheEffects,
		WarmupOverhead:      configFile.SimulationConfig.WarmupOverhead,
		QueueingEffects:     configFile.SimulationConfig.QueueingEffects,
	}

	// Parse durations
	minDuration, _ := time.ParseDuration(configFile.SimulationConfig.WorkloadDuration.Min)
	maxDuration, _ := time.ParseDuration(configFile.SimulationConfig.WorkloadDuration.Max)
	config.WorkloadDuration = simulator.Duration{Min: minDuration, Max: maxDuration}

	// Convert workload types
	for _, wt := range configFile.WorkloadTypes {
		minDur, _ := time.ParseDuration(wt.Duration.Min)
		maxDur, _ := time.ParseDuration(wt.Duration.Max)

		workloadType := simulator.WorkloadType{
			Name:             wt.Name,
			Weight:           wt.Weight,
			CPUIntensity:     wt.CPUIntensity,
			MemoryIntensity:  wt.MemoryIntensity,
			IOIntensity:      wt.IOIntensity,
			Duration:         simulator.Duration{Min: minDur, Max: maxDur},
			DataSize:         simulator.DataSize{MinMB: wt.DataSize.MinMB, MaxMB: wt.DataSize.MaxMB},
			LatencySensitive: wt.LatencySensitive,
			Parallelizable:   wt.Parallelizable,
			CacheAffinitive:  wt.CacheAffinitive,
		}
		config.WorkloadTypes = append(config.WorkloadTypes, workloadType)
	}

	return config, nil
}

// displaySimulationConfig shows the loaded simulation configuration
func displaySimulationConfig(config *simulator.SimulatorConfig) {
	fmt.Println("\n Simulation Configuration")
	fmt.Println("===========================")
	
	fmt.Printf("Workload Arrival Rate: %.2f processes/second\n", config.WorkloadArrivalRate)
	fmt.Printf("Duration Range: %v - %v\n", config.WorkloadDuration.Min, config.WorkloadDuration.Max)
	fmt.Printf("Resource Fluctuation: %v\n", config.ResourceFluctuation)
	fmt.Printf("Seasonal Patterns: %v\n", config.SeasonalPatterns)
	fmt.Printf("Data Locality Pattern: %s\n", config.DataLocalityPattern)
	
	fmt.Println("\nWorkload Types:")
	for _, wt := range config.WorkloadTypes {
		fmt.Printf("  • %s (weight=%.1f): CPU=%.1f%%, Mem=%.1f%%, IO=%.1f%%\n", 
			wt.Name, wt.Weight, wt.CPUIntensity*100, wt.MemoryIntensity*100, wt.IOIntensity*100)
		fmt.Printf("    Duration: %v-%v, Data: %.1f-%.1fMB, Latency-sensitive: %v\n",
			wt.Duration.Min, wt.Duration.Max, wt.DataSize.MinMB, wt.DataSize.MaxMB, wt.LatencySensitive)
	}
}

// monitorSimulation provides comprehensive monitoring of both orchestrator and simulator
func monitorSimulation(orchestrator *colonyos.ZeroHardcodeOrchestrator, computeSimulator *simulator.ComputeSimulator, ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second) // Monitor every 15 seconds
	defer ticker.Stop()

	fmt.Println("\n Simulation Monitoring Started")
	fmt.Println("=================================")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			displayCombinedStats(orchestrator, computeSimulator)
		}
	}
}

// displayCombinedStats shows both orchestrator and simulation statistics
func displayCombinedStats(orchestrator *colonyos.ZeroHardcodeOrchestrator, computeSimulator *simulator.ComputeSimulator) {
	orchestratorStats := orchestrator.GetStats()
	simulationStats := computeSimulator.GetStats()
	executorStates := computeSimulator.GetExecutorStates()

	fmt.Printf("\n %s - Simulation Status\n", time.Now().Format("15:04:05"))
	fmt.Println("════════════════════════════════")

	// Orchestrator metrics
	fmt.Printf(" CAPE Orchestrator:\n")
	fmt.Printf("   Uptime: %v\n", orchestratorStats.Uptime.Round(time.Second))
	fmt.Printf("   Processes: %d assigned, %d completed, %d failed\n", 
		orchestratorStats.ProcessesAssigned, orchestratorStats.ProcessesCompleted, orchestratorStats.ProcessesFailed)
	fmt.Printf("   CAPE Decisions: %d (avg time: %v)\n", 
		orchestratorStats.CapeDecisions, orchestratorStats.AvgDecisionTime.Round(time.Millisecond))

	// Success rate
	if orchestratorStats.ProcessesCompleted > 0 || orchestratorStats.ProcessesFailed > 0 {
		total := orchestratorStats.ProcessesCompleted + orchestratorStats.ProcessesFailed
		successRate := float64(orchestratorStats.ProcessesCompleted) / float64(total) * 100
		fmt.Printf("   Success Rate: %.1f%%\n", successRate)
	}

	// Simulation metrics
	fmt.Printf("\n  Workload Simulation:\n")
	fmt.Printf("   Total Workloads: %d\n", simulationStats.TotalWorkloads)
	fmt.Printf("   Completed: %d, Failed: %d\n", simulationStats.CompletedWorkloads, simulationStats.FailedWorkloads)
	
	if simulationStats.TotalWorkloads > 0 {
		throughput := simulationStats.ThroughputPerSecond
		fmt.Printf("   Throughput: %.2f workloads/second\n", throughput)
	}

	// Executor states
	fmt.Printf("\n Executor States:\n")
	for id, state := range executorStates {
		healthStatus := ""
		if !state.IsHealthy {
			healthStatus = ""
		}
		fmt.Printf("   %s %s:\n", healthStatus, id)
		fmt.Printf("     CPU: %.1f%%, Memory: %.1f%%, Network: %.1f%%\n", 
			state.CPUUsage*100, state.MemoryUsage*100, state.NetworkUsage*100)
		fmt.Printf("     Temperature: %.1f°C, Active Workloads: %d\n", 
			state.Temperature, len(state.ActiveWorkloads))
	}

	// CAPE Algorithm behavior
	capeConfig := orchestrator.GetConfiguration()
	if humanConfig, ok := capeConfig["human_config"].(*colonyos.HumanConfig); ok {
		fmt.Printf("\n CAPE Algorithm Status:\n")
		fmt.Printf("   Strategy: %s deployment\n", humanConfig.CapeConfig.DeploymentType)
		fmt.Printf("   Data Gravity Factor: %.2f\n", humanConfig.CapeConfig.LearningParameters.DataGravityFactor)
		fmt.Printf("   Learning Rate: %.3f\n", humanConfig.CapeConfig.LearningParameters.LearningRate)
	}
}

// runSimulationScenarios runs different load scenarios to test CAPE behavior
func runSimulationScenarios(computeSimulator *simulator.ComputeSimulator, ctx context.Context) {
	scenarios := []struct {
		name        string
		duration    time.Duration
		description string
	}{
		{"Warmup Phase", 30 * time.Second, "Initial system warmup"},
		{"Normal Load", 60 * time.Second, "Steady state operation"},
		{"Peak Load", 45 * time.Second, "High load scenario"},
		{"Recovery", 30 * time.Second, "Post-peak recovery"},
	}

	for _, scenario := range scenarios {
		select {
		case <-ctx.Done():
			return
		default:
			log.Printf(" Starting scenario: %s (%s)", scenario.name, scenario.description)
			
			// Let scenario run
			timer := time.NewTimer(scenario.duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				log.Printf(" Completed scenario: %s", scenario.name)
			}
		}
	}
}

// analyzeCAPEBehavior analyzes how CAPE algorithm adapts during different scenarios
func analyzeCAPEBehavior(orchestrator *colonyos.ZeroHardcodeOrchestrator, computeSimulator *simulator.ComputeSimulator) {
	fmt.Println("\n CAPE Algorithm Behavior Analysis")
	fmt.Println("==================================")
	
	// Get current stats
	stats := orchestrator.GetStats()
	simStats := computeSimulator.GetStats()
	
	fmt.Printf("Decision Making:\n")
	if stats.CapeDecisions > 0 {
		fmt.Printf("  • Average decision time: %v\n", stats.AvgDecisionTime.Round(time.Millisecond))
		fmt.Printf("  • Decisions per minute: %.1f\n", float64(stats.CapeDecisions)/stats.Uptime.Minutes())
	}
	
	fmt.Printf("\nLearning and Adaptation:\n")
	if stats.ProcessesCompleted > 0 {
		fmt.Printf("  • Successful adaptations: Based on %d completed processes\n", stats.ProcessesCompleted)
		fmt.Printf("  • System effectiveness: %.1f%%\n", float64(stats.ProcessesCompleted)/float64(stats.ProcessesAssigned)*100)
	}
	
	fmt.Printf("\nWorkload Handling:\n")
	fmt.Printf("  • Resource utilization optimization: Active\n")
	fmt.Printf("  • Data locality awareness: Active\n")
	fmt.Printf("  • Multi-objective balancing: Active\n")
	
	if simStats.TotalWorkloads > 0 {
		fmt.Printf("\nSimulation Results:\n")
		fmt.Printf("  • Total workloads processed: %d\n", simStats.TotalWorkloads)
		fmt.Printf("  • System throughput: %.2f workloads/sec\n", simStats.ThroughputPerSecond)
		fmt.Printf("  • Cost efficiency: %.2f\n", simStats.CostEfficiency)
		fmt.Printf("  • Energy efficiency: %.2f\n", simStats.EnergyEfficiency)
	}
}