package colonyos

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/casperlundberg/colony-process-offloader-algorithm/pkg/models"
)

// ConfigLoader handles loading configuration from JSON files
type ConfigLoader struct {
	humanConfigPath   string
	metricsDataPath   string
	resourceParser    *ResourceParser
}

// HumanConfig represents the human-provided configuration
type HumanConfig struct {
	CapeConfig struct {
		DeploymentType string `json:"deployment_type"`
		Description    string `json:"description"`
		OptimizationGoals []struct {
			Metric      string  `json:"metric"`
			Weight      float64 `json:"weight"`
			Minimize    bool    `json:"minimize"`
			Description string  `json:"description"`
		} `json:"optimization_goals"`
		Constraints []struct {
			Type        string `json:"type"`
			Value       string `json:"value"`
			IsHard      bool   `json:"is_hard"`
			Description string `json:"description"`
		} `json:"constraints"`
		LearningParameters struct {
			DataGravityFactor          float64 `json:"data_gravity_factor"`
			ExplorationFactor          float64 `json:"exploration_factor"`
			LearningRate               float64 `json:"learning_rate"`
			AdaptationIntervalMinutes  int     `json:"adaptation_interval_minutes"`
			MinDecisionsBeforeAdaptation int   `json:"min_decisions_before_adaptation"`
			StrategyEnabled            bool    `json:"strategy_enabled"`
		} `json:"learning_parameters"`
		AlgorithmParameters struct {
			ArimaOrder                []int   `json:"arima_order"`
			EwmaAlpha                 float64 `json:"ewma_alpha"`
			CusumDriftParam           float64 `json:"cusum_drift_param"`
			CusumDetectionThreshold   float64 `json:"cusum_detection_threshold"`
			SgdLearningRate          float64 `json:"sgd_learning_rate"`
			ThompsonSamplingEnabled   bool    `json:"thompson_sampling_enabled"`
			QLearningEnabled          bool    `json:"q_learning_enabled"`
			QLearningDiscount         float64 `json:"q_learning_discount"`
		} `json:"algorithm_parameters"`
	} `json:"cape_config"`
	OrchestratorConfig struct {
		ExecutorName string `json:"executor_name"`
		ExecutorType string `json:"executor_type"`
		Location struct {
			Longitude   float64 `json:"longitude"`
			Latitude    float64 `json:"latitude"`
			Description string  `json:"description"`
		} `json:"location"`
		Capabilities struct {
			Hardware struct {
				Nodes   int      `json:"nodes"`
				CPU     string   `json:"cpu"`
				Memory  string   `json:"memory"`
				Storage string   `json:"storage"`
				GPUs    []string `json:"gpus"`
			} `json:"hardware"`
			Software struct {
				Name    string `json:"name"`
				Type    string `json:"type"`
				Version string `json:"version"`
			} `json:"software"`
		} `json:"capabilities"`
		Behavior struct {
			AssignIntervalSeconds         int  `json:"assign_interval_seconds"`
			MetricsUpdateIntervalSeconds  int  `json:"metrics_update_interval_seconds"`
			DecisionTimeoutSeconds        int  `json:"decision_timeout_seconds"`
			MaxConcurrentProcesses        int  `json:"max_concurrent_processes"`
			RequireFunctionRegistration   bool `json:"require_function_registration"`
		} `json:"behavior"`
		SupportedFunctions []string `json:"supported_functions"`
	} `json:"orchestrator_config"`
	ColonyOSConnection struct {
		ServerURL                string `json:"server_url"`
		ColonyName               string `json:"colony_name"`
		UseTLS                   bool   `json:"use_tls"`
		SkipTLSVerify            bool   `json:"skip_tls_verify"`
		ConnectionTimeoutSeconds int    `json:"connection_timeout_seconds"`
		RetryIntervalSeconds     int    `json:"retry_interval_seconds"`
		MaxRetryAttempts         int    `json:"max_retry_attempts"`
	} `json:"colonyos_connection"`
	Monitoring struct {
		EnablePrometheusMetrics bool   `json:"enable_prometheus_metrics"`
		PrometheusPort          int    `json:"prometheus_port"`
		MetricsPrefix           string `json:"metrics_prefix"`
		LogLevel                string `json:"log_level"`
		EnableDecisionAudit     bool   `json:"enable_decision_audit"`
		AuditLogPath            string `json:"audit_log_path"`
	} `json:"monitoring"`
}

