package plugin_test

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
	"github.com/grafana/mqtt-datasource/pkg/plugin"
	"github.com/stretchr/testify/require"
)

func TestCheckHealthHandler(t *testing.T) {
	t.Run("HealthStatusOK when can connect", func(t *testing.T) {
		ds := plugin.NewMQTTDatasource(&fakeMQTTClient{
			connected:  true,
			subscribed: false,
		}, 5)

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
		}, 5)

		res, _ := ds.CheckHealth(
			context.Background(),
			&backend.CheckHealthRequest{},
		)

		require.Equal(t, res.Status, backend.HealthStatusError)
		require.Equal(t, res.Message, "MQTT Disconnected")
	})
}

func TestToFrame(t *testing.T) {
	frame := plugin.ToFrame("test/data", []mqtt.Message{
		{
			Timestamp: time.Unix(1, 0),
			Value:     "1",
		},
	})
	v, err := frame.Fields[1].FloatAt(0)
	require.NoError(t, err)
	require.Equal(t, v, float64(1))

	frame = plugin.ToFrame("test/data", []mqtt.Message{
		{
			Timestamp: time.Unix(1, 0),
			Value:     `{"gx": -0.5182926829268293, "gy": -0.3582317073170732, "gz": 0.1753048780487805, "ax": 0.20599365234375, "ay": -0.050048828125, "az": 1.03582763671875}`,
		},
	})
	v, err = frame.Fields[1].FloatAt(0)
	require.NoError(t, err)
	require.Equal(t, v, float64(1))
}

type fakeMQTTClient struct {
	connected  bool
	subscribed bool
}

func (c *fakeMQTTClient) IsConnected() bool {
	return c.connected
}

func (c *fakeMQTTClient) IsSubscribed(_ string) bool {
	return c.subscribed
}

func (c *fakeMQTTClient) Messages(_ string) ([]mqtt.Message, bool) {
	return []mqtt.Message{}, true
}

func (c *fakeMQTTClient) Stream() chan mqtt.StreamMessage {
	return make(chan mqtt.StreamMessage)
}

func (c *fakeMQTTClient) Subscribe(topic string) {}

func (c *fakeMQTTClient) Unsubscribe(topic string) {}
