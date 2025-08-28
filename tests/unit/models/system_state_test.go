package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SystemState test requirements:
// 1. All utilization metrics must be normalized to [0.0, 1.0] range
// 2. State capture must complete within 100ms
// 3. SystemState must be completely observable and deterministic

type SystemStateTestSuite struct {
	suite.Suite
}

// Test that all utilization metrics are properly normalized
func (suite *SystemStateTestSuite) TestUtilizationMetricsNormalization() {
	testCases := []struct {
		name          string
		inputState    SystemState
		expectValid   bool
		expectMessage string
	}{
		{
			name: "valid_normal_load",
			inputState: SystemState{
				ComputeUsage:  0.5,
				MemoryUsage:   0.6,
				DiskUsage:     0.3,
				NetworkUsage:  0.4,
				MasterUsage:   0.2,
				QueueDepth:    10,
				QueueThreshold: 20,
			},
			expectValid: true,
		},
		{
			name: "valid_high_load",
			inputState: SystemState{
				ComputeUsage:  0.95,
				MemoryUsage:   0.99,
				DiskUsage:     0.85,
				NetworkUsage:  0.90,
				MasterUsage:   0.88,
				QueueDepth:    100,
				QueueThreshold: 20,
			},
			expectValid: true,
		},
		{
			name: "valid_idle_system",
			inputState: SystemState{
				ComputeUsage:  0.0,
				MemoryUsage:   0.1,
				DiskUsage:     0.05,
				NetworkUsage:  0.0,
				MasterUsage:   0.01,
				QueueDepth:    0,
				QueueThreshold: 20,
			},
			expectValid: true,
		},
		{
			name: "invalid_compute_usage_too_high",
			inputState: SystemState{
				ComputeUsage:  1.1, // Invalid: > 1.0
				MemoryUsage:   0.5,
				DiskUsage:     0.3,
				NetworkUsage:  0.4,
				MasterUsage:   0.2,
			},
			expectValid:   false,
			expectMessage: "ComputeUsage must be in range [0.0, 1.0]",
		},
		{
			name: "invalid_negative_memory_usage",
			inputState: SystemState{
				ComputeUsage:  0.5,
				MemoryUsage:   -0.1, // Invalid: < 0.0
				DiskUsage:     0.3,
				NetworkUsage:  0.4,
				MasterUsage:   0.2,
			},
			expectValid:   false,
			expectMessage: "MemoryUsage must be in range [0.0, 1.0]",
		},
		{
			name: "invalid_multiple_out_of_range",
			inputState: SystemState{
				ComputeUsage:  1.5,  // Invalid
				MemoryUsage:   -0.2, // Invalid
				DiskUsage:     2.0,  // Invalid
				NetworkUsage:  0.4,
				MasterUsage:   0.2,
			},
			expectValid:   false,
			expectMessage: "Multiple utilization metrics out of range",
		},
		{
			name: "edge_case_exactly_one",
			inputState: SystemState{
				ComputeUsage:  1.0, // Valid: exactly 1.0
				MemoryUsage:   1.0,
				DiskUsage:     1.0,
				NetworkUsage:  1.0,
				MasterUsage:   1.0,
			},
			expectValid: true,
		},
		{
			name: "edge_case_exactly_zero",
			inputState: SystemState{
				ComputeUsage:  0.0, // Valid: exactly 0.0
				MemoryUsage:   0.0,
				DiskUsage:     0.0,
				NetworkUsage:  0.0,
				MasterUsage:   0.0,
			},
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.inputState.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, "Valid state should not return error")
			} else {
				assert.Error(suite.T(), err, "Invalid state should return error")
				if tc.expectMessage != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectMessage)
				}
			}
		})
	}
}

// Test state capture performance requirement (must complete within 100ms)
func (suite *SystemStateTestSuite) TestStateCapturePerformance() {
	// Create a state collector (mocked for testing)
	collector := NewSystemStateCollector()

	// Run multiple capture iterations to ensure consistency
	iterations := 100
	captureTimeouts := 0

	for i := 0; i < iterations; i++ {
		start := time.Now()
		state, err := collector.CaptureState()
		duration := time.Since(start)

		require.NoError(suite.T(), err, "State capture should not error")
		require.NotNil(suite.T(), state, "Captured state should not be nil")

		// Verify capture completed within 100ms requirement
		if duration > 100*time.Millisecond {
			captureTimeouts++
			suite.T().Logf("Capture %d exceeded 100ms: %v", i, duration)
		}

		// Verify all metrics are within range
		assert.True(suite.T(), state.ComputeUsage >= 0.0 && state.ComputeUsage <= 1.0)
		assert.True(suite.T(), state.MemoryUsage >= 0.0 && state.MemoryUsage <= 1.0)
		assert.True(suite.T(), state.DiskUsage >= 0.0 && state.DiskUsage <= 1.0)
		assert.True(suite.T(), state.NetworkUsage >= 0.0 && state.NetworkUsage <= 1.0)
		assert.True(suite.T(), state.MasterUsage >= 0.0 && state.MasterUsage <= 1.0)
	}

	// Allow up to 5% of captures to exceed timeout (for system variance)
	assert.LessOrEqual(suite.T(), captureTimeouts, iterations/20,
		"More than 5%% of captures exceeded 100ms timeout")
}

