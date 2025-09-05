package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// SpikeType defines the pattern of the spike
type SpikeType string

const (
	// Legacy types (kept for compatibility)
	PredictableDaily SpikeType = "predictable_daily"
	RandomPoisson    SpikeType = "random_poisson"
	Seasonal         SpikeType = "seasonal"
	Cascade          SpikeType = "cascade"
	GradualRamp      SpikeType = "gradual_ramp"
	
	// Realistic scheduling types
	DailyWithSchedule   SpikeType = "daily_with_schedule"
	PoissonWithSchedule SpikeType = "poisson_with_schedule"
	WeeklyWithSchedule  SpikeType = "weekly_with_schedule"
	MultiPeakDaily      SpikeType = "multi_peak_daily"
	CascadeWithSchedule SpikeType = "cascade_with_schedule"
	
	// Chaotic patterns
	ChaosBurst        SpikeType = "chaos_burst"
	RandomScatter     SpikeType = "random_scatter"
	BackgroundScatter SpikeType = "background_scatter"
)

// SpikeScenario defines a configurable spike pattern
type SpikeScenario struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Pattern     SpikePattern           `json:"pattern"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Peak defines a peak time with jitter for multi-peak patterns
type Peak struct {
	BaseTime           string  `json:"base_time"`
	TimeJitterMinutes  int     `json:"time_jitter_minutes"`
	DurationMinutes    int     `json:"duration_minutes"`
	Intensity          float64 `json:"intensity"`
}

// SpikePattern defines the characteristics of a spike
type SpikePattern struct {
	// Core fields
	Type                 SpikeType `json:"type"`
	ProcessRateMultiplier float64   `json:"process_rate_multiplier"`
	ExecutorType         string    `json:"executor_type"`
	PriorityDistribution []int     `json:"priority_distribution"`
	DataLocation         string    `json:"data_location"`
	DataSizeGB          float64   `json:"data_size_gb,omitempty"`
	
	// Legacy timing fields (for backward compatibility)
	TriggerTime          string    `json:"trigger_time,omitempty"`
	MeanIntervalMinutes  int       `json:"mean_interval_minutes,omitempty"`
	DurationMinutes      int       `json:"duration_minutes,omitempty"`
	RampUpMinutes       int       `json:"ramp_up_minutes,omitempty"`
	
	// Realistic scheduling fields
	BaseTime             string    `json:"base_time,omitempty"`              // Base time like "08:45:00"
	TimeJitterMinutes    int       `json:"time_jitter_minutes,omitempty"`    // ±random variance in minutes
	DurationJitterMinutes int      `json:"duration_jitter_minutes,omitempty"` // Duration variance
	BaseIntervalMinutes  int       `json:"base_interval_minutes,omitempty"`  // Base interval for Poisson
	IntervalJitterMinutes int      `json:"interval_jitter_minutes,omitempty"` // Interval variance
	
	// Scheduling constraints
	ActiveDays           []string  `json:"active_days,omitempty"`            // ["monday", "tuesday", ...]
	ActiveHours          []int     `json:"active_hours,omitempty"`           // [8, 9, 10, 17, 18, 19]
	Probability          float64   `json:"probability,omitempty"`            // 0.0-1.0 chance of occurring
	WeekendRateReduction float64   `json:"weekend_rate_reduction,omitempty"` // Multiplier for weekends
	WeekendIntensityReduction float64 `json:"weekend_intensity_reduction,omitempty"` // Intensity reduction on weekends
	
	// Multi-peak specific
	Peaks                []Peak    `json:"peaks,omitempty"`                  // For multi_peak_daily
	
	// Cascade specific
	CascadeStages        int       `json:"cascade_stages,omitempty"`         // Number of cascade stages
	StageDelayMinutes    int       `json:"stage_delay_minutes,omitempty"`    // Delay between stages
	StageJitterMinutes   int       `json:"stage_jitter_minutes,omitempty"`   // Jitter in stage delays
	
	// Chaos burst specific
	BurstIntensity       int       `json:"burst_intensity,omitempty"`        // Number of processes in burst
	MinQuietPeriodHours  int       `json:"min_quiet_period_hours,omitempty"` // Min hours between bursts
	MaxQuietPeriodHours  int       `json:"max_quiet_period_hours,omitempty"` // Max hours between bursts
	RandomTiming         bool      `json:"random_timing,omitempty"`          // Ignore schedule, use random timing
	PreferredHours       []int     `json:"preferred_hours,omitempty"`        // Preferred hours for random timing
	
	// Random scatter specific
	ScatterFrequencyHours int       `json:"scatter_frequency_hours,omitempty"` // Base hours between scatters
	ScatterJitterHours   int       `json:"scatter_jitter_hours,omitempty"`   // Jitter in scatter timing
	BurstSize            int       `json:"burst_size,omitempty"`             // Processes per scatter
	BurstSizeJitter      int       `json:"burst_size_jitter,omitempty"`      // Jitter in burst size
	
	// Background scatter specific
	BackgroundRate        int       `json:"background_rate,omitempty"`         // Constant background processes/hour
	MiniSpikeprobability float64   `json:"mini_spike_probability,omitempty"`  // Chance of mini spike per interval
	MiniSpikeMultiplier  float64   `json:"mini_spike_multiplier,omitempty"`  // Multiplier for mini spikes
	MiniSpikeDurationMinutes int   `json:"mini_spike_duration_minutes,omitempty"` // Duration of mini spikes
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
	// Legacy patterns (kept for compatibility)
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
	
	// Realistic patterns with scheduling
	case DailyWithSchedule:
		events = sg.generateScheduledDailySpikes(scenario, startTime, endTime)
	case PoissonWithSchedule:
		events = sg.generateScheduledPoissonSpikes(scenario, startTime, endTime)
	case WeeklyWithSchedule:
		events = sg.generateScheduledWeeklySpikes(scenario, startTime, endTime)
	case MultiPeakDaily:
		events = sg.generateMultiPeakDailySpikes(scenario, startTime, endTime)
	case CascadeWithSchedule:
		events = sg.generateScheduledCascadeSpikes(scenario, startTime, endTime)
	
	// Chaotic patterns
	case ChaosBurst:
		events = sg.generateChaosBurstSpikes(scenario, startTime, endTime)
	case RandomScatter:
		events = sg.generateRandomScatterSpikes(scenario, startTime, endTime)
	case BackgroundScatter:
		events = sg.generateBackgroundScatterSpikes(scenario, startTime, endTime)
	
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

// generateScheduledDailySpikes creates realistic daily spikes with jitter and scheduling constraints
func (sg *SpikeGenerator) generateScheduledDailySpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	pattern := scenario.Pattern
	
	// Parse base time
	baseHour, baseMin := sg.parseTriggerTime(pattern.BaseTime)
	if baseHour == -1 {
		baseHour, baseMin = sg.parseTriggerTime(pattern.TriggerTime) // fallback to legacy
	}
	
	currentDay := startTime
	eventID := 1
	
	for currentDay.Before(endTime) {
		// Check if this day is active
		if !sg.isDayActive(currentDay, pattern.ActiveDays) {
			currentDay = currentDay.Add(24 * time.Hour)
			continue
		}
		
		// Check probability
		if pattern.Probability > 0 && sg.random.Float64() > pattern.Probability {
			currentDay = currentDay.Add(24 * time.Hour)
			continue
		}
		
		// Calculate actual trigger time with jitter
		jitter := 0
		if pattern.TimeJitterMinutes > 0 {
			jitter = sg.random.Intn(2*pattern.TimeJitterMinutes) - pattern.TimeJitterMinutes
		}
		
		spikeTime := time.Date(
			currentDay.Year(), currentDay.Month(), currentDay.Day(),
			baseHour, baseMin, 0, 0, currentDay.Location(),
		).Add(time.Duration(jitter) * time.Minute)
		
		// Check if spike time is in active hours
		if !sg.isHourActive(spikeTime.Hour(), pattern.ActiveHours) {
			currentDay = currentDay.Add(24 * time.Hour)
			continue
		}
		
		// Calculate duration with jitter
		duration := pattern.DurationMinutes
		if duration == 0 {
			duration = 45 // default
		}
		if pattern.DurationJitterMinutes > 0 {
			durationJitter := sg.random.Intn(2*pattern.DurationJitterMinutes) - pattern.DurationJitterMinutes
			duration += durationJitter
			if duration < 5 {
				duration = 5 // minimum duration
			}
		}
		
		// Adjust intensity for weekends
		intensity := pattern.ProcessRateMultiplier
		if sg.isWeekend(spikeTime) && pattern.WeekendIntensityReduction > 0 {
			intensity *= (1.0 - pattern.WeekendIntensityReduction)
		}
		
		if spikeTime.After(startTime) && spikeTime.Before(endTime) {
			event := SpikeEvent{
				ID:                    fmt.Sprintf("%s-%d", scenario.Name, eventID),
				Name:                  scenario.Name,
				StartTime:             spikeTime,
				EndTime:               spikeTime.Add(time.Duration(duration) * time.Minute),
				PeakTime:              spikeTime.Add(time.Duration(duration/2) * time.Minute),
				ExecutorType:          pattern.ExecutorType,
				PriorityDistribution:  pattern.PriorityDistribution,
				DataLocation:          pattern.DataLocation,
				DataSizeGB:           pattern.DataSizeGB,
				IntensityProfile:      sg.createIntensityProfile(duration, intensity),
			}
			events = append(events, event)
			eventID++
		}
		
		currentDay = currentDay.Add(24 * time.Hour)
	}
	
	return events
}

// generateScheduledPoissonSpikes creates Poisson-distributed spikes with scheduling constraints
func (sg *SpikeGenerator) generateScheduledPoissonSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	pattern := scenario.Pattern
	
	// Use base interval with jitter
	baseInterval := pattern.BaseIntervalMinutes
	if baseInterval == 0 {
		baseInterval = pattern.MeanIntervalMinutes // fallback to legacy
	}
	if baseInterval == 0 {
		baseInterval = 90 // default
	}
	
	currentTime := startTime
	eventID := 1
	
	for currentTime.Before(endTime) {
		// Calculate next spike time with jitter
		interval := baseInterval
		if pattern.IntervalJitterMinutes > 0 {
			jitter := sg.random.Intn(2*pattern.IntervalJitterMinutes) - pattern.IntervalJitterMinutes
			interval += jitter
			if interval < 15 {
				interval = 15 // minimum interval
			}
		}
		
		// Add Poisson randomness
		poissonInterval := sg.poissonInterval(float64(interval))
		currentTime = currentTime.Add(time.Duration(poissonInterval) * time.Minute)
		
		if currentTime.After(endTime) {
			break
		}
		
		// Check scheduling constraints
		if !sg.isDayActive(currentTime, pattern.ActiveDays) {
			continue
		}
		
		if !sg.isHourActive(currentTime.Hour(), pattern.ActiveHours) {
			continue
		}
		
		if pattern.Probability > 0 && sg.random.Float64() > pattern.Probability {
			continue
		}
		
		// Calculate duration with jitter
		duration := pattern.DurationMinutes
		if duration == 0 {
			duration = 15 // default
		}
		if pattern.DurationJitterMinutes > 0 {
			durationJitter := sg.random.Intn(2*pattern.DurationJitterMinutes) - pattern.DurationJitterMinutes
			duration += durationJitter
			if duration < 5 {
				duration = 5
			}
		}
		
		// Adjust intensity for weekends
		intensity := pattern.ProcessRateMultiplier
		if sg.isWeekend(currentTime) && pattern.WeekendRateReduction > 0 {
			intensity *= pattern.WeekendRateReduction
		}
		
		event := SpikeEvent{
			ID:                    fmt.Sprintf("%s-%d", scenario.Name, eventID),
			Name:                  scenario.Name,
			StartTime:             currentTime,
			EndTime:               currentTime.Add(time.Duration(duration) * time.Minute),
			PeakTime:              currentTime.Add(time.Duration(duration/2) * time.Minute),
			ExecutorType:          pattern.ExecutorType,
			PriorityDistribution:  pattern.PriorityDistribution,
			DataLocation:          pattern.DataLocation,
			DataSizeGB:           pattern.DataSizeGB,
			IntensityProfile:      sg.createIntensityProfile(duration, intensity),
		}
		events = append(events, event)
		eventID++
	}
	
	return events
}

// Helper functions for scheduling
func (sg *SpikeGenerator) isDayActive(t time.Time, activeDays []string) bool {
	if len(activeDays) == 0 {
		return true // no restriction
	}
	
	dayName := strings.ToLower(t.Weekday().String())
	for _, activeDay := range activeDays {
		if strings.ToLower(activeDay) == dayName {
			return true
		}
	}
	return false
}

func (sg *SpikeGenerator) isHourActive(hour int, activeHours []int) bool {
	if len(activeHours) == 0 {
		return true // no restriction
	}
	
	for _, activeHour := range activeHours {
		if hour == activeHour {
			return true
		}
	}
	return false
}

func (sg *SpikeGenerator) isWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

// Stub implementations for other realistic spike types (can be implemented later)
func (sg *SpikeGenerator) generateScheduledWeeklySpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	// For now, delegate to scheduled daily spikes
	return sg.generateScheduledDailySpikes(scenario, startTime, endTime)
}

func (sg *SpikeGenerator) generateMultiPeakDailySpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	pattern := scenario.Pattern
	
	// Generate events for each peak
	for _, peak := range pattern.Peaks {
		// Create temporary scenario for this peak
		tempScenario := scenario
		tempScenario.Pattern.BaseTime = peak.BaseTime
		tempScenario.Pattern.TimeJitterMinutes = peak.TimeJitterMinutes
		tempScenario.Pattern.DurationMinutes = peak.DurationMinutes
		tempScenario.Pattern.ProcessRateMultiplier = peak.Intensity
		
		peakEvents := sg.generateScheduledDailySpikes(tempScenario, startTime, endTime)
		events = append(events, peakEvents...)
	}
	
	return events
}

func (sg *SpikeGenerator) generateScheduledCascadeSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	// For now, delegate to scheduled Poisson spikes
	return sg.generateScheduledPoissonSpikes(scenario, startTime, endTime)
}

// poissonInterval calculates Poisson-distributed interval
func (sg *SpikeGenerator) poissonInterval(lambda float64) int {
	// Simple exponential distribution approximation
	u := sg.random.Float64()
	return int(-lambda * math.Log(1-u))
}

// createIntensityProfile creates a simple bell-curve intensity profile for a spike
func (sg *SpikeGenerator) createIntensityProfile(durationMinutes int, intensity float64) []float64 {
	profile := make([]float64, durationMinutes)
	
	for i := 0; i < durationMinutes; i++ {
		t := float64(i) / float64(durationMinutes)
		
		// Bell curve intensity (normal distribution shape)
		bellCurve := math.Exp(-math.Pow((t-0.5)*4, 2) / 2)
		profile[i] = intensity * bellCurve
	}
	
	return profile
}

// generateChaosBurstSpikes creates unpredictable burst events with long quiet periods
func (sg *SpikeGenerator) generateChaosBurstSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	current := startTime
	eventCounter := 0
	
	for current.Before(endTime) {
		// Check if this day is active
		if !sg.isDayActive(current, scenario.Pattern.ActiveDays) {
			current = current.Add(24 * time.Hour)
			continue
		}
		
		// Check probability
		if scenario.Pattern.Probability > 0 && sg.random.Float64() > scenario.Pattern.Probability {
			current = current.Add(24 * time.Hour)
			continue
		}
		
		// Generate burst timing
		var burstTime time.Time
		if scenario.Pattern.RandomTiming {
			// Random timing within preferred hours or full day
			if len(scenario.Pattern.PreferredHours) > 0 {
				hour := scenario.Pattern.PreferredHours[sg.random.Intn(len(scenario.Pattern.PreferredHours))]
				minute := sg.random.Intn(60)
				burstTime = time.Date(current.Year(), current.Month(), current.Day(), hour, minute, 0, 0, current.Location())
			} else {
				hour := sg.random.Intn(24)
				minute := sg.random.Intn(60)
				burstTime = time.Date(current.Year(), current.Month(), current.Day(), hour, minute, 0, 0, current.Location())
			}
		} else {
			// Use schedule-based timing if base_time is provided
			if scenario.Pattern.BaseTime != "" {
				hour, minute := sg.parseTriggerTime(scenario.Pattern.BaseTime)
				burstTime = time.Date(current.Year(), current.Month(), current.Day(), hour, minute, 0, 0, current.Location())
				if scenario.Pattern.TimeJitterMinutes > 0 {
					jitter := sg.random.Intn(scenario.Pattern.TimeJitterMinutes*2) - scenario.Pattern.TimeJitterMinutes
					burstTime = burstTime.Add(time.Duration(jitter) * time.Minute)
				}
			} else {
				burstTime = current.Add(time.Duration(sg.random.Intn(24*60)) * time.Minute)
			}
		}
		
		// Skip if burst time is outside simulation window
		if burstTime.Before(startTime) || burstTime.After(endTime) {
			current = current.Add(24 * time.Hour)
			continue
		}
		
		// Create burst event
		duration := time.Duration(scenario.Pattern.DurationMinutes) * time.Minute
		if scenario.Pattern.DurationJitterMinutes > 0 {
			jitter := sg.random.Intn(scenario.Pattern.DurationJitterMinutes*2) - scenario.Pattern.DurationJitterMinutes
			duration += time.Duration(jitter) * time.Minute
		}
		
		eventCounter++
		event := SpikeEvent{
			ID:                  fmt.Sprintf("%s-burst-%d", scenario.Name, eventCounter),
			Name:                scenario.Name,
			StartTime:           burstTime,
			EndTime:             burstTime.Add(duration),
			PeakTime:            burstTime.Add(duration / 3), // Peak early in burst
			ProcessesGenerated:  scenario.Pattern.BurstIntensity,
			ExecutorType:        scenario.Pattern.ExecutorType,
			PriorityDistribution: scenario.Pattern.PriorityDistribution,
			DataLocation:        scenario.Pattern.DataLocation,
			DataSizeGB:          scenario.Pattern.DataSizeGB,
			IntensityProfile:    sg.createChaosIntensityProfile(int(duration.Minutes()), scenario.Pattern.BurstIntensity),
		}
		
		events = append(events, event)
		
		// Jump to next possible burst time (quiet period)
		quietHours := scenario.Pattern.MinQuietPeriodHours
		if scenario.Pattern.MaxQuietPeriodHours > scenario.Pattern.MinQuietPeriodHours {
			quietHours += sg.random.Intn(scenario.Pattern.MaxQuietPeriodHours - scenario.Pattern.MinQuietPeriodHours)
		}
		current = burstTime.Add(time.Duration(quietHours) * time.Hour)
	}
	
	return events
}

// generateRandomScatterSpikes creates irregular scattered spike events
func (sg *SpikeGenerator) generateRandomScatterSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	current := startTime
	eventCounter := 0
	
	for current.Before(endTime) {
		// Check if this day is active
		if !sg.isDayActive(current, scenario.Pattern.ActiveDays) {
			current = current.Add(24 * time.Hour)
			continue
		}
		
		// Check probability
		if scenario.Pattern.Probability > 0 && sg.random.Float64() > scenario.Pattern.Probability {
			current = current.Add(24 * time.Hour)
			continue
		}
		
		// Apply weekend rate reduction
		multiplier := 1.0
		if sg.isWeekend(current) && scenario.Pattern.WeekendRateReduction > 0 {
			multiplier = scenario.Pattern.WeekendRateReduction
		}
		
		// Generate scatter events throughout the day
		dayStart := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
		dayEnd := dayStart.Add(24 * time.Hour)
		
		// Calculate number of scatters for this day
		baseScatters := int(24 / scenario.Pattern.ScatterFrequencyHours)
		if baseScatters < 1 {
			baseScatters = 1
		}
		
		for scatter := 0; scatter < int(float64(baseScatters)*multiplier)+1; scatter++ {
			// Random scatter time with jitter
			baseInterval := time.Duration(scenario.Pattern.ScatterFrequencyHours) * time.Hour
			jitter := time.Duration(sg.random.Intn(scenario.Pattern.ScatterJitterHours*120)-scenario.Pattern.ScatterJitterHours*60) * time.Minute
			scatterTime := dayStart.Add(time.Duration(scatter)*baseInterval + jitter)
			
			// Skip if outside day bounds or simulation window
			if scatterTime.Before(dayStart) || scatterTime.After(dayEnd) || scatterTime.Before(startTime) || scatterTime.After(endTime) {
				continue
			}
			
			// Calculate burst size with jitter
			burstSize := scenario.Pattern.BurstSize
			if scenario.Pattern.BurstSizeJitter > 0 {
				jitter := sg.random.Intn(scenario.Pattern.BurstSizeJitter*2) - scenario.Pattern.BurstSizeJitter
				burstSize += jitter
				if burstSize < 1 {
					burstSize = 1
				}
			}
			
			// Create scatter event
			duration := time.Duration(5+sg.random.Intn(10)) * time.Minute // Short bursts
			
			eventCounter++
			event := SpikeEvent{
				ID:                  fmt.Sprintf("%s-scatter-%d", scenario.Name, eventCounter),
				Name:                scenario.Name,
				StartTime:           scatterTime,
				EndTime:             scatterTime.Add(duration),
				PeakTime:            scatterTime.Add(duration / 2),
				ProcessesGenerated:  burstSize,
				ExecutorType:        scenario.Pattern.ExecutorType,
				PriorityDistribution: scenario.Pattern.PriorityDistribution,
				DataLocation:        scenario.Pattern.DataLocation,
				DataSizeGB:          scenario.Pattern.DataSizeGB,
				IntensityProfile:    sg.createIntensityProfile(int(duration.Minutes()), float64(burstSize)),
			}
			
			events = append(events, event)
		}
		
		current = current.Add(24 * time.Hour)
	}
	
	return events
}

// generateBackgroundScatterSpikes creates low-intensity background processing with mini-spikes
func (sg *SpikeGenerator) generateBackgroundScatterSpikes(scenario SpikeScenario, startTime, endTime time.Time) []SpikeEvent {
	var events []SpikeEvent
	
	current := startTime
	eventCounter := 0
	
	// Generate background events every hour
	for current.Before(endTime) {
		// Check if this day is active
		if !sg.isDayActive(current, scenario.Pattern.ActiveDays) {
			current = current.Add(time.Hour)
			continue
		}
		
		// Check probability
		if scenario.Pattern.Probability > 0 && sg.random.Float64() > scenario.Pattern.Probability {
			current = current.Add(time.Hour)
			continue
		}
		
		// Background processing
		if scenario.Pattern.BackgroundRate > 0 {
			processCount := scenario.Pattern.BackgroundRate
			
			// Check for mini-spike
			if sg.random.Float64() < scenario.Pattern.MiniSpikeprobability {
				processCount = int(float64(processCount) * scenario.Pattern.MiniSpikeMultiplier)
				
				// Create mini-spike event
				duration := time.Duration(scenario.Pattern.MiniSpikeDurationMinutes) * time.Minute
				
				eventCounter++
				event := SpikeEvent{
					ID:                  fmt.Sprintf("%s-minispike-%d", scenario.Name, eventCounter),
					Name:                scenario.Name + " Mini-Spike",
					StartTime:           current,
					EndTime:             current.Add(duration),
					PeakTime:            current.Add(duration / 2),
					ProcessesGenerated:  processCount,
					ExecutorType:        scenario.Pattern.ExecutorType,
					PriorityDistribution: scenario.Pattern.PriorityDistribution,
					DataLocation:        scenario.Pattern.DataLocation,
					DataSizeGB:          scenario.Pattern.DataSizeGB,
					IntensityProfile:    sg.createIntensityProfile(int(duration.Minutes()), float64(processCount)),
				}
				
				events = append(events, event)
			} else if processCount > 0 {
				// Create regular background event
				duration := 60 * time.Minute // 1 hour background processing
				
				eventCounter++
				event := SpikeEvent{
					ID:                  fmt.Sprintf("%s-background-%d", scenario.Name, eventCounter),
					Name:                scenario.Name + " Background",
					StartTime:           current,
					EndTime:             current.Add(duration),
					PeakTime:            current.Add(duration / 2),
					ProcessesGenerated:  processCount,
					ExecutorType:        scenario.Pattern.ExecutorType,
					PriorityDistribution: scenario.Pattern.PriorityDistribution,
					DataLocation:        scenario.Pattern.DataLocation,
					DataSizeGB:          scenario.Pattern.DataSizeGB,
					IntensityProfile:    sg.createFlatIntensityProfile(60, float64(processCount)),
				}
				
				events = append(events, event)
			}
		}
		
		current = current.Add(time.Hour)
	}
	
	return events
}

// createChaosIntensityProfile creates an intense burst profile with rapid spike
func (sg *SpikeGenerator) createChaosIntensityProfile(durationMinutes int, totalProcesses int) []float64 {
	profile := make([]float64, durationMinutes)
	
	if durationMinutes == 0 {
		return profile
	}
	
	// Chaos burst: very high intensity in first 1/3, then rapid decline
	peakDuration := durationMinutes / 3
	if peakDuration < 1 {
		peakDuration = 1
	}
	
	for i := 0; i < durationMinutes; i++ {
		if i < peakDuration {
			// High intensity peak
			profile[i] = float64(totalProcesses) * 0.7 / float64(peakDuration)
		} else {
			// Rapid decline
			falloff := float64(i-peakDuration) / float64(durationMinutes-peakDuration)
			profile[i] = float64(totalProcesses) * 0.3 * (1 - falloff) / float64(durationMinutes-peakDuration)
		}
	}
	
	return profile
}

// createFlatIntensityProfile creates a flat, steady processing profile
func (sg *SpikeGenerator) createFlatIntensityProfile(durationMinutes int, totalProcesses float64) []float64 {
	profile := make([]float64, durationMinutes)
	
	if durationMinutes == 0 {
		return profile
	}
	
	processesPerMinute := totalProcesses / float64(durationMinutes)
	for i := 0; i < durationMinutes; i++ {
		profile[i] = processesPerMinute
	}
	
	return profile
}