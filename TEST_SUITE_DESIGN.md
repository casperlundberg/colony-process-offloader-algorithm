# Comprehensive Test Suite Design for Adaptive Offloading Algorithm

## Overview

This document defines the complete test suite architecture for the adaptive multi-objective offloading algorithm. It provides the foundation for test-driven development and ensures comprehensive validation of all algorithm components.

## 1. Test Architecture

### 1.1 Test Package Structure

```
tests/
├── unit/                    # Unit tests for individual components
│   ├── decision/           # Decision engine tests
│   ├── learning/           # Learning component tests  
│   ├── policy/             # Policy engine tests
│   ├── models/             # Data model tests
│   └── utils/              # Utility function tests
├── integration/            # Integration tests
│   ├── colonyos/          # ColonyOS integration
│   ├── config/            # Configuration management
│   └── monitoring/        # Monitoring integration
├── performance/           # Performance and benchmark tests
│   ├── latency/           # Decision latency tests
│   ├── throughput/        # Throughput tests
│   └── memory/            # Memory usage tests
├── scenario/              # End-to-end scenario tests
│   ├── learning/          # Learning behavior scenarios
│   ├── failure/           # Failure handling scenarios
│   └── policy/            # Policy compliance scenarios
├── fixtures/              # Test data and fixtures
│   ├── processes/         # Sample process definitions
│   ├── targets/           # Sample target configurations
│   ├── states/            # Sample system states
│   └── outcomes/          # Sample execution outcomes
└── mocks/                 # Mock implementations
    ├── colonyos/          # ColonyOS API mocks
    ├── network/           # Network service mocks
    └── storage/           # Storage service mocks
```

### 1.2 Test Framework Configuration

```go
// test_config.go - Global test configuration
package tests

import (
    "testing"
    "time"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
)

// BaseTestSuite provides common test utilities
type BaseTestSuite struct {
    suite.Suite
    
    // Test timing
    TestTimeout    time.Duration
    StartTime      time.Time
    
    // Mock services
    MockColony     *MockColonyServer
    MockNetwork    *MockNetworkService
    MockStorage    *MockStorageService
    
    // Test data
    TestProcesses  []Process
    TestTargets    []OffloadTarget
    TestStates     []SystemState
    TestOutcomes   []OffloadOutcome
}

// SetupSuite initializes test suite
func (suite *BaseTestSuite) SetupSuite() {
    suite.TestTimeout = 30 * time.Second
    
    // Initialize mocks
    suite.MockColony = NewMockColonyServer()
    suite.MockNetwork = NewMockNetworkService()
    suite.MockStorage = NewMockStorageService()
    
    // Load test data
    suite.loadTestFixtures()
}

// TearDownSuite cleans up after all tests
func (suite *BaseTestSuite) TearDownSuite() {
    suite.MockColony.Close()
    suite.MockNetwork.Close() 
    suite.MockStorage.Close()
}

// SetupTest runs before each test
func (suite *BaseTestSuite) SetupTest() {
    suite.StartTime = time.Now()
    
    // Reset mocks to clean state
    suite.MockColony.Reset()
    suite.MockNetwork.Reset()
    suite.MockStorage.Reset()
}

// TearDownTest runs after each test
func (suite *BaseTestSuite) TearDownTest() {
    duration := time.Since(suite.StartTime)
    if duration > suite.TestTimeout {
        suite.T().Errorf("Test exceeded timeout: %v > %v", duration, suite.TestTimeout)
    }
}
```

## 2. Unit Test Specifications

### 2.1 Decision Engine Tests

```go
// decision_engine_test.go
package decision

import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type DecisionEngineTestSuite struct {
    BaseTestSuite
    engine *DecisionEngine
}

func (suite *DecisionEngineTestSuite) SetupTest() {
    suite.BaseTestSuite.SetupTest()
    suite.engine = NewDecisionEngine(DefaultWeights())
}

// Test score computation for various scenarios
func (suite *DecisionEngineTestSuite) TestScoreComputation() {
    testCases := []struct {
        name           string
        process        Process
        localState     SystemState
        target         OffloadTarget
        expectedScore  float64
        tolerance      float64
    }{
        {
            name: "HighQueuePressure_FastTarget",
            process: Process{
                ID: "test-1",
                CPURequirement: 2.0,
                InputSize: 1024,
                OutputSize: 512,
            },
            localState: SystemState{
                QueueDepth: 50,
                QueueThreshold: 20,
                ComputeUsage: 0.9,
            },
            target: OffloadTarget{
                ID: "fast-edge",
                AvailableCapacity: 8.0,
                NetworkLatency: 10 * time.Millisecond,
                NetworkBandwidth: 1000000, // 1MB/s
                Reliability: 0.95,
            },
            expectedScore: 0.8, // High score due to queue pressure relief
            tolerance: 0.1,
        },
        {
            name: "LowQueuePressure_SlowTarget",
            process: Process{
                ID: "test-2", 
                CPURequirement: 1.0,
                InputSize: 10485760, // 10MB
                OutputSize: 5242880,  // 5MB
            },
            localState: SystemState{
                QueueDepth: 2,
                QueueThreshold: 20,
                ComputeUsage: 0.3,
            },
            target: OffloadTarget{
                ID: "slow-cloud",
                AvailableCapacity: 2.0,
                NetworkLatency: 200 * time.Millisecond,
                NetworkBandwidth: 10000, // 10KB/s (very slow)
                Reliability: 0.99,
            },
            expectedScore: 0.2, // Low score due to high network cost
            tolerance: 0.1,
        },
    }
    
    for _, tc := range testCases {
        suite.Run(tc.name, func() {
            score := suite.engine.computeScore(tc.process, tc.localState, tc.target)
            suite.InDelta(tc.expectedScore, score, tc.tolerance, 
                "Score computation failed for %s", tc.name)
            
            // Verify score is in valid range
            suite.GreaterOrEqual(score, 0.0, "Score must be non-negative")
            suite.LessOrEqual(score, 1.0, "Score must not exceed 1.0")
        })
    }
}

// Test pattern application
func (suite *DecisionEngineTestSuite) TestPatternApplication() {
    // Create a validated pattern
    pattern := &DiscoveredPattern{
        ID: "high-queue-pattern",
        Conditions: []PatternCondition{
            {Field: "QueueDepth", Operator: GT, Value: 30},
            {Field: "ComputeUsage", Operator: GT, Value: 0.8},
        },
        RecommendedAction: OFFLOAD_TO,
        PreferredTargets: []string{"edge"},
        SuccessRate: 0.9,
        Confidence: 0.85,
    }
    
    suite.engine.patterns = []*DiscoveredPattern{pattern}
    
    // Test scenario that matches pattern
    process := Process{ID: "test-pattern"}
    state := SystemState{
        QueueDepth: 40,
        ComputeUsage: 0.85,
    }
    targets := []OffloadTarget{
        {ID: "edge-1", Type: "edge"},
        {ID: "cloud-1", Type: "public_cloud"},
    }
    
    decision := suite.engine.ComputeOffloadDecision(process, state, targets)
    
    // Verify pattern was applied
    suite.True(decision.ShouldOffload, "Pattern should recommend offload")
    suite.Equal("edge-1", decision.Target.ID, "Should prefer edge target")
    suite.Equal(pattern.ID, decision.AppliedPattern.ID, "Should record applied pattern")
    suite.Greater(decision.Confidence, 0.8, "Pattern-based decision should have high confidence")
}

// Test policy filtering
func (suite *DecisionEngineTestSuite) TestPolicyFiltering() {
    // Test data sovereignty constraint
    process := Process{
        ID: "sensitive-data",
        DataSensitivity: 5, // Highly sensitive
    }
    
    targets := []OffloadTarget{
        {ID: "local-edge", Type: "edge", DataJurisdiction: "local"},
        {ID: "foreign-cloud", Type: "public_cloud", DataJurisdiction: "foreign"},
        {ID: "private-cloud", Type: "private_cloud", DataJurisdiction: "local"},
    }
    
    // Apply hard policy: sensitive data stays local jurisdiction
    policyEngine := NewPolicyEngine()
    policyEngine.AddRule(PolicyRule{
        Type: HARD,
        Condition: func(p Process, t OffloadTarget) bool {
            if p.DataSensitivity >= 4 {
                return t.DataJurisdiction == "local"
            }
            return true
        },
        Description: "Data sovereignty",
    })
    
    filtered := policyEngine.FilterTargets(process, targets)
    
    suite.Len(filtered, 2, "Should filter out foreign targets")
    
    // Verify only local jurisdiction targets remain
    for _, target := range filtered {
        suite.Equal("local", target.DataJurisdiction, 
            "Filtered targets should be in local jurisdiction")
    }
}

// Test edge cases and error conditions
func (suite *DecisionEngineTestSuite) TestEdgeCases() {
    // Empty targets list
    decision := suite.engine.ComputeOffloadDecision(
        Process{ID: "test"}, 
        SystemState{}, 
        []OffloadTarget{},
    )
    suite.False(decision.ShouldOffload, "Should not offload with no targets")
    suite.Nil(decision.Target, "Target should be nil with no options")
    
    // Invalid process data
    invalidProcess := Process{
        ID: "",  // Empty ID
        CPURequirement: -1, // Negative requirement
    }
    
    decision = suite.engine.ComputeOffloadDecision(
        invalidProcess,
        SystemState{},
        []OffloadTarget{{ID: "valid-target"}},
    )
    
    suite.False(decision.ShouldOffload, "Should reject invalid process")
    
    // Extremely high resource requirements
    hugeProcss := Process{
        ID: "huge-process",
        CPURequirement: 1000, // More than any target can handle
        MemoryRequirement: 1024 * 1024 * 1024 * 1024, // 1TB
    }
    
    targets := []OffloadTarget{
        {ID: "small-target", AvailableCapacity: 4, MemoryAvailable: 8 * 1024 * 1024 * 1024}, // 8GB
    }
    
    decision = suite.engine.ComputeOffloadDecision(hugeProcess, SystemState{}, targets)
    suite.False(decision.ShouldOffload, "Should not offload oversized process")
}

func TestDecisionEngineTestSuite(t *testing.T) {
    suite.Run(t, new(DecisionEngineTestSuite))
}
```