// Test that SystemState is completely observable
func (suite *SystemStateTestSuite) TestStateObservability() {
	state := SystemState{
		QueueDepth:        25,
		QueueThreshold:    20,
		QueueWaitTime:     5 * time.Second,
		QueueThroughput:   10.5,
		ComputeUsage:      0.75,
		MemoryUsage:       0.60,
		DiskUsage:         0.40,
		NetworkUsage:      0.30,
		MasterUsage:       0.50,
		ActiveConnections: 150,
		Timestamp:         time.Now(),
		TimeSlot:          14, // 2 PM
		DayOfWeek:         3,  // Wednesday
	}

	// Test all fields are accessible
	assert.Equal(suite.T(), 25, state.QueueDepth)
	assert.Equal(suite.T(), 20, state.QueueThreshold)
	assert.Equal(suite.T(), 5*time.Second, state.QueueWaitTime)
	assert.Equal(suite.T(), 10.5, state.QueueThroughput)
	assert.Equal(suite.T(), 0.75, state.ComputeUsage)
	assert.Equal(suite.T(), 0.60, state.MemoryUsage)
	assert.Equal(suite.T(), 0.40, state.DiskUsage)
	assert.Equal(suite.T(), 0.30, state.NetworkUsage)
	assert.Equal(suite.T(), 0.50, state.MasterUsage)
	assert.Equal(suite.T(), 150, state.ActiveConnections)
	assert.NotZero(suite.T(), state.Timestamp)
	assert.Equal(suite.T(), 14, state.TimeSlot)
	assert.Equal(suite.T(), 3, state.DayOfWeek)

	// Test state can be serialized/deserialized (observable)
	serialized := state.Serialize()
	assert.NotEmpty(suite.T(), serialized)

	deserialized, err := DeserializeSystemState(serialized)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), state, deserialized)
}

// Test that SystemState is deterministic
func (suite *SystemStateTestSuite) TestStateDeterminism() {
	// Given same inputs, state should be identical
	input := SystemStateInput{
		CPUCores:          8,
		CPUUsedCores:      6,
		TotalMemory:       16 * 1024 * 1024 * 1024, // 16 GB
		UsedMemory:        10 * 1024 * 1024 * 1024, // 10 GB
		TotalDisk:         1024 * 1024 * 1024 * 1024, // 1 TB
		UsedDisk:          512 * 1024 * 1024 * 1024, // 512 GB
		NetworkBandwidth:  1000 * 1024 * 1024, // 1 Gbps
		NetworkUsed:       300 * 1024 * 1024, // 300 Mbps
		QueueLength:       15,
		ActiveConnections: 25,
	}

	// Create multiple states from same input
	states := []SystemState{}
	for i := 0; i < 10; i++ {
		state := CreateSystemStateFromInput(input)
		states = append(states, state)
	}

	// All states should be identical (deterministic)
	for i := 1; i < len(states); i++ {
		// Compare normalized values
		assert.Equal(suite.T(), states[0].ComputeUsage, states[i].ComputeUsage,
			"ComputeUsage should be deterministic")
		assert.Equal(suite.T(), states[0].MemoryUsage, states[i].MemoryUsage,
			"MemoryUsage should be deterministic")
		assert.Equal(suite.T(), states[0].DiskUsage, states[i].DiskUsage,
			"DiskUsage should be deterministic")
		assert.Equal(suite.T(), states[0].NetworkUsage, states[i].NetworkUsage,
			"NetworkUsage should be deterministic")
		assert.Equal(suite.T(), states[0].QueueDepth, states[i].QueueDepth,
			"QueueDepth should be deterministic")
	}
}

