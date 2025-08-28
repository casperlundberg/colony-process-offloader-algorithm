package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Process test requirements:
// 1. Process model must support both simple and DAG-based workloads
// 2. All size fields must be non-negative
// 3. Priority must be in range [1,10]
// 4. EstimatedDuration must be > 0 for valid processes

type ProcessTestSuite struct {
	suite.Suite
}

// Test that process supports both simple and DAG-based workloads
func (suite *ProcessTestSuite) TestProcessWorkloadTypes() {
	// Test simple process without DAG
	simpleProcess := Process{
		ID:                "simple-1",
		Type:              "compute",
		Priority:          5,
		CPURequirement:    2.0,
		MemoryRequirement: 4 * 1024 * 1024 * 1024, // 4GB
		EstimatedDuration: 30 * time.Second,
		HasDAG:            false,
		DAG:               nil,
	}

	err := simpleProcess.Validate()
	assert.NoError(suite.T(), err, "Simple process should be valid")
	assert.False(suite.T(), simpleProcess.HasDAG)
	assert.Nil(suite.T(), simpleProcess.DAG)

	// Test DAG-based process
	dag := &DAG{
		ID: "dag-1",
		Stages: []Stage{
			{
				ID:           "stage-1",
				Name:         "data-ingestion",
				InputSize:    1024 * 1024, // 1MB
				OutputSize:   512 * 1024,  // 512KB
				Dependencies: []string{},
			},
			{
				ID:           "stage-2",
				Name:         "processing",
				InputSize:    512 * 1024,
				OutputSize:   256 * 1024,
				Dependencies: []string{"stage-1"},
			},
			{
				ID:           "stage-3",
				Name:         "aggregation",
				InputSize:    256 * 1024,
				OutputSize:   128 * 1024,
				Dependencies: []string{"stage-2"},
			},
		},
	}

	dagProcess := Process{
		ID:                "dag-process-1",
		Type:              "pipeline",
		Priority:          7,
		CPURequirement:    4.0,
		MemoryRequirement: 8 * 1024 * 1024 * 1024, // 8GB
		EstimatedDuration: 120 * time.Second,
		HasDAG:            true,
		DAG:               dag,
	}

	err = dagProcess.Validate()
	assert.NoError(suite.T(), err, "DAG process should be valid")
	assert.True(suite.T(), dagProcess.HasDAG)
	assert.NotNil(suite.T(), dagProcess.DAG)
	assert.Len(suite.T(), dagProcess.DAG.Stages, 3)

	// Test process with inconsistent DAG flag
	inconsistentProcess := Process{
		ID:                "inconsistent-1",
		Priority:          5,
		EstimatedDuration: 30 * time.Second,
		HasDAG:            true,  // Says it has DAG
		DAG:               nil,   // But DAG is nil
	}

	err = inconsistentProcess.Validate()
	assert.Error(suite.T(), err, "Process with HasDAG=true but nil DAG should be invalid")
	assert.Contains(suite.T(), err.Error(), "DAG")

	// Test process with DAG but flag is false
	inconsistentProcess2 := Process{
		ID:                "inconsistent-2",
		Priority:          5,
		EstimatedDuration: 30 * time.Second,
		HasDAG:            false, // Says no DAG
		DAG:               dag,   // But has DAG
	}

	err = inconsistentProcess2.Validate()
	assert.Error(suite.T(), err, "Process with HasDAG=false but non-nil DAG should be invalid")
}

