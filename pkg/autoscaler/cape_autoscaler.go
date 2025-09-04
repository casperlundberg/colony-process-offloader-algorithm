package autoscaler

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/learning"
	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// CAPEAutoscaler makes intelligent scaling decisions based on queue state and predictions
type CAPEAutoscaler struct {
	// Core components
	ExecutorCatalog  *ExecutorCatalog
	PriorityAnalyzer *PriorityAnalyzer
	
	// Learning algorithms
	ARIMA    *learning.ARIMA
	EWMA     *learning.EWMA
	CUSUM    *learning.CUSUM
	Thompson *learning.ThompsonSampler
	QLearning *learning.QLearning
	
	// Configuration
	Config           AutoscalerConfig
	ExecutorConfigs  map[string]ExecutorConfig
	
	// State tracking
	LastDecisionTime time.Time
	DecisionHistory  []ScalingDecision
	ActiveExecutors  map[string]int // executor_id -> count
	
	// Metrics
	TotalDecisions   int
	CorrectPredictions int
	TotalCost        float64
}

// AutoscalerConfig contains configuration parameters for the autoscaler
type AutoscalerConfig struct {
	DecisionIntervalMinutes int     `json:"decision_interval_minutes"`
	LookaheadMinutes        int     `json:"lookahead_minutes"`
	MaxCostPerHour          float64 `json:"max_cost_per_hour"`
	TargetSLAPercentile     float64 `json:"target_sla_percentile"`
	ExplorationRate         float64 `json:"exploration_rate"`
	LearningRate            float64 `json:"learning_rate"`
	MinExecutors            map[string]int `json:"min_executors"`
	MaxExecutors            map[string]int `json:"max_executors"`
}

// ExecutorCatalog represents the available executor templates
type ExecutorCatalog struct {
	Executors []ExecutorSpec            `json:"executors"`
	Config    map[string]ExecutorConfig `json:"config"`
	DataTransferCosts map[string]DataTransferCost `json:"data_transfer_costs"`
	Version   string                    `json:"version"`
}

// ExecutorSpec represents a ColonyOS executor specification
type ExecutorSpec struct {
	ID           string    `json:"id"`
	ExecutorName string    `json:"executorname"`
	ExecutorType string    `json:"executortype"`
	Location     Location  `json:"location"`
	Capabilities Capabilities `json:"capabilities"`
	State        int       `json:"state"`
	CommissionTime time.Time `json:"commission_time"`
	LastHeardFrom  time.Time `json:"last_heard_from"`
}

// Location represents geographic location
type Location struct {
	Longitude   float64 `json:"long"`
	Latitude    float64 `json:"lat"`
	Description string  `json:"desc"`
}

// Capabilities represents executor capabilities
type Capabilities struct {
	Hardware Hardware `json:"hardware"`
	Software Software `json:"software"`
}

// Hardware represents hardware specifications
type Hardware struct {
	Model   string  `json:"model"`
	CPU     string  `json:"cpu"`
	Memory  string  `json:"mem"`
	Storage string  `json:"storage"`
	GPU     *GPU    `json:"gpu,omitempty"`
}

// GPU represents GPU specifications
type GPU struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Software represents software specifications
type Software struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// ExecutorConfig contains CAPE-specific configuration for an executor
type ExecutorConfig struct {
	Type                  string  `json:"type"`
	LocationType          string  `json:"location_type"`
	StartupTimeSec        float64 `json:"startup_time_sec"`
	TasksPerSec           float64 `json:"tasks_per_sec"`
	PowerConsumptionW     float64 `json:"power_consumption_w"`
	CostPerHour           float64 `json:"cost_per_hour"`
	CostPerGBEgress       float64 `json:"cost_per_gb_egress"`
	MaxInstances          float64 `json:"max_instances"`
	MinLeaseTimeMin       float64 `json:"min_lease_time_min"`
	SupportsSpot          bool    `json:"supports_spot"`
}

// DataTransferCost represents network transfer costs between locations
type DataTransferCost struct {
	CostPerGB           float64 `json:"cost_per_gb"`
	TransferTimeSecPerGB float64 `json:"transfer_time_sec_per_gb"`
}

