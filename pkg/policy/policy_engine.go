package policy

import (
	"fmt"
	"sync"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// PolicyEngine enforces policy constraints on offloading decisions
type PolicyEngine struct {
	rules             []PolicyRule
	safetyConstraints SafetyConstraints
	auditLogs         []AuditLog
	violations        []PolicyViolation
	stats             PolicyStats
	mu                sync.RWMutex
	immutable         bool
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		rules:      make([]PolicyRule, 0),
		auditLogs:  make([]AuditLog, 0),
		violations: make([]PolicyViolation, 0),
		stats:      PolicyStats{
			ViolationsByType: make(map[PolicyViolationType]int64),
			ViolationsByRule: make(map[string]int64),
		},
		safetyConstraints: SafetyConstraints{
			MinLocalCompute:       0.2,
			MinLocalMemory:        0.2,
			MaxConcurrentOffloads: 10,
			DataSovereignty:       true,
			SecurityClearance:     true,
			MaxLatencyTolerance:   500 * time.Millisecond,
			MinReliability:        0.5,
			LocalFallback:         true,
			MaxRetries:            3,
			BackoffStrategy:       EXPONENTIAL,
		},
		immutable: false,
	}
}

// AddRule adds a new policy rule
func (pe *PolicyEngine) AddRule(rule PolicyRule) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.immutable {
		return fmt.Errorf("policy engine is immutable during execution")
	}

	// Validate rule
	if rule.Condition == nil {
		return fmt.Errorf("rule condition cannot be nil")
	}

	// Set timestamps
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	rule.UpdatedAt = time.Now()
	rule.Enabled = true

	// Generate ID if not provided
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d_%s", len(pe.rules)+1, rule.Type)
	}

	pe.rules = append(pe.rules, rule)
	return nil
}

// EvaluatePolicy evaluates all policy rules for a process-target pair
func (pe *PolicyEngine) EvaluatePolicy(
	process models.Process,
	target models.OffloadTarget,
) PolicyEvaluation {
	startTime := time.Now()
	
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	evaluation := PolicyEvaluation{
		Process:       process,
		Target:        target,
		Allowed:       true,
		ViolatedRules: make([]PolicyRule, 0),
		AppliedRules:  make([]PolicyRule, 0),
	}

	// Track evaluation
	pe.stats.TotalEvaluations++

	// Evaluate each rule
	for _, rule := range pe.rules {
		if !rule.Enabled {
			continue
		}

		evaluation.AppliedRules = append(evaluation.AppliedRules, rule)

		// Check if rule condition is met (true means no violation)
		if !rule.Condition(process, target) {
			evaluation.ViolatedRules = append(evaluation.ViolatedRules, rule)
			
			// Hard constraints block the decision
			if rule.Type == models.HARD {
				evaluation.Allowed = false
				pe.stats.HardViolations++
				pe.stats.ViolationsByRule[rule.ID]++
				
				// Log violation
				pe.logViolation(rule, process, target, CRITICAL)
			} else {
				// Soft constraints affect scoring
				evaluation.ScoreAdjustment -= 0.2 // Penalty for soft violation
				pe.stats.SoftViolations++
				pe.stats.ViolationsByRule[rule.ID]++
				
				// Log violation
				pe.logViolation(rule, process, target, MEDIUM)
			}
		}
	}

	// Update statistics
	if evaluation.Allowed {
		pe.stats.AllowedDecisions++
	} else {
		pe.stats.BlockedDecisions++
	}

	evaluation.EvaluationTime = time.Since(startTime)
	
	// Update average evaluation time
	if pe.stats.AverageEvalTime == 0 {
		pe.stats.AverageEvalTime = evaluation.EvaluationTime
	} else {
		pe.stats.AverageEvalTime = (pe.stats.AverageEvalTime + evaluation.EvaluationTime) / 2
	}

	// Audit log
	pe.logEvaluation(evaluation)

	return evaluation
}

// FilterTargetsByPolicy filters targets based on hard policy constraints
func (pe *PolicyEngine) FilterTargetsByPolicy(
	process models.Process,
	targets []models.OffloadTarget,
) []models.OffloadTarget {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	filtered := make([]models.OffloadTarget, 0)

	for _, target := range targets {
		evaluation := pe.EvaluatePolicy(process, target)
		if evaluation.Allowed {
			filtered = append(filtered, target)
		}
	}

	return filtered
}

