package fixtures

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// TestDataGenerator creates realistic test data for algorithm validation
type TestDataGenerator struct {
	rand *rand.Rand
	seed int64
}

func NewTestDataGenerator(seed int64) *TestDataGenerator {
	return &TestDataGenerator{
		rand: rand.New(rand.NewSource(seed)),
		seed: seed,
	}
}

func (g *TestDataGenerator) GetSeed() int64 {
	return g.seed
}

// Generate system states for various scenarios
func (g *TestDataGenerator) GenerateSystemStates(count int) []models.SystemState {
	states := make([]models.SystemState, count)
	
	for i := 0; i < count; i++ {
		// Create realistic load patterns
		baseLoad := 0.2 + g.rand.Float64()*0.6 // 20-80% base load
		
		// Add some correlation between metrics
		computeUsage := baseLoad + (g.rand.Float64()-0.5)*0.3
		memoryUsage := computeUsage*0.8 + g.rand.Float64()*0.4
		diskUsage := 0.1 + g.rand.Float64()*0.5
		networkUsage := 0.1 + g.rand.Float64()*0.7
		
		// Clamp to valid ranges
		computeUsage = clampFloat64(computeUsage, 0.0, 1.0)
		memoryUsage = clampFloat64(memoryUsage, 0.0, 1.0)
		diskUsage = clampFloat64(diskUsage, 0.0, 1.0)
		networkUsage = clampFloat64(networkUsage, 0.0, 1.0)
		
		// Queue depth correlates with compute usage
		queueDepth := int(computeUsage*50 + g.rand.Float64()*20)
		
		masterUsage := 0.1 + computeUsage*0.4 + g.rand.Float64()*0.2
		masterUsage = clampFloat64(masterUsage, 0.0, 1.0)
		
		states[i] = models.SystemState{
			QueueDepth:        queueDepth,
			QueueThreshold:    20 + g.rand.Intn(30),
			QueueWaitTime:     time.Duration(queueDepth*2+g.rand.Intn(20)) * time.Second,
			QueueThroughput:   5.0 + g.rand.Float64()*10.0,
			ComputeUsage:      models.Utilization(computeUsage),
			MemoryUsage:       models.Utilization(memoryUsage),
			DiskUsage:         models.Utilization(diskUsage),
			NetworkUsage:      models.Utilization(networkUsage),
			MasterUsage:       models.Utilization(masterUsage),
			ActiveConnections: 10 + g.rand.Intn(200),
			Timestamp:         time.Now().Add(-time.Duration(g.rand.Intn(3600)) * time.Second),
			TimeSlot:          g.rand.Intn(24),
			DayOfWeek:         g.rand.Intn(7),
		}
	}
	
	return states
}

