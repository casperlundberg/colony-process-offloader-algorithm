# ColonyOS Integration Test Specifications

## Overview

This document defines comprehensive integration tests for the adaptive offloading algorithm with ColonyOS. These tests ensure seamless integration, proper API usage, and correct behavior within the ColonyOS ecosystem.

## 1. Integration Test Architecture

### 1.1 Test Environment Setup

```go
// integration/colonyos_test_env.go
package integration

import (
    "context"
    "database/sql"
    "net"
    "testing"
    "time"
    
    "github.com/colonyos/colonies/pkg/core"
    "github.com/colonyos/colonies/pkg/server"
    "github.com/docker/go-connections/nat"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

type ColonyOSTestEnvironment struct {
    // ColonyOS components
    ColonyServer    *server.ColoniesServer
    Database        *sql.DB
    ServerContainer testcontainers.Container
    
    // Test colonies and executors
    TestColony      *core.Colony
    LocalExecutor   *core.Executor
    EdgeExecutors   []*core.Executor
    CloudExecutors  []*core.Executor
    
    // Network configuration
    ServerHost      string
    ServerPort      string
    DatabaseURL     string
    
    // Test data
    TestProcesses   []core.Process
    TestFunctions   []core.FunctionSpec
    
    // Offloader integration
    Offloader       *AdaptiveOffloader
    OffloaderConfig *OffloadConfig
}

func NewColonyOSTestEnvironment() *ColonyOSTestEnvironment {
    return &ColonyOSTestEnvironment{
        EdgeExecutors:  []*core.Executor{},
        CloudExecutors: []*core.Executor{},
        TestProcesses:  []core.Process{},
        TestFunctions:  []core.FunctionSpec{},
    }
}

func (env *ColonyOSTestEnvironment) Start(ctx context.Context) error {
    // Start PostgreSQL container
    if err := env.startDatabase(ctx); err != nil {
        return fmt.Errorf("failed to start database: %w", err)
    }
    
    // Start ColonyOS server
    if err := env.startColonyServer(ctx); err != nil {
        return fmt.Errorf("failed to start colony server: %w", err)
    }
    
    // Create test colony
    if err := env.createTestColony(ctx); err != nil {
        return fmt.Errorf("failed to create test colony: %w", err)
    }
    
    // Register test executors
    if err := env.registerExecutors(ctx); err != nil {
        return fmt.Errorf("failed to register executors: %w", err)
    }
    
    // Initialize offloader
    if err := env.initializeOffloader(ctx); err != nil {
        return fmt.Errorf("failed to initialize offloader: %w", err)
    }
    
    // Load test data
    if err := env.loadTestData(ctx); err != nil {
        return fmt.Errorf("failed to load test data: %w", err)
    }
    
    return nil
}

func (env *ColonyOSTestEnvironment) startDatabase(ctx context.Context) error {
    req := testcontainers.ContainerRequest{
        Image:        "postgres:13",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_DB":       "colonies_test",
            "POSTGRES_USER":     "postgres",
            "POSTGRES_PASSWORD": "test123",
        },
        WaitingFor: wait.ForListeningPort("5432/tcp"),
    }
    
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:         true,
    })
    
    if err != nil {
        return err
    }
    
    host, err := container.Host(ctx)
    if err != nil {
        return err
    }
    
    port, err := container.MappedPort(ctx, "5432")
    if err != nil {
        return err
    }
    
    env.DatabaseURL = fmt.Sprintf("postgresql://postgres:test123@%s:%s/colonies_test?sslmode=disable", 
        host, port.Port())
    
    env.ServerContainer = container
    
    // Connect to database
    db, err := sql.Open("postgres", env.DatabaseURL)
    if err != nil {
        return err
    }
    
    env.Database = db
    return nil
}

func (env *ColonyOSTestEnvironment) startColonyServer(ctx context.Context) error {
    // Initialize ColonyOS server with test database
    serverConfig := &server.Config{
        Host:         "localhost",
        Port:         0, // Let system assign port
        DatabaseURL:  env.DatabaseURL,
        LogLevel:     "error", // Quiet during tests
        DevMode:      true,
    }
    
    colonyServer, err := server.NewColoniesServer(serverConfig)
    if err != nil {
        return err
    }
    
    // Start server in goroutine
    go func() {
        if err := colonyServer.Start(); err != nil {
            panic(fmt.Sprintf("Failed to start colony server: %v", err))
        }
    }()
    
    // Wait for server to be ready
    time.Sleep(2 * time.Second)
    
    env.ColonyServer = colonyServer
    env.ServerHost = serverConfig.Host
    env.ServerPort = fmt.Sprintf("%d", colonyServer.GetPort())
    
    return nil
}

func (env *ColonyOSTestEnvironment) createTestColony(ctx context.Context) error {
    colony := &core.Colony{
        ID:   core.GenerateRandomID(),
        Name: "test-colony-offloader",
    }
    
    if err := env.ColonyServer.AddColony(colony); err != nil {
        return err
    }
    
    env.TestColony = colony
    return nil
}

func (env *ColonyOSTestEnvironment) registerExecutors(ctx context.Context) error {
    // Register local executor
    localExecutor := &core.Executor{
        ID:         core.GenerateRandomID(),
        Type:       "local",
        Name:       "local-executor",
        ColonyName: env.TestColony.Name,
        State:      core.APPROVED,
    }
    
    if err := env.ColonyServer.AddExecutor(localExecutor); err != nil {
        return err
    }
    
    env.LocalExecutor = localExecutor
    
    // Register edge executors
    for i := 0; i < 3; i++ {
        edgeExecutor := &core.Executor{
            ID:         core.GenerateRandomID(),
            Type:       "edge",
            Name:       fmt.Sprintf("edge-executor-%d", i),
            ColonyName: env.TestColony.Name,
            State:      core.APPROVED,
            Capabilities: []string{
                "compute_optimized",
                "low_latency",
            },
        }
        
        if err := env.ColonyServer.AddExecutor(edgeExecutor); err != nil {
            return err
        }
        
        env.EdgeExecutors = append(env.EdgeExecutors, edgeExecutor)
    }
    
    // Register cloud executors
    cloudTypes := []string{"compute_optimized", "memory_optimized", "cost_optimized"}
    
    for i, cloudType := range cloudTypes {
        cloudExecutor := &core.Executor{
            ID:         core.GenerateRandomID(),
            Type:       "cloud",
            Name:       fmt.Sprintf("cloud-executor-%s", cloudType),
            ColonyName: env.TestColony.Name,
            State:      core.APPROVED,
            Capabilities: []string{
                cloudType,
                "high_capacity",
                "scalable",
            },
        }
        
        if err := env.ColonyServer.AddExecutor(cloudExecutor); err != nil {
            return err
        }
        
        env.CloudExecutors = append(env.CloudExecutors, cloudExecutor)
    }
    
    return nil
}

func (env *ColonyOSTestEnvironment) initializeOffloader(ctx context.Context) error {
    config := &OffloadConfig{
        ColonyName: env.TestColony.Name,
        ServerHost: env.ServerHost,
        ServerPort: env.ServerPort,
        
        Margins: SafetyMargins{
            Compute: 0.3,
            Master:  0.4,
            Network: 0.5,
        },
        
        Weights: AdaptiveWeights{
            QueueDepth:    0.2,
            ProcessorLoad: 0.2,
            NetworkCost:   0.2,
            LatencyCost:   0.2,
            EnergyCost:    0.1,
            PolicyCost:    0.1,
        },
        
        Learning: LearningConfig{
            WindowSize:      100,
            LearningRate:    0.01,
            ExplorationRate: 0.1,
            MinSamples:      10,
        },
    }
    
    offloader, err := NewAdaptiveOffloader(env.ColonyServer, config)
    if err != nil {
        return err
    }
    
    env.Offloader = offloader
    env.OffloaderConfig = config
    
    return nil
}

func (env *ColonyOSTestEnvironment) Stop() error {
    if env.Offloader != nil {
        env.Offloader.Stop()
    }
    
    if env.ColonyServer != nil {
        env.ColonyServer.Stop()
    }
    
    if env.Database != nil {
        env.Database.Close()
    }
    
    if env.ServerContainer != nil {
        return env.ServerContainer.Terminate(context.Background())
    }
    
    return nil
}
```

