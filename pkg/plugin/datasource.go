package plugin

import (
	"context"
	"encoding/json"
	"path"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

// Make sure MQTTDatasource implements required interfaces.
// This is important to do since otherwise we will only get a
// not implemented error response from plugin in runtime.
var (
	_ backend.QueryDataHandler      = (*MQTTDatasource)(nil)
	_ backend.CheckHealthHandler    = (*MQTTDatasource)(nil)
	_ backend.StreamHandler         = (*MQTTDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*MQTTDatasource)(nil)
)

// NewMQTTDatasource creates a new datasource instance.
func NewMQTTInstance(ctx context.Context, s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	settings, err := getDatasourceSettings(s)
	if err != nil {
		return nil, err
	}

	client, err := mqtt.NewClient(ctx, *settings)
	if err != nil {
		return nil, err
	}

	return NewMQTTDatasource(client, s.UID), nil
}

type MQTTDatasource struct {
	Client        mqtt.Client
	channelPrefix string
}

// NewMQTTDatasource creates a new datasource instance.
func NewMQTTDatasource(client mqtt.Client, uid string) *MQTTDatasource {
	return &MQTTDatasource{
		Client:        client,
		channelPrefix: path.Join("ds", uid),
	}
}

// Dispose here tells plugin SDK that plugin wants to clean up resources
// when a new instance created. As soon as datasource settings change detected
// by SDK old datasource instance will be disposed and a new one will be created
// using NewMQTTDatasource factory function.
func (ds *MQTTDatasource) Dispose() {
	ds.Client.Dispose()
}

func getDatasourceSettings(s backend.DataSourceInstanceSettings) (*mqtt.Options, error) {
	settings := &mqtt.Options{}

	if err := json.Unmarshal(s.JSONData, settings); err != nil {
		return nil, err
	}

	if password, exists := s.DecryptedSecureJSONData["password"]; exists {
		settings.Password = password
	}

	if tlsClientCert, exists := s.DecryptedSecureJSONData["tlsClientCert"]; exists {
		settings.TLSClientCert = tlsClientCert
	}

	if tlsClientKey, exists := s.DecryptedSecureJSONData["tlsClientKey"]; exists {
		settings.TLSClientKey = tlsClientKey
	}

	if tlsCACert, exists := s.DecryptedSecureJSONData["tlsCACert"]; exists {
		settings.TLSCACert = tlsCACert
	}

	return settings, nil
}
