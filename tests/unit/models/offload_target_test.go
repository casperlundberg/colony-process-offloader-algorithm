package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// OffloadTarget test requirements:
// 1. Target model must support heterogeneous infrastructure
// 2. All capacity metrics must be non-negative
// 3. Scores must be in [0.0, 1.0] range
// 4. Latency must be measurable and recent (< 60s old)

type OffloadTargetTestSuite struct {
	suite.Suite
}

// Test heterogeneous infrastructure support
func (suite *OffloadTargetTestSuite) TestHeterogeneousInfrastructure() {
	testCases := []struct {
		name         string
		targetType   TargetType
		capabilities []string
		expectValid  bool
	}{
		{
			name:         "edge_server",
			targetType:   EDGE,
			capabilities: []string{"low_latency", "compute_optimized"},
			expectValid:  true,
		},
		{
			name:         "private_cloud",
			targetType:   PRIVATE_CLOUD,
			capabilities: []string{"high_capacity", "secure", "cost_optimized"},
			expectValid:  true,
		},
		{
			name:         "public_cloud",
			targetType:   PUBLIC_CLOUD,
			capabilities: []string{"scalable", "gpu_accelerated", "ml_optimized"},
			expectValid:  true,
		},
		{
			name:         "local_executor",
			targetType:   LOCAL,
			capabilities: []string{"always_available", "no_network_cost"},
			expectValid:  true,
		},
		{
			name:         "hybrid_cloud",
			targetType:   HYBRID_CLOUD,
			capabilities: []string{"burst_capacity", "data_locality"},
			expectValid:  true,
		},
		{
			name:         "fog_computing",
			targetType:   FOG,
			capabilities: []string{"ultra_low_latency", "iot_optimized"},
			expectValid:  true,
		},
		{
			name:         "invalid_type",
			targetType:   "invalid",
			capabilities: []string{"test"},
			expectValid:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			target := OffloadTarget{
				ID:               tc.name,
				Type:             tc.targetType,
				Capabilities:     tc.capabilities,
				TotalCapacity:    8.0,
				AvailableCapacity: 4.0,
				MemoryTotal:      16 * 1024 * 1024 * 1024, // 16GB
				MemoryAvailable:  8 * 1024 * 1024 * 1024,  // 8GB
				NetworkLatency:   10 * time.Millisecond,
				LastSeen:         time.Now(),
			}

			err := target.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, "Target type %s should be valid", tc.targetType)
				assert.Equal(suite.T(), tc.targetType, target.Type)
				assert.ElementsMatch(suite.T(), tc.capabilities, target.Capabilities)
			} else {
				assert.Error(suite.T(), err, "Target type %s should be invalid", tc.targetType)
			}
		})
	}
}

// Test capability matching
func (suite *OffloadTargetTestSuite) TestCapabilityMatching() {
	target := OffloadTarget{
		ID:           "test-target",
		Type:         EDGE,
		Capabilities: []string{"compute_optimized", "low_latency", "gpu_accelerated"},
		TotalCapacity: 8.0,
		NetworkLatency: 5 * time.Millisecond,
		LastSeen:     time.Now(),
	}

	// Test single capability match
	assert.True(suite.T(), target.HasCapability("compute_optimized"))
	assert.True(suite.T(), target.HasCapability("low_latency"))
	assert.True(suite.T(), target.HasCapability("gpu_accelerated"))
	assert.False(suite.T(), target.HasCapability("memory_optimized"))

	// Test multiple capability requirements
	requirements := []string{"compute_optimized", "low_latency"}
	assert.True(suite.T(), target.HasAllCapabilities(requirements))

	requirements = []string{"compute_optimized", "memory_optimized"}
	assert.False(suite.T(), target.HasAllCapabilities(requirements))

	// Test any capability match
	anyRequirements := []string{"memory_optimized", "storage_optimized", "gpu_accelerated"}
	assert.True(suite.T(), target.HasAnyCapability(anyRequirements))

	anyRequirements = []string{"memory_optimized", "storage_optimized"}
	assert.False(suite.T(), target.HasAnyCapability(anyRequirements))
}

