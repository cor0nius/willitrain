package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

// mockHandler is a test HTTP handler that simulates the behavior of real handlers.
func mockHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Simulate a successful response
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "OK")
	case http.MethodPost:
		// Simulate a "Not Found" response for a different status code test
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "Not Found")
	case http.MethodPut:
		// Simulate a handler that doesn't explicitly write a status code
		_, _ = io.WriteString(w, "Implicit OK")
	default:
		// Simulate a "Method Not Allowed" response
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "Method Not Allowed")
	}
}

func TestMetricsMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedLabels prometheus.Labels
	}{
		{
			name:           "Successful GET request",
			method:         http.MethodGet,
			path:           "/test",
			expectedStatus: http.StatusOK,
			expectedLabels: prometheus.Labels{"path": "/test", "method": "GET", "code": "200"},
		},
		{
			name:           "Not Found POST request",
			method:         http.MethodPost,
			path:           "/test",
			expectedStatus: http.StatusNotFound,
			expectedLabels: prometheus.Labels{"path": "/test", "method": "POST", "code": "404"},
		},
		{
			name:           "Method Not Allowed DELETE request",
			method:         http.MethodDelete,
			path:           "/another",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedLabels: prometheus.Labels{"path": "/another", "method": "DELETE", "code": "405"},
		},
		{
			name:           "Implicit OK for PUT request",
			method:         http.MethodPut,
			path:           "/implicit",
			expectedStatus: http.StatusOK,
			expectedLabels: prometheus.Labels{"path": "/implicit", "method": "PUT", "code": "200"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the metric before each test
			httpRequestsTotal.Reset()

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			handler := metricsMiddleware(http.HandlerFunc(mockHandler))
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			counter := httpRequestsTotal.With(tt.expectedLabels)
			if err := testutil.CollectAndCompare(counter, strings.NewReader(
				`# HELP willitrain_http_requests_total Total number of HTTP requests by path, method and code.
				# TYPE willitrain_http_requests_total counter
				willitrain_http_requests_total{code="`+strconv.Itoa(tt.expectedStatus)+`",method="`+tt.method+`",path="`+tt.path+`"} 1
				`,
			), "willitrain_http_requests_total"); err != nil {
				t.Errorf("unexpected metric value:\n%s", err)
			}
		})
	}
}

func TestCorsMiddleware(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	// Create a dummy handler to be wrapped by the middleware
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := corsMiddleware(dummyHandler)
	handler.ServeHTTP(rr, req)

	if header := rr.Header().Get("Access-Control-Allow-Origin"); header != "*" {
		t.Errorf("handler returned wrong CORS header: got %q want %q", header, "*")
	}
}

// mockTransport is a custom http.RoundTripper for testing client-side middleware.
type mockTransport struct {
	resp *http.Response
	err  error
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.resp, t.err
}

func TestMetricsTransport(t *testing.T) {
	tests := []struct {
		name        string
		transport   http.RoundTripper
		expectError bool
	}{
		{
			name: "Successful RoundTrip",
			transport: &mockTransport{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("OK")),
				},
				err: nil,
			},
			expectError: false,
		},
		{
			name: "Error RoundTrip",
			transport: &errorTransport{
				err: errors.New("network error"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			externalRequestDuration.Reset()

			metricsT := &metricsTransport{wrapped: tt.transport}
			req := httptest.NewRequest("GET", "http://testhost/api", nil)

			resp, err := metricsT.RoundTrip(req)

			if tt.expectError {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, but got: %v", err)
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("expected status OK, got: %v", resp.StatusCode)
				}
			}

			// Check if the metric was observed by checking the count.
			observer := externalRequestDuration.WithLabelValues(req.URL.Host)
			metric := &dto.Metric{}
			_ = observer.(prometheus.Metric).Write(metric)

			if metric.Histogram == nil {
				t.Fatal("metric.Histogram is nil, metric is not a histogram")
			}

			if *metric.Histogram.SampleCount != 1 {
				t.Errorf("expected metric count to be 1, got %d", *metric.Histogram.SampleCount)
			}
		})
	}
}