// Test edge cases for queue metrics
func (suite *SystemStateTestSuite) TestQueueMetricsEdgeCases() {
	testCases := []struct {
		name        string
		state       SystemState
		expectValid bool
	}{
		{
			name: "empty_queue",
			state: SystemState{
				QueueDepth:      0,
				QueueThreshold:  20,
				QueueWaitTime:   0,
				QueueThroughput: 0.0,
			},
			expectValid: true,
		},
		{
			name: "queue_at_threshold",
			state: SystemState{
				QueueDepth:      20,
				QueueThreshold:  20,
				QueueWaitTime:   10 * time.Second,
				QueueThroughput: 5.0,
			},
			expectValid: true,
		},
		{
			name: "queue_over_threshold",
			state: SystemState{
				QueueDepth:      50,
				QueueThreshold:  20,
				QueueWaitTime:   30 * time.Second,
				QueueThroughput: 2.0,
			},
			expectValid: true,
		},
		{
			name: "negative_queue_depth_invalid",
			state: SystemState{
				QueueDepth:      -1,
				QueueThreshold:  20,
				QueueWaitTime:   5 * time.Second,
				QueueThroughput: 3.0,
			},
			expectValid: false,
		},
		{
			name: "negative_throughput_invalid",
			state: SystemState{
				QueueDepth:      10,
				QueueThreshold:  20,
				QueueWaitTime:   5 * time.Second,
				QueueThroughput: -1.0,
			},
			expectValid: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Set valid utilization metrics
			tc.state.ComputeUsage = 0.5
			tc.state.MemoryUsage = 0.5
			tc.state.DiskUsage = 0.5
			tc.state.NetworkUsage = 0.5
			tc.state.MasterUsage = 0.5

			err := tc.state.Validate()
			if tc.expectValid {
				assert.NoError(suite.T(), err)
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// Test temporal context fields
func (suite *SystemStateTestSuite) TestTemporalContext() {
	now := time.Now()
	state := SystemState{
		Timestamp: now,
		TimeSlot:  now.Hour(),
		DayOfWeek: int(now.Weekday()),
	}

	// Verify time slot is in valid range
	assert.GreaterOrEqual(suite.T(), state.TimeSlot, 0)
	assert.LessOrEqual(suite.T(), state.TimeSlot, 23)

	// Verify day of week is in valid range
	assert.GreaterOrEqual(suite.T(), state.DayOfWeek, 0)
	assert.LessOrEqual(suite.T(), state.DayOfWeek, 6)

	// Test all hours of the day
	for hour := 0; hour < 24; hour++ {
		state.TimeSlot = hour
		err := state.ValidateTemporalContext()
		assert.NoError(suite.T(), err, "Hour %d should be valid", hour)
	}

	// Test invalid hours
	state.TimeSlot = -1
	err := state.ValidateTemporalContext()
	assert.Error(suite.T(), err, "Negative hour should be invalid")

	state.TimeSlot = 24
	err = state.ValidateTemporalContext()
	assert.Error(suite.T(), err, "Hour 24 should be invalid")

	// Test all days of week
	for day := 0; day < 7; day++ {
		state.DayOfWeek = day
		state.TimeSlot = 12 // Reset to valid hour
		err := state.ValidateTemporalContext()
		assert.NoError(suite.T(), err, "Day %d should be valid", day)
	}

	// Test invalid days
	state.DayOfWeek = -1
	err = state.ValidateTemporalContext()
	assert.Error(suite.T(), err, "Negative day should be invalid")

	state.DayOfWeek = 7
	err = state.ValidateTemporalContext()
	assert.Error(suite.T(), err, "Day 7 should be invalid")
}

// Test system state snapshot consistency
func (suite *SystemStateTestSuite) TestSnapshotConsistency() {
	// Create a rapidly changing system state
	collector := NewSystemStateCollector()
	
	// Take multiple snapshots in quick succession
	snapshots := []SystemState{}
	for i := 0; i < 10; i++ {
		snapshot, err := collector.CaptureState()
		require.NoError(suite.T(), err)
		snapshots = append(snapshots, snapshot)
		time.Sleep(10 * time.Millisecond)
	}

	// Verify temporal ordering
	for i := 1; i < len(snapshots); i++ {
		assert.True(suite.T(), snapshots[i].Timestamp.After(snapshots[i-1].Timestamp),
			"Timestamps should be in increasing order")
	}

	// Verify metrics are realistic (no wild jumps)
	for i := 1; i < len(snapshots); i++ {
		cpuDelta := absFloat64(snapshots[i].ComputeUsage - snapshots[i-1].ComputeUsage)
		memDelta := absFloat64(snapshots[i].MemoryUsage - snapshots[i-1].MemoryUsage)
		
		// CPU and memory shouldn't change by more than 50% in 10ms
		assert.LessOrEqual(suite.T(), cpuDelta, 0.5,
			"CPU usage shouldn't jump more than 50%% in 10ms")
		assert.LessOrEqual(suite.T(), memDelta, 0.5,
			"Memory usage shouldn't jump more than 50%% in 10ms")
	}
}

func TestSystemStateTestSuite(t *testing.T) {
	suite.Run(t, new(SystemStateTestSuite))
}

// Helper functions

func absFloat64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}