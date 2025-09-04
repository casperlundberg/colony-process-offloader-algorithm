package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// SpikeType defines the pattern of the spike
type SpikeType string

const (
	PredictableDaily SpikeType = "predictable_daily"
	RandomPoisson    SpikeType = "random_poisson"
	Seasonal         SpikeType = "seasonal"
	Cascade          SpikeType = "cascade"
	GradualRamp      SpikeType = "gradual_ramp"
)

// SpikeScenario defines a configurable spike pattern
type SpikeScenario struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Pattern     SpikePattern           `json:"pattern"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SpikePattern defines the characteristics of a spike
type SpikePattern struct {
	Type                 SpikeType `json:"type"`
	TriggerTime          string    `json:"trigger_time,omitempty"`          // For daily patterns, e.g., "09:00:00"
	MeanIntervalMinutes  int       `json:"mean_interval_minutes,omitempty"` // For Poisson distribution
	DurationMinutes      int       `json:"duration_minutes"`
	ProcessRateMultiplier float64   `json:"process_rate_multiplier"`
	ExecutorType         string    `json:"executor_type"`
	PriorityDistribution []int     `json:"priority_distribution"`
	DataLocation         string    `json:"data_location"`
	DataSizeGB          float64   `json:"data_size_gb,omitempty"`
	RampUpMinutes       int       `json:"ramp_up_minutes,omitempty"` // For gradual ramp
}

// SpikeEvent represents an actual spike occurrence during simulation
type SpikeEvent struct {
	ID                  string
	Name                string
	StartTime           time.Time
	EndTime             time.Time
	PeakTime            time.Time
	ProcessesGenerated  int
	ExecutorType        string
	PriorityDistribution []int
	DataLocation        string
	DataSizeGB          float64
	IntensityProfile    []float64 // Process rate over time
}

// SpikeGenerator creates spike events from scenarios
type SpikeGenerator struct {
	scenarios       []SpikeScenario
	baseProcessRate float64 // Normal processes per minute
	simulationStart time.Time
	random          *rand.Rand
}

// NewSpikeGenerator creates a new spike generator
func NewSpikeGenerator(baseProcessRate float64, scenarios []SpikeScenario) *SpikeGenerator {
	return &SpikeGenerator{
		scenarios:       scenarios,
		baseProcessRate: baseProcessRate,
		simulationStart: time.Now(),
		random:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateSpikeTimeline creates a timeline of spike events for the simulation duration
func (sg *SpikeGenerator) GenerateSpikeTimeline(durationHours int) ([]SpikeEvent, error) {
	var events []SpikeEvent
	endTime := sg.simulationStart.Add(time.Duration(durationHours) * time.Hour)
	
	for _, scenario := range sg.scenarios {
		scenarioEvents, err := sg.generateScenarioEvents(scenario, sg.simulationStart, endTime)
		if err != nil {
			return nil, fmt.Errorf("failed to generate events for scenario %s: %w", scenario.Name, err)
		}
		events = append(events, scenarioEvents...)
	}
	
	// Sort events by start time
	sg.sortEventsByTime(events)
	
	return events, nil
}

// generateScenarioEvents generates spike events for a specific scenario
func (sg *SpikeGenerator) generateScenarioEvents(scenario SpikeScenario, startTime, endTime time.Time) ([]SpikeEvent, error) {
	var events []SpikeEvent
	
	switch scenario.Pattern.Type {
	case PredictableDaily:
		events = sg.generateDailySpikes(scenario, startTime, endTime)
	case RandomPoisson:
		events = sg.generatePoissonSpikes(scenario, startTime, endTime)
	case Seasonal:
		events = sg.generateSeasonalSpikes(scenario, startTime, endTime)
	case Cascade:
		events = sg.generateCascadeSpikes(scenario, startTime, endTime)
	case GradualRamp:
		events = sg.generateGradualRampSpikes(scenario, startTime, endTime)
	default:
		return nil, fmt.Errorf("unknown spike type: %s", scenario.Pattern.Type)
	}
	
	return events, nil
}

// generateDailySpikes creates recurring daily spike events
func (sg *SpikeGenerator) generateDailySpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	// Parse trigger time (e.g., "09:00:00")
	triggerHour, triggerMin := sg.parseTriggerTime(scenario.Pattern.TriggerTime)
	
	currentDay := startTime
	eventID := 1
	
	for currentDay.Before(endTime) {
		// Calculate spike time for this day
		spikeStart := time.Date(
			currentDay.Year(), currentDay.Month(), currentDay.Day(),
			triggerHour, triggerMin, 0, 0, currentDay.Location(),
		)
		
		if spikeStart.After(startTime) && spikeStart.Before(endTime) {
			event := SpikeEvent{
				ID:                   fmt.Sprintf("%s-%d", scenario.Name, eventID),
				Name:                 scenario.Name,
				StartTime:            spikeStart,
				EndTime:              spikeStart.Add(time.Duration(scenario.Pattern.DurationMinutes) * time.Minute),
				PeakTime:             spikeStart.Add(time.Duration(scenario.Pattern.DurationMinutes/2) * time.Minute),
				ExecutorType:         scenario.Pattern.ExecutorType,
				PriorityDistribution: scenario.Pattern.PriorityDistribution,
				DataLocation:         scenario.Pattern.DataLocation,
				DataSizeGB:           scenario.Pattern.DataSizeGB,
			}
			
			// Generate intensity profile
			event.IntensityProfile = sg.generateIntensityProfile(scenario.Pattern, event.StartTime, event.EndTime)
			event.ProcessesGenerated = sg.calculateTotalProcesses(event.IntensityProfile)
			
			events = append(events, event)
			eventID++
		}
		
		// Move to next day
		currentDay = currentDay.Add(24 * time.Hour)
	}
	
	return events
}

// generatePoissonSpikes creates random spikes following Poisson distribution
func (sg *SpikeGenerator) generatePoissonSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	currentTime := startTime
	eventID := 1
	meanInterval := time.Duration(scenario.Pattern.MeanIntervalMinutes) * time.Minute
	
	for currentTime.Before(endTime) {
		// Generate next spike time using exponential distribution
		interval := sg.exponentialInterval(meanInterval)
		nextSpikeTime := currentTime.Add(interval)
		
		if nextSpikeTime.After(endTime) {
			break
		}
		
		event := SpikeEvent{
			ID:                   fmt.Sprintf("%s-%d", scenario.Name, eventID),
			Name:                 scenario.Name,
			StartTime:            nextSpikeTime,
			EndTime:              nextSpikeTime.Add(time.Duration(scenario.Pattern.DurationMinutes) * time.Minute),
			PeakTime:             nextSpikeTime.Add(time.Duration(scenario.Pattern.DurationMinutes/2) * time.Minute),
			ExecutorType:         scenario.Pattern.ExecutorType,
			PriorityDistribution: scenario.Pattern.PriorityDistribution,
			DataLocation:         scenario.Pattern.DataLocation,
			DataSizeGB:           scenario.Pattern.DataSizeGB,
		}
		
		// Add some randomness to intensity
		multiplier := scenario.Pattern.ProcessRateMultiplier * (0.8 + sg.random.Float64()*0.4)
		modifiedPattern := scenario.Pattern
		modifiedPattern.ProcessRateMultiplier = multiplier
		
		event.IntensityProfile = sg.generateIntensityProfile(modifiedPattern, event.StartTime, event.EndTime)
		event.ProcessesGenerated = sg.calculateTotalProcesses(event.IntensityProfile)
		
		events = append(events, event)
		eventID++
		currentTime = nextSpikeTime
	}
	
	return events
}

// generateSeasonalSpikes creates weekly/monthly pattern spikes
func (sg *SpikeGenerator) generateSeasonalSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	// Weekly pattern: higher load on Monday-Friday, peak on Wednesday
	currentTime := startTime
	eventID := 1
	
	for currentTime.Before(endTime) {
		weekday := currentTime.Weekday()
		
		// Skip weekends for business seasonal patterns
		if weekday != time.Saturday && weekday != time.Sunday {
			// Intensity varies by day of week
			dayMultiplier := 1.0
			switch weekday {
			case time.Monday, time.Friday:
				dayMultiplier = 0.8
			case time.Tuesday, time.Thursday:
				dayMultiplier = 0.9
			case time.Wednesday:
				dayMultiplier = 1.2
			}
			
			// Create morning and afternoon spikes
			for _, hour := range []int{10, 15} {
				spikeStart := time.Date(
					currentTime.Year(), currentTime.Month(), currentTime.Day(),
					hour, 0, 0, 0, currentTime.Location(),
				)
				
				if spikeStart.Before(endTime) && spikeStart.After(startTime) {
					event := SpikeEvent{
						ID:                   fmt.Sprintf("%s-%d", scenario.Name, eventID),
						Name:                 scenario.Name,
						StartTime:            spikeStart,
						EndTime:              spikeStart.Add(time.Duration(scenario.Pattern.DurationMinutes) * time.Minute),
						PeakTime:             spikeStart.Add(time.Duration(scenario.Pattern.DurationMinutes/2) * time.Minute),
						ExecutorType:         scenario.Pattern.ExecutorType,
						PriorityDistribution: scenario.Pattern.PriorityDistribution,
						DataLocation:         scenario.Pattern.DataLocation,
						DataSizeGB:           scenario.Pattern.DataSizeGB,
					}
					
					// Adjust intensity based on day
					modifiedPattern := scenario.Pattern
					modifiedPattern.ProcessRateMultiplier *= dayMultiplier
					
					event.IntensityProfile = sg.generateIntensityProfile(modifiedPattern, event.StartTime, event.EndTime)
					event.ProcessesGenerated = sg.calculateTotalProcesses(event.IntensityProfile)
					
					events = append(events, event)
					eventID++
				}
			}
		}
		
		currentTime = currentTime.Add(24 * time.Hour)
	}
	
	return events
}

// generateCascadeSpikes creates spikes that trigger secondary spikes
func (sg *SpikeGenerator) generateCascadeSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	// Generate primary spike
	primaryTime := startTime.Add(2 * time.Hour) // Start after 2 hours
	eventID := 1
	
	for primaryTime.Before(endTime) {
		// Primary spike
		primaryEvent := SpikeEvent{
			ID:                   fmt.Sprintf("%s-primary-%d", scenario.Name, eventID),
			Name:                 fmt.Sprintf("%s-primary", scenario.Name),
			StartTime:            primaryTime,
			EndTime:              primaryTime.Add(time.Duration(scenario.Pattern.DurationMinutes) * time.Minute),
			PeakTime:             primaryTime.Add(time.Duration(scenario.Pattern.DurationMinutes/2) * time.Minute),
			ExecutorType:         scenario.Pattern.ExecutorType,
			PriorityDistribution: scenario.Pattern.PriorityDistribution,
			DataLocation:         scenario.Pattern.DataLocation,
			DataSizeGB:           scenario.Pattern.DataSizeGB,
		}
		
		primaryEvent.IntensityProfile = sg.generateIntensityProfile(scenario.Pattern, primaryEvent.StartTime, primaryEvent.EndTime)
		primaryEvent.ProcessesGenerated = sg.calculateTotalProcesses(primaryEvent.IntensityProfile)
		events = append(events, primaryEvent)
		
		// Generate cascade (secondary) spikes
		cascadeDelay := 10 * time.Minute
		for i := 1; i <= 2; i++ {
			cascadeTime := primaryEvent.EndTime.Add(time.Duration(i) * cascadeDelay)
			if cascadeTime.Before(endTime) {
				cascadeEvent := SpikeEvent{
					ID:                   fmt.Sprintf("%s-cascade-%d-%d", scenario.Name, eventID, i),
					Name:                 fmt.Sprintf("%s-cascade-%d", scenario.Name, i),
					StartTime:            cascadeTime,
					EndTime:              cascadeTime.Add(time.Duration(scenario.Pattern.DurationMinutes/2) * time.Minute),
					PeakTime:             cascadeTime.Add(time.Duration(scenario.Pattern.DurationMinutes/4) * time.Minute),
					ExecutorType:         scenario.Pattern.ExecutorType,
					PriorityDistribution: scenario.Pattern.PriorityDistribution,
					DataLocation:         scenario.Pattern.DataLocation,
					DataSizeGB:           scenario.Pattern.DataSizeGB / 2, // Smaller secondary spikes
				}
				
				// Cascade spikes are smaller
				modifiedPattern := scenario.Pattern
				modifiedPattern.ProcessRateMultiplier *= (0.6 - float64(i)*0.1)
				
				cascadeEvent.IntensityProfile = sg.generateIntensityProfile(modifiedPattern, cascadeEvent.StartTime, cascadeEvent.EndTime)
				cascadeEvent.ProcessesGenerated = sg.calculateTotalProcesses(cascadeEvent.IntensityProfile)
				events = append(events, cascadeEvent)
			}
		}
		
		// Next primary spike after 6 hours
		primaryTime = primaryTime.Add(6 * time.Hour)
		eventID++
	}
	
	return events
}

// generateGradualRampSpikes creates slowly building spike patterns
func (sg *SpikeGenerator) generateGradualRampSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	currentTime := startTime.Add(1 * time.Hour)
	eventID := 1
	
	for currentTime.Before(endTime) {
		event := SpikeEvent{
			ID:                   fmt.Sprintf("%s-%d", scenario.Name, eventID),
			Name:                 scenario.Name,
			StartTime:            currentTime,
			EndTime:              currentTime.Add(time.Duration(scenario.Pattern.DurationMinutes) * time.Minute),
			PeakTime:             currentTime.Add(time.Duration(scenario.Pattern.DurationMinutes*3/4) * time.Minute), // Peak near end
			ExecutorType:         scenario.Pattern.ExecutorType,
			PriorityDistribution: scenario.Pattern.PriorityDistribution,
			DataLocation:         scenario.Pattern.DataLocation,
			DataSizeGB:           scenario.Pattern.DataSizeGB,
		}
		
		// Generate gradual ramp intensity profile
		event.IntensityProfile = sg.generateGradualRampProfile(scenario.Pattern, event.StartTime, event.EndTime)
		event.ProcessesGenerated = sg.calculateTotalProcesses(event.IntensityProfile)
		
		events = append(events, event)
		
		// Next ramp after 4 hours
		currentTime = currentTime.Add(4 * time.Hour)
		eventID++
	}
	
	return events
}

// generateIntensityProfile creates the process rate profile over the spike duration
func (sg *SpikeGenerator) generateIntensityProfile(pattern SpikePattern, startTime, endTime time.Time) []float64 {
	duration := endTime.Sub(startTime)
	minutes := int(duration.Minutes())
	profile := make([]float64, minutes)
	
	for i := 0; i < minutes; i++ {
		t := float64(i) / float64(minutes)
		
		// Bell curve intensity (normal distribution shape)
		intensity := math.Exp(-math.Pow((t-0.5)*4, 2) / 2)
		
		// Apply multiplier and base rate
		profile[i] = sg.baseProcessRate * (1 + intensity*(pattern.ProcessRateMultiplier-1))
	}
	
	return profile
}

// generateGradualRampProfile creates a gradually increasing intensity profile
func (sg *SpikeGenerator) generateGradualRampProfile(pattern SpikePattern, startTime, endTime time.Time) []float64 {
	duration := endTime.Sub(startTime)
	minutes := int(duration.Minutes())
	profile := make([]float64, minutes)
	
	rampMinutes := pattern.RampUpMinutes
	if rampMinutes == 0 {
		rampMinutes = minutes / 2
	}
	
	for i := 0; i < minutes; i++ {
		var intensity float64
		
		if i < rampMinutes {
			// Ramp up phase
			intensity = float64(i) / float64(rampMinutes)
		} else if i < minutes-10 {
			// Sustained peak
			intensity = 1.0
		} else {
			// Quick drop off
			intensity = float64(minutes-i) / 10.0
		}
		
		// Apply multiplier and base rate
		profile[i] = sg.baseProcessRate * (1 + intensity*(pattern.ProcessRateMultiplier-1))
	}
	
	return profile
}

// calculateTotalProcesses sums up the total processes from an intensity profile
func (sg *SpikeGenerator) calculateTotalProcesses(profile []float64) int {
	total := 0.0
	for _, rate := range profile {
		total += rate
	}
	return int(total)
}

// parseTriggerTime parses time string like "09:00:00" or "14:30:00"
func (sg *SpikeGenerator) parseTriggerTime(timeStr string) (hour, minute int) {
	// Default to 9 AM if parsing fails
	hour, minute = 9, 0
	
	if timeStr == "" {
		return hour, minute
	}
	
	var h, m, s int
	fmt.Sscanf(timeStr, "%d:%d:%d", &h, &m, &s)
	
	if h >= 0 && h < 24 {
		hour = h
	}
	if m >= 0 && m < 60 {
		minute = m
	}
	
	return hour, minute
}

// exponentialInterval generates random interval following exponential distribution
func (sg *SpikeGenerator) exponentialInterval(mean time.Duration) time.Duration {
	// Generate exponential random variable
	u := sg.random.Float64()
	interval := -math.Log(1-u) * mean.Seconds()
	return time.Duration(interval) * time.Second
}

// sortEventsByTime sorts spike events by start time
func (sg *SpikeGenerator) sortEventsByTime(events []SpikeEvent) {
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].StartTime.Before(events[i].StartTime) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}