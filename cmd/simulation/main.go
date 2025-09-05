package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/database"
	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/simulation"
)

func main() {
	var (
		configPath      = flag.String("config", "configs/simulation_config.json", "Path to simulation config")
		catalogPath     = flag.String("catalog", "configs/executor_catalog.json", "Path to executor catalog")
		autoscalerPath  = flag.String("autoscaler", "configs/autoscaler_config.json", "Path to autoscaler config")
		dbPath         = flag.String("db", "analytics.db", "Path to SQLite database file")
		simName        = flag.String("name", "Cape Simulation", "Simulation name")
		simDescription = flag.String("description", "CAPE autoscaler spike simulation", "Simulation description")
	)
	flag.Parse()
	
	log.Printf("Starting CAPE simulation with database analytics")
	
	// Ensure database directory exists
	dbDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}
	
	// Initialize database
	log.Printf("Connecting to database at %s", *dbPath)
	db, err := database.NewDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	
	// Create repository and metrics collector
	repo := database.NewRepository(db)
	
	// Create database collector for this simulation run
	dbCollector, err := simulation.NewDBMetricsCollector(repo, *simName, *simDescription)
	if err != nil {
		log.Fatalf("Failed to create database collector: %v", err)
	}
	
	log.Printf("Created simulation with ID: %s", dbCollector.GetSimulationID())
	
	// Create simulation runner with database collection
	runner, err := simulation.NewSimulationRunner(*configPath, *catalogPath, *autoscalerPath, dbCollector)
	if err != nil {
		log.Fatalf("Failed to create simulation runner: %v", err)
	}
	
	log.Printf("Starting simulation at %s", time.Now().Format(time.RFC3339))
	
	// Run simulation
	start := time.Now()
	if err := runner.Run(); err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}
	
	duration := time.Since(start)
	log.Printf("Simulation completed in %v", duration)
	log.Printf("Results stored in database. Simulation ID: %s", dbCollector.GetSimulationID())
	log.Printf("Start analytics server to view results: ./analytics-server -db %s", *dbPath)
}