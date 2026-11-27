//nolint:varnamelen
package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

// GetDeviceMetricsByName implements [api.Handler].
func (h *Handler) GetDeviceMetricsByName(
	ctx context.Context,
	params api.GetDeviceMetricsByNameParams,
) (*api.Metrics, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	query := metrics.Query{
		From:  nil,
		To:    nil,
		Limit: params.Limit.Value,
		Name:  params.Metric,
	}
	if params.From.Set {
		query.From = &params.From.Value
	}

	if params.To.Set {
		query.To = &params.To.Value
	}

	ms, err := h.metrics.GetDeviceMetrics(
		ctx,
		actor,
		params.DeviceID,
		query,
	)
	if errors.Is(err, device.ErrNotFound) {
		return nil, errDeviceNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get device metrics: %w", err)
	}

	return toAPIMetrics(ms), nil
}

// GetSensorMetricsByName implements [api.Handler].
func (h *Handler) GetSensorMetricsByName(
	ctx context.Context,
	params api.GetSensorMetricsByNameParams,
) (*api.Metrics, error) {
	actor, err := currentActor(ctx)
	if err != nil {
		return nil, fmt.Errorf("get actor: %w", err)
	}

	query := queryFromParams(params)

	ms, err := h.metrics.GetSensorMetrics(
		ctx,
		actor,
		params.DeviceID,
		params.SensorID,
		query,
	)
	if errors.Is(err, device.ErrNotFound) {
		return nil, errDeviceNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get sensor metrics: %w", err)
	}

	return toAPIMetrics(ms), nil
}

// queryFromParams maps the API query parameters to a domain metrics
// query. Zero from/to values mean unbounded range.
func queryFromParams(params api.GetSensorMetricsByNameParams) metrics.Query {
	var query metrics.Query

	if params.From.Set {
		query.From = &params.From.Value
	}

	if params.To.Set {
		query.To = &params.To.Value
	}

	query.Name = params.Metric

	if params.Limit.Set {
		query.Limit = params.Limit.Value
	}

	return query
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
