package autoscaler

import (
	"sort"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// PriorityAnalyzer analyzes process queues with priority weighting
type PriorityAnalyzer struct {
	// Priority thresholds
	HighPriorityThreshold int `json:"high_priority_threshold"` // Priority >= this is considered high
	UrgentPriorityThreshold int `json:"urgent_priority_threshold"` // Priority >= this is urgent
	
	// Weighting factors
	PriorityWeightMultiplier float64 `json:"priority_weight_multiplier"` // How much priority affects demand calculation
	TimeDecayFactor         float64 `json:"time_decay_factor"`          // How wait time affects urgency
	
	// Historical tracking
	priorityHistory []PrioritySnapshot `json:"priority_history"`
	maxHistorySize  int                `json:"max_history_size"`
}

// PrioritySnapshot captures queue state at a point in time
type PrioritySnapshot struct {
	Timestamp         time.Time                             `json:"timestamp"`
	ProcessCountByType map[string]int                       `json:"process_count_by_type"`    // executor_type -> count
	PriorityDistribution map[string]map[int]int             `json:"priority_distribution"`    // executor_type -> priority -> count
	AverageWaitTimes  map[string]time.Duration             `json:"average_wait_times"`       // executor_type -> avg wait
	WeightedDemands   map[string]float64                   `json:"weighted_demands"`         // executor_type -> weighted demand
}

// NewPriorityAnalyzer creates a new priority analyzer
func NewPriorityAnalyzer() *PriorityAnalyzer {
	return &PriorityAnalyzer{
		HighPriorityThreshold:    7,  // Priority 7+ is high
		UrgentPriorityThreshold:  9,  // Priority 9+ is urgent
		PriorityWeightMultiplier: 2.0, // High priority processes count as 2x demand
		TimeDecayFactor:         0.1, // Wait time urgency factor
		maxHistorySize:          100, // Keep 100 snapshots
		priorityHistory:         make([]PrioritySnapshot, 0),
	}
}

// AnalyzeQueueDemand analyzes current queue state and calculates priority-weighted demand
func (pa *PriorityAnalyzer) AnalyzeQueueDemand(processes []models.ColonyOSProcess, executors []models.ColonyOSExecutor) []models.PriorityWeightedDemand {
	currentTime := time.Now()
	
	// Group processes by executor type
	processByType := make(map[string][]models.ColonyOSProcess)
	for _, process := range processes {
		if process.State == models.ProcessStateWaiting {
			executorType := process.Spec.Conditions.ExecutorType
			processByType[executorType] = append(processByType[executorType], process)
		}
	}
	
	var demands []models.PriorityWeightedDemand
	
	// Calculate weighted demand for each executor type
	for executorType, typeProcesses := range processByType {
		demand := pa.calculateWeightedDemand(executorType, typeProcesses, currentTime)
		demands = append(demands, demand)
	}
	
	// Sort by urgency score (highest first)
	sort.Slice(demands, func(i, j int) bool {
		return demands[i].UrgencyScore > demands[j].UrgencyScore
	})
	
	// Record snapshot for historical analysis
	pa.recordSnapshot(demands, currentTime)
	
	return demands
}

// calculateWeightedDemand computes priority-weighted demand for a specific executor type
func (pa *PriorityAnalyzer) calculateWeightedDemand(executorType string, processes []models.ColonyOSProcess, currentTime time.Time) models.PriorityWeightedDemand {
	totalProcesses := len(processes)
	highPriorityCount := 0
	weightedDemand := 0.0
	totalWaitTime := time.Duration(0)
	urgentProcesses := 0
	
	priorityDistribution := make(map[int]int)
	
	// Analyze each process
	for _, process := range processes {
		priority := process.Spec.Priority
		if priority == 0 {
			priority = 5 // Default priority
		}
		
		// Count priority distribution
		priorityDistribution[priority]++
		
		// Calculate wait time
		waitTime := currentTime.Sub(process.SubmissionTime)
		totalWaitTime += waitTime
		
		// Priority weighting
		priorityWeight := 1.0
		if priority >= pa.HighPriorityThreshold {
			highPriorityCount++
			priorityWeight = pa.PriorityWeightMultiplier
		}
		if priority >= pa.UrgentPriorityThreshold {
			urgentProcesses++
			priorityWeight = pa.PriorityWeightMultiplier * 1.5 // Extra weight for urgent
		}
		
		// Time decay factor (longer wait = higher urgency)
		timeWeight := 1.0 + (waitTime.Minutes() * pa.TimeDecayFactor)
		
		weightedDemand += priorityWeight * timeWeight
	}
	
	// Calculate averages and scores
	averageWaitTime := time.Duration(0)
	if totalProcesses > 0 {
		averageWaitTime = totalWaitTime / time.Duration(totalProcesses)
	}
	
	// Urgency score calculation (0-1)
	urgencyScore := 0.0
	if totalProcesses > 0 {
		priorityFactor := float64(highPriorityCount) / float64(totalProcesses)
		urgentFactor := float64(urgentProcesses) / float64(totalProcesses)
		waitTimeFactor := averageWaitTime.Minutes() / 60.0 // Normalize to hours
		
		urgencyScore = (priorityFactor*0.4 + urgentFactor*0.4 + waitTimeFactor*0.2)
		if urgencyScore > 1.0 {
			urgencyScore = 1.0
		}
	}
	
	// Determine recommended action
	recommendedAction := "maintain"
	if urgencyScore > 0.7 {
		recommendedAction = "scale_up"
	} else if urgencyScore < 0.2 && totalProcesses == 0 {
		recommendedAction = "scale_down"
	}
	
	return models.PriorityWeightedDemand{
		ExecutorType:      executorType,
		TotalProcesses:    totalProcesses,
		WeightedDemand:    weightedDemand,
		HighPriorityCount: highPriorityCount,
		AverageWaitTime:   averageWaitTime,
		RecommendedAction: recommendedAction,
		UrgencyScore:      urgencyScore,
		PredictedGrowth:   0.0, // Will be filled by ARIMA predictor
	}
}

// recordSnapshot stores current analysis for historical tracking
func (pa *PriorityAnalyzer) recordSnapshot(demands []models.PriorityWeightedDemand, timestamp time.Time) {
	snapshot := PrioritySnapshot{
		Timestamp:            timestamp,
		ProcessCountByType:   make(map[string]int),
		PriorityDistribution: make(map[string]map[int]int),
		AverageWaitTimes:     make(map[string]time.Duration),
		WeightedDemands:      make(map[string]float64),
	}
	
	// Fill snapshot data
	for _, demand := range demands {
		snapshot.ProcessCountByType[demand.ExecutorType] = demand.TotalProcesses
		snapshot.AverageWaitTimes[demand.ExecutorType] = demand.AverageWaitTime
		snapshot.WeightedDemands[demand.ExecutorType] = demand.WeightedDemand
	}
	
	// Add to history
	pa.priorityHistory = append(pa.priorityHistory, snapshot)
	
	// Maintain history size
	if len(pa.priorityHistory) > pa.maxHistorySize {
		pa.priorityHistory = pa.priorityHistory[1:]
	}
}

// GetDemandTrends returns historical demand trends for ARIMA prediction
func (pa *PriorityAnalyzer) GetDemandTrends(executorType string, lookbackMinutes int) []float64 {
	cutoffTime := time.Now().Add(-time.Duration(lookbackMinutes) * time.Minute)
	var trends []float64
	
	for _, snapshot := range pa.priorityHistory {
		if snapshot.Timestamp.After(cutoffTime) {
			if demand, exists := snapshot.WeightedDemands[executorType]; exists {
				trends = append(trends, demand)
			}
		}
	}
	
	return trends
}

// GetHighestPriorityInQueue returns the highest priority currently waiting for an executor type
func (pa *PriorityAnalyzer) GetHighestPriorityInQueue(processes []models.ColonyOSProcess, executorType string) int {
	highestPriority := 0
	
	for _, process := range processes {
		if process.State == models.ProcessStateWaiting && 
		   process.Spec.Conditions.ExecutorType == executorType {
			if process.Spec.Priority > highestPriority {
				highestPriority = process.Spec.Priority
			}
		}
	}
	
	return highestPriority
}

// CalculateUrgencyByDataLocality factors in data gravity for urgency calculation
func (pa *PriorityAnalyzer) CalculateUrgencyByDataLocality(demand models.PriorityWeightedDemand, dataHints []models.DataLocalityHint) float64 {
	baseUrgency := demand.UrgencyScore
	
	// Factor in data movement costs
	dataUrgencyBoost := 0.0
	for _, hint := range dataHints {
		// High data movement cost increases urgency to deploy closer
		if hint.MovementCost > 1000.0 { // Expensive data movement
			dataUrgencyBoost += 0.2
		}
		
		// Many processes accessing same data increases urgency
		if hint.AccessCount > 5 {
			dataUrgencyBoost += 0.1
		}
	}
	
	finalUrgency := baseUrgency + dataUrgencyBoost
	if finalUrgency > 1.0 {
		finalUrgency = 1.0
	}
	
	return finalUrgency
}