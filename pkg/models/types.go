package models

import (
	"fmt"
)

// TargetType represents different types of offload destinations
type TargetType string

const (
	LOCAL        TargetType = "local"
	EDGE         TargetType = "edge"
	PRIVATE_CLOUD TargetType = "private_cloud"
	PUBLIC_CLOUD TargetType = "public_cloud"
	HYBRID_CLOUD TargetType = "hybrid_cloud"
	HPC_CLUSTER  TargetType = "hpc_cluster"
	FOG          TargetType = "fog"
)

// ProcessStatus represents the current state of a process
type ProcessStatus string

const (
	QUEUED     ProcessStatus = "queued"
	ASSIGNED   ProcessStatus = "assigned"
	EXECUTING  ProcessStatus = "executing"
	COMPLETED  ProcessStatus = "completed"
	FAILED     ProcessStatus = "failed"
	CANCELLED  ProcessStatus = "cancelled"
)

// PolicyType represents whether a policy rule is hard or soft
type PolicyType string

const (
	HARD PolicyType = "hard"
	SOFT PolicyType = "soft"
)

// Operator represents comparison operators for pattern conditions
type Operator string

const (
	EQUAL_TO        Operator = "eq"
	NOT_EQUAL_TO    Operator = "ne"
	GREATER_THAN    Operator = "gt"
	LESS_THAN       Operator = "lt"
	GREATER_EQUAL   Operator = "ge"
	LESS_EQUAL      Operator = "le"
	BETWEEN         Operator = "between"
	NOT_BETWEEN     Operator = "not_between"
	CONTAINS        Operator = "contains"
	NOT_CONTAINS    Operator = "not_contains"
)

// ActionType represents recommended actions for discovered patterns
type ActionType string

const (
	OFFLOAD_TO   ActionType = "OFFLOAD_TO"
	KEEP_LOCAL   ActionType = "KEEP_LOCAL"
	DELAY        ActionType = "DELAY_EXECUTION"
)

// ValidTargetTypes returns all valid target types
func ValidTargetTypes() []TargetType {
	return []TargetType{LOCAL, EDGE, PRIVATE_CLOUD, PUBLIC_CLOUD, HYBRID_CLOUD, FOG}
}

// IsValid checks if a TargetType is valid
func (tt TargetType) IsValid() bool {
	for _, valid := range ValidTargetTypes() {
		if tt == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of TargetType
func (tt TargetType) String() string {
	return string(tt)
}

// ValidProcessStatuses returns all valid process statuses
func ValidProcessStatuses() []ProcessStatus {
	return []ProcessStatus{QUEUED, ASSIGNED, EXECUTING, COMPLETED, FAILED, CANCELLED}
}

// IsValid checks if a ProcessStatus is valid
func (ps ProcessStatus) IsValid() bool {
	for _, valid := range ValidProcessStatuses() {
		if ps == valid {
			return true
		}
	}
	return false
}

// CanTransitionTo checks if a process can transition from current status to target status
func (ps ProcessStatus) CanTransitionTo(target ProcessStatus) bool {
	transitions := map[ProcessStatus][]ProcessStatus{
		QUEUED:    {ASSIGNED, CANCELLED},
		ASSIGNED:  {EXECUTING, FAILED, CANCELLED},
		EXECUTING: {COMPLETED, FAILED},
		COMPLETED: {}, // Terminal state
		FAILED:    {}, // Terminal state
		CANCELLED: {}, // Terminal state
	}
	
	allowedTransitions, exists := transitions[ps]
	if !exists {
		return false
	}
	
	for _, allowed := range allowedTransitions {
		if target == allowed {
			return true
		}
	}
	
	return false
}

// String returns the string representation of ProcessStatus
func (ps ProcessStatus) String() string {
	return string(ps)
}

// String returns the string representation of PolicyType
func (pt PolicyType) String() string {
	return string(pt)
}

// String returns the string representation of Operator
func (op Operator) String() string {
	return string(op)
}

// String returns the string representation of ActionType
func (at ActionType) String() string {
	return string(at)
}

// TimeSlot represents an hour of the day (0-23)
type TimeSlot int

// IsValid checks if a TimeSlot is valid
func (ts TimeSlot) IsValid() bool {
	return ts >= 0 && ts <= 23
}

// DayOfWeek represents a day of the week (0-6, Sunday=0)
type DayOfWeek int

// IsValid checks if a DayOfWeek is valid
func (dow DayOfWeek) IsValid() bool {
	return dow >= 0 && dow <= 6
}

// SecurityLevel represents a security classification level (0-5)
type SecurityLevel int

// IsValid checks if a SecurityLevel is valid
func (sl SecurityLevel) IsValid() bool {
	return sl >= 0 && sl <= 5
}

// DataSensitivity represents data sensitivity level (0-5)
type DataSensitivity int

// IsValid checks if a DataSensitivity is valid
func (ds DataSensitivity) IsValid() bool {
	return ds >= 0 && ds <= 5
}

// Priority represents process priority (1-10, 10=highest)
type Priority int

// IsValid checks if a Priority is valid
func (p Priority) IsValid() bool {
	return p >= 1 && p <= 10
}

// Utilization represents a utilization percentage (0.0-1.0)
type Utilization float64

// IsValid checks if a Utilization is valid
func (u Utilization) IsValid() bool {
	return u >= 0.0 && u <= 1.0
}

// DetailedUtilization represents detailed resource utilization metrics
type DetailedUtilization struct {
	ComputeUsage float64 `json:"compute_usage"` // CPU utilization (0.0-1.0)
	MemoryUsage  float64 `json:"memory_usage"`  // Memory utilization (0.0-1.0)
	DiskUsage    float64 `json:"disk_usage"`    // Disk utilization (0.0-1.0)
	NetworkUsage float64 `json:"network_usage"` // Network utilization (0.0-1.0)
}

// IsValid checks if a DetailedUtilization is valid
func (u DetailedUtilization) IsValid() bool {
	return u.ComputeUsage >= 0.0 && u.ComputeUsage <= 1.0 &&
		   u.MemoryUsage >= 0.0 && u.MemoryUsage <= 1.0 &&
		   u.DiskUsage >= 0.0 && u.DiskUsage <= 1.0 &&
		   u.NetworkUsage >= 0.0 && u.NetworkUsage <= 1.0
}

// Score represents a normalized score (0.0-1.0)
type Score float64

// IsValid checks if a Score is valid
func (s Score) IsValid() bool {
	return s >= 0.0 && s <= 1.0
}

// Confidence represents confidence level (0.0-1.0)
type Confidence float64

// IsValid checks if a Confidence is valid
func (c Confidence) IsValid() bool {
	return c >= 0.0 && c <= 1.0
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s' (value: %v): %s", 
		ve.Field, ve.Value, ve.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", ve[0].Error(), len(ve)-1)
}

// HasErrors returns true if there are validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Add adds a validation error
func (ve *ValidationErrors) Add(field string, value interface{}, message string) {
	*ve = append(*ve, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

// AddIf adds a validation error if the condition is true
func (ve *ValidationErrors) AddIf(condition bool, field string, value interface{}, message string) {
	if condition {
		ve.Add(field, value, message)
	}
}