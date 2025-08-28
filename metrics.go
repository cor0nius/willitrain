package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// This file defines the Prometheus metrics that are exposed by the application.

var (
	// httpRequestsTotal is a Prometheus counter vector that tracks the total number of HTTP requests.
	// It is partitioned by the request's URL path, HTTP method, and the resulting status code.
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "willitrain_http_requests_total",
		Help: "Total number of HTTP requests by path, method and code.",
	}, []string{"path", "method", "code"})

	// externalRequestDuration is a Prometheus histogram that tracks the duration of outgoing HTTP requests
	// to external APIs. It is partitioned by the target host.
	externalRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "willitrain_external_request_duration_seconds",
		Help:    "Duration of outgoing HTTP requests to external APIs.",
		Buckets: prometheus.LinearBuckets(1.0, 1.0, 10), // 10 buckets from 1s to 10s
	}, []string{"host"})

	// parserDuration is a Prometheus histogram that tracks the duration of parsing API responses.
	// It is partitioned by the weather provider (e.g., GMP, OWM) and the type of forecast.
	parserDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "willitrain_parser_duration_seconds",
		Help:    "Duration of parsing API responses.",
		Buckets: prometheus.DefBuckets, // Default buckets
	}, []string{"provider", "forecast_type"})
)
