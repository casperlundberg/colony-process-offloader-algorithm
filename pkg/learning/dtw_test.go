package learning

import (
	"math"
	"strings"
	"testing"
)

func TestDTW_NewDTW(t *testing.T) {
	dtw := NewDTW()
	if dtw == nil {
		t.Fatal("NewDTW() returned nil")
	}

	if len(dtw.patterns) != 0 {
		t.Errorf("Expected 0 initial patterns, got %d", len(dtw.patterns))
	}

	if dtw.maxPatterns != 100 {
		t.Errorf("Expected maxPatterns=100, got %d", dtw.maxPatterns)
	}

	if dtw.minPatternLen != 3 {
		t.Errorf("Expected minPatternLen=3, got %d", dtw.minPatternLen)
	}

	if dtw.maxPatternLen != 50 {
		t.Errorf("Expected maxPatternLen=50, got %d", dtw.maxPatternLen)
	}

	if dtw.windowSize != -1 {
		t.Errorf("Expected windowSize=-1, got %d", dtw.windowSize)
	}

	if dtw.distanceMetric != DistanceEuclidean {
		t.Errorf("Expected DistanceEuclidean, got %v", dtw.distanceMetric)
	}

	if dtw.threshold != 0.8 {
		t.Errorf("Expected threshold=0.8, got %f", dtw.threshold)
	}

	if dtw.matchCount != 0 {
		t.Errorf("Expected matchCount=0, got %d", dtw.matchCount)
	}

	if dtw.totalQueries != 0 {
		t.Errorf("Expected totalQueries=0, got %d", dtw.totalQueries)
	}
}

func TestDTW_NewDTWWithConstraints(t *testing.T) {
	windowSize := 10
	dtw := NewDTWWithConstraints(windowSize)

	if dtw == nil {
		t.Fatal("NewDTWWithConstraints() returned nil")
	}

	if dtw.windowSize != windowSize {
		t.Errorf("Expected windowSize=%d, got %d", windowSize, dtw.windowSize)
	}
}

func TestDTW_Distance(t *testing.T) {
	dtw := NewDTW()

	// Test identical series
	series1 := []float64{1.0, 2.0, 3.0, 4.0}
	series2 := []float64{1.0, 2.0, 3.0, 4.0}

	distance, err := dtw.Distance(series1, series2)
	if err != nil {
		t.Fatalf("Distance() failed: %v", err)
	}

	if distance != 0.0 {
		t.Errorf("Expected distance=0.0 for identical series, got %f", distance)
	}

	// Test different series
	series3 := []float64{2.0, 3.0, 4.0, 5.0}
	distance, err = dtw.Distance(series1, series3)
	if err != nil {
		t.Fatalf("Distance() failed: %v", err)
	}

	if distance <= 0.0 {
		t.Errorf("Expected positive distance for different series, got %f", distance)
	}

	// Test with different lengths
	series4 := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
	distance, err = dtw.Distance(series1, series4)
	if err != nil {
		t.Fatalf("Distance() failed for different lengths: %v", err)
	}

	if distance <= 0.0 {
		t.Errorf("Expected positive distance for different length series, got %f", distance)
	}
}

func TestDTW_DistanceEmptySeries(t *testing.T) {
	dtw := NewDTW()

	empty := []float64{}
	nonEmpty := []float64{1.0, 2.0, 3.0}

	_, err := dtw.Distance(empty, nonEmpty)
	if err == nil {
		t.Error("Expected error for empty series")
	}

	_, err = dtw.Distance(nonEmpty, empty)
	if err == nil {
		t.Error("Expected error for empty series")
	}

	_, err = dtw.Distance(empty, empty)
	if err == nil {
		t.Error("Expected error for empty series")
	}
}

func TestDTW_DistanceWithConstraints(t *testing.T) {
	dtw := NewDTWWithConstraints(2) // Small window

	series1 := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	series2 := []float64{1.5, 2.5, 3.5, 4.5, 5.5}

	distance, err := dtw.Distance(series1, series2)
	if err != nil {
		t.Fatalf("Distance() with constraints failed: %v", err)
	}

	if distance <= 0.0 {
		t.Errorf("Expected positive distance with constraints, got %f", distance)
	}
}