// Generate diverse process workloads
func (g *TestDataGenerator) GenerateProcesses(count int) []models.Process {
	processes := make([]models.Process, count)
	
	processTypes := []struct {
		name              string
		cpuRange          [2]float64
		memoryRange       [2]int64
		dataRange         [2]int64
		durationRange     [2]time.Duration
		priorityRange     [2]int
		realTimeProb      float64
		safetyCriticalProb float64
		sensitivityRange  [2]int
		securityRange     [2]int
	}{
		{
			name:              "web-service",
			cpuRange:          [2]float64{0.5, 2.0},
			memoryRange:       [2]int64{512 * MB, 4 * GB},
			dataRange:         [2]int64{1 * KB, 500 * KB},
			durationRange:     [2]time.Duration{100 * time.Millisecond, 2 * time.Second},
			priorityRange:     [2]int{7, 10},
			realTimeProb:      0.8,
			safetyCriticalProb: 0.1,
			sensitivityRange:  [2]int{0, 3},
			securityRange:     [2]int{1, 4},
		},
		{
			name:              "compute-intensive",
			cpuRange:          [2]float64{4.0, 16.0},
			memoryRange:       [2]int64{2 * GB, 32 * GB},
			dataRange:         [2]int64{10 * KB, 1 * MB},
			durationRange:     [2]time.Duration{30 * time.Second, 10 * time.Minute},
			priorityRange:     [2]int{4, 8},
			realTimeProb:      0.1,
			safetyCriticalProb: 0.05,
			sensitivityRange:  [2]int{0, 4},
			securityRange:     [2]int{1, 5},
		},
		{
			name:              "data-processing",
			cpuRange:          [2]float64{1.0, 4.0},
			memoryRange:       [2]int64{4 * GB, 64 * GB},
			dataRange:         [2]int64{1 * MB, 1 * GB},
			durationRange:     [2]time.Duration{2 * time.Minute, 1 * time.Hour},
			priorityRange:     [2]int{3, 7},
			realTimeProb:      0.0,
			safetyCriticalProb: 0.0,
			sensitivityRange:  [2]int{2, 5},
			securityRange:     [2]int{2, 5},
		},
		{
			name:              "batch-job",
			cpuRange:          [2]float64{2.0, 8.0},
			memoryRange:       [2]int64{1 * GB, 16 * GB},
			dataRange:         [2]int64{100 * MB, 10 * GB},
			durationRange:     [2]time.Duration{10 * time.Minute, 4 * time.Hour},
			priorityRange:     [2]int{1, 5},
			realTimeProb:      0.0,
			safetyCriticalProb: 0.0,
			sensitivityRange:  [2]int{1, 4},
			securityRange:     [2]int{1, 4},
		},
		{
			name:              "ml-training",
			cpuRange:          [2]float64{8.0, 32.0},
			memoryRange:       [2]int64{16 * GB, 128 * GB},
			dataRange:         [2]int64{1 * GB, 100 * GB},
			durationRange:     [2]time.Duration{1 * time.Hour, 24 * time.Hour},
			priorityRange:     [2]int{4, 8},
			realTimeProb:      0.0,
			safetyCriticalProb: 0.0,
			sensitivityRange:  [2]int{2, 5},
			securityRange:     [2]int{3, 5},
		},
		{
			name:              "iot-processing",
			cpuRange:          [2]float64{0.25, 1.0},
			memoryRange:       [2]int64{128 * MB, 1 * GB},
			dataRange:         [2]int64{100, 10 * KB},
			durationRange:     [2]time.Duration{10 * time.Millisecond, 1 * time.Second},
			priorityRange:     [2]int{6, 9},
			realTimeProb:      0.9,
			safetyCriticalProb: 0.3,
			sensitivityRange:  [2]int{0, 2},
			securityRange:     [2]int{1, 3},
		},
	}
	
	for i := 0; i < count; i++ {
		processType := processTypes[g.rand.Intn(len(processTypes))]
		
		cpuReq := processType.cpuRange[0] + g.rand.Float64()*(processType.cpuRange[1]-processType.cpuRange[0])
		memReq := processType.memoryRange[0] + g.rand.Int63n(processType.memoryRange[1]-processType.memoryRange[0])
		
		inputSize := processType.dataRange[0] + g.rand.Int63n(processType.dataRange[1]-processType.dataRange[0])
		outputSize := inputSize/2 + g.rand.Int63n(inputSize/2) // Output typically smaller than input
		
		durationMin := processType.durationRange[0]
		durationMax := processType.durationRange[1]
		duration := durationMin + time.Duration(g.rand.Int63n(int64(durationMax-durationMin)))
		
		priority := processType.priorityRange[0] + g.rand.Intn(processType.priorityRange[1]-processType.priorityRange[0]+1)
		
		realTime := g.rand.Float64() < processType.realTimeProb
		safetyCritical := g.rand.Float64() < processType.safetyCriticalProb
		
		sensitivity := processType.sensitivityRange[0] + g.rand.Intn(processType.sensitivityRange[1]-processType.sensitivityRange[0]+1)
		security := processType.securityRange[0] + g.rand.Intn(processType.securityRange[1]-processType.securityRange[0]+1)
		
		// Generate dependencies occasionally
		var dependencies []string
		if i > 0 && g.rand.Float64() < 0.2 { // 20% chance of having dependencies
			depCount := 1 + g.rand.Intn(3) // 1-3 dependencies
			for j := 0; j < depCount && j < i; j++ {
				depIndex := g.rand.Intn(i)
				dependencies = append(dependencies, fmt.Sprintf("process-%d", depIndex))
			}
		}
		
		processes[i] = models.Process{
			ID:                fmt.Sprintf("generated-process-%d", i),
			Type:              processType.name,
			Priority:          priority,
			CPURequirement:    cpuReq,
			MemoryRequirement: memReq,
			DiskRequirement:   memReq / 10, // Assume 10:1 memory to disk ratio
			InputSize:         inputSize,
			OutputSize:        outputSize,
			EstimatedDuration: duration,
			MaxDuration:       duration + duration/2, // 50% buffer
			RealTime:          realTime,
			SafetyCritical:    safetyCritical,
			DataSensitivity:   sensitivity,
			SecurityLevel:     security,
			Dependencies:      dependencies,
			SubmissionTime:    time.Now().Add(-time.Duration(g.rand.Intn(3600)) * time.Second),
			Status:            models.QUEUED,
		}
	}
	
	return processes
}

