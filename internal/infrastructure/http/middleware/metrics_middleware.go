package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once
	poolOnce    sync.Once

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bikko_http_requests_total",
			Help: "Total number of HTTP requests processed by the Bikko API.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bikko_http_request_duration_seconds",
			Help:    "HTTP request latency distributions (p50, p95, p99) in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

func initHTTPMetrics() {
	metricsOnce.Do(func() {
		_ = prometheus.Register(httpRequestsTotal)
		_ = prometheus.Register(httpRequestDuration)
	})
}

type pgxPoolCollector struct {
	pool          *pgxpool.Pool
	acquiredDesc  *prometheus.Desc
	idleDesc      *prometheus.Desc
	totalDesc     *prometheus.Desc
	maxDesc       *prometheus.Desc
}

func (c *pgxPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredDesc
	ch <- c.idleDesc
	ch <- c.totalDesc
	ch <- c.maxDesc
}

func (c *pgxPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool == nil {
		return
	}
	stat := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.acquiredDesc, prometheus.GaugeValue, float64(stat.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idleDesc, prometheus.GaugeValue, float64(stat.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.totalDesc, prometheus.GaugeValue, float64(stat.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.maxDesc, prometheus.GaugeValue, float64(stat.MaxConns()))
}

// RegisterPgxPoolMetrics registers a Prometheus collector for pgxpool connection statistics.
func RegisterPgxPoolMetrics(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	poolOnce.Do(func() {
		collector := &pgxPoolCollector{
			pool: pool,
			acquiredDesc: prometheus.NewDesc(
				"bikko_db_pool_acquired_conns",
				"Number of currently acquired connections in the PostgreSQL pgxpool.",
				nil, nil,
			),
			idleDesc: prometheus.NewDesc(
				"bikko_db_pool_idle_conns",
				"Number of currently idle connections in the PostgreSQL pgxpool.",
				nil, nil,
			),
			totalDesc: prometheus.NewDesc(
				"bikko_db_pool_total_conns",
				"Total number of connections currently in the PostgreSQL pgxpool.",
				nil, nil,
			),
			maxDesc: prometheus.NewDesc(
				"bikko_db_pool_max_conns",
				"Maximum number of connections allowed in the PostgreSQL pgxpool.",
				nil, nil,
			),
		}
		_ = prometheus.Register(collector)
	})
}

// MetricsMiddleware records Prometheus metrics for HTTP request count and duration.
func MetricsMiddleware() gin.HandlerFunc {
	initHTTPMetrics()

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())
		elapsedSeconds := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(elapsedSeconds)
	}
}
