package policy_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/policy"
)

// PolicyEngine test requirements:
// 1. Safety constraints are immutable during execution
// 2. Violations must trigger immediate corrective action
// 3. All safety violations must be logged and auditable
// 4. Hard constraints must never be violated
// 5. Soft constraints should influence scoring but not filter

type PolicyEngineTestSuite struct {
	suite.Suite
	policyEngine *policy.PolicyEngine
}

func (suite *PolicyEngineTestSuite) SetupTest() {
	suite.policyEngine = policy.NewPolicyEngine()
}

// Test that hard constraints must never be violated
func (suite *PolicyEngineTestSuite) TestHardConstraintsNeverViolated() {
	// Add critical safety constraints
	safetyConstraints := []policy.PolicyRule{
		{
			Type:     models.HARD,
			Priority: 1,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Safety-critical processes must stay local
				return !(p.SafetyCritical && t.Type != models.LOCAL)
			},
			Description: "Safety-critical processes must stay local",
		},
		{
			Type:     models.HARD,
			Priority: 1,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// High-security processes need secure targets
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Security level compliance",
		},
		{
			Type:     models.HARD,
			Priority: 1,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
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
		process           models.Process
		targets           []models.OffloadTarget
		expectedFiltered  int
		violationTypes    []string
	}{
		{
			name: "safety_critical_violation",
			process: models.Process{
				ID:             "safety-test-1",
				SafetyCritical: true,
				SecurityLevel:  3,
				DataSensitivity: 2,
				EstimatedDuration: 30 * time.Second,
				Priority:       5,
			},
			targets: []models.OffloadTarget{
				{ID: "local", Type: models.LOCAL, SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "edge", Type: models.EDGE, SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "cloud", Type: models.PUBLIC_CLOUD, SecurityLevel: 3, DataJurisdiction: "domestic"},
			},
			expectedFiltered: 1, // Only local should remain
			violationTypes:   []string{"safety_critical"},
		},
		{
			name: "security_level_violation",
			process: models.Process{
				ID:             "security-test-1",
				SafetyCritical: false,
				SecurityLevel:  5, // Very high security requirement
				DataSensitivity: 2,
				EstimatedDuration: 30 * time.Second,
				Priority:       7,
			},
			targets: []models.OffloadTarget{
				{ID: "low-sec", Type: models.EDGE, SecurityLevel: 2, DataJurisdiction: "domestic"},
				{ID: "med-sec", Type: models.EDGE, SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "high-sec", Type: models.PRIVATE_CLOUD, SecurityLevel: 5, DataJurisdiction: "domestic"},
			},
			expectedFiltered: 1, // Only high-sec should remain
			violationTypes:   []string{"security_level"},
		},
		{
			name: "data_sovereignty_violation",
			process: models.Process{
				ID:             "sovereignty-test-1",
				SafetyCritical: false,
				SecurityLevel:  2,
				DataSensitivity: 5, // Very sensitive data
				EstimatedDuration: 30 * time.Second,
				Priority:       6,
			},
			targets: []models.OffloadTarget{
				{ID: "domestic", Type: models.EDGE, SecurityLevel: 3, DataJurisdiction: "domestic"},
				{ID: "international", Type: models.PUBLIC_CLOUD, SecurityLevel: 3, DataJurisdiction: "international"},
				{ID: "eu", Type: models.PUBLIC_CLOUD, SecurityLevel: 3, DataJurisdiction: "eu"},
			},
			expectedFiltered: 1, // Only domestic should remain
			violationTypes:   []string{"data_sovereignty"},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Filter targets by policy
			filtered := suite.policyEngine.FilterTargetsByPolicy(tc.process, tc.targets)
			
			// Verify correct number of targets passed
			assert.Len(suite.T(), filtered, tc.expectedFiltered,
				"Should filter targets correctly based on hard constraints")
			
			// Check violations were logged
			violations := suite.policyEngine.GetViolations()
			assert.Greater(suite.T(), len(violations), 0,
				"Violations should be logged")
			
			// Verify audit logs exist
			auditLogs := suite.policyEngine.GetAuditLogs()
			assert.Greater(suite.T(), len(auditLogs), 0,
				"Audit logs should be created for all evaluations")
			
			// Clear violations for next test
			suite.policyEngine.ClearViolations()
		})
	}
}

