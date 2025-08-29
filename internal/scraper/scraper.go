// This file implements a standalone metrics scraper for the WillItRain application.
// It is designed to be deployed as a separate, serverless container (e.g., on Cloud Run)
// and triggered periodically by a scheduler (e.g., Cloud Scheduler).
//
// The scraper performs the following steps:
//  1. Receives an HTTP request from the scheduler.
//  2. Fetches Prometheus metrics from the main application's /metrics endpoint.
//  3. Parses the text-based Prometheus exposition format, handling counters, gauges,
//     and histograms.
//  4. Converts the parsed metrics into the format required by Google Cloud's
//     Managed Service for Prometheus.
//  5. Ingests the converted metrics into Google Cloud Monitoring.
//
// This approach decouples metrics collection from the main application, ensuring
// that scraping is reliable and independently managed.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"google.golang.org/genproto/googleapis/api/distribution"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// main is the entry point for the scraper service.
// It sets up a JSON-based structured logger, configures an HTTP server as required
// by the Cloud Run environment, and registers the scrapeHandler to process
// incoming requests.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Info("starting server", "port", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		scrapeHandler(w, r, logger)
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}

// scrapeHandler handles incoming HTTP requests from Cloud Scheduler.
// It orchestrates the scraping and ingestion process and logs the outcome.
func scrapeHandler(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	logger.Info("scrape request received")
	if err := scrapeAndIngest(r.Context(), logger); err != nil {
		logger.Error("error during scrape and ingest", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("successfully scraped and ingested metrics")
	fmt.Fprintln(w, "Success")
}

// scrapeAndIngest performs the core logic of fetching, parsing, and ingesting metrics.
// It reads configuration from environment variables, calls the function to convert
// Prometheus metrics to Google Cloud Monitoring TimeSeries, and then writes them.
func scrapeAndIngest(ctx context.Context, logger *slog.Logger) error {
	metricsURL := os.Getenv("METRICS_URL")
	if metricsURL == "" {
		return fmt.Errorf("environment variable METRICS_URL must be set")
	}
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		return fmt.Errorf("environment variable PROJECT_ID must be set")
	}

	// Fetch metrics and convert them to the Google Cloud Monitoring format.
	timeSeries, err := fetchAndConvertToTimeSeries(ctx, projectID, metricsURL, logger)
	if err != nil {
		return fmt.Errorf("failed to fetch and convert metrics: %w", err)
	}

	if len(timeSeries) == 0 {
		logger.Info("no metric samples found to ingest")
		return nil
	}

	// Ingest the TimeSeries data into Google Cloud Monitoring.
	if err := ingestMetrics(ctx, projectID, timeSeries); err != nil {
		return fmt.Errorf("failed to ingest metrics: %w", err)
	}

	return nil
}

// fetchAndConvertToTimeSeries scrapes a Prometheus endpoint, parses the response,
// and converts the metrics into Google Cloud Monitoring's TimeSeries format.
// It handles Counter, Gauge, Untyped, and Histogram metric types.
func fetchAndConvertToTimeSeries(ctx context.Context, projectID, url string, logger *slog.Logger) ([]*monitoringpb.TimeSeries, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http request failed with status code %d", resp.StatusCode)
	}

	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus metrics: %w", err)
	}

	resource := &monitoredres.MonitoredResource{
		Type: "prometheus_target",
		Labels: map[string]string{
			"project_id": projectID,
			"location":   "europe-west1",
			"cluster":    "__gce__",
			"namespace":  "willitrain",
			"job":        "willitrain",
			"instance":   url,
		},
	}

	var timeSeriesList []*monitoringpb.TimeSeries
	now := timestamppb.New(time.Now())

	for name, mf := range metricFamilies {
		for _, m := range mf.GetMetric() {
			labels := make(map[string]string)
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}

			ts := &monitoringpb.TimeSeries{
				Metric: &metric.Metric{
					Type:   "prometheus.googleapis.com/" + name,
					Labels: labels,
				},
				Resource: resource,
			}

			var point *monitoringpb.Point
			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				point = createPoint(now, m.GetCounter().GetValue())
			case dto.MetricType_GAUGE:
				point = createPoint(now, m.GetGauge().GetValue())
			case dto.MetricType_UNTYPED:
				point = createPoint(now, m.GetUntyped().GetValue())
			case dto.MetricType_HISTOGRAM:
				point = createDistributionPoint(now, m.GetHistogram(), logger)
			case dto.MetricType_SUMMARY:
				logger.Debug("skipping metric with unhandled summary type", "metric", name)
				continue
			default:
				logger.Warn("skipping metric with unhandled type", "metric", name, "type", mf.GetType())
				continue
			}

			ts.Points = []*monitoringpb.Point{point}
			timeSeriesList = append(timeSeriesList, ts)
		}
	}
	return timeSeriesList, nil
}