### 2.2 Learning Component Tests

```go
// adaptive_learner_test.go
package learning

import (
    "testing"
    "time"
    "github.com/stretchr/testify/suite"
)

type AdaptiveLearnerTestSuite struct {
    BaseTestSuite
    learner *AdaptiveLearner
}

func (suite *AdaptiveLearnerTestSuite) SetupTest() {
    suite.BaseTestSuite.SetupTest()
    suite.learner = NewAdaptiveLearner(LearningConfig{
        WindowSize: 100,
        LearningRate: 0.01,
        ExplorationRate: 0.1,
        MinSamples: 10,
    })
}

// Test weight adaptation from positive outcomes
func (suite *AdaptiveLearnerTestSuite) TestWeightAdaptationPositive() {
    initialWeights := AdaptiveWeights{
        QueueDepth: 0.2,
        ProcessorLoad: 0.2,
        NetworkCost: 0.2,
        LatencyCost: 0.2,
        EnergyCost: 0.1,
        PolicyCost: 0.1,
    }
    
    // Create positive outcome strongly attributed to queue management
    outcome := OffloadOutcome{
        DecisionID: "test-decision",
        CompletedOnTime: true,
        QueueReduction: 0.8, // Strong queue improvement
        Success: true,
        Reward: 2.0, // High reward
        Attribution: map[string]float64{
            "QueueDepth": 0.7,    // Queue factor dominated
            "ProcessorLoad": 0.1,
            "NetworkCost": 0.1,
            "LatencyCost": 0.1,
        },
    }
    
    // Apply learning update
    suite.learner.UpdateWeights(&initialWeights, outcome)
    
    // Verify queue weight increased
    suite.Greater(initialWeights.QueueDepth, 0.2, 
        "Queue weight should increase after positive outcome")
    
    // Verify weights still sum to 1.0
    total := initialWeights.QueueDepth + initialWeights.ProcessorLoad + 
             initialWeights.NetworkCost + initialWeights.LatencyCost + 
             initialWeights.EnergyCost + initialWeights.PolicyCost
    suite.InDelta(1.0, total, 0.001, "Weights must sum to 1.0")
    
    // Verify no weight goes negative or above max
    suite.GreaterOrEqual(initialWeights.QueueDepth, 0.0)
    suite.GreaterOrEqual(initialWeights.ProcessorLoad, 0.0)
    suite.LessOrEqual(initialWeights.QueueDepth, 0.5) // Max weight constraint
}

// Test weight adaptation from negative outcomes
func (suite *AdaptiveLearnerTestSuite) TestWeightAdaptationNegative() {
    initialWeights := AdaptiveWeights{
        QueueDepth: 0.2,
        ProcessorLoad: 0.2,
        NetworkCost: 0.2,
        LatencyCost: 0.2,
        EnergyCost: 0.1,
        PolicyCost: 0.1,
    }
    
    // Create negative outcome attributed to poor network decision
    outcome := OffloadOutcome{
        DecisionID: "test-bad-decision",
        CompletedOnTime: false,
        Success: false,
        NetworkCongestion: true,
        LocalWorkDelayed: true,
        Reward: -1.5, // Negative reward
        Attribution: map[string]float64{
            "NetworkCost": 0.8, // Network decision was primary factor
            "QueueDepth": 0.1,
            "ProcessorLoad": 0.1,
        },
    }
    
    suite.learner.UpdateWeights(&initialWeights, outcome)
    
    // Verify network weight decreased
    suite.Less(initialWeights.NetworkCost, 0.2,
        "Network weight should decrease after negative outcome")
    
    // Verify compensation in other weights
    total := initialWeights.QueueDepth + initialWeights.ProcessorLoad + 
             initialWeights.NetworkCost + initialWeights.LatencyCost + 
             initialWeights.EnergyCost + initialWeights.PolicyCost
    suite.InDelta(1.0, total, 0.001, "Weights must sum to 1.0")
}

// Test pattern discovery
func (suite *AdaptiveLearnerTestSuite) TestPatternDiscovery() {
    // Create consistent history: high queue + high CPU -> offload to edge works well
    outcomes := []OffloadOutcome{}
    
    for i := 0; i < 15; i++ { // Above minimum sample threshold
        outcome := OffloadOutcome{
            DecisionID: fmt.Sprintf("decision-%d", i),
            ProcessID: fmt.Sprintf("process-%d", i),
            TargetID: "edge-server",
            Success: true,
            CompletedOnTime: true,
            QueueReduction: 0.7,
            Reward: 1.5,
            
            // Context that should become pattern signature
            SystemContext: SystemState{
                QueueDepth: 35 + i, // Consistently high
                ComputeUsage: 0.85 + float64(i)*0.01, // Consistently high
            },
            ProcessContext: Process{
                CPURequirement: 2.0,
            },
            TargetContext: OffloadTarget{
                Type: "edge",
            },
        }
        outcomes = append(outcomes, outcome)
        suite.learner.history = append(suite.learner.history, outcome)
    }
    
    // Trigger pattern discovery
    suite.learner.discoverPatterns()
    
    // Verify pattern was discovered
    suite.NotEmpty(suite.learner.patterns, "Should discover patterns from consistent data")
    
    pattern := suite.learner.patterns[0]
    suite.Greater(pattern.SuccessRate, 0.8, "Pattern should have high success rate")
    suite.Greater(pattern.Confidence, 0.7, "Pattern should have reasonable confidence")
    suite.Equal("OFFLOAD_TO", string(pattern.RecommendedAction))
    
    // Verify pattern conditions
    foundQueueCondition := false
    foundCPUCondition := false
    for _, condition := range pattern.Conditions {
        if condition.Field == "QueueDepth" && condition.Operator == GT {
            foundQueueCondition = true
        }
        if condition.Field == "ComputeUsage" && condition.Operator == GT {
            foundCPUCondition = true
        }
    }
    suite.True(foundQueueCondition, "Pattern should include queue condition")
    suite.True(foundCPUCondition, "Pattern should include CPU condition")
}

// Test convergence behavior
func (suite *AdaptiveLearnerTestSuite) TestConvergence() {
    weights := AdaptiveWeights{
        QueueDepth: 0.5,    // Start far from optimal
        ProcessorLoad: 0.1, 
        NetworkCost: 0.1,
        LatencyCost: 0.1,
        EnergyCost: 0.1,
        PolicyCost: 0.1,
    }
    
    // Simulate learning with consistent feedback that queue is less important
    for i := 0; i < 200; i++ {
        outcome := OffloadOutcome{
            DecisionID: fmt.Sprintf("convergence-%d", i),
            Success: true,
            Reward: 1.0,
            Attribution: map[string]float64{
                "QueueDepth": 0.1,      // Queue not important
                "ProcessorLoad": 0.4,   // Load balancing important
                "NetworkCost": 0.3,     // Network important
                "LatencyCost": 0.2,     // Latency somewhat important
            },
        }
        
        suite.learner.UpdateWeights(&weights, outcome)
    }
    
    // Verify convergence toward optimal weights
    suite.Less(weights.QueueDepth, 0.3, "Queue weight should decrease")
    suite.Greater(weights.ProcessorLoad, 0.2, "Load weight should increase")
    suite.Greater(weights.NetworkCost, 0.2, "Network weight should increase")
    
    // Test weight stability (small changes after convergence)
    previousWeights := weights
    for i := 0; i < 20; i++ {
        outcome := OffloadOutcome{
            DecisionID: fmt.Sprintf("stable-%d", i),
            Success: true,
            Reward: 1.0,
            Attribution: map[string]float64{
                "QueueDepth": 0.1,
                "ProcessorLoad": 0.4,
                "NetworkCost": 0.3,
                "LatencyCost": 0.2,
            },
        }
        suite.learner.UpdateWeights(&weights, outcome)
    }
    
    // Verify small changes (convergence)
    queueChange := math.Abs(weights.QueueDepth - previousWeights.QueueDepth)
    suite.Less(queueChange, 0.05, "Weights should be stable after convergence")
}

func TestAdaptiveLearnerTestSuite(t *testing.T) {
    suite.Run(t, new(AdaptiveLearnerTestSuite))
}
```