// Test that all capacity metrics must be non-negative
func (suite *OffloadTargetTestSuite) TestCapacityMetricsNonNegative() {
	testCases := []struct {
		name          string
		target        OffloadTarget
		expectValid   bool
		expectMessage string
	}{
		{
			name: "all_capacities_valid",
			target: OffloadTarget{
				ID:                "valid-capacity",
				Type:              EDGE,
				TotalCapacity:     16.0,
				AvailableCapacity: 12.0,
				MemoryTotal:       32 * 1024 * 1024 * 1024,
				MemoryAvailable:   24 * 1024 * 1024 * 1024,
				NetworkBandwidth:  1000 * 1024 * 1024,
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid: true,
		},
		{
			name: "zero_capacities_valid",
			target: OffloadTarget{
				ID:                "zero-capacity",
				Type:              EDGE,
				TotalCapacity:     0,
				AvailableCapacity: 0,
				MemoryTotal:       0,
				MemoryAvailable:   0,
				NetworkBandwidth:  0,
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid: true, // Zero capacity means unavailable, but still valid
		},
		{
			name: "negative_total_capacity",
			target: OffloadTarget{
				ID:                "negative-total",
				Type:              EDGE,
				TotalCapacity:     -8.0, // Invalid
				AvailableCapacity: 4.0,
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid:   false,
			expectMessage: "TotalCapacity must be non-negative",
		},
		{
			name: "negative_available_capacity",
			target: OffloadTarget{
				ID:                "negative-available",
				Type:              EDGE,
				TotalCapacity:     8.0,
				AvailableCapacity: -4.0, // Invalid
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid:   false,
			expectMessage: "AvailableCapacity must be non-negative",
		},
		{
			name: "negative_memory_total",
			target: OffloadTarget{
				ID:           "negative-memory-total",
				Type:         EDGE,
				TotalCapacity: 8.0,
				MemoryTotal:  -1024, // Invalid
				NetworkLatency: 10 * time.Millisecond,
				LastSeen:     time.Now(),
			},
			expectValid:   false,
			expectMessage: "MemoryTotal must be non-negative",
		},
		{
			name: "negative_memory_available",
			target: OffloadTarget{
				ID:              "negative-memory-available",
				Type:            EDGE,
				TotalCapacity:   8.0,
				MemoryTotal:     1024,
				MemoryAvailable: -512, // Invalid
				NetworkLatency:  10 * time.Millisecond,
				LastSeen:        time.Now(),
			},
			expectValid:   false,
			expectMessage: "MemoryAvailable must be non-negative",
		},
		{
			name: "negative_network_bandwidth",
			target: OffloadTarget{
				ID:               "negative-bandwidth",
				Type:             EDGE,
				TotalCapacity:    8.0,
				NetworkBandwidth: -1000, // Invalid
				NetworkLatency:   10 * time.Millisecond,
				LastSeen:         time.Now(),
			},
			expectValid:   false,
			expectMessage: "NetworkBandwidth must be non-negative",
		},
		{
			name: "available_greater_than_total_warning",
			target: OffloadTarget{
				ID:                "available-exceeds-total",
				Type:              EDGE,
				TotalCapacity:     8.0,
				AvailableCapacity: 12.0, // Warning: available > total
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid: true, // Valid but should generate warning
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.target.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, "Valid target should not return error")
				
				// Check for warnings
				if tc.target.AvailableCapacity > tc.target.TotalCapacity {
					warnings := tc.target.GetWarnings()
					assert.Contains(suite.T(), warnings, "available capacity exceeds total")
				}
			} else {
				assert.Error(suite.T(), err, "Invalid target should return error")
				if tc.expectMessage != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectMessage)
				}
			}
		})
	}
}

