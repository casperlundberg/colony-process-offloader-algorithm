// Analytics Dashboard JavaScript
const API_BASE = '/api/v1';

let currentSimulation = null;
let currentMetrics = [];
let charts = {};

// Initialize the dashboard
document.addEventListener('DOMContentLoaded', function() {
    refreshSimulations();
});

// Utility functions
function formatNumber(num) {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function formatDuration(seconds) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
}

// API functions
async function apiRequest(endpoint) {
    try {
        const response = await fetch(`${API_BASE}${endpoint}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return await response.json();
    } catch (error) {
        console.error('API request failed:', error);
        showError(`API request failed: ${error.message}`);
        throw error;
    }
}

// Simulation management
async function refreshSimulations() {
    try {
        const simulations = await apiRequest('/simulations');
        const select = document.getElementById('simulationSelect');
        
        // Clear existing options except the first one
        select.innerHTML = '<option value="">Select a simulation...</option>';
        
        simulations.forEach(sim => {
            const option = document.createElement('option');
            option.value = sim.id;
            option.textContent = `${sim.name} (${new Date(sim.start_time).toLocaleString()})`;
            option.dataset.status = sim.status;
            select.appendChild(option);
        });
        
        console.log(`Loaded ${simulations.length} simulations`);
    } catch (error) {
        console.error('Failed to refresh simulations:', error);
    }
}

async function loadSimulation() {
    const select = document.getElementById('simulationSelect');
    const simulationId = select.value;
    
    if (!simulationId) {
        showError('Please select a simulation');
        return;
    }
    
    showLoading('Loading simulation data...');
    
    try {
        // Load simulation details and metrics
        const [simulation, metrics] = await Promise.all([
            apiRequest(`/simulations/${simulationId}`),
            apiRequest(`/simulations/${simulationId}/metrics?limit=10000`)
        ]);
        
        currentSimulation = simulation;
        currentMetrics = metrics.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
        
        // Update status
        const statusElement = document.getElementById('simulationStatus');
        statusElement.textContent = simulation.status;
        statusElement.className = `status ${simulation.status}`;
        
        // Set up time range controls
        if (currentMetrics.length > 0) {
            const startTime = new Date(currentMetrics[0].timestamp);
            const endTime = new Date(currentMetrics[currentMetrics.length - 1].timestamp);
            
            document.getElementById('startTime').value = startTime.toISOString().slice(0, 16);
            document.getElementById('endTime').value = endTime.toISOString().slice(0, 16);
        }
        
        // Load additional data
        await Promise.all([
            loadScalingDecisions(simulationId),
            loadPredictionAccuracy(simulationId)
        ]);
        
        // Show analytics
        showAnalytics();
        
        // Enable management buttons
        enableManagementButtons();
        
    } catch (error) {
        showError(`Failed to load simulation: ${error.message}`);
        disableManagementButtons();
    }
}

async function loadScalingDecisions(simulationId) {
    try {
        const decisions = await apiRequest(`/simulations/${simulationId}/decisions`);
        currentSimulation.scalingDecisions = decisions;
        console.log(`Loaded ${decisions.length} scaling decisions`);
    } catch (error) {
        console.warn('Failed to load scaling decisions:', error);
        currentSimulation.scalingDecisions = [];
    }
}

async function loadPredictionAccuracy(simulationId) {
    try {
        const accuracy = await apiRequest(`/simulations/${simulationId}/predictions`);
        currentSimulation.predictionAccuracy = accuracy;
        console.log(`Loaded ${accuracy.length} prediction accuracy records`);
    } catch (error) {
        console.warn('Failed to load prediction accuracy:', error);
        currentSimulation.predictionAccuracy = [];
    }
}

// UI functions
function showLoading(message) {
    document.getElementById('loadingMessage').textContent = message;
    document.getElementById('loadingMessage').style.display = 'block';
    document.getElementById('analyticsContainer').style.display = 'none';
    document.getElementById('errorMessage').style.display = 'none';
}

function showError(message) {
    document.getElementById('errorMessage').textContent = message;
    document.getElementById('errorMessage').style.display = 'block';
    document.getElementById('loadingMessage').style.display = 'none';
}

function showAnalytics() {
    document.getElementById('loadingMessage').style.display = 'none';
    document.getElementById('errorMessage').style.display = 'none';
    document.getElementById('analyticsContainer').style.display = 'block';
    
    updateMetricsSummary();
    createCharts();
}

function updateMetricsSummary() {
    if (!currentMetrics.length) return;
    
    // Calculate summary statistics
    const latest = currentMetrics[currentMetrics.length - 1];
    const avgQueueDepth = currentMetrics.reduce((sum, m) => sum + m.queue_depth, 0) / currentMetrics.length;
    const maxQueueDepth = Math.max(...currentMetrics.map(m => m.queue_depth));
    const avgExecutors = currentMetrics.reduce((sum, m) => sum + m.actual_executors, 0) / currentMetrics.length;
    const maxExecutors = Math.max(...currentMetrics.map(m => m.actual_executors));
    
    const scalingDecisions = currentSimulation.scalingDecisions?.length || 0;
    const scaleUpCount = currentSimulation.scalingDecisions?.filter(d => d.decision_type === 'scale_up').length || 0;
    
    const summaryHTML = `
        <div class="metric-card">
            <div class="metric-value">${latest.queue_depth}</div>
            <div class="metric-label">Current Queue Depth</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">${latest.actual_executors}</div>
            <div class="metric-label">Current Executors</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">${avgQueueDepth.toFixed(1)}</div>
            <div class="metric-label">Avg Queue Depth</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">${maxQueueDepth}</div>
            <div class="metric-label">Max Queue Depth</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">${avgExecutors.toFixed(1)}</div>
            <div class="metric-label">Avg Executors</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">${scalingDecisions}</div>
            <div class="metric-label">Scaling Decisions</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">${(latest.system_load * 100).toFixed(1)}%</div>
            <div class="metric-label">System Load</div>
        </div>
        <div class="metric-card">
            <div class="metric-value">${(latest.queue_pressure * 100).toFixed(1)}%</div>
            <div class="metric-label">Queue Pressure</div>
        </div>
    `;
    
    document.getElementById('metricsSummary').innerHTML = summaryHTML;
}

// Chart creation functions
function createCharts() {
    destroyExistingCharts();
    
    const timestamps = currentMetrics.map(m => new Date(m.timestamp));
    
    // Queue Depth Chart
    createQueueChart(timestamps);
    
    // Executor Count Chart
    createExecutorChart(timestamps);
    
    // Queue Dynamics Chart
    createQueueDynamicsChart(timestamps);
    
    // System Load Chart
    createSystemLoadChart(timestamps);
    
    // Scaling Decisions Chart
    createScalingChart(timestamps);
    
    // Prediction Accuracy Chart
    createPredictionChart();
}

function destroyExistingCharts() {
    Object.values(charts).forEach(chart => {
        if (chart && chart.destroy) {
            chart.destroy();
        }
    });
    charts = {};
}

function createQueueChart(timestamps) {
    const ctx = document.getElementById('queueChart').getContext('2d');
    
    charts.queue = new Chart(ctx, {
        type: 'line',
        data: {
            labels: timestamps,
            datasets: [{
                label: 'Queue Depth',
                data: currentMetrics.map(m => m.queue_depth),
                borderColor: 'rgb(75, 192, 192)',
                backgroundColor: 'rgba(75, 192, 192, 0.2)',
                tension: 0.1,
                fill: true
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'time',
                    time: {
                        displayFormats: {
                            hour: 'HH:mm',
                            minute: 'HH:mm'
                        }
                    }
                },
                y: {
                    beginAtZero: true
                }
            },
            plugins: {
                legend: {
                    display: false
                }
            }
        }
    });
}

function createExecutorChart(timestamps) {
    const ctx = document.getElementById('executorChart').getContext('2d');
    
    charts.executor = new Chart(ctx, {
        type: 'line',
        data: {
            labels: timestamps,
            datasets: [{
                label: 'Actual Executors',
                data: currentMetrics.map(m => m.actual_executors),
                borderColor: 'rgb(54, 162, 235)',
                backgroundColor: 'rgba(54, 162, 235, 0.2)',
                tension: 0.1
            }, {
                label: 'Planned Executors',
                data: currentMetrics.map(m => m.planned_executors),
                borderColor: 'rgb(255, 99, 132)',
                backgroundColor: 'rgba(255, 99, 132, 0.2)',
                tension: 0.1,
                borderDash: [5, 5]
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'time',
                    time: {
                        displayFormats: {
                            hour: 'HH:mm',
                            minute: 'HH:mm'
                        }
                    }
                },
                y: {
                    beginAtZero: true
                }
            }
        }
    });
}

function createQueueDynamicsChart(timestamps) {
    const ctx = document.getElementById('queueDynamicsChart').getContext('2d');
    
    charts.dynamics = new Chart(ctx, {
        type: 'line',
        data: {
            labels: timestamps,
            datasets: [{
                label: 'Velocity (items/sec)',
                data: currentMetrics.map(m => m.queue_velocity),
                borderColor: 'rgb(255, 159, 64)',
                backgroundColor: 'rgba(255, 159, 64, 0.2)',
                yAxisID: 'y'
            }, {
                label: 'Acceleration (items/sec²)',
                data: currentMetrics.map(m => m.queue_acceleration),
                borderColor: 'rgb(153, 102, 255)',
                backgroundColor: 'rgba(153, 102, 255, 0.2)',
                yAxisID: 'y1'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'time',
                    time: {
                        displayFormats: {
                            hour: 'HH:mm',
                            minute: 'HH:mm'
                        }
                    }
                },
                y: {
                    type: 'linear',
                    display: true,
                    position: 'left',
                },
                y1: {
                    type: 'linear',
                    display: true,
                    position: 'right',
                    grid: {
                        drawOnChartArea: false,
                    },
                }
            }
        }
    });
}

function createSystemLoadChart(timestamps) {
    const ctx = document.getElementById('systemLoadChart').getContext('2d');
    
    charts.systemLoad = new Chart(ctx, {
        type: 'line',
        data: {
            labels: timestamps,
            datasets: [{
                label: 'CPU Usage',
                data: currentMetrics.map(m => m.compute_usage * 100),
                borderColor: 'rgb(255, 99, 132)',
                backgroundColor: 'rgba(255, 99, 132, 0.1)'
            }, {
                label: 'Memory Usage',
                data: currentMetrics.map(m => m.memory_usage * 100),
                borderColor: 'rgb(54, 162, 235)',
                backgroundColor: 'rgba(54, 162, 235, 0.1)'
            }, {
                label: 'System Load',
                data: currentMetrics.map(m => m.system_load * 100),
                borderColor: 'rgb(75, 192, 192)',
                backgroundColor: 'rgba(75, 192, 192, 0.1)'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'time',
                    time: {
                        displayFormats: {
                            hour: 'HH:mm',
                            minute: 'HH:mm'
                        }
                    }
                },
                y: {
                    beginAtZero: true,
                    max: 100,
                    ticks: {
                        callback: function(value) {
                            return value + '%';
                        }
                    }
                }
            }
        }
    });
}

function createScalingChart(timestamps) {
    const ctx = document.getElementById('scalingChart').getContext('2d');
    
    // Create scatter plot of scaling decisions
    const scalingData = (currentSimulation.scalingDecisions || []).map(decision => ({
        x: new Date(decision.timestamp),
        y: decision.to_count,
        type: decision.decision_type,
        delta: decision.delta
    }));
    
    const scaleUpData = scalingData.filter(d => d.type === 'scale_up');
    const scaleDownData = scalingData.filter(d => d.type === 'scale_down');
    
    charts.scaling = new Chart(ctx, {
        type: 'scatter',
        data: {
            datasets: [{
                label: 'Scale Up',
                data: scaleUpData.map(d => ({ x: d.x, y: d.y })),
                backgroundColor: 'rgba(75, 192, 192, 0.6)',
                borderColor: 'rgb(75, 192, 192)',
                pointRadius: 6
            }, {
                label: 'Scale Down',
                data: scaleDownData.map(d => ({ x: d.x, y: d.y })),
                backgroundColor: 'rgba(255, 99, 132, 0.6)',
                borderColor: 'rgb(255, 99, 132)',
                pointRadius: 6
            }, {
                label: 'Actual Executors',
                data: currentMetrics.map((m, i) => ({ x: timestamps[i], y: m.actual_executors })),
                type: 'line',
                borderColor: 'rgba(54, 162, 235, 0.3)',
                backgroundColor: 'rgba(54, 162, 235, 0.1)',
                pointRadius: 0,
                tension: 0.1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'time',
                    time: {
                        displayFormats: {
                            hour: 'HH:mm',
                            minute: 'HH:mm'
                        }
                    }
                },
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Executor Count'
                    }
                }
            }
        }
    });
}

function createPredictionChart() {
    const ctx = document.getElementById('predictionChart').getContext('2d');
    
    const accuracy = currentSimulation.predictionAccuracy || [];
    
    if (accuracy.length === 0) {
        ctx.font = '16px Arial';
        ctx.fillStyle = '#666';
        ctx.textAlign = 'center';
        ctx.fillText('No prediction data available', ctx.canvas.width / 2, ctx.canvas.height / 2);
        return;
    }
    
    const timestamps = accuracy.map(a => new Date(a.timestamp));
    
    charts.prediction = new Chart(ctx, {
        type: 'line',
        data: {
            labels: timestamps,
            datasets: [{
                label: 'Queue Depth Error',
                data: accuracy.map(a => a.queue_depth_error),
                borderColor: 'rgb(255, 99, 132)',
                backgroundColor: 'rgba(255, 99, 132, 0.2)'
            }, {
                label: 'Load Error',
                data: accuracy.map(a => a.load_error * 100),
                borderColor: 'rgb(54, 162, 235)',
                backgroundColor: 'rgba(54, 162, 235, 0.2)'
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'time',
                    time: {
                        displayFormats: {
                            hour: 'HH:mm',
                            minute: 'HH:mm'
                        }
                    }
                },
                y: {
                    beginAtZero: true
                }
            }
        }
    });
}

// Time range filtering
function updateTimeRange() {
    const startTime = new Date(document.getElementById('startTime').value);
    const endTime = new Date(document.getElementById('endTime').value);
    
    if (startTime >= endTime) {
        showError('Start time must be before end time');
        return;
    }
    
    // Filter metrics by time range
    const filteredMetrics = currentMetrics.filter(m => {
        const timestamp = new Date(m.timestamp);
        return timestamp >= startTime && timestamp <= endTime;
    });
    
    if (filteredMetrics.length === 0) {
        showError('No data found in selected time range');
        return;
    }
    
    // Update current metrics and recreate charts
    const originalMetrics = [...currentMetrics];
    currentMetrics = filteredMetrics;
    
    try {
        updateMetricsSummary();
        createCharts();
    } catch (error) {
        console.error('Failed to update time range:', error);
        currentMetrics = originalMetrics;
        showError('Failed to apply time range filter');
    }
}

function resetTimeRange() {
    if (!currentSimulation) return;
    
    // Reload all metrics
    loadSimulation();
}

// CRUD Operations for Simulation Management

function enableManagementButtons() {
    document.getElementById('editBtn').disabled = false;
    document.getElementById('cloneBtn').disabled = false;
    document.getElementById('deleteBtn').disabled = false;
}

function disableManagementButtons() {
    document.getElementById('editBtn').disabled = true;
    document.getElementById('cloneBtn').disabled = true;
    document.getElementById('deleteBtn').disabled = true;
}

// Edit simulation
function showEditDialog() {
    if (!currentSimulation) return;
    
    document.getElementById('editName').value = currentSimulation.name;
    document.getElementById('editDescription').value = currentSimulation.description;
    document.getElementById('editModal').style.display = 'block';
}

function hideEditDialog() {
    document.getElementById('editModal').style.display = 'none';
}

document.getElementById('editForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const name = document.getElementById('editName').value;
    const description = document.getElementById('editDescription').value;
    
    try {
        const response = await fetch(`${API_BASE}/simulations/${currentSimulation.id}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ name, description })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        hideEditDialog();
        await refreshSimulations();
        showSuccess('Simulation updated successfully');
        
        // Update current simulation object
        currentSimulation.name = name;
        currentSimulation.description = description;
        
    } catch (error) {
        showError(`Failed to update simulation: ${error.message}`);
    }
});