func TestDTW_AddPattern(t *testing.T) {
	dtw := NewDTW()

	data := []float64{1.0, 2.0, 3.0, 2.0, 1.0}
	label := "test_pattern"
	context := "unit_test"

	patternID, err := dtw.AddPattern(data, label, context)
	if err != nil {
		t.Fatalf("AddPattern() failed: %v", err)
	}

	if patternID == "" {
		t.Error("Expected non-empty pattern ID")
	}

	if len(dtw.patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(dtw.patterns))
	}

	pattern := dtw.patterns[0]
	if pattern.ID != patternID {
		t.Errorf("Expected pattern ID=%s, got %s", patternID, pattern.ID)
	}

	if pattern.Label != label {
		t.Errorf("Expected label=%s, got %s", label, pattern.Label)
	}

	if pattern.Context != context {
		t.Errorf("Expected context=%s, got %s", context, pattern.Context)
	}

	if pattern.Length != len(data) {
		t.Errorf("Expected length=%d, got %d", len(data), pattern.Length)
	}

	if pattern.Frequency != 1 {
		t.Errorf("Expected frequency=1, got %d", pattern.Frequency)
	}

	// Check data is copied correctly
	for i, expected := range data {
		if pattern.Data[i] != expected {
			t.Errorf("Data[%d]: expected %f, got %f", i, expected, pattern.Data[i])
		}
	}
}

func TestDTW_AddPatternInvalidLength(t *testing.T) {
	dtw := NewDTW()

	// Too short
	shortData := []float64{1.0, 2.0} // Length 2, min is 3
	_, err := dtw.AddPattern(shortData, "short", "test")
	if err == nil {
		t.Error("Expected error for too short pattern")
	}

	// Too long
	longData := make([]float64, 100) // Length 100, max is 50
	_, err = dtw.AddPattern(longData, "long", "test")
	if err == nil {
		t.Error("Expected error for too long pattern")
	}
}

func TestDTW_AddPatternMaxCapacity(t *testing.T) {
	dtw := NewDTW()
	dtw.maxPatterns = 3 // Small limit for testing

	data := []float64{1.0, 2.0, 3.0}

	// Add patterns up to capacity
	for i := 0; i < 3; i++ {
		_, err := dtw.AddPattern(data, "pattern", "test")
		if err != nil {
			t.Fatalf("AddPattern %d failed: %v", i, err)
		}
	}

	if len(dtw.patterns) != 3 {
		t.Errorf("Expected 3 patterns, got %d", len(dtw.patterns))
	}

	// Add one more (should evict oldest)
	_, err := dtw.AddPattern(data, "new_pattern", "test")
	if err != nil {
		t.Fatalf("AddPattern beyond capacity failed: %v", err)
	}

	if len(dtw.patterns) != 3 {
		t.Errorf("Expected 3 patterns after eviction, got %d", len(dtw.patterns))
	}
}

func TestDTW_FindBestMatch(t *testing.T) {
	dtw := NewDTW()
	dtw.threshold = 0.1 // Very low threshold for testing
	dtw.minConfidence = 0.1 // Very low confidence for testing

	// Add a pattern
	patternData := []float64{1.0, 2.0, 3.0, 2.0, 1.0}
	patternID, err := dtw.AddPattern(patternData, "test_pattern", "test")
	if err != nil {
		t.Fatalf("AddPattern() failed: %v", err)
	}

	// Query with similar pattern
	query := []float64{1.1, 2.1, 3.1, 2.1, 1.1}

	match, err := dtw.FindBestMatch(query)
	if err != nil {
		t.Fatalf("FindBestMatch() failed: %v", err)
	}

	if match.PatternID != patternID {
		t.Errorf("Expected pattern ID=%s, got %s", patternID, match.PatternID)
	}

	if match.Distance <= 0.0 {
		t.Errorf("Expected positive distance, got %f", match.Distance)
	}

	if match.Similarity <= 0.0 || match.Similarity > 1.0 {
		t.Errorf("Expected similarity in (0,1], got %f", match.Similarity)
	}

	if match.Confidence <= 0.0 || match.Confidence > 1.0 {
		t.Errorf("Expected confidence in (0,1], got %f", match.Confidence)
	}

	if match.QueryLength != len(query) {
		t.Errorf("Expected query length=%d, got %d", len(query), match.QueryLength)
	}

	if match.PatternLength != len(patternData) {
		t.Errorf("Expected pattern length=%d, got %d", len(patternData), match.PatternLength)
	}

	// Check that statistics are updated
	if dtw.totalQueries != 1 {
		t.Errorf("Expected totalQueries=1, got %d", dtw.totalQueries)
	}

	if dtw.matchCount != 1 {
		t.Errorf("Expected matchCount=1, got %d", dtw.matchCount)
	}

	// Check that pattern frequency is updated
	if dtw.patterns[0].Frequency != 2 { // Initial 1 + 1 from match
		t.Errorf("Expected pattern frequency=2, got %d", dtw.patterns[0].Frequency)
	}
}