// ScalingDecision represents a scaling action to take
type ScalingDecision struct {
	DecisionID      string    `json:"decision_id"`
	Timestamp       time.Time `json:"timestamp"`
	Action          string    `json:"action"` // "deploy", "remove", "maintain"
	ExecutorID      string    `json:"executor_id"`
	Count           int       `json:"count"`
	Reason          string    `json:"reason"`
	PredictedDemand float64   `json:"predicted_demand"`
	ConfidenceScore float64   `json:"confidence_score"`
	EstimatedCost   float64   `json:"estimated_cost"`
	ReadyInSeconds  int       `json:"ready_in_seconds"`
}

// NewCAPEAutoscaler creates a new CAPE autoscaler
func NewCAPEAutoscaler(configPath, catalogPath string) (*CAPEAutoscaler, error) {
	// Load configuration
	config, err := loadAutoscalerConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	// Load executor catalog
	catalog, err := loadExecutorCatalog(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load catalog: %w", err)
	}
	
	// Initialize learning algorithms
	arima := learning.NewARIMA()
	ewma := learning.NewEWMADefault()
	cusum := learning.NewCUSUMFromSigma(1.0, 0.0)
	
	// Initialize Thompson Sampling
	thompson := learning.NewThompsonSampler(0.1)
	
	// Initialize Q-Learning
	qlearning := learning.NewQLearning(0.1, 0.9, 0.3, 100)
	
	return &CAPEAutoscaler{
		ExecutorCatalog:  catalog,
		PriorityAnalyzer: NewPriorityAnalyzer(),
		ARIMA:           arima,
		EWMA:            ewma,
		CUSUM:           cusum,
		Thompson:        thompson,
		QLearning:       qlearning,
		Config:          config,
		ExecutorConfigs: catalog.Config,
		ActiveExecutors: make(map[string]int),
		DecisionHistory: make([]ScalingDecision, 0),
	}, nil
}

// MakeScalingDecision analyzes the queue and makes scaling recommendations
func (ca *CAPEAutoscaler) MakeScalingDecision(queueState []models.ColonyOSProcess, currentExecutors []ExecutorSpec) []ScalingDecision {
	decisions := make([]ScalingDecision, 0)
	
	// For simulation: Scale aggressively when there are queued processes
	if len(queueState) == 0 {
		// No processes to handle, consider scaling down
		return ca.considerScaleDownAll(currentExecutors)
	}
	
	// Group processes by executor type
	processByType := ca.groupProcessesByExecutorType(queueState)
	currentExecutorCounts := ca.countCurrentExecutorsByType(currentExecutors)
	
	
	// For each executor type that has processes, ensure we have enough capacity
	for executorType, processes := range processByType {
		if len(processes) == 0 {
			continue
		}
		
		currentCount := currentExecutorCounts[executorType]
		
		// Get executor capacity (processes per minute)
		executorCapacity := ca.getExecutorCapacity(executorType)
		if executorCapacity == 0 {
			executorCapacity = 10 // default capacity
		}
		
		// Calculate needed executors (aim to clear queue in 1 minute)
		targetProcessingRate := float64(len(processes)) // per minute
		neededExecutors := int(math.Ceil(targetProcessingRate / executorCapacity))
		
		
		if neededExecutors > currentCount {
			deployCount := neededExecutors - currentCount
			
			decision := ScalingDecision{
				DecisionID:      fmt.Sprintf("deploy-%s-%d", executorType, time.Now().Unix()),
				Timestamp:       time.Now(),
				Action:          "deploy",
				ExecutorID:      ca.selectExecutorForType(executorType),
				Count:           deployCount,
				Reason:          fmt.Sprintf("Queue buildup: %d processes for %s executors", len(processes), executorType),
				PredictedDemand: float64(len(processes)),
				ConfidenceScore: 0.9,
				EstimatedCost:   ca.calculateDeploymentCost(executorType, deployCount),
				ReadyInSeconds:  ca.getExecutorStartupTime(executorType),
			}
			
			decisions = append(decisions, decision)
		}
	}
	
	// Record decisions
	ca.DecisionHistory = append(ca.DecisionHistory, decisions...)
	ca.LastDecisionTime = time.Now()
	ca.TotalDecisions += len(decisions)
	
	return decisions
}

