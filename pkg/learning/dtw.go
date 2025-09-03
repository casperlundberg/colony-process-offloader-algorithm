package learning

import (
	"fmt"
	"math"
	"time"
)

// DTW implements Dynamic Time Warping for pattern matching
// Referenced in FURTHER_CONTROL_THEORY.md as [5] for pattern discovery
type DTW struct {
	// Pattern storage
	patterns       []Pattern      // Stored patterns for matching
	maxPatterns    int           // Maximum patterns to store
	minPatternLen  int           // Minimum pattern length
	maxPatternLen  int           // Maximum pattern length
	
	// Distance parameters
	windowSize     int           // Sakoe-Chiba band constraint
	distanceMetric DistanceMetric // Distance metric to use
	
	// Pattern matching
	threshold      float64       // Similarity threshold
	minConfidence  float64       // Minimum confidence for pattern match
	
	// Statistics
	matchCount     int           // Number of patterns matched
	totalQueries   int           // Total pattern queries
	avgDistance    float64       // Average DTW distance
	lastMatch      time.Time     // Last pattern match time
}

// Pattern represents a time series pattern
type Pattern struct {
	ID          string    `json:"id"`           // Pattern identifier
	Data        []float64 `json:"data"`         // Pattern data points
	Length      int       `json:"length"`       // Pattern length
	Label       string    `json:"label"`        // Pattern label/category
	Frequency   int       `json:"frequency"`    // How often pattern appears
	Confidence  float64   `json:"confidence"`   // Pattern confidence score
	CreatedAt   time.Time `json:"created_at"`   // When pattern was created
	LastSeen    time.Time `json:"last_seen"`    // When pattern was last seen
	Context     string    `json:"context"`      // Context where pattern occurs
}

// DistanceMetric represents different distance metrics
type DistanceMetric int

const (
	DistanceEuclidean DistanceMetric = iota
	DistanceManhattan
	DistanceCosine
	DistancePearson
)

// NewDTW creates a new DTW pattern matcher
func NewDTW() *DTW {
	return &DTW{
		patterns:       make([]Pattern, 0),
		maxPatterns:    100,
		minPatternLen:  3,
		maxPatternLen:  50,
		windowSize:     -1, // No constraint by default
		distanceMetric: DistanceEuclidean,
		threshold:      0.8,
		minConfidence:  0.6,
		matchCount:     0,
		totalQueries:   0,
		avgDistance:    0.0,
	}
}

// NewDTWWithConstraints creates DTW with Sakoe-Chiba band constraint
func NewDTWWithConstraints(windowSize int) *DTW {
	dtw := NewDTW()
	dtw.windowSize = windowSize
	return dtw
}

// Distance calculates DTW distance between two time series
func (dtw *DTW) Distance(series1, series2 []float64) (float64, error) {
	if len(series1) == 0 || len(series2) == 0 {
		return math.Inf(1), fmt.Errorf("empty time series")
	}
	
	m, n := len(series1), len(series2)
	
	// Initialize DTW matrix with infinity
	dtwMatrix := make([][]float64, m+1)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, n+1)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.Inf(1)
		}
	}
	
	// Base case
	dtwMatrix[0][0] = 0
	
	// Fill DTW matrix
	for i := 1; i <= m; i++ {
		// Apply Sakoe-Chiba band constraint if specified
		jStart := 1
		jEnd := n
		
		if dtw.windowSize > 0 {
			jStart = int(math.Max(1, float64(i-dtw.windowSize)))
			jEnd = int(math.Min(float64(n), float64(i+dtw.windowSize)))
		}
		
		for j := jStart; j <= jEnd; j++ {
			// Calculate point-wise distance
			cost := dtw.pointDistance(series1[i-1], series2[j-1])
			
			// DTW recurrence relation
			dtwMatrix[i][j] = cost + math.Min(
				dtwMatrix[i-1][j],     // Insertion
				math.Min(
					dtwMatrix[i][j-1],     // Deletion
					dtwMatrix[i-1][j-1],   // Match
				),
			)
		}
	}
	
	// Return normalized DTW distance
	distance := dtwMatrix[m][n]
	normalizedDistance := distance / float64(m+n) // Normalize by path length
	
	return normalizedDistance, nil
}

