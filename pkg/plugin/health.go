package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func (ds *MQTTDatasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if !ds.Client.IsConnected() {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "MQTT Disconnected",
		}, nil
	}

	if ds.Publisher.enabled() {
		if err := ds.Publisher.checkHealth(ctx); err != nil {
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "MQTT Connected, but publishing is not available: " + err.Error(),
			}, nil
		}
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "MQTT Connected, publishing enabled",
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "MQTT Connected",
	}, nil
}
