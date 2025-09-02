package models

import (
	"strings"
	"time"
)

// OffloadTarget represents a potential execution destination
type OffloadTarget struct {
	// Identity
	ID       string     `json:"id"`
	Type     TargetType `json:"type"`
	Location string     `json:"location"` // Geographic/logical location

	// Capacity metrics
	TotalCapacity     float64 `json:"total_capacity"`     // Total processing capacity
	AvailableCapacity float64 `json:"available_capacity"` // Currently available capacity
	MemoryTotal       int64   `json:"memory_total"`       // Total memory bytes
	MemoryAvailable   int64   `json:"memory_available"`   // Available memory bytes

	// Network characteristics
	NetworkLatency   time.Duration `json:"network_latency"`   // Round-trip latency
	NetworkBandwidth float64       `json:"network_bandwidth"` // Available bandwidth (bytes/sec)
	NetworkStability float64       `json:"network_stability"` // Stability score (0.0-1.0)
	NetworkCost      float64       `json:"network_cost"`      // Cost per byte transferred

	// Performance characteristics
	ProcessingSpeed float64 `json:"processing_speed"` // Relative speed multiplier
	Reliability     float64 `json:"reliability"`      // Reliability score (0.0-1.0)

	// Economic factors
	ComputeCost float64 `json:"compute_cost"` // Cost per compute unit
	EnergyCost  float64 `json:"energy_cost"`  // Energy cost factor

	// Policy compliance
	SecurityLevel     int      `json:"security_level"`      // Available security level (0-5)
	DataJurisdiction  string   `json:"data_jurisdiction"`   // Legal jurisdiction
	ComplianceFlags   []string `json:"compliance_flags"`    // Compliance certifications
	EnergySource      string   `json:"energy_source"`       // Energy source type
	Capabilities      []string `json:"capabilities"`        // Target capabilities

	// Runtime state
	CurrentLoad       float64       `json:"current_load"`        // Current utilization (0.0-1.0)
	EstimatedWaitTime time.Duration `json:"estimated_wait_time"` // Expected queue wait
	LastSeen          time.Time     `json:"last_seen"`           // Last health check

	// Learning state (updated by algorithm)
	PolicyBonus       float64 `json:"policy_bonus"`        // Policy-derived score modifier
	HistoricalSuccess float64 `json:"historical_success"`  // Success rate with this target
}

