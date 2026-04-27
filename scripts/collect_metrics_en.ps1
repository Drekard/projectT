#!/usr/bin/env pwsh

# Script for collecting and saving Prometheus metrics

param (
    [string]$OutputFile = "metrics_$(Get-Date -Format 'yyyy-MM-dd_HH-mm-ss').txt",
    [int]$Port = 9090,
    [string]$ServerHost = "localhost",
    [switch]$SummaryOnly
)

function Get-Metrics {
    param (
        [string]$Url
    )
    
    try {
        Write-Host "Collecting metrics from $Url..." -ForegroundColor Green
        $response = Invoke-WebRequest -Uri $Url -ErrorAction Stop
        return $response.Content
    }
    catch {
        Write-Host "Error collecting metrics: $_" -ForegroundColor Red
        return $null
    }
}

function Extract-Summary {
    param (
        [string]$MetricsContent
    )
    
    $summary = @()
    $summary += "=== METRICS SUMMARY ==="
    $summary += "Collection time: $(Get-Date)"
    
    # Process metrics
    $summary += "`n=== PROCESS METRICS ==="
    if ($MetricsContent -match 'process_cpu_seconds_total\s+(\d+\.?\d*)') {
        $summary += "CPU time: $($matches[1]) seconds"
    }
    if ($MetricsContent -match 'process_open_fds\s+(\d+)') {
        $summary += "Open file descriptors: $($matches[1])"
    }
    if ($MetricsContent -match 'process_resident_memory_bytes\s+(\d+)') {
        $memoryMB = [math]::Round([int]$matches[1] / 1MB, 2)
        $summary += "Resident memory: $memoryMB MB ($($matches[1]) bytes)"
    }
    if ($MetricsContent -match 'process_start_time_seconds\s+(\d+\.?\d*)') {
        $startTime = [datetime]::FromUnixTimeSeconds($matches[1])
        $summary += "Process start time: $startTime"
    }
    
    # Go runtime metrics
    $summary += "`n=== GO RUNTIME METRICS ==="
    if ($MetricsContent -match 'projectt_runtime_goroutines\s+(\d+)') {
        $summary += "Goroutines: $($matches[1])"
    }
    if ($MetricsContent -match 'projectt_runtime_memory_alloc_bytes\s+(\d+\.?\d*(?:e[+-]\d+)?)') {
        $allocMB = [math]::Round([double]$matches[1] / 1MB, 2)
        $summary += "Allocated memory: $allocMB MB"
    }
    
    # Application metrics
    $summary += "`n=== APPLICATION METRICS ==="
    
    $patterns = @{
        'projectt_items_total' = 'Items'
        'projectt_tags_total' = 'Tags'
        'projectt_chat_contacts_total' = 'Chat contacts'
        'projectt_chat_messages_total' = 'Chat messages'
        'projectt_p2p_peers_total' = 'P2P peers'
        'projectt_p2p_connections_total' = 'P2P connections'
    }
    
    foreach ($pattern in $patterns.Keys) {
        if ($MetricsContent -match "$pattern\s+(\d+\.?\d*)") {
            $summary += "$($patterns[$pattern]): $($matches[1])"
        }
    }
    
    # Database metrics
    $summary += "`n=== DATABASE ==="
    if ($MetricsContent -match 'projectt_db_queries_total\s+(\d+)') {
        $summary += "SQL queries: $($matches[1])"
    }
    if ($MetricsContent -match 'projectt_db_errors_total\s+(\d+)') {
        $summary += "DB errors: $($matches[1])"
    }
    
    return $summary -join "`n"
}

# Main logic
$url = "http://${ServerHost}:${Port}/metrics"
$metrics = Get-Metrics -Url $url

if (-not $metrics) {
    Write-Host "Failed to collect metrics. Make sure the application is running." -ForegroundColor Red
    exit 1
}

if ($SummaryOnly) {
    $summary = Extract-Summary -MetricsContent $metrics
    Write-Host $summary
}
else {
    Write-Host "Saving full metrics to: $OutputFile" -ForegroundColor Green
    $metrics | Out-File -FilePath $OutputFile -Encoding UTF8
    
    $summary = Extract-Summary -MetricsContent $metrics
    $summary | Out-File -FilePath "${OutputFile}.summary.txt" -Encoding UTF8
    
    Write-Host "`nSummary:" -ForegroundColor Cyan
    Write-Host $summary -ForegroundColor Yellow
    Write-Host "`nFull metrics saved to: $OutputFile" -ForegroundColor Green
}

# Health check
try {
    Write-Host "`nChecking health..." -ForegroundColor Cyan
    $healthUrl = "http://${ServerHost}:${Port}/health"
    $healthResponse = Invoke-WebRequest -Uri $healthUrl -ErrorAction Stop
    Write-Host "Health check: OK ($($healthResponse.StatusCode))" -ForegroundColor Green
}
catch {
    Write-Host "Health check unavailable" -ForegroundColor Yellow
}