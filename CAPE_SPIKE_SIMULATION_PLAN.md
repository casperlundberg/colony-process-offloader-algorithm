# CAPE Spike Simulation & Weight Adaptation Plan

## Overview
This document outlines the development plan for a simulator that tests CAPE's ability to learn and adapt to process spikes through weight optimization. The simulator will generate configurable spike patterns and measure how well CAPE learns to predict and handle them over time.

## Phase 1: Simulation Architecture

```
CAPE Spike Simulator
├── Spike Generator (configurable patterns)
├── Queue State Manager (realistic process queues) 
├── CAPE Decision Engine (with learning algorithms)
├── Executor Deployment Simulator (startup delays, costs)
├── Metrics Collector (adaptation tracking)
└── Weight Evolution Tracker (learning progress)
```

## Phase 2: Spike Pattern Configuration

### Configuration Example
```json
{
  "spike_scenarios": [
    {
      "name": "ml_training_burst",
      "description": "Daily ML training job spike at 9 AM",
      "pattern": {
        "type": "predictable_daily",
        "trigger_time": "09:00:00",
        "duration_minutes": 45,
        "process_rate_multiplier": 8.0,
        "executor_type": "ml",
        "priority_distribution": [7, 8, 9],
        "data_location": "iceland"
      }
    },
    {
      "name": "edge_iot_surge", 
      "description": "Unpredictable IoT sensor data surge",
      "pattern": {
        "type": "random_poisson",
        "mean_interval_minutes": 120,
        "duration_minutes": 15,
        "process_rate_multiplier": 15.0,
        "executor_type": "edge",
        "priority_distribution": [8, 9, 9],
        "data_location": "stockholm"
      }
    },
    {
      "name": "batch_processing_wave",
      "description": "End-of-day batch processing wave",
      "pattern": {
        "type": "predictable_daily",
        "trigger_time": "18:00:00", 
        "duration_minutes": 90,
        "process_rate_multiplier": 5.0,
        "executor_type": "cloud",
        "priority_distribution": [5, 6, 7],
        "data_location": "aws_east"
      }
    }
  ]
}
```

### Spike Pattern Types
- **predictable_daily**: Recurring daily patterns (e.g., business hours)
- **random_poisson**: Random spikes following Poisson distribution
- **seasonal**: Weekly/monthly patterns
- **cascade**: One spike triggering secondary spikes
- **gradual_ramp**: Slow build-up rather than sudden spike

## Phase 3: CAPE Learning Framework

### Weight Adaptation Engine
```go
type CAPEWeightAdaptation struct {
    // Initial weights from config
    ExecutorWeights     map[string]*models.ExecutorOptimizationWeights
    SpikeHistory       []SpikeEvent
    PerformanceHistory []SpikeOutcome
    
    // Learning parameters
    LearningRate       float64 // 0.01
    ExplorationFactor  float64 // 0.1
    MemoryWindow       int     // 100 decisions to remember
}

func NewCAPEWeightAdaptation() *CAPEWeightAdaptation {
    return &CAPEWeightAdaptation{
        ExecutorWeights:    loadExecutorWeights(),
        SpikeHistory:      make([]SpikeEvent, 0),
        PerformanceHistory: make([]SpikeOutcome, 0),
        LearningRate:      0.01,
        ExplorationFactor: 0.1,
        MemoryWindow:      100,
    }
}

func (cwa *CAPEWeightAdaptation) HandleSpikeOutcome(spikeEvent SpikeEvent, capeDecision CAPEDecision, actualOutcome SpikeOutcome) error {
    // Calculate performance metrics
    performance := SpikePerformance{
        QueueClearTime:      actualOutcome.QueueClearTime,
        TotalCost:          actualOutcome.TotalCost,
        MissedSLAs:         actualOutcome.MissedSLAs,
        OverProvisioning:   actualOutcome.OverProvisioning,
        PredictionAccuracy: cwa.calculatePredictionAccuracy(spikeEvent, capeDecision),
    }
    
    // Update weights based on outcomes
    if performance.QueueClearTime > cwa.TargetClearTime {
        // Too slow - increase urgency weights
        cwa.increaseStartupTimePreference()
        cwa.increasePrioritySensitivity()
    }
    
    if performance.TotalCost > cwa.BudgetThreshold {
        // Too expensive - increase cost sensitivity  
        cwa.increaseCostWeights()
        cwa.decreaseOverProvisioningTendency()
    }
    
    if performance.MissedSLAs > 0 {
        // Missed deadlines - increase latency focus
        cwa.increaseLatencyWeights()
        cwa.increasePreemptiveScaling()
    }
    
    return nil
}
```

### Learning Algorithms Integration
- **ARIMA(3,1,2)**: Learn spike timing patterns
- **EWMA(0.167)**: Smooth spike intensity predictions
- **CUSUM(0.5σ,5σ)**: Detect unusual spike patterns
- **SGD(0.001)**: Optimize weight adjustments
- **Thompson Sampling**: Explore vs exploit scaling strategies
- **Q-Learning**: Learn long-term spike handling policies

## Phase 4: Simulation Execution Flow