// Test that all size fields must be non-negative
func (suite *ProcessTestSuite) TestSizeFieldsNonNegative() {
	testCases := []struct {
		name          string
		process       Process
		expectValid   bool
		expectMessage string
	}{
		{
			name: "all_sizes_valid",
			process: Process{
				ID:                "valid-sizes",
				Priority:          5,
				InputSize:         1024 * 1024,     // 1MB
				OutputSize:        512 * 1024,       // 512KB
				MemoryRequirement: 1024 * 1024 * 1024, // 1GB
				DiskRequirement:   10 * 1024 * 1024 * 1024, // 10GB
				EstimatedDuration: 30 * time.Second,
			},
			expectValid: true,
		},
		{
			name: "zero_sizes_valid",
			process: Process{
				ID:                "zero-sizes",
				Priority:          5,
				InputSize:         0, // Valid: can have no input
				OutputSize:        0, // Valid: can produce no output
				MemoryRequirement: 0, // Valid: minimal memory
				DiskRequirement:   0, // Valid: no disk needed
				EstimatedDuration: 1 * time.Second,
			},
			expectValid: true,
		},
		{
			name: "negative_input_size",
			process: Process{
				ID:                "negative-input",
				Priority:          5,
				InputSize:         -1024, // Invalid
				OutputSize:        512,
				EstimatedDuration: 30 * time.Second,
			},
			expectValid:   false,
			expectMessage: "InputSize must be non-negative",
		},
		{
			name: "negative_output_size",
			process: Process{
				ID:                "negative-output",
				Priority:          5,
				InputSize:         1024,
				OutputSize:        -512, // Invalid
				EstimatedDuration: 30 * time.Second,
			},
			expectValid:   false,
			expectMessage: "OutputSize must be non-negative",
		},
		{
			name: "negative_memory_requirement",
			process: Process{
				ID:                "negative-memory",
				Priority:          5,
				MemoryRequirement: -1024, // Invalid
				EstimatedDuration: 30 * time.Second,
			},
			expectValid:   false,
			expectMessage: "MemoryRequirement must be non-negative",
		},
		{
			name: "negative_disk_requirement",
			process: Process{
				ID:                "negative-disk",
				Priority:          5,
				DiskRequirement:   -1024, // Invalid
				EstimatedDuration: 30 * time.Second,
			},
			expectValid:   false,
			expectMessage: "DiskRequirement must be non-negative",
		},
		{
			name: "multiple_negative_sizes",
			process: Process{
				ID:                "multiple-negative",
				Priority:          5,
				InputSize:         -1024,
				OutputSize:        -512,
				MemoryRequirement: -2048,
				DiskRequirement:   -4096,
				EstimatedDuration: 30 * time.Second,
			},
			expectValid:   false,
			expectMessage: "must be non-negative",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.process.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, "Valid process should not return error")
			} else {
				assert.Error(suite.T(), err, "Invalid process should return error")
				if tc.expectMessage != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectMessage)
				}
			}
		})
	}
}

// Test that priority must be in range [1,10]
func (suite *ProcessTestSuite) TestPriorityRange() {
	testCases := []struct {
		name        string
		priority    int
		expectValid bool
	}{
		// Valid priorities
		{name: "priority_1_min", priority: 1, expectValid: true},
		{name: "priority_2", priority: 2, expectValid: true},
		{name: "priority_5_mid", priority: 5, expectValid: true},
		{name: "priority_9", priority: 9, expectValid: true},
		{name: "priority_10_max", priority: 10, expectValid: true},
		
		// Invalid priorities
		{name: "priority_0_invalid", priority: 0, expectValid: false},
		{name: "priority_negative", priority: -1, expectValid: false},
		{name: "priority_11_invalid", priority: 11, expectValid: false},
		{name: "priority_100_invalid", priority: 100, expectValid: false},
		{name: "priority_very_negative", priority: -100, expectValid: false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			process := Process{
				ID:                "test-priority",
				Priority:          tc.priority,
				EstimatedDuration: 30 * time.Second, // Required field
			}

			err := process.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, 
					"Priority %d should be valid", tc.priority)
			} else {
				assert.Error(suite.T(), err, 
					"Priority %d should be invalid", tc.priority)
				assert.Contains(suite.T(), err.Error(), "Priority")
				assert.Contains(suite.T(), err.Error(), "[1,10]")
			}
		})
	}
}

// Test that EstimatedDuration must be > 0 for valid processes
func (suite *ProcessTestSuite) TestEstimatedDurationRequirement() {
	testCases := []struct {
		name              string
		estimatedDuration time.Duration
		expectValid       bool
	}{
		// Valid durations
		{name: "duration_1ms", estimatedDuration: 1 * time.Millisecond, expectValid: true},
		{name: "duration_1s", estimatedDuration: 1 * time.Second, expectValid: true},
		{name: "duration_30s", estimatedDuration: 30 * time.Second, expectValid: true},
		{name: "duration_5min", estimatedDuration: 5 * time.Minute, expectValid: true},
		{name: "duration_1hour", estimatedDuration: 1 * time.Hour, expectValid: true},
		{name: "duration_24hours", estimatedDuration: 24 * time.Hour, expectValid: true},
		
		// Invalid durations
		{name: "duration_zero", estimatedDuration: 0, expectValid: false},
		{name: "duration_negative", estimatedDuration: -1 * time.Second, expectValid: false},
		{name: "duration_very_negative", estimatedDuration: -100 * time.Hour, expectValid: false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			process := Process{
				ID:                "test-duration",
				Priority:          5, // Valid priority
				EstimatedDuration: tc.estimatedDuration,
			}

			err := process.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, 
					"Duration %v should be valid", tc.estimatedDuration)
			} else {
				assert.Error(suite.T(), err, 
					"Duration %v should be invalid", tc.estimatedDuration)
				assert.Contains(suite.T(), err.Error(), "EstimatedDuration")
				assert.Contains(suite.T(), err.Error(), "> 0")
			}
		})
	}
}

