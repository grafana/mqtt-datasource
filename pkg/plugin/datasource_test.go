package plugin_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
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

type fakeInstanceManager struct {
	client *fakeMQTTClient
	err    error
}

func (im *fakeInstanceManager) Get(pc backend.PluginContext) (instancemgmt.Instance, error) {
	return &plugin.MQTTDatasource{
		Client: im.client,
	}, im.err
}

func (im *fakeInstanceManager) Do(pc backend.PluginContext, fn instancemgmt.InstanceCallbackFunc) error {
	return nil
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