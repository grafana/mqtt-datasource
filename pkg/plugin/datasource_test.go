package plugin_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/ISSACS-PSG/mqtt-datasource/pkg/mqtt"
	"github.com/ISSACS-PSG/mqtt-datasource/pkg/plugin"
	"github.com/stretchr/testify/require"
)

func TestCheckHealthHandler(t *testing.T) {
	t.Run("HealthStatusOK when can connect", func(t *testing.T) {
		ds := plugin.NewMQTTDatasource(&fakeMQTTClient{
			connected:  true,
			subscribed: false,
		}, "xyz")

		res, _ := ds.CheckHealth(
			context.Background(),
			&backend.CheckHealthRequest{},
		)

		require.Equal(t, res.Status, backend.HealthStatusOk)
		require.Equal(t, res.Message, "MQTT Connected")
	})

	t.Run("HealthStatusError when disconnected", func(t *testing.T) {
		ds := plugin.NewMQTTDatasource(&fakeMQTTClient{
			connected:  false,
			subscribed: false,
		}, "xyz")

		res, _ := ds.CheckHealth(
			context.Background(),
			&backend.CheckHealthRequest{},
		)

		require.Equal(t, res.Status, backend.HealthStatusError)
		require.Equal(t, res.Message, "MQTT Disconnected")
	})
}

type fakeMQTTClient struct {
	connected  bool
	subscribed bool
}

func (c *fakeMQTTClient) GetTopic(_ string) (*mqtt.Topic, bool) {
	return nil, false
}

func (c *fakeMQTTClient) IsConnected() bool {
	return c.connected
}

func (c *fakeMQTTClient) IsSubscribed(_ string) bool {
	return c.subscribed
}

func (c *fakeMQTTClient) Subscribe(_ string) *mqtt.Topic { return nil }
func (c *fakeMQTTClient) Unsubscribe(_ string)           {}
func (c *fakeMQTTClient) Dispose()                       {}