// Generate diverse offload targets
func (g *TestDataGenerator) GenerateTargets(count int) []models.OffloadTarget {
	targets := make([]models.OffloadTarget, count)
	
	targetTypes := []struct {
		targetType   models.TargetType
		capacityRange [2]float64
		latencyRange [2]time.Duration
		bandwidthRange [2]float64
		reliabilityRange [2]float64
		costRange [2]float64
		securityRange [2]int
		locations []string
		jurisdictions []string
		capabilities [][]string
	}{
		{
			targetType:   models.LOCAL,
			capacityRange: [2]float64{4.0, 16.0},
			latencyRange: [2]time.Duration{0, 5 * time.Millisecond},
			bandwidthRange: [2]float64{1 * GB, 10 * GB}, // Very high bandwidth
			reliabilityRange: [2]float64{0.99, 1.0},
			costRange: [2]float64{0.0, 0.01}, // Minimal cost
			securityRange: [2]int{5, 5}, // Maximum security
			locations: []string{"local"},
			jurisdictions: []string{"domestic"},
			capabilities: [][]string{
				{"always_available", "high_security", "low_latency"},
			},
		},
		{
			targetType:   models.EDGE,
			capacityRange: [2]float64{8.0, 64.0},
			latencyRange: [2]time.Duration{5 * time.Millisecond, 50 * time.Millisecond},
			bandwidthRange: [2]float64{100 * MB, 1 * GB},
			reliabilityRange: [2]float64{0.90, 0.99},
			costRange: [2]float64{0.05, 0.20},
			securityRange: [2]int{3, 5},
			locations: []string{"local", "regional"},
			jurisdictions: []string{"domestic", "regional"},
			capabilities: [][]string{
				{"low_latency", "edge_computing"},
				{"iot_optimized", "real_time"},
				{"compute_optimized", "gpu_accelerated"},
			},
		},
		{
			targetType:   models.PRIVATE_CLOUD,
			capacityRange: [2]float64{32.0, 256.0},
			latencyRange: [2]time.Duration{20 * time.Millisecond, 100 * time.Millisecond},
			bandwidthRange: [2]float64{50 * MB, 500 * MB},
			reliabilityRange: [2]float64{0.95, 0.999},
			costRange: [2]float64{0.08, 0.25},
			securityRange: [2]int{4, 5},
			locations: []string{"regional", "national"},
			jurisdictions: []string{"domestic"},
			capabilities: [][]string{
				{"high_security", "compliant"},
				{"memory_optimized", "storage_optimized"},
				{"enterprise_grade", "managed"},
			},
		},
		{
			targetType:   models.PUBLIC_CLOUD,
			capacityRange: [2]float64{64.0, 1024.0}, // Virtually unlimited
			latencyRange: [2]time.Duration{50 * time.Millisecond, 300 * time.Millisecond},
			bandwidthRange: [2]float64{10 * MB, 100 * MB},
			reliabilityRange: [2]float64{0.999, 0.9999},
			costRange: [2]float64{0.02, 0.50},
			securityRange: [2]int{1, 4},
			locations: []string{"national", "international"},
			jurisdictions: []string{"domestic", "eu", "asia", "americas"},
			capabilities: [][]string{
				{"scalable", "auto_scaling"},
				{"ml_optimized", "gpu_accelerated"},
				{"cost_optimized", "spot_instances"},
				{"serverless", "container_native"},
			},
		},
		{
			targetType:   models.FOG,
			capacityRange: [2]float64{2.0, 16.0},
			latencyRange: [2]time.Duration{1 * time.Millisecond, 20 * time.Millisecond},
			bandwidthRange: [2]float64{50 * MB, 200 * MB},
			reliabilityRange: [2]float64{0.80, 0.95},
			costRange: [2]float64{0.10, 0.30},
			securityRange: [2]int{2, 4},
			locations: []string{"local", "regional"},
			jurisdictions: []string{"domestic"},
			capabilities: [][]string{
				{"ultra_low_latency", "iot_optimized"},
				{"mobile_edge", "5g_enabled"},
			},
		},
	}
	
	for i := 0; i < count; i++ {
		typeIndex := g.rand.Intn(len(targetTypes))
		targetTypeInfo := targetTypes[typeIndex]
		
		// Generate capacity
		totalCapacity := targetTypeInfo.capacityRange[0] + 
			g.rand.Float64()*(targetTypeInfo.capacityRange[1]-targetTypeInfo.capacityRange[0])
		availableCapacity := totalCapacity * (0.3 + g.rand.Float64()*0.6) // 30-90% available
		
		// Generate memory
		memoryTotal := int64(totalCapacity * 4 * GB) // Assume 4GB per core
		memoryAvailable := int64(float64(memoryTotal) * (availableCapacity / totalCapacity))
		
		// Generate network characteristics
		latencyMin := targetTypeInfo.latencyRange[0]
		latencyMax := targetTypeInfo.latencyRange[1]
		latency := latencyMin + time.Duration(g.rand.Int63n(int64(latencyMax-latencyMin)))
		
		bandwidth := targetTypeInfo.bandwidthRange[0] + 
			g.rand.Float64()*(targetTypeInfo.bandwidthRange[1]-targetTypeInfo.bandwidthRange[0])
		
		stability := targetTypeInfo.reliabilityRange[0] + 
			g.rand.Float64()*(targetTypeInfo.reliabilityRange[1]-targetTypeInfo.reliabilityRange[0])
		
		reliability := targetTypeInfo.reliabilityRange[0] + 
			g.rand.Float64()*(targetTypeInfo.reliabilityRange[1]-targetTypeInfo.reliabilityRange[0])
		
		// Generate costs
		computeCost := targetTypeInfo.costRange[0] + 
			g.rand.Float64()*(targetTypeInfo.costRange[1]-targetTypeInfo.costRange[0])
		
		networkCost := computeCost * 0.1 * (1 + g.rand.Float64()) // Network cost related to compute cost
		energyCost := computeCost * 0.3 * (1 + g.rand.Float64()) // Energy cost related to compute cost
		
		// Select location and jurisdiction
		location := targetTypeInfo.locations[g.rand.Intn(len(targetTypeInfo.locations))]
		jurisdiction := targetTypeInfo.jurisdictions[g.rand.Intn(len(targetTypeInfo.jurisdictions))]
		
		// Select capabilities
		capabilitySet := targetTypeInfo.capabilities[g.rand.Intn(len(targetTypeInfo.capabilities))]
		
		// Select security level
		securityLevel := targetTypeInfo.securityRange[0] + 
			g.rand.Intn(targetTypeInfo.securityRange[1]-targetTypeInfo.securityRange[0]+1)
		
		targets[i] = models.OffloadTarget{
			ID:                fmt.Sprintf("generated-target-%s-%d", targetTypeInfo.targetType, i),
			Type:              targetTypeInfo.targetType,
			Location:          location,
			TotalCapacity:     totalCapacity,
			AvailableCapacity: availableCapacity,
			MemoryTotal:       memoryTotal,
			MemoryAvailable:   memoryAvailable,
			NetworkLatency:    latency,
			NetworkBandwidth:  bandwidth,
			NetworkStability:  stability,
			NetworkCost:       networkCost,
			ProcessingSpeed:   0.8 + g.rand.Float64()*0.8, // 0.8x - 1.6x speed
			Reliability:       reliability,
			ComputeCost:       computeCost,
			EnergyCost:        energyCost,
			SecurityLevel:     securityLevel,
			DataJurisdiction:  jurisdiction,
			Capabilities:      capabilitySet,
			CurrentLoad:       1.0 - (availableCapacity / totalCapacity),
			EstimatedWaitTime: time.Duration(g.rand.Intn(60)) * time.Second,
			LastSeen:          time.Now().Add(-time.Duration(g.rand.Intn(30)) * time.Second),
			HistoricalSuccess: 0.5 + g.rand.Float64()*0.4, // 50-90% historical success
		}
	}
	
	return targets
}