// ColonyOSMetrics represents the metrics and data from ColonyOS
type ColonyOSMetrics struct {
	Timestamp         time.Time  `json:"timestamp"`
	ColonyStatistics  Statistics `json:"colony_statistics"`
	PrometheusMetrics map[string]float64 `json:"prometheus_metrics"`
	SystemHealth struct {
		ServerUptimeSeconds       int64     `json:"server_uptime_seconds"`
		MemoryUsageBytes          int64     `json:"memory_usage_bytes"`
		CPUUsagePercent           float64   `json:"cpu_usage_percent"`
		ActiveConnections         int       `json:"active_connections"`
		AvgProcessQueueTimeMs     int       `json:"avg_process_queue_time_ms"`
		AvgProcessExecutionTimeMs int       `json:"avg_process_execution_time_ms"`
		LastHealthCheck           time.Time `json:"last_health_check"`
	} `json:"system_health"`
	ResourceUtilization struct {
		TotalCapacity struct {
			CPUMillicores int64 `json:"cpu_millicores"`
			MemoryBytes   int64 `json:"memory_bytes"`
			StorageBytes  int64 `json:"storage_bytes"`
			GPUCount      int   `json:"gpu_count"`
		} `json:"total_capacity"`
		AllocatedCapacity struct {
			CPUMillicores int64 `json:"cpu_millicores"`
			MemoryBytes   int64 `json:"memory_bytes"`
			StorageBytes  int64 `json:"storage_bytes"`
			GPUCount      int   `json:"gpu_count"`
		} `json:"allocated_capacity"`
		AvailableCapacity struct {
			CPUMillicores int64 `json:"cpu_millicores"`
			MemoryBytes   int64 `json:"memory_bytes"`
			StorageBytes  int64 `json:"storage_bytes"`
			GPUCount      int   `json:"gpu_count"`
		} `json:"available_capacity"`
	} `json:"resource_utilization"`
	ExecutorSummary []struct {
		ID                string       `json:"id"`
		Name              string       `json:"name"`
		Type              string       `json:"type"`
		ColonyName        string       `json:"colonyname"`
		State             int          `json:"state"`
		CommissionTime    time.Time    `json:"commissiontime"`
		LastHeardFromTime time.Time    `json:"lastheardfromtime"`
		Location          Location     `json:"location"`
		Capabilities      Capabilities `json:"capabilities"`
		CurrentLoad struct {
			RunningProcesses       int     `json:"running_processes"`
			CPUUsagePercent        float64 `json:"cpu_usage_percent"`
			MemoryUsagePercent     float64 `json:"memory_usage_percent"`
			StorageUsagePercent    float64 `json:"storage_usage_percent"`
		} `json:"current_load"`
	} `json:"executor_summary"`
	RecentProcesses []struct {
		ID                 string       `json:"id"`
		State              int          `json:"state"`
		AssignedExecutorID string       `json:"assignedexecutorid"`
		SubmissionTime     time.Time    `json:"submissiontime"`
		StartTime          *time.Time   `json:"starttime,omitempty"`
		EndTime            *time.Time   `json:"endtime,omitempty"`
		FunctionSpec       FunctionSpec `json:"functionspec"`
		ExecutionMetrics struct {
			WaitTimeMs         int   `json:"wait_time_ms"`
			ExecutionTimeMs    int   `json:"execution_time_ms"`
			CPUUsagePercent    float64 `json:"cpu_usage_percent"`
			MemoryUsageBytes   int64 `json:"memory_usage_bytes"`
			NetworkBytesIn     int64 `json:"network_bytes_in"`
			NetworkBytesOut    int64 `json:"network_bytes_out"`
		} `json:"execution_metrics"`
	} `json:"recent_processes"`
	PerformanceTrends struct {
		AvgProcessSuccessRate24h    float64 `json:"avg_process_success_rate_24h"`
		AvgQueueDepth24h            float64 `json:"avg_queue_depth_24h"`
		PeakConcurrentProcesses24h  int     `json:"peak_concurrent_processes_24h"`
		AvgResourceUtilization24h struct {
			CPUPercent     float64 `json:"cpu_percent"`
			MemoryPercent  float64 `json:"memory_percent"`
			StoragePercent float64 `json:"storage_percent"`
			GPUPercent     float64 `json:"gpu_percent"`
		} `json:"avg_resource_utilization_24h"`
		CostMetrics24h struct {
			TotalComputeCostUSD     float64 `json:"total_compute_cost_usd"`
			TotalDataTransferCostUSD float64 `json:"total_data_transfer_cost_usd"`
			CostPerProcessUSD       float64 `json:"cost_per_process_usd"`
		} `json:"cost_metrics_24h"`
	} `json:"performance_trends"`
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(humanConfigPath, metricsDataPath string) *ConfigLoader {
	return &ConfigLoader{
		humanConfigPath: humanConfigPath,
		metricsDataPath: metricsDataPath,
		resourceParser:  &ResourceParser{},
	}
}

// LoadHumanConfig loads the human-provided configuration
func (cl *ConfigLoader) LoadHumanConfig() (*HumanConfig, error) {
	data, err := os.ReadFile(cl.humanConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read human config file: %w", err)
	}

	var config HumanConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse human config JSON: %w", err)
	}

	return &config, nil
}

