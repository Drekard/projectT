package metrics

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := New(registry)

	assert.NotNil(t, m)
	assert.NotNil(t, m.ItemsTotal)
	assert.NotNil(t, m.TagsTotal)
	assert.NotNil(t, m.ChatMessagesTotal)
	assert.NotNil(t, m.P2PPeersTotal)
}

func TestMetricsIncrement(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := New(registry)

	// Инкрементируем счётчики
	m.ItemsCreated.Inc()
	m.ItemsCreated.Inc()
	m.ItemsDeleted.Inc()

	// Проверяем значения
	assert.Equal(t, float64(2), testutil.ToFloat64(m.ItemsCreated))
	assert.Equal(t, float64(1), testutil.ToFloat64(m.ItemsDeleted))
}

func TestGaugeMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := New(registry)

	// Устанавливаем значения
	m.ItemsTotal.Set(100)
	m.TagsTotal.Set(25)
	m.P2PPeersTotal.Set(5)

	// Проверяем значения
	assert.Equal(t, float64(100), testutil.ToFloat64(m.ItemsTotal))
	assert.Equal(t, float64(25), testutil.ToFloat64(m.TagsTotal))
	assert.Equal(t, float64(5), testutil.ToFloat64(m.P2PPeersTotal))
}

func TestManager(t *testing.T) {
	mgr := NewManager()

	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.Registry)
	assert.NotNil(t, mgr.Metrics)
}

func TestGlobalManager(t *testing.T) {
	// Сбрасываем глобальный менеджер для чистоты теста
	globalManager = nil
	once = sync.Once{}

	mgr := Init()
	assert.NotNil(t, mgr)

	// Повторный вызов должен вернуть тот же экземпляр
	mgr2 := Get()
	assert.Same(t, mgr, mgr2)

	// IsInitialized должен вернуть true
	assert.True(t, IsInitialized())
}
