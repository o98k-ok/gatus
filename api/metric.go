package api

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/gofiber/fiber/v2"
)

// MetricSeries represents a single metric's historical data
type MetricSeries struct {
	Name       string   `json:"name"`
	Unit       string   `json:"unit"`
	Timestamps []int64  `json:"timestamps"`
	Values     []string `json:"values"`
}

// MetricHistory returns the history of the configured metrics for an endpoint
func MetricHistory(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		duration := c.Params("duration")
		var from time.Time
		switch duration {
		case "30d":
			from = time.Now().Truncate(time.Hour).Add(-30 * 24 * time.Hour)
		case "7d":
			from = time.Now().Truncate(time.Hour).Add(-7 * 24 * time.Hour)
		case "24h":
			from = time.Now().Truncate(time.Hour).Add(-24 * time.Hour)
		case "1h":
			from = time.Now().Truncate(time.Minute).Add(-1 * time.Hour)
		default:
			return c.Status(400).SendString("Durations supported: 30d, 7d, 24h, 1h")
		}

		endpointKey, err := url.QueryUnescape(c.Params("key"))
		if err != nil {
			return c.Status(400).SendString("invalid key encoding")
		}

		// Find the endpoint's metrics configuration
		var metrics []*endpoint.Metric
		for _, ep := range cfg.Endpoints {
			if ep.Key() == endpointKey && len(ep.Metrics) > 0 {
				metrics = ep.Metrics
				break
			}
		}

		// If no metrics configured, return empty response with both legacy and new format
		if len(metrics) == 0 {
			return c.Status(200).JSON(map[string]interface{}{
				"metric_name": "",
				"unit":        "",
				"timestamps":  []int64{},
				"values":      []string{},
				"series":      []MetricSeries{},
			})
		}

		// TODO: v1 uses N+1 query pattern (one query per metric). Acceptable for 2-5 metrics.
		// If metric count grows, consider a batch SQL query in store layer.
		series := make([]MetricSeries, 0, len(metrics))
		for _, m := range metrics {
			metricData, err := store.Get().GetMetricHistory(endpointKey, m.Value, from, time.Now())
			if err != nil {
				if errors.Is(err, common.ErrEndpointNotFound) {
					return c.Status(404).SendString(err.Error())
				}
				if errors.Is(err, common.ErrInvalidTimeRange) {
					return c.Status(400).SendString(err.Error())
				}
				return c.Status(500).SendString(err.Error())
			}
			series = append(series, MetricSeries{
				Name:       m.Name,
				Unit:       m.Unit,
				Timestamps: metricData.Timestamps,
				Values:     metricData.Values,
			})
		}

		// For backward compatibility: when only 1 metric is configured,
		// include legacy flat fields (metric_name, unit, timestamps, values)
		// alongside the new series[] field.
		if len(series) == 1 {
			return c.Status(http.StatusOK).JSON(map[string]interface{}{
				"metric_name": series[0].Name,
				"unit":        series[0].Unit,
				"timestamps":  series[0].Timestamps,
				"values":      series[0].Values,
				"series":      series,
			})
		}

		return c.Status(http.StatusOK).JSON(map[string]interface{}{
			"series": series,
		})
	}
}