// Clone simulation
function showCloneDialog() {
    if (!currentSimulation) return;
    
    document.getElementById('cloneName').value = currentSimulation.name + ' (Copy)';
    document.getElementById('cloneDescription').value = currentSimulation.description + ' - Cloned from original';
    document.getElementById('cloneModal').style.display = 'block';
}

function hideCloneDialog() {
    document.getElementById('cloneModal').style.display = 'none';
}

document.getElementById('cloneForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const name = document.getElementById('cloneName').value;
    const description = document.getElementById('cloneDescription').value;
    
    try {
        const response = await fetch(`${API_BASE}/simulations/${currentSimulation.id}/clone`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ name, description })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const result = await response.json();
        hideCloneDialog();
        await refreshSimulations();
        showSuccess(`Simulation cloned successfully! New ID: ${result.simulation_id}`);
        
    } catch (error) {
        showError(`Failed to clone simulation: ${error.message}`);
    }
});

// Delete simulation
function deleteSimulation() {
    if (!currentSimulation) return;
    
    if (!confirm(`Are you sure you want to delete simulation "${currentSimulation.name}"? This action cannot be undone and will delete all associated data.`)) {
        return;
    }
    
    deleteSimulationConfirmed();
}

async function deleteSimulationConfirmed() {
    try {
        const response = await fetch(`${API_BASE}/simulations/${currentSimulation.id}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        showSuccess('Simulation deleted successfully');
        
        // Reset UI
        currentSimulation = null;
        currentMetrics = [];
        disableManagementButtons();
        document.getElementById('analyticsContainer').style.display = 'none';
        document.getElementById('loadingMessage').style.display = 'block';
        document.getElementById('loadingMessage').textContent = 'Select a simulation to view analytics data';
        
        await refreshSimulations();
        
    } catch (error) {
        showError(`Failed to delete simulation: ${error.message}`);
    }
}

// Success message helper
function showSuccess(message) {
    // Create a temporary success message element
    const successDiv = document.createElement('div');
    successDiv.style.cssText = 'position: fixed; top: 20px; right: 20px; background: #d4edda; color: #155724; padding: 10px 15px; border-radius: 4px; z-index: 1001; border: 1px solid #c3e6cb;';
    successDiv.textContent = message;
    
    document.body.appendChild(successDiv);
    
    setTimeout(() => {
        document.body.removeChild(successDiv);
    }, 3000);
}