// considerScaleUp evaluates whether to deploy more executors
func (ca *CAPEAutoscaler) considerScaleUp(demand models.PriorityWeightedDemand, queueState []models.ColonyOSProcess) []ScalingDecision {
	decisions := make([]ScalingDecision, 0)
	
	// Get historical demand for ARIMA prediction
	historicalDemand := ca.PriorityAnalyzer.GetDemandTrends(demand.ExecutorType, 60)
	
	// Add current demand
	ca.ARIMA.AddObservation(demand.WeightedDemand)
	
	// Predict future demand
	predictedDemand, _ := ca.ARIMA.Predict() // Single prediction
	
	// Smooth prediction with EWMA
	smoothedDemand := ca.EWMA.Update(predictedDemand)
	
	// Check for anomaly with CUSUM
	cusumResult := ca.CUSUM.Update(smoothedDemand)
	anomalyDetected := cusumResult.IsAnomaly
	
	// Find best executor for this workload
	bestExecutor := ca.selectBestExecutor(demand, queueState)
	if bestExecutor == nil {
		return decisions
	}
	
	// Calculate how many instances needed
	executorConfig := ca.ExecutorConfigs[bestExecutor.ID]
	capacityPerExecutor := executorConfig.TasksPerSec * 60 // per minute
	instancesNeeded := int(math.Ceil(smoothedDemand / capacityPerExecutor))
	
	// Apply limits
	currentCount := ca.ActiveExecutors[bestExecutor.ID]
	maxAllowed := int(executorConfig.MaxInstances) - currentCount
	instancesNeeded = min(instancesNeeded, maxAllowed)
	
	if instancesNeeded > 0 {
		// Consider data transfer costs
		dataTransferCost := ca.calculateDataTransferCost(demand, bestExecutor)
		
		decision := ScalingDecision{
			DecisionID:      fmt.Sprintf("scale-%s-%d", bestExecutor.ID, time.Now().Unix()),
			Timestamp:       time.Now(),
			Action:          "deploy",
			ExecutorID:      bestExecutor.ID,
			Count:           instancesNeeded,
			Reason:          ca.generateScaleUpReason(demand, anomalyDetected, predictedDemand),
			PredictedDemand: smoothedDemand,
			ConfidenceScore: ca.calculateConfidence(historicalDemand),
			EstimatedCost:   executorConfig.CostPerHour * float64(instancesNeeded) + dataTransferCost,
			ReadyInSeconds:  int(executorConfig.StartupTimeSec),
		}
		
		decisions = append(decisions, decision)
		ca.ActiveExecutors[bestExecutor.ID] += instancesNeeded
		
		// Update Thompson Sampling (simplified)
		// ca.Thompson.Update(bestExecutor.ID, 1.0) // Will implement later
	}
	
	return decisions
}

// considerScaleDown evaluates whether to remove executors
func (ca *CAPEAutoscaler) considerScaleDown(demand models.PriorityWeightedDemand, currentExecutors []ExecutorSpec) []ScalingDecision {
	decisions := make([]ScalingDecision, 0)
	
	// Find executors of this type that can be removed
	for executorID, count := range ca.ActiveExecutors {
		if count > 0 {
			// Check if this executor handles the demand type
			executor := ca.findExecutor(executorID)
			if executor != nil && executor.ExecutorType == demand.ExecutorType {
				// Keep minimum instances
				minInstances := ca.Config.MinExecutors[executorID]
				removeCount := min(1, count-minInstances)
				
				if removeCount > 0 {
					executorConfig := ca.ExecutorConfigs[executorID]
					
					decision := ScalingDecision{
						DecisionID:      fmt.Sprintf("scale-%s-%d", executorID, time.Now().Unix()),
						Timestamp:       time.Now(),
						Action:          "remove",
						ExecutorID:      executorID,
						Count:           removeCount,
						Reason:          "Low demand detected, scaling down to save costs",
						PredictedDemand: demand.WeightedDemand,
						ConfidenceScore: 0.8,
						EstimatedCost:   -executorConfig.CostPerHour * float64(removeCount),
						ReadyInSeconds:  30,
					}
					
					decisions = append(decisions, decision)
					ca.ActiveExecutors[executorID] -= removeCount
				}
			}
		}
	}
	
	return decisions
}

