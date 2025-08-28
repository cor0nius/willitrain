package main

import (
	"net/http"
	"strconv"
	"time"
)

// This file contains the HTTP middleware functions used by the application.
// Middleware are handlers that wrap other handlers to provide cross-cutting
// functionality like logging, metrics, and CORS.

// --- Server-Side Middleware ---

// responseWriter is a wrapper around http.ResponseWriter that allows us to capture
// the HTTP status code written to the response. This is essential for metrics,
// as the standard ResponseWriter interface doesn't expose the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	// Default to 200 OK if WriteHeader is not called.
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code before calling the underlying ResponseWriter's method.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware is a wrapping handler that captures the HTTP status code of a
// response and records it as a Prometheus metric, along with the request path and method.
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := newResponseWriter(w)
		next.ServeHTTP(rw, r)

		statusCodeStr := strconv.Itoa(rw.statusCode)
		httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, statusCodeStr).Inc()
	})
}

// corsMiddleware is a wrapping handler that adds the Access-Control-Allow-Origin
// header to all responses to allow cross-origin requests from any domain.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

// --- Client-Side Middleware (RoundTripper) ---

// metricsTransport is a custom http.RoundTripper that wraps another RoundTripper
// to record the duration of outgoing HTTP requests as a Prometheus metric.
type metricsTransport struct {
	wrapped http.RoundTripper
}

// RoundTrip executes a single HTTP transaction, wrapping the call to the nested
// RoundTripper to measure the request's duration.
func (t *metricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.wrapped.RoundTrip(req)
	duration := time.Since(start).Seconds()

	// Record the duration metric, using the request's host as a label.
	externalRequestDuration.WithLabelValues(req.URL.Host).Observe(duration)

	return resp, err
}