// Generate realistic outcomes for learning scenarios
func (g *TestDataGenerator) GenerateOutcomes(decisions []DecisionScenario) []decision.OffloadOutcome {
	outcomes := make([]decision.OffloadOutcome, len(decisions))
	
	for i, decision := range decisions {
		outcome := g.simulateOutcome(decision)
		outcome.DecisionID = decision.DecisionID
		outcome.ProcessID = decision.ProcessID
		outcomes[i] = outcome
	}
	
	return outcomes
}

func (g *TestDataGenerator) simulateOutcome(scenario DecisionScenario) decision.OffloadOutcome {
	// Simulate realistic outcome based on decision scenario
	success := true
	completedOnTime := true
	queueReduction := 0.0
	reward := 0.0
	
	// Base success probability
	successProb := 0.9
	
	if scenario.Decision.ShouldOffload && scenario.Decision.Target != nil {
		target := scenario.Decision.Target
		
		// Factor in target reliability
		successProb *= target.Reliability
		
		// Factor in network stability for data-intensive processes
		if scenario.Process.InputSize+scenario.Process.OutputSize > 10*MB {
			successProb *= target.NetworkStability
		}
		
		// Factor in capacity utilization
		utilizationFactor := target.AvailableCapacity / target.TotalCapacity
		if utilizationFactor < 0.2 { // Very low capacity
			successProb *= 0.7
		}
		
		success = g.rand.Float64() < successProb
		
		if success {
			// Calculate queue reduction benefit
			if scenario.SystemState.QueueDepth > scenario.SystemState.QueueThreshold {
				queueReduction = 0.3 + g.rand.Float64()*0.5 // 30-80% reduction
			} else {
				queueReduction = g.rand.Float64() * 0.3 // 0-30% reduction
			}
			
			// Check if completed on time
			networkDelay := float64(target.NetworkLatency) / float64(time.Millisecond)
			if scenario.Process.RealTime && networkDelay > 100 {
				completedOnTime = g.rand.Float64() < 0.7 // 70% chance of meeting deadline
			} else {
				completedOnTime = g.rand.Float64() < 0.95 // 95% chance
			}
			
			// Calculate reward
			reward = 0.5 + queueReduction + (utilizationFactor * 0.3)
			if completedOnTime {
				reward += 0.5
			} else {
				reward -= 0.3
			}
		} else {
			completedOnTime = false
			reward = -1.0 - g.rand.Float64() // Negative reward for failures
		}
	} else {
		// Local execution
		localCapacity := 1.0 - scenario.SystemState.ComputeUsage
		if localCapacity < 0.2 {
			successProb = 0.6 // Lower success with high local load
			completedOnTime = g.rand.Float64() < 0.7
		}
		
		success = g.rand.Float64() < successProb
		
		if success && scenario.SystemState.QueueDepth > 0 {
			queueReduction = -0.1 + g.rand.Float64()*0.2 // -10% to +10%
		}
		
		reward = 0.2 + g.rand.Float64()*0.3
		if !completedOnTime {
			reward -= 0.5
		}
	}
	
	// Generate attribution map
	attribution := g.generateAttribution(scenario, success, completedOnTime)
	
	outcome := decision.OffloadOutcome{
		Success:           success,
		CompletedOnTime:   completedOnTime,
		QueueReduction:    queueReduction,
		Reward:            reward,
		Attribution:       attribution,
		// SystemContext and ProcessContext removed - not in decision.OffloadOutcome
	}
	
	if scenario.Decision.Target != nil {
		outcome.TargetID = scenario.Decision.Target.ID
		// TargetContext removed - not in decision.OffloadOutcome
		
		// Add target-specific outcomes
		if !success {
			target := scenario.Decision.Target
			if target.Reliability < 0.8 {
				outcome.TargetOverloaded = g.rand.Float64() < 0.3
			}
			
			if target.NetworkStability < 0.8 {
				outcome.NetworkCongestion = g.rand.Float64() < 0.4
			}
		}
	}
	
	// Add policy violations occasionally
	if g.rand.Float64() < 0.05 { // 5% chance
		outcome.PolicyViolation = true
		outcome.ViolationType = []string{"soft_policy_deviation"}
	}
	
	return outcome
}

