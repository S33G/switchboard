package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dockerEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "switchboard_docker_events_total",
			Help: "Total number of Docker events received",
		},
		[]string{"host", "action"},
	)

	syncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "switchboard_sync_duration_seconds",
			Help:    "Duration of container sync operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host"},
	)

	containerCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "switchboard_containers_total",
			Help: "Total number of containers by host and state",
		},
		[]string{"host", "state"},
	)

	wsClientsConnected = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "switchboard_websocket_clients",
			Help: "Number of connected WebSocket clients",
		},
	)

	nginxReloadsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "switchboard_nginx_reloads_total",
			Help: "Total number of nginx reloads",
		},
	)

	nginxReloadErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "switchboard_nginx_reload_errors_total",
			Help: "Total number of nginx reload errors",
		},
	)

	nginxConfigGenDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "switchboard_nginx_config_gen_duration_seconds",
			Help:    "Duration of nginx config generation",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
	)
)