// selectBestExecutor chooses the optimal executor for the workload
func (ca *CAPEAutoscaler) selectBestExecutor(demand models.PriorityWeightedDemand, queueState []models.ColonyOSProcess) *ExecutorSpec {
	candidates := make([]ExecutorSpec, 0)
	
	// Filter executors by type
	for _, executor := range ca.ExecutorCatalog.Executors {
		if executor.ExecutorType == demand.ExecutorType {
			candidates = append(candidates, executor)
		}
	}
	
	if len(candidates) == 0 {
		return nil
	}
	
	// Score each candidate
	type scoredExecutor struct {
		executor ExecutorSpec
		score    float64
	}
	
	scored := make([]scoredExecutor, 0)
	
	for _, executor := range candidates {
		config := ca.ExecutorConfigs[executor.ID]
		
		// Multi-factor scoring
		score := 0.0
		
		// Performance score
		performanceScore := config.TasksPerSec / (float64(config.StartupTimeSec)/60.0 + 1.0)
		score += performanceScore * 10
		
		// Cost efficiency
		costScore := config.TasksPerSec / (config.CostPerHour + 0.01)
		score += costScore * 5
		
		// Urgency bonus for fast startup
		if demand.UrgencyScore > 0.7 && config.StartupTimeSec < 120 {
			score += 20
		}
		
		// Data locality bonus
		for _, hint := range demand.DataLocalityFactors {
			if ca.isNearData(executor, hint) {
				score += 30
			}
		}
		
		// GPU matching
		hasGPU := executor.Capabilities.Hardware.GPU != nil
		for _, process := range queueState {
			if process.Spec.Conditions.RequiredGPU && hasGPU {
				score += 15
				break
			}
		}
		
		scored = append(scored, scoredExecutor{executor, score})
	}
	
	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	
	// Use Thompson Sampling for exploration (simplified)
	// if ca.Config.ExplorationRate > 0 && len(scored) > 1 {
	//     // Will implement proper exploration later
	// }
	
	// Return highest scored
	if len(scored) > 0 {
		return &scored[0].executor
	}
	
	return nil
}

// calculateDataTransferCost estimates data movement costs
func (ca *CAPEAutoscaler) calculateDataTransferCost(demand models.PriorityWeightedDemand, executor *ExecutorSpec) float64 {
	totalCost := 0.0
	
	for _, dataHint := range demand.DataLocalityFactors {
		// Check if executor is in same location as data
		if !ca.isNearData(*executor, dataHint) {
			// Look up transfer cost
			key := fmt.Sprintf("%s_to_%s", dataHint.DataSource, executor.Location.Description)
			if transferCost, exists := ca.ExecutorCatalog.DataTransferCosts[key]; exists {
				totalCost += dataHint.SizeGB * transferCost.CostPerGB
			} else {
				// Default transfer cost
				totalCost += dataHint.SizeGB * 0.1
			}
		}
	}
	
	return totalCost
}

// isNearData checks if executor is close to data location
func (ca *CAPEAutoscaler) isNearData(executor ExecutorSpec, dataHint models.DataLocalityHint) bool {
	// Simple distance check (could be more sophisticated)
	distance := ca.calculateDistance(
		executor.Location.Latitude, executor.Location.Longitude,
		dataHint.Location.Latitude, dataHint.Location.Longitude,
	)
	
	// Consider "near" if within 500km
	return distance < 500
}

// calculateDistance computes distance between two points (in km)
func (ca *CAPEAutoscaler) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // km
	
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180
	
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadius * c
}

// generateScaleUpReason creates human-readable scaling reason
func (ca *CAPEAutoscaler) generateScaleUpReason(demand models.PriorityWeightedDemand, anomalyDetected bool, predictedDemand float64) string {
	reason := fmt.Sprintf("High priority processes detected (urgency: %.2f)", demand.UrgencyScore)
	
	if demand.HighPriorityCount > 0 {
		reason += fmt.Sprintf(", %d high-priority tasks waiting", demand.HighPriorityCount)
	}
	
	if anomalyDetected {
		reason += ", spike detected by CUSUM"
	}
	
	if predictedDemand > demand.WeightedDemand*1.5 {
		reason += fmt.Sprintf(", ARIMA predicts %.0f%% increase", (predictedDemand/demand.WeightedDemand-1)*100)
	}
	
	return reason
}

