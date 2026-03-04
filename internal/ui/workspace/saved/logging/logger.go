package logging

import (
	"log"
	"os"
	"sync"
	"time"
)

// Logger предоставляет структурированное логирование с временными метками
type Logger struct {
	file         *os.File
	mu           sync.Mutex
	startTime    time.Time
	asyncCounter int64
	asyncMu      sync.Mutex
	minDuration  time.Duration // Минимальная длительность для логирования
}

// TimingSession сессия замера времени для операции
type TimingSession struct {
	logger    *Logger
	name      string
	startTime time.Time
	parentID  int
	itemCount int
	isAsync   bool
	mu        sync.Mutex
	subOps    []*SubOp
	completed bool
}

// SubOp под-операция в сессии
type SubOp struct {
	Name      string
	Duration  time.Duration
	Timestamp time.Time
}

var (
	globalLogger *Logger
	once         sync.Once
)

// GetLogger возвращает глобальный экземпляр логгера
func GetLogger() *Logger {
	once.Do(func() {
		var err error
		file, err := os.OpenFile("grid_manager_timing.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("Failed to create log file: %v", err)
			return
		}
		globalLogger = &Logger{
			file:        file,
			startTime:   time.Now(),
			minDuration: 100 * time.Microsecond, // Фильтруем операции быстрее 100µs
		}
		globalLogger.writeHeader()
	})
	return globalLogger
}

// writeHeader записывает заголовок лога
func (l *Logger) writeHeader() {
	if l.file == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.file.WriteString("\n")                                                                                 //nolint:errcheck
	l.file.WriteString("================================================================================\n") //nolint:errcheck
	l.file.WriteString("GRID MANAGER TIMING LOG - Session started at: " + timestamp + "\n")                  //nolint:errcheck
	l.file.WriteString("================================================================================\n") //nolint:errcheck
	l.file.WriteString("\n")                                                                                 //nolint:errcheck
}

// StartTiming начинает новую сессию замера времени
func (l *Logger) StartTiming(operationName string, parentID int, itemCount int) *TimingSession {
	return &TimingSession{
		logger:    l,
		name:      operationName,
		startTime: time.Now(),
		parentID:  parentID,
		itemCount: itemCount,
		isAsync:   false,
	}
}

// StartAsyncTiming начинает сессию замера времени для асинхронной операции
func (l *Logger) StartAsyncTiming(operationName string, parentID int, itemCount int) *TimingSession {
	session := &TimingSession{
		logger:    l,
		name:      operationName,
		startTime: time.Now(),
		parentID:  parentID,
		itemCount: itemCount,
		isAsync:   true,
	}
	l.incrementAsyncCounter()
	return session
}

// incrementAsyncCounter увеличивает счетчик асинхронных операций
func (l *Logger) incrementAsyncCounter() {
	l.asyncMu.Lock()
	defer l.asyncMu.Unlock()
	l.asyncCounter++
}

// decrementAsyncCounter уменьшает счетчик асинхронных операций
func (l *Logger) decrementAsyncCounter() {
	l.asyncMu.Lock()
	defer l.asyncMu.Unlock()
	l.asyncCounter--
}

// getAsyncCounter возвращает текущее количество активных асинхронных операций
func (l *Logger) getAsyncCounter() int64 {
	l.asyncMu.Lock()
	defer l.asyncMu.Unlock()
	return l.asyncCounter
}

// End завершает сессию и записывает результат в лог
func (ts *TimingSession) End() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.completed {
		return
	}
	ts.completed = true

	duration := time.Since(ts.startTime)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	if ts.logger.file == nil {
		return
	}

	ts.logger.mu.Lock()
	defer ts.logger.mu.Unlock()

	// Форматируем строку лога
	logLine := ""
	if ts.isAsync {
		activeAsync := ts.logger.getAsyncCounter()
		logLine = formatAsyncLogLine(timestamp, ts.name, duration, ts.parentID, ts.itemCount, activeAsync)
		ts.logger.decrementAsyncCounter()
	} else {
		logLine = formatSyncLogLine(timestamp, ts.name, duration, ts.parentID, ts.itemCount)
	}

	ts.logger.file.WriteString(logLine + "\n") //nolint:errcheck

	// Записываем под-операции если они есть
	// Фильтруем мгновенные операции (< minDuration)
	if len(ts.subOps) > 0 {
		// Сначала пишем summary с общей статистикой
		hasSignificantOps := false
		for _, subOp := range ts.subOps {
			if subOp.Duration >= ts.logger.minDuration || subOp.Duration == 0 {
				hasSignificantOps = true
				break
			}
		}

		if hasSignificantOps {
			for _, subOp := range ts.subOps {
				// Пропускаем мгновенные операции (но пишем с duration=0 для статистики)
				if subOp.Duration < ts.logger.minDuration && subOp.Duration > 0 {
					continue
				}
				subOpLine := formatSubOpLogLine(subOp.Name, subOp.Duration, subOp.Timestamp)
				ts.logger.file.WriteString(subOpLine + "\n") //nolint:errcheck
			}
		}
	}
}

