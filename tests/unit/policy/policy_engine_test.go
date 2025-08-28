package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PolicyEngine test requirements:
// 1. Safety constraints are immutable during execution
// 2. Violations must trigger immediate corrective action
// 3. All safety violations must be logged and auditable
// 4. Hard constraints must never be violated
// 5. Soft constraints should influence scoring but not filter

type PolicyEngineTestSuite struct {
	suite.Suite
	policyEngine *PolicyEngine
}

func (suite *PolicyEngineTestSuite) SetupTest() {
	suite.policyEngine = NewPolicyEngine()
}

// Test that hard constraints must never be violated
func (suite *PolicyEngineTestSuite) TestHardConstraintsNeverViolated() {
	// Add critical safety constraints
	safetyConstraints := []PolicyRule{
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				// Safety-critical processes must stay local
				return !(p.SafetyCritical && t.Type != "local")
			},
			Description: "Safety-critical processes must stay local",
		},
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				// High-security processes need secure targets
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Security level compliance",
		},
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				// Sensitive data must stay in jurisdiction
				if p.DataSensitivity >= 4 {
					return t.DataJurisdiction == "domestic"
				}
				return true
			},
			Description: "Data sovereignty requirement",
		},
	}

	// Add all hard constraints to policy engine
	for _, rule := range safetyConstraints {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err, "Should be able to add hard constraint")
	}

	// Test various violation scenarios
	testCases := []struct {
		name              string
		process           Process
		targets           []OffloadTarget
		expectedFiltered  int
		violationTypes    []string
	}{
		{
			name: "safety_critical_violation",
			process: Process{
				ID:             "safety-test-1",
				SafetyCritical: true,
				SecurityLevel:  3,
				DataSensitivity: 2,
			},
			targets: []OffloadTarget{
				{ID: "local", Type: "local", SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "edge", Type: "edge", SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "cloud", Type: "cloud", SecurityLevel: 3, DataJurisdiction: "domestic"},
			},
			expectedFiltered: 1, // Only local should remain
			violationTypes:   []string{"safety_critical"},
		},
		{
			name: "security_level_violation",
			process: Process{
				ID:             "security-test-1",
				SafetyCritical: false,
				SecurityLevel:  5, // Very high security requirement
				DataSensitivity: 2,
			},
			targets: []OffloadTarget{
				{ID: "secure-edge", Type: "edge", SecurityLevel: 5, DataJurisdiction: "domestic"},
				{ID: "normal-edge", Type: "edge", SecurityLevel: 3, DataJurisdiction: "domestic"}, // Violation
				{ID: "insecure-cloud", Type: "cloud", SecurityLevel: 1, DataJurisdiction: "domestic"}, // Violation
			},
			expectedFiltered: 1, // Only secure-edge should remain
			violationTypes:   []string{"security_level"},
		},
		{
			name: "data_sovereignty_violation",
			process: Process{
				ID:             "data-test-1",
				SafetyCritical: false,
				SecurityLevel:  3,
				DataSensitivity: 5, // Very sensitive data
			},
			targets: []OffloadTarget{
				{ID: "domestic-cloud", Type: "cloud", SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "foreign-cloud", Type: "cloud", SecurityLevel: 3, DataJurisdiction: "foreign"}, // Violation
				{ID: "international-edge", Type: "edge", SecurityLevel: 3, DataJurisdiction: "international"}, // Violation
			},
			expectedFiltered: 1, // Only domestic-cloud should remain
			violationTypes:   []string{"data_sovereignty"},
		},
		{
			name: "multiple_violations",
			process: Process{
				ID:             "multi-violation-test",
				SafetyCritical: true,    // Requires local
				SecurityLevel:  5,       // Requires high security
				DataSensitivity: 5,      // Requires domestic
			},
			targets: []OffloadTarget{
				{ID: "perfect-local", Type: "local", SecurityLevel: 5, DataJurisdiction: "domestic"},     // Valid
				{ID: "foreign-local", Type: "local", SecurityLevel: 5, DataJurisdiction: "foreign"},     // Invalid (foreign)
				{ID: "insecure-local", Type: "local", SecurityLevel: 2, DataJurisdiction: "domestic"},   // Invalid (insecure)
				{ID: "secure-edge", Type: "edge", SecurityLevel: 5, DataJurisdiction: "domestic"},       // Invalid (not local)
			},
			expectedFiltered: 1, // Only perfect-local should remain
			violationTypes:   []string{"safety_critical", "security_level", "data_sovereignty"},
		},
		{
			name: "no_violations",
			process: Process{
				ID:             "compliant-test",
				SafetyCritical: false,
				SecurityLevel:  2,
				DataSensitivity: 2,
			},
			targets: []OffloadTarget{
				{ID: "compliant-edge", Type: "edge", SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "compliant-cloud", Type: "cloud", SecurityLevel: 4, DataJurisdiction: "domestic"},
			},
			expectedFiltered: 2, // All targets should remain
			violationTypes:   []string{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			filtered := suite.policyEngine.FilterTargets(tc.process, tc.targets)
			
			assert.Len(suite.T(), filtered, tc.expectedFiltered,
				"Expected %d targets after filtering, got %d", tc.expectedFiltered, len(filtered))
			
			// Verify no hard constraint violations in filtered results
			for _, target := range filtered {
				for _, rule := range safetyConstraints {
					if rule.Type == HARD {
						assert.True(suite.T(), rule.Condition(tc.process, target),
							"Filtered target %s should not violate hard constraint: %s", 
							target.ID, rule.Description)
					}
				}
			}
			
			// Verify violations are logged
			violations := suite.policyEngine.GetViolationLog()
			
			if len(tc.violationTypes) > 0 {
				assert.NotEmpty(suite.T(), violations,
					"Violations should be logged for test case %s", tc.name)
				
				// Check for expected violation types
				latestViolations := violations[len(violations)-len(tc.targets):]
				violationFound := make(map[string]bool)
				
				for _, violation := range latestViolations {
					for _, expectedType := range tc.violationTypes {
						if strings.Contains(violation.RuleDescription, expectedType) ||
						   strings.Contains(violation.ViolationType, expectedType) {
							violationFound[expectedType] = true
						}
					}
				}
				
				for _, expectedType := range tc.violationTypes {
					assert.True(suite.T(), violationFound[expectedType],
						"Should log violation type: %s", expectedType)
				}
			}
		})
	}
}