func (g *TestDataGenerator) generateAttribution(scenario DecisionScenario, success bool, completedOnTime bool) map[string]float64 {
	attribution := make(map[string]float64)
	
	// Base attribution - distribute among factors
	factors := []string{"QueueDepth", "ProcessorLoad", "NetworkCost", "LatencyCost", "EnergyCost", "PolicyCost"}
	
	// Start with equal attribution
	for _, factor := range factors {
		attribution[factor] = 1.0 / float64(len(factors))
	}
	
	// Adjust based on scenario characteristics
	if scenario.SystemState.QueueDepth > scenario.SystemState.QueueThreshold {
		attribution["QueueDepth"] *= 1.5 // Queue factor more important
	}
	
	if scenario.SystemState.ComputeUsage > 0.8 {
		attribution["ProcessorLoad"] *= 1.4 // Processor load more important
	}
	
	if scenario.Decision.ShouldOffload && scenario.Decision.Target != nil {
		target := scenario.Decision.Target
		
		// Network attribution based on data size and target characteristics
		dataSize := scenario.Process.InputSize + scenario.Process.OutputSize
		if dataSize > 10*MB {
			attribution["NetworkCost"] *= 1.6
		}
		
		if target.NetworkLatency > 100*time.Millisecond {
			attribution["LatencyCost"] *= 1.3
		}
		
		if target.ComputeCost > 0.20 {
			attribution["EnergyCost"] *= 1.2
		}
	}
	
	// Adjust based on outcome
	if !success {
		// Amplify the factor that most likely caused the failure
		maxFactor := ""
		maxValue := 0.0
		for factor, value := range attribution {
			if value > maxValue {
				maxValue = value
				maxFactor = factor
			}
		}
		if maxFactor != "" {
			attribution[maxFactor] *= 1.5
		}
	}
	
	// Normalize to sum to 1.0
	total := 0.0
	for _, value := range attribution {
		total += value
	}
	
	for factor := range attribution {
		attribution[factor] /= total
	}
	
	return attribution
}

