package database

import (
	"fmt"
	"time"
	
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB holds the database connection
type DB struct {
	*gorm.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string) (*DB, error) {
	// Use SQLite for simplicity and portability
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	
	// Auto-migrate schemas
	err = db.AutoMigrate(
		&Simulation{},
		&MetricSnapshot{},
		&ScalingDecision{},
		&Event{},
		&PredictionAccuracy{},
		&LearningMetrics{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	
	return &DB{db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}