```go
type SpikeSimulation struct {
    CAPE          *algorithm.ConfigurableCAPE
    ExecutorCatalog *models.ExecutorCatalog
    QueueManager   *QueueManager
    MetricsCollector *MetricsCollector
}

func (ss *SpikeSimulation) RunSimulation(durationHours int, scenarios []SpikeScenario) (*AdaptationReport, error) {
    // Run multi-spike simulation to test CAPE adaptation
    timeline := make([]SimulationEvent, 0)
    currentTime := time.Now()
    
    // Generate spike events
    spikeEvents, err := ss.generateSpikeTimeline(durationHours, scenarios)
    if err != nil {
        return nil, fmt.Errorf("failed to generate spike timeline: %w", err)
    }
    
    for _, spike := range spikeEvents {
        log.Printf("⚡ Spike: %s at T+%v", spike.Name, spike.StartTime)
        
        // Pre-spike: CAPE makes predictions
        capePrediction, err := ss.CAPE.PredictAndDecide(
            ss.QueueManager.GetState(),
            ss.getPatternContext(spike),
            30*time.Minute, // lookahead
        )
        if err != nil {
            log.Printf("CAPE prediction failed: %v", err)
            continue
        }
        
        // Execute CAPE recommendations
        deployedExecutors, err := ss.deployRecommendedExecutors(capePrediction)
        if err != nil {
            log.Printf("Executor deployment failed: %v", err)
        }
        
        // Simulate the actual spike
        spikeOutcome, err := ss.simulateSpikeExecution(spike, deployedExecutors, capePrediction)
        if err != nil {
            log.Printf("Spike simulation failed: %v", err)
            continue
        }
        
        // CAPE learns from outcome
        err = ss.CAPE.LearnFromOutcome(spike, capePrediction, spikeOutcome)
        if err != nil {
            log.Printf("Learning from outcome failed: %v", err)
        }
        
        // Track weight evolution
        ss.recordWeightChanges(spike.Name, ss.CAPE.GetCurrentWeights())
        
        currentTime = spike.EndTime
    }
    
    return ss.generateAdaptationReport(), nil
}
```

### Simulation Loop Details
1. **Baseline Period**: Normal load to establish baseline performance
2. **Spike Generation**: Create spike according to configured patterns
3. **CAPE Prediction**: Use current weights to predict and recommend scaling
4. **Deployment Simulation**: Account for startup times, costs, failures
5. **Spike Execution**: Run actual workload with deployed infrastructure
6. **Outcome Analysis**: Measure performance against SLAs and costs
7. **Weight Update**: Adjust CAPE weights based on performance
8. **Pattern Recognition**: Update historical spike patterns

## Phase 5: Key Metrics to Track

### Performance Metrics
```go
type AdaptationMetrics struct {
    // Prediction accuracy over time
    ARIMAAccuracyTrend      []float64 // [0.65, 0.72, 0.78, 0.81, 0.85] - Improving
    SpikeDetectionRate      []float64 // [0.40, 0.55, 0.70, 0.82, 0.90] - Learning patterns
    
    // Cost optimization learning
    CostEfficiencyTrend     []float64 // [0.60, 0.65, 0.72, 0.76, 0.83] - Reducing waste
    OverProvisioningRate    []float64 // [0.35, 0.28, 0.20, 0.15, 0.12] - Less excess
    
    // Latency performance 
    QueueClearTimeImprovement []time.Duration // [300s, 250s, 200s, 180s, 150s]
    SLAMissRate             []float64 // [0.15, 0.12, 0.08, 0.05, 0.02] - Fewer missed deadlines
    
    // Weight adaptation
    WeightStability         []float64 // [0.1, 0.3, 0.5, 0.7, 0.9] - Converging to optimal
    ExplorationVsExploitation []float64 // [0.9, 0.7, 0.5, 0.3, 0.1] - Less random, more learned
    
    // Timestamps for correlation
    Timestamps              []time.Time
}

func NewAdaptationMetrics() *AdaptationMetrics {
    return &AdaptationMetrics{
        ARIMAAccuracyTrend:        make([]float64, 0),
        SpikeDetectionRate:        make([]float64, 0),
        CostEfficiencyTrend:       make([]float64, 0),
        OverProvisioningRate:      make([]float64, 0),
        QueueClearTimeImprovement: make([]time.Duration, 0),
        SLAMissRate:               make([]float64, 0),
        WeightStability:           make([]float64, 0),
        ExplorationVsExploitation: make([]float64, 0),
        Timestamps:                make([]time.Time, 0),
    }
}
```

### Learning Progress Indicators
- **Prediction Accuracy**: How well CAPE predicts spike timing and intensity
- **Cost Efficiency**: Ratio of value delivered to resources consumed
- **SLA Compliance**: Percentage of processes meeting deadlines
- **Resource Utilization**: Efficiency of deployed executor capacity
- **Weight Convergence**: Stability of learned optimization weights
- **Pattern Recognition**: Ability to identify recurring spike patterns

## Phase 6: Expected Learning Behaviors