### 1.2 Base Integration Test Suite

```go
// integration/colonyos_integration_test.go
package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/suite"
)

type ColonyOSIntegrationTestSuite struct {
    suite.Suite
    env           *ColonyOSTestEnvironment
    ctx           context.Context
    cancel        context.CancelFunc
    testTimeout   time.Duration
}

func (suite *ColonyOSIntegrationTestSuite) SetupSuite() {
    suite.testTimeout = 60 * time.Second
    suite.ctx, suite.cancel = context.WithTimeout(context.Background(), suite.testTimeout)
    
    suite.env = NewColonyOSTestEnvironment()
    
    err := suite.env.Start(suite.ctx)
    suite.Require().NoError(err, "Failed to start test environment")
    
    // Wait for system to stabilize
    time.Sleep(2 * time.Second)
}

func (suite *ColonyOSIntegrationTestSuite) TearDownSuite() {
    if suite.env != nil {
        err := suite.env.Stop()
        suite.NoError(err, "Failed to clean up test environment")
    }
    
    if suite.cancel != nil {
        suite.cancel()
    }
}

func (suite *ColonyOSIntegrationTestSuite) SetupTest() {
    // Reset offloader state between tests
    suite.env.Offloader.Reset()
    
    // Clear process queue
    suite.clearProcessQueue()
}

func (suite *ColonyOSIntegrationTestSuite) clearProcessQueue() {
    processes, err := suite.env.ColonyServer.GetWaitingProcesses(suite.env.TestColony.Name)
    suite.NoError(err)
    
    for _, process := range processes {
        err := suite.env.ColonyServer.DeleteProcess(process.ID)
        suite.NoError(err)
    }
}

func TestColonyOSIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(ColonyOSIntegrationTestSuite))
}
```