// Test resource requirements validation
func (suite *ProcessTestSuite) TestResourceRequirements() {
	testCases := []struct {
		name           string
		cpuRequirement float64
		expectValid    bool
	}{
		// Valid CPU requirements
		{name: "cpu_0.1", cpuRequirement: 0.1, expectValid: true},
		{name: "cpu_0.5", cpuRequirement: 0.5, expectValid: true},
		{name: "cpu_1.0", cpuRequirement: 1.0, expectValid: true},
		{name: "cpu_2.5", cpuRequirement: 2.5, expectValid: true},
		{name: "cpu_8.0", cpuRequirement: 8.0, expectValid: true},
		{name: "cpu_16.0", cpuRequirement: 16.0, expectValid: true},
		{name: "cpu_32.0", cpuRequirement: 32.0, expectValid: true},
		{name: "cpu_0_valid", cpuRequirement: 0, expectValid: true}, // 0 means no specific requirement
		
		// Invalid CPU requirements
		{name: "cpu_negative", cpuRequirement: -1.0, expectValid: false},
		{name: "cpu_very_negative", cpuRequirement: -100.0, expectValid: false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			process := Process{
				ID:                "test-cpu",
				Priority:          5,
				CPURequirement:    tc.cpuRequirement,
				EstimatedDuration: 30 * time.Second,
			}

			err := process.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, 
					"CPU requirement %f should be valid", tc.cpuRequirement)
			} else {
				assert.Error(suite.T(), err, 
					"CPU requirement %f should be invalid", tc.cpuRequirement)
				assert.Contains(suite.T(), err.Error(), "CPURequirement")
			}
		})
	}
}

// Test security and sensitivity fields
func (suite *ProcessTestSuite) TestSecurityFields() {
	testCases := []struct {
		name             string
		dataSensitivity  int
		securityLevel    int
		expectValid      bool
	}{
		// Valid security levels
		{name: "security_0_0", dataSensitivity: 0, securityLevel: 0, expectValid: true},
		{name: "security_3_3", dataSensitivity: 3, securityLevel: 3, expectValid: true},
		{name: "security_5_5", dataSensitivity: 5, securityLevel: 5, expectValid: true},
		
		// Invalid security levels
		{name: "sensitivity_negative", dataSensitivity: -1, securityLevel: 3, expectValid: false},
		{name: "sensitivity_too_high", dataSensitivity: 6, securityLevel: 3, expectValid: false},
		{name: "security_negative", dataSensitivity: 3, securityLevel: -1, expectValid: false},
		{name: "security_too_high", dataSensitivity: 3, securityLevel: 6, expectValid: false},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			process := Process{
				ID:                "test-security",
				Priority:          5,
				DataSensitivity:   tc.dataSensitivity,
				SecurityLevel:     tc.securityLevel,
				EstimatedDuration: 30 * time.Second,
			}

			err := process.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err)
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// Test deadline constraints
func (suite *ProcessTestSuite) TestDeadlineConstraints() {
	testCases := []struct {
		name              string
		estimatedDuration time.Duration
		maxDuration       time.Duration
		expectValid       bool
	}{
		{
			name:              "deadline_after_estimate",
			estimatedDuration: 30 * time.Second,
			maxDuration:       60 * time.Second,
			expectValid:       true,
		},
		{
			name:              "deadline_equals_estimate",
			estimatedDuration: 30 * time.Second,
			maxDuration:       30 * time.Second,
			expectValid:       true,
		},
		{
			name:              "deadline_before_estimate_warning",
			estimatedDuration: 60 * time.Second,
			maxDuration:       30 * time.Second,
			expectValid:       true, // Valid but should log warning
		},
		{
			name:              "deadline_zero_means_no_limit",
			estimatedDuration: 30 * time.Second,
			maxDuration:       0, // No deadline
			expectValid:       true,
		},
		{
			name:              "deadline_negative_invalid",
			estimatedDuration: 30 * time.Second,
			maxDuration:       -1 * time.Second,
			expectValid:       false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			process := Process{
				ID:                "test-deadline",
				Priority:          5,
				EstimatedDuration: tc.estimatedDuration,
				MaxDuration:       tc.maxDuration,
			}

			err := process.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err)
				
				// Check for warning case
				if tc.maxDuration > 0 && tc.maxDuration < tc.estimatedDuration {
					warnings := process.GetWarnings()
					assert.Contains(suite.T(), warnings, "deadline")
				}
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// Test process state transitions
func (suite *ProcessTestSuite) TestProcessStateTransitions() {
	process := Process{
		ID:                "state-test",
		Priority:          5,
		EstimatedDuration: 30 * time.Second,
		Status:            QUEUED,
	}

	// Valid transitions from QUEUED
	assert.True(suite.T(), process.CanTransitionTo(ASSIGNED))
	assert.True(suite.T(), process.CanTransitionTo(CANCELLED))
	assert.False(suite.T(), process.CanTransitionTo(COMPLETED))

	// Move to ASSIGNED
	err := process.TransitionTo(ASSIGNED)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), ASSIGNED, process.Status)

	// Valid transitions from ASSIGNED
	assert.True(suite.T(), process.CanTransitionTo(EXECUTING))
	assert.True(suite.T(), process.CanTransitionTo(FAILED))
	assert.False(suite.T(), process.CanTransitionTo(QUEUED))

	// Move to EXECUTING
	err = process.TransitionTo(EXECUTING)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), EXECUTING, process.Status)
	assert.NotZero(suite.T(), process.StartTime)

	// Valid transitions from EXECUTING
	assert.True(suite.T(), process.CanTransitionTo(COMPLETED))
	assert.True(suite.T(), process.CanTransitionTo(FAILED))
	assert.False(suite.T(), process.CanTransitionTo(QUEUED))

	// Complete the process
	err = process.TransitionTo(COMPLETED)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), COMPLETED, process.Status)

	// No transitions from terminal state
	assert.False(suite.T(), process.CanTransitionTo(QUEUED))
	assert.False(suite.T(), process.CanTransitionTo(ASSIGNED))
	assert.False(suite.T(), process.CanTransitionTo(EXECUTING))
}

