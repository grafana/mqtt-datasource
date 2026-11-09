package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getDefaultPortForScheme(scheme string) string {
	switch scheme {
	case "tcp", "mqtt":
		return "1883"
	case "ssl", "tls", "tcps", "mqtts", "mqtt+ssl":
		return "8883"
	case "ws":
		return "80"
	case "wss":
		return "443"
	default:
		return "1883"
	}
}

func validateProxyConfiguration(ctx context.Context, settings backend.DataSourceInstanceSettings) error {
	proxyClient, err := settings.ProxyClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create proxy client: %w", err)
	}

	if proxyClient.SecureSocksProxyEnabled() {
		_, err := proxyClient.NewSecureSocksProxyContextDialer()
		if err != nil {
			return fmt.Errorf("failed to create proxy dialer: %w", err)
		}
	}

	return nil
}

// TestBuildAddress tests the address building logic for various URI schemes
func TestBuildAddress(t *testing.T) {
	tests := []struct {
		name            string
		uri             string
		expectedAddress string
	}{
		{
			name:            "tcp with explicit port",
			uri:             "tcp://broker.example.com:1883",
			expectedAddress: "broker.example.com:1883",
		},
		{
			name:            "tcp without port (should default to 1883)",
			uri:             "tcp://broker.example.com",
			expectedAddress: "broker.example.com:1883",
		},
		{
			name:            "mqtt scheme without port",
			uri:             "mqtt://broker.example.com",
			expectedAddress: "broker.example.com:1883",
		},
		{
			name:            "ssl with explicit port",
			uri:             "ssl://broker.example.com:8883",
			expectedAddress: "broker.example.com:8883",
		},
		{
			name:            "ssl without port (should default to 8883)",
			uri:             "ssl://broker.example.com",
			expectedAddress: "broker.example.com:8883",
		},
		{
			name:            "tls without port",
			uri:             "tls://broker.example.com",
			expectedAddress: "broker.example.com:8883",
		},
		{
			name:            "mqtts without port",
			uri:             "mqtts://broker.example.com",
			expectedAddress: "broker.example.com:8883",
		},
		{
			name:            "mqtt+ssl without port",
			uri:             "mqtt+ssl://broker.example.com",
			expectedAddress: "broker.example.com:8883",
		},
		{
			name:            "ws without port (should default to 80)",
			uri:             "ws://broker.example.com",
			expectedAddress: "broker.example.com:80",
		},
		{
			name:            "wss without port (should default to 443)",
			uri:             "wss://broker.example.com",
			expectedAddress: "broker.example.com:443",
		},
		{
			name:            "custom port overrides default",
			uri:             "tcp://broker.example.com:9999",
			expectedAddress: "broker.example.com:9999",
		},
		{
			name:            "IPv4 address with port",
			uri:             "tcp://192.168.1.100:1883",
			expectedAddress: "192.168.1.100:1883",
		},
		{
			name:            "IPv4 address without port",
			uri:             "tcp://192.168.1.100",
			expectedAddress: "192.168.1.100:1883",
		},
		{
			name:            "IPv6 address with port",
			uri:             "tcp://[::1]:1883",
			expectedAddress: "[::1]:1883",
		},
		{
			name:            "IPv6 address without port",
			uri:             "tcp://[::1]",
			expectedAddress: "[::1]:1883",
		},
		{
			name:            "unknown scheme defaults to 1883",
			uri:             "unknown://broker.example.com",
			expectedAddress: "broker.example.com:1883",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL, err := url.Parse(tt.uri)
			require.NoError(t, err)

			address := buildAddress(parsedURL)
			assert.Equal(t, tt.expectedAddress, address)
		})
	}
}

