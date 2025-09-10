package mqtt

import (
	"encoding/base64"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type Message struct {
	Timestamp time.Time
	Value     []byte
}

// Topic represents a MQTT topic.
type Topic struct {
	Path         string `json:"topic"`
	StreamingKey string `json:"streamingKey,omitempty"`
	Interval     time.Duration
	Messages     []Message
	framer       *framer
}

// Key returns the key for the topic.
// The key is a combination of the interval string, the path, and the streaming key.
// For example, if the path is "my/topic" and the interval is 1s, the key will be "1s/my/topic/streamingkey".
func (t *Topic) Key() string {
	return path.Join(t.Interval.String(), t.Path, t.StreamingKey)
}

// ToDataFrame converts the topic to a data frame.
func (t *Topic) ToDataFrame(logger log.Logger) (*data.Frame, error) {
	if t.framer == nil {
		t.framer = newFramer()
	}
	return t.framer.toFrame(t.Messages, logger)
}

// TopicMap is a thread-safe map of topics
type TopicMap struct {
	sync.Map
}

// Load returns the topic for the given topic key.
func (tm *TopicMap) Load(key string) (*Topic, bool) {
	t, ok := tm.Map.Load(key)
	if !ok {
		return nil, false
	}

	topic, ok := t.(*Topic)
	return topic, ok
}

// AddMessage adds a message to the topic for the given path.
func (tm *TopicMap) AddMessage(path string, message Message) {
	tm.Range(func(key, t any) bool {
		topic, ok := t.(*Topic)
		if !ok {
			return false
		}
		if topic.Path == path {
			topic.Messages = append(topic.Messages, message)
			tm.Store(topic)
		}
		return true
	})
}

// HasSubscription returns true if the topic map has a subscription for the given path.
func (tm *TopicMap) HasSubscription(path string) bool {
	found := false

	tm.Range(func(key, t any) bool {
		topic, ok := t.(*Topic)
		if !ok {
			return true // this shouldn't happen, but continue iterating
		}

		if topic.Path == path {
			found = true
			return false // topic found, stop iterating
		}

		return true // continue iterating
	})

	return found
}

// Store stores the topic in the map.
func (tm *TopicMap) Store(t *Topic) {
	tm.Map.Store(t.Key(), t)
}

// Delete deletes the topic for the given key.
func (tm *TopicMap) Delete(key string) {
	tm.Map.Delete(key)
}

// decodeTopic decodes an MQTT topic name from base64 URL encoding.
//
// There are some restrictions to what characters are allowed to use in a Grafana Live channel:
//
//	https://github.com/grafana/grafana-plugin-sdk-go/blob/7470982de35f3b0bb5d17631b4163463153cc204/live/channel.go#L33
//
// To comply with these restrictions, the topic is encoded using URL-safe base64
// encoding. (RFC 4648; 5. Base 64 Encoding with URL and Filename Safe Alphabet)
func decodeTopic(topicPath string, logger log.Logger) (string, error) {
	chunks := strings.Split(topicPath, "/")
	topic := chunks[0]
	logger.Debug("Decoding MQTT topic name", "encodedTopic", topic)
	decoded, err := base64.RawURLEncoding.DecodeString(topic)

	if err != nil {
		return "", err
	}

	return string(decoded), nil
}
