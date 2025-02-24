package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Client interface {
	GetTopic(string) (*Topic, bool)
	IsConnected() bool
	Subscribe(string) *Topic
	Unsubscribe(string)
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

func NewClient(o Options) (Client, error) {
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
			return nil, fmt.Errorf("failed to setup TLSClientCert: %w", err)
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
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		log.DefaultLogger.Error("MQTT Connection lost", "error", err)
	})
	opts.SetReconnectingHandler(func(c paho.Client, options *paho.ClientOptions) {
		log.DefaultLogger.Debug("MQTT Reconnecting")
	})

	log.DefaultLogger.Info("MQTT Connecting", "clientID", clientID)

	pahoClient := paho.NewClient(opts)
	if token := pahoClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting to MQTT broker: %s", token.Error())
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

func (c *client) Subscribe(reqPath string) *Topic {
	chunks := strings.Split(reqPath, "/")
	if len(chunks) < 2 {
		log.DefaultLogger.Error("Invalid path", "path", reqPath)
		return nil
	}
	interval, err := time.ParseDuration(chunks[0])
	if err != nil {
		log.DefaultLogger.Error("Invalid interval", "path", reqPath, "interval", chunks[0])
		return nil
	}

	topicPath := path.Join(chunks[1:]...)
	t := &Topic{
		Path:     topicPath,
		Interval: interval,
	}
	if t, ok := c.topics.Load(topicPath); ok {
		return t
	}

	log.DefaultLogger.Debug("Subscribing to MQTT topic", "topic", topicPath)

	topic := resolveTopic(t.Path)

	if token := c.client.Subscribe(topic, 0, func(_ paho.Client, m paho.Message) {
		// by wrapping HandleMessage we can directly get the correct topicPath for the incoming topic
		// and don't need to regex it against + and #.
		c.HandleMessage(topicPath, []byte(m.Payload()))
	}); token.Wait() && token.Error() != nil {
		log.DefaultLogger.Error("Error subscribing to MQTT topic", "topic", topicPath, "error", token.Error())
	}
	c.topics.Store(t)
	return t
}

func (c *client) Unsubscribe(reqPath string) {
	t, ok := c.GetTopic(reqPath)
	if !ok {
		return
	}
	c.topics.Delete(t.Key())

	if exists := c.topics.HasSubscription(t.Path); exists {
		// There are still other subscriptions to this path,
		// so we shouldn't unsubscribe yet.
		return
	}

	log.DefaultLogger.Debug("Unsubscribing from MQTT topic", "topic", t.Path)

	topic := resolveTopic(t.Path)
	if token := c.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		log.DefaultLogger.Error("Error unsubscribing from MQTT topic", "topic", t.Path, "error", token.Error())
	}
}

func (c *client) Dispose() {
	log.DefaultLogger.Info("MQTT Disconnecting")
	c.client.Disconnect(250)
}
