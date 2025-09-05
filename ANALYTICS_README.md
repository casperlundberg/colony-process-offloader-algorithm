# CAPE Autoscaler Analytics System

A comprehensive web-based analytics system for monitoring and analyzing CAPE (Colony Adaptive Process Engine) autoscaler performance.

## Overview

This system provides real-time visualization and historical analysis of:
- Queue depth and dynamics (velocity, acceleration)  
- Executor scaling decisions and performance
- System resource utilization
- Algorithm prediction accuracy
- Scaling decision effectiveness

## Quick Start

### 1. Build the components

```bash
# Build the simulation runner with database support
go build -o ./bin/simulation ./cmd/simulation

# Build the analytics web server  
go build -o ./bin/analytics-server ./cmd/analytics-server
```

### 2. Run a simulation with database collection

```bash
./bin/simulation \
  -config configs/simulation_config.json \
  -catalog configs/executor_catalog.json \
  -autoscaler configs/autoscaler_config.json \
  -db analytics.db \
  -name "CAPE Performance Test" \
  -description "Testing queue acceleration with EWMA smoothing"
```

### 3. Start the analytics server

```bash
./bin/analytics-server -db analytics.db -port 8080
```

### 4. View analytics

Open your browser to: http://localhost:8080

## Features

### Dashboard Components

**Metrics Summary Cards**
- Current queue depth and executor count
- Average and maximum values
- System load and queue pressure percentages

**Queue Depth Over Time** 
- Real-time queue depth visualization
- Shows queue buildup and clearing patterns
- Filled area chart with timestamps

**Executor Count Over Time**
- Actual vs planned executor comparison  
- Shows scaling responsiveness
- Dotted line for planned executors

**Queue Dynamics**
- Velocity (items/second) and acceleration (items/second²)
- Dual Y-axis visualization
- Early warning indicators for queue growth

**System Load**
- CPU, memory, and overall system utilization
- Percentage-based visualization
- Multiple resource tracking

**Scaling Decisions Timeline**
- Scatter plot of scale up/down events
- Overlaid with actual executor count
- Decision timing and effectiveness

**Prediction Accuracy** 
- Queue depth and load prediction errors
- Algorithm performance over time
- Helps tune prediction models

### API Endpoints

All simulation data is isolated by simulation ID:

- `GET /api/v1/simulations` - List all simulations
- `GET /api/v1/simulations/:id` - Get simulation details  
- `GET /api/v1/simulations/:id/metrics` - Get metrics data
- `GET /api/v1/simulations/:id/decisions` - Get scaling decisions
- `GET /api/v1/simulations/:id/events` - Get simulation events
- `GET /api/v1/simulations/:id/predictions` - Get prediction accuracy
- `GET /api/v1/simulations/:id/summary` - Get aggregated statistics

### Database Schema

**Core Tables:**
- `simulations` - Simulation metadata and status
- `metric_snapshots` - Time-series metrics data  
- `scaling_decisions` - Autoscaler decisions and outcomes
- `events` - Simulation events and errors
- `prediction_accuracy` - Algorithm prediction performance
- `learning_metrics` - ML algorithm statistics

## Usage Examples

### Running Different Scenarios

```bash
# Test spike handling
./bin/simulation -name "Spike Test" -description "High-intensity spike scenario"

# Test baseline performance  
./bin/simulation -name "Baseline Test" -description "Normal load patterns"

# Test with different configurations
./bin/simulation -config configs/aggressive_config.json -name "Aggressive Scaling"
```

### Analyzing Results

1. **Queue Performance**: Look for patterns in queue depth and velocity
2. **Scaling Responsiveness**: Compare planned vs actual executor deployment
3. **Resource Efficiency**: Monitor system utilization during scaling events
4. **Prediction Quality**: Track prediction accuracy to tune algorithms
5. **Decision Effectiveness**: Analyze scaling decisions and their outcomes

### Time Range Filtering

Use the time range controls in the dashboard to:
- Focus on specific events or time periods
- Compare before/after scaling decisions
- Analyze algorithm behavior during spikes
- Export data for specific timeframes

## Architecture

**Database**: SQLite for portability and simplicity
**Backend**: Go with Gin web framework
**Frontend**: Vanilla JavaScript with Chart.js
**Storage**: Isolated per simulation for clean analysis

**Key Benefits:**
- Each simulation run is completely isolated
- Historical data preserved for comparison
- Real-time visualization during active simulations  
- RESTful API for integration with other tools
- Lightweight and portable (single SQLite file)

## Development

### Adding New Metrics

1. Update `database/models.go` with new fields
2. Modify `simulation/db_collector.go` collection methods
3. Update API endpoints in `api/server.go`
4. Add visualization in `web/analytics.js`

### Customizing Charts

Charts are built with Chart.js and can be customized by modifying the chart creation functions in `analytics.js`. All chart configurations support:
- Custom colors and styling
- Different chart types (line, bar, scatter)
- Multi-axis displays
- Time-based X-axes
- Interactive tooltips and legends

## Troubleshooting

**Database Issues**: Check file permissions on the SQLite database
**API Errors**: Verify the analytics server is running on the correct port  
**No Data**: Ensure simulations are using the database collector
**Charts Not Loading**: Check browser console for JavaScript errors
**CORS Issues**: The server is configured for localhost:3000 and localhost:8080

## Performance Considerations

- Database grows with simulation length and frequency
- Use time range filtering for large datasets
- Consider periodic cleanup of old simulation data
- SQLite performs well for typical simulation workloads
- Charts render efficiently up to ~10,000 data points