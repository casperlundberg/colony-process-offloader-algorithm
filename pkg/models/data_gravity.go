package models

import (
	"math"
)

// DataGravityModel implements the data gravity scoring concept
// Based on the idea that compute should move to data, not vice versa
type DataGravityModel struct {
	// Gravity scores between different location types
	gravityMatrix map[DataLocation]map[DataLocation]float64
}

// NewDataGravityModel creates a new data gravity model with default scores
func NewDataGravityModel() *DataGravityModel {
	return &DataGravityModel{
		gravityMatrix: defaultGravityScores(),
	}
}

// defaultGravityScores returns the default gravity scores between locations
func defaultGravityScores() map[DataLocation]map[DataLocation]float64 {
	return map[DataLocation]map[DataLocation]float64{
		DataLocationLocal: {
			DataLocationLocal: 1.0,  // Perfect locality
			DataLocationEdge:  0.8,  // High locality within local network
			DataLocationFog:   0.7,  // Good locality for mobile edge
			DataLocationCloud: 0.3,  // Moderate locality, network dependency
			DataLocationHPC:   0.2,  // Low locality, specialized link needed
		},
		DataLocationEdge: {
			DataLocationLocal: 0.8,
			DataLocationEdge:  1.0,  // Perfect edge locality
			DataLocationFog:   0.9,  // Very high edge-fog affinity
			DataLocationCloud: 0.4,  // Moderate edge-cloud connection
			DataLocationHPC:   0.1,  // Poor edge-HPC connectivity
		},
		DataLocationFog: {
			DataLocationLocal: 0.7,
			DataLocationEdge:  0.9,
			DataLocationFog:   1.0,  // Perfect fog locality
			DataLocationCloud: 0.3,  // Limited fog-cloud bandwidth
			DataLocationHPC:   0.1,  // Very poor fog-HPC connectivity
		},
		DataLocationCloud: {
			DataLocationLocal: 0.3,
			DataLocationEdge:  0.4,
			DataLocationFog:   0.3,
			DataLocationCloud: 1.0,  // Perfect cloud locality
			DataLocationHPC:   0.6,  // Good cloud-HPC interconnects
		},
		DataLocationHPC: {
			DataLocationLocal: 0.2,
			DataLocationEdge:  0.1,
			DataLocationFog:   0.1,
			DataLocationCloud: 0.6,
			DataLocationHPC:   1.0,  // Perfect HPC locality
		},
	}
}

// CalculateDataGravityScore calculates the gravity score between data location and executor location
func (dgm *DataGravityModel) CalculateDataGravityScore(dataLocation, executorLocation DataLocation) float64 {
	if executorMatrix, exists := dgm.gravityMatrix[dataLocation]; exists {
		if score, exists := executorMatrix[executorLocation]; exists {
			return score
		}
	}
	
	// Default to very low gravity if unknown combination
	return 0.1
}

// CalculatePlacementScore calculates the final placement score incorporating data gravity
func (dgm *DataGravityModel) CalculatePlacementScore(
	computeScore float64, 
	dataLocation, 
	executorLocation DataLocation, 
	dataGravityFactor float64,
) float64 {
	gravityScore := dgm.CalculateDataGravityScore(dataLocation, executorLocation)
	
	// Apply gravity factor as an exponent to control influence
	gravityWeight := math.Pow(gravityScore, dataGravityFactor)
	
	// Combine compute score with gravity-weighted factor
	return computeScore * gravityWeight
}

// EstimateTransferPenalty calculates penalty for data transfer between locations
func (dgm *DataGravityModel) EstimateTransferPenalty(
	fromLocation, 
	toLocation DataLocation, 
	dataSizeGB float64,
) float64 {
	if fromLocation == toLocation {
		return 0.0 // No transfer needed
	}
	
	gravityScore := dgm.CalculateDataGravityScore(fromLocation, toLocation)
	
	// Lower gravity scores result in higher transfer penalties
	// Penalty increases with data size and decreases with locality
	basePenalty := dataSizeGB * (1.0 - gravityScore)
	
	// Apply logarithmic scaling to prevent excessive penalties for large datasets
	scaledPenalty := basePenalty * math.Log10(1.0+dataSizeGB)
	
	return scaledPenalty
}

