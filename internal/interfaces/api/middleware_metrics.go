package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/riverisagame/godeploy/internal/application"
)

// metricsResponseWriter intercepts HTTP status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// @Ref: docs/sps/plans/20260722_v3.0_monitor_ir.md | @Date: 2026-07-22
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		rw := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		method := r.Method
		status := strconv.Itoa(rw.statusCode)

		if application.HttpRequestsTotal != nil {
			application.HttpRequestsTotal.WithLabelValues(method, path, status).Inc()
		}
		if application.HttpRequestDuration != nil {
			application.HttpRequestDuration.WithLabelValues(method, path).Observe(duration)
		}
	})
}
