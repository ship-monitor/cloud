//nolint:varnamelen
package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	deviceservice "github.com/ship-monitor/cloud/internal/v2/services/device"
)

// GetDeviceMetrics implements [api.Handler].
func (h *Handler) GetDeviceMetrics(
	ctx context.Context,
	params api.GetDeviceMetricsParams,
) (*api.Metrics, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	ms, err := h.metrics.GetDeviceMetrics(
		ctx,
		actor,
		params.DeviceID,
		metrics.Query{}, //nolint:exhaustruct_v5
	)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return nil, errDeviceNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get device metrics: %w", err)
	}

	return toAPIMetrics(ms), nil
}

// GetSensorMetrics implements [api.Handler].
func (h *Handler) GetSensorMetrics(
	ctx context.Context,
	params api.GetSensorMetricsParams,
) (*api.Metrics, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	query, err := queryFromParams(params)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	ms, err := h.metrics.GetSensorMetrics(
		ctx,
		actor,
		params.DeviceID,
		params.SensorID,
		query,
	)
	if errors.Is(err, deviceservice.ErrNotFound) {
		return nil, errDeviceNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get sensor metrics: %w", err)
	}

	return toAPIMetrics(ms), nil
}

// queryFromParams maps the API query parameters to a domain metrics
// query. Zero from/to values mean unbounded range.
func queryFromParams(params api.GetSensorMetricsParams) (metrics.Query, error) {
	var query metrics.Query

	if !params.From.IsZero() {
		from := params.From
		query.From = &from
	}

	if !params.To.IsZero() {
		to := params.To
		query.To = &to
	}

	if params.Metric != "" {
		query.Names = []string{params.Metric}
	}

	if limit, ok := params.Limit.Get(); ok {
		if limit < 0 {
			return metrics.Query{}, errInvalidLimit
		}

		query.Limit = uint64(limit)
	}

	return query, nil
}

// toAPIMetrics maps domain metric samples to the API response. The
// top-level name, type and unit are taken from the first sample; every
// sample keeps its own values in Values.
func toAPIMetrics(ms []metrics.Metric) *api.Metrics {
	res := &api.Metrics{ //nolint:exhaustruct_v5
		Values: make([]api.Metric, 0, len(ms)),
	}

	if len(ms) == 0 {
		return res
	}

	res.Name = ms[0].Name
	res.Type = api.MetricType(ms[0].Type)
	res.Unit = ms[0].Unit

	for _, m := range ms {
		res.Values = append(res.Values, api.Metric{
			Name:      m.Name,
			Type:      api.MetricType(m.Type),
			Unit:      m.Unit,
			Value:     m.Value,
			Timestamp: m.Timestamp,
		})
	}

	return res
}
