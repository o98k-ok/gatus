package api

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/gofiber/fiber/v2"
)

// MetricHistory returns the history of the configured metric for an endpoint
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

		// Find the endpoint's metric configuration
		var metricName, metricPattern, metricUnit string
		for _, ep := range cfg.Endpoints {
			if ep.Key() == endpointKey && ep.Metric != nil {
				metricName = ep.Metric.Name
				metricPattern = ep.Metric.Value
				metricUnit = ep.Metric.Unit
				break
			}
		}

		// If no metric configured, return empty response
		if metricName == "" {
			return c.Status(200).JSON(map[string]interface{}{
				"metric_name": "",
				"unit":        "",
				"timestamps":  []int64{},
				"values":      []string{},
			})
		}

		// Query historical data
		metricData, err := store.Get().GetMetricHistory(endpointKey, metricPattern, from, time.Now())
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(404).SendString(err.Error())
			}
			if errors.Is(err, common.ErrInvalidTimeRange) {
				return c.Status(400).SendString(err.Error())
			}
			return c.Status(500).SendString(err.Error())
		}

		return c.Status(http.StatusOK).JSON(map[string]interface{}{
			"metric_name": metricName,
			"unit":        metricUnit,
			"timestamps":  metricData.Timestamps,
			"values":      metricData.Values,
		})
	}
}
