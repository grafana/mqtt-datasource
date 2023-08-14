package mqtt

import (
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

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
	URI      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type client struct {
	client paho.Client
	topics TopicMap
}

// Adapted from https://github.com/eclipse/paho.mqtt.golang/blob/master/cmd/ssl/main.go
// Also see https://www.eclipse.org/paho/clients/golang/
// https://adrianhesketh.com/2019/11/04/aws-iot-with-go/
func NewTLSConfig() (config *tls.Config, err error) {
	// Import trusted certificates from CAfile.pem.
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile("/home/kxm613/mqtt-datasource/pkg/mqtt/AmazonRootCA1.pem")
	if err != nil {
		fmt.Errorf("ReadFile: %v", err)
		return
	}
	certpool.AppendCertsFromPEM(pemCerts)

	// Import client certificate/key pair.
	cert, err := tls.LoadX509KeyPair(
		"/home/kxm613/mqtt-datasource/pkg/mqtt/mqtt5.certificate.pem",
		"/home/kxm613/mqtt-datasource/pkg/mqtt/mqtt5.private.key")

	if err != nil {
		fmt.Errorf("failedLoadX509KeyPair: %v", err)
		return
	}

	// Create tls.Config with desired tls properties
	config = &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}
	return
}

func NewClient(o Options) (Client, error) {
	tlsconfig, err := NewTLSConfig()

	if err != nil {
		fmt.Errorf("failed to create TLS configuration: %v", err)
	}
	
	opts := paho.NewClientOptions()
	opts.AddBroker(o.URI)
	opts.SetClientID(fmt.Sprintf("grafana_%d", rand.Int())).SetTLSConfig(tlsconfig)

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
	// for testing
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
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

	log.DefaultLogger.Debug("Subscribing to MQTT topic", "topic", t.Path)
	if token := c.client.Subscribe(t.Path, 0, c.HandleMessage); token.Wait() && token.Error() != nil {
		log.DefaultLogger.Error("Error subscribing to MQTT topic", "topic", t.Path, "error", token.Error())
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