// Test that scores must be in [0.0, 1.0] range
func (suite *OffloadTargetTestSuite) TestScoreRanges() {
	testCases := []struct {
		name          string
		target        OffloadTarget
		expectValid   bool
		expectMessage string
	}{
		{
			name: "valid_scores",
			target: OffloadTarget{
				ID:                "valid-scores",
				Type:              EDGE,
				TotalCapacity:     8.0,
				NetworkStability:  0.95,
				Reliability:       0.99,
				HistoricalSuccess: 0.85,
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid: true,
		},
		{
			name: "boundary_scores_valid",
			target: OffloadTarget{
				ID:                "boundary-scores",
				Type:              EDGE,
				TotalCapacity:     8.0,
				NetworkStability:  0.0, // Valid: exactly 0.0
				Reliability:       1.0, // Valid: exactly 1.0
				HistoricalSuccess: 0.0,
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid: true,
		},
		{
			name: "network_stability_too_high",
			target: OffloadTarget{
				ID:               "stability-high",
				Type:             EDGE,
				TotalCapacity:    8.0,
				NetworkStability: 1.1, // Invalid: > 1.0
				NetworkLatency:   10 * time.Millisecond,
				LastSeen:         time.Now(),
			},
			expectValid:   false,
			expectMessage: "NetworkStability must be in range [0.0, 1.0]",
		},
		{
			name: "network_stability_negative",
			target: OffloadTarget{
				ID:               "stability-negative",
				Type:             EDGE,
				TotalCapacity:    8.0,
				NetworkStability: -0.1, // Invalid: < 0.0
				NetworkLatency:   10 * time.Millisecond,
				LastSeen:         time.Now(),
			},
			expectValid:   false,
			expectMessage: "NetworkStability must be in range [0.0, 1.0]",
		},
		{
			name: "reliability_too_high",
			target: OffloadTarget{
				ID:            "reliability-high",
				Type:          EDGE,
				TotalCapacity: 8.0,
				Reliability:   1.5, // Invalid: > 1.0
				NetworkLatency: 10 * time.Millisecond,
				LastSeen:      time.Now(),
			},
			expectValid:   false,
			expectMessage: "Reliability must be in range [0.0, 1.0]",
		},
		{
			name: "historical_success_negative",
			target: OffloadTarget{
				ID:                "history-negative",
				Type:              EDGE,
				TotalCapacity:     8.0,
				HistoricalSuccess: -0.05, // Invalid: < 0.0
				NetworkLatency:    10 * time.Millisecond,
				LastSeen:          time.Now(),
			},
			expectValid:   false,
			expectMessage: "HistoricalSuccess must be in range [0.0, 1.0]",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.target.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, "Valid scores should not return error")
			} else {
				assert.Error(suite.T(), err, "Invalid scores should return error")
				if tc.expectMessage != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectMessage)
				}
			}
		})
	}
}

// Test that latency must be measurable and recent (< 60s old)
func (suite *OffloadTargetTestSuite) TestLatencyRequirements() {
	now := time.Now()

	testCases := []struct {
		name           string
		networkLatency time.Duration
		lastSeen       time.Time
		expectValid    bool
		expectMessage  string
	}{
		{
			name:           "recent_valid_latency",
			networkLatency: 10 * time.Millisecond,
			lastSeen:       now.Add(-30 * time.Second), // 30s ago
			expectValid:    true,
		},
		{
			name:           "boundary_recent_latency",
			networkLatency: 5 * time.Millisecond,
			lastSeen:       now.Add(-60 * time.Second), // Exactly 60s ago
			expectValid:    true, // Should be valid at boundary
		},
		{
			name:           "very_recent_latency",
			networkLatency: 1 * time.Millisecond,
			lastSeen:       now.Add(-1 * time.Second), // 1s ago
			expectValid:    true,
		},
		{
			name:           "zero_latency_valid",
			networkLatency: 0, // Zero latency (local or theoretical)
			lastSeen:       now.Add(-10 * time.Second),
			expectValid:    true,
		},
		{
			name:           "high_latency_but_recent",
			networkLatency: 500 * time.Millisecond, // High but valid
			lastSeen:       now.Add(-5 * time.Second),
			expectValid:    true,
		},
		{
			name:           "stale_latency_measurement",
			networkLatency: 10 * time.Millisecond,
			lastSeen:       now.Add(-70 * time.Second), // 70s ago, too old
			expectValid:    false,
			expectMessage:  "LastSeen must be within 60 seconds",
		},
		{
			name:           "very_stale_measurement",
			networkLatency: 10 * time.Millisecond,
			lastSeen:       now.Add(-300 * time.Second), // 5 minutes ago
			expectValid:    false,
			expectMessage:  "LastSeen must be within 60 seconds",
		},
		{
			name:           "future_timestamp_invalid",
			networkLatency: 10 * time.Millisecond,
			lastSeen:       now.Add(10 * time.Second), // Future timestamp
			expectValid:    false,
			expectMessage:  "LastSeen cannot be in the future",
		},
		{
			name:           "negative_latency_invalid",
			networkLatency: -5 * time.Millisecond, // Invalid negative latency
			lastSeen:       now.Add(-10 * time.Second),
			expectValid:    false,
			expectMessage:  "NetworkLatency must be non-negative",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			target := OffloadTarget{
				ID:             tc.name,
				Type:           EDGE,
				TotalCapacity:  8.0,
				NetworkLatency: tc.networkLatency,
				LastSeen:       tc.lastSeen,
			}

			err := target.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, "Valid latency/timestamp should not return error")
			} else {
				assert.Error(suite.T(), err, "Invalid latency/timestamp should return error")
				if tc.expectMessage != "" {
					assert.Contains(suite.T(), err.Error(), tc.expectMessage)
				}
			}
		})
	}
}