// Test dependency validation
func (suite *ProcessTestSuite) TestDependencyValidation() {
	// Process with no dependencies
	process1 := Process{
		ID:                "process-1",
		Priority:          5,
		EstimatedDuration: 30 * time.Second,
		Dependencies:      []string{},
	}
	
	err := process1.Validate()
	assert.NoError(suite.T(), err)

	// Process with valid dependencies
	process2 := Process{
		ID:                "process-2",
		Priority:          5,
		EstimatedDuration: 30 * time.Second,
		Dependencies:      []string{"process-1"},
	}
	
	err = process2.Validate()
	assert.NoError(suite.T(), err)

	// Process with self-dependency (invalid)
	process3 := Process{
		ID:                "process-3",
		Priority:          5,
		EstimatedDuration: 30 * time.Second,
		Dependencies:      []string{"process-3"}, // Self-dependency
	}
	
	err = process3.Validate()
	assert.Error(suite.T(), err, "Self-dependency should be invalid")
	assert.Contains(suite.T(), err.Error(), "self-dependency")

	// Process with duplicate dependencies
	process4 := Process{
		ID:                "process-4",
		Priority:          5,
		EstimatedDuration: 30 * time.Second,
		Dependencies:      []string{"process-1", "process-2", "process-1"}, // Duplicate
	}
	
	err = process4.Validate()
	assert.Error(suite.T(), err, "Duplicate dependencies should be invalid")
	assert.Contains(suite.T(), err.Error(), "duplicate")
}

// Test special process flags
func (suite *ProcessTestSuite) TestSpecialProcessFlags() {
	// Real-time process
	rtProcess := Process{
		ID:                "rt-process",
		Priority:          9, // High priority typical for real-time
		EstimatedDuration: 100 * time.Millisecond,
		MaxDuration:       500 * time.Millisecond, // Strict deadline
		RealTime:          true,
	}

	err := rtProcess.Validate()
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), rtProcess.RealTime)
	
	// Real-time processes should have high priority
	if rtProcess.RealTime && rtProcess.Priority < 7 {
		warnings := rtProcess.GetWarnings()
		assert.Contains(suite.T(), warnings, "priority")
	}

	// Safety-critical process
	safetyCriticalProcess := Process{
		ID:                "safety-critical",
		Priority:          10, // Maximum priority
		EstimatedDuration: 50 * time.Millisecond,
		SafetyCritical:    true,
		LocalityRequired:  true, // Usually safety-critical stays local
	}

	err = safetyCriticalProcess.Validate()
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), safetyCriticalProcess.SafetyCritical)
	assert.True(suite.T(), safetyCriticalProcess.LocalityRequired)

	// Safety-critical should have maximum priority
	if safetyCriticalProcess.SafetyCritical && safetyCriticalProcess.Priority < 9 {
		warnings := safetyCriticalProcess.GetWarnings()
		assert.Contains(suite.T(), warnings, "priority")
	}
}

func TestProcessTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessTestSuite))
}