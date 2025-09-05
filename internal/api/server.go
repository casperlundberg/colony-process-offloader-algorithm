package api

import (
	"fmt"
	"net/http"
	"time"
	
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/database"
)

// Server represents the API server
type Server struct {
	router *gin.Engine
	repo   *database.Repository
	port   string
}

// NewServer creates a new API server
func NewServer(repo *database.Repository, port string) *Server {
	router := gin.Default()
	
	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	router.Use(cors.New(config))
	
	server := &Server{
		router: router,
		repo:   repo,
		port:   port,
	}
	
	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Serve static files
	s.router.Static("/static", "./web")
	s.router.StaticFile("/", "./web/index.html")
	
	api := s.router.Group("/api/v1")
	
	// Simulation endpoints
	api.GET("/simulations", s.listSimulations)
	api.GET("/simulations/:id", s.getSimulation)
	api.POST("/simulations", s.createSimulation)
	api.PUT("/simulations/:id", s.updateSimulation)
	api.DELETE("/simulations/:id", s.deleteSimulation)
	
	// Metrics endpoints (isolated by simulation)
	api.GET("/simulations/:id/metrics", s.getMetrics)
	api.GET("/simulations/:id/metrics/latest", s.getLatestMetrics)
	api.GET("/simulations/:id/metrics/range", s.getMetricsInRange)
	
	// Scaling decisions endpoints
	api.GET("/simulations/:id/decisions", s.getScalingDecisions)
	
	// Events endpoints
	api.GET("/simulations/:id/events", s.getEvents)
	
	// Prediction accuracy endpoints
	api.GET("/simulations/:id/predictions", s.getPredictionAccuracy)
	
	// Learning metrics endpoints
	api.GET("/simulations/:id/learning", s.getLearningMetrics)
	
	// Summary endpoint
	api.GET("/simulations/:id/summary", s.getSimulationSummary)
	
	// Health check
	api.GET("/health", s.healthCheck)
}

// Start starts the server
func (s *Server) Start() error {
	return s.router.Run(":" + s.port)
}

// Handler implementations

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now(),
	})
}

func (s *Server) listSimulations(c *gin.Context) {
	simulations, err := s.repo.ListSimulations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, simulations)
}

func (s *Server) getSimulation(c *gin.Context) {
	id := c.Param("id")
	
	simulation, err := s.repo.GetSimulation(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Simulation not found"})
		return
	}
	
	c.JSON(http.StatusOK, simulation)
}

func (s *Server) createSimulation(c *gin.Context) {
	var sim database.Simulation
	if err := c.ShouldBindJSON(&sim); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Set timestamps
	sim.CreatedAt = time.Now()
	sim.UpdatedAt = time.Now()
	sim.StartTime = time.Now()
	sim.Status = "running"
	
	if err := s.repo.CreateSimulation(&sim); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, sim)
}

func (s *Server) updateSimulation(c *gin.Context) {
	id := c.Param("id")
	
	var sim database.Simulation
	if err := c.ShouldBindJSON(&sim); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	sim.ID = id
	sim.UpdatedAt = time.Now()
	
	if err := s.repo.UpdateSimulation(&sim); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, sim)
}

func (s *Server) deleteSimulation(c *gin.Context) {
	id := c.Param("id")
	
	if err := s.repo.DeleteSimulation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Simulation deleted"})
}

func (s *Server) getMetrics(c *gin.Context) {
	simulationID := c.Param("id")
	
	// Parse query parameters
	limit := 1000 // Default limit
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	
	metrics, err := s.repo.GetMetricSnapshots(simulationID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getLatestMetrics(c *gin.Context) {
	simulationID := c.Param("id")
	
	metric, err := s.repo.GetLatestMetricSnapshot(simulationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No metrics found"})
		return
	}
	
	c.JSON(http.StatusOK, metric)
}

func (s *Server) getMetricsInRange(c *gin.Context) {
	simulationID := c.Param("id")
	
	// Parse time range from query parameters
	startStr := c.Query("start")
	endStr := c.Query("end")
	
	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start and end times required"})
		return
	}
	
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time"})
		return
	}
	
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time"})
		return
	}
	
	metrics, err := s.repo.GetMetricSnapshotsInRange(simulationID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getScalingDecisions(c *gin.Context) {
	simulationID := c.Param("id")
	
	decisions, err := s.repo.GetScalingDecisions(simulationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, decisions)
}

func (s *Server) getEvents(c *gin.Context) {
	simulationID := c.Param("id")
	eventType := c.Query("type")
	
	events, err := s.repo.GetEvents(simulationID, eventType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, events)
}

func (s *Server) getPredictionAccuracy(c *gin.Context) {
	simulationID := c.Param("id")
	
	accuracy, err := s.repo.GetPredictionAccuracy(simulationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, accuracy)
}

func (s *Server) getLearningMetrics(c *gin.Context) {
	simulationID := c.Param("id")
	algorithm := c.Query("algorithm")
	
	metrics, err := s.repo.GetLearningMetrics(simulationID, algorithm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

func (s *Server) getSimulationSummary(c *gin.Context) {
	simulationID := c.Param("id")
	
	summary, err := s.repo.GetSimulationSummary(simulationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, summary)
}