### Learning Timeline
```
Spike Pattern Learning:
Day 1-3: "Random" responses, high cost, missed predictions
Day 4-7: Pattern recognition emerges, better timing
Day 8-14: Preemptive scaling, cost optimization  
Day 15+: Predictive deployment before spikes
```

### Weight Evolution Examples
```
Initial → Learned Weights:

ML Executors (Iceland):
- startup_time_importance: 0.3 → 0.7 (learns speed matters)
- cost_sensitivity: 0.8 → 0.4 (learns performance > cost for spikes)
- data_gravity_factor: 0.6 → 0.9 (learns data transfer kills performance)

Edge Executors (Stockholm):
- latency_weight: 0.5 → 0.9 (learns ultra-low latency critical)
- cost_per_hour: 0.7 → 0.2 (learns cost less important than speed)
- max_instances: 15 → 25 (learns to scale aggressively)

Cloud Executors (AWS):
- supports_spot: false → true (learns cost optimization)
- min_lease_time_min: 60 → 5 (learns flexibility important)
- startup_time_sec: high penalty → low penalty (learns batch can wait)
```

## Phase 7: Simulation Output

### Expected Results Report
```
🎯 CAPE Adaptation Report (14-day simulation)
═══════════════════════════════════════════════

📊 Spike Handling Performance:
   Day 1-3:   Average queue clear: 285s, Cost: $45/spike, SLA miss: 15%
   Day 4-7:   Average queue clear: 220s, Cost: $38/spike, SLA miss: 8% 
   Day 8-11:  Average queue clear: 180s, Cost: $32/spike, SLA miss: 4%
   Day 12-14: Average queue clear: 150s, Cost: $28/spike, SLA miss: 1%

🧠 Learning Convergence:
   ✅ ML spikes: Learned to pre-deploy in Iceland (data gravity)
   ✅ Edge spikes: Learned to prioritize startup_time over cost
   ✅ Batch waves: Learned to use spot instances for cost savings

📈 Weight Evolution:
   exec-ml-iceland-01: startup_time_sec importance: 0.2 → 0.8
   exec-edge-stockholm-01: cost_per_hour importance: 0.7 → 0.2  
   data_transfer_costs: threshold sensitivity: 0.5 → 0.9

🔍 Pattern Recognition:
   ✅ Daily 9 AM ML spike: Detected after day 4, preemptive scaling by day 8
   ✅ Random IoT surges: Pattern learned, 85% prediction accuracy by day 12
   ✅ Evening batch waves: Cost optimization learned, 40% cost reduction

⚡ Algorithm Performance:
   ARIMA accuracy: 65% → 85% (learned seasonal patterns)
   CUSUM sensitivity: Reduced false positives by 60%
   Thompson Sampling: Exploration→exploitation transition at day 10
   Q-Learning: Convergence to optimal policy at day 14
```

### Visualization Outputs
- **Weight Evolution Graphs**: Show how executor weights change over time
- **Spike Response Timeline**: Before/after learning performance comparison  
- **Cost vs Performance Trade-offs**: Pareto frontier improvements
- **Prediction Accuracy Trends**: ARIMA/CUSUM improvement curves
- **SLA Compliance Dashboard**: Real-time learning progress

## Phase 8: Implementation Components

### Required Files
```
pkg/simulation/
├── spike_generator.go          # Generate configurable spike patterns
├── weight_adaptation.go        # CAPE learning from spike outcomes  
├── executor_deployment_sim.go  # Simulate infrastructure deployment
├── metrics_collector.go        # Track learning progress
├── pattern_recognition.go      # Identify recurring spikes
└── simulation_runner.go        # Orchestrate full simulation

config/
├── spike_scenarios.json        # Spike pattern configurations
└── simulation_config.json      # Simulation parameters

examples/
└── spike-simulation/
    └── main.go                 # Run spike simulation demo
```

### Success Criteria
1. **Learning Demonstrated**: Measurable improvement in spike handling over time
2. **Weight Convergence**: CAPE weights stabilize to optimal values
3. **Cost Optimization**: Reduced infrastructure costs while meeting SLAs
4. **Predictive Scaling**: CAPE learns to deploy before spikes occur
5. **Pattern Recognition**: Different spike types handled with appropriate strategies

## Implementation Priority
1. **Phase 1-2**: Basic spike generation and pattern configuration ✅ High Priority
2. **Phase 3**: Weight adaptation learning framework ✅ High Priority  
3. **Phase 4**: Simulation execution loop ✅ Medium Priority
4. **Phase 5-6**: Metrics collection and analysis ✅ Medium Priority
5. **Phase 7-8**: Reporting and visualization ✅ Low Priority

## Notes
- Start with simple daily predictable spikes before adding complex patterns
- Use the existing executor catalog (v3) as the foundation
- Integrate with existing CAPE learning algorithms (ARIMA, CUSUM, etc.)
- Focus on demonstrating weight adaptation rather than perfect prediction
- Measure learning convergence - the key success metric

---
**Goal**: Demonstrate that CAPE can learn from experience to handle recurring spike patterns more efficiently over time through intelligent weight adaptation and preemptive executor deployment.