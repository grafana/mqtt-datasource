package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
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
	Publish(string, map[string]any, string) (json.RawMessage, error)
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
	opts.SetCleanSession(false)
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
	// Check if there's already a topic with this exact key (reqPath)
	if existingTopic, ok := c.topics.Load(reqPath); ok {
		return existingTopic
	}

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

	// For MQTT subscription, we only need the actual topic path (without streaming key)
	// The streaming key is used for topic uniqueness in storage, but MQTT only cares about the topic path
	topicPath := path.Join(chunks[1:]...)

	// Create topic with the reqPath as the key for storage
	// The actual topic components will be parsed when needed
	t := &Topic{
		Path:     topicPath,
		Interval: interval,
	}

	topic, err := decodeTopic(t.Path)
	if err != nil {
		log.DefaultLogger.Error("Error decoding MQTT topic name", "encodedTopic", t.Path, "error", err)
		return nil
	}

	log.DefaultLogger.Debug("Subscribing to MQTT topic", "topic", topic)

	if token := c.client.Subscribe(topic, 0, func(_ paho.Client, m paho.Message) {
		// by wrapping HandleMessage we can directly get the correct topicPath for the incoming topic
		// and don't need to regex it against + and #.
		c.HandleMessage(topicPath, []byte(m.Payload()))
	}); token.Wait() && token.Error() != nil {
		log.DefaultLogger.Error("Error subscribing to MQTT topic", "topic", topic, "error", token.Error())
	}
	// Store the topic using reqPath as the key (which includes streaming key)
	c.topics.Map.Store(reqPath, t)
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

	topic, err := decodeTopic(t.Path)
	if err != nil {
		log.DefaultLogger.Error("Error decoding MQTT topic name", "encodedTopic", t.Path, "error", err)
		return
	}

	if token := c.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		log.DefaultLogger.Error("Error unsubscribing from MQTT topic", "topic", t.Path, "error", token.Error())
	}
}

func (c *client) Publish(topic string, payload map[string]any, responseTopic string) (json.RawMessage, error) {
	var response json.RawMessage
	var err error
	done := make(chan struct{}, 1)

	if responseTopic != "" {
		tokenSub := c.client.Subscribe(responseTopic, 2, func(c paho.Client, m paho.Message) {
			response = m.Payload()
			done <- struct{}{}
		})

		if !tokenSub.WaitTimeout(time.Second) && tokenSub.Error() != nil {
			err = errors.Join(err, tokenSub.Error())
			return response, err
		}

		defer c.client.Unsubscribe(responseTopic)
	} else {
		done <- struct{}{}
	}

	data, errMarshal := json.Marshal(&payload)
	if errMarshal != nil {
		err = errors.Join(err, errMarshal)
		return response, err
	}

	token := c.client.Publish(topic, 2, false, data)

	if token.Error() != nil {
		err = errors.Join(err, token.Error())
		return response, err
	}

	if !token.WaitTimeout(time.Second) {
		err = errors.Join(err, errors.New("publish timeout"))
		return response, err
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		err = errors.Join(err, errors.New("subscribe timeout"))
	}

	return response, err
}

func (c *client) Dispose() {
	log.DefaultLogger.Info("MQTT Disconnecting")
	c.client.Disconnect(250)
}
