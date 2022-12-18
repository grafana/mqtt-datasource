package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
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
	CertificatePath string `json:"certificatePath"`
	RootCertPath    string `json:"rootCertPath"`
	PrivateKeyPath  string `json:"privateKeyPath"`
	URI             string `json:"uri"`
	Username        string `json:"username"`
	Password        string `json:"password"`
}

type client struct {
	client paho.Client
	topics TopicMap
}

func NewTLSConfig(o Options) (config *tls.Config, err error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(o.RootCertPath)
	if err != nil {
		return
	}
	certpool.AppendCertsFromPEM(pemCerts)

	cert, err := tls.LoadX509KeyPair(o.CertificatePath, o.PrivateKeyPath)
	if err != nil {
		return
	}

	config = &tls.Config{
		RootCAs:      certpool,
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    nil,
		Certificates: []tls.Certificate{cert},
	}
	return
}

func NewClient(o Options) (Client, error) {

	opts := paho.NewClientOptions()

	opts.AddBroker(o.URI)
	opts.SetClientID(fmt.Sprintf("grafana_%d", rand.Int()))

	if o.CertificatePath != "" {
		tlsconfig, err := NewTLSConfig(o)
		if err != nil {
			log.DefaultLogger.Error("Failed to create TLS configuration: %v", err)
		}
		opts.SetTLSConfig(tlsconfig)
	}

	if o.Username != "" {
		opts.SetUsername(o.Username)
	}

	if o.Password != "" {
		opts.SetPassword(o.Password)
	}

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

	log.DefaultLogger.Info("MQTT Connecting")

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

func (c *client) HandleMessage(_ paho.Client, msg paho.Message) {
	message := Message{
		Timestamp: time.Now(),
		Value:     msg.Payload(),
	}
	c.topics.AddMessage(msg.Topic(), message)
}

func (c *client) GetTopic(reqPath string) (*Topic, bool) {
	return c.topics.Load(reqPath)
}

func (c *client) Subscribe(reqPath string) *Topic {
	log.DefaultLogger.Info("==:>GetTopic:Subscribe")
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
	var topic = strings.Replace(t.Path, "__WILDCARD__", "#", -1)
	log.DefaultLogger.Info("Subscribing to MQTT topic", "topic", topic)
	if token := c.client.Subscribe(topic, 0, c.HandleMessage); token.Wait() && token.Error() != nil {
		log.DefaultLogger.Error("Error subscribing to MQTT topic", "topic", topic, "error", token.Error())
	}
	c.topics.Store(t)
	return t
}

func (c *client) Unsubscribe(reqPath string) {
	t, ok := c.GetTopic(reqPath)
	if !ok {
		return
	}
	log.DefaultLogger.Debug("Unsubscribing from MQTT topic", "topic", t.Path)
	if token := c.client.Unsubscribe(t.Path); token.Wait() && token.Error() != nil {
		log.DefaultLogger.Error("Error unsubscribing from MQTT topic", "topic", t.Path, "error", token.Error())
	}
	c.topics.Delete(t.Key())
}

func (c *client) Dispose() {
	log.DefaultLogger.Info("MQTT Disconnecting")
	c.client.Disconnect(250)
}