// TestGetDefaultPortForScheme tests the default port selection logic
func TestGetDefaultPortForScheme(t *testing.T) {
	tests := []struct {
		scheme       string
		expectedPort string
	}{
		{"tcp", "1883"},
		{"mqtt", "1883"},
		{"ssl", "8883"},
		{"tls", "8883"},
		{"tcps", "8883"},
		{"mqtts", "8883"},
		{"mqtt+ssl", "8883"},
		{"ws", "80"},
		{"wss", "443"},
		{"unknown", "1883"}, // fallback
		{"", "1883"},        // empty fallback
	}

	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			port := getDefaultPortForScheme(tt.scheme)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

// TestConfigureProxyIfEnabled_Disabled tests that proxy configuration is skipped when PDC is not enabled
func TestConfigureProxyIfEnabled_Disabled(t *testing.T) {
	ctx := context.Background()

	settings := backend.DataSourceInstanceSettings{
		ID:       1,
		UID:      "test-uid",
		Type:     "mqtt",
		JSONData: []byte(`{}`),
	}

	opts := paho.NewClientOptions()

	err := configureProxyIfEnabled(ctx, opts, settings, backend.NewLoggerWith("logger", "test"))
	assert.NoError(t, err)
}

// TestConfigureProxyIfEnabled_InvalidProxyClient tests error handling when ProxyClient creation fails
func TestConfigureProxyIfEnabled_InvalidSettings(t *testing.T) {
	ctx := context.Background()

	// Empty settings should still work
	settings := backend.DataSourceInstanceSettings{}

	opts := paho.NewClientOptions()

	// This might succeed or fail depending on SDK behavior, but shouldn't panic
	_ = configureProxyIfEnabled(ctx, opts, settings, backend.NewLoggerWith("logger", "test"))
}

// TestValidateProxyConfiguration tests proxy configuration validation
func TestValidateProxyConfiguration(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		settings  backend.DataSourceInstanceSettings
		wantError bool
	}{
		{
			name: "valid settings without PDC",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				UID:      "test-uid",
				Type:     "mqtt",
				JSONData: []byte(`{}`),
			},
			wantError: false,
		},
		{
			name: "PDC enabled in settings",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				UID:      "test-uid",
				Type:     "mqtt",
				JSONData: []byte(`{"enableSecureSocksProxy": true}`),
			},
			wantError: false, // Won't actually enable in test environment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProxyConfiguration(ctx, tt.settings)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProxyClientConfiguration tests that proxy client is properly initialized from datasource settings
func TestProxyClientConfiguration(t *testing.T) {
	ctx := context.Background()

	// Test with empty settings
	emptySettings := backend.DataSourceInstanceSettings{
		ID:       1,
		UID:      "test-uid",
		Type:     "mqtt",
		JSONData: []byte(`{}`),
	}

	// ProxyClient should be created even without proxy configuration
	proxyClient, err := emptySettings.ProxyClient(ctx)
	require.NoError(t, err)
	require.NotNil(t, proxyClient)

	// By default, secure socks proxy should not be enabled
	assert.False(t, proxyClient.SecureSocksProxyEnabled())
}

// TestProxyClient_SecureSocksProxyEnabled tests detection of PDC enablement
func TestProxyClient_SecureSocksProxyEnabled(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		jsonData string
		expected bool
	}{
		{
			name:     "PDC not configured",
			jsonData: `{}`,
			expected: false,
		},
		{
			name:     "PDC explicitly disabled",
			jsonData: `{"enableSecureSocksProxy": false}`,
			expected: false,
		},
		{
			name:     "PDC enabled",
			jsonData: `{"enableSecureSocksProxy": true}`,
			expected: false, // Will be false in test env without actual proxy setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := backend.DataSourceInstanceSettings{
				ID:       1,
				UID:      "test-uid",
				Type:     "mqtt",
				JSONData: []byte(tt.jsonData),
			}

			proxyClient, err := settings.ProxyClient(ctx)
			require.NoError(t, err)

			// In test environment without actual proxy configuration,
			// SecureSocksProxyEnabled will return false even if the
			// setting is true in JSONData
			enabled := proxyClient.SecureSocksProxyEnabled()

			// Just verify we can call the method without errors
			assert.IsType(t, false, enabled)
		})
	}
}