// Test that soft constraints influence scoring but don't filter
func (suite *PolicyEngineTestSuite) TestSoftConstraintsInfluenceScoring() {
	// Add soft constraints with different priorities
	softConstraints := []PolicyRule{
		{
			Type:     SOFT,
			Priority: 2,
			Condition: func(p Process, t OffloadTarget) bool {
				// Prefer renewable energy
				return t.EnergySource == "renewable"
			},
			Description: "Renewable energy preference",
		},
		{
			Type:     SOFT,
			Priority: 3,
			Condition: func(p Process, t OffloadTarget) bool {
				// Prefer low-cost options
				return t.ComputeCost < 0.10
			},
			Description: "Cost optimization preference",
		},
		{
			Type:     SOFT,
			Priority: 4,
			Condition: func(p Process, t OffloadTarget) bool {
				// Prefer local data processing
				return t.Location == "local" || t.Location == "regional"
			},
			Description: "Data locality preference",
		},
	}

	// Add soft constraints
	for _, rule := range softConstraints {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}

	process := Process{
		ID:             "soft-constraint-test",
		SafetyCritical: false,
		SecurityLevel:  2,
		DataSensitivity: 2,
	}

	targets := []OffloadTarget{
		{
			ID:           "perfect-target",
			Type:         "edge",
			EnergySource: "renewable",    // Satisfies energy preference
			ComputeCost:  0.05,          // Satisfies cost preference
			Location:     "local",       // Satisfies locality preference
			SecurityLevel: 3,
			DataJurisdiction: "domestic",
		},
		{
			ID:           "good-target",
			Type:         "edge",
			EnergySource: "renewable",    // Satisfies energy preference
			ComputeCost:  0.15,          // Violates cost preference
			Location:     "international", // Violates locality preference
			SecurityLevel: 3,
			DataJurisdiction: "domestic",
		},
		{
			ID:           "poor-target",
			Type:         "cloud",
			EnergySource: "fossil",       // Violates energy preference
			ComputeCost:  0.25,          // Violates cost preference
			Location:     "international", // Violates locality preference
			SecurityLevel: 3,
			DataJurisdiction: "domestic",
		},
	}

	filtered := suite.policyEngine.FilterTargets(process, targets)

	// All targets should remain (soft constraints don't filter)
	assert.Len(suite.T(), filtered, 3, "Soft constraints should not filter out targets")

	// Verify policy bonuses reflect constraint satisfaction
	var perfectTarget, goodTarget, poorTarget *OffloadTarget
	for i := range filtered {
		switch filtered[i].ID {
		case "perfect-target":
			perfectTarget = &filtered[i]
		case "good-target":
			goodTarget = &filtered[i]
		case "poor-target":
			poorTarget = &filtered[i]
		}
	}

	require.NotNil(suite.T(), perfectTarget, "Perfect target should be in filtered results")
	require.NotNil(suite.T(), goodTarget, "Good target should be in filtered results")
	require.NotNil(suite.T(), poorTarget, "Poor target should be in filtered results")

	// Perfect target should have highest policy bonus
	assert.Greater(suite.T(), perfectTarget.PolicyBonus, goodTarget.PolicyBonus,
		"Perfect target should have higher policy bonus than good target")
	
	assert.Greater(suite.T(), goodTarget.PolicyBonus, poorTarget.PolicyBonus,
		"Good target should have higher policy bonus than poor target")
	
	// Verify specific bonus values
	assert.Greater(suite.T(), perfectTarget.PolicyBonus, 0.2,
		"Perfect target should have significant positive bonus")
	
	assert.Less(suite.T(), poorTarget.PolicyBonus, 0.0,
		"Poor target should have negative or zero bonus")

	suite.T().Logf("Policy bonuses: Perfect=%.3f, Good=%.3f, Poor=%.3f",
		perfectTarget.PolicyBonus, goodTarget.PolicyBonus, poorTarget.PolicyBonus)
}

