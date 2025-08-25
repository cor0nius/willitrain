package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// This file defines the Prometheus metrics that are exposed by the application.

// httpRequestsTotal is a Prometheus counter vector that tracks the total number of HTTP requests.
// It is partitioned by the request's URL path, HTTP method, and the resulting status code.
var httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "willitrain_http_requests_total",
	Help: "Total number of HTTP requests by path, method and code.",
}, []string{"path", "method", "code"})