func TestDTW_FindBestMatchNoPatterns(t *testing.T) {
	dtw := NewDTW()

	query := []float64{1.0, 2.0, 3.0}
	_, err := dtw.FindBestMatch(query)
	if err == nil {
		t.Error("Expected error when no patterns stored")
	}
}

func TestDTW_FindBestMatchTooShort(t *testing.T) {
	dtw := NewDTW()

	query := []float64{1.0, 2.0} // Too short (< minPatternLen)
	_, err := dtw.FindBestMatch(query)
	if err == nil {
		t.Error("Expected error for too short query")
	}
}

func TestDTW_DiscoverPatterns(t *testing.T) {
	dtw := NewDTW()

	// Create time series with repeating pattern
	timeSeries := []float64{
		1.0, 2.0, 3.0, 2.0, 1.0, // Pattern 1
		0.5, 1.5, 2.5, 1.5, 0.5, // Similar pattern
		1.0, 2.0, 3.0, 2.0, 1.0, // Pattern 1 again
		4.0, 5.0, 6.0, 5.0, 4.0, // Different pattern
		1.1, 2.1, 3.1, 2.1, 1.1, // Similar to pattern 1
	}

	minLen := 3
	maxLen := 5
	patterns, err := dtw.DiscoverPatterns(timeSeries, minLen, maxLen)
	if err != nil {
		t.Fatalf("DiscoverPatterns() failed: %v", err)
	}

	if len(patterns) == 0 {
		t.Error("Expected at least one discovered pattern")
	}

	// Check pattern properties
	for i, pattern := range patterns {
		if pattern.ID == "" {
			t.Errorf("Pattern %d: expected non-empty ID", i)
		}

		if pattern.Length < minLen || pattern.Length > maxLen {
			t.Errorf("Pattern %d: length %d outside range [%d, %d]", 
				i, pattern.Length, minLen, maxLen)
		}

		if pattern.Frequency < 2 {
			t.Errorf("Pattern %d: expected frequency >= 2, got %d", i, pattern.Frequency)
		}

		if pattern.Context != "auto_discovered" {
			t.Errorf("Pattern %d: expected context=auto_discovered, got %s", 
				i, pattern.Context)
		}

		if len(pattern.Data) != pattern.Length {
			t.Errorf("Pattern %d: data length mismatch", i)
		}
	}
}

func TestDTW_DiscoverPatternsTooShort(t *testing.T) {
	dtw := NewDTW()

	shortSeries := []float64{1.0, 2.0, 3.0} // Too short for pattern discovery
	_, err := dtw.DiscoverPatterns(shortSeries, 3, 5)
	if err == nil {
		t.Error("Expected error for too short time series")
	}
}

func TestDTW_GetPatterns(t *testing.T) {
	dtw := NewDTW()

	// Initially should be empty
	patterns := dtw.GetPatterns()
	if len(patterns) != 0 {
		t.Errorf("Expected 0 initial patterns, got %d", len(patterns))
	}

	// Add some patterns
	data1 := []float64{1.0, 2.0, 3.0}
	data2 := []float64{4.0, 5.0, 6.0}

	dtw.AddPattern(data1, "pattern1", "test")
	dtw.AddPattern(data2, "pattern2", "test")

	patterns = dtw.GetPatterns()
	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
	}

	// Should be a copy, not the same slice
	patterns[0].Label = "modified"
	if dtw.patterns[0].Label == "modified" {
		t.Error("GetPatterns() should return a copy")
	}
}