## 2. Core Integration Tests

### 2.1 Process Queue Integration

```go
// Test process queue monitoring and interaction
func (suite *ColonyOSIntegrationTestSuite) TestProcessQueueIntegration() {
    // Submit test processes to ColonyOS
    testProcesses := []core.ProcessSpec{
        {
            FuncName:    "compute_task",
            Args:        []string{"--input", "test1.dat"},
            ColonyName:  suite.env.TestColony.Name,
            MaxWaitTime: 300,
            MaxExecTime: 600,
            MaxRetries:  3,
            Priority:    5,
        },
        {
            FuncName:    "data_processing", 
            Args:        []string{"--dataset", "large_data.csv", "--output", "results.json"},
            ColonyName:  suite.env.TestColony.Name,
            MaxWaitTime: 600,
            MaxExecTime: 1200,
            MaxRetries:  2,
            Priority:    7,
        },
        {
            FuncName:    "web_service",
            Args:        []string{"--port", "8080", "--workers", "4"},
            ColonyName:  suite.env.TestColony.Name,
            MaxWaitTime: 30,
            MaxExecTime: 3600,
            MaxRetries:  5,
            Priority:    9, // High priority
        },
    }
    
    processIDs := []string{}
    
    // Submit processes
    for _, spec := range testProcesses {
        process, err := suite.env.ColonyServer.SubmitProcessSpec(spec)
        suite.NoError(err, "Failed to submit process")
        suite.NotEmpty(process.ID, "Process ID should not be empty")
        
        processIDs = append(processIDs, process.ID)
    }
    
    // Verify processes are in queue
    queuedProcesses, err := suite.env.ColonyServer.GetWaitingProcesses(suite.env.TestColony.Name)
    suite.NoError(err, "Failed to get queued processes")
    suite.Len(queuedProcesses, 3, "Should have 3 processes in queue")
    
    // Test offloader can read queue
    candidates := suite.env.Offloader.identifyOffloadCandidates()
    suite.NotEmpty(candidates, "Offloader should identify offload candidates")
    
    // Verify candidates match submitted processes
    candidateIDs := make(map[string]bool)
    for _, candidate := range candidates {
        candidateIDs[candidate.ID] = true
    }
    
    for _, processID := range processIDs {
        suite.True(candidateIDs[processID], 
            "Process %s should be identified as candidate", processID)
    }
    
    // Test priority ordering
    suite.Greater(candidates[0].Priority, candidates[len(candidates)-1].Priority,
        "Candidates should be ordered by priority")
}

// Test process execution delegation
func (suite *ColonyOSIntegrationTestSuite) TestProcessExecutionDelegation() {
    // Submit a process that should be offloaded
    processSpec := core.ProcessSpec{
        FuncName:    "cpu_intensive_task",
        Args:        []string{"--iterations", "1000000"},
        ColonyName:  suite.env.TestColony.Name,
        MaxWaitTime: 300,
        MaxExecTime: 600,
        Priority:    5,
    }
    
    submittedProcess, err := suite.env.ColonyServer.SubmitProcessSpec(processSpec)
    suite.NoError(err, "Failed to submit process")
    
    // Create high load conditions to trigger offloading
    suite.simulateHighLoad()
    
    // Start offloader
    ctx, cancel := context.WithTimeout(suite.ctx, 30*time.Second)
    defer cancel()
    
    go suite.env.Offloader.Run(ctx)
    
    // Wait for offloading decision
    decision := suite.waitForOffloadDecision(submittedProcess.ID, 10*time.Second)
    suite.NotNil(decision, "Should make offload decision")
    
    if decision.ShouldOffload {
        suite.T().Logf("Process %s offloaded to %s", submittedProcess.ID, decision.Target.ID)
        
        // Verify process was assigned to correct executor
        assignedProcess, err := suite.env.ColonyServer.GetProcess(submittedProcess.ID)
        suite.NoError(err, "Failed to get assigned process")
        
        // Check that the process was assigned to the selected target
        suite.Equal(decision.Target.ID, assignedProcess.AssignedExecutorID,
            "Process should be assigned to selected target executor")
        
        // Wait for execution to complete
        suite.waitForProcessCompletion(submittedProcess.ID, 60*time.Second)
        
        // Verify execution outcome was recorded
        outcome := suite.waitForExecutionOutcome(submittedProcess.ID, 5*time.Second)
        suite.NotNil(outcome, "Should record execution outcome")
        suite.Equal(submittedProcess.ID, outcome.ProcessID, "Outcome should match process")
        
    } else {
        suite.T().Logf("Process %s kept local", submittedProcess.ID)
        
        // Verify process executes locally
        suite.waitForProcessCompletion(submittedProcess.ID, 60*time.Second)
        
        completedProcess, err := suite.env.ColonyServer.GetProcess(submittedProcess.ID)
        suite.NoError(err, "Failed to get completed process")
        suite.Equal(core.SUCCESSFUL, completedProcess.State, "Process should complete successfully")
    }
}

func (suite *ColonyOSIntegrationTestSuite) simulateHighLoad() {
    // Create system state with high resource usage
    highLoadState := SystemState{
        QueueDepth:      45, // Above typical threshold
        ComputeUsage:    0.85,
        MemoryUsage:     0.80,
        NetworkUsage:    0.60,
        MasterUsage:     0.70,
        QueueThroughput: 2.0, // Low throughput indicates bottleneck
    }
    
    // Inject this state into the offloader's monitoring
    suite.env.Offloader.InjectSystemState(highLoadState)
}
```