// calculateConfidence estimates confidence in the scaling decision
func (ca *CAPEAutoscaler) calculateConfidence(historicalData []float64) float64 {
	if len(historicalData) < 10 {
		return 0.5 // Low confidence with little data
	}
	
	// Calculate variance
	mean := 0.0
	for _, v := range historicalData {
		mean += v
	}
	mean /= float64(len(historicalData))
	
	variance := 0.0
	for _, v := range historicalData {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(historicalData))
	
	// Lower variance = higher confidence
	confidence := 1.0 / (1.0 + variance/mean)
	
	// Factor in prediction accuracy
	if ca.TotalDecisions > 0 {
		accuracy := float64(ca.CorrectPredictions) / float64(ca.TotalDecisions)
		confidence = confidence*0.5 + accuracy*0.5
	}
	
	return math.Min(confidence, 0.95)
}

// findExecutor finds an executor by ID
func (ca *CAPEAutoscaler) findExecutor(executorID string) *ExecutorSpec {
	for _, executor := range ca.ExecutorCatalog.Executors {
		if executor.ID == executorID {
			return &executor
		}
	}
	return nil
}

// LearnFromOutcome updates learning algorithms based on actual outcomes
func (ca *CAPEAutoscaler) LearnFromOutcome(decision ScalingDecision, actualDemand float64, slasMet bool) {
	// Update ARIMA with actual demand
	ca.ARIMA.AddObservation(actualDemand)
	
	// Update prediction accuracy
	if slasMet && math.Abs(actualDemand-decision.PredictedDemand) < actualDemand*0.2 {
		ca.CorrectPredictions++
	}
	
	// Update Q-Learning (simplified for now)
	// state := fmt.Sprintf("demand_%.0f_urgency_%.1f", actualDemand, decision.ConfidenceScore)
	// action := fmt.Sprintf("%s_%d", decision.Action, decision.Count)
	// qReward := -decision.EstimatedCost // Negative cost as reward
	// if slasMet {
	//     qReward += 100 // Bonus for meeting SLAs
	// }
	// nextState := fmt.Sprintf("demand_%.0f", actualDemand)
	// ca.QLearning.Learn(state, action, qReward, nextState)
}

// Helper functions

func loadAutoscalerConfig(path string) (AutoscalerConfig, error) {
	var config AutoscalerConfig
	
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	
	err = json.Unmarshal(data, &config)
	return config, err
}

func loadExecutorCatalog(path string) (*ExecutorCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	// Parse into a map first to handle mixed types in config section
	var rawData map[string]interface{}
	err = json.Unmarshal(data, &rawData)
	if err != nil {
		return nil, err
	}
	
	// Create catalog
	catalog := &ExecutorCatalog{
		Config: make(map[string]ExecutorConfig),
	}
	
	// Unmarshal executors array
	if executorsData, ok := rawData["executors"]; ok {
		executorsJSON, _ := json.Marshal(executorsData)
		err = json.Unmarshal(executorsJSON, &catalog.Executors)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal executors: %w", err)
		}
	}
	
	// Unmarshal config map, skipping non-executor entries
	if configData, ok := rawData["config"].(map[string]interface{}); ok {
		for key, value := range configData {
			// Skip comment fields
			if key == "_comment" {
				continue
			}
			
			// Marshal and unmarshal individual executor config
			valueJSON, _ := json.Marshal(value)
			var execConfig ExecutorConfig
			err = json.Unmarshal(valueJSON, &execConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal executor config %s: %w", key, err)
			}
			catalog.Config[key] = execConfig
		}
	}
	
	// Unmarshal data transfer costs, skipping comments
	if transferData, ok := rawData["data_transfer_costs"].(map[string]interface{}); ok {
		catalog.DataTransferCosts = make(map[string]DataTransferCost)
		for key, value := range transferData {
			// Skip comment fields
			if key == "_comment" {
				continue
			}
			
			// Marshal and unmarshal individual transfer cost
			valueJSON, _ := json.Marshal(value)
			var transferCost DataTransferCost
			err = json.Unmarshal(valueJSON, &transferCost)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal data transfer cost %s: %w", key, err)
			}
			catalog.DataTransferCosts[key] = transferCost
		}
	}
	
	if version, ok := rawData["version"].(string); ok {
		catalog.Version = version
	}
	
	return catalog, nil
}