### 2.3 Policy Engine Tests

```go
// policy_engine_test.go  
package policy

import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type PolicyEngineTestSuite struct {
    BaseTestSuite
    policyEngine *PolicyEngine
}

func (suite *PolicyEngineTestSuite) SetupTest() {
    suite.BaseTestSuite.SetupTest()
    suite.policyEngine = NewPolicyEngine()
}

// Test hard constraint enforcement
func (suite *PolicyEngineTestSuite) TestHardConstraints() {
    // Add safety-critical constraint
    suite.policyEngine.AddRule(PolicyRule{
        Type: HARD,
        Condition: func(p Process, t OffloadTarget) bool {
            return !(p.SafetyCritical && t.Type != "local")
        },
        Priority: 1,
        Description: "Safety-critical processes must stay local",
    })
    
    // Add data sensitivity constraint
    suite.policyEngine.AddRule(PolicyRule{
        Type: HARD,
        Condition: func(p Process, t OffloadTarget) bool {
            if p.DataSensitivity >= 4 {
                return t.DataJurisdiction == "domestic"
            }
            return true
        },
        Priority: 1,
        Description: "Sensitive data stays domestic",
    })
    
    // Test safety-critical process
    safetyCriticalProcess := Process{
        ID: "safety-critical-1",
        SafetyCritical: true,
    }
    
    targets := []OffloadTarget{
        {ID: "local", Type: "local"},
        {ID: "edge", Type: "edge"},
        {ID: "cloud", Type: "public_cloud"},
    }
    
    filtered := suite.policyEngine.FilterTargets(safetyCriticalProcess, targets)
    
    suite.Len(filtered, 1, "Should filter to only local target")
    suite.Equal("local", filtered[0].ID, "Should keep only local target")
    
    // Test sensitive data process
    sensitiveProcess := Process{
        ID: "sensitive-data",
        DataSensitivity: 5,
    }
    
    targets = []OffloadTarget{
        {ID: "domestic-cloud", Type: "private_cloud", DataJurisdiction: "domestic"},
        {ID: "foreign-cloud", Type: "public_cloud", DataJurisdiction: "foreign"},
        {ID: "local-edge", Type: "edge", DataJurisdiction: "domestic"},
    }
    
    filtered = suite.policyEngine.FilterTargets(sensitiveProcess, targets)
    
    suite.Len(filtered, 2, "Should filter out foreign targets")
    for _, target := range filtered {
        suite.Equal("domestic", target.DataJurisdiction, 
            "All targets should be domestic")
    }
}

// Test soft constraint scoring
func (suite *PolicyEngineTestSuite) TestSoftConstraints() {
    // Add preference for green energy
    suite.policyEngine.AddRule(PolicyRule{
        Type: SOFT,
        Condition: func(p Process, t OffloadTarget) bool {
            return t.EnergySource == "renewable"
        },
        Priority: 2,
        Description: "Prefer renewable energy sources",
    })
    
    // Add preference for low latency
    suite.policyEngine.AddRule(PolicyRule{
        Type: SOFT,
        Condition: func(p Process, t OffloadTarget) bool {
            if p.RealTime {
                return t.NetworkLatency <= 50*time.Millisecond
            }
            return true
        },
        Priority: 3,
        Description: "Real-time processes prefer low latency",
    })
    
    realTimeProcess := Process{
        ID: "real-time-process",
        RealTime: true,
    }
    
    targets := []OffloadTarget{
        {
            ID: "fast-green",
            EnergySource: "renewable",
            NetworkLatency: 10 * time.Millisecond,
        },
        {
            ID: "fast-dirty", 
            EnergySource: "fossil",
            NetworkLatency: 20 * time.Millisecond,
        },
        {
            ID: "slow-green",
            EnergySource: "renewable", 
            NetworkLatency: 100 * time.Millisecond,
        },
    }
    
    filtered := suite.policyEngine.FilterTargets(realTimeProcess, targets)
    
    // All targets should remain (soft constraints don't filter)
    suite.Len(filtered, 3, "Soft constraints should not filter targets")
    
    // Check policy bonus scores
    for _, target := range filtered {
        switch target.ID {
        case "fast-green":
            suite.Equal(0.2, target.PolicyBonus, "Should get bonus for both preferences")
        case "fast-dirty":
            suite.Equal(0.1, target.PolicyBonus, "Should get bonus for latency only")
        case "slow-green":
            suite.Equal(0.0, target.PolicyBonus, "Should get no net bonus")
        }
    }
}

// Test policy priority handling
func (suite *PolicyEngineTestSuite) TestPolicyPriorities() {
    // High priority security rule
    suite.policyEngine.AddRule(PolicyRule{
        Type: HARD,
        Priority: 1, // Highest
        Condition: func(p Process, t OffloadTarget) bool {
            return t.SecurityLevel >= p.SecurityLevel
        },
        Description: "Security level must be sufficient",
    })
    
    // Lower priority cost rule
    suite.policyEngine.AddRule(PolicyRule{
        Type: SOFT,
        Priority: 5, // Lower
        Condition: func(p Process, t OffloadTarget) bool {
            return t.ComputeCost < 0.10 // Prefer cheap targets
        },
        Description: "Prefer cost-effective targets",
    })
    
    highSecurityProcess := Process{
        ID: "classified-work",
        SecurityLevel: 4,
    }
    
    targets := []OffloadTarget{
        {
            ID: "secure-expensive",
            SecurityLevel: 5,
            ComputeCost: 0.50, // Expensive
        },
        {
            ID: "insecure-cheap", 
            SecurityLevel: 2, // Too low
            ComputeCost: 0.05, // Cheap
        },
    }
    
    filtered := suite.policyEngine.FilterTargets(highSecurityProcess, targets)
    
    // High priority security rule should filter out insecure target
    suite.Len(filtered, 1, "Security constraint should filter targets")
    suite.Equal("secure-expensive", filtered[0].ID, 
        "Should keep secure target despite cost")
}

func TestPolicyEngineTestSuite(t *testing.T) {
    suite.Run(t, new(PolicyEngineTestSuite))
}
```

## 3. Integration Test Specifications

### 3.1 ColonyOS Integration Tests

