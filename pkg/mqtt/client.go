package mqtt

import (
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