### 2.2 Executor Discovery and Management

```go
// Test executor discovery and capability matching
func (suite *ColonyOSIntegrationTestSuite) TestExecutorDiscovery() {
    // Test discovering available executors
    targets := suite.env.Offloader.discoverTargets()
    suite.NotEmpty(targets, "Should discover available targets")
    
    // Verify all registered executors are discovered
    expectedExecutorCount := 1 + len(suite.env.EdgeExecutors) + len(suite.env.CloudExecutors)
    suite.Len(targets, expectedExecutorCount, 
        "Should discover all registered executors as targets")
    
    // Verify executor types are correctly mapped
    targetsByType := make(map[string]int)
    for _, target := range targets {
        targetsByType[target.Type]++
    }
    
    suite.Equal(1, targetsByType["local"], "Should have 1 local target")
    suite.Equal(3, targetsByType["edge"], "Should have 3 edge targets")  
    suite.Equal(3, targetsByType["cloud"], "Should have 3 cloud targets")
    
    // Test capability-based filtering
    computeIntensiveProcess := Process{
        ID:             "compute-test",
        CPURequirement: 8.0,
        Type:           "compute_intensive",
    }
    
    suitableTargets := suite.env.Offloader.filterTargetsByCapability(
        computeIntensiveProcess, targets)
    
    // Should prefer compute-optimized executors
    hasComputeOptimized := false
    for _, target := range suitableTargets {
        if contains(target.Capabilities, "compute_optimized") {
            hasComputeOptimized = true
            break
        }
    }
    
    suite.True(hasComputeOptimized, 
        "Should include compute-optimized targets for CPU-intensive process")
}

// Test executor health monitoring
func (suite *ColonyOSIntegrationTestSuite) TestExecutorHealthMonitoring() {
    targets := suite.env.Offloader.discoverTargets()
    suite.NotEmpty(targets, "Should discover targets")
    
    // Test connectivity to each target
    for _, target := range targets {
        connectivity := suite.env.Offloader.testConnectivity(target)
        suite.NotNil(connectivity, "Should test connectivity to target %s", target.ID)
        
        if target.Type == "local" {
            suite.True(connectivity.Reachable, "Local target should be reachable")
            suite.Less(connectivity.Latency, 5*time.Millisecond, 
                "Local latency should be very low")
        } else {
            // Remote targets may have variable connectivity in test environment
            suite.T().Logf("Target %s (%s): reachable=%v, latency=%v", 
                target.ID, target.Type, connectivity.Reachable, connectivity.Latency)
        }
    }
    
    // Test handling of unreachable executor
    unreachableExecutor := &core.Executor{
        ID:         "unreachable-executor",
        Type:       "edge",
        Name:       "unreachable",
        ColonyName: suite.env.TestColony.Name,
        State:      core.APPROVED,
    }
    
    // Add unreachable executor to ColonyOS
    err := suite.env.ColonyServer.AddExecutor(unreachableExecutor)
    suite.NoError(err, "Failed to add unreachable executor")
    
    // Simulate executor being offline
    suite.env.ColonyServer.SetExecutorState(unreachableExecutor.ID, core.OFFLINE)
    
    // Rediscover targets
    updatedTargets := suite.env.Offloader.discoverTargets()
    
    // Unreachable executor should be filtered out or marked as unavailable
    unreachableFound := false
    for _, target := range updatedTargets {
        if target.ID == unreachableExecutor.ID {
            unreachableFound = true
            suite.False(target.Available, "Unreachable target should be marked unavailable")
        }
    }
    
    // Depending on implementation, unreachable targets might be filtered out entirely
    if !unreachableFound {
        suite.T().Log("Unreachable target was filtered out (acceptable behavior)")
    }
}
```

