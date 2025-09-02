# Colony Process Offloader Algorithm

An **Adaptive Multi-Objective Offloading Algorithm** for ColonyOS that intelligently decides whether to offload computational processes to remote execution targets or keep them local based on system state, resource availability, and learned patterns.

## Overview

This algorithm implements a sophisticated decision-making system that:

- **Learns and adapts** weights based on execution outcomes
- **Discovers patterns** in system behavior over time
- **Enforces safety and policy constraints** rigorously
- **Optimizes multiple objectives** including queue reduction, load balancing, network costs, latency, energy efficiency, and policy compliance
- **Provides explainable decisions** with full audit trails

## Architecture

### Core Components

1. **Decision Engine** (`pkg/decision/`)
   - Makes offloading decisions using adaptive scoring
   - Applies discovered patterns to improve decision quality
   - Supports heterogeneous infrastructure (local, edge, fog, cloud)

2. **Adaptive Learner** (`pkg/learning/`)
   - Updates decision weights based on outcome feedback
   - Discovers behavioral patterns from historical data
   - Tracks performance improvement over static baseline

3. **Policy Engine** (`pkg/policy/`)
   - Enforces hard and soft policy constraints
   - Maintains comprehensive audit logs
   - Handles safety-critical and compliance requirements

4. **Algorithm Orchestrator** (`pkg/algorithm/`)
   - Integrates all components into a cohesive system
   - Manages configuration and health monitoring
   - Provides unified API for offloading decisions

### Data Models (`pkg/models/`)

- **Process**: Workloads that can be offloaded (supports both simple tasks and DAG pipelines)
- **OffloadTarget**: Execution destinations with capacity, network, and cost characteristics
- **SystemState**: Real-time system metrics and resource utilization
- **Types**: Common validation and utility types

## Features

### Multi-Objective Optimization

The algorithm balances six key objectives with adaptive weights:

- **Queue Reduction** (20% default): Minimize queue buildup
- **Load Balancing** (20% default): Distribute load effectively
- **Network Optimization** (20% default): Minimize network costs
- **Latency Minimization** (20% default): Reduce end-to-end latency
- **Energy Efficiency** (10% default): Optimize energy consumption
- **Policy Compliance** (10% default): Maintain policy adherence

### Adaptive Learning

- **Weight Adaptation**: Learns optimal weight combinations from outcomes
- **Pattern Discovery**: Identifies recurring conditions that predict good decisions
- **Performance Tracking**: Measures improvement over static baselines
- **Convergence Detection**: Automatically detects when weights stabilize

### Safety Guarantees

- **Hard Constraints**: Never violated (security, safety-critical, data sovereignty)
- **Soft Constraints**: Influence scoring but don't block decisions
- **Resource Protection**: Maintains minimum local compute and memory reserves
- **Failure Handling**: Graceful degradation with local fallback

### Policy Enforcement

- **Security Levels**: Ensures targets meet process security requirements
- **Data Jurisdiction**: Respects legal and compliance boundaries  
- **Safety-Critical**: Keeps critical processes local
- **Audit Logging**: Complete traceability of all decisions and violations

## Performance Requirements

The algorithm meets strict performance targets:

- **Decision Latency**: <500ms (95th percentile)
- **Throughput**: >100 decisions/second
- **Memory Usage**: <100MB steady state
- **Learning Convergence**: Within 200 decisions
- **Performance Gain**: >10% improvement over static baseline

## Getting Started

### Prerequisites

- Go 1.22.2 or later
- Dependencies managed via `go.mod`

### Installation

```bash
git clone https://github.com/casperlundberg/colony-process-offloader-algorithm
cd colony-process-offloader-algorithm
go mod download
```

### Basic Usage

```go
import (
    "github.com/casperlundberg/colony-process-offloader-algorithm/pkg/algorithm"
    "github.com/casperlundberg/colony-process-offloader-algorithm/pkg/decision"
    "github.com/casperlundberg/colony-process-offloader-algorithm/pkg/learning"
    "github.com/casperlundberg/colony-process-offloader-algorithm/pkg/policy"
)

// Create configuration
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
    // ... additional configuration
}

// Initialize algorithm
alg, err := algorithm.NewAlgorithm(config)
if err != nil {
    log.Fatal(err)
}

// Make offloading decision
decision, err := alg.MakeOffloadDecision(process, targets, systemState)
if err != nil {
    log.Fatal(err)
}

// Process execution outcome for learning
err = alg.ProcessOutcome(outcome)
if err != nil {
    log.Fatal(err)
}
```

### Running the Demo

```bash
go run main.go
```

