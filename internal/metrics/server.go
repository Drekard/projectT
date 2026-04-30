package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server HTTP сервер для экспорта метрик Prometheus
type Server struct {
	server   *http.Server
	metrics  *Metrics
	registry *prometheus.Registry
}

// NewServer создаёт новый сервер экспорта метрик
func NewServer(address, path string, metrics *Metrics, registry *prometheus.Registry) *Server {
	mux := http.NewServeMux()

	// Endpoint для сбора метрик
	mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		Registry:          registry,
	}))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) //nolint:errcheck
	})

	// Info endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html>
<head><title>ProjectT Metrics</title></head>
<body>
<h1>ProjectT Prometheus Metrics</h1>
<p><a href="` + path + `">Metrics endpoint</a></p>
<p><a href="/health">Health check</a></p>
<h2>Полезные ссылки:</h2>
<ul>
<li><a href="https://prometheus.io/docs/prometheus/latest/getting_started/">Prometheus Getting Started</a></li>
<li><a href="https://grafana.com/docs/grafana/latest/getting-started/">Grafana Getting Started</a></li>
</ul>
</body>
</html>`
		w.Write([]byte(html)) //nolint:errcheck
	})

	srv := &http.Server{
		Addr:         address,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	return &Server{
		server:   srv,
		metrics:  metrics,
		registry: registry,
	}
}

// Start запускает HTTP сервер метрик в фоновом режиме
func (s *Server) Start() error {
	// Запускаем сбор runtime метрик
	go s.collectRuntime()

	// Запускаем HTTP сервер
	go func() {
		_ = s.server.ListenAndServe()
	}()

	return nil
}

// Stop останавливает HTTP сервер метрик
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// collectRuntime периодически собирает метрики рантайма
func (s *Server) collectRuntime() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Собираем сразу при запуске
	s.metrics.CollectRuntime()

	for range ticker.C {
		s.metrics.CollectRuntime()
	}
}
