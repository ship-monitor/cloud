package handler

import (
	devicedomain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/domain/sensor"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

func toAPIDevice(device devicedomain.Device) *api.Device {
	return &api.Device{
		ID:           device.ID,
		Name:         device.Name,
		OwnerID:      device.OwnerID,
		Manufacturer: device.Manufacturer,
		Model:        device.Model,
		Revision:     device.Revision,
		Software:     device.Software,
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
		ID:           sensor.ID,
		Name:         sensor.Name,
		DeviceID:     sensor.DeviceID,
		DisplayName:  sensor.DisplayName,
		Manufacturer: sensor.Manufacturer,
		Model:        sensor.Model,
		Revision:     sensor.Revision,
		Software:     sensor.Software,
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