// LoadColonyOSMetrics loads the ColonyOS metrics and data
func (cl *ConfigLoader) LoadColonyOSMetrics() (*ColonyOSMetrics, error) {
	data, err := os.ReadFile(cl.metricsDataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ColonyOS metrics file: %w", err)
	}

	var metrics ColonyOSMetrics
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ColonyOS metrics JSON: %w", err)
	}

	return &metrics, nil
}

// ConvertToDeploymentConfig converts human config to CAPE DeploymentConfig
func (cl *ConfigLoader) ConvertToDeploymentConfig(humanConfig *HumanConfig) (*models.DeploymentConfig, error) {
	// Map deployment type string to enum
	var deploymentType models.DeploymentType
	switch humanConfig.CapeConfig.DeploymentType {
	case "edge":
		deploymentType = models.DeploymentTypeEdge
	case "cloud":
		deploymentType = models.DeploymentTypeCloud
	case "hpc":
		deploymentType = models.DeploymentTypeHPC
	case "hybrid":
		deploymentType = models.DeploymentTypeHybrid
	case "fog":
		deploymentType = models.DeploymentTypeFog
	default:
		return nil, fmt.Errorf("unknown deployment type: %s", humanConfig.CapeConfig.DeploymentType)
	}

	// Convert optimization goals
	var optimizationGoals []models.OptimizationGoal
	for _, goal := range humanConfig.CapeConfig.OptimizationGoals {
		optimizationGoals = append(optimizationGoals, models.OptimizationGoal{
			Metric:   goal.Metric,
			Weight:   goal.Weight,
			Minimize: goal.Minimize,
		})
	}

	// Convert constraints
	var constraints []models.DeploymentConstraint
	for _, constraint := range humanConfig.CapeConfig.Constraints {
		var constraintType models.ConstraintType
		switch constraint.Type {
		case "sla_deadline":
			constraintType = models.ConstraintTypeSLADeadline
		case "budget_hourly":
			constraintType = models.ConstraintTypeBudgetHourly
		case "data_sovereignty":
			constraintType = models.ConstraintTypeDataSovereignty
		default:
			constraintType = models.ConstraintType(constraint.Type) // Custom constraint
		}

		constraints = append(constraints, models.DeploymentConstraint{
			Type:   constraintType,
			Value:  constraint.Value,
			IsHard: constraint.IsHard,
		})
	}

	return &models.DeploymentConfig{
		DeploymentType:            deploymentType,
		Description:               humanConfig.CapeConfig.Description,
		OptimizationGoals:         optimizationGoals,
		Constraints:              constraints,
		DataGravityFactor:        humanConfig.CapeConfig.LearningParameters.DataGravityFactor,
		ExplorationFactor:        humanConfig.CapeConfig.LearningParameters.ExplorationFactor,
		LearningRate:             humanConfig.CapeConfig.LearningParameters.LearningRate,
		StrategyEnabled:          humanConfig.CapeConfig.LearningParameters.StrategyEnabled,
		AdaptationEnabled:        true,
		Name:                     fmt.Sprintf("CAPE %s Deployment", deploymentType),
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}, nil
}

// ConvertToNativeExecutor converts metrics data executor to native ColonyOS executor
func (cl *ConfigLoader) ConvertToNativeExecutor(execSummary interface{}) (*Executor, error) {
	// Marshal to JSON and back to get proper type conversion
	jsonData, err := json.Marshal(execSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal executor summary: %w", err)
	}

	var executor Executor
	err = json.Unmarshal(jsonData, &executor)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal to native executor: %w", err)
	}

	return &executor, nil
}