func TestDTW_GetStats(t *testing.T) {
	dtw := NewDTW()

	// Initial stats
	stats := dtw.GetStats()
	if stats.PatternsStored != 0 {
		t.Errorf("Expected patternsStored=0, got %d", stats.PatternsStored)
	}

	if stats.TotalQueries != 0 {
		t.Errorf("Expected totalQueries=0, got %d", stats.TotalQueries)
	}

	if stats.MatchRate != 0.0 {
		t.Errorf("Expected matchRate=0.0, got %f", stats.MatchRate)
	}

	// Add pattern and perform match
	data := []float64{1.0, 2.0, 3.0}
	dtw.AddPattern(data, "test", "test")

	query := []float64{1.1, 2.1, 3.1}
	dtw.threshold = 0.1 // Very low threshold to ensure match
	dtw.FindBestMatch(query)

	// Updated stats
	stats = dtw.GetStats()
	if stats.PatternsStored != 1 {
		t.Errorf("Expected patternsStored=1, got %d", stats.PatternsStored)
	}

	if stats.TotalQueries != 1 {
		t.Errorf("Expected totalQueries=1, got %d", stats.TotalQueries)
	}

	if stats.SuccessfulMatches != 1 {
		t.Errorf("Expected successfulMatches=1, got %d", stats.SuccessfulMatches)
	}

	if stats.MatchRate != 1.0 {
		t.Errorf("Expected matchRate=1.0, got %f", stats.MatchRate)
	}

	if stats.AvgDistance <= 0.0 {
		t.Errorf("Expected positive avgDistance, got %f", stats.AvgDistance)
	}

	if stats.Threshold != dtw.threshold {
		t.Errorf("Expected threshold=%f, got %f", dtw.threshold, stats.Threshold)
	}

	if stats.DistanceMetric != "euclidean" {
		t.Errorf("Expected distanceMetric=euclidean, got %s", stats.DistanceMetric)
	}
}

func TestDTW_Reset(t *testing.T) {
	dtw := NewDTW()

	// Add some data
	data := []float64{1.0, 2.0, 3.0}
	dtw.AddPattern(data, "test", "test")
	dtw.FindBestMatch(data)

	// Verify data exists
	if len(dtw.patterns) == 0 {
		t.Error("Expected patterns before reset")
	}
	if dtw.totalQueries == 0 {
		t.Error("Expected queries before reset")
	}
	if dtw.matchCount == 0 {
		t.Error("Expected matches before reset")
	}

	dtw.Reset()

	// Verify reset
	if len(dtw.patterns) != 0 {
		t.Errorf("Expected 0 patterns after reset, got %d", len(dtw.patterns))
	}

	if dtw.totalQueries != 0 {
		t.Errorf("Expected totalQueries=0 after reset, got %d", dtw.totalQueries)
	}

	if dtw.matchCount != 0 {
		t.Errorf("Expected matchCount=0 after reset, got %d", dtw.matchCount)
	}

	if dtw.avgDistance != 0.0 {
		t.Errorf("Expected avgDistance=0.0 after reset, got %f", dtw.avgDistance)
	}

	if !dtw.lastMatch.IsZero() {
		t.Error("Expected zero lastMatch after reset")
	}
}

func TestDTW_SetThreshold(t *testing.T) {
	dtw := NewDTW()

	// Valid threshold
	newThreshold := 0.5
	dtw.SetThreshold(newThreshold)
	if dtw.threshold != newThreshold {
		t.Errorf("Expected threshold=%f, got %f", newThreshold, dtw.threshold)
	}

	// Invalid thresholds
	oldThreshold := dtw.threshold
	dtw.SetThreshold(-0.1) // Too low
	if dtw.threshold != oldThreshold {
		t.Error("Threshold should not change for invalid value")
	}

	dtw.SetThreshold(1.5) // Too high
	if dtw.threshold != oldThreshold {
		t.Error("Threshold should not change for invalid value")
	}
}

func TestDTW_SetDistanceMetric(t *testing.T) {
	dtw := NewDTW()

	newMetric := DistanceManhattan
	dtw.SetDistanceMetric(newMetric)
	if dtw.distanceMetric != newMetric {
		t.Errorf("Expected distanceMetric=%v, got %v", newMetric, dtw.distanceMetric)
	}
}