// RecordSubOp записывает под-операцию
func (ts *TimingSession) RecordSubOp(name string, duration time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	subOp := &SubOp{
		Name:      name,
		Duration:  duration,
		Timestamp: time.Now(),
	}
	ts.subOps = append(ts.subOps, subOp)
}

// Elapsed возвращает прошедшее время с начала сессии
func (ts *TimingSession) Elapsed() time.Duration {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return time.Since(ts.startTime)
}

// formatSyncLogLine форматирует строку лога для синхронной операции
func formatSyncLogLine(timestamp, operation string, duration time.Duration, parentID, itemCount int) string {
	return formatLogLine(timestamp, operation, duration, parentID, itemCount, 0, false)
}

// formatAsyncLogLine форматирует строку лога для асинхронной операции
func formatAsyncLogLine(timestamp, operation string, duration time.Duration, parentID, itemCount int, activeAsync int64) string {
	return formatLogLine(timestamp, operation, duration, parentID, itemCount, activeAsync, true)
}

// formatLogLine универсальная функция форматирования
func formatLogLine(timestamp, operation string, duration time.Duration, parentID, itemCount int, activeAsync int64, isAsync bool) string {
	durationStr := duration.String()
	if duration < time.Millisecond {
		durationStr = formatMicroseconds(duration)
	}

	if isAsync {
		return formatWithAsync(timestamp, operation, durationStr, parentID, itemCount, activeAsync)
	}
	return formatWithoutAsync(timestamp, operation, durationStr, parentID, itemCount)
}

// formatMicroseconds форматирует длительность в микросекундах
func formatMicroseconds(d time.Duration) string {
	return string(rune(d.Microseconds())) + "µs"
}

// formatWithAsync форматирует строку с информацией об асинхронности
func formatWithAsync(timestamp, operation, duration string, parentID, itemCount int, activeAsync int64) string {
	return padRight(timestamp, 22) + " | ASYNC  | " +
		padRight(operation, 35) + " | " +
		padLeft(duration, 12) + " | ParentID: " + padLeftInt(parentID, 5) +
		" | Items: " + padLeftInt(itemCount, 5) +
		" | Active Async: " + padLeftInt64(activeAsync, 3)
}

// formatWithoutAsync форматирует строку без информации об асинхронности
func formatWithoutAsync(timestamp, operation, duration string, parentID, itemCount int) string {
	return padRight(timestamp, 22) + " | SYNC   | " +
		padRight(operation, 35) + " | " +
		padLeft(duration, 12) + " | ParentID: " + padLeftInt(parentID, 5) +
		" | Items: " + padLeftInt(itemCount, 5)
}

// formatSubOpLogLine форматирует строку под-операции
func formatSubOpLogLine(name string, duration time.Duration, timestamp time.Time) string {
	// Для операций с duration=0 (статистика) не пишем время
	if duration == 0 {
		return "    └─ " + name
	}

	durationStr := duration.String()
	if duration < time.Millisecond {
		durationStr = formatMicroseconds(duration)
	}
	return "    └─ " + padRight(name, 30) + ": " + padLeft(durationStr, 12)
}

// padRight дополняет строку пробелами справа до указанной длины
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + repeatString(" ", length-len(s))
}

// padLeft дополняет строку пробелами слева до указанной длины
func padLeft(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return repeatString(" ", length-len(s)) + s
}

// padLeftInt дополняет целое число пробелами слева
func padLeftInt(n int, length int) string {
	s := intToString(n)
	if len(s) >= length {
		return s
	}
	return repeatString(" ", length-len(s)) + s
}

// padLeftInt64 дополняет int64 пробелами слева
func padLeftInt64(n int64, length int) string {
	s := int64ToString(n)
	if len(s) >= length {
		return s
	}
	return repeatString(" ", length-len(s)) + s
}

// repeatString повторяет строку n раз
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

// intToString преобразует int в string без импорта strconv
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// int64ToString преобразует int64 в string без импорта strconv
func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// Close закрывает файл лога
func (l *Logger) Close() {
	if l.file != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.file.Close()
	}
}
