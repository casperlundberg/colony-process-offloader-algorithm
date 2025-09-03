# Compute Workload Simulation for CAPE Algorithm

This document describes the compute workload simulator that demonstrates how the CAPE algorithm behaves under realistic computing conditions.

##  Purpose

The compute simulator provides:

1. **Realistic Workload Generation** - Different types of compute workloads (web requests, batch processing, ML inference, data analytics)
2. **Resource Dynamics** - Simulates CPU, memory, network, and thermal behavior of executors  
3. **System Behavior** - Shows how CAPE adapts to changing conditions
4. **Performance Analysis** - Measures throughput, latency, resource utilization, and cost efficiency
5. **Algorithm Validation** - Demonstrates CAPE's learning and optimization capabilities

##  Simulation Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Workload      │    │     CAPE        │    │   Resource      │
│   Generator     │────│  Orchestrator   │────│   Simulator     │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ • Web requests  │    │ • Decision      │    │ • CPU usage     │
│ • Batch jobs    │    │   making        │    │ • Memory usage  │
│ • ML inference  │    │ • Learning      │    │ • Temperature   │
│ • Analytics     │    │ • Adaptation    │    │ • Health status │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

##  Simulation Components

### 1. **Workload Types** (`config/simulator_config.json`)

**Web Requests** (40% of workloads):
- Low CPU/Memory intensity
- High latency sensitivity  
- Cacheable and parallelizable
- Duration: 100ms - 2s

**Batch Processing** (30% of workloads):
- High CPU intensity
- Low latency sensitivity
- Large data processing
- Duration: 30s - 300s

**ML Inference** (20% of workloads):
- Very high CPU/Memory intensity
- Medium latency sensitivity
- GPU acceleration beneficial
- Duration: 5s - 60s

**Data Analytics** (10% of workloads):
- High memory and I/O intensity
- Low latency sensitivity
- Massive data processing
- Duration: 60s - 600s

### 2. **Resource Simulation**

**CPU Usage**:
- Dynamic allocation based on workload intensity
- Thermal throttling simulation
- Context switching overhead

**Memory Usage**:
- Working set simulation
- Memory pressure effects
- Cache behavior modeling

**Network Usage**:
- Data transfer simulation
- Latency variations
- Bandwidth contention

**Thermal Simulation**:
- Temperature modeling based on CPU usage
- Thermal throttling at >80°C
- Cooling curve simulation

### 3. **System Dynamics**

**Resource Fluctuation**:
- Random resource spikes (10% probability)
- Seasonal load patterns (business hours)
- Executor health variations

**Realistic Behavior**:
- Cold start penalties
- Cache warming effects
- Queueing theory implementation
- Resource contention modeling

##  Running the Simulation

### Basic Simulation
```bash
cd examples/compute-simulation
go run main.go
```

### With Custom Configuration
```bash
# Edit config/simulator_config.json first
go run main.go
```

### Expected Output
```
  Compute Workload Simulation for CAPE Algorithm
==================================================

 Initializing CAPE Orchestrator...
 Initializing Compute Workload Simulator...

 Simulation Configuration
===========================
Workload Arrival Rate: 0.50 processes/second
Duration Range: 30s - 5m0s
Resource Fluctuation: true
Seasonal Patterns: true

Workload Types:
  • web-request (weight=0.4): CPU=20.0%, Mem=10.0%, IO=30.0%
  • batch-processing (weight=0.3): CPU=80.0%, Mem=60.0%, IO=50.0%
  • ml-inference (weight=0.2): CPU=90.0%, Mem=80.0%, IO=40.0%
  • data-analytics (weight=0.1): CPU=70.0%, Mem=90.0%, IO=80.0%

 Starting CAPE Orchestrator...
 Starting Compute Workload Simulation...

 Simulation running... Press Ctrl+C to stop
════════════════════════════════════════════════

 Starting Compute Workload Simulator
Workload arrival rate: 0.50 processes/second
Simulation duration: 5m0s

 14:32:15 - Simulation Status
════════════════════════════════
 CAPE Orchestrator:
   Uptime: 45s
   Processes: 23 assigned, 18 completed, 2 failed
   CAPE Decisions: 23 (avg time: 15ms)
   Success Rate: 90.0%

  Workload Simulation:
   Total Workloads: 23
   Completed: 18, Failed: 2
   Throughput: 0.51 workloads/second

 Executor States:
    executor-edge-stockholm-01:
     CPU: 34.2%, Memory: 28.1%, Network: 15.3%
     Temperature: 56.4°C, Active Workloads: 2
    executor-cloud-aws-us-east-1-01:
     CPU: 67.8%, Memory: 45.9%, Network: 22.7%
     Temperature: 68.2°C, Active Workloads: 3
    executor-hpc-iceland-01:
     CPU: 12.1%, Memory: 8.4%, Network: 3.2%
     Temperature: 47.3°C, Active Workloads: 0

 CAPE Algorithm Status:
   Strategy: hybrid deployment
   Data Gravity Factor: 0.70
   Learning Rate: 0.001
```

