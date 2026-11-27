package schema_test

import (
	_ "embed"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/grafana/dsconfig/schema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
	"k8s.io/kube-openapi/pkg/spec3"
)

//go:embed dsconfig.json
var configSchemaJSON []byte

const pluginID = "grafana-mqtt-datasource"

// mqtt.Options contains both broker settings and decrypted secrets. Only its
// non-secret fields belong to jsonData, alongside the editor and SDK settings.
func jsonDataModel(secureKeys []string) any {
	type editorAndProxySettings struct {
		TLSAuth                  bool    `json:"tlsAuth"`
		TLSAuthWithCACert        bool    `json:"tlsAuthWithCACert"`
		EnableSecureSocksProxy   bool    `json:"enableSecureSocksProxy"`
		SecureSocksProxyUsername string  `json:"secureSocksProxyUsername"`
		Timeout                  float64 `json:"timeout"`
		KeepAlive                float64 `json:"keepAlive"`
	}
	fields := reflect.VisibleFields(reflect.TypeFor[editorAndProxySettings]())
	for _, field := range reflect.VisibleFields(reflect.TypeFor[mqtt.Options]()) {
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if !slices.Contains(secureKeys, key) {
			fields = append(fields, field)
		}
	}
	return reflect.New(reflect.StructOf(fields)).Interface()
}

//go:generate go test -run TestPlugin -generateArtifacts
func TestPlugin(t *testing.T) {
	secureKeys := []string{"password", "tlsCACert", "tlsClientCert", "tlsClientKey", "secureSocksProxyPassword"}
	schema.RunPluginTests(t, schema.PluginUnderTest{
		ID:                pluginID,
		ConfigSchemaJSON:  configSchemaJSON,
		SettingsJSONModel: jsonDataModel(secureKeys),
		SecureKeys:        secureKeys,
		SettingsExamples: &pluginschema.SettingsExamples{Examples: map[string]*spec3.Example{
			"": example("Anonymous TCP broker", map[string]any{"uri": "tcp://localhost:1883"}, nil),
			"usernameOnly": example("Broker username without a password",
				map[string]any{"uri": "tcp://localhost:1883", "username": "<USERNAME>"}, nil),
			"brokerCredentials": example("Broker username and password over TLS",
				map[string]any{"uri": "tls://broker.example.com:8883", "username": "<USERNAME>"},
				map[string]string{"password": "<PASSWORD>"}),
			"serverTLS": example("TLS with system trust roots", map[string]any{"uri": "tls://broker.example.com:8883"}, nil),
			"privateCA": example("TLS with a private CA",
				map[string]any{"uri": "tls://broker.example.com:8883", "tlsAuthWithCACert": true},
				map[string]string{"tlsCACert": "<CA_CERTIFICATE_PEM>"}),
			"mutualTLS": example("Mutual TLS",
				map[string]any{"uri": "tls://broker.example.com:8883", "tlsAuth": true},
				map[string]string{"tlsClientCert": "<CLIENT_CERTIFICATE_PEM>", "tlsClientKey": "<CLIENT_KEY_PEM>"}),
			"combinedAuth": example("Broker credentials and mutual TLS with a private CA",
				map[string]any{"uri": "tls://broker.example.com:8883", "username": "<USERNAME>", "tlsAuth": true, "tlsAuthWithCACert": true},
				map[string]string{"password": "<PASSWORD>", "tlsCACert": "<CA_CERTIFICATE_PEM>", "tlsClientCert": "<CLIENT_CERTIFICATE_PEM>", "tlsClientKey": "<CLIENT_KEY_PEM>"}),
			"webSocket":       example("WebSocket broker", map[string]any{"uri": "ws://broker.example.com/mqtt"}, nil),
			"secureWebSocket": example("Secure WebSocket broker", map[string]any{"uri": "wss://broker.example.com/mqtt"}, nil),
			"privateDataSourceConnect": example("TLS through Private Data Source Connect",
				map[string]any{"uri": "tls://broker.example.com:8883", "enableSecureSocksProxy": true}, nil),
			"proxyCredentials": example("Private Data Source Connect with proxy credentials",
				map[string]any{"uri": "tls://broker.example.com:8883", "enableSecureSocksProxy": true, "secureSocksProxyUsername": "<PROXY_USERNAME>"},
				map[string]string{"secureSocksProxyPassword": "<PROXY_PASSWORD>"}),
			"unixSocket": example("Local Unix domain socket", map[string]any{"uri": "unix:///path/to/mqtt.sock"}, nil),
		}},
	})
}

func example(summary string, jsonData map[string]any, secrets map[string]string) *spec3.Example {
	value := map[string]any{
		"apiVersion": pluginID + ".datasource.grafana.app/v0alpha1",
		"kind":       "DataSource",
		"metadata":   map[string]string{"name": "mqtt"},
		"spec":       map[string]any{"title": "MQTT", "jsonData": jsonData},
	}
	if len(secrets) > 0 {
		secure := make(map[string]map[string]string, len(secrets))
		for key, placeholder := range secrets {
			secure[key] = map[string]string{"create": placeholder}
		}
		value["secure"] = secure
	}
	return &spec3.Example{ExampleProps: spec3.ExampleProps{Summary: summary, Value: value}}
}