```go
// colonyos_integration_test.go
package integration

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/suite"
)

type ColonyOSIntegrationTestSuite struct {
    BaseTestSuite
    offloader     *AdaptiveOffloader
    testColony    *TestColonyServer
}

func (suite *ColonyOSIntegrationTestSuite) SetupSuite() {
    suite.BaseTestSuite.SetupSuite()
    
    // Start test ColonyOS instance
    suite.testColony = StartTestColonyServer()
    
    // Initialize offloader with test colony
    config := DefaultOffloadConfig()
    suite.offloader = NewAdaptiveOffloader(suite.testColony, config)
}

func (suite *ColonyOSIntegrationTestSuite) TearDownSuite() {
    suite.offloader.Stop()
    suite.testColony.Stop()
    suite.BaseTestSuite.TearDownSuite()
}

// Test complete offloading workflow
func (suite *ColonyOSIntegrationTestSuite) TestCompleteOffloadingWorkflow() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Start offloader
    go suite.offloader.Run(ctx)
    
    // Submit test process
    testProcess := Process{
        ID: "integration-test-1",
        CPURequirement: 2.0,
        EstimatedDuration: 10 * time.Second,
        MaxDuration: 30 * time.Second,
    }
    
    processID, err := suite.testColony.SubmitProcess(testProcess)
    suite.NoError(err, "Process submission should succeed")
    
    // Register test target
    testTarget := OffloadTarget{
        ID: "test-edge-server",
        Type: "edge",
        AvailableCapacity: 8.0,
        NetworkLatency: 20 * time.Millisecond,
        NetworkBandwidth: 100000000, // 100MB/s
        Reliability: 0.95,
    }
    
    suite.testColony.RegisterTarget(testTarget)
    
    // Wait for system to reach offloading threshold
    suite.testColony.SimulateLoad(0.85) // High load to trigger offloading
    
    // Wait for offloading decision
    decision, err := suite.waitForOffloadDecision(processID, 10*time.Second)
    suite.NoError(err, "Should make offload decision")
    suite.True(decision.ShouldOffload, "Should decide to offload")
    suite.Equal("test-edge-server", decision.Target.ID, "Should select test target")
    
    // Wait for execution completion
    outcome, err := suite.waitForExecutionOutcome(processID, 20*time.Second)
    suite.NoError(err, "Execution should complete")
    suite.True(outcome.Success, "Execution should succeed")
    suite.True(outcome.CompletedOnTime, "Should meet deadline")
    
    // Verify learning update
    updatedWeights := suite.offloader.GetCurrentWeights()
    suite.NotEqual(DefaultWeights(), updatedWeights, "Weights should be updated")
}

// Test failure handling
func (suite *ColonyOSIntegrationTestSuite) TestFailureHandling() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    go suite.offloader.Run(ctx)
    
    // Submit process
    testProcess := Process{
        ID: "failure-test",
        CPURequirement: 1.0,
        EstimatedDuration: 5 * time.Second,
        MaxDuration: 15 * time.Second,
    }
    
    processID, err := suite.testColony.SubmitProcess(testProcess)
    suite.NoError(err)
    
    // Register unreliable target
    unreliableTarget := OffloadTarget{
        ID: "unreliable-target",
        Type: "edge",
        AvailableCapacity: 4.0,
        Reliability: 0.3, // Very unreliable
    }
    
    suite.testColony.RegisterTarget(unreliableTarget)
    suite.testColony.SimulateTargetFailure("unreliable-target")
    
    // Trigger offloading conditions
    suite.testColony.SimulateLoad(0.9)
    
    // Wait for decision and execution
    decision, err := suite.waitForOffloadDecision(processID, 10*time.Second)
    suite.NoError(err)
    
    if decision.ShouldOffload {
        // If it decided to offload, it should handle failure gracefully
        outcome, err := suite.waitForExecutionOutcome(processID, 20*time.Second)
        suite.NoError(err)
        
        // Should either succeed with retry or fall back to local
        suite.True(outcome.Success || outcome.LocalFallback,
            "Should succeed or fall back to local")
    } else {
        // If it decided not to offload, should execute locally
        outcome, err := suite.waitForExecutionOutcome(processID, 20*time.Second)
        suite.NoError(err)
        suite.True(outcome.Success, "Local execution should succeed")
    }
    
    // Verify system remains healthy
    health := suite.offloader.GetHealth()
    suite.NotEqual(CRITICAL, health.Overall, "System should not be critical")
}

// Test policy compliance
func (suite *ColonyOSIntegrationTestSuite) TestPolicyCompliance() {
    // Configure strict policies
    policies := []PolicyRule{
        {
            Type: HARD,
            Condition: func(p Process, t OffloadTarget) bool {
                return !(p.DataSensitivity >= 4 && t.Type == "public_cloud")
            },
            Description: "Sensitive data policy",
        },
    }
    
    suite.offloader.policyEngine.SetRules(policies)
    
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()
    
    go suite.offloader.Run(ctx)
    
    // Submit sensitive process
    sensitiveProcess := Process{
        ID: "sensitive-process",
        DataSensitivity: 5,
        CPURequirement: 2.0,
    }
    
    processID, err := suite.testColony.SubmitProcess(sensitiveProcess)
    suite.NoError(err)
    
    // Register targets including policy-violating one
    targets := []OffloadTarget{
        {ID: "local-edge", Type: "edge", DataJurisdiction: "local"},
        {ID: "public-cloud", Type: "public_cloud", DataJurisdiction: "foreign"},
    }
    
    for _, target := range targets {
        suite.testColony.RegisterTarget(target)
    }
    
    // Trigger offloading conditions
    suite.testColony.SimulateLoad(0.9)
    
    // Wait for decision
    decision, err := suite.waitForOffloadDecision(processID, 10*time.Second)
    suite.NoError(err)
    
    if decision.ShouldOffload {
        // If it offloads, should not violate policy
        suite.NotEqual("public-cloud", decision.Target.ID,
            "Should not select policy-violating target")
        suite.Empty(decision.PolicyViolations,
            "Should not have policy violations")
    }
    
    // Wait for completion
    outcome, err := suite.waitForExecutionOutcome(processID, 15*time.Second)
    suite.NoError(err)
    suite.False(outcome.PolicyViolation, "Should not violate policies")
}

func TestColonyOSIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(ColonyOSIntegrationTestSuite))
}
```

## 4. Performance Test Specifications

### 4.1 Latency Tests

```go
// performance_latency_test.go
package performance

import (
    "testing"
    "time"
    "github.com/stretchr/testify/suite"
)

type LatencyTestSuite struct {
    BaseTestSuite
    offloader *AdaptiveOffloader
}

func (suite *LatencyTestSuite) SetupTest() {
    suite.BaseTestSuite.SetupTest()
    config := DefaultOffloadConfig()
    suite.offloader = NewAdaptiveOffloader(suite.MockColony, config)
}

// Test decision latency under normal conditions
func (suite *LatencyTestSuite) TestDecisionLatencyNormal() {
    // Prepare test data
    process := Process{ID: "latency-test", CPURequirement: 2.0}
    state := SystemState{QueueDepth: 10, ComputeUsage: 0.7}
    targets := []OffloadTarget{
        {ID: "target-1", AvailableCapacity: 4.0},
        {ID: "target-2", AvailableCapacity: 8.0},
    }
    
    // Measure decision latency
    latencies := []time.Duration{}
    
    for i := 0; i < 100; i++ {
        start := time.Now()
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, targets)
        latency := time.Since(start)
        
        latencies = append(latencies, latency)
        
        // Verify decision is valid
        suite.NotNil(decision, "Decision should not be nil")
    }
    
    // Calculate statistics
    avgLatency := calculateAverage(latencies)
    p50Latency := calculatePercentile(latencies, 50)
    p95Latency := calculatePercentile(latencies, 95)
    p99Latency := calculatePercentile(latencies, 99)
    
    // Verify performance requirements
    suite.Less(p50Latency, 50*time.Millisecond, 
        "50th percentile should be < 50ms")
    suite.Less(p95Latency, 200*time.Millisecond, 
        "95th percentile should be < 200ms")
    suite.Less(p99Latency, 500*time.Millisecond, 
        "99th percentile should be < 500ms")
    
    // Log performance results
    suite.T().Logf("Decision Latency Results:")
    suite.T().Logf("  Average: %v", avgLatency)
    suite.T().Logf("  50th percentile: %v", p50Latency)
    suite.T().Logf("  95th percentile: %v", p95Latency)
    suite.T().Logf("  99th percentile: %v", p99Latency)
}

// Test decision latency under high load
func (suite *LatencyTestSuite) TestDecisionLatencyHighLoad() {
    // Create high-load scenario
    process := Process{ID: "high-load-test", CPURequirement: 4.0}
    state := SystemState{
        QueueDepth: 100,
        ComputeUsage: 0.95,
        NetworkUsage: 0.8,
    }
    
    // Create many target options
    targets := []OffloadTarget{}
    for i := 0; i < 20; i++ {
        targets = append(targets, OffloadTarget{
            ID: fmt.Sprintf("target-%d", i),
            AvailableCapacity: float64(2 + i%6),
            NetworkLatency: time.Duration(10+i*5) * time.Millisecond,
        })
    }
    
    // Measure latency under load
    latencies := []time.Duration{}
    
    for i := 0; i < 50; i++ {
        start := time.Now()
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, targets)
        latency := time.Since(start)
        
        latencies = append(latencies, latency)
        suite.NotNil(decision)
    }
    
    p95Latency := calculatePercentile(latencies, 95)
    
    // High-load latency should still be acceptable
    suite.Less(p95Latency, 500*time.Millisecond,
        "95th percentile under high load should be < 500ms")
}

// Test concurrent decision latency
func (suite *LatencyTestSuite) TestConcurrentDecisionLatency() {
    concurrency := 10
    decisionsPerWorker := 20
    
    results := make(chan time.Duration, concurrency*decisionsPerWorker)
    
    // Start concurrent workers
    for i := 0; i < concurrency; i++ {
        go func(workerID int) {
            for j := 0; j < decisionsPerWorker; j++ {
                process := Process{
                    ID: fmt.Sprintf("concurrent-%d-%d", workerID, j),
                    CPURequirement: 2.0,
                }
                state := SystemState{QueueDepth: 15, ComputeUsage: 0.6}
                targets := []OffloadTarget{
                    {ID: "target-1", AvailableCapacity: 4.0},
                    {ID: "target-2", AvailableCapacity: 6.0},
                }
                
                start := time.Now()
                decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
                    process, state, targets)
                latency := time.Since(start)
                
                results <- latency
                suite.NotNil(decision)
            }
        }(i)
    }
    
    // Collect results
    latencies := []time.Duration{}
    for i := 0; i < concurrency*decisionsPerWorker; i++ {
        latency := <-results
        latencies = append(latencies, latency)
    }
    
    p95Latency := calculatePercentile(latencies, 95)
    
    // Concurrent performance should not degrade significantly
    suite.Less(p95Latency, 300*time.Millisecond,
        "Concurrent 95th percentile should be < 300ms")
}

func TestLatencyTestSuite(t *testing.T) {
    suite.Run(t, new(LatencyTestSuite))
}
```

### 4.2 Memory Usage Tests

