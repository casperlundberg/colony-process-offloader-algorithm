package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	
	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/api"
	"github.com/casperlundberg/colony-process-offloader-algorithm/internal/database"
)

func main() {
	var (
		dbPath = flag.String("db", "analytics.db", "Path to SQLite database file")
		port   = flag.String("port", "8080", "Port to run API server on")
	)
	flag.Parse()
	
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
	
	// Create repository
	repo := database.NewRepository(db)
	
	// Create and start API server
	log.Printf("Starting analytics API server on port %s", *port)
	server := api.NewServer(repo, *port)
	
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}