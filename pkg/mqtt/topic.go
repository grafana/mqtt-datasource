package mqtt

import (
	"path"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type Message struct {
	Timestamp time.Time
	Value     []byte
}

// Topic represents a MQTT topic.
type Topic struct {
	Path         string         `json:"topic"`
	Payload      map[string]any `json:"payload,omitempty"`
	ResponsePath string         `json:"response,omitempty"`
	Interval     time.Duration
	Messages     []Message
	framer       *framer
}

// Key returns the key for the topic.
// The key is a combination of the interval string and the path.
// For example, if the path is "my/topic" and the interval is 1s, the key will be "1s/my/topic".
func (t *Topic) Key() string {
	return path.Join(t.Interval.String(), t.Path)
}

// ToDataFrame converts the topic to a data frame.
func (t *Topic) ToDataFrame() (*data.Frame, error) {
	if t.framer == nil {
		t.framer = newFramer()
	}
	return t.framer.toFrame(t.Messages)
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
	tm.Map.Range(func(key, t any) bool {
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

	tm.Map.Range(func(key, t any) bool {
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

// replace all __PLUS__ with + and one __HASH__ with #
// Question: Why does grafana not allow + and # in query?
func resolveTopic(topic string) string {
	resolvedTopic := strings.ReplaceAll(topic, "__PLUS__", "+")
	return strings.Replace(resolvedTopic, "__HASH__", "#", -1)
}