```go
// performance_memory_test.go
package performance

import (
    "runtime"
    "testing"
    "time"
    "github.com/stretchr/testify/suite"
)

type MemoryTestSuite struct {
    BaseTestSuite
    offloader *AdaptiveOffloader
}

func (suite *MemoryTestSuite) SetupTest() {
    suite.BaseTestSuite.SetupTest()
    config := DefaultOffloadConfig()
    suite.offloader = NewAdaptiveOffloader(suite.MockColony, config)
}

// Test memory usage under normal operation
func (suite *MemoryTestSuite) TestMemoryUsageNormal() {
    // Force garbage collection to get baseline
    runtime.GC()
    runtime.GC()
    
    var memBefore runtime.MemStats
    runtime.ReadMemStats(&memBefore)
    
    // Run normal decision-making for extended period
    for i := 0; i < 1000; i++ {
        process := Process{
            ID: fmt.Sprintf("memory-test-%d", i),
            CPURequirement: float64(1 + i%4),
        }
        state := SystemState{
            QueueDepth: 10 + i%20,
            ComputeUsage: 0.5 + float64(i%30)/100,
        }
        targets := []OffloadTarget{
            {ID: "target-1", AvailableCapacity: 4.0},
            {ID: "target-2", AvailableCapacity: 8.0},
        }
        
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, targets)
        suite.NotNil(decision)
        
        // Simulate learning updates
        outcome := OffloadOutcome{
            DecisionID: decision.DecisionID,
            Success: true,
            Reward: 1.0,
            Attribution: map[string]float64{
                "QueueDepth": 0.3,
                "ProcessorLoad": 0.3,
                "NetworkCost": 0.2,
                "LatencyCost": 0.2,
            },
        }
        
        weights := suite.offloader.GetCurrentWeights()
        suite.offloader.learner.UpdateWeights(&weights, outcome)
    }
    
    // Force garbage collection again
    runtime.GC()
    runtime.GC()
    
    var memAfter runtime.MemStats
    runtime.ReadMemStats(&memAfter)
    
    // Calculate memory usage
    memoryUsed := memAfter.Alloc - memBefore.Alloc
    
    // Memory usage should be reasonable
    maxMemoryMB := int64(50) // 50MB limit for normal operation
    memoryUsedMB := int64(memoryUsed) / (1024 * 1024)
    
    suite.LessOrEqual(memoryUsedMB, maxMemoryMB,
        "Memory usage should be < %dMB, used: %dMB", maxMemoryMB, memoryUsedMB)
    
    suite.T().Logf("Memory usage: %d MB", memoryUsedMB)
}

// Test memory usage with learning history
func (suite *MemoryTestSuite) TestMemoryUsageWithLearningHistory() {
    runtime.GC()
    runtime.GC()
    
    var memBefore runtime.MemStats
    runtime.ReadMemStats(&memBefore)
    
    // Fill learning history to maximum
    config := suite.offloader.learner.config
    maxHistory := config.WindowSize
    
    for i := 0; i < maxHistory*2; i++ { // Overfill to test pruning
        outcome := OffloadOutcome{
            DecisionID: fmt.Sprintf("history-test-%d", i),
            ProcessID: fmt.Sprintf("process-%d", i),
            Success: i%10 != 0, // 90% success rate
            CompletedOnTime: i%8 != 0,
            QueueReduction: float64(i%10) / 10.0,
            Reward: float64(i%3) - 1.0, // -1, 0, 1
            Attribution: map[string]float64{
                "QueueDepth": float64(i%10) / 10.0,
                "ProcessorLoad": float64((i+1)%10) / 10.0,
                "NetworkCost": float64((i+2)%10) / 10.0,
                "LatencyCost": float64((i+3)%10) / 10.0,
            },
            SystemContext: SystemState{
                QueueDepth: i % 50,
                ComputeUsage: float64(i%100) / 100.0,
            },
        }
        
        suite.offloader.learner.history = append(suite.offloader.learner.history, outcome)
        
        if len(suite.offloader.learner.history) > maxHistory {
            suite.offloader.learner.history = suite.offloader.learner.history[1:]
        }
    }
    
    // Trigger pattern discovery (memory intensive)
    suite.offloader.learner.discoverPatterns()
    
    runtime.GC()
    runtime.GC()
    
    var memAfter runtime.MemStats
    runtime.ReadMemStats(&memAfter)
    
    memoryUsed := memAfter.Alloc - memBefore.Alloc
    memoryUsedMB := int64(memoryUsed) / (1024 * 1024)
    
    // Should maintain reasonable memory usage even with full history
    maxMemoryWithHistoryMB := int64(100) // 100MB limit with full history
    suite.LessOrEqual(memoryUsedMB, maxMemoryWithHistoryMB,
        "Memory with history should be < %dMB, used: %dMB", 
        maxMemoryWithHistoryMB, memoryUsedMB)
    
    // Verify history size is controlled
    suite.LessOrEqual(len(suite.offloader.learner.history), maxHistory,
        "History size should be limited to %d entries", maxHistory)
    
    suite.T().Logf("Memory with history: %d MB", memoryUsedMB)
    suite.T().Logf("History entries: %d", len(suite.offloader.learner.history))
    suite.T().Logf("Patterns discovered: %d", len(suite.offloader.learner.patterns))
}

// Test for memory leaks
func (suite *MemoryTestSuite) TestMemoryLeaks() {
    runtime.GC()
    runtime.GC()
    
    var memBaseline runtime.MemStats
    runtime.ReadMemStats(&memBaseline)
    
    // Run multiple cycles to detect leaks
    for cycle := 0; cycle < 10; cycle++ {
        // Run decision-making cycle
        for i := 0; i < 100; i++ {
            process := Process{
                ID: fmt.Sprintf("leak-test-%d-%d", cycle, i),
                CPURequirement: 2.0,
            }
            state := SystemState{QueueDepth: 15, ComputeUsage: 0.7}
            targets := []OffloadTarget{
                {ID: "target", AvailableCapacity: 6.0},
            }
            
            decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
                process, state, targets)
            
            // Simulate outcome and learning
            outcome := OffloadOutcome{
                DecisionID: decision.DecisionID,
                Success: true,
                Reward: 1.0,
                Attribution: map[string]float64{"QueueDepth": 1.0},
            }
            
            weights := suite.offloader.GetCurrentWeights()
            suite.offloader.learner.UpdateWeights(&weights, outcome)
        }
        
        // Force cleanup
        runtime.GC()
        runtime.GC()
        
        var memCurrent runtime.MemStats
        runtime.ReadMemStats(&memCurrent)
        
        memoryGrowth := memCurrent.Alloc - memBaseline.Alloc
        memoryGrowthMB := int64(memoryGrowth) / (1024 * 1024)
        
        // Memory should not grow unboundedly
        maxGrowthMB := int64(20) // 20MB maximum growth
        suite.LessOrEqual(memoryGrowthMB, maxGrowthMB,
            "Memory growth in cycle %d should be < %dMB, growth: %dMB",
            cycle, maxGrowthMB, memoryGrowthMB)
    }
}

func TestMemoryTestSuite(t *testing.T) {
    suite.Run(t, new(MemoryTestSuite))
}
```

## 5. Scenario-Based Tests

### 5.1 Learning Behavior Scenarios

