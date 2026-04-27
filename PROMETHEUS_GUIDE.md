# Руководство по мониторингу ProjectT с Prometheus и Grafana

## Быстрый старт

### 1. Запуск с метриками
По умолчанию метрики включены в `config.yaml`. Для запуска:
```bash
go build -o projectT.exe cmd/main.go
.\projectT.exe
```

### 2. Проверка метрик
Метрики доступны по адресу:
- http://localhost:9090 - информация о сервере метрик
- http://localhost:9090/metrics - сырые метрики Prometheus
- http://localhost:9090/health - проверка здоровья

### 3. Сбор метрик через скрипт
```powershell
# Краткая сводка
powershell -File .\scripts\collect_metrics_en.ps1 -SummaryOnly

# Полный сбор в файл
powershell -File .\scripts\collect_metrics_en.ps1

# Сбор с другим портом
powershell -File .\scripts\collect_metrics_en.ps1 -Port 9091 -SummaryOnly
```

## Мониторинг в реальном времени

### Создание службы мониторинга
Создайте файл `monitor_projectt.ps1`:

```powershell
# monitor_projectt.ps1
while ($true) {
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "`n[$timestamp] Проверка метрик ProjectT..." -ForegroundColor Cyan
    
    powershell -File .\scripts\collect_metrics_en.ps1 -SummaryOnly
    
    # Ждем 30 секунд
    Start-Sleep -Seconds 30
}
```

Запуск:
```
powershell -File .\monitor_projectt.ps1
```

## Настройка Prometheus

### Базовая конфигурация
Используйте файл `prometheus.yml` в корне проекта:

```bash
# Запуск Prometheus
prometheus.exe --config.file=prometheus.yml
```

### Конфигурация графаны
1. Добавьте Prometheus как источник данных:
   - URL: http://localhost:9090
   - Тип: Prometheus

2. Примеры панелей:
   - **Runtime метрики**: goroutines, память, GC паузы
   - **Приложение**: элементы, теги, сообщения
   - **P2P сеть**: пиры, соединения, трафик
   - **База данных**: запросы, ошибки, время выполнения

## Примеры полезных запросов PromQL

### Ресурсы приложения
```
# Goroutines
projectt_runtime_goroutines

# Память (МБ)
projectt_runtime_memory_alloc_bytes / 1024 / 1024

# Паузы GC
projectt_runtime_gc_pause_seconds_total
```

### Активность приложения
```
# Созданные элементы
rate(projectt_items_created_total[5m])

# Сообщения чата
rate(projectt_chat_messages_total[5m])

# Ошибки БД
rate(projectt_db_errors_total[5m])
```

### P2P сеть
```
# Активные пиры
projectt_p2p_peers_total

# Трафик P2P (МБ)
projectt_p2p_transfer_bytes_total / 1024 / 1024

# Скорость передачи файлов
rate(projectt_p2p_files_transferred_total[5m])
```

## Оповещения (AlertRules)

Пример `alerts.yml` для Prometheus:
```yaml
groups:
  - name: projectt_alerts
    rules:
      - alert: HighMemoryUsage
        expr: projectt_runtime_memory_alloc_bytes > 100 * 1024 * 1024  # > 100MB
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Высокое использование памяти"
          
      - alert: DatabaseErrorsHigh
        expr: rate(projectt_db_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Высокая частота ошибок БД"
          
      - alert: NoPeersConnected
        expr: projectt_p2p_peers_total == 0
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Нет подключенных P2P пиров"
```

## Автоматизация

### Планировщик задач Windows
1. Откройте `Планировщик заданий`
2. Создайте новую задачу:
   - Триггер: При запуске системы
   - Действие: Запуск программы
   - Программа: `C:\Users\egors\Desktop\projectT\projectT.exe`
   - Аргументы: `--config=config.yaml`

### Сбор метрик в файл по расписанию
```powershell
# save_metrics_hourly.ps1
$date = Get-Date -Format "yyyy-MM-dd"
$filename = "metrics_${date}_hourly.txt"

powershell -File .\scripts\collect_metrics_en.ps1 -OutputFile "logs\$filename"
```

## Поиск проблем

### Проверка доступности
```powershell
# Проверка порта
Test-NetConnection -ComputerName localhost -Port 9090

# Проверка health
Invoke-WebRequest -Uri "http://localhost:9090/health" -UseBasicParsing

# Проверка метрик
(Invoke-WebRequest -Uri "http://localhost:9090/metrics" -UseBasicParsing).Content | Select-String "projectt_" | Select-Object -First 5
```

### Логирование
Приложение логирует запуск Prometheus:
```
[App] Prometheus сервер инициализирован на порту 9090
[Prometheus] Сервер метрик запущен на http://:9090
```

## Полезные команды
```powershell
# Сборка приложения
go build -o projectT.exe cmd/main.go

# Запуск с выводом логов
.\projectT.exe

# Запуск в фоне
Start-Process -FilePath ".\projectT.exe" -WindowStyle Hidden

# Остановка приложения
Stop-Process -Name "projectT" -Force

# Проверка процессов
Get-Process | Where-Object { $_.ProcessName -like "*project*" }

# Проверка занятых портов
netstat -an | findstr :9090
```

## Ссылки
- [Код метрик](internal/metrics/) - реализация метрик Prometheus
- [Конфигурация](config.yaml) - настройки Prometheus
- [Скрипты](scripts/) - утилиты для работы с метриками