// Test network characteristics validation
func (suite *OffloadTargetTestSuite) TestNetworkCharacteristics() {
	target := OffloadTarget{
		ID:               "network-test",
		Type:             EDGE,
		TotalCapacity:    8.0,
		NetworkLatency:   25 * time.Millisecond,
		NetworkBandwidth: 100 * 1024 * 1024, // 100 MB/s
		NetworkStability: 0.95,
		NetworkCost:      0.05, // $0.05 per MB
		LastSeen:         time.Now(),
	}

	err := target.Validate()
	assert.NoError(suite.T(), err, "Valid network characteristics should pass")

	// Test network cost validation
	target.NetworkCost = -0.01
	err = target.Validate()
	assert.Error(suite.T(), err, "Negative network cost should be invalid")

	target.NetworkCost = 0.0 // Free network (valid)
	err = target.Validate()
	assert.NoError(suite.T(), err, "Zero network cost should be valid")

	// Test bandwidth vs latency sanity
	target.NetworkBandwidth = 1000000000      // 1 GB/s (very high)
	target.NetworkLatency = 1000 * time.Millisecond // 1s (very high)
	
	warnings := target.GetWarnings()
	// Should warn about inconsistent network characteristics
	assert.Contains(suite.T(), warnings, "network characteristics inconsistent")
}

// Test economic factors validation
func (suite *OffloadTargetTestSuite) TestEconomicFactors() {
	testCases := []struct {
		name         string
		computeCost  float64
		energyCost   float64
		networkCost  float64
		expectValid  bool
		expectWarning string
	}{
		{
			name:        "reasonable_costs",
			computeCost: 0.10, // $0.10/hour
			energyCost:  0.05, // $0.05/hour
			networkCost: 0.01, // $0.01/MB
			expectValid: true,
		},
		{
			name:        "free_resources",
			computeCost: 0.0, // Free compute
			energyCost:  0.0, // Free energy
			networkCost: 0.0, // Free network
			expectValid: true,
		},
		{
			name:        "high_costs",
			computeCost: 2.0,  // $2.00/hour (expensive)
			energyCost:  1.0,  // $1.00/hour (expensive)
			networkCost: 0.50, // $0.50/MB (very expensive)
			expectValid: true,
			expectWarning: "high cost",
		},
		{
			name:        "negative_compute_cost",
			computeCost: -0.10, // Invalid
			energyCost:  0.05,
			networkCost: 0.01,
			expectValid: false,
		},
		{
			name:        "negative_energy_cost",
			computeCost: 0.10,
			energyCost:  -0.05, // Invalid
			networkCost: 0.01,
			expectValid: false,
		},
		{
			name:        "negative_network_cost",
			computeCost: 0.10,
			energyCost:  0.05,
			networkCost: -0.01, // Invalid
			expectValid: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			target := OffloadTarget{
				ID:             tc.name,
				Type:           EDGE,
				TotalCapacity:  8.0,
				ComputeCost:    tc.computeCost,
				EnergyCost:     tc.energyCost,
				NetworkCost:    tc.networkCost,
				NetworkLatency: 10 * time.Millisecond,
				LastSeen:       time.Now(),
			}

			err := target.Validate()
			
			if tc.expectValid {
				assert.NoError(suite.T(), err, "Valid costs should not return error")
				
				if tc.expectWarning != "" {
					warnings := target.GetWarnings()
					assert.Contains(suite.T(), warnings, tc.expectWarning)
				}
			} else {
				assert.Error(suite.T(), err, "Invalid costs should return error")
			}
		})
	}
}