```go
// scenario_learning_test.go
package scenario

import (
    "testing"
    "time"
    "github.com/stretchr/testify/suite"
)

type LearningScenarioTestSuite struct {
    BaseTestSuite
    offloader *AdaptiveOffloader
}

func (suite *LearningScenarioTestSuite) SetupTest() {
    suite.BaseTestSuite.SetupTest()
    config := DefaultOffloadConfig()
    suite.offloader = NewAdaptiveOffloader(suite.MockColony, config)
}

// Test adaptation to changing network conditions
func (suite *LearningScenarioTestSuite) TestAdaptationToNetworkChanges() {
    // Phase 1: Fast network conditions
    suite.T().Log("Phase 1: Fast network conditions")
    
    fastTarget := OffloadTarget{
        ID: "fast-target",
        NetworkLatency: 10 * time.Millisecond,
        NetworkBandwidth: 1000000, // 1MB/s
        AvailableCapacity: 8.0,
        NetworkStability: 0.95,
    }
    
    // Run decisions in fast network environment
    for i := 0; i < 50; i++ {
        process := Process{
            ID: fmt.Sprintf("fast-phase-%d", i),
            InputSize: 100000,  // 100KB
            OutputSize: 50000,  // 50KB
            CPURequirement: 2.0,
        }
        
        state := SystemState{
            QueueDepth: 25,
            ComputeUsage: 0.8,
        }
        
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, []OffloadTarget{fastTarget})
        
        // Simulate successful outcomes in fast network
        outcome := OffloadOutcome{
            DecisionID: decision.DecisionID,
            Success: true,
            CompletedOnTime: true,
            QueueReduction: 0.3,
            NetworkCostActual: 0.1, // Low network cost
            Reward: 1.5,
            Attribution: map[string]float64{
                "NetworkCost": 0.4,    // Network decisions work well
                "QueueDepth": 0.3,
                "ProcessorLoad": 0.2,
                "LatencyCost": 0.1,
            },
        }
        
        weights := suite.offloader.GetCurrentWeights()
        suite.offloader.learner.UpdateWeights(&weights, outcome)
    }
    
    weightsAfterFast := suite.offloader.GetCurrentWeights()
    suite.T().Logf("Weights after fast network: NetworkCost=%.3f", 
        weightsAfterFast.NetworkCost)
    
    // Phase 2: Slow network conditions
    suite.T().Log("Phase 2: Slow network conditions")
    
    slowTarget := OffloadTarget{
        ID: "slow-target", 
        NetworkLatency: 200 * time.Millisecond,
        NetworkBandwidth: 10000, // 10KB/s (very slow)
        AvailableCapacity: 8.0,
        NetworkStability: 0.7,
    }
    
    // Run decisions in slow network environment
    for i := 0; i < 50; i++ {
        process := Process{
            ID: fmt.Sprintf("slow-phase-%d", i),
            InputSize: 100000,  // Same data size
            OutputSize: 50000,
            CPURequirement: 2.0,
        }
        
        state := SystemState{
            QueueDepth: 25,
            ComputeUsage: 0.8,
        }
        
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, []OffloadTarget{slowTarget})
        
        // Simulate poor outcomes in slow network
        outcome := OffloadOutcome{
            DecisionID: decision.DecisionID,
            Success: true,
            CompletedOnTime: i < 20, // First 20 succeed, then start failing deadlines
            QueueReduction: 0.2,
            NetworkCostActual: 0.8, // High network cost
            NetworkCongestion: i > 30, // Congestion starts appearing
            Reward: float64(20-i) / 20.0, // Decreasing rewards
            Attribution: map[string]float64{
                "NetworkCost": 0.6,    // Network becomes problematic
                "QueueDepth": 0.2,
                "ProcessorLoad": 0.1,
                "LatencyCost": 0.1,
            },
        }
        
        weights := suite.offloader.GetCurrentWeights()
        suite.offloader.learner.UpdateWeights(&weights, outcome)
    }
    
    weightsAfterSlow := suite.offloader.GetCurrentWeights()
    suite.T().Logf("Weights after slow network: NetworkCost=%.3f", 
        weightsAfterSlow.NetworkCost)
    
    // Verify adaptation: NetworkCost weight should decrease
    suite.Less(weightsAfterSlow.NetworkCost, weightsAfterFast.NetworkCost,
        "NetworkCost weight should decrease after poor network performance")
    
    // Other weights should compensate
    totalWeightsBefore := weightsAfterFast.QueueDepth + weightsAfterFast.ProcessorLoad
    totalWeightsAfter := weightsAfterSlow.QueueDepth + weightsAfterSlow.ProcessorLoad
    
    suite.Greater(totalWeightsAfter, totalWeightsBefore,
        "Other weights should increase to compensate")
}

// Test pattern discovery for workload types
func (suite *LearningScenarioTestSuite) TestWorkloadPatternDiscovery() {
    // Simulate consistent pattern: CPU-intensive + high queue -> offload to edge
    suite.T().Log("Creating consistent CPU-intensive workload pattern")
    
    edgeTarget := OffloadTarget{
        ID: "edge-server",
        Type: "edge",
        AvailableCapacity: 16.0,
        ProcessingSpeed: 1.2, // Faster than local
    }
    
    cloudTarget := OffloadTarget{
        ID: "cloud-server",
        Type: "public_cloud", 
        AvailableCapacity: 32.0,
        ProcessingSpeed: 0.8, // Slower than local but more capacity
        NetworkLatency: 100 * time.Millisecond,
    }
    
    // Create consistent scenario: high CPU + high queue
    for i := 0; i < 30; i++ { // Above pattern discovery threshold
        process := Process{
            ID: fmt.Sprintf("cpu-intensive-%d", i),
            CPURequirement: 4.0,  // Consistent high CPU
            Type: "computation",  // Consistent type
            EstimatedDuration: 30 * time.Second,
        }
        
        state := SystemState{
            QueueDepth: 40 + i,          // Consistently high queue
            ComputeUsage: 0.85 + float64(i)*0.001, // Consistently high CPU
            MemoryUsage: 0.6,            // Moderate memory
        }
        
        // Algorithm should learn to prefer edge for this pattern
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, []OffloadTarget{edgeTarget, cloudTarget})
        
        // Simulate that edge works better for CPU-intensive work
        var outcome OffloadOutcome
        if decision.ShouldOffload && decision.Target.ID == "edge-server" {
            // Edge works well for CPU-intensive
            outcome = OffloadOutcome{
                DecisionID: decision.DecisionID,
                Success: true,
                CompletedOnTime: true,
                QueueReduction: 0.7,
                Reward: 2.0,
                SystemContext: state,
                ProcessContext: process,
                TargetContext: *decision.Target,
            }
        } else if decision.ShouldOffload && decision.Target.ID == "cloud-server" {
            // Cloud works but not as well
            outcome = OffloadOutcome{
                DecisionID: decision.DecisionID,
                Success: true,
                CompletedOnTime: i%4 != 0, // Occasional deadline misses
                QueueReduction: 0.5,
                Reward: 0.5,
                SystemContext: state,
                ProcessContext: process,
                TargetContext: *decision.Target,
            }
        } else {
            // Local execution struggles with high load
            outcome = OffloadOutcome{
                DecisionID: decision.DecisionID,
                Success: true,
                CompletedOnTime: i%3 != 0,
                QueueReduction: 0.1,
                LocalWorkDelayed: true,
                Reward: -0.5,
                SystemContext: state,
                ProcessContext: process,
            }
        }
        
        suite.offloader.learner.history = append(suite.offloader.learner.history, outcome)
        weights := suite.offloader.GetCurrentWeights()
        suite.offloader.learner.UpdateWeights(&weights, outcome)
    }
    
    // Trigger pattern discovery
    suite.offloader.learner.discoverPatterns()
    
    // Verify pattern was discovered
    patterns := suite.offloader.learner.patterns
    suite.NotEmpty(patterns, "Should discover patterns from consistent data")
    
    // Look for CPU-intensive + high queue pattern
    var cpuPattern *DiscoveredPattern
    for _, pattern := range patterns {
        hasQueueCondition := false
        hasCPUCondition := false
        
        for _, condition := range pattern.Conditions {
            if condition.Field == "QueueDepth" && condition.Operator == GT {
                hasQueueCondition = true
            }
            if condition.Field == "ComputeUsage" && condition.Operator == GT {
                hasCPUCondition = true
            }
        }
        
        if hasQueueCondition && hasCPUCondition {
            cpuPattern = &pattern
            break
        }
    }
    
    suite.NotNil(cpuPattern, "Should discover CPU-intensive + high queue pattern")
    suite.Greater(cpuPattern.SuccessRate, 0.8, "Pattern should have high success rate")
    suite.Contains(cpuPattern.PreferredTargets, "edge", "Should prefer edge targets")
    
    suite.T().Logf("Discovered pattern: %s (success rate: %.2f)", 
        cpuPattern.Description, cpuPattern.SuccessRate)
}

// Test adaptation to new workload types
func (suite *LearningScenarioTestSuite) TestNewWorkloadAdaptation() {
    // Phase 1: Train on compute-heavy workloads
    suite.T().Log("Phase 1: Training on compute-heavy workloads")
    
    for i := 0; i < 40; i++ {
        process := Process{
            ID: fmt.Sprintf("compute-%d", i),
            Type: "compute",
            CPURequirement: 4.0,
            MemoryRequirement: 2048 * 1024 * 1024, // 2GB
            InputSize: 1024,    // Small data
            OutputSize: 512,
        }
        
        state := SystemState{QueueDepth: 20, ComputeUsage: 0.8}
        
        target := OffloadTarget{
            ID: "compute-server",
            AvailableCapacity: 8.0,
            ProcessingSpeed: 1.5,
        }
        
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, []OffloadTarget{target})
        
        outcome := OffloadOutcome{
            DecisionID: decision.DecisionID,
            Success: true,
            CompletedOnTime: true,
            Reward: 1.5,
            Attribution: map[string]float64{
                "ProcessorLoad": 0.6, // Processor load very important
                "QueueDepth": 0.3,
                "NetworkCost": 0.1,
            },
        }
        
        weights := suite.offloader.GetCurrentWeights()
        suite.offloader.learner.UpdateWeights(&weights, outcome)
    }
    
    weightsAfterCompute := suite.offloader.GetCurrentWeights()
    suite.T().Logf("Weights after compute training: ProcessorLoad=%.3f", 
        weightsAfterCompute.ProcessorLoad)
    
    // Phase 2: Introduce data-heavy workloads
    suite.T().Log("Phase 2: Introducing data-heavy workloads")
    
    for i := 0; i < 40; i++ {
        process := Process{
            ID: fmt.Sprintf("data-%d", i),
            Type: "data-processing",
            CPURequirement: 1.0,        // Light CPU
            MemoryRequirement: 512 * 1024 * 1024, // 512MB
            InputSize: 10 * 1024 * 1024,  // 10MB input
            OutputSize: 5 * 1024 * 1024,  // 5MB output
        }
        
        state := SystemState{QueueDepth: 20, ComputeUsage: 0.8}
        
        target := OffloadTarget{
            ID: "data-server",
            AvailableCapacity: 8.0,
            NetworkBandwidth: 100000,    // Limited bandwidth
            NetworkLatency: 50 * time.Millisecond,
        }
        
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, []OffloadTarget{target})
        
        // Data-heavy workloads suffer from network costs
        var outcome OffloadOutcome
        if decision.ShouldOffload {
            outcome = OffloadOutcome{
                DecisionID: decision.DecisionID,
                Success: true,
                CompletedOnTime: i%3 != 0,  // Network delays cause issues
                NetworkCostActual: 0.8,
                Reward: float64(i%3) - 0.5, // Variable success
                Attribution: map[string]float64{
                    "NetworkCost": 0.7,    // Network becomes critical
                    "ProcessorLoad": 0.1,
                    "QueueDepth": 0.2,
                },
            }
        } else {
            outcome = OffloadOutcome{
                DecisionID: decision.DecisionID,
                Success: true,
                CompletedOnTime: true,
                Reward: 1.0,
                Attribution: map[string]float64{
                    "NetworkCost": 0.5,    // Good decision to avoid network
                    "ProcessorLoad": 0.2,
                    "QueueDepth": 0.3,
                },
            }
        }
        
        weights := suite.offloader.GetCurrentWeights()
        suite.offloader.learner.UpdateWeights(&weights, outcome)
    }
    
    weightsAfterData := suite.offloader.GetCurrentWeights()
    suite.T().Logf("Weights after data training: NetworkCost=%.3f", 
        weightsAfterData.NetworkCost)
    
    // Verify adaptation to new workload characteristics
    suite.Greater(weightsAfterData.NetworkCost, weightsAfterCompute.NetworkCost,
        "NetworkCost weight should increase for data-heavy workloads")
    
    // Test algorithm performance on mixed workload
    suite.T().Log("Phase 3: Testing on mixed workloads")
    
    correctDecisions := 0
    totalDecisions := 20
    
    for i := 0; i < totalDecisions; i++ {
        var process Process
        var expectedOffload bool
        
        if i%2 == 0 {
            // Compute-heavy: should offload
            process = Process{
                ID: fmt.Sprintf("mixed-compute-%d", i),
                Type: "compute",
                CPURequirement: 4.0,
                InputSize: 1024,
                OutputSize: 512,
            }
            expectedOffload = true
        } else {
            // Data-heavy: should keep local
            process = Process{
                ID: fmt.Sprintf("mixed-data-%d", i),
                Type: "data-processing",
                CPURequirement: 1.0,
                InputSize: 20 * 1024 * 1024,  // 20MB
                OutputSize: 10 * 1024 * 1024, // 10MB
            }
            expectedOffload = false
        }
        
        state := SystemState{QueueDepth: 25, ComputeUsage: 0.8}
        
        targets := []OffloadTarget{
            {
                ID: "mixed-server",
                AvailableCapacity: 8.0,
                NetworkBandwidth: 100000,
                ProcessingSpeed: 1.3,
            },
        }
        
        decision := suite.offloader.decisionEngine.ComputeOffloadDecision(
            process, state, targets)
        
        if decision.ShouldOffload == expectedOffload {
            correctDecisions++
        }
    }
    
    accuracy := float64(correctDecisions) / float64(totalDecisions)
    suite.Greater(accuracy, 0.7, 
        "Should make correct decisions >70%% of time on mixed workloads, got %.1f%%", 
        accuracy*100)
    
    suite.T().Logf("Mixed workload accuracy: %.1f%% (%d/%d)", 
        accuracy*100, correctDecisions, totalDecisions)
}

func TestLearningScenarioTestSuite(t *testing.T) {
    suite.Run(t, new(LearningScenarioTestSuite))
}
```