func TestDTW_PointDistance(t *testing.T) {
	dtw := NewDTW()

	x, y := 3.0, 1.0

	// Test Euclidean distance
	dtw.distanceMetric = DistanceEuclidean
	distance := dtw.pointDistance(x, y)
	expected := (x - y) * (x - y) // 4.0
	if distance != expected {
		t.Errorf("Euclidean distance: expected %f, got %f", expected, distance)
	}

	// Test Manhattan distance
	dtw.distanceMetric = DistanceManhattan
	distance = dtw.pointDistance(x, y)
	expected = math.Abs(x - y) // 2.0
	if distance != expected {
		t.Errorf("Manhattan distance: expected %f, got %f", expected, distance)
	}

	// Test Cosine distance (should fall back to Euclidean for single points)
	dtw.distanceMetric = DistanceCosine
	distance = dtw.pointDistance(x, y)
	expected = (x - y) * (x - y) // 4.0
	if distance != expected {
		t.Errorf("Cosine distance fallback: expected %f, got %f", expected, distance)
	}

	// Test Pearson distance (should fall back to Euclidean for single points)
	dtw.distanceMetric = DistancePearson
	distance = dtw.pointDistance(x, y)
	expected = (x - y) * (x - y) // 4.0
	if distance != expected {
		t.Errorf("Pearson distance fallback: expected %f, got %f", expected, distance)
	}
}

