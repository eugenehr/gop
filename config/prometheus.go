package config

// PrometheusConfig holds prometheus endpoint settings.
//
// Настройки экспорта метрик Prometheus.
type PrometheusConfig struct {
	// Host:Port to bind prometheus endpoint.
	// Адрес:Порт для запуска экспортера Prometheus.
	Bind string `json:"bind"`
	// Prometheus metrics path. Defaults to /metrics.
	// HTTP-путь для экспорта метрик. По умолчанию /metrics.
	Path string `yaml:"path"`
}
