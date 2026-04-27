# Анализ метрик Prometheus

## Обзор
Приложение ProjectT предоставляет метрики через HTTP на порту 9090 (по умолчанию) по пути `/metrics`.

## Основные категории метрик

### 1. Кастомные метрики приложения (`projectt_*`)

#### 1.1 Элементы (Items)
- `projectt_items_total` - Общее количество элементов в системе (Gauge)
- `projectt_items_created_total` - Общее количество созданных элементов (Counter)
- `projectt_items_deleted_total` - Общее количество удалённых элементов (Counter)
- `projectt_items_by_type` - Количество элементов по типам (element, folder, link) (GaugeVec, по метке `type`)

#### 1.2 Теги (Tags)
- `projectt_tags_total` - Общее количество тегов (Gauge)

#### 1.3 Чат (Chat)
- `projectt_chat_messages_total` - Общее количество отправленных/полученных сообщений (Counter)
- `projectt_chat_contacts_total` - Общее количество контактов (Gauge)
- `projectt_chat_active_contacts` - Количество активных контактов (онлайн) (Gauge)

#### 1.4 P2P сеть
- `projectt_p2p_peers_total` - Текущее количество подключенных пиров (Gauge)
- `projectt_p2p_connections_total` - Общее количество установленных соединений (Counter)
- `projectt_p2p_transfer_bytes_total` - Общее количество переданных байт через P2P (Counter)
- `projectt_p2p_files_transferred_total` - Общее количество переданных файлов (Counter)

#### 1.5 База данных
- `projectt_db_query_duration_seconds` - Время выполнения SQL запросов (Histogram)
- `projectt_db_queries_total` - Общее количество SQL запросов (Counter)
- `projectt_db_errors_total` - Общее количество ошибок БД (Counter)

#### 1.6 Runtime (Go)
- `projectt_runtime_goroutines` - Количество goroutines (Gauge)
- `projectt_runtime_memory_alloc_bytes` - Текущий объём выделенной памяти (bytes) (Gauge)
- `projectt_runtime_memory_total_alloc_bytes` - Общий объём выделенной памяти за всё время (bytes) (Gauge)
- `projectt_runtime_memory_sys_bytes` - Общий объём памяти запрошенный у ОС (bytes) (Gauge)
- `projectt_runtime_gc_pause_seconds_total` - Общее время пауз GC (seconds) (Gauge)

### 2. Стандартные метрики Go процесса
- `go_*` - метрики сборщика мусора, планировщика, потоков (предоставляются GoCollector)
- `process_*` - метрики процесса (использование CPU, памяти, файловых дескрипторов)

### 3. Метрики libp2p
Метрики P2P библиотеки (при включении `prometheus.enable_p2p_metrics`):
- `libp2p_*` - метрики сетевых соединений, потоков, ресурсов

## Конфигурация
По умолчанию в `config.yaml`:
```yaml
prometheus:
    enabled: true
    port: 9090
    path: /metrics
    enable_p2p_metrics: true
```

## Получение метрик
```
# Получить все метрики
curl http://localhost:9090/metrics

# Получить только метрики приложения
curl http://localhost:9090/metrics | grep "^projectt_"
```

## Пример использования в Grafana
В Prometheus следует добавить target:
```yaml
scrape_configs:
  - job_name: 'projectt'
    static_configs:
      - targets: ['localhost:9090']
```

## Отладка
Метрики можно проверить через веб-интерфейс:
- http://localhost:9090/ - информация о сервере метрик
- http://localhost:9090/metrics - raw метрики
- http://localhost:9090/health - health check

## Текущие значения (пример из запуска)
На основе последнего сбора метрик:

### Runtime метрики
- Goroutines: 82
- Выделенная память: ~55.9 MB
- Общая выделенная память: ~149.9 MB
- GC паузы: 0.0017 секунд
- CPU time: 4.81 секунд

### Приложение
- Элементы: 0
- Теги: 0
- Контакты чата: 0
- P2P соединения: 0

Все метрики доступны и работают корректно.