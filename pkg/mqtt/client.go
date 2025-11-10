package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Client interface {
	GetTopic(string) (*Topic, bool)
	IsConnected() bool
	Subscribe(string, log.Logger) (*Topic, error)
	Unsubscribe(string, log.Logger) error
	Dispose()
}

type Options struct {
	URI           string `json:"uri"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	ClientID      string `json:"clientID"`
	TLSCACert     string `json:"tlsCACert"`
	TLSClientCert string `json:"tlsClientCert"`
	TLSClientKey  string `json:"tlsClientKey"`
	TLSSkipVerify bool   `json:"tlsSkipVerify"`
}

type client struct {
	client paho.Client
	topics TopicMap
}

func NewClient(ctx context.Context, o Options) (Client, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	opts := paho.NewClientOptions()

	opts.AddBroker(o.URI)

	clientID := o.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("grafana_%d", rand.Int())
	}
	opts.SetClientID(clientID)

	if o.Username != "" {
		opts.SetUsername(o.Username)
	}

	if o.Password != "" {
		opts.SetPassword(o.Password)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: o.TLSSkipVerify,
	}

	if o.TLSClientCert != "" || o.TLSClientKey != "" {
		cert, err := tls.X509KeyPair([]byte(o.TLSClientCert), []byte(o.TLSClientKey))
		if err != nil {
			return nil, backend.DownstreamErrorf("failed to setup TLSClientCert: %w", err)
		}

		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	if o.TLSCACert != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(o.TLSCACert))
		tlsConfig.RootCAs = caCertPool
	}

	opts.SetTLSConfig(tlsConfig)
	opts.SetPingTimeout(60 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(false)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		logger.Warn("MQTT Connection lost", "error", err)
	})
	opts.SetReconnectingHandler(func(c paho.Client, options *paho.ClientOptions) {
		logger.Debug("MQTT Reconnecting")
	})

	logger.Info("MQTT Connecting", "clientID", clientID)

	pahoClient := paho.NewClient(opts)
	if token := pahoClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, backend.DownstreamErrorf("error connecting to MQTT broker: %s", token.Error())
	}

	return &client{
		client: pahoClient,
	}, nil
}

func (c *client) IsConnected() bool {
	return c.client.IsConnectionOpen()
}

func (c *client) HandleMessage(topic string, payload []byte) {
	message := Message{
		Timestamp: time.Now(),
		Value:     payload,
	}

	c.topics.AddMessage(topic, message)
}

func (c *client) GetTopic(reqPath string) (*Topic, bool) {
	return c.topics.Load(reqPath)
}

func (c *client) Subscribe(reqPath string, logger log.Logger) (*Topic, error) {
	// Check if there's already a topic with this exact key (reqPath)
	if existingTopic, ok := c.topics.Load(reqPath); ok {
		return existingTopic, nil
	}

	chunks := strings.Split(reqPath, "/")
	if len(chunks) < 2 {
		return nil, backend.DownstreamErrorf("invalid path: %s", reqPath)
	}
	interval, err := time.ParseDuration(chunks[0])
	if err != nil {
		return nil, backend.DownstreamErrorf("invalid interval %s: %s", chunks[0], err)
	}

	// For MQTT subscription, we only need the actual topic path (without streaming key)
	// The streaming key is used for topic uniqueness in storage, but MQTT only cares about the topic path
	topicPath := path.Join(chunks[1:]...)

	// Create topic with the reqPath as the key for storage
	// The actual topic components will be parsed when needed
	t := &Topic{
		Path:     topicPath,
		Interval: interval,
	}

	topic, err := decodeTopic(t.Path, logger)
	if err != nil {
		return nil, backend.DownstreamErrorf("error decoding MQTT topic name %s: %s", t.Path, err)
	}

	logger.Debug("Subscribing to MQTT topic", "topic", topic)

	if token := c.client.Subscribe(topic, 0, func(_ paho.Client, m paho.Message) {
		// by wrapping HandleMessage we can directly get the correct topicPath for the incoming topic
		// and don't need to regex it against + and #.
		c.HandleMessage(topicPath, []byte(m.Payload()))
	}); token.Wait() && token.Error() != nil {
		return nil, backend.DownstreamErrorf("error subscribing to MQTT topic %s: %s", topic, token.Error())
	}
	// Store the topic using reqPath as the key (which includes streaming key)
	c.topics.Map.Store(reqPath, t)
	return t, nil
}

func (c *client) Unsubscribe(reqPath string, logger log.Logger) error {
	t, ok := c.GetTopic(reqPath)
	if !ok {
		return nil // No error if topic doesn't exist
	}
	c.topics.Delete(t.Key())

	if exists := c.topics.HasSubscription(t.Path); exists {
		// There are still other subscriptions to this path,
		// so we shouldn't unsubscribe yet.
		return nil
	}

	logger.Debug("Unsubscribing from MQTT topic", "topic", t.Path)

	topic, err := decodeTopic(t.Path, logger)
	if err != nil {
		return backend.DownstreamErrorf("error decoding MQTT topic name %s: %s", t.Path, err)
	}

	if token := c.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return backend.DownstreamErrorf("error unsubscribing from MQTT topic %s: %s", t.Path, token.Error())
	}

	return nil
}

func (c *client) Dispose() {
	log.DefaultLogger.Info("MQTT Disconnecting")
	c.client.Disconnect(250)
}