// Test that safety constraints are immutable during execution
func (suite *PolicyEngineTestSuite) TestSafetyConstraintsImmutability() {
	// Add initial safety constraints
	initialRules := []PolicyRule{
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Initial security constraint",
		},
		{
			Type:     SOFT,
			Priority: 2,
			Condition: func(p Process, t OffloadTarget) bool {
				return t.ComputeCost < 0.20
			},
			Description: "Initial cost preference",
		},
	}

	for _, rule := range initialRules {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}

	// Start execution mode (safety constraints become immutable)
	suite.policyEngine.StartExecution()

	// Verify we can still add soft constraints
	softRule := PolicyRule{
		Type:     SOFT,
		Priority: 3,
		Condition: func(p Process, t OffloadTarget) bool {
			return t.EnergySource == "renewable"
		},
		Description: "Runtime energy preference",
	}

	err := suite.policyEngine.AddRule(softRule)
	assert.NoError(suite.T(), err, "Should be able to add soft rules during execution")

	// Verify we cannot add/modify hard constraints during execution
	hardRule := PolicyRule{
		Type:     HARD,
		Priority: 1,
		Condition: func(p Process, t OffloadTarget) bool {
			return t.DataJurisdiction == "domestic"
		},
		Description: "Runtime hard constraint",
	}

	err = suite.policyEngine.AddRule(hardRule)
	assert.Error(suite.T(), err, "Should not be able to add hard rules during execution")
	assert.Contains(suite.T(), err.Error(), "immutable",
		"Error should mention immutability")

	// Verify we cannot remove hard constraints during execution
	err = suite.policyEngine.RemoveRule("Initial security constraint")
	assert.Error(suite.T(), err, "Should not be able to remove hard rules during execution")

	// Verify we can still modify soft constraint priorities
	err = suite.policyEngine.UpdateRulePriority("Initial cost preference", 5)
	assert.NoError(suite.T(), err, "Should be able to update soft rule priorities during execution")

	err = suite.policyEngine.UpdateRulePriority("Initial security constraint", 2)
	assert.Error(suite.T(), err, "Should not be able to update hard rule priorities during execution")

	// Stop execution mode
	suite.policyEngine.StopExecution()

	// Verify constraints become mutable again
	err = suite.policyEngine.AddRule(hardRule)
	assert.NoError(suite.T(), err, "Should be able to add hard rules after stopping execution")
}