// pointDistance calculates distance between two points
func (dtw *DTW) pointDistance(x, y float64) float64 {
	switch dtw.distanceMetric {
	case DistanceEuclidean:
		return (x - y) * (x - y)
	case DistanceManhattan:
		return math.Abs(x - y)
	case DistanceCosine:
		// For single points, cosine distance is not meaningful
		// Fall back to Euclidean
		return (x - y) * (x - y)
	case DistancePearson:
		// For single points, Pearson correlation is not meaningful
		// Fall back to Euclidean
		return (x - y) * (x - y)
	default:
		return (x - y) * (x - y)
	}
}

// FindBestMatch finds the best matching pattern for a query series
func (dtw *DTW) FindBestMatch(query []float64) (PatternMatch, error) {
	if len(query) < dtw.minPatternLen {
		return PatternMatch{}, fmt.Errorf("query too short: minimum length %d", dtw.minPatternLen)
	}
	
	dtw.totalQueries++
	
	if len(dtw.patterns) == 0 {
		return PatternMatch{}, fmt.Errorf("no patterns stored")
	}
	
	bestMatch := PatternMatch{
		Distance:   math.Inf(1),
		Similarity: 0.0,
		Confidence: 0.0,
	}
	
	// Compare query against all stored patterns
	for i, pattern := range dtw.patterns {
		distance, err := dtw.Distance(query, pattern.Data)
		if err != nil {
			continue
		}
		
		// Convert distance to similarity (0-1, higher is better)
		similarity := 1.0 / (1.0 + distance)
		
		// Calculate confidence based on pattern frequency and similarity
		confidence := similarity * (float64(pattern.Frequency) / 10.0)
		if confidence > 1.0 {
			confidence = 1.0
		}
		
		if distance < bestMatch.Distance && similarity >= dtw.threshold && confidence >= dtw.minConfidence {
			bestMatch = PatternMatch{
				PatternIndex: i,
				PatternID:    pattern.ID,
				PatternLabel: pattern.Label,
				Distance:     distance,
				Similarity:   similarity,
				Confidence:   confidence,
				QueryLength:  len(query),
				PatternLength: pattern.Length,
				MatchedAt:    time.Now(),
			}
		}
	}
	
	// Update statistics
	if bestMatch.Distance != math.Inf(1) {
		dtw.matchCount++
		dtw.lastMatch = time.Now()
		dtw.updateAvgDistance(bestMatch.Distance)
		
		// Update pattern frequency
		dtw.patterns[bestMatch.PatternIndex].Frequency++
		dtw.patterns[bestMatch.PatternIndex].LastSeen = time.Now()
	}
	
	return bestMatch, nil
}

// AddPattern adds a new pattern to the pattern library
func (dtw *DTW) AddPattern(data []float64, label, context string) (string, error) {
	if len(data) < dtw.minPatternLen || len(data) > dtw.maxPatternLen {
		return "", fmt.Errorf("pattern length %d outside valid range [%d, %d]", 
			len(data), dtw.minPatternLen, dtw.maxPatternLen)
	}
	
	// Generate pattern ID
	patternID := fmt.Sprintf("pattern_%d_%d", len(dtw.patterns), time.Now().Unix())
	
	// Create new pattern
	pattern := Pattern{
		ID:         patternID,
		Data:       make([]float64, len(data)),
		Length:     len(data),
		Label:      label,
		Frequency:  1,
		Confidence: 1.0,
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
		Context:    context,
	}
	
	copy(pattern.Data, data)
	
	// Check if we need to remove old patterns
	if len(dtw.patterns) >= dtw.maxPatterns {
		dtw.removeOldestPattern()
	}
	
	// Add new pattern
	dtw.patterns = append(dtw.patterns, pattern)
	
	return patternID, nil
}