This runs a simulation demonstrating:
- Algorithm initialization
- Decision-making for various process types
- Adaptive learning from outcomes  
- Policy enforcement
- Performance metrics tracking

## Testing

### Unit Tests

Run tests for individual components:

```bash
# Test all components
go test ./tests/unit/... -v

# Test specific components
go test ./tests/unit/models -v
go test ./tests/unit/decision -v
go test ./tests/unit/learning -v
go test ./tests/unit/policy -v
```

### Test Coverage

The test suite includes:

- **Models**: Validation, capabilities, resource calculations
- **Decision Engine**: Determinism, latency, score compliance
- **Adaptive Learner**: Weight normalization, convergence, pattern discovery
- **Policy Engine**: Hard/soft constraints, audit logging, safety enforcement

### Performance Benchmarks

Tests validate all performance requirements:

- Decision latency under load
- Throughput with concurrent requests
- Memory usage over time
- Learning convergence speed

## Configuration

### Algorithm Configuration

```go
type Config struct {
    InitialWeights      decision.AdaptiveWeights
    LearningConfig      learning.LearningConfig
    SafetyConstraints   policy.SafetyConstraints  
    PerformanceTargets  PerformanceTargets
    MonitoringConfig    MonitoringConfig
}
```

### Learning Parameters

- `WindowSize`: Number of outcomes in learning window (default: 100)
- `LearningRate`: Weight adjustment rate (default: 0.01)
- `ExplorationRate`: Exploration vs exploitation (default: 0.1)
- `MinSamples`: Minimum samples for pattern discovery (default: 10)

### Safety Constraints

- `MinLocalCompute`: Always keep this compute capacity local
- `MinLocalMemory`: Always keep this memory capacity local
- `MaxConcurrentOffloads`: Limit simultaneous offloads
- `MaxLatencyTolerance`: Maximum acceptable latency
- `DataSovereignty`: Enforce data jurisdiction rules

## Monitoring

### Performance Metrics

The algorithm provides comprehensive metrics:

```go
metrics := alg.GetPerformanceMetrics()
fmt.Printf("Decision Count: %d\n", metrics.DecisionCount)
fmt.Printf("Performance Gain: %.2f%%\n", metrics.PerformanceGain*100)
fmt.Printf("Convergence Status: %v\n", metrics.IsConverged)
fmt.Printf("Policy Violations: %d\n", metrics.PolicyStats.HardViolations)
```

### Health Monitoring

```go
healthy := alg.IsHealthy()
if !healthy {
    // Take corrective action
}
```

### Audit Logging

All decisions and policy evaluations are logged with:
- Timestamp and decision details
- Applied rules and violations
- Performance metrics
- Outcome attribution

## Development

### Project Structure

```
├── pkg/
│   ├── algorithm/     # Main orchestrator
│   ├── decision/      # Decision engine and scoring
│   ├── learning/      # Adaptive learning components
│   ├── models/        # Core data models
│   └── policy/        # Policy enforcement
├── tests/
│   ├── unit/          # Unit tests for all components
│   ├── fixtures/      # Test data and utilities
│   └── mocks/         # Mock implementations
├── main.go            # Demo application
└── *.md              # Documentation
```

### Adding New Target Types

1. Add the target type to `models.TargetType`
2. Update validation logic in `models.ValidTargetTypes()`
3. Add any specific policy rules
4. Update tests

### Adding New Policy Rules

```go
rule := policy.PolicyRule{
    Type:     models.HARD,  // or models.SOFT
    Priority: 1,
    Condition: func(p models.Process, t models.OffloadTarget) bool {
        // Return true if rule is satisfied
        return /* your condition */
    },
    Description: "Rule description",
}

err := policyEngine.AddRule(rule)
```

## Integration with ColonyOS

The algorithm is designed to integrate with ColonyOS through:

1. **Process Queue Interface**: Get processes awaiting execution
2. **Target Discovery**: Discover available execution targets
3. **Execution Interface**: Submit processes to selected targets
4. **Monitoring Interface**: Collect outcomes and performance data

See `COLONYOS_INTEGRATION_TESTS.md` for detailed integration scenarios.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write comprehensive tests
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

### Code Quality

- Follow Go conventions
- Maintain >95% test coverage
- Include performance benchmarks
- Document public APIs
- Add integration tests for new features

## License

[License information here]

## References

- [Algorithm Specification](ALGORITHM_SPECIFICATION.md)
- [Test Suite Design](TEST_SUITE_DESIGN.md)
- [Learning Behavior Scenarios](LEARNING_BEHAVIOR_SCENARIOS.md)
- [ColonyOS Integration](COLONYOS_INTEGRATION_TESTS.md)
- [TDD Status](TDD_STATUS.md)