// GetSystemState converts ColonyOS metrics to CAPE SystemState
func (cl *ConfigLoader) GetSystemState(metrics *ColonyOSMetrics) (*models.SystemState, error) {
	return &models.SystemState{
		QueueDepth:        metrics.ColonyStatistics.Processes.Waiting,
		QueueThreshold:    20, // From human config or default
		ComputeUsage:      models.Utilization(float64(metrics.ResourceUtilization.AllocatedCapacity.CPUMillicores) / float64(metrics.ResourceUtilization.TotalCapacity.CPUMillicores)),
		MemoryUsage:       models.Utilization(float64(metrics.ResourceUtilization.AllocatedCapacity.MemoryBytes) / float64(metrics.ResourceUtilization.TotalCapacity.MemoryBytes)),
		DiskUsage:         models.Utilization(float64(metrics.ResourceUtilization.AllocatedCapacity.StorageBytes) / float64(metrics.ResourceUtilization.TotalCapacity.StorageBytes)),
		NetworkUsage:      models.Utilization(0.2), // Placeholder - not in current metrics
		MasterUsage:       models.Utilization(metrics.SystemHealth.CPUUsagePercent / 100.0),
		ActiveConnections: metrics.SystemHealth.ActiveConnections,
		Timestamp:         metrics.Timestamp,
		TimeSlot:          metrics.Timestamp.Hour(),
		DayOfWeek:         int(metrics.Timestamp.Weekday()),
	}, nil
}

// GetActiveExecutors converts metrics executor summaries to native executors
func (cl *ConfigLoader) GetActiveExecutors(metrics *ColonyOSMetrics) ([]Executor, error) {
	var executors []Executor
	
	for _, execSummary := range metrics.ExecutorSummary {
		executor, err := cl.ConvertToNativeExecutor(execSummary)
		if err != nil {
			return nil, fmt.Errorf("failed to convert executor %s: %w", execSummary.Name, err)
		}
		executors = append(executors, *executor)
	}
	
	return executors, nil
}

// GetProcessQueue converts recent processes to native processes
func (cl *ConfigLoader) GetProcessQueue(metrics *ColonyOSMetrics) ([]Process, error) {
	var processes []Process
	
	for _, procSummary := range metrics.RecentProcesses {
		process := Process{
			ID:                 procSummary.ID,
			State:              procSummary.State,
			AssignedExecutorID: procSummary.AssignedExecutorID,
			SubmissionTime:     procSummary.SubmissionTime,
			FunctionSpec:       procSummary.FunctionSpec,
		}
		
		if procSummary.StartTime != nil {
			process.StartTime = *procSummary.StartTime
		}
		if procSummary.EndTime != nil {
			process.EndTime = *procSummary.EndTime
		}
		
		processes = append(processes, process)
	}
	
	return processes, nil
}

// ValidateConfig validates the human configuration
func (cl *ConfigLoader) ValidateConfig(config *HumanConfig) error {
	// Validate optimization goals weights sum to 1.0
	totalWeight := 0.0
	for _, goal := range config.CapeConfig.OptimizationGoals {
		totalWeight += goal.Weight
	}
	
	if totalWeight < 0.95 || totalWeight > 1.05 {
		return fmt.Errorf("optimization goals weights sum to %.3f, should be close to 1.0", totalWeight)
	}
	
	// Validate learning parameters
	if config.CapeConfig.LearningParameters.DataGravityFactor < 0 || config.CapeConfig.LearningParameters.DataGravityFactor > 1 {
		return fmt.Errorf("data_gravity_factor must be between 0 and 1, got %.3f", config.CapeConfig.LearningParameters.DataGravityFactor)
	}
	
	// Validate orchestrator behavior parameters
	if config.OrchestratorConfig.Behavior.MaxConcurrentProcesses <= 0 {
		return fmt.Errorf("max_concurrent_processes must be positive, got %d", config.OrchestratorConfig.Behavior.MaxConcurrentProcesses)
	}
	
	return nil
}

// ValidateMetrics validates the ColonyOS metrics data
func (cl *ConfigLoader) ValidateMetrics(metrics *ColonyOSMetrics) error {
	// Check if metrics are recent (within last hour)
	age := time.Since(metrics.Timestamp)
	if age > time.Hour {
		return fmt.Errorf("metrics data is %v old, may be stale", age)
	}
	
	// Validate resource utilization makes sense
	if metrics.ResourceUtilization.AllocatedCapacity.CPUMillicores > metrics.ResourceUtilization.TotalCapacity.CPUMillicores {
		return fmt.Errorf("allocated CPU exceeds total CPU capacity")
	}
	
	if metrics.ResourceUtilization.AllocatedCapacity.MemoryBytes > metrics.ResourceUtilization.TotalCapacity.MemoryBytes {
		return fmt.Errorf("allocated memory exceeds total memory capacity")
	}
	
	return nil
}

// RefreshMetrics reloads the ColonyOS metrics from file
func (cl *ConfigLoader) RefreshMetrics() (*ColonyOSMetrics, error) {
	return cl.LoadColonyOSMetrics()
}