package mqtt

import (
	"context"
	"net"
	"net/url"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// proxyDialer is an interface for dialers that can establish network connections.
// This matches the interface returned by NewSecureSocksProxyContextDialer.
type proxyDialer interface {
	Dial(network, address string) (net.Conn, error)
}

// configureProxyIfEnabled configures the MQTT client options to use a secure SOCKS proxy
// if Private Datasource Connect (PDC) is enabled in the datasource settings.
//
// It checks if secure SOCKS proxy is enabled and if so, creates a custom connection function
// that routes all MQTT traffic through the configured proxy.
func configureProxyIfEnabled(ctx context.Context, opts *paho.ClientOptions, settings backend.DataSourceInstanceSettings, logger log.Logger) error {
	proxyClient, err := settings.ProxyClient(ctx)
	if err != nil {
		logger.Error("MQTT proxy client creation failed", "error", err)
		return backend.DownstreamErrorf("MQTT proxy client creation failed: %s", err)
	}

	if !proxyClient.SecureSocksProxyEnabled() {
		// PDC not enabled, use standard connection
		return nil
	}

	logger.Info("MQTT using secure socks proxy")
	proxyDialer, err := proxyClient.NewSecureSocksProxyContextDialer()
	if err != nil {
		logger.Error("MQTT secure socks proxy dialer creation failed", "error", err)
		return backend.DownstreamErrorf("MQTT secure socks proxy dialer creation failed: %s", err)
	}

	opts.SetCustomOpenConnectionFn(newProxyConnectionFunc(proxyDialer, logger))
	return nil
}

// newProxyConnectionFunc creates a custom connection function for Paho MQTT
func newProxyConnectionFunc(dialer proxyDialer, logger log.Logger) paho.OpenConnectionFunc {
	return func(uri *url.URL, options paho.ClientOptions) (net.Conn, error) {
		network := "tcp"
		address := buildAddress(uri)

		logger.Debug("MQTT connecting via secure socks proxy",
			"network", network,
			"address", address,
			"scheme", uri.Scheme)

		return dialer.Dial(network, address)
	}
}

// buildAddress constructs a "host:port" address string from a URL.
func buildAddress(uri *url.URL) string {
	// If port is already specified, use the full host:port
	if uri.Port() != "" {
		return uri.Host
	}

	// No port specified, use default based on scheme
	var port string
	switch uri.Scheme {
	case "tcp", "mqtt":
		port = "1883" // Standard MQTT port
	case "ssl", "tls", "tcps", "mqtts":
		port = "8883" // Standard MQTT over TLS port
	case "ws":
		port = "80" // WebSocket port
	case "wss":
		port = "443" // WebSocket Secure port
	default:
		port = "1883"
	}

	return net.JoinHostPort(uri.Hostname(), port)
}