// Test that violations trigger immediate corrective action
func (suite *PolicyEngineTestSuite) TestViolationCorrectiveActions() {
	// Add rules with associated corrective actions
	actionTriggered := make(map[string]bool)
	
	rules := []PolicyRule{
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Security level compliance",
			CorrectiveAction: func(violation PolicyViolation) error {
				actionTriggered["security_escalation"] = true
				// Simulate escalation to security team
				return nil
			},
		},
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				if p.DataSensitivity >= 4 {
					return t.DataJurisdiction == "domestic"
				}
				return true
			},
			Description: "Data sovereignty compliance",
			CorrectiveAction: func(violation PolicyViolation) error {
				actionTriggered["data_protection_alert"] = true
				// Simulate alerting data protection officer
				return nil
			},
		},
		{
			Type:     SOFT,
			Priority: 3,
			Condition: func(p Process, t OffloadTarget) bool {
				return t.ComputeCost < 0.50 // High cost threshold
			},
			Description: "Cost control guideline",
			CorrectiveAction: func(violation PolicyViolation) error {
				actionTriggered["cost_alert"] = true
				// Simulate cost alert
				return nil
			},
		},
	}

	for _, rule := range rules {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}

	// Create process that will trigger violations
	violatingProcess := Process{
		ID:             "violation-trigger-test",
		SafetyCritical: false,
		SecurityLevel:  5,     // High security requirement
		DataSensitivity: 5,    // High data sensitivity
	}

	violatingTargets := []OffloadTarget{
		{
			ID:               "insecure-expensive-foreign",
			Type:             "cloud",
			SecurityLevel:    2,            // Violates security requirement
			DataJurisdiction: "foreign",    // Violates data sovereignty
			ComputeCost:      1.00,         // Violates cost guideline
		},
		{
			ID:               "secure-domestic-expensive",
			Type:             "cloud",
			SecurityLevel:    5,            // Meets security requirement
			DataJurisdiction: "domestic",   // Meets data sovereignty
			ComputeCost:      0.75,         // Violates cost guideline (soft)
		},
	}

	// Reset action tracking
	for key := range actionTriggered {
		actionTriggered[key] = false
	}

	// This should trigger violations and corrective actions
	filtered := suite.policyEngine.FilterTargets(violatingProcess, violatingTargets)

	// Hard constraint violations should trigger corrective actions
	assert.True(suite.T(), actionTriggered["security_escalation"],
		"Security violation should trigger corrective action")
	
	assert.True(suite.T(), actionTriggered["data_protection_alert"],
		"Data sovereignty violation should trigger corrective action")

	// Soft constraint violations may or may not trigger actions depending on implementation
	// (they should at least be logged)
	
	// Verify violations are logged with corrective action status
	violations := suite.policyEngine.GetViolationLog()
	assert.NotEmpty(suite.T(), violations, "Violations should be logged")
	
	recentViolations := violations[len(violations)-2:] // Last 2 violations
	
	for _, violation := range recentViolations {
		assert.NotZero(suite.T(), violation.Timestamp, "Violation should have timestamp")
		assert.NotEmpty(suite.T(), violation.ProcessID, "Violation should reference process")
		assert.NotEmpty(suite.T(), violation.TargetID, "Violation should reference target")
		assert.NotEmpty(suite.T(), violation.ViolationType, "Violation should have type")
		
		if violation.RuleType == HARD {
			assert.True(suite.T(), violation.CorrectiveActionTaken,
				"Hard violations should trigger corrective actions")
		}
	}

	// Verify immediate response (corrective actions should complete quickly)
	for _, violation := range recentViolations {
		if violation.CorrectiveActionTaken {
			actionDuration := violation.CorrectiveActionCompletedAt.Sub(violation.Timestamp)
			assert.Less(suite.T(), actionDuration, 1*time.Second,
				"Corrective actions should complete within 1 second")
		}
	}
}