// groupProcessesByExecutorType groups processes by their required executor type
func (ca *CAPEAutoscaler) groupProcessesByExecutorType(processes []models.ColonyOSProcess) map[string][]models.ColonyOSProcess {
	groups := make(map[string][]models.ColonyOSProcess)
	
	for _, process := range processes {
		executorType := process.Spec.Conditions.ExecutorType
		if executorType == "" {
			executorType = "cloud" // default
		}
		groups[executorType] = append(groups[executorType], process)
	}
	
	return groups
}

// countCurrentExecutorsByType counts currently active executors by type
func (ca *CAPEAutoscaler) countCurrentExecutorsByType(executors []ExecutorSpec) map[string]int {
	counts := make(map[string]int)
	
	for _, executor := range executors {
		counts[executor.ExecutorType]++
	}
	
	return counts
}

// getExecutorCapacity gets the processing capacity of an executor type (processes per minute)
func (ca *CAPEAutoscaler) getExecutorCapacity(executorType string) float64 {
	// Find first executor of this type in catalog
	for _, executor := range ca.ExecutorCatalog.Executors {
		if executor.ExecutorType == executorType {
			if config, exists := ca.ExecutorConfigs[executor.ID]; exists {
				// For simulation: use a more realistic capacity
				// The simulation processes in 1-minute steps, so capacity should be processes per minute step
				return config.TasksPerSec * 0.5 // Much more conservative: ~0.5 tasks per simulation step
			}
		}
	}
	return 5.0 // default: ~5 processes per minute step
}

// selectExecutorForType selects the best executor ID for a given type
func (ca *CAPEAutoscaler) selectExecutorForType(executorType string) string {
	// Find first available executor of this type
	for _, executor := range ca.ExecutorCatalog.Executors {
		if executor.ExecutorType == executorType {
			return executor.ID
		}
	}
	return fmt.Sprintf("exec-%s-default", executorType)
}

// calculateDeploymentCost estimates the cost of deploying executors
func (ca *CAPEAutoscaler) calculateDeploymentCost(executorType string, count int) float64 {
	// Find executor config for cost calculation
	for _, executor := range ca.ExecutorCatalog.Executors {
		if executor.ExecutorType == executorType {
			if config, exists := ca.ExecutorConfigs[executor.ID]; exists {
				return config.CostPerHour * float64(count) * 0.1 // Assume 6-minute deployment
			}
		}
	}
	return float64(count) * 0.5 // default cost
}

// getExecutorStartupTime gets startup time for executor type
func (ca *CAPEAutoscaler) getExecutorStartupTime(executorType string) int {
	// Find executor config for startup time
	for _, executor := range ca.ExecutorCatalog.Executors {
		if executor.ExecutorType == executorType {
			if config, exists := ca.ExecutorConfigs[executor.ID]; exists {
				return int(config.StartupTimeSec)
			}
		}
	}
	return 60 // default: 1 minute
}

// considerScaleDownAll considers scaling down when no processes are queued
func (ca *CAPEAutoscaler) considerScaleDownAll(currentExecutors []ExecutorSpec) []ScalingDecision {
	decisions := make([]ScalingDecision, 0)
	
	// For simulation, keep some minimal capacity but scale down excess
	executorCounts := ca.countCurrentExecutorsByType(currentExecutors)
	
	for executorType, count := range executorCounts {
		if count > 1 { // Keep at least 1 executor of each type
			scaleDownCount := count - 1
			
			decision := ScalingDecision{
				DecisionID:      fmt.Sprintf("remove-%s-%d", executorType, time.Now().Unix()),
				Timestamp:       time.Now(),
				Action:          "remove",
				ExecutorID:      ca.selectExecutorForType(executorType),
				Count:           scaleDownCount,
				Reason:          "No queued processes, scaling down excess capacity",
				PredictedDemand: 0,
				ConfidenceScore: 0.8,
				EstimatedCost:   -ca.calculateDeploymentCost(executorType, scaleDownCount),
				ReadyInSeconds:  30,
			}
			
			decisions = append(decisions, decision)
		}
	}
	
	return decisions
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}