// GetLocationAffinity returns the affinity matrix for a given location
func (dgm *DataGravityModel) GetLocationAffinity(location DataLocation) map[DataLocation]float64 {
	if affinity, exists := dgm.gravityMatrix[location]; exists {
		// Return a copy to prevent external modification
		result := make(map[DataLocation]float64)
		for k, v := range affinity {
			result[k] = v
		}
		return result
	}
	
	return make(map[DataLocation]float64)
}

// RankLocationsByAffinity returns locations ranked by their affinity to the given data location
func (dgm *DataGravityModel) RankLocationsByAffinity(dataLocation DataLocation) []LocationAffinityRank {
	affinity := dgm.GetLocationAffinity(dataLocation)
	
	var ranks []LocationAffinityRank
	for location, score := range affinity {
		if location != dataLocation { // Exclude self
			ranks = append(ranks, LocationAffinityRank{
				Location: location,
				Score:    score,
			})
		}
	}
	
	// Sort by score (descending)
	for i := 0; i < len(ranks)-1; i++ {
		for j := i + 1; j < len(ranks); j++ {
			if ranks[i].Score < ranks[j].Score {
				ranks[i], ranks[j] = ranks[j], ranks[i]
			}
		}
	}
	
	return ranks
}

// LocationAffinityRank represents a location with its affinity score
type LocationAffinityRank struct {
	Location DataLocation `json:"location"`
	Score    float64      `json:"score"`
}

// UpdateGravityScore allows customization of gravity scores between specific locations
func (dgm *DataGravityModel) UpdateGravityScore(from, to DataLocation, score float64) {
	if dgm.gravityMatrix[from] == nil {
		dgm.gravityMatrix[from] = make(map[DataLocation]float64)
	}
	dgm.gravityMatrix[from][to] = score
}

// GetOptimalExecutorLocation returns the best executor location for given data location
func (dgm *DataGravityModel) GetOptimalExecutorLocation(
	dataLocation DataLocation, 
	availableLocations []DataLocation,
) DataLocation {
	bestLocation := DataLocationUnknown
	bestScore := -1.0
	
	for _, location := range availableLocations {
		score := dgm.CalculateDataGravityScore(dataLocation, location)
		if score > bestScore {
			bestScore = score
			bestLocation = location
		}
	}
	
	return bestLocation
}

// CalculateDataMovementCost calculates comprehensive cost of moving data
func (dgm *DataGravityModel) CalculateDataMovementCost(
	fromLocation, 
	toLocation DataLocation,
	dataSizeGB float64,
	transferCosts TransferCostMatrix,
) DataMovementCost {
	
	if fromLocation == toLocation {
		return DataMovementCost{
			TransferCost:     0.0,
			TransferTime:     0.0,
			GravityPenalty:   0.0,
			TotalCost:        0.0,
		}
	}
	
	// Get base transfer cost
	transferCost := transferCosts.GetTransferCost(fromLocation, toLocation, dataSizeGB)
	
	// Estimate transfer time (in seconds)
	transferTime := EstimateTransferTime(fromLocation, toLocation, dataSizeGB)
	
	// Calculate gravity penalty
	gravityPenalty := dgm.EstimateTransferPenalty(fromLocation, toLocation, dataSizeGB)
	
	// Total cost combines monetary cost and gravity-based penalty
	totalCost := transferCost + gravityPenalty
	
	return DataMovementCost{
		TransferCost:     transferCost,
		TransferTime:     transferTime.Seconds(),
		GravityPenalty:   gravityPenalty,
		TotalCost:        totalCost,
	}
}

// DataMovementCost represents the comprehensive cost of moving data
type DataMovementCost struct {
	TransferCost   float64 `json:"transfer_cost"`   // Monetary cost ($)
	TransferTime   float64 `json:"transfer_time"`   // Time cost (seconds)
	GravityPenalty float64 `json:"gravity_penalty"` // Locality penalty
	TotalCost      float64 `json:"total_cost"`      // Combined cost metric
}

