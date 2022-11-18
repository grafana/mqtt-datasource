package mqtt

import (
	"fmt"
	"math/rand"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Client interface {
	GetTopic(path string) (*Topic, bool)
	IsConnected() bool
	Subscribe(topic *Topic)
	Unsubscribe(topic string)
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

func NewClient(o Options) (Client, error) {
	opts := paho.NewClientOptions()

	opts.AddBroker(o.URI)
	opts.SetClientID(fmt.Sprintf("grafana_%d", rand.Int()))

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
		log.DefaultLogger.Error(fmt.Sprintf("MQTT Connection Lost: %s", err.Error()))
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

func (c *client) Subscribe(t *Topic) {
	if _, ok := c.topics.Load(t.Key()); ok {
		return
	}
	log.DefaultLogger.Debug(fmt.Sprintf("Subscribing to MQTT topic: %s", t.Path))
	c.topics.Store(t)
	c.client.Subscribe(t.Path, 0, c.HandleMessage)
}

func (c *client) Unsubscribe(reqPath string) {
	t, ok := c.GetTopic(reqPath)
	if !ok {
		return
	}
	log.DefaultLogger.Debug(fmt.Sprintf("Unsubscribing from MQTT topic: %s", t.Path))
	c.client.Unsubscribe(t.Path)
	c.topics.Delete(t.Key())
}

func (c *client) Dispose() {
	log.DefaultLogger.Info("MQTT Disconnecting")
	c.client.Disconnect(250)
}