// Test policy compliance attributes
func (suite *OffloadTargetTestSuite) TestPolicyComplianceAttributes() {
	target := OffloadTarget{
		ID:               "policy-test",
		Type:             PRIVATE_CLOUD,
		TotalCapacity:    16.0,
		SecurityLevel:    4,
		DataJurisdiction: "domestic",
		ComplianceFlags:  []string{"SOC2", "GDPR", "HIPAA"},
		NetworkLatency:   15 * time.Millisecond,
		LastSeen:         time.Now(),
	}

	err := target.Validate()
	assert.NoError(suite.T(), err, "Valid policy attributes should pass")

	// Test security level validation
	target.SecurityLevel = -1 // Invalid
	err = target.Validate()
	assert.Error(suite.T(), err, "Negative security level should be invalid")

	target.SecurityLevel = 6 // Invalid (assuming max is 5)
	err = target.Validate()
	assert.Error(suite.T(), err, "Security level above maximum should be invalid")

	target.SecurityLevel = 3 // Reset to valid

	// Test jurisdiction validation
	validJurisdictions := []string{"domestic", "eu", "asia", "americas", "international"}
	for _, jurisdiction := range validJurisdictions {
		target.DataJurisdiction = jurisdiction
		err = target.Validate()
		assert.NoError(suite.T(), err, "Jurisdiction %s should be valid", jurisdiction)
	}

	target.DataJurisdiction = "invalid-jurisdiction"
	err = target.Validate()
	assert.Error(suite.T(), err, "Invalid jurisdiction should be invalid")

	// Test compliance flags
	target.DataJurisdiction = "domestic" // Reset to valid
	target.ComplianceFlags = []string{}  // Empty is valid
	err = target.Validate()
	assert.NoError(suite.T(), err, "Empty compliance flags should be valid")

	// Test duplicate compliance flags
	target.ComplianceFlags = []string{"SOC2", "GDPR", "SOC2"} // Duplicate
	warnings := target.GetWarnings()
	assert.Contains(suite.T(), warnings, "duplicate compliance flags")
}

// Test runtime state consistency
func (suite *OffloadTargetTestSuite) TestRuntimeStateConsistency() {
	target := OffloadTarget{
		ID:                "state-test",
		Type:              EDGE,
		TotalCapacity:     8.0,
		AvailableCapacity: 6.0,
		CurrentLoad:       0.25, // Should be 1 - (6.0/8.0) = 0.25
		EstimatedWaitTime: 5 * time.Second,
		NetworkLatency:    10 * time.Millisecond,
		LastSeen:          time.Now(),
	}

	err := target.Validate()
	assert.NoError(suite.T(), err, "Consistent state should be valid")

	// Check state consistency
	expectedLoad := 1.0 - (target.AvailableCapacity / target.TotalCapacity)
	actualLoad := target.CurrentLoad

	// Allow small floating point differences
	assert.InDelta(suite.T(), expectedLoad, actualLoad, 0.01,
		"CurrentLoad should be consistent with capacity values")

	// Test inconsistent state (should generate warning)
	target.CurrentLoad = 0.9 // High load but high availability
	warnings := target.GetWarnings()
	assert.Contains(suite.T(), warnings, "inconsistent load")

	// Test wait time correlation with load
	target.CurrentLoad = 0.1  // Low load
	target.EstimatedWaitTime = 60 * time.Second // High wait time
	
	warnings = target.GetWarnings()
	assert.Contains(suite.T(), warnings, "wait time inconsistent with load")
}

// Test target availability and health
func (suite *OffloadTargetTestSuite) TestTargetAvailabilityAndHealth() {
	now := time.Now()
	
	// Healthy, available target
	healthyTarget := OffloadTarget{
		ID:                "healthy",
		Type:              EDGE,
		TotalCapacity:     8.0,
		AvailableCapacity: 6.0,
		NetworkLatency:    10 * time.Millisecond,
		Reliability:       0.99,
		LastSeen:          now.Add(-5 * time.Second),
	}

	assert.True(suite.T(), healthyTarget.IsAvailable())
	assert.True(suite.T(), healthyTarget.IsHealthy())

	// Unavailable target (no capacity)
	unavailableTarget := healthyTarget
	unavailableTarget.AvailableCapacity = 0
	
	assert.False(suite.T(), unavailableTarget.IsAvailable())
	assert.True(suite.T(), unavailableTarget.IsHealthy()) // Still healthy, just busy

	// Unhealthy target (stale)
	unhealthyTarget := healthyTarget
	unhealthyTarget.LastSeen = now.Add(-120 * time.Second) // 2 minutes ago
	
	assert.True(suite.T(), unhealthyTarget.IsAvailable())   // Has capacity
	assert.False(suite.T(), unhealthyTarget.IsHealthy())    // But stale

	// Unreliable target
	unreliableTarget := healthyTarget
	unreliableTarget.Reliability = 0.3 // Very unreliable
	
	assert.True(suite.T(), unreliableTarget.IsAvailable())
	assert.False(suite.T(), unreliableTarget.IsHealthy()) // Unreliable = unhealthy
}

func TestOffloadTargetTestSuite(t *testing.T) {
	suite.Run(t, new(OffloadTargetTestSuite))
}