// Test comprehensive auditing of safety violations
func (suite *PolicyEngineTestSuite) TestComprehensiveViolationAuditing() {
	// Add auditable rules
	rules := []PolicyRule{
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				return !(p.SafetyCritical && t.Type != "local")
			},
			Description: "Safety-critical locality requirement",
			AuditLevel:  AUDIT_FULL,
		},
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Security level enforcement",
			AuditLevel:  AUDIT_FULL,
		},
		{
			Type:     SOFT,
			Priority: 2,
			Condition: func(p Process, t OffloadTarget) bool {
				return t.ComputeCost < 0.15
			},
			Description: "Cost optimization preference",
			AuditLevel:  AUDIT_SUMMARY,
		},
	}

	for _, rule := range rules {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}

	// Create test scenarios that will generate various violations
	testScenarios := []struct {
		name     string
		process  Process
		targets  []OffloadTarget
		expected []string // Expected violation types
	}{
		{
			name: "safety_critical_violation",
			process: Process{
				ID:             "audit-test-1",
				SafetyCritical: true,
				SecurityLevel:  3,
			},
			targets: []OffloadTarget{
				{ID: "edge-unsafe", Type: "edge", SecurityLevel: 3},
			},
			expected: []string{"safety_critical_locality"},
		},
		{
			name: "security_level_violation",
			process: Process{
				ID:             "audit-test-2",
				SafetyCritical: false,
				SecurityLevel:  5,
			},
			targets: []OffloadTarget{
				{ID: "insecure-target", Type: "cloud", SecurityLevel: 2},
			},
			expected: []string{"security_level"},
		},
		{
			name: "combined_violations",
			process: Process{
				ID:             "audit-test-3",
				SafetyCritical: true,
				SecurityLevel:  4,
			},
			targets: []OffloadTarget{
				{ID: "bad-target", Type: "cloud", SecurityLevel: 1, ComputeCost: 0.50},
			},
			expected: []string{"safety_critical_locality", "security_level", "cost_optimization"},
		},
	}

	auditRecordsBefore := len(suite.policyEngine.GetAuditLog())

	for _, scenario := range testScenarios {
		suite.Run(scenario.name, func() {
			// Execute filtering which should trigger violations
			filtered := suite.policyEngine.FilterTargets(scenario.process, scenario.targets)
			
			// Should filter out violating targets
			if len(scenario.expected) > 0 {
				// Hard violations should result in filtered targets
				hasHardViolations := false
				for _, rule := range rules {
					if rule.Type == HARD {
						for _, target := range scenario.targets {
							if !rule.Condition(scenario.process, target) {
								hasHardViolations = true
								break
							}
						}
						if hasHardViolations {
							break
						}
					}
				}
				
				if hasHardViolations {
					assert.Empty(suite.T(), filtered, 
						"Targets with hard violations should be filtered out")
				}
			}
		})
	}

	// Verify comprehensive audit logging
	auditLog := suite.policyEngine.GetAuditLog()
	newAuditRecords := auditLog[auditRecordsBefore:]
	
	assert.NotEmpty(suite.T(), newAuditRecords, "Should generate audit records")

	for _, record := range newAuditRecords {
		// Verify audit record completeness
		assert.NotEmpty(suite.T(), record.EventID, "Audit record should have unique event ID")
		assert.NotZero(suite.T(), record.Timestamp, "Audit record should have timestamp")
		assert.NotEmpty(suite.T(), record.EventType, "Audit record should have event type")
		assert.NotEmpty(suite.T(), record.ProcessID, "Audit record should reference process")
		assert.NotEmpty(suite.T(), record.Description, "Audit record should have description")
		
		// Verify audit level compliance
		if record.AuditLevel == AUDIT_FULL {
			assert.NotEmpty(suite.T(), record.ProcessSnapshot, "Full audit should include process snapshot")
			assert.NotEmpty(suite.T(), record.TargetSnapshot, "Full audit should include target snapshot")
			assert.NotEmpty(suite.T(), record.RuleEvaluation, "Full audit should include rule evaluation")
		}
		
		// Verify tamper protection
		assert.NotEmpty(suite.T(), record.Signature, "Audit record should be signed")
		assert.True(suite.T(), suite.policyEngine.VerifyAuditRecord(record),
			"Audit record signature should be valid")
		
		suite.T().Logf("Audit record: %s - %s", record.EventType, record.Description)
	}

	// Verify audit log integrity
	assert.True(suite.T(), suite.policyEngine.VerifyAuditLogIntegrity(),
		"Audit log integrity should be maintained")

	// Verify violation statistics are tracked
	stats := suite.policyEngine.GetViolationStatistics()
	assert.Greater(suite.T(), stats.TotalViolations, 0, "Should track total violations")
	assert.Greater(suite.T(), stats.HardViolations, 0, "Should track hard violations")
	assert.NotEmpty(suite.T(), stats.ViolationsByType, "Should track violations by type")
	assert.NotEmpty(suite.T(), stats.ViolationsByRule, "Should track violations by rule")

	suite.T().Logf("Violation statistics: Total=%d, Hard=%d, Types=%v",
		stats.TotalViolations, stats.HardViolations, stats.ViolationsByType)
}

