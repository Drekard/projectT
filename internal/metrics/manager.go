package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Manager управляет регистром метрик и предоставляет удобный доступ
type Manager struct {
	Registry *prometheus.Registry
	Metrics  *Metrics
	Server   *Server
}

// NewManager создаёт новый менеджер метрик
func NewManager() *Manager {
	registry := prometheus.NewRegistry()

	// Регистрируем стандартные метрики Go процесса
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	metrics := New(registry)

	return &Manager{
		Registry: registry,
		Metrics:  metrics,
	}
}

// InitServer инициализирует HTTP сервер метрик
func (m *Manager) InitServer(enabled bool, port int, path string) error {
	if !enabled {
		return nil
	}

	address := fmt.Sprintf(":%d", port)
	m.Server = NewServer(address, path, m.Metrics, m.Registry)

	return m.Server.Start()
}

// Stop останавливает все сервисы метрик
func (m *Manager) Stop() error {
	if m.Server != nil {
		return m.Server.Stop()
	}
	return nil
}