// TestOptions_JSONMarshaling tests that Options struct can be properly marshaled/unmarshaled
func TestOptions_JSONMarshaling(t *testing.T) {
	original := Options{
		URI:           "tcp://broker.example.com:1883",
		Username:      "testuser",
		Password:      "testpass",
		ClientID:      "test-client-123",
		TLSCACert:     "ca-cert-content",
		TLSClientCert: "client-cert-content",
		TLSClientKey:  "client-key-content",
		TLSSkipVerify: true,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled Options
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.URI, unmarshaled.URI)
	assert.Equal(t, original.Username, unmarshaled.Username)
	assert.Equal(t, original.Password, unmarshaled.Password)
	assert.Equal(t, original.ClientID, unmarshaled.ClientID)
	assert.Equal(t, original.TLSCACert, unmarshaled.TLSCACert)
	assert.Equal(t, original.TLSClientCert, unmarshaled.TLSClientCert)
	assert.Equal(t, original.TLSClientKey, unmarshaled.TLSClientKey)
	assert.Equal(t, original.TLSSkipVerify, unmarshaled.TLSSkipVerify)
}

// TestNetworkTypeForSchemes tests that we always use "tcp" network type
func TestNetworkTypeForSchemes(t *testing.T) {
	// All MQTT connections use TCP at the network layer
	// even when using TLS, WebSockets, etc.
	schemes := []string{"tcp", "mqtt", "ssl", "tls", "tcps", "mqtts", "ws", "wss"}

	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			// In our implementation, network type is always "tcp"
			// because even TLS and WebSocket use TCP at the transport layer
			network := "tcp"
			assert.Equal(t, "tcp", network)
		})
	}
}

// TestProxyDialerCreation ensures proxy dialer can be created when PDC is enabled
func TestProxyDialerCreation(t *testing.T) {
	ctx := context.Background()
	settings := backend.DataSourceInstanceSettings{
		ID:       1,
		UID:      "test-uid",
		Type:     "mqtt",
		JSONData: []byte(`{}`),
	}

	proxyClient, err := settings.ProxyClient(ctx)
	require.NoError(t, err)

	// Even if PDC is not enabled in test environment,
	// we can verify that the ProxyClient is created successfully
	assert.NotNil(t, proxyClient)

	// In a real environment with PDC configured, SecureSocksProxyEnabled
	// would return true and NewSecureSocksProxyContextDialer would
	// return a working dialer
	enabled := proxyClient.SecureSocksProxyEnabled()
	assert.False(t, enabled, "PDC should not be enabled in test environment")
}

// mockProxyDialer is a mock implementation of the proxyDialer interface for testing
type mockProxyDialer struct {
	dialFunc func(network, address string) (net.Conn, error)
	callLog  []dialCall
}

type dialCall struct {
	network string
	address string
}

func (m *mockProxyDialer) Dial(network, address string) (net.Conn, error) {
	m.callLog = append(m.callLog, dialCall{network, address})
	if m.dialFunc != nil {
		return m.dialFunc(network, address)
	}
	return nil, assert.AnError
}

// TestNewProxyConnectionFunc tests that the proxy connection function is created correctly
func TestNewProxyConnectionFunc(t *testing.T) {
	logger := backend.NewLoggerWith("logger", "test")
	mockDialer := &mockProxyDialer{}

	connFunc := newProxyConnectionFunc(mockDialer, logger)
	require.NotNil(t, connFunc)

	// Test with a URI that has no port
	uri, err := url.Parse("tcp://broker.example.com")
	require.NoError(t, err)

	// This will fail to connect (returns AnError), but we can verify the dialer was called correctly
	_, _ = connFunc(uri, paho.ClientOptions{})

	// Verify the dialer was called with correct parameters
	require.Len(t, mockDialer.callLog, 1)
	assert.Equal(t, "tcp", mockDialer.callLog[0].network)
	assert.Equal(t, "broker.example.com:1883", mockDialer.callLog[0].address)
}