##  Simulation Scenarios

The simulator automatically runs through different scenarios:

### 1. **Warmup Phase** (30 seconds)
- System initialization
- Cold start effects
- Cache warming

### 2. **Normal Load** (60 seconds)  
- Steady state operation
- Consistent workload arrival
- Baseline performance measurement

### 3. **Peak Load** (45 seconds)
- Increased arrival rate
- Resource contention
- CAPE adaptation behavior

### 4. **Recovery** (30 seconds)
- Load reduction
- Resource recovery
- System stabilization

##  Configuration Options

### Load Scenarios (`config/simulator_config.json`)

**Light Load** (0.1 processes/second):
```json
"light_load": {
  "workload_arrival_rate": 0.1,
  "description": "Light load scenario for testing basic functionality"
}
```

**Peak Load** (2.0 processes/second):
```json
"peak_load": {
  "workload_arrival_rate": 2.0,
  "description": "Peak load scenario for stress testing"  
}
```

**Burst Load** (5.0 processes/second):
```json
"burst_load": {
  "workload_arrival_rate": 5.0,
  "description": "Burst load scenario with intermittent spikes"
}
```

### Realism Features

**Resource Fluctuation**: Simulates random resource spikes and thermal effects
**Seasonal Patterns**: Models daily usage patterns (business hours vs night)
**Cache Effects**: Simulates cache warming and hit rate benefits
**Warmup Overhead**: Models cold start penalties for containers/functions
**Queueing Effects**: Realistic queue management and waiting times

##  Metrics and Analysis

### Primary Metrics
- **Throughput**: Workloads processed per second
- **Latency**: Average workload completion time
- **Success Rate**: Percentage of successful workloads
- **Resource Utilization**: CPU, memory, network usage across executors

### CAPE Algorithm Metrics
- **Decision Accuracy**: How well CAPE picks optimal executors
- **Adaptation Speed**: How quickly CAPE learns from outcomes  
- **Data Locality Score**: Effectiveness of data-gravity optimization
- **Cost Efficiency**: Cost per workload processed
- **Energy Efficiency**: Energy consumption optimization

### System Health Metrics
- **Queue Depth**: Number of waiting workloads
- **Executor Health**: Individual executor status and performance
- **Thermal Events**: Thermal throttling occurrences
- **Cache Performance**: Hit rates and warming effects

##  Algorithm Behavior Insights

### What the Simulation Reveals

**Learning Behavior**:
- CAPE adapts placement decisions based on executor performance
- Learns optimal strategies for different workload types
- Balances multiple objectives (latency, cost, throughput, energy)

**Resource Optimization**:
- Demonstrates data gravity effects (keeping compute near data)
- Shows thermal management (avoiding overheated executors)
- Exhibits load balancing across heterogeneous infrastructure

**Adaptation Dynamics**:
- Algorithm converges on optimal strategies within 20-50 decisions
- Handles resource fluctuations and executor failures gracefully
- Maintains performance during load spikes through intelligent placement

**Multi-Objective Optimization**:
- Real-time balancing of conflicting objectives
- Configuration-driven priority adjustment
- Learning from actual outcomes to improve future decisions

##  Use Cases

### Algorithm Development
- Test CAPE behavior under different conditions
- Validate learning and adaptation mechanisms
- Benchmark performance against different configurations

### System Design
- Size compute infrastructure for expected workloads
- Understand resource utilization patterns
- Plan capacity for peak load scenarios  

### Performance Tuning
- Optimize CAPE configuration parameters
- Test different deployment strategies
- Measure impact of various optimization goals

### Research and Analysis
- Study distributed computing placement strategies
- Analyze multi-objective optimization in practice
- Validate theoretical models with realistic simulations

##  Extending the Simulation

### Adding New Workload Types
```json
{
  "name": "video-processing",
  "weight": 0.1,
  "cpu_intensity": 0.95,
  "memory_intensity": 0.7,
  "io_intensity": 0.9,
  "gpu_required": true,
  "latency_sensitive": false
}
```

### Custom Executor Types
- Modify executor capabilities in `colonyos_metrics.json`
- Add specialized hardware (GPUs, FPGAs, etc.)
- Implement custom resource models

### Advanced Scenarios
- Network partitions
- Executor failures and recovery
- Data migration costs
- Compliance constraints

This compute simulation provides a comprehensive testing environment for the CAPE algorithm, demonstrating its behavior under realistic conditions and validating its optimization capabilities.