### 2.3 Process Lifecycle Integration

```go
// Test complete process lifecycle with offloading
func (suite *ColonyOSIntegrationTestSuite) TestProcessLifecycleIntegration() {
    // Define a test workflow with multiple processes
    workflowProcesses := []core.ProcessSpec{
        {
            FuncName:    "data_ingestion",
            Args:        []string{"--source", "input.csv"},
            ColonyName:  suite.env.TestColony.Name,
            MaxWaitTime: 300,
            MaxExecTime: 600,
            Priority:    8,
        },
        {
            FuncName:    "data_processing", 
            Args:        []string{"--algorithm", "ml_training"},
            ColonyName:  suite.env.TestColony.Name,
            MaxWaitTime: 600,
            MaxExecTime: 1800,
            Priority:    6,
            Dependencies: []string{}, // Will be set after first process is submitted
        },
        {
            FuncName:    "result_aggregation",
            Args:        []string{"--format", "json"},
            ColonyName:  suite.env.TestColony.Name,
            MaxWaitTime: 180,
            MaxExecTime: 300,
            Priority:    7,
            Dependencies: []string{}, // Will be set after second process
        },
    }
    
    // Start offloader
    ctx, cancel := context.WithTimeout(suite.ctx, 120*time.Second)
    defer cancel()
    
    go suite.env.Offloader.Run(ctx)
    
    processIDs := []string{}
    
    // Submit first process
    process1, err := suite.env.ColonyServer.SubmitProcessSpec(workflowProcesses[0])
    suite.NoError(err, "Failed to submit first process")
    processIDs = append(processIDs, process1.ID)
    
    // Wait for first process to complete
    suite.waitForProcessCompletion(process1.ID, 60*time.Second)
    
    completedProcess1, err := suite.env.ColonyServer.GetProcess(process1.ID)
    suite.NoError(err, "Failed to get completed process")
    suite.Equal(core.SUCCESSFUL, completedProcess1.State, "First process should succeed")
    
    // Submit second process with dependency
    workflowProcesses[1].Dependencies = []string{process1.ID}
    process2, err := suite.env.ColonyServer.SubmitProcessSpec(workflowProcesses[1])
    suite.NoError(err, "Failed to submit second process")
    processIDs = append(processIDs, process2.ID)
    
    // Wait for second process
    suite.waitForProcessCompletion(process2.ID, 90*time.Second)
    
    completedProcess2, err := suite.env.ColonyServer.GetProcess(process2.ID)
    suite.NoError(err, "Failed to get second completed process")
    suite.Equal(core.SUCCESSFUL, completedProcess2.State, "Second process should succeed")
    
    // Submit final process
    workflowProcesses[2].Dependencies = []string{process2.ID}
    process3, err := suite.env.ColonyServer.SubmitProcessSpec(workflowProcesses[2])
    suite.NoError(err, "Failed to submit third process")
    processIDs = append(processIDs, process3.ID)
    
    // Wait for final process
    suite.waitForProcessCompletion(process3.ID, 30*time.Second)
    
    completedProcess3, err := suite.env.ColonyServer.GetProcess(process3.ID)
    suite.NoError(err, "Failed to get third completed process")
    suite.Equal(core.SUCCESSFUL, completedProcess3.State, "Third process should succeed")
    
    // Verify offloader made decisions for all processes
    for _, processID := range processIDs {
        outcome := suite.getOffloadOutcome(processID)
        suite.NotNil(outcome, "Should have outcome for process %s", processID)
        
        suite.T().Logf("Process %s: Success=%v, CompletedOnTime=%v, Target=%s",
            processID, outcome.Success, outcome.CompletedOnTime, outcome.TargetID)
    }
    
    // Verify learning occurred
    initialWeights := suite.env.OffloaderConfig.Weights
    finalWeights := suite.env.Offloader.GetCurrentWeights()
    
    suite.NotEqual(initialWeights, finalWeights, "Weights should have adapted")
    
    // Verify patterns might have been discovered
    patterns := suite.env.Offloader.GetDiscoveredPatterns()
    suite.T().Logf("Discovered %d patterns during workflow execution", len(patterns))
}
```

### 2.4 Error Handling and Recovery

