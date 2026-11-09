package mqtt

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"time"

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
		return backend.DownstreamErrorf("MQTT proxy client creation failed: %s", err)
	}

	if !proxyClient.SecureSocksProxyEnabled() {
		// PDC not enabled, use standard connection
		return nil
	}

	logger.Info("MQTT using secure socks proxy")
	proxyDialer, err := proxyClient.NewSecureSocksProxyContextDialer()
	if err != nil {
		return backend.DownstreamErrorf("MQTT secure socks proxy dialer creation failed: %s", err)
	}

	opts.SetCustomOpenConnectionFn(newProxyConnectionFunc(proxyDialer, logger))
	return nil
}

// newProxyConnectionFunc creates a custom connection function for Paho MQTT.
//
// A custom connection function replaces Paho's own TLS handshake, so TLS is
// applied here for TLS schemes.
func newProxyConnectionFunc(dialer proxyDialer, logger log.Logger) paho.OpenConnectionFunc {
	return func(uri *url.URL, options paho.ClientOptions) (net.Conn, error) {
		// The WebSocket upgrade is done by Paho's own connection path, which this
		// function replaces.
		switch uri.Scheme {
		case "ws", "wss":
			return nil, backend.DownstreamErrorf("MQTT over %s is not supported with private data source connect (PDC), use a tcp:// or tls:// URI", uri.Scheme)
		}

		network := "tcp"
		address := buildAddress(uri)

		logger.Debug("MQTT connecting via secure socks proxy",
			"network", network,
			"address", address,
			"scheme", uri.Scheme)

		conn, err := dialer.Dial(network, address)
		if err != nil {
			return nil, err
		}

		if !isTLSScheme(uri.Scheme) {
			return conn, nil
		}

		tlsConfig := options.TLSConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
		if tlsConfig.ServerName == "" {
			tlsConfig = tlsConfig.Clone()
			tlsConfig.ServerName = uri.Hostname()
		}

		if options.ConnectTimeout > 0 {
			if err := conn.SetDeadline(time.Now().Add(options.ConnectTimeout)); err != nil {
				_ = conn.Close()
				return nil, err
			}
		}

		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			_ = conn.Close()
			return nil, err
		}

		if err := conn.SetDeadline(time.Time{}); err != nil {
			_ = conn.Close()
			return nil, err
		}

		return tlsConn, nil
	}
}

// isTLSScheme reports whether scheme is one Paho treats as TLS.
func isTLSScheme(scheme string) bool {
	switch scheme {
	case "ssl", "tls", "mqtts", "mqtt+ssl", "tcps":
		return true
	default:
		return false
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
	case "ssl", "tls", "tcps", "mqtts", "mqtt+ssl":
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