func TestDTW_NormalizePattern(t *testing.T) {
	dtw := NewDTW()

	// Test normal case
	data := []float64{1.0, 3.0, 5.0, 2.0, 4.0}
	normalized := dtw.normalizePattern(data)

	// Should be in [0, 1] range
	for i, val := range normalized {
		if val < 0.0 || val > 1.0 {
			t.Errorf("Normalized[%d]=%f not in [0,1] range", i, val)
		}
	}

	// Min should be 0, max should be 1
	min, max := normalized[0], normalized[0]
	for _, val := range normalized {
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	if min != 0.0 {
		t.Errorf("Expected min=0.0, got %f", min)
	}

	if max != 1.0 {
		t.Errorf("Expected max=1.0, got %f", max)
	}

	// Test constant values
	constant := []float64{5.0, 5.0, 5.0}
	normalizedConstant := dtw.normalizePattern(constant)
	for i, val := range normalizedConstant {
		if val != 0.5 {
			t.Errorf("Constant normalized[%d]: expected 0.5, got %f", i, val)
		}
	}

	// Test empty data
	empty := []float64{}
	normalizedEmpty := dtw.normalizePattern(empty)
	if len(normalizedEmpty) != 0 {
		t.Errorf("Expected empty result for empty input, got %d elements", len(normalizedEmpty))
	}
}

func TestDTW_CreatePatternSignature(t *testing.T) {
	dtw := NewDTW()

	// Test normal case
	data := []float64{1.0, 2.0, 3.0, 2.0, 1.0}
	signature := dtw.createPatternSignature(data)

	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	if !strings.Contains(signature, "_") {
		t.Error("Expected underscore separators in signature")
	}

	// Same data should produce same signature
	signature2 := dtw.createPatternSignature(data)
	if signature != signature2 {
		t.Errorf("Same data should produce same signature: %s != %s", signature, signature2)
	}

	// Different data should produce different signature
	differentData := []float64{2.0, 3.0, 4.0, 3.0, 2.0}
	differentSignature := dtw.createPatternSignature(differentData)
	if signature == differentSignature {
		t.Error("Different data should produce different signatures")
	}

	// Empty data
	emptySignature := dtw.createPatternSignature([]float64{})
	if emptySignature != "" {
		t.Errorf("Expected empty signature for empty data, got %s", emptySignature)
	}
}

func TestDistanceMetric_String(t *testing.T) {
	testCases := []struct {
		metric   DistanceMetric
		expected string
	}{
		{DistanceEuclidean, "euclidean"},
		{DistanceManhattan, "manhattan"},
		{DistanceCosine, "cosine"},
		{DistancePearson, "pearson"},
	}

	for _, tc := range testCases {
		if tc.metric.String() != tc.expected {
			t.Errorf("Expected %v.String()=%s, got %s",
				tc.metric, tc.expected, tc.metric.String())
		}
	}
}

func TestDTW_RemoveOldestPattern(t *testing.T) {
	dtw := NewDTW()
	dtw.maxPatterns = 3

	// Add patterns with different frequencies
	data := []float64{1.0, 2.0, 3.0}
	dtw.AddPattern(data, "pattern1", "test") // Frequency: 1
	dtw.AddPattern(data, "pattern2", "test") // Frequency: 1
	dtw.AddPattern(data, "pattern3", "test") // Frequency: 1

	// Increase frequency of first and third patterns
	dtw.patterns[0].Frequency = 5
	dtw.patterns[2].Frequency = 3
	// patterns[1] remains at frequency 1 (lowest)

	// Add another pattern (should remove the one with lowest frequency)
	dtw.AddPattern(data, "pattern4", "test")

	if len(dtw.patterns) != 3 {
		t.Errorf("Expected 3 patterns after eviction, got %d", len(dtw.patterns))
	}

	// Check that pattern with lowest frequency was removed
	for _, pattern := range dtw.patterns {
		if pattern.Label == "pattern2" {
			t.Error("Pattern with lowest frequency should have been removed")
		}
	}
}

func TestDTW_UpdateAvgDistance(t *testing.T) {
	dtw := NewDTW()

	// First update
	distance1 := 5.0
	dtw.matchCount = 1
	dtw.updateAvgDistance(distance1)

	if dtw.avgDistance != distance1 {
		t.Errorf("First update: expected avgDistance=%f, got %f", distance1, dtw.avgDistance)
	}

	// Second update (should use exponential moving average)
	distance2 := 3.0
	dtw.matchCount = 2
	expectedAvg := 0.1*distance2 + 0.9*distance1 // alpha=0.1
	dtw.updateAvgDistance(distance2)

	if math.Abs(dtw.avgDistance-expectedAvg) > 0.0001 {
		t.Errorf("Second update: expected avgDistance=%f, got %f", expectedAvg, dtw.avgDistance)
	}
}

// Test DTW distance calculation with known values
func TestDTW_DistanceKnownValues(t *testing.T) {
	dtw := NewDTW()

	// Simple test case
	series1 := []float64{1.0, 2.0, 3.0}
	series2 := []float64{1.0, 2.0, 3.0}

	distance, err := dtw.Distance(series1, series2)
	if err != nil {
		t.Fatalf("Distance calculation failed: %v", err)
	}

	// Should be 0 for identical series
	if distance != 0.0 {
		t.Errorf("Expected distance=0.0 for identical series, got %f", distance)
	}

	// Test with offset series
	series3 := []float64{2.0, 3.0, 4.0}
	distance, err = dtw.Distance(series1, series3)
	if err != nil {
		t.Fatalf("Distance calculation failed: %v", err)
	}

	// Should be positive for different series
	if distance <= 0.0 {
		t.Errorf("Expected positive distance for different series, got %f", distance)
	}

	// Distance should be symmetric
	distance2, err := dtw.Distance(series3, series1)
	if err != nil {
		t.Fatalf("Reverse distance calculation failed: %v", err)
	}

	if math.Abs(distance-distance2) > 0.0001 {
		t.Errorf("Distance should be symmetric: %f != %f", distance, distance2)
	}
}

// Benchmark DTW performance
func BenchmarkDTW_Distance(b *testing.B) {
	dtw := NewDTW()

	series1 := make([]float64, 20)
	series2 := make([]float64, 20)

	for i := range series1 {
		series1[i] = float64(i)
		series2[i] = float64(i) + 0.1*float64(i%3)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dtw.Distance(series1, series2)
	}
}

func BenchmarkDTW_FindBestMatch(b *testing.B) {
	dtw := NewDTW()

	// Add some patterns
	for i := 0; i < 10; i++ {
		data := make([]float64, 10)
		for j := range data {
			data[j] = float64(i*j) * 0.1
		}
		dtw.AddPattern(data, "pattern", "test")
	}

	query := make([]float64, 10)
	for i := range query {
		query[i] = float64(i) * 0.15
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dtw.FindBestMatch(query)
	}
}

func BenchmarkDTW_AddPattern(b *testing.B) {
	dtw := NewDTW()

	data := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 4.0, 3.0, 2.0, 1.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dtw.AddPattern(data, "pattern", "test")
	}
}