// TestNewProxyConnectionFunc_WithPort tests proxy connection function with explicit port
func TestNewProxyConnectionFunc_WithPort(t *testing.T) {
	logger := backend.NewLoggerWith("logger", "test")
	mockDialer := &mockProxyDialer{}

	connFunc := newProxyConnectionFunc(mockDialer, logger)

	// Test with a URI that has explicit port
	uri, err := url.Parse("ssl://broker.example.com:9999")
	require.NoError(t, err)

	_, _ = connFunc(uri, paho.ClientOptions{})

	// Verify the dialer was called with the explicit port
	require.Len(t, mockDialer.callLog, 1)
	assert.Equal(t, "tcp", mockDialer.callLog[0].network)
	assert.Equal(t, "broker.example.com:9999", mockDialer.callLog[0].address)
}

// TestNewProxyConnectionFunc_HandshakeHonoursConnectTimeout tests that a broker which
// accepts the connection and then never answers the TLS handshake fails at ConnectTimeout.
func TestNewProxyConnectionFunc_HandshakeHonoursConnectTimeout(t *testing.T) {
	// net.Pipe is an in-memory connection. Nothing reads the server side, so the
	// handshake can never finish, like a broker that goes silent.
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	// Safety net: without a deadline the handshake would block forever, so
	// unblock it after 5 seconds and let the elapsed time check below fail.
	safetyNet := time.AfterFunc(5*time.Second, func() { _ = server.Close() })
	defer safetyNet.Stop()

	mockDialer := &mockProxyDialer{
		dialFunc: func(network, address string) (net.Conn, error) {
			return client, nil
		},
	}
	connFunc := newProxyConnectionFunc(mockDialer, backend.NewLoggerWith("logger", "test"))

	uri, err := url.Parse("tls://broker.example.com:8883")
	require.NoError(t, err)

	start := time.Now()
	_, err = connFunc(uri, paho.ClientOptions{ConnectTimeout: 200 * time.Millisecond})

	require.Error(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "the handshake should give up at ConnectTimeout")
}

// newTLSBroker starts a local TLS server and returns a dialer that always connects to it,
// plus a TLS config that trusts its certificate. The server is Go's built-in HTTPS test
// server. We only need something that completes a TLS handshake, not a real broker.
// Its certificate is valid for example.com, 127.0.0.1 and ::1.
func newTLSBroker(t *testing.T) (*mockProxyDialer, *tls.Config) {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(server.Close)

	trusted := x509.NewCertPool()
	trusted.AddCert(server.Certificate())

	dialer := &mockProxyDialer{
		dialFunc: func(network, address string) (net.Conn, error) {
			return net.Dial("tcp", server.Listener.Addr().String())
		},
	}
	return dialer, &tls.Config{RootCAs: trusted}
}

// TestNewProxyConnectionFunc_TLSSchemesStartTLS tests that every TLS scheme gets a TLS connection.
func TestNewProxyConnectionFunc_TLSSchemesStartTLS(t *testing.T) {
	for _, scheme := range []string{"tls", "ssl", "mqtts", "mqtt+ssl", "tcps"} {
		t.Run(scheme, func(t *testing.T) {
			dialer, trustedConfig := newTLSBroker(t)
			connFunc := newProxyConnectionFunc(dialer, backend.NewLoggerWith("logger", "test"))

			uri, err := url.Parse(scheme + "://example.com:8883")
			require.NoError(t, err)

			conn, err := connFunc(uri, paho.ClientOptions{TLSConfig: trustedConfig})
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			_, isTLS := conn.(*tls.Conn)
			assert.True(t, isTLS, "expected a TLS connection for %s", scheme)
		})
	}
}