// Validate validates the offload target
func (ot OffloadTarget) Validate() error {
	var errors ValidationErrors

	// Validate identity
	errors.AddIf(ot.ID == "", "ID", ot.ID, "ID cannot be empty")
	errors.AddIf(!ot.Type.IsValid(), "Type", ot.Type, "Invalid target type")

	// Validate capacity metrics are non-negative
	errors.AddIf(ot.TotalCapacity < 0, "TotalCapacity", ot.TotalCapacity,
		"TotalCapacity must be non-negative")
	errors.AddIf(ot.AvailableCapacity < 0, "AvailableCapacity", ot.AvailableCapacity,
		"AvailableCapacity must be non-negative")
	errors.AddIf(ot.MemoryTotal < 0, "MemoryTotal", ot.MemoryTotal,
		"MemoryTotal must be non-negative")
	errors.AddIf(ot.MemoryAvailable < 0, "MemoryAvailable", ot.MemoryAvailable,
		"MemoryAvailable must be non-negative")

	// Validate network characteristics
	errors.AddIf(ot.NetworkLatency < 0, "NetworkLatency", ot.NetworkLatency,
		"NetworkLatency must be non-negative")
	errors.AddIf(ot.NetworkBandwidth < 0, "NetworkBandwidth", ot.NetworkBandwidth,
		"NetworkBandwidth must be non-negative")

	// Validate scores are in [0.0, 1.0] range
	errors.AddIf(ot.NetworkStability < 0.0 || ot.NetworkStability > 1.0, 
		"NetworkStability", ot.NetworkStability,
		"NetworkStability must be in range [0.0, 1.0]")
	errors.AddIf(ot.Reliability < 0.0 || ot.Reliability > 1.0, 
		"Reliability", ot.Reliability,
		"Reliability must be in range [0.0, 1.0]")
	errors.AddIf(ot.HistoricalSuccess < 0.0 || ot.HistoricalSuccess > 1.0, 
		"HistoricalSuccess", ot.HistoricalSuccess,
		"HistoricalSuccess must be in range [0.0, 1.0]")

	// Validate economic factors are non-negative
	errors.AddIf(ot.ComputeCost < 0, "ComputeCost", ot.ComputeCost,
		"ComputeCost must be non-negative")
	errors.AddIf(ot.EnergyCost < 0, "EnergyCost", ot.EnergyCost,
		"EnergyCost must be non-negative")
	errors.AddIf(ot.NetworkCost < 0, "NetworkCost", ot.NetworkCost,
		"NetworkCost must be non-negative")

	// Validate security level
	errors.AddIf(ot.SecurityLevel < 0 || ot.SecurityLevel > 5, 
		"SecurityLevel", ot.SecurityLevel,
		"SecurityLevel must be in range [0,5]")

	// Validate jurisdiction
	validJurisdictions := []string{"domestic", "eu", "asia", "americas", "international", "regional"}
	validJurisdiction := false
	for _, valid := range validJurisdictions {
		if ot.DataJurisdiction == valid {
			validJurisdiction = true
			break
		}
	}
	errors.AddIf(!validJurisdiction && ot.DataJurisdiction != "", 
		"DataJurisdiction", ot.DataJurisdiction,
		"Invalid data jurisdiction")

	// Validate latency measurement recency
	if !ot.LastSeen.IsZero() {
		age := time.Since(ot.LastSeen)
		errors.AddIf(age > 60*time.Second, "LastSeen", ot.LastSeen,
			"LastSeen must be within 60 seconds for valid latency measurement")
		errors.AddIf(ot.LastSeen.After(time.Now()), "LastSeen", ot.LastSeen,
			"LastSeen cannot be in the future")
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// GetWarnings returns validation warnings (non-fatal issues)
func (ot OffloadTarget) GetWarnings() []string {
	var warnings []string

	// Warn if available capacity exceeds total capacity
	if ot.AvailableCapacity > ot.TotalCapacity {
		warnings = append(warnings, "available capacity exceeds total capacity")
	}

	// Warn if available memory exceeds total memory
	if ot.MemoryAvailable > ot.MemoryTotal {
		warnings = append(warnings, "available memory exceeds total memory")
	}

	// Warn about inconsistent load calculation
	if ot.TotalCapacity > 0 {
		expectedLoad := 1.0 - (ot.AvailableCapacity / ot.TotalCapacity)
		if abs(ot.CurrentLoad-expectedLoad) > 0.1 {
			warnings = append(warnings, "inconsistent load calculation")
		}
	}

	// Warn about inconsistent wait time vs load
	if ot.CurrentLoad < 0.3 && ot.EstimatedWaitTime > 30*time.Second {
		warnings = append(warnings, "wait time inconsistent with load")
	}

	// Warn about inconsistent network characteristics
	if ot.NetworkBandwidth > 1000*1024*1024 && ot.NetworkLatency > 100*time.Millisecond {
		warnings = append(warnings, "network characteristics inconsistent (high bandwidth but high latency)")
	}

	// Warn about high costs
	if ot.ComputeCost > 1.0 || ot.NetworkCost > 0.20 {
		warnings = append(warnings, "high cost target")
	}

	// Check for duplicate compliance flags
	flagSet := make(map[string]bool)
	for _, flag := range ot.ComplianceFlags {
		if flagSet[flag] {
			warnings = append(warnings, "duplicate compliance flags")
			break
		}
		flagSet[flag] = true
	}

	return warnings
}

// HasCapability checks if the target has a specific capability
func (ot OffloadTarget) HasCapability(capability string) bool {
	for _, cap := range ot.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// HasAllCapabilities checks if the target has all required capabilities
func (ot OffloadTarget) HasAllCapabilities(required []string) bool {
	for _, req := range required {
		if !ot.HasCapability(req) {
			return false
		}
	}
	return true
}

// HasAnyCapability checks if the target has any of the specified capabilities
func (ot OffloadTarget) HasAnyCapability(options []string) bool {
	for _, option := range options {
		if ot.HasCapability(option) {
			return true
		}
	}
	return false
}

// IsAvailable returns true if the target has available capacity
func (ot OffloadTarget) IsAvailable() bool {
	return ot.AvailableCapacity > 0 && ot.MemoryAvailable > 0
}

// IsHealthy returns true if the target is healthy and reachable
func (ot OffloadTarget) IsHealthy() bool {
	// Check if target is recently seen
	if !ot.LastSeen.IsZero() && time.Since(ot.LastSeen) > 90*time.Second {
		return false
	}

	// Check reliability threshold
	if ot.Reliability < 0.5 {
		return false
	}

	return true
}

// GetUtilization returns the current utilization percentage
func (ot OffloadTarget) GetUtilization() float64 {
	if ot.TotalCapacity <= 0 {
		return 1.0 // Fully utilized if no capacity
	}
	return 1.0 - (ot.AvailableCapacity / ot.TotalCapacity)
}

// GetMemoryUtilization returns the current memory utilization percentage
func (ot OffloadTarget) GetMemoryUtilization() float64 {
	if ot.MemoryTotal <= 0 {
		return 0.0
	}
	return 1.0 - (float64(ot.MemoryAvailable) / float64(ot.MemoryTotal))
}

// CanAccommodate checks if the target can accommodate a process
func (ot OffloadTarget) CanAccommodate(process Process) bool {
	// Check capacity
	if process.CPURequirement > ot.AvailableCapacity {
		return false
	}

	// Check memory
	if process.MemoryRequirement > ot.MemoryAvailable {
		return false
	}

	// Check security level
	if process.SecurityLevel > ot.SecurityLevel {
		return false
	}

	return true
}

// EstimateExecutionTime estimates how long a process would take on this target
func (ot OffloadTarget) EstimateExecutionTime(process Process) time.Duration {
	baseTime := process.EstimatedDuration

	// Adjust for processing speed
	adjustedTime := time.Duration(float64(baseTime) / ot.ProcessingSpeed)

	// Add network transfer time for data
	dataSize := process.InputSize + process.OutputSize
	if dataSize > 0 && ot.NetworkBandwidth > 0 {
		transferTime := time.Duration(float64(dataSize)/ot.NetworkBandwidth) * time.Second
		adjustedTime += transferTime
	}

	// Add network latency
	adjustedTime += ot.NetworkLatency * 2 // Round trip

	// Add estimated wait time
	adjustedTime += ot.EstimatedWaitTime

	return adjustedTime
}

// GetTotalCost estimates the total cost of running a process on this target
func (ot OffloadTarget) GetTotalCost(process Process) float64 {
	// Compute cost based on estimated duration
	executionTime := ot.EstimateExecutionTime(process)
	computeCost := ot.ComputeCost * executionTime.Hours()

	// Network cost based on data transfer
	dataSize := process.InputSize + process.OutputSize
	networkCost := ot.NetworkCost * float64(dataSize)/(1024*1024) // Cost per MB

	// Energy cost
	energyCost := ot.EnergyCost * executionTime.Hours()

	return computeCost + networkCost + energyCost
}

// GetCompatibilityScore returns a compatibility score for a process (0.0-1.0)
func (ot OffloadTarget) GetCompatibilityScore(process Process) float64 {
	score := 1.0

	// Check capacity match
	if ot.TotalCapacity > 0 {
		capacityRatio := process.CPURequirement / ot.AvailableCapacity
		if capacityRatio > 1.0 {
			score *= 0.0 // Cannot accommodate
		} else if capacityRatio > 0.8 {
			score *= 0.6 // High utilization penalty
		} else if capacityRatio > 0.5 {
			score *= 0.8 // Moderate utilization penalty
		}
	}

	// Check memory match
	if ot.MemoryTotal > 0 {
		memoryRatio := float64(process.MemoryRequirement) / float64(ot.MemoryAvailable)
		if memoryRatio > 1.0 {
			score *= 0.0 // Cannot accommodate
		} else if memoryRatio > 0.8 {
			score *= 0.7 // High memory usage penalty
		}
	}

	// Check security level compatibility
	if process.SecurityLevel > ot.SecurityLevel {
		score *= 0.0 // Security requirement not met
	}

	// Bonus for high reliability
	score *= ot.Reliability

	// Penalty for high latency on real-time processes
	if process.RealTime && ot.NetworkLatency > 50*time.Millisecond {
		score *= 0.5
	}

	return score
}

// GetTargetProfile returns a string describing the target profile
func (ot OffloadTarget) GetTargetProfile() string {
	var profile []string

	// Add type
	profile = append(profile, string(ot.Type))

	// Add key capabilities
	if ot.HasCapability("low_latency") {
		profile = append(profile, "low_latency")
	}
	if ot.HasCapability("high_security") {
		profile = append(profile, "high_security")
	}
	if ot.HasCapability("cost_optimized") {
		profile = append(profile, "cost_optimized")
	}
	if ot.HasCapability("gpu_accelerated") {
		profile = append(profile, "gpu_accelerated")
	}

	// Add capacity class
	if ot.TotalCapacity >= 32 {
		profile = append(profile, "high_capacity")
	} else if ot.TotalCapacity >= 8 {
		profile = append(profile, "medium_capacity")
	} else {
		profile = append(profile, "low_capacity")
	}

	return strings.Join(profile, ",")
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}