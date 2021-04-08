package plugin

import "github.com/toddtreece/mqtt-datasource/pkg/mqtt"

type MQTTClient interface {
	IsConnected() bool
	GetMessages(topic string) ([]mqtt.Message, bool)
	Subscribe(topic string)
}

type MQTTDatasource struct {
	Client MQTTClient
}

func (m *MQTTDatasource) Dispose() {
	// Called before creating a a new instance to allow plugin authors
	// to cleanup.
}
