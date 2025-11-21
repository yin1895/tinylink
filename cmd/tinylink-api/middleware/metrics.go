package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 记录请求总数 (Counter)
	// 标签: method(GET/POST), path(/shorten), status(200/500)
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// 记录请求耗时 (Histogram)
	// 标签: method, path
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets, // 默认的时间桶 (0.005s, 0.01s, ... 10s)
		},
		[]string{"method", "path"},
	)
)

// PrometheusMiddleware 是 Gin 的中间件函数
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 继续处理请求
		c.Next()

		// 请求处理完后，记录指标
		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()
		path := c.FullPath()

		// 如果路径为空（比如 404），记录为 unknown
		if path == "" {
			path = "unknown"
		}

		// 1. 计数器 +1
		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()

		// 2. 记录耗时
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