## 6. Test Data and Fixtures

### 6.1 Test Data Generation

```go
// fixtures/test_data_generator.go
package fixtures

import (
    "math/rand"
    "time"
)

type TestDataGenerator struct {
    rand *rand.Rand
}

func NewTestDataGenerator(seed int64) *TestDataGenerator {
    return &TestDataGenerator{
        rand: rand.New(rand.NewSource(seed)),
    }
}

// Generate realistic process workloads
func (tg *TestDataGenerator) GenerateProcesses(count int) []Process {
    processes := []Process{}
    
    processTypes := []string{"compute", "data-processing", "web-service", "batch-job", "ml-training"}
    
    for i := 0; i < count; i++ {
        processType := processTypes[tg.rand.Intn(len(processTypes))]
        
        var process Process
        switch processType {
        case "compute":
            process = Process{
                ID: fmt.Sprintf("compute-%d", i),
                Type: "compute",
                CPURequirement: 2.0 + tg.rand.Float64()*6.0,  // 2-8 cores
                MemoryRequirement: int64(1024*1024*1024) * int64(1+tg.rand.Intn(8)), // 1-8GB
                InputSize: int64(1024 + tg.rand.Intn(1024*100)),  // 1KB-100KB
                OutputSize: int64(512 + tg.rand.Intn(1024*50)),   // 0.5KB-50KB
                EstimatedDuration: time.Duration(30+tg.rand.Intn(300)) * time.Second, // 30s-5min
                MaxDuration: time.Duration(60+tg.rand.Intn(600)) * time.Second,      // 1min-10min
                Priority: 5 + tg.rand.Intn(5),                    // Priority 5-9
                DataSensitivity: tg.rand.Intn(3),                 // Low sensitivity
                SecurityLevel: 1 + tg.rand.Intn(3),               // Security 1-3
            }
            
        case "data-processing":
            process = Process{
                ID: fmt.Sprintf("data-%d", i),
                Type: "data-processing",
                CPURequirement: 1.0 + tg.rand.Float64()*2.0,     // 1-3 cores
                MemoryRequirement: int64(1024*1024*1024) * int64(2+tg.rand.Intn(16)), // 2-16GB
                InputSize: int64(1024*1024) * int64(1+tg.rand.Intn(100)),    // 1-100MB
                OutputSize: int64(1024*1024) * int64(1+tg.rand.Intn(50)),     // 1-50MB
                EstimatedDuration: time.Duration(60+tg.rand.Intn(600)) * time.Second,  // 1-10min
                MaxDuration: time.Duration(120+tg.rand.Intn(1200)) * time.Second,     // 2-20min
                Priority: 3 + tg.rand.Intn(6),                    // Priority 3-8
                DataSensitivity: 1 + tg.rand.Intn(4),             // Moderate sensitivity
                SecurityLevel: 2 + tg.rand.Intn(3),               // Security 2-4
            }
            
        case "web-service":
            process = Process{
                ID: fmt.Sprintf("web-%d", i),
                Type: "web-service",
                CPURequirement: 0.5 + tg.rand.Float64()*1.5,     // 0.5-2 cores
                MemoryRequirement: int64(1024*1024*512) * int64(1+tg.rand.Intn(4)), // 512MB-2GB
                InputSize: int64(1024) * int64(1+tg.rand.Intn(100)),        // 1-100KB
                OutputSize: int64(1024) * int64(1+tg.rand.Intn(500)),       // 1-500KB
                EstimatedDuration: time.Duration(100+tg.rand.Intn(900)) * time.Millisecond, // 100ms-1s
                MaxDuration: time.Duration(1+tg.rand.Intn(9)) * time.Second,        // 1-10s
                Priority: 7 + tg.rand.Intn(3),                    // High priority 7-9
                RealTime: tg.rand.Float32() < 0.7,                // 70% real-time
                DataSensitivity: tg.rand.Intn(3),                 // Low sensitivity
                SecurityLevel: 1 + tg.rand.Intn(4),               // Security 1-4
            }
            
        case "batch-job":
            process = Process{
                ID: fmt.Sprintf("batch-%d", i),
                Type: "batch-job",
                CPURequirement: 1.0 + tg.rand.Float64()*4.0,     // 1-5 cores
                MemoryRequirement: int64(1024*1024*1024) * int64(1+tg.rand.Intn(32)), // 1-32GB
                InputSize: int64(1024*1024) * int64(10+tg.rand.Intn(1000)),   // 10MB-1GB
                OutputSize: int64(1024*1024) * int64(5+tg.rand.Intn(500)),    // 5-500MB
                EstimatedDuration: time.Duration(300+tg.rand.Intn(7200)) * time.Second, // 5min-2hrs
                MaxDuration: time.Duration(600+tg.rand.Intn(14400)) * time.Second,     // 10min-4hrs
                Priority: 1 + tg.rand.Intn(5),                    // Low priority 1-5
                DataSensitivity: 2 + tg.rand.Intn(3),             // Moderate-high sensitivity
                SecurityLevel: 2 + tg.rand.Intn(3),               // Security 2-4
            }
            
        case "ml-training":
            process = Process{
                ID: fmt.Sprintf("ml-%d", i),
                Type: "ml-training",
                CPURequirement: 4.0 + tg.rand.Float64()*12.0,    // 4-16 cores
                MemoryRequirement: int64(1024*1024*1024) * int64(8+tg.rand.Intn(24)), // 8-32GB
                InputSize: int64(1024*1024) * int64(100+tg.rand.Intn(10000)), // 100MB-10GB
                OutputSize: int64(1024*1024) * int64(10+tg.rand.Intn(1000)),  // 10MB-1GB
                EstimatedDuration: time.Duration(1800+tg.rand.Intn(21600)) * time.Second, // 30min-6hrs
                MaxDuration: time.Duration(3600+tg.rand.Intn(43200)) * time.Second,      // 1hr-12hrs
                Priority: 4 + tg.rand.Intn(4),                    // Priority 4-7
                DataSensitivity: 3 + tg.rand.Intn(2),             // High sensitivity
                SecurityLevel: 3 + tg.rand.Intn(2),               // Security 3-4
            }
        }
        
        // Common fields
        process.SubmissionTime = time.Now().Add(-time.Duration(tg.rand.Intn(3600)) * time.Second)
        process.Status = QUEUED
        process.LocalityRequired = tg.rand.Float32() < 0.1  // 10% require locality
        process.SafetyCritical = tg.rand.Float32() < 0.05   // 5% safety critical
        
        processes = append(processes, process)
    }
    
    return processes
}

// Generate realistic target configurations
func (tg *TestDataGenerator) GenerateTargets(count int) []OffloadTarget {
    targets := []OffloadTarget{}
    
    targetTypes := []TargetType{"edge", "private_cloud", "public_cloud"}
    locations := []string{"local", "regional", "national", "international"}
    jurisdictions := []string{"domestic", "eu", "asia", "americas"}
    
    for i := 0; i < count; i++ {
        targetType := targetTypes[tg.rand.Intn(len(targetTypes))]
        
        var target OffloadTarget
        switch targetType {
        case "edge":
            target = OffloadTarget{
                ID: fmt.Sprintf("edge-%d", i),
                Type: "edge",
                Location: locations[tg.rand.Intn(2)], // local or regional
                TotalCapacity: 8.0 + tg.rand.Float64()*24.0,  // 8-32 cores
                AvailableCapacity: 0.3 + tg.rand.Float64()*0.6, // 30-90% available
                MemoryTotal: int64(1024*1024*1024) * int64(16+tg.rand.Intn(48)), // 16-64GB
                NetworkLatency: time.Duration(5+tg.rand.Intn(50)) * time.Millisecond, // 5-55ms
                NetworkBandwidth: float64(1000000 * (100 + tg.rand.Intn(900))), // 100MB/s-1GB/s
                NetworkStability: 0.85 + tg.rand.Float64()*0.14,  // 85-99%
                NetworkCost: 0.01 + tg.rand.Float64()*0.04,       // $0.01-$0.05 per MB
                ProcessingSpeed: 0.8 + tg.rand.Float64()*0.6,     // 0.8x-1.4x speed
                Reliability: 0.90 + tg.rand.Float64()*0.09,       // 90-99%
                ComputeCost: 0.05 + tg.rand.Float64()*0.10,       // $0.05-$0.15 per hour
                SecurityLevel: 2 + tg.rand.Intn(3),               // Security 2-4
                DataJurisdiction: jurisdictions[tg.rand.Intn(2)], // domestic or regional
            }
            
        case "private_cloud":
            target = OffloadTarget{
                ID: fmt.Sprintf("private-cloud-%d", i),
                Type: "private_cloud", 
                Location: locations[1+tg.rand.Intn(2)], // regional or national
                TotalCapacity: 32.0 + tg.rand.Float64()*96.0, // 32-128 cores
                AvailableCapacity: 0.4 + tg.rand.Float64()*0.5, // 40-90% available
                MemoryTotal: int64(1024*1024*1024) * int64(64+tg.rand.Intn(192)), // 64-256GB
                NetworkLatency: time.Duration(20+tg.rand.Intn(80)) * time.Millisecond, // 20-100ms
                NetworkBandwidth: float64(1000000 * (50 + tg.rand.Intn(450))), // 50-500MB/s
                NetworkStability: 0.95 + tg.rand.Float64()*0.04,  // 95-99%
                NetworkCost: 0.005 + tg.rand.Float64()*0.015,     // $0.005-$0.02 per MB
                ProcessingSpeed: 0.9 + tg.rand.Float64()*0.4,     // 0.9x-1.3x speed
                Reliability: 0.95 + tg.rand.Float64()*0.04,       // 95-99%
                ComputeCost: 0.08 + tg.rand.Float64()*0.12,       // $0.08-$0.20 per hour
                SecurityLevel: 3 + tg.rand.Intn(2),               // Security 3-4
                DataJurisdiction: jurisdictions[tg.rand.Intn(len(jurisdictions))],
            }
            
        case "public_cloud":
            target = OffloadTarget{
                ID: fmt.Sprintf("public-cloud-%d", i),
                Type: "public_cloud",
                Location: locations[2+tg.rand.Intn(2)], // national or international
                TotalCapacity: 64.0 + tg.rand.Float64()*192.0, // 64-256 cores (virtually unlimited)
                AvailableCapacity: 0.8 + tg.rand.Float64()*0.19, // 80-99% available
                MemoryTotal: int64(1024*1024*1024) * int64(128+tg.rand.Intn(384)), // 128-512GB
                NetworkLatency: time.Duration(50+tg.rand.Intn(200)) * time.Millisecond, // 50-250ms
                NetworkBandwidth: float64(1000000 * (10 + tg.rand.Intn(190))), // 10-200MB/s
                NetworkStability: 0.99 + tg.rand.Float64()*0.009, // 99-99.9%
                NetworkCost: 0.02 + tg.rand.Float64()*0.08,       // $0.02-$0.10 per MB
                ProcessingSpeed: 0.7 + tg.rand.Float64()*0.8,     // 0.7x-1.5x speed
                Reliability: 0.999,                               // 99.9%
                ComputeCost: 0.10 + tg.rand.Float64()*0.20,       // $0.10-$0.30 per hour
                SecurityLevel: 1 + tg.rand.Intn(4),               // Security 1-4
                DataJurisdiction: jurisdictions[tg.rand.Intn(len(jurisdictions))],
            }
        }
        
        // Common fields
        target.MemoryAvailable = int64(float64(target.MemoryTotal) * (target.AvailableCapacity * 0.8))
        target.CurrentLoad = 1.0 - target.AvailableCapacity
        target.EstimatedWaitTime = time.Duration(target.CurrentLoad*30) * time.Second
        target.LastSeen = time.Now().Add(-time.Duration(tg.rand.Intn(60)) * time.Second)
        target.PolicyBonus = 0.0
        target.HistoricalSuccess = 0.5 + tg.rand.Float64()*0.4 // 50-90%
        
        targets = append(targets, target)
    }
    
    return targets
}

// Generate realistic system states
func (tg *TestDataGenerator) GenerateSystemStates(count int) []SystemState {
    states := []SystemState{}
    
    for i := 0; i < count; i++ {
        // Generate realistic usage patterns
        baseLoad := 0.3 + tg.rand.Float64()*0.4 // Base load 30-70%
        
        // Add some correlation between metrics
        computeUsage := baseLoad + tg.rand.Float64()*0.2 - 0.1
        memoryUsage := computeUsage*0.8 + tg.rand.Float64()*0.3
        networkUsage := 0.1 + tg.rand.Float64()*0.6
        
        // Clamp to valid ranges
        computeUsage = math.Max(0.0, math.Min(1.0, computeUsage))
        memoryUsage = math.Max(0.0, math.Min(1.0, memoryUsage))
        networkUsage = math.Max(0.0, math.Min(1.0, networkUsage))
        
        queueDepth := int(computeUsage*50 + tg.rand.Float64()*20) // Queue correlates with load
        
        state := SystemState{
            QueueDepth: queueDepth,
            QueueThreshold: 20 + tg.rand.Intn(30), // Threshold 20-50
            QueueWaitTime: time.Duration(queueDepth*2+tg.rand.Intn(20)) * time.Second,
            QueueThroughput: 5.0 + tg.rand.Float64()*10.0, // 5-15 processes/second
            ComputeUsage: computeUsage,
            MemoryUsage: memoryUsage,
            DiskUsage: 0.2 + tg.rand.Float64()*0.4, // 20-60%
            NetworkUsage: networkUsage,
            MasterUsage: 0.1 + computeUsage*0.3 + tg.rand.Float64()*0.2, // Master load correlates with system load
            ActiveConnections: 10 + tg.rand.Intn(100),
            Timestamp: time.Now().Add(-time.Duration(tg.rand.Intn(3600)) * time.Second),
            TimeSlot: tg.rand.Intn(24),
            DayOfWeek: tg.rand.Intn(7),
        }
        
        states = append(states, state)
    }
    
    return states
}
```