```go
// Test error handling and recovery scenarios
func (suite *ColonyOSIntegrationTestSuite) TestErrorHandlingAndRecovery() {
    // Test 1: Executor failure during process execution
    suite.T().Log("Testing executor failure handling")
    
    processSpec := core.ProcessSpec{
        FuncName:    "long_running_task",
        Args:        []string{"--duration", "30"},
        ColonyName:  suite.env.TestColony.Name,
        MaxWaitTime: 60,
        MaxExecTime: 120,
        Priority:    5,
    }
    
    process, err := suite.env.ColonyServer.SubmitProcessSpec(processSpec)
    suite.NoError(err, "Failed to submit process")
    
    // Force offloading to edge executor
    suite.simulateHighLoad()
    
    ctx, cancel := context.WithTimeout(suite.ctx, 90*time.Second)
    defer cancel()
    
    go suite.env.Offloader.Run(ctx)
    
    // Wait for process to start executing
    time.Sleep(5 * time.Second)
    
    executingProcess, err := suite.env.ColonyServer.GetProcess(process.ID)
    suite.NoError(err, "Failed to get executing process")
    
    if executingProcess.AssignedExecutorID != "" {
        suite.T().Logf("Process assigned to executor: %s", executingProcess.AssignedExecutorID)
        
        // Simulate executor failure
        err = suite.env.ColonyServer.SetExecutorState(
            executingProcess.AssignedExecutorID, core.OFFLINE)
        suite.NoError(err, "Failed to set executor offline")
        
        // Wait for recovery/reassignment
        time.Sleep(10 * time.Second)
        
        // Check process status
        recoveredProcess, err := suite.env.ColonyServer.GetProcess(process.ID)
        suite.NoError(err, "Failed to get recovered process")
        
        // Process should either complete on another executor or fail gracefully
        suite.True(recoveredProcess.State == core.SUCCESSFUL || 
                  recoveredProcess.State == core.FAILED,
                  "Process should reach final state after executor failure")
        
        // Verify offloader learned from the failure
        outcome := suite.getOffloadOutcome(process.ID)
        if outcome != nil && !outcome.Success {
            suite.T().Log("Offloader recorded failure outcome for learning")
            
            // Should reduce trust in the failed executor type
            targets := suite.env.Offloader.discoverTargets()
            for _, target := range targets {
                if target.ID == executingProcess.AssignedExecutorID {
                    suite.Less(target.HistoricalSuccess, 0.9,
                        "Failed executor should have reduced success rate")
                }
            }
        }
    }
    
    // Test 2: Network connectivity issues
    suite.T().Log("Testing network connectivity issues")
    
    // Submit data-intensive process
    dataProcessSpec := core.ProcessSpec{
        FuncName:    "data_transfer_task",
        Args:        []string{"--size", "100MB"},
        ColonyName:  suite.env.TestColony.Name,
        MaxWaitTime: 180,
        MaxExecTime: 300,
        Priority:    6,
    }
    
    dataProcess, err := suite.env.ColonyServer.SubmitProcessSpec(dataProcessSpec)
    suite.NoError(err, "Failed to submit data process")
    
    // Simulate network issues by injecting poor network conditions
    poorNetworkConditions := NetworkConditions{
        Latency:    500 * time.Millisecond,
        Bandwidth:  1000,  // Very low bandwidth
        PacketLoss: 0.1,   // 10% packet loss
        Stability:  0.3,   // Very unstable
    }
    
    suite.env.Offloader.InjectNetworkConditions(poorNetworkConditions)
    
    // Process should either be kept local or handled with appropriate timeouts
    suite.waitForProcessCompletion(dataProcess.ID, 300*time.Second)
    
    completedDataProcess, err := suite.env.ColonyServer.GetProcess(dataProcess.ID)
    suite.NoError(err, "Failed to get completed data process")
    
    // Should complete successfully despite network issues
    suite.Equal(core.SUCCESSFUL, completedDataProcess.State,
        "Process should complete successfully despite network issues")
    
    // Test 3: Resource exhaustion
    suite.T().Log("Testing resource exhaustion handling")
    
    // Submit multiple resource-intensive processes simultaneously
    resourceIntensiveSpecs := []core.ProcessSpec{}
    for i := 0; i < 5; i++ {
        spec := core.ProcessSpec{
            FuncName:    "resource_intensive_task",
            Args:        []string{"--memory", "8GB", "--cpu", "4"},
            ColonyName:  suite.env.TestColony.Name,
            MaxWaitTime: 300,
            MaxExecTime: 600,
            Priority:    5,
        }
        resourceIntensiveSpecs = append(resourceIntensiveSpecs, spec)
    }
    
    resourceProcessIDs := []string{}
    for _, spec := range resourceIntensiveSpecs {
        process, err := suite.env.ColonyServer.SubmitProcessSpec(spec)
        suite.NoError(err, "Failed to submit resource-intensive process")
        resourceProcessIDs = append(resourceProcessIDs, process.ID)
    }
    
    // Simulate resource exhaustion
    exhaustedState := SystemState{
        QueueDepth:   len(resourceProcessIDs) * 2,
        ComputeUsage: 0.95,
        MemoryUsage:  0.90,
        NetworkUsage: 0.85,
        MasterUsage:  0.80,
    }
    
    suite.env.Offloader.InjectSystemState(exhaustedState)
    
    // Wait for all processes to complete
    for _, processID := range resourceProcessIDs {
        suite.waitForProcessCompletion(processID, 180*time.Second)
    }
    
    // Verify system handled resource exhaustion gracefully
    completedCount := 0
    for _, processID := range resourceProcessIDs {
        process, err := suite.env.ColonyServer.GetProcess(processID)
        suite.NoError(err, "Failed to get resource-intensive process")
        
        if process.State == core.SUCCESSFUL {
            completedCount++
        }
    }
    
    // Should complete at least some processes successfully
    suite.Greater(completedCount, 0, 
        "Should complete at least some processes despite resource exhaustion")
    
    suite.T().Logf("Completed %d/%d resource-intensive processes", 
        completedCount, len(resourceProcessIDs))
}
```

