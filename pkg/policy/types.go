package policy

import (
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// PolicyRule defines a policy constraint
type PolicyRule struct {
	ID          string                                                       `json:"id"`
	Type        models.PolicyType                                            `json:"type"`
	Priority    int                                                          `json:"priority"`
	Condition   func(p models.Process, t models.OffloadTarget) bool         `json:"-"`
	Description string                                                       `json:"description"`
	CreatedAt   time.Time                                                    `json:"created_at"`
	UpdatedAt   time.Time                                                    `json:"updated_at"`
	Enabled     bool                                                         `json:"enabled"`
}

// PolicyViolation represents a policy violation event
type PolicyViolation struct {
	RuleID      string                `json:"rule_id"`
	RuleType    models.PolicyType     `json:"rule_type"`
	ProcessID   string                `json:"process_id"`
	TargetID    string                `json:"target_id"`
	Description string                `json:"description"`
	Timestamp   time.Time             `json:"timestamp"`
	Severity    ViolationSeverity     `json:"severity"`
	Action      CorrectiveAction      `json:"action"`
}

// ViolationSeverity represents the severity of a policy violation
type ViolationSeverity string

const (
	LOW      ViolationSeverity = "low"
	MEDIUM   ViolationSeverity = "medium"
	HIGH     ViolationSeverity = "high"
	CRITICAL ViolationSeverity = "critical"
)

// CorrectiveAction represents the action taken for a violation
type CorrectiveAction string

const (
	BLOCKED      CorrectiveAction = "blocked"
	WARNED       CorrectiveAction = "warned"
	ALLOWED      CorrectiveAction = "allowed"
	REDIRECTED   CorrectiveAction = "redirected"
)

// PolicyEvaluation represents the result of policy evaluation
type PolicyEvaluation struct {
	Process          models.Process          `json:"process"`
	Target           models.OffloadTarget    `json:"target"`
	Allowed          bool                    `json:"allowed"`
	ViolatedRules    []PolicyRule            `json:"violated_rules"`
	AppliedRules     []PolicyRule            `json:"applied_rules"`
	ScoreAdjustment  float64                 `json:"score_adjustment"`
	EvaluationTime   time.Duration           `json:"evaluation_time"`
}

// SafetyConstraints defines non-negotiable safety requirements
type SafetyConstraints struct {
	MinLocalCompute       float64               `json:"min_local_compute"`
	MinLocalMemory        float64               `json:"min_local_memory"`
	MaxConcurrentOffloads int                   `json:"max_concurrent_offloads"`
	HardPolicyViolations  []PolicyViolationType `json:"hard_policy_violations"`
	DataSovereignty       bool                  `json:"data_sovereignty"`
	SecurityClearance     bool                  `json:"security_clearance"`
	MaxLatencyTolerance   time.Duration         `json:"max_latency_tolerance"`
	MinReliability        float64               `json:"min_reliability"`
	LocalFallback         bool                  `json:"local_fallback"`
	MaxRetries            int                   `json:"max_retries"`
	BackoffStrategy       BackoffType           `json:"backoff_strategy"`
}

// PolicyViolationType represents types of policy violations
type PolicyViolationType string

const (
	SECURITY_VIOLATION      PolicyViolationType = "security_violation"
	SOVEREIGNTY_VIOLATION   PolicyViolationType = "sovereignty_violation"
	COMPLIANCE_VIOLATION    PolicyViolationType = "compliance_violation"
	PERFORMANCE_VIOLATION   PolicyViolationType = "performance_violation"
	RESOURCE_VIOLATION      PolicyViolationType = "resource_violation"
)

// BackoffType represents retry backoff strategies
type BackoffType string

const (
	EXPONENTIAL BackoffType = "exponential"
	LINEAR      BackoffType = "linear"
	FIXED       BackoffType = "fixed"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           string           `json:"id"`
	Timestamp    time.Time        `json:"timestamp"`
	EventType    string           `json:"event_type"`
	ProcessID    string           `json:"process_id"`
	TargetID     string           `json:"target_id"`
	RuleID       string           `json:"rule_id"`
	Decision     string           `json:"decision"`
	Violation    *PolicyViolation `json:"violation,omitempty"`
	Details      map[string]interface{} `json:"details"`
}

// PolicyStats tracks policy enforcement statistics
type PolicyStats struct {
	TotalEvaluations   int64                          `json:"total_evaluations"`
	HardViolations     int64                          `json:"hard_violations"`
	SoftViolations     int64                          `json:"soft_violations"`
	BlockedDecisions   int64                          `json:"blocked_decisions"`
	AllowedDecisions   int64                          `json:"allowed_decisions"`
	ViolationsByType   map[PolicyViolationType]int64  `json:"violations_by_type"`
	ViolationsByRule   map[string]int64               `json:"violations_by_rule"`
	AverageEvalTime    time.Duration                  `json:"average_eval_time"`
}