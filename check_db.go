package main

import (
	"fmt"
	"log"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/database"
)

func main() {
	// Connect to database
	db, err := database.NewDatabase("analytics.db")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	// Create repository
	repo := database.NewRepository(db)
	
	// List all simulations
	simulations, err := repo.ListSimulations()
	if err != nil {
		log.Fatalf("Failed to list simulations: %v", err)
	}
	
	fmt.Printf("Found %d simulations in database:\n\n", len(simulations))
	
	for _, sim := range simulations {
		fmt.Printf("ID: %s\n", sim.ID)
		fmt.Printf("Name: %s\n", sim.Name)
		fmt.Printf("Description: %s\n", sim.Description)
		fmt.Printf("Status: %s\n", sim.Status)
		fmt.Printf("Start Time: %s\n", sim.StartTime.Format("2006-01-02 15:04:05"))
		if sim.EndTime != nil {
			fmt.Printf("End Time: %s\n", sim.EndTime.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("Created: %s\n", sim.CreatedAt.Format("2006-01-02 15:04:05"))
		
		// Get metrics count
		metrics, err := repo.GetMetricSnapshots(sim.ID, 0)
		if err == nil {
			fmt.Printf("Metrics Points: %d\n", len(metrics))
		}
		
		// Get decisions count
		decisions, err := repo.GetScalingDecisions(sim.ID)
		if err == nil {
			fmt.Printf("Scaling Decisions: %d\n", len(decisions))
		}
		
		fmt.Println("---")
	}
}