// removeOldestPattern removes the least frequently used pattern
func (dtw *DTW) removeOldestPattern() {
	if len(dtw.patterns) == 0 {
		return
	}
	
	// Find pattern with lowest frequency (LFU eviction)
	minFreq := dtw.patterns[0].Frequency
	minIndex := 0
	
	for i, pattern := range dtw.patterns {
		if pattern.Frequency < minFreq {
			minFreq = pattern.Frequency
			minIndex = i
		}
	}
	
	// Remove the pattern
	dtw.patterns = append(dtw.patterns[:minIndex], dtw.patterns[minIndex+1:]...)
}

// DiscoverPatterns automatically discovers patterns in a time series
func (dtw *DTW) DiscoverPatterns(timeSeries []float64, minPatternLen, maxPatternLen int) ([]Pattern, error) {
	if len(timeSeries) < minPatternLen*2 {
		return nil, fmt.Errorf("time series too short for pattern discovery")
	}
	
	discoveredPatterns := make([]Pattern, 0)
	patternCandidates := make(map[string]PatternCandidate)
	
	// Sliding window approach to extract potential patterns
	for length := minPatternLen; length <= maxPatternLen && length <= len(timeSeries)/2; length++ {
		for start := 0; start <= len(timeSeries)-length; start++ {
			candidate := timeSeries[start : start+length]
			
			// Create a signature for this pattern
			signature := dtw.createPatternSignature(candidate)
			
			if existing, exists := patternCandidates[signature]; exists {
				// Pattern already seen, increment count
				existing.Count++
				existing.Positions = append(existing.Positions, start)
				patternCandidates[signature] = existing
			} else {
				// New pattern candidate
				patternCandidates[signature] = PatternCandidate{
					Data:      make([]float64, len(candidate)),
					Length:    length,
					Count:     1,
					Positions: []int{start},
					Signature: signature,
				}
				copy(patternCandidates[signature].Data, candidate)
			}
		}
	}
	
	// Filter candidates based on frequency and convert to patterns
	minOccurrences := 2 // Minimum occurrences to be considered a pattern
	
	for _, candidate := range patternCandidates {
		if candidate.Count >= minOccurrences {
			patternID := fmt.Sprintf("discovered_%s_%d", candidate.Signature[:8], len(discoveredPatterns))
			
			pattern := Pattern{
				ID:         patternID,
				Data:       candidate.Data,
				Length:     candidate.Length,
				Label:      fmt.Sprintf("pattern_len_%d", candidate.Length),
				Frequency:  candidate.Count,
				Confidence: float64(candidate.Count) / 10.0, // Simple confidence metric
				CreatedAt:  time.Now(),
				LastSeen:   time.Now(),
				Context:    "auto_discovered",
			}
			
			// Limit confidence to [0,1]
			if pattern.Confidence > 1.0 {
				pattern.Confidence = 1.0
			}
			
			discoveredPatterns = append(discoveredPatterns, pattern)
		}
	}
	
	return discoveredPatterns, nil
}

// createPatternSignature creates a signature for pattern matching
func (dtw *DTW) createPatternSignature(data []float64) string {
	if len(data) == 0 {
		return ""
	}
	
	// Simple signature based on normalized patterns
	normalized := dtw.normalizePattern(data)
	
	signature := ""
	for i, val := range normalized {
		// Quantize to reduce signature space
		quantized := int(val * 10) // 10 levels
		if i > 0 {
			signature += "_"
		}
		signature += fmt.Sprintf("%d", quantized)
		
		// Limit signature length
		if len(signature) > 50 {
			break
		}
	}
	
	return signature
}

// normalizePattern normalizes a pattern to [0,1] range
func (dtw *DTW) normalizePattern(data []float64) []float64 {
	if len(data) == 0 {
		return data
	}
	
	// Find min and max
	min, max := data[0], data[0]
	for _, val := range data {
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}
	
	// Normalize
	normalized := make([]float64, len(data))
	if max-min == 0 {
		// All values are the same
		for i := range normalized {
			normalized[i] = 0.5
		}
	} else {
		for i, val := range data {
			normalized[i] = (val - min) / (max - min)
		}
	}
	
	return normalized
}

// updateAvgDistance updates running average of DTW distances
func (dtw *DTW) updateAvgDistance(distance float64) {
	if dtw.matchCount == 1 {
		dtw.avgDistance = distance
	} else {
		// Exponential moving average
		alpha := 0.1
		dtw.avgDistance = alpha*distance + (1-alpha)*dtw.avgDistance
	}
}