This comprehensive test suite design provides:

1. **Complete test coverage** across all algorithm components
2. **Performance validation** for latency, throughput, and memory usage
3. **Learning behavior verification** through scenario-based tests
4. **Integration validation** with ColonyOS and external systems
5. **Realistic test data generation** for diverse scenarios
6. **Structured test organization** following Go testing best practices

The test suite supports test-driven development by providing clear specifications for expected behavior, comprehensive edge case coverage, and realistic performance benchmarks that the implementation must meet.

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "Create detailed algorithm specification documentation", "status": "completed", "activeForm": "Creating detailed algorithm specification documentation"}, {"content": "Design comprehensive test suite for core components", "status": "completed", "activeForm": "Designing comprehensive test suite for core components"}, {"content": "Define test scenarios for adaptive learning behavior", "status": "pending", "activeForm": "Defining test scenarios for adaptive learning behavior"}, {"content": "Create integration tests for ColonyOS compatibility", "status": "pending", "activeForm": "Creating integration tests for ColonyOS compatibility"}, {"content": "Implement core algorithm components using TDD", "status": "pending", "activeForm": "Implementing core algorithm components using TDD"}, {"content": "Implement learning and adaptation mechanisms", "status": "pending", "activeForm": "Implementing learning and adaptation mechanisms"}, {"content": "Create performance benchmarking tests", "status": "pending", "activeForm": "Creating performance benchmarking tests"}]