### 2.5 Configuration and Policy Integration

```go
// Test configuration management and policy enforcement
func (suite *ColonyOSIntegrationTestSuite) TestConfigurationAndPolicyIntegration() {
    // Test 1: Dynamic configuration updates
    suite.T().Log("Testing dynamic configuration updates")
    
    // Get initial configuration
    initialWeights := suite.env.Offloader.GetCurrentWeights()
    initialMargins := suite.env.Offloader.GetSafetyMargins()
    
    // Update weights dynamically
    newWeights := AdaptiveWeights{
        QueueDepth:    0.4,  // Increase queue priority
        ProcessorLoad: 0.3,
        NetworkCost:   0.1,
        LatencyCost:   0.1,
        EnergyCost:    0.05,
        PolicyCost:    0.05,
    }
    
    err := suite.env.Offloader.UpdateWeights(newWeights)
    suite.NoError(err, "Failed to update weights")
    
    // Verify weights were updated
    updatedWeights := suite.env.Offloader.GetCurrentWeights()
    suite.Equal(newWeights.QueueDepth, updatedWeights.QueueDepth,
        "Queue depth weight should be updated")
    
    // Test 2: Policy rule enforcement
    suite.T().Log("Testing policy rule enforcement")
    
    // Add security policy
    securityPolicy := PolicyRule{
        Type: HARD,
        Condition: func(p Process, t OffloadTarget) bool {
            return p.SecurityLevel <= t.SecurityLevel
        },
        Priority: 1,
        Description: "Security level compliance",
    }
    
    err = suite.env.Offloader.AddPolicyRule(securityPolicy)
    suite.NoError(err, "Failed to add security policy")
    
    // Submit high-security process
    highSecuritySpec := core.ProcessSpec{
        FuncName:   "classified_computation",
        Args:       []string{"--classification", "top-secret"},
        ColonyName: suite.env.TestColony.Name,
        MaxWaitTime: 300,
        MaxExecTime: 600,
        Priority:   9,
        Env: map[string]string{
            "SECURITY_LEVEL": "4",
        },
    }
    
    secureProcess, err := suite.env.ColonyServer.SubmitProcessSpec(highSecuritySpec)
    suite.NoError(err, "Failed to submit secure process")
    
    // Start offloader
    ctx, cancel := context.WithTimeout(suite.ctx, 60*time.Second)
    defer cancel()
    
    go suite.env.Offloader.Run(ctx)
    
    // Wait for decision
    decision := suite.waitForOffloadDecision(secureProcess.ID, 15*time.Second)
    suite.NotNil(decision, "Should make decision for secure process")
    
    if decision.ShouldOffload {
        // Verify selected target meets security requirements
        suite.GreaterOrEqual(decision.Target.SecurityLevel, 4,
            "Selected target should meet security requirements")
        suite.Empty(decision.PolicyViolations,
            "Should not have policy violations")
    }
    
    // Wait for completion
    suite.waitForProcessCompletion(secureProcess.ID, 60*time.Second)
    
    // Verify no policy violations occurred
    outcome := suite.getOffloadOutcome(secureProcess.ID)
    if outcome != nil {
        suite.False(outcome.PolicyViolation,
            "Should not violate security policy")
    }
    
    // Test 3: Data locality requirements
    suite.T().Log("Testing data locality requirements")
    
    // Add data locality policy
    localityPolicy := PolicyRule{
        Type: HARD,
        Condition: func(p Process, t OffloadTarget) bool {
            if p.DataSensitivity >= 3 {
                return t.DataJurisdiction == "domestic"
            }
            return true
        },
        Priority: 1,
        Description: "Data locality compliance",
    }
    
    err = suite.env.Offloader.AddPolicyRule(localityPolicy)
    suite.NoError(err, "Failed to add locality policy")
    
    // Submit process with sensitive data
    sensitiveDataSpec := core.ProcessSpec{
        FuncName:   "personal_data_processing",
        Args:       []string{"--dataset", "customer_records.db"},
        ColonyName: suite.env.TestColony.Name,
        MaxWaitTime: 300,
        MaxExecTime: 600,
        Priority:   6,
        Env: map[string]string{
            "DATA_SENSITIVITY": "4",
        },
    }
    
    sensitiveProcess, err := suite.env.ColonyServer.SubmitProcessSpec(sensitiveDataSpec)
    suite.NoError(err, "Failed to submit sensitive data process")
    
    // Wait for decision
    sensitiveDecision := suite.waitForOffloadDecision(sensitiveProcess.ID, 15*time.Second)
    suite.NotNil(sensitiveDecision, "Should make decision for sensitive process")
    
    if sensitiveDecision.ShouldOffload {
        // Verify target is in domestic jurisdiction
        suite.Equal("domestic", sensitiveDecision.Target.DataJurisdiction,
            "Should select domestic target for sensitive data")
    }
    
    // Complete process
    suite.waitForProcessCompletion(sensitiveProcess.ID, 60*time.Second)
    
    // Verify compliance
    sensitiveOutcome := suite.getOffloadOutcome(sensitiveProcess.ID)
    if sensitiveOutcome != nil {
        suite.False(sensitiveOutcome.PolicyViolation,
            "Should not violate data locality policy")
    }
}
```

