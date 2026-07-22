package application

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal *prometheus.CounterVec
	HttpRequestDuration *prometheus.HistogramVec
	DeployTotal *prometheus.CounterVec
	DeployRunningCurrent prometheus.Gauge
	DeployDuration *prometheus.HistogramVec
)

// @Ref: docs/sps/plans/20260722_v3.0_monitor_ir.md | @Date: 2026-07-22
func InitMetrics() {
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pdeploy_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pdeploy_http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	DeployTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pdeploy_deploy_total",
		Help: "Total number of deployments",
	}, []string{"project_id", "status"})

	DeployRunningCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pdeploy_deploy_running_current",
		Help: "Current number of running deployments",
	})

	DeployDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pdeploy_deploy_duration_seconds",
		Help:    "Duration of deployments",
		Buckets: prometheus.DefBuckets,
	}, []string{"project_id", "status"})
}
