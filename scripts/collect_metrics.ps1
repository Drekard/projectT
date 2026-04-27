#!/usr/bin/env pwsh

# Скрипт для сбора и сохранения метрик Prometheus

param (
    [string]$OutputFile = "metrics_$(Get-Date -Format 'yyyy-MM-dd_HH-mm-ss').txt",
    [int]$Port = 9090,
    [string]$Host = "localhost",
    [switch]$SummaryOnly
)

function Get-Metrics {
    param (
        [string]$Url
    )
    
    try {
        Write-Host "Получение метрик из $Url..." -ForegroundColor Green
        $response = Invoke-WebRequest -Uri $Url -ErrorAction Stop
        return $response.Content
    }
    catch {
        Write-Host "Ошибка при получении метрик: $_" -ForegroundColor Red
        return $null
    }
}

function Extract-Summary {
    param (
        [string]$MetricsContent
    )
    
    $summary = @()
    $summary += "=== ОБЗОР МЕТРИК ==="
    $summary += "Время сбора: $(Get-Date)"
    
    # Процессные метрики
    $summary += "`n=== ПРОЦЕССНЫЕ МЕТРИКИ ==="
    if ($MetricsContent -match 'process_cpu_seconds_total\s+(\d+\.?\d*)') {
        $summary += "CPU время: $($matches[1]) секунд"
    }
    if ($MetricsContent -match 'process_open_fds\s+(\d+)') {
        $summary += "Открытых файловых дескрипторов: $($matches[1])"
    }
    if ($MetricsContent -match 'process_resident_memory_bytes\s+(\d+)') {
        $memoryMB = [math]::Round([int]$matches[1] / 1MB, 2)
        $summary += "Используемая память: $memoryMB MB ($($matches[1]) bytes)"
    }
    if ($MetricsContent -match 'process_start_time_seconds\s+(\d+\.?\d*)') {
        $startTime = [datetime]::FromUnixTimeSeconds($matches[1])
        $summary += "Время запуска процесса: $startTime"
    }
    
    # Go runtime метрики
    $summary += "`n=== GO RUNTIME МЕТРИКИ ==="
    if ($MetricsContent -match 'projectt_runtime_goroutines\s+(\d+)') {
        $summary += "Goroutines: $($matches[1])"
    }
    if ($MetricsContent -match 'projectt_runtime_memory_alloc_bytes\s+(\d+\.?\d*(?:e[+-]\d+)?)') {
        $allocMB = [math]::Round([double]$matches[1] / 1MB, 2)
        $summary += "Выделенная память: $allocMB MB"
    }
    
    # Прикладные метрики
    $summary += "`n=== МЕТРИКИ ПРИЛОЖЕНИЯ ==="
    
    $patterns = @{
        'projectt_items_total' = 'Элементы'
        'projectt_tags_total' = 'Теги'
        'projectt_chat_contacts_total' = 'Контакты чата'
        'projectt_chat_messages_total' = 'Сообщения чата'
        'projectt_p2p_peers_total' = 'P2P пиры'
        'projectt_p2p_connections_total' = 'P2P соединения'
    }
    
    foreach ($pattern in $patterns.Keys) {
        if ($MetricsContent -match "$pattern\s+(\d+\.?\d*)") {
            $summary += "$($patterns[$pattern]): $($matches[1])"
        }
    }
    
    # Метрики БД
    $summary += "`n=== БАЗА ДАННЫХ ==="
    if ($MetricsContent -match 'projectt_db_queries_total\s+(\d+)') {
        $summary += "SQL запросов: $($matches[1])"
    }
    if ($MetricsContent -match 'projectt_db_errors_total\s+(\d+)') {
        $summary += "Ошибок БД: $($matches[1])"
    }
    
    return $summary -join "`n"
}

# Основная логика
$url = "http://${Host}:${Port}/metrics"
$metrics = Get-Metrics -Url $url

if (-not $metrics) {
    Write-Host "Не удалось получить метрики. Убедитесь, что приложение запущено." -ForegroundColor Red
    exit 1
}

if ($SummaryOnly) {
    $summary = Extract-Summary -MetricsContent $metrics
    Write-Host $summary
}
else {
    Write-Host "Сохранение полных метрик в файл: $OutputFile" -ForegroundColor Green
    $metrics | Out-File -FilePath $OutputFile -Encoding UTF8
    
    $summary = Extract-Summary -MetricsContent $metrics
    $summary | Out-File -FilePath "${OutputFile}.summary.txt" -Encoding UTF8
    
    Write-Host "`nКраткая сводка:" -ForegroundColor Cyan
    Write-Host $summary -ForegroundColor Yellow
    Write-Host "`nПолные метрики сохранены в: $OutputFile" -ForegroundColor Green
}

# Проверка health
try {
    Write-Host "`nПроверка health check..." -ForegroundColor Cyan
    $healthUrl = "http://${Host}:${Port}/health"
    $healthResponse = Invoke-WebRequest -Uri $healthUrl -ErrorAction Stop
    Write-Host "Health check: OK ($($healthResponse.StatusCode))" -ForegroundColor Green
}
catch {
    Write-Host "Health check недоступен" -ForegroundColor Yellow
}