// Generate diverse scenarios for comprehensive testing
func (g *TestDataGenerator) GenerateDiverseScenarios(count int) []DecisionScenario {
	scenarios := make([]DecisionScenario, count)
	
	processes := g.GenerateProcesses(count)
	states := g.GenerateSystemStates(count)
	targets := g.GenerateTargets(20) // Reuse targets across scenarios
	
	for i := 0; i < count; i++ {
		// Select random subset of targets for this scenario
		targetCount := 2 + g.rand.Intn(4) // 2-5 targets
		scenarioTargets := make([]models.OffloadTarget, targetCount)
		
		for j := 0; j < targetCount; j++ {
			scenarioTargets[j] = targets[g.rand.Intn(len(targets))]
		}
		
		// Generate decision (simplified)
		decision := decision.OffloadDecision{
			ShouldOffload: g.rand.Float64() < 0.6, // 60% offload rate
			Score:        g.rand.Float64(),
			Confidence:   0.5 + g.rand.Float64()*0.5,
		}
		
		if decision.ShouldOffload && len(scenarioTargets) > 0 {
			decision.Target = &scenarioTargets[g.rand.Intn(len(scenarioTargets))]
		}
		
		scenarios[i] = DecisionScenario{
			DecisionID:      fmt.Sprintf("scenario-decision-%d", i),
			ProcessID:       processes[i].ID,
			Process:         processes[i],
			SystemState:     states[i],
			AvailableTargets: scenarioTargets,
			Decision:        decision,
		}
	}
	
	return scenarios
}

// Constants for data sizes
const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

// Helper functions
func clampFloat64(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// DecisionScenario represents a test scenario
type DecisionScenario struct {
	DecisionID       string
	ProcessID        string
	Process          models.Process
	SystemState      models.SystemState
	AvailableTargets []models.OffloadTarget
	Decision         decision.OffloadDecision
	ExpectedOutcome  *decision.OffloadOutcome
	OptimalAction    string
	OptimalTarget    string
	ExpectedReward   float64
}