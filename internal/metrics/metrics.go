package metrics

import (
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics хранит все кастомные метрики приложения
type Metrics struct {
	// Item метрики элементов
	ItemsTotal   prometheus.Gauge
	ItemsCreated prometheus.Counter
	ItemsDeleted prometheus.Counter
	ItemsByType  *prometheus.GaugeVec

	// Tag метрики тегов
	TagsTotal prometheus.Gauge

	// Chat метрики чата
	ChatMessagesTotal  prometheus.Counter
	ChatContactsTotal  prometheus.Gauge
	ChatActiveContacts prometheus.Gauge

	// P2P метрики пиров
	P2PPeersTotal         prometheus.Gauge
	P2PConnectionsTotal   prometheus.Counter
	P2PTransferBytesTotal prometheus.Counter
	P2PFilesTransferred   prometheus.Counter

	// Database метрики БД
	DBQueryDuration prometheus.Histogram
	DBQueriesTotal  prometheus.Counter
	DBErrorsTotal   prometheus.Counter

	// Runtime метрики рантайма
	GoRoutines       prometheus.Gauge
	MemoryAlloc      prometheus.Gauge
	MemoryTotalAlloc prometheus.Gauge
	MemorySys        prometheus.Gauge
	GCTotalPause     prometheus.Gauge
}

// New создаёт и регистрирует все метрики приложения
func New(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)

	m := &Metrics{
		ItemsTotal: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "items",
			Name:      "total",
			Help:      "Общее количество элементов в системе",
		}),

		ItemsCreated: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "items",
			Name:      "created_total",
			Help:      "Общее количество созданных элементов",
		}),

		ItemsDeleted: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "items",
			Name:      "deleted_total",
			Help:      "Общее количество удалённых элементов",
		}),

		ItemsByType: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "items",
			Name:      "by_type",
			Help:      "Количество элементов по типам (element, folder, link)",
		}, []string{"type"}),

		TagsTotal: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "tags",
			Name:      "total",
			Help:      "Общее количество тегов",
		}),

		ChatMessagesTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "chat",
			Name:      "messages_total",
			Help:      "Общее количество отправленных/полученных сообщений",
		}),

		ChatContactsTotal: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "chat",
			Name:      "contacts_total",
			Help:      "Общее количество контактов",
		}),

		ChatActiveContacts: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "chat",
			Name:      "active_contacts",
			Help:      "Количество активных контактов (онлайн)",
		}),

		P2PPeersTotal: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "p2p",
			Name:      "peers_total",
			Help:      "Текущее количество подключенных пиров",
		}),

		P2PConnectionsTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "p2p",
			Name:      "connections_total",
			Help:      "Общее количество установленных соединений",
		}),

		P2PTransferBytesTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "p2p",
			Name:      "transfer_bytes_total",
			Help:      "Общее количество переданных байт через P2P",
		}),

		P2PFilesTransferred: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "p2p",
			Name:      "files_transferred_total",
			Help:      "Общее количество переданных файлов",
		}),

		DBQueryDuration: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: "projectt",
			Subsystem: "db",
			Name:      "query_duration_seconds",
			Help:      "Время выполнения SQL запросов",
			Buckets:   prometheus.DefBuckets,
		}),

		DBQueriesTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "db",
			Name:      "queries_total",
			Help:      "Общее количество SQL запросов",
		}),

		DBErrorsTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "projectt",
			Subsystem: "db",
			Name:      "errors_total",
			Help:      "Общее количество ошибок БД",
		}),

		GoRoutines: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "runtime",
			Name:      "goroutines",
			Help:      "Количество goroutines",
		}),

		MemoryAlloc: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "runtime",
			Name:      "memory_alloc_bytes",
			Help:      "Текущий объём выделенной памяти (bytes)",
		}),

		MemoryTotalAlloc: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "runtime",
			Name:      "memory_total_alloc_bytes",
			Help:      "Общий объём выделенной памяти за всё время (bytes)",
		}),

		MemorySys: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "runtime",
			Name:      "memory_sys_bytes",
			Help:      "Общий объём памяти запрошенный у ОС (bytes)",
		}),

		GCTotalPause: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "projectt",
			Subsystem: "runtime",
			Name:      "gc_pause_seconds_total",
			Help:      "Общее время пауз GC (seconds)",
		}),
	}

	return m
}

// CollectRuntime собирает метрики Go рантайма
func (m *Metrics) CollectRuntime() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.GoRoutines.Set(float64(runtime.NumGoroutine()))
	m.MemoryAlloc.Set(float64(memStats.Alloc))
	m.MemoryTotalAlloc.Set(float64(memStats.TotalAlloc))
	m.MemorySys.Set(float64(memStats.Sys))
	m.GCTotalPause.Set(float64(memStats.PauseTotalNs) / 1e9)
}