// AnalyzeDataDistribution analyzes the distribution of data across locations
func (dgm *DataGravityModel) AnalyzeDataDistribution(dataDistribution map[DataLocation]float64) DataDistributionAnalysis {
	analysis := DataDistributionAnalysis{
		TotalDataGB:     0.0,
		LocationBreakdown: make(map[DataLocation]float64),
		GravityCenter:   DataLocationUnknown,
		Fragmentation:   0.0,
	}
	
	// Calculate total data and breakdown
	for location, sizeGB := range dataDistribution {
		analysis.TotalDataGB += sizeGB
		analysis.LocationBreakdown[location] = sizeGB
	}
	
	// Find gravity center (location with most data)
	maxData := 0.0
	for location, sizeGB := range dataDistribution {
		if sizeGB > maxData {
			maxData = sizeGB
			analysis.GravityCenter = location
		}
	}
	
	// Calculate fragmentation (how spread out the data is)
	if analysis.TotalDataGB > 0 {
		maxRatio := maxData / analysis.TotalDataGB
		analysis.Fragmentation = 1.0 - maxRatio // 0 = all data in one place, 1 = evenly distributed
	}
	
	return analysis
}

// DataDistributionAnalysis provides insights into data distribution
type DataDistributionAnalysis struct {
	TotalDataGB       float64                       `json:"total_data_gb"`
	LocationBreakdown map[DataLocation]float64      `json:"location_breakdown"`
	GravityCenter     DataLocation                  `json:"gravity_center"`     // Location with most data
	Fragmentation     float64                       `json:"fragmentation"`      // 0-1, higher means more fragmented
}

// RecommendOptimalPlacement recommends best placement considering data gravity
func (dgm *DataGravityModel) RecommendOptimalPlacement(
	dataDistribution map[DataLocation]float64,
	availableExecutors []DataLocation,
	dataGravityFactor float64,
) PlacementRecommendation {
	
	analysis := dgm.AnalyzeDataDistribution(dataDistribution)
	
	// Score each available executor location
	var scoredPlacements []ScoredPlacement
	
	for _, executorLocation := range availableExecutors {
		totalScore := 0.0
		totalMovementCost := 0.0
		
		// Calculate score based on data gravity to all data locations
		for dataLocation, dataSizeGB := range dataDistribution {
			if dataSizeGB > 0 {
				gravityScore := dgm.CalculateDataGravityScore(dataLocation, executorLocation)
				weightedScore := gravityScore * (dataSizeGB / analysis.TotalDataGB)
				totalScore += weightedScore
				
				// Add movement cost if data needs to be transferred
				if dataLocation != executorLocation {
					penalty := dgm.EstimateTransferPenalty(dataLocation, executorLocation, dataSizeGB)
					totalMovementCost += penalty
				}
			}
		}
		
		// Apply data gravity factor
		finalScore := math.Pow(totalScore, dataGravityFactor)
		
		scoredPlacements = append(scoredPlacements, ScoredPlacement{
			Location:     executorLocation,
			GravityScore: finalScore,
			MovementCost: totalMovementCost,
		})
	}
	
	// Sort by gravity score (descending) 
	for i := 0; i < len(scoredPlacements)-1; i++ {
		for j := i + 1; j < len(scoredPlacements); j++ {
			if scoredPlacements[i].GravityScore < scoredPlacements[j].GravityScore {
				scoredPlacements[i], scoredPlacements[j] = scoredPlacements[j], scoredPlacements[i]
			}
		}
	}
	
	var recommendation DataLocation = DataLocationUnknown
	if len(scoredPlacements) > 0 {
		recommendation = scoredPlacements[0].Location
	}
	
	return PlacementRecommendation{
		RecommendedLocation: recommendation,
		ScoredPlacements:    scoredPlacements,
		DataAnalysis:        analysis,
	}
}

// ScoredPlacement represents an executor location with its gravity score
type ScoredPlacement struct {
	Location     DataLocation `json:"location"`
	GravityScore float64      `json:"gravity_score"`
	MovementCost float64      `json:"movement_cost"`
}

// PlacementRecommendation provides a complete placement recommendation
type PlacementRecommendation struct {
	RecommendedLocation DataLocation               `json:"recommended_location"`
	ScoredPlacements    []ScoredPlacement          `json:"scored_placements"`
	DataAnalysis        DataDistributionAnalysis  `json:"data_analysis"`
}