// createPoint creates a monitoring TimeSeries point with a double value.
// This is used for simple metrics like counters and gauges.
func createPoint(timestamp *timestamppb.Timestamp, value float64) *monitoringpb.Point {
	return &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime: timestamp,
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DoubleValue{
				DoubleValue: value,
			},
		},
	}
}

// createDistributionPoint creates a monitoring TimeSeries point for a histogram.
// It converts a Prometheus histogram DTO into a Google Cloud Monitoring Distribution value.
func createDistributionPoint(timestamp *timestamppb.Timestamp, h *dto.Histogram, logger *slog.Logger) *monitoringpb.Point {
	promBuckets := h.GetBucket()
	bounds := make([]float64, len(promBuckets)-1)
	bucketCounts := make([]int64, len(promBuckets))
	var lastCumulativeCount uint64

	for i, b := range promBuckets {
		// The last bucket in Prometheus is +Inf, which we don't need for bounds.
		if i < len(promBuckets)-1 {
			bounds[i] = b.GetUpperBound()
		}
		cumulativeCount := b.GetCumulativeCount()
		countInBucket := cumulativeCount - lastCumulativeCount
		if countInBucket > math.MaxInt64 {
			logger.Warn("histogram bucket count exceeds MaxInt64, capping value", "bucket", i, "value", countInBucket)
			bucketCounts[i] = math.MaxInt64
		} else {
			bucketCounts[i] = int64(countInBucket)
		}
		lastCumulativeCount = cumulativeCount
	}

	sampleCount := h.GetSampleCount()
	var finalSampleCount int64
	if sampleCount > math.MaxInt64 {
		logger.Warn("histogram sample count exceeds MaxInt64, capping value", "value", sampleCount)
		finalSampleCount = math.MaxInt64
	} else {
		finalSampleCount = int64(sampleCount)
	}

	dist := &distribution.Distribution{
		Count: finalSampleCount,
		Mean:  h.GetSampleSum() / float64(h.GetSampleCount()),
		BucketOptions: &distribution.Distribution_BucketOptions{
			Options: &distribution.Distribution_BucketOptions_ExplicitBuckets{
				ExplicitBuckets: &distribution.Distribution_BucketOptions_Explicit{
					Bounds: bounds,
				},
			},
		},
		BucketCounts: bucketCounts,
	}

	return &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime: timestamp,
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DistributionValue{
				DistributionValue: dist,
			},
		},
	}
}

// ingestMetrics writes the TimeSeries data to the Google Cloud Monitoring API.
// It creates a new client for each call to ensure freshness but relies on
// underlying connection pooling.
func ingestMetrics(ctx context.Context, projectID string, timeSeries []*monitoringpb.TimeSeries) error {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create monitoring client: %w", err)
	}
	defer client.Close()

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       "projects/" + projectID,
		TimeSeries: timeSeries,
	}

	if err := client.CreateTimeSeries(ctx, req); err != nil {
		return fmt.Errorf("failed to write time series data: %w", err)
	}
	return nil
}
