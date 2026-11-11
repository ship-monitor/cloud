package handler

import (
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

func toAPIDevice(d devicedomain.Device) *api.Device {
	return &api.Device{
		ID:      d.ID,
		Name:    d.Name,
		OwnerID: d.OwnerID,
	}
}

func toAPIDevices(devices []devicedomain.Device) []api.Device {
	out := make([]api.Device, len(devices))
	for i, d := range devices {
		out[i] = *toAPIDevice(d)
	}

	return out
}

func toApiSensor(sensor *sensor.Sensor) *api.Sensor {
	return &api.Sensor{
		ID:          sensor.ID,
		Name:        sensor.Name,
		DeviceID:    sensor.DeviceID,
		DisplayName: sensor.DisplayName,
	}
}

// pagingFromParams converts API pagination parameters to domain paging.
//
// The generated code enforces page >= 0 and pageSize >= 1,
// but not the maximum page size or integer values, so both are checked here.
func pagingFromParams(
	params api.GetDevicesParams,
) (devicedomain.Paging, error) {
	paging := devicedomain.Paging{
		Page: 0,
		Size: devicedomain.DefaultPageSize,
	}

	if v, ok := params.Page.Get(); ok {
		paging.Page = v
	}

	if v, ok := params.PageSize.Get(); ok {
		if v > devicedomain.MaxPageSize {
			return paging, errInvalidPagination
		}

		paging.Size = v
	}

	return paging, nil
}
