# Zero Hardcode ColonyOS-CAPE Architecture

This document describes the completely configuration-driven CAPE system that uses **ZERO hardcoded values** and follows native ColonyOS data structures.

##  Architecture Principles

### 1. Configuration-Driven Behavior
- **Human Configuration**: `config/human_config.json` - All behavior parameters set by humans
- **ColonyOS Metrics**: `config/colonyos_metrics.json` - Real system data in ColonyOS format  
- **ColonyOS Specs**: Native data structures from `colonyOS-examples/` directory
- **Zero Hardcoded Values**: No magic numbers or behavior baked into code

### 2. Native ColonyOS Integration  
- Uses exact ColonyOS data structures from source code
- Follows ColonyOS parsing conventions (CPU: "4000m", Memory: "16Gi")
- Implements ColonyOS process/executor state management
- Compatible with ColonyOS Prometheus metrics format

### 3. Clear Data Source Separation

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   Human Config      │    │  ColonyOS Metrics   │    │  ColonyOS Examples  │
│   (human_config.json)│    │ (colonyos_metrics.json)│    │  (executor.json, etc)│
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
           │                           │                           │
           ▼                           ▼                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Configuration Loader                                     │
│  • Validates all configs   • Converts formats   • Provides data access     │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                Zero Hardcode CAPE Orchestrator                              │
│  • Uses only config data   • Native ColonyOS types   • Real-time metrics   │
└─────────────────────────────────────────────────────────────────────────────┘
```

##  Configuration File Structures

### Human Configuration (`config/human_config.json`)

All parameters that humans should configure:

```json
{
  "cape_config": {
    "deployment_type": "hybrid",
    "optimization_goals": [
      {"metric": "latency", "weight": 0.35, "minimize": true},
      {"metric": "compute_cost", "weight": 0.25, "minimize": true}
    ],
    "constraints": [
      {"type": "sla_deadline", "value": "5000ms", "is_hard": true}
    ],
    "learning_parameters": {
      "data_gravity_factor": 0.7,
      "exploration_factor": 0.2,
      "learning_rate": 0.001
    }
  },
  "orchestrator_config": {
    "executor_name": "cape-orchestrator-001",
    "behavior": {
      "assign_interval_seconds": 5,
      "max_concurrent_processes": 5,
      "decision_timeout_seconds": 10
    },
    "supported_functions": ["echo", "compute", "ml-inference"]
  }
}
```

### ColonyOS Metrics (`config/colonyos_metrics.json`)

Real system data in ColonyOS Prometheus format:

```json
{
  "timestamp": "2024-09-03T14:30:00Z",
  "colony_statistics": {
    "colonies": 1,
    "executors": 12,
    "processes": {"waiting": 8, "running": 5}
  },
  "prometheus_metrics": {
    "colonies_server_executors": 12.0,
    "colonies_server_processes_waiting": 8.0
  },
  "executor_summary": [
    {
      "id": "executor-edge-stockholm-01",
      "type": "edge", 
      "capabilities": {
        "hardware": {"cpu": "4000m", "memory": "16Gi"}
      }
    }
  ]
}
```

##  System Components

### 1. Native ColonyOS Types (`pkg/colonyos/native_types.go`)

Exact data structures from ColonyOS source code:

```go
// From github.com/colonyos/colonies/pkg/core/executor.go
type Executor struct {
    ID                string        `json:"executorid"`
    Type              string        `json:"executortype"`
    Name              string        `json:"executorname"`
    ColonyName        string        `json:"colonyname"`
    State             int           `json:"state"`
    CommissionTime    time.Time     `json:"commissiontime"`
    LastHeardFromTime time.Time     `json:"lastheardfromtime"`
    Location          Location      `json:"location"`
    Capabilities      Capabilities  `json:"capabilities"`
}

// From github.com/colonyos/colonies/pkg/core/process.go  
type Process struct {
    ID                 string       `json:"processid"`
    State              int          `json:"state"`
    AssignedExecutorID string       `json:"assignedexecutorid"`
    FunctionSpec       FunctionSpec `json:"functionspec"`
    SubmissionTime     time.Time    `json:"submissiontime"`
    StartTime          time.Time    `json:"starttime"`
    EndTime            time.Time    `json:"endtime"`
}

// From github.com/colonyos/colonies/pkg/core/function_spec.go
type FunctionSpec struct {
    NodeName     string                 `json:"nodename"`
    FuncName     string                 `json:"funcname"`  
    Args         []interface{}          `json:"args"`
    KwArgs       map[string]interface{} `json:"kwargs"`
    Priority     int                    `json:"priority"`
    MaxWaitTime  int                    `json:"maxwaittime"`
    MaxExecTime  int                    `json:"maxexectime"`
    Conditions   Conditions             `json:"conditions"`
    Env          map[string]string      `json:"env"`
}
```

### 2. Configuration Loader (`pkg/colonyos/config_loader.go`)

Handles all configuration loading and validation:

```go
type ConfigLoader struct {
    humanConfigPath   string
    metricsDataPath   string
    resourceParser    *ResourceParser
}

// Load and validate human configuration
func (cl *ConfigLoader) LoadHumanConfig() (*HumanConfig, error)

// Load ColonyOS metrics in Prometheus format
func (cl *ConfigLoader) LoadColonyOSMetrics() (*ColonyOSMetrics, error)

// Convert to CAPE format for algorithm
func (cl *ConfigLoader) ConvertToDeploymentConfig(*HumanConfig) (*models.DeploymentConfig, error)