// Test that safety constraints are immutable during execution
func (suite *PolicyEngineTestSuite) TestSafetyConstraintsImmutability() {
	// Set initial safety constraints
	initialConstraints := policy.SafetyConstraints{
		MinLocalCompute:       0.3,
		MinLocalMemory:        0.3,
		MaxConcurrentOffloads: 5,
		DataSovereignty:       true,
		SecurityClearance:     true,
	}
	
	suite.policyEngine.SetSafetyConstraints(initialConstraints)
	
	// Make engine immutable (simulating execution phase)
	suite.policyEngine.SetImmutable(true)
	
	// Try to modify constraints
	newConstraints := policy.SafetyConstraints{
		MinLocalCompute:       0.1, // Trying to reduce safety margin
		MinLocalMemory:        0.1,
		MaxConcurrentOffloads: 20,
		DataSovereignty:       false,
		SecurityClearance:     false,
	}
	
	suite.policyEngine.SetSafetyConstraints(newConstraints)
	
	// Verify constraints didn't change
	actualConstraints := suite.policyEngine.GetSafetyConstraints()
	assert.Equal(suite.T(), initialConstraints.MinLocalCompute, actualConstraints.MinLocalCompute,
		"Safety constraints should be immutable during execution")
	assert.Equal(suite.T(), initialConstraints.MinLocalMemory, actualConstraints.MinLocalMemory,
		"Safety constraints should be immutable during execution")
	
	// Try to add new rule while immutable
	err := suite.policyEngine.AddRule(policy.PolicyRule{
		Type:        models.HARD,
		Description: "New rule during execution",
		Condition: func(p models.Process, t models.OffloadTarget) bool {
			return true
		},
	})
	
	assert.Error(suite.T(), err, "Should not be able to add rules while immutable")
}

// Test that violations trigger immediate corrective action
func (suite *PolicyEngineTestSuite) TestViolationCorrectiveAction() {
	// Add a hard constraint
	rule := policy.PolicyRule{
		Type:     models.HARD,
		Priority: 1,
		Condition: func(p models.Process, t models.OffloadTarget) bool {
			return p.RealTime == false || t.NetworkLatency < 10*time.Millisecond
		},
		Description: "Real-time processes need low latency",
	}
	
	err := suite.policyEngine.AddRule(rule)
	require.NoError(suite.T(), err)
	
	// Test with violating process and target
	process := models.Process{
		ID:                "realtime-1",
		RealTime:          true,
		EstimatedDuration: 10 * time.Second,
		Priority:          9,
	}
	
	target := models.OffloadTarget{
		ID:             "high-latency",
		Type:           models.PUBLIC_CLOUD,
		NetworkLatency: 100 * time.Millisecond, // Too high for real-time
		SecurityLevel:  3,
	}
	
	// Evaluate policy
	evaluation := suite.policyEngine.EvaluatePolicy(process, target)
	
	// Verify violation was detected and blocked
	assert.False(suite.T(), evaluation.Allowed,
		"Violating decision should be blocked immediately")
	assert.Len(suite.T(), evaluation.ViolatedRules, 1,
		"Should have one violated rule")
	
	// Check that violation was logged
	violations := suite.policyEngine.GetViolations()
	assert.Len(suite.T(), violations, 1,
		"Violation should be logged")
	assert.Equal(suite.T(), policy.BLOCKED, violations[0].Action,
		"Hard violation should result in BLOCKED action")
}