// Test policy rule priority handling
func (suite *PolicyEngineTestSuite) TestPolicyRulePriorityHandling() {
	// Add rules with different priorities
	rules := []PolicyRule{
		{
			Type:        HARD,
			Priority:    1, // Highest priority
			Condition: func(p Process, t OffloadTarget) bool {
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Critical security constraint",
		},
		{
			Type:        HARD,
			Priority:    2, // Lower priority
			Condition: func(p Process, t OffloadTarget) bool {
				return t.DataJurisdiction == "domestic"
			},
			Description: "Data residency requirement",
		},
		{
			Type:        SOFT,
			Priority:    3,
			Condition: func(p Process, t OffloadTarget) bool {
				return t.ComputeCost < 0.10
			},
			Description: "Cost optimization",
		},
		{
			Type:        SOFT,
			Priority:    4, // Lowest priority
			Condition: func(p Process, t OffloadTarget) bool {
				return t.EnergySource == "renewable"
			},
			Description: "Environmental preference",
		},
	}

	for _, rule := range rules {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}

	// Test that higher priority rules are enforced first
	process := Process{
		ID:            "priority-test",
		SecurityLevel: 5,
		DataSensitivity: 3,
	}

	targets := []OffloadTarget{
		{
			ID:               "compliant-target",
			SecurityLevel:    5,                // Meets highest priority rule
			DataJurisdiction: "domestic",       // Meets second priority rule
			ComputeCost:      0.05,            // Meets third priority rule
			EnergySource:     "renewable",     // Meets lowest priority rule
		},
		{
			ID:               "partial-target",
			SecurityLevel:    5,                // Meets highest priority rule
			DataJurisdiction: "foreign",        // Violates second priority rule
			ComputeCost:      0.05,            // Meets third priority rule
			EnergySource:     "fossil",        // Violates lowest priority rule
		},
		{
			ID:               "violating-target",
			SecurityLevel:    2,                // Violates highest priority rule
			DataJurisdiction: "domestic",       // Meets second priority rule
			ComputeCost:      0.20,            // Violates third priority rule
			EnergySource:     "renewable",     // Meets lowest priority rule
		},
	}

	filtered := suite.policyEngine.FilterTargets(process, targets)

	// High priority violations should filter out targets
	// Low priority violations should only affect scoring
	
	targetIDs := []string{}
	for _, target := range filtered {
		targetIDs = append(targetIDs, target.ID)
	}

	// Violating-target should be filtered due to security violation (priority 1)
	assert.NotContains(suite.T(), targetIDs, "violating-target",
		"Target violating highest priority rule should be filtered")

	// Partial-target should be filtered due to jurisdiction violation (priority 2)
	assert.NotContains(suite.T(), targetIDs, "partial-target",
		"Target violating second priority rule should be filtered")

	// Compliant-target should remain
	assert.Contains(suite.T(), targetIDs, "compliant-target",
		"Fully compliant target should remain")

	// Verify priority ordering in rule evaluation
	evalOrder := suite.policyEngine.GetLastEvaluationOrder()
	assert.Equal(suite.T(), "Critical security constraint", evalOrder[0],
		"Highest priority rule should be evaluated first")
	assert.Equal(suite.T(), "Environmental preference", evalOrder[len(evalOrder)-1],
		"Lowest priority rule should be evaluated last")
}

// Test policy rule conflict resolution
func (suite *PolicyEngineTestSuite) TestPolicyRuleConflictResolution() {
	// Add potentially conflicting rules
	conflictingRules := []PolicyRule{
		{
			Type:     SOFT,
			Priority: 2,
			Condition: func(p Process, t OffloadTarget) bool {
				// Prefer low cost
				return t.ComputeCost < 0.10
			},
			Description: "Cost minimization",
		},
		{
			Type:     SOFT,
			Priority: 2, // Same priority
			Condition: func(p Process, t OffloadTarget) bool {
				// Prefer high performance (usually more expensive)
				return t.ProcessingSpeed > 1.5
			},
			Description: "Performance optimization",
		},
		{
			Type:     HARD,
			Priority: 1,
			Condition: func(p Process, t OffloadTarget) bool {
				// Security requirement (may conflict with cost/performance)
				return t.SecurityLevel >= 4
			},
			Description: "High security requirement",
		},
	}

	for _, rule := range conflictingRules {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}

	process := Process{
		ID:            "conflict-test",
		SecurityLevel: 3,
		Priority:      5,
	}

	targets := []OffloadTarget{
		{
			ID:              "cheap-slow-insecure",
			ComputeCost:     0.05,  // Good for cost rule
			ProcessingSpeed: 0.8,   // Bad for performance rule
			SecurityLevel:   2,     // Bad for security rule
		},
		{
			ID:              "expensive-fast-secure",
			ComputeCost:     0.30,  // Bad for cost rule
			ProcessingSpeed: 2.0,   // Good for performance rule
			SecurityLevel:   5,     // Good for security rule
		},
		{
			ID:              "balanced-secure",
			ComputeCost:     0.15,  // Moderate for cost rule
			ProcessingSpeed: 1.2,   // Moderate for performance rule
			SecurityLevel:   4,     // Good for security rule
		},
	}

	filtered := suite.policyEngine.FilterTargets(process, targets)

	// Hard constraint should eliminate insecure target
	targetIDs := []string{}
	for _, target := range filtered {
		targetIDs = append(targetIDs, target.ID)
	}

	assert.NotContains(suite.T(), targetIDs, "cheap-slow-insecure",
		"Target not meeting hard security requirement should be filtered")

	// Remaining targets should be scored based on conflicting soft constraints
	var expensiveTarget, balancedTarget *OffloadTarget
	for i := range filtered {
		switch filtered[i].ID {
		case "expensive-fast-secure":
			expensiveTarget = &filtered[i]
		case "balanced-secure":
			balancedTarget = &filtered[i]
		}
	}

	require.NotNil(suite.T(), expensiveTarget, "Expensive target should remain after filtering")
	require.NotNil(suite.T(), balancedTarget, "Balanced target should remain after filtering")

	// Verify conflict resolution through scoring
	// Policy engine should handle conflicting soft constraints by weighting them
	suite.T().Logf("Expensive target policy bonus: %.3f", expensiveTarget.PolicyBonus)
	suite.T().Logf("Balanced target policy bonus: %.3f", balancedTarget.PolicyBonus)

	// The better balanced target might score higher overall
	// (this depends on the specific conflict resolution strategy)
}

func TestPolicyEngineTestSuite(t *testing.T) {
	suite.Run(t, new(PolicyEngineTestSuite))
}