// Get native executors from metrics
func (cl *ConfigLoader) GetActiveExecutors(*ColonyOSMetrics) ([]Executor, error)
```

### 3. Zero Hardcode Orchestrator (`pkg/colonyos/zero_hardcode_orchestrator.go`)

Main orchestrator with **zero hardcoded values**:

```go
type ZeroHardcodeOrchestrator struct {
    // Configuration sources (NO hardcoded values)
    configLoader     *ConfigLoader
    humanConfig      *HumanConfig
    currentMetrics   *ColonyOSMetrics
    
    // Runtime behavior from config
    assignInterval        time.Duration  // From config
    metricsUpdateInterval time.Duration  // From config  
    decisionTimeout       time.Duration  // From config
}

// All intervals come from configuration
func (o *ZeroHardcodeOrchestrator) processAssignmentLoop() {
    ticker := time.NewTicker(o.assignInterval) // From config file!
    maxConcurrent := o.humanConfig.OrchestratorConfig.Behavior.MaxConcurrentProcesses
}

// All timeouts come from configuration  
func (o *ZeroHardcodeOrchestrator) executeProcessWithCAPE() {
    ctx, cancel := context.WithTimeout(ctx, o.decisionTimeout) // From config file!
}
```

##  Data Flow

### 1. Startup Phase
```
1. Load config/human_config.json → Validate → Parse behavior parameters
2. Load config/colonyos_metrics.json → Validate → Extract system state
3. Convert human config → CAPE DeploymentConfig
4. Initialize CAPE algorithm with config-driven parameters
5. Start orchestrator loops with config-driven intervals
```

### 2. Runtime Phase  
```
1. Process Assignment Loop (interval from config)
   ├─ Get processes from metrics file
   ├─ Check against supported_functions list (from config) 
   ├─ Run CAPE decision with config-driven optimization goals
   └─ Execute with timeout from config

2. Metrics Update Loop (interval from config)
   ├─ Reload colonyos_metrics.json
   ├─ Validate data freshness 
   ├─ Update executor and process lists
   └─ Convert to CAPE format

3. Adaptation Loop (interval from config)
   ├─ Check minimum decisions threshold (from config)
   ├─ Run CAPE learning with learning_rate (from config)
   └─ Adapt based on data_gravity_factor (from config)
```

##  Configuration Examples

### Low Latency Edge Deployment
```json
{
  "cape_config": {
    "deployment_type": "edge",
    "optimization_goals": [
      {"metric": "latency", "weight": 0.6, "minimize": true},
      {"metric": "energy_efficiency", "weight": 0.4, "minimize": false}
    ],
    "learning_parameters": {
      "data_gravity_factor": 0.2,
      "exploration_factor": 0.1
    }
  },
  "orchestrator_config": {
    "behavior": {
      "assign_interval_seconds": 1,
      "decision_timeout_seconds": 5
    }
  }
}
```

### Cost-Optimized Cloud Deployment
```json
{
  "cape_config": {
    "deployment_type": "cloud", 
    "optimization_goals": [
      {"metric": "compute_cost", "weight": 0.5, "minimize": true},
      {"metric": "data_movement", "weight": 0.3, "minimize": true},
      {"metric": "throughput", "weight": 0.2, "minimize": false}
    ],
    "learning_parameters": {
      "data_gravity_factor": 0.9,
      "exploration_factor": 0.3
    }
  },
  "orchestrator_config": {
    "behavior": {
      "assign_interval_seconds": 10,
      "max_concurrent_processes": 20
    }
  }
}
```

##  Benefits of Zero Hardcode Approach

### 1. **Operational Flexibility**
- Change behavior without recompilation
- Different configs for dev/staging/production
- A/B testing of optimization strategies
- Runtime configuration updates

### 2. **ColonyOS Native Compatibility**  
- Uses exact ColonyOS data structures
- Follows ColonyOS parsing conventions
- Compatible with ColonyOS APIs
- Ready for real deployment

### 3. **Configuration Validation**
- Validates optimization goal weights sum to 1.0
- Checks constraint value formats
- Validates resource specifications
- Ensures metric data freshness

### 4. **Audit and Monitoring**
- Configuration change tracking
- Decision audit trails (configurable)
- Performance metrics collection
- System health monitoring

### 5. **Production Ready**
- No magic numbers in code
- Clear separation of concerns  
- Comprehensive error handling
- Real-time metrics integration

##  Usage Example

```bash
# Run with configuration files
go run examples/zero_hardcode_example.go

# System loads:
# 1. config/human_config.json     → All behavior parameters
# 2. config/colonyos_metrics.json → Real system data  
# 3. Native ColonyOS structures   → Exact compatibility

# Output shows:
#  All behavior: config/human_config.json
#  All data: config/colonyos_metrics.json  
#  Native ColonyOS formats throughout
```

##  Configuration Management

### Deployment Scenarios
1. **Development**: Quick iteration configs with debug logging
2. **Staging**: Production-like configs with extended monitoring  
3. **Production**: Optimized configs with audit trails enabled
4. **A/B Testing**: Different optimization strategies

### Configuration Validation
- Automated config file validation on startup
- Runtime metric data freshness checks
- Optimization goal weight normalization
- Resource specification parsing verification

This architecture ensures the CAPE system is completely configuration-driven, uses native ColonyOS formats, and contains **zero hardcoded behavioral values**.