// CheckSafetyConstraints checks if safety constraints are met
func (pe *PolicyEngine) CheckSafetyConstraints(
	state models.SystemState,
	concurrentOffloads int,
) bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Check resource protection
	if float64(state.ComputeUsage) < pe.safetyConstraints.MinLocalCompute {
		return false
	}
	if float64(state.MemoryUsage) < pe.safetyConstraints.MinLocalMemory {
		return false
	}

	// Check concurrent offload limit
	if concurrentOffloads >= pe.safetyConstraints.MaxConcurrentOffloads {
		return false
	}

	return true
}

// logViolation logs a policy violation
func (pe *PolicyEngine) logViolation(
	rule PolicyRule,
	process models.Process,
	target models.OffloadTarget,
	severity ViolationSeverity,
) {
	violation := PolicyViolation{
		RuleID:      rule.ID,
		RuleType:    rule.Type,
		ProcessID:   process.ID,
		TargetID:    target.ID,
		Description: rule.Description,
		Timestamp:   time.Now(),
		Severity:    severity,
		Action:      BLOCKED,
	}

	if rule.Type == models.SOFT {
		violation.Action = WARNED
	}

	pe.violations = append(pe.violations, violation)

	// Map to violation type
	violationType := COMPLIANCE_VIOLATION
	if process.SafetyCritical {
		violationType = SECURITY_VIOLATION
		pe.stats.ViolationsByType[SECURITY_VIOLATION]++
	} else if process.LocalityRequired {
		violationType = SOVEREIGNTY_VIOLATION
		pe.stats.ViolationsByType[SOVEREIGNTY_VIOLATION]++
	} else {
		pe.stats.ViolationsByType[COMPLIANCE_VIOLATION]++
	}

	// Create audit log
	auditLog := AuditLog{
		ID:        fmt.Sprintf("audit_%d", len(pe.auditLogs)+1),
		Timestamp: time.Now(),
		EventType: "policy_violation",
		ProcessID: process.ID,
		TargetID:  target.ID,
		RuleID:    rule.ID,
		Decision:  string(violation.Action),
		Violation: &violation,
		Details: map[string]interface{}{
			"violation_type": violationType,
			"severity":       severity,
		},
	}

	pe.auditLogs = append(pe.auditLogs, auditLog)
}

// logEvaluation logs a policy evaluation
func (pe *PolicyEngine) logEvaluation(evaluation PolicyEvaluation) {
	decision := "allowed"
	if !evaluation.Allowed {
		decision = "blocked"
	}

	auditLog := AuditLog{
		ID:        fmt.Sprintf("audit_%d", len(pe.auditLogs)+1),
		Timestamp: time.Now(),
		EventType: "policy_evaluation",
		ProcessID: evaluation.Process.ID,
		TargetID:  evaluation.Target.ID,
		Decision:  decision,
		Details: map[string]interface{}{
			"applied_rules":    len(evaluation.AppliedRules),
			"violated_rules":   len(evaluation.ViolatedRules),
			"score_adjustment": evaluation.ScoreAdjustment,
			"evaluation_time":  evaluation.EvaluationTime,
		},
	}

	pe.auditLogs = append(pe.auditLogs, auditLog)
}

// GetViolations returns policy violations
func (pe *PolicyEngine) GetViolations() []PolicyViolation {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	violations := make([]PolicyViolation, len(pe.violations))
	copy(violations, pe.violations)
	return violations
}

// GetAuditLogs returns audit logs
func (pe *PolicyEngine) GetAuditLogs() []AuditLog {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	logs := make([]AuditLog, len(pe.auditLogs))
	copy(logs, pe.auditLogs)
	return logs
}

// GetStats returns policy enforcement statistics
func (pe *PolicyEngine) GetStats() PolicyStats {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	return pe.stats
}

// SetImmutable makes the policy engine immutable (for execution phase)
func (pe *PolicyEngine) SetImmutable(immutable bool) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	pe.immutable = immutable
}

// SetSafetyConstraints updates safety constraints
func (pe *PolicyEngine) SetSafetyConstraints(constraints SafetyConstraints) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	if !pe.immutable {
		pe.safetyConstraints = constraints
	}
}

// GetSafetyConstraints returns current safety constraints
func (pe *PolicyEngine) GetSafetyConstraints() SafetyConstraints {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	return pe.safetyConstraints
}

// ClearViolations clears violation history (for testing)
func (pe *PolicyEngine) ClearViolations() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	pe.violations = make([]PolicyViolation, 0)
}

// GetRules returns all policy rules
func (pe *PolicyEngine) GetRules() []PolicyRule {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	rules := make([]PolicyRule, len(pe.rules))
	copy(rules, pe.rules)
	return rules
}