// GetPatterns returns all stored patterns
func (dtw *DTW) GetPatterns() []Pattern {
	result := make([]Pattern, len(dtw.patterns))
	copy(result, dtw.patterns)
	return result
}

// GetStats returns DTW statistics
func (dtw *DTW) GetStats() DTWStats {
	matchRate := 0.0
	if dtw.totalQueries > 0 {
		matchRate = float64(dtw.matchCount) / float64(dtw.totalQueries)
	}
	
	return DTWStats{
		PatternsStored:  len(dtw.patterns),
		MaxPatterns:     dtw.maxPatterns,
		TotalQueries:    dtw.totalQueries,
		SuccessfulMatches: dtw.matchCount,
		MatchRate:       matchRate,
		AvgDistance:     dtw.avgDistance,
		Threshold:       dtw.threshold,
		MinConfidence:   dtw.minConfidence,
		WindowSize:      dtw.windowSize,
		DistanceMetric:  dtw.distanceMetric.String(),
		LastMatch:       dtw.lastMatch,
	}
}

// Reset resets DTW to initial state
func (dtw *DTW) Reset() {
	dtw.patterns = make([]Pattern, 0)
	dtw.matchCount = 0
	dtw.totalQueries = 0
	dtw.avgDistance = 0.0
	dtw.lastMatch = time.Time{}
}

// SetThreshold sets the similarity threshold
func (dtw *DTW) SetThreshold(threshold float64) {
	if threshold >= 0 && threshold <= 1 {
		dtw.threshold = threshold
	}
}

// SetDistanceMetric sets the distance metric
func (dtw *DTW) SetDistanceMetric(metric DistanceMetric) {
	dtw.distanceMetric = metric
}

// String returns string representation of distance metric
func (dm DistanceMetric) String() string {
	switch dm {
	case DistanceManhattan:
		return "manhattan"
	case DistanceCosine:
		return "cosine"
	case DistancePearson:
		return "pearson"
	default:
		return "euclidean"
	}
}

// PatternMatch represents the result of pattern matching
type PatternMatch struct {
	PatternIndex  int       `json:"pattern_index"`   // Index of matched pattern
	PatternID     string    `json:"pattern_id"`      // ID of matched pattern
	PatternLabel  string    `json:"pattern_label"`   // Label of matched pattern
	Distance      float64   `json:"distance"`        // DTW distance
	Similarity    float64   `json:"similarity"`      // Similarity score (0-1)
	Confidence    float64   `json:"confidence"`      // Confidence score (0-1)
	QueryLength   int       `json:"query_length"`    // Length of query series
	PatternLength int       `json:"pattern_length"`  // Length of matched pattern
	MatchedAt     time.Time `json:"matched_at"`      // When match was found
}

// PatternCandidate represents a potential pattern during discovery
type PatternCandidate struct {
	Data      []float64 `json:"data"`      // Pattern data
	Length    int       `json:"length"`    // Pattern length
	Count     int       `json:"count"`     // Number of occurrences
	Positions []int     `json:"positions"` // Positions where pattern occurs
	Signature string    `json:"signature"` // Pattern signature
}

// DTWStats represents statistics about DTW pattern matching
type DTWStats struct {
	PatternsStored    int           `json:"patterns_stored"`     // Number of patterns stored
	MaxPatterns       int           `json:"max_patterns"`        // Maximum patterns allowed
	TotalQueries      int           `json:"total_queries"`       // Total pattern queries
	SuccessfulMatches int           `json:"successful_matches"`  // Successful matches
	MatchRate         float64       `json:"match_rate"`          // Success rate (0-1)
	AvgDistance       float64       `json:"avg_distance"`        // Average DTW distance
	Threshold         float64       `json:"threshold"`           // Similarity threshold
	MinConfidence     float64       `json:"min_confidence"`      // Minimum confidence
	WindowSize        int           `json:"window_size"`         // Sakoe-Chiba window size
	DistanceMetric    string        `json:"distance_metric"`     // Distance metric used
	LastMatch         time.Time     `json:"last_match"`          // Last successful match
}