### 2.6 Helper Methods

```go
// Helper methods for integration tests

func (suite *ColonyOSIntegrationTestSuite) waitForOffloadDecision(
    processID string, 
    timeout time.Duration,
) *OffloadDecision {
    
    ctx, cancel := context.WithTimeout(suite.ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            decision := suite.env.Offloader.GetDecisionForProcess(processID)
            if decision != nil {
                return decision
            }
        }
    }
}

func (suite *ColonyOSIntegrationTestSuite) waitForProcessCompletion(
    processID string,
    timeout time.Duration,
) {
    
    ctx, cancel := context.WithTimeout(suite.ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            suite.Fail("Process completion timeout", "Process %s did not complete within %v", processID, timeout)
            return
        case <-ticker.C:
            process, err := suite.env.ColonyServer.GetProcess(processID)
            suite.NoError(err, "Failed to get process status")
            
            if process.State == core.SUCCESSFUL || process.State == core.FAILED {
                return
            }
        }
    }
}

func (suite *ColonyOSIntegrationTestSuite) waitForExecutionOutcome(
    processID string,
    timeout time.Duration,
) *OffloadOutcome {
    
    ctx, cancel := context.WithTimeout(suite.ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            outcome := suite.getOffloadOutcome(processID)
            if outcome != nil {
                return outcome
            }
        }
    }
}

func (suite *ColonyOSIntegrationTestSuite) getOffloadOutcome(processID string) *OffloadOutcome {
    outcomes := suite.env.Offloader.GetOutcomesForProcess(processID)
    if len(outcomes) > 0 {
        return &outcomes[len(outcomes)-1] // Return most recent outcome
    }
    return nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

These integration tests provide comprehensive validation of the adaptive offloading algorithm's integration with ColonyOS, covering process lifecycle management, executor discovery, error handling, configuration management, and policy enforcement. They ensure the algorithm works correctly within the ColonyOS ecosystem and handles real-world scenarios effectively.

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "Create detailed algorithm specification documentation", "status": "completed", "activeForm": "Creating detailed algorithm specification documentation"}, {"content": "Design comprehensive test suite for core components", "status": "completed", "activeForm": "Designing comprehensive test suite for core components"}, {"content": "Define test scenarios for adaptive learning behavior", "status": "completed", "activeForm": "Defining test scenarios for adaptive learning behavior"}, {"content": "Create integration tests for ColonyOS compatibility", "status": "completed", "activeForm": "Creating integration tests for ColonyOS compatibility"}, {"content": "Implement core algorithm components using TDD", "status": "pending", "activeForm": "Implementing core algorithm components using TDD"}, {"content": "Implement learning and adaptation mechanisms", "status": "pending", "activeForm": "Implementing learning and adaptation mechanisms"}, {"content": "Create performance benchmarking tests", "status": "pending", "activeForm": "Creating performance benchmarking tests"}]