// Test that soft constraints influence scoring but don't filter
func (suite *PolicyEngineTestSuite) TestSoftConstraintScoring() {
	// Add soft constraints
	softRules := []policy.PolicyRule{
		{
			Type:     models.SOFT,
			Priority: 2,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Prefer low-cost targets
				return t.ComputeCost < 0.10
			},
			Description: "Prefer low-cost targets",
		},
		{
			Type:     models.SOFT,
			Priority: 3,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				// Prefer high-reliability targets
				return t.Reliability > 0.95
			},
			Description: "Prefer high-reliability targets",
		},
	}
	
	for _, rule := range softRules {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}
	
	process := models.Process{
		ID:                "test-soft",
		EstimatedDuration: 30 * time.Second,
		Priority:          5,
	}
	
	// Test with target that violates soft constraints
	expensiveTarget := models.OffloadTarget{
		ID:          "expensive",
		Type:        models.PUBLIC_CLOUD,
		ComputeCost: 0.50, // Expensive
		Reliability: 0.90, // Below preferred reliability
	}
	
	// Test with target that meets soft constraints
	cheapTarget := models.OffloadTarget{
		ID:          "cheap",
		Type:        models.EDGE,
		ComputeCost: 0.05, // Cheap
		Reliability: 0.98, // High reliability
	}
	
	// Evaluate both targets
	expensiveEval := suite.policyEngine.EvaluatePolicy(process, expensiveTarget)
	cheapEval := suite.policyEngine.EvaluatePolicy(process, cheapTarget)
	
	// Both should be allowed (soft constraints don't block)
	assert.True(suite.T(), expensiveEval.Allowed,
		"Soft constraint violations should not block decisions")
	assert.True(suite.T(), cheapEval.Allowed,
		"Target meeting soft constraints should be allowed")
	
	// But scores should be different
	assert.Less(suite.T(), expensiveEval.ScoreAdjustment, cheapEval.ScoreAdjustment,
		"Target violating soft constraints should have lower score adjustment")
	
	// Check that soft violations were logged differently
	violations := suite.policyEngine.GetViolations()
	softViolationFound := false
	for _, v := range violations {
		if v.RuleType == models.SOFT {
			softViolationFound = true
			assert.Equal(suite.T(), policy.WARNED, v.Action,
				"Soft violations should result in WARNED action")
		}
	}
	assert.True(suite.T(), softViolationFound,
		"Soft violations should be logged")
}

// Test that all safety violations are logged and auditable
func (suite *PolicyEngineTestSuite) TestAuditLogging() {
	// Add rules
	rules := []policy.PolicyRule{
		{
			Type:     models.HARD,
			Priority: 1,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				return p.SecurityLevel <= t.SecurityLevel
			},
			Description: "Security level check",
		},
		{
			Type:     models.SOFT,
			Priority: 2,
			Condition: func(p models.Process, t models.OffloadTarget) bool {
				return t.CurrentLoad < 0.8
			},
			Description: "Load preference",
		},
	}
	
	for _, rule := range rules {
		err := suite.policyEngine.AddRule(rule)
		require.NoError(suite.T(), err)
	}
	
	// Run multiple evaluations
	processes := []models.Process{
		{ID: "p1", SecurityLevel: 3, EstimatedDuration: 10 * time.Second, Priority: 5},
		{ID: "p2", SecurityLevel: 5, EstimatedDuration: 10 * time.Second, Priority: 7},
		{ID: "p3", SecurityLevel: 1, EstimatedDuration: 10 * time.Second, Priority: 3},
	}
	
	targets := []models.OffloadTarget{
		{ID: "t1", SecurityLevel: 3, CurrentLoad: 0.5, Type: models.EDGE},
		{ID: "t2", SecurityLevel: 2, CurrentLoad: 0.9, Type: models.PUBLIC_CLOUD},
	}
	
	evaluationCount := 0
	for _, process := range processes {
		for _, target := range targets {
			suite.policyEngine.EvaluatePolicy(process, target)
			evaluationCount++
		}
	}
	
	// Check audit logs
	auditLogs := suite.policyEngine.GetAuditLogs()
	assert.GreaterOrEqual(suite.T(), len(auditLogs), evaluationCount,
		"All evaluations should be logged")
	
	// Verify audit log structure
	for _, log := range auditLogs {
		assert.NotEmpty(suite.T(), log.ID, "Audit log should have ID")
		assert.NotZero(suite.T(), log.Timestamp, "Audit log should have timestamp")
		assert.NotEmpty(suite.T(), log.EventType, "Audit log should have event type")
		assert.NotEmpty(suite.T(), log.ProcessID, "Audit log should have process ID")
		assert.NotEmpty(suite.T(), log.TargetID, "Audit log should have target ID")
		assert.NotEmpty(suite.T(), log.Decision, "Audit log should have decision")
	}
	
	// Check statistics
	stats := suite.policyEngine.GetStats()
	assert.Equal(suite.T(), int64(evaluationCount), stats.TotalEvaluations,
		"Statistics should track all evaluations")
	assert.Greater(suite.T(), stats.HardViolations+stats.SoftViolations, int64(0),
		"Statistics should track violations")
	assert.Greater(suite.T(), stats.AverageEvalTime, time.Duration(0),
		"Statistics should track evaluation time")
}

// Run the test suite
func TestPolicyEngineSuite(t *testing.T) {
	suite.Run(t, new(PolicyEngineTestSuite))
}