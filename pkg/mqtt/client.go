package mqtt

import (
	"fmt"
	"math/rand"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Options struct {
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type StreamMessage struct {
	Topic string
	Value string
}

type Client struct {
	client *paho.Client
	topics TopicMap
}

func NewClient(o Options) (*Client, error) {
	opts := paho.NewClientOptions()

	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", o.Host, o.Port))
	opts.SetClientID(fmt.Sprintf("grafana_%d", rand.Int()))

	if o.Username != "" {
		opts.SetUsername(o.Username)
	}

	if o.Password != "" {
		opts.SetPassword(o.Password)
	}

	opts.SetPingTimeout(10 * time.Second)
	opts.SetKeepAlive(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		log.DefaultLogger.Error(fmt.Sprintf("MQTT Connection Lost: %s\n" + err.Error()))
	})
	opts.SetReconnectingHandler(func(c paho.Client, options *paho.ClientOptions) {
		log.DefaultLogger.Debug("MQTT Reconnecting")
	})

	log.DefaultLogger.Info("MQTT Connecting")

	client := paho.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting to MQTT broker: %s", token.Error())
	}

	return &Client{
		client: &client,
	}, nil
}

func (c *Client) IsConnected() bool {
	client := *c.client
	return client.IsConnectionOpen()
}

func (c *Client) IsSubscribed(path string) bool {
	_, ok := c.topics.Load(path)
	return ok
}

func (c *Client) Messages(path string) ([]Message, bool) {
	topic, ok := c.topics.Load(path)
	if !ok {
		return nil, ok
	}
	return topic.messages, true
}

func (c *Client) HandleMessage(client paho.Client, msg paho.Message) {
	log.DefaultLogger.Debug(fmt.Sprintf("Received MQTT Message %+v", msg))
	topic, ok := c.topics.Load(msg.Topic())

	if ok {
		// store message for query
		message := Message{
			Timestamp: time.Now(),
			Value:     string(msg.Payload()),
		}
		topic.messages = append(topic.messages, message)

		// limit the size of the retained messages
		if len(topic.messages) > 1000 {
			topic.messages = topic.messages[1:]
		}

		c.topics.Store(topic)
	}
}

func (c *Client) Subscribe(t string) {
	client := *c.client
	_, ok := c.topics.Load(t)

	if !ok {
		log.DefaultLogger.Debug(fmt.Sprintf("Subscribing to MQTT topic: %s", t))
		topic := Topic{
			path: t,
		}
		c.topics.Store(&topic)
		client.Subscribe(t, 0, c.HandleMessage)
	}
}

func (c *Client) Dispose() {
	client := *c.client
	log.DefaultLogger.Info("MQTT Disconnecting")
	client.Disconnect(250)
}
