package plugin_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
	"github.com/grafana/mqtt-datasource/pkg/plugin"
	"github.com/stretchr/testify/require"
)

func TestCheckHealthHandler(t *testing.T) {
	t.Run("HealthStatusOK when can connect", func(t *testing.T) {
		ds := plugin.NewMQTTDatasource(&fakeMQTTClient{
			connected: true,
		}, "xyz", false)

		res, _ := ds.CheckHealth(
			context.Background(),
			&backend.CheckHealthRequest{},
		)

		require.Equal(t, res.Status, backend.HealthStatusOk)
		require.Equal(t, res.Message, "MQTT Connected")
	})

	t.Run("HealthStatusError when disconnected", func(t *testing.T) {
		ds := plugin.NewMQTTDatasource(&fakeMQTTClient{
			connected: false,
		}, "xyz", false)

		res, _ := ds.CheckHealth(
			context.Background(),
			&backend.CheckHealthRequest{},
		)

		require.Equal(t, res.Status, backend.HealthStatusError)
		require.Equal(t, res.Message, "MQTT Disconnected")
	})
}

type fakeMQTTClient struct {
	connected bool
}

func (c *fakeMQTTClient) GetTopic(_ string) (*mqtt.Topic, bool) {
	return nil, false
}

func (c *fakeMQTTClient) IsConnected() bool {
	return c.connected
}

func (c *fakeMQTTClient) Subscribe(_ string, _ log.Logger) *mqtt.Topic { return nil }
func (c *fakeMQTTClient) Unsubscribe(_ string, _ log.Logger)           {}
func (c *fakeMQTTClient) Dispose()                                     {}
func (c *fakeMQTTClient) Publish(_ string, _ map[string]any, _ string) (json.RawMessage, error) {
	return nil, nil
}
