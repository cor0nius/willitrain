package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "willitrain_http_requests_total",
	Help: "Total number of HTTP requests by path, method and code.",
}, []string{"path", "method", "code"})