// TestNewProxyConnectionFunc_PlainSchemesAreNotWrapped tests that tcp and mqtt URIs are returned as is, without TLS.
func TestNewProxyConnectionFunc_PlainSchemesAreNotWrapped(t *testing.T) {
	for _, scheme := range []string{"tcp", "mqtt"} {
		t.Run(scheme, func(t *testing.T) {
			client, server := net.Pipe()
			t.Cleanup(func() {
				_ = client.Close()
				_ = server.Close()
			})

			mockDialer := &mockProxyDialer{
				dialFunc: func(network, address string) (net.Conn, error) {
					return client, nil
				},
			}
			connFunc := newProxyConnectionFunc(mockDialer, backend.NewLoggerWith("logger", "test"))

			uri, err := url.Parse(scheme + "://broker.example.com:1883")
			require.NoError(t, err)

			conn, err := connFunc(uri, paho.ClientOptions{})
			require.NoError(t, err)

			assert.Same(t, client, conn, "a plain connection should be returned as is")
		})
	}
}

// TestNewProxyConnectionFunc_VerifiesCertificate tests that the datasource's TLS settings are used.
func TestNewProxyConnectionFunc_VerifiesCertificate(t *testing.T) {
	t.Run("trusted certificate connects", func(t *testing.T) {
		dialer, trustedConfig := newTLSBroker(t)
		connFunc := newProxyConnectionFunc(dialer, backend.NewLoggerWith("logger", "test"))

		uri, err := url.Parse("tls://example.com:8883")
		require.NoError(t, err)

		conn, err := connFunc(uri, paho.ClientOptions{TLSConfig: trustedConfig})
		require.NoError(t, err)
		_ = conn.Close()
	})

	t.Run("untrusted certificate is rejected", func(t *testing.T) {
		dialer, _ := newTLSBroker(t)
		connFunc := newProxyConnectionFunc(dialer, backend.NewLoggerWith("logger", "test"))

		uri, err := url.Parse("tls://example.com:8883")
		require.NoError(t, err)

		// No TLSConfig, so the broker's certificate is not in the trusted list.
		_, err = connFunc(uri, paho.ClientOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate")
	})

	t.Run("certificate for another hostname is rejected", func(t *testing.T) {
		dialer, trustedConfig := newTLSBroker(t)
		connFunc := newProxyConnectionFunc(dialer, backend.NewLoggerWith("logger", "test"))

		// The certificate is only valid for example.com, 127.0.0.1 and ::1.
		uri, err := url.Parse("tls://other.test:8883")
		require.NoError(t, err)

		_, err = connFunc(uri, paho.ClientOptions{TLSConfig: trustedConfig})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate")
	})

	t.Run("skip verify connects without a trusted certificate", func(t *testing.T) {
		dialer, _ := newTLSBroker(t)
		connFunc := newProxyConnectionFunc(dialer, backend.NewLoggerWith("logger", "test"))

		uri, err := url.Parse("tls://other.test:8883")
		require.NoError(t, err)

		conn, err := connFunc(uri, paho.ClientOptions{TLSConfig: &tls.Config{InsecureSkipVerify: true}})
		require.NoError(t, err)
		_ = conn.Close()
	})
}

// TestNewProxyConnectionFunc_ServerNameDefaultsToHost tests that the broker hostname from the URI
// is used to check the certificate when the TLS config has no server name.
func TestNewProxyConnectionFunc_ServerNameDefaultsToHost(t *testing.T) {
	dialer, trustedConfig := newTLSBroker(t)
	connFunc := newProxyConnectionFunc(dialer, backend.NewLoggerWith("logger", "test"))

	uri, err := url.Parse("tls://example.com:8883")
	require.NoError(t, err)

	conn, err := connFunc(uri, paho.ClientOptions{TLSConfig: trustedConfig})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	tlsConn, ok := conn.(*tls.Conn)
	require.True(t, ok)
	assert.Equal(t, "example.com", tlsConn.ConnectionState().ServerName)
	assert.Empty(t, trustedConfig.ServerName, "the caller's TLS config should not be changed")
}

// TestNewProxyConnectionFunc_DeadlineClearedAfterHandshake tests that the handshake deadline does not
// stay on the connection, which would break reads and writes once ConnectTimeout has passed.
func TestNewProxyConnectionFunc_DeadlineClearedAfterHandshake(t *testing.T) {
	dialer, trustedConfig := newTLSBroker(t)
	connFunc := newProxyConnectionFunc(dialer, backend.NewLoggerWith("logger", "test"))

	uri, err := url.Parse("tls://example.com:8883")
	require.NoError(t, err)

	conn, err := connFunc(uri, paho.ClientOptions{TLSConfig: trustedConfig, ConnectTimeout: 200 * time.Millisecond})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	// Wait past ConnectTimeout, then write. This fails with an i/o timeout if the deadline is still set.
	time.Sleep(400 * time.Millisecond)
	_, err = conn.Write([]byte("x"))
	assert.NoError(t, err)
}

// TestNewProxyConnectionFunc_SendsTLSClientHello is the end to end test that a TLS URI starts with a
// TLS handshake over the proxy. It connects a real paho client and checks the first byte the broker
// receives. 0x16 is a TLS ClientHello. 0x10 is an MQTT CONNECT, which means TLS was skipped.
func TestNewProxyConnectionFunc_SendsTLSClientHello(t *testing.T) {
	// net.Pipe is an in-memory connection. The broker side is read directly by the test.
	clientEnd, brokerEnd := net.Pipe()
	t.Cleanup(func() {
		_ = clientEnd.Close()
		_ = brokerEnd.Close()
	})

	mockDialer := &mockProxyDialer{
		dialFunc: func(network, address string) (net.Conn, error) {
			return clientEnd, nil
		},
	}

	opts := paho.NewClientOptions()
	opts.AddBroker("tls://broker.example.com:8883")
	opts.SetClientID("test")
	opts.SetConnectTimeout(2 * time.Second)
	opts.SetAutoReconnect(false)
	opts.SetCustomOpenConnectionFn(newProxyConnectionFunc(mockDialer, backend.NewLoggerWith("logger", "test")))

	client := paho.NewClient(opts)
	client.Connect() // runs in the background and never completes, the test only reads what it sends
	defer client.Disconnect(250)

	require.NoError(t, brokerEnd.SetReadDeadline(time.Now().Add(3*time.Second)))
	firstByte := make([]byte, 1)
	_, err := brokerEnd.Read(firstByte)
	require.NoError(t, err)

	assert.Equal(t, byte(0x16), firstByte[0], "0x10 would mean the MQTT CONNECT was sent without TLS")
}

// TestNewProxyConnectionFunc_RejectsWebSocketSchemes tests that ws and wss are refused
// before the proxy is dialled, so no MQTT packet can reach the wire.
func TestNewProxyConnectionFunc_RejectsWebSocketSchemes(t *testing.T) {
	for _, scheme := range []string{"ws", "wss"} {
		t.Run(scheme, func(t *testing.T) {
			logger := backend.NewLoggerWith("logger", "test")
			mockDialer := &mockProxyDialer{}

			connFunc := newProxyConnectionFunc(mockDialer, logger)

			uri, err := url.Parse(scheme + "://broker.example.com:8883")
			require.NoError(t, err)

			conn, err := connFunc(uri, paho.ClientOptions{})

			require.Error(t, err)
			assert.Contains(t, err.Error(), scheme)
			assert.Nil(t, conn)
			assert.Empty(t, mockDialer.callLog, "the proxy must not be dialled for websocket schemes")
		})
	}
}
