# Логирование производительности Grid Manager

## Обзор

В код менеджера сетки (`internal/ui/workspace/saved/grid_manager.go`) добавлено детальное логирование производительности для отслеживания:
- Времени выполнения каждой операции
- Количества одновременно запущенных асинхронных процессов
- Причин задержек и узких мест

## Файлы

### 1. `logging/logger.go`
Новый файл с системой логирования. Предоставляет:
- `Logger` - основной логгер с поддержкой временных меток
- `TimingSession` - сессия замера времени для операции
- Методы для синхронного и асинхронного логирования

### 2. Изменения в `grid_manager.go`
Добавлены замеры времени в ключевых функциях:
- `LoadItemsByParentWithSort` - загрузка элементов с сортировкой
- `loadItems` - основная загрузка элементов
- `createCardsConcurrently` - параллельное создание карточек
- `updateLayout` - обновление макета сетки

## Формат логов

Логи записываются в файл `grid_manager_timing.log` в корне проекта.

### Пример вывода:

```
================================================================================
GRID MANAGER TIMING LOG - Session started at: 2026-03-04 15:30:45
================================================================================

2026-03-04 15:30:45.123 | SYNC   | LoadItemsByParentWithSort         | 156.789ms    | ParentID:     5 | Items:     0
2026-03-04 15:30:45.125 | SYNC   | DB_LoadAndSortItems               |  12.345ms    | ParentID:     5 | Items:     0
2026-03-04 15:30:45.140 | SYNC   | LoadItems                         | 142.567ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.142 | SYNC   | clear                             |   1.234ms    | ParentID:     5 | Items:     0
2026-03-04 15:30:45.145 | ASYNC  | createCardsConcurrently           |  89.012ms    | ParentID:     5 | Items:    25 | Active Async:   8
2026-03-04 15:30:45.150 | SYNC   | spawn_goroutines                  |   2.345ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.230 | SYNC   | collect_results                   |  78.901ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.235 | SYNC   | widget_refresh_and_sizing         |  65.432ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.240 | SYNC   | widget_Refresh                    |   2.123ms    | ParentID:     5 | Items:     0
2026-03-04 15:30:45.242 | SYNC   | widget_MinSize                    |   1.876ms    | ParentID:     5 | Items:     0
2026-03-04 15:30:45.310 | SYNC   | updateLayout_initial              |  52.345ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.315 | SYNC   | container_clear                   |   0.567ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.316 | SYNC   | calculate_columns                 |   0.123ms    | ParentID:     5 | Items:     0
2026-03-04 15:30:45.318 | SYNC   | layoutEngine_CalculatePositions   |   1.234ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.325 | SYNC   | widget_resize_move                |   6.789ms    | ParentID:     5 | Items:    25
2026-03-04 15:30:45.332 | SYNC   | updateContainerSize               |   0.456ms    | ParentID:     5 | Items:     0
2026-03-04 15:30:45.358 | SYNC   | canvas_Refresh                    |  25.678ms    | ParentID:     5 | Items:     0
```

## Расшифровка полей

| Поле | Описание |
|------|----------|
| Timestamp | Время операции с точностью до миллисекунд |
| SYNC/ASYNC | Тип операции: синхронная или асинхронная |
| Operation | Имя операции |
| Duration | Длительность выполнения |
| ParentID | ID текущей папки |
| Items | Количество элементов |
| Active Async | Количество активных асинхронных операций (только для ASYNC) |

## Ключевые операции для анализа

### 1. Загрузка элементов
- `DB_LoadAndSortItems` - время получения данных из БД и сортировки
- `LoadItems` - общее время загрузки элементов в сетку

### 2. Создание карточек
- `createCardsConcurrently` - общее время параллельного создания карточек
  - `spawn_goroutines` - время запуска горутин
  - `collect_results` - время сбора результатов
  - `widget_refresh_and_sizing` - время refresh и вычисления размеров виджетов
    - `widget_Refresh` - refresh одного виджета
    - `widget_MinSize` - вычисление минимального размера

### 3. Обновление макета
- `updateLayout_initial` - первоначальное обновление макета
- `layoutEngine_CalculatePositions` - расчет позиций карточек
- `widget_resize_move` - изменение размеров и перемещение виджетов
- `canvas_Refresh` - обновление canvas

## Анализ производительности

### Что искать в логах:

1. **Большое время `DB_LoadAndSortItems`** (>100ms)
   - Проблема с базой данных
   - Медленная сортировка большого количества элементов

2. **Большое время `widget_refresh_and_sizing`**
   - Много элементов (пропорционально количеству)
   - Медленный refresh виджетов

3. **Большое время `canvas_Refresh`** (>50ms)
   - Тяжелый рендеринг
   - Большое количество объектов на экране

4. **Высокое `Active Async`** (>числа ядер CPU)
   - Перегрузка потоками
   - Стоит ограничить количество воркеров

5. **Большое время `layoutEngine_CalculatePositions`**
   - Сложный алгоритм расстановки
   - Много карточек для расчета

## Асинхронность

Код использует worker pool для параллельного создания карточек:
- Количество воркеров = `min(runtime.NumCPU(), len(items))`
- Все горутины запускаются одновременно
- Результаты собираются в канал и обрабатываются последовательно
- Refresh и MinSize выполняются в main goroutine (требование Fyne)

Поле `Active Async` показывает, сколько асинхронных операций выполнялось одновременно в момент логирования.

## Пример анализа

```
2026-03-04 15:30:45.145 | ASYNC  | createCardsConcurrently           |  89.012ms    | ParentID:     5 | Items:    25 | Active Async:   8
```

**Анализ:**
- 25 карточек создано за 89ms
- Одновременно работало 8 асинхронных процессов (вероятно, 8-ядерный CPU)
- Средняя скорость: ~3.5ms на карточку
- Это нормальная производительность

Если видите:
```
2026-03-04 15:30:45.145 | ASYNC  | createCardsConcurrently           | 500.000ms    | ParentID:     5 | Items:    25 | Active Async:   8
```

**Проблема:** 500ms для 25 карточек - это медленно. Стоит проверить:
- Время `widget_Refresh` и `widget_MinSize`
- Не блокируется ли main goroutine
- Количество и сложность виджетов

## Отключение логирования

Если логирование не нужно, можно закомментировать вызовы `logger.StartTiming()` или удалить импорт `logging` из `grid_manager.go`.

## Файл лога

Файл `grid_manager_timing.log` создается в корне проекта. Каждая сессия начинается с разделителя и временной метки. Логи добавляются в конец файла (append mode).

Для очистки логов:
```bash
# Windows
echo. > grid_manager_timing.log

# Linux/Mac
> grid_manager_timing.log
```
