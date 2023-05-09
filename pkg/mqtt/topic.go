package mqtt

import (
	"path"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type Message struct {
	Timestamp time.Time
	Value     []byte
}

type Topic struct {
	Path     string `json:"topic"`
	Interval time.Duration
	Messages []Message
	framer   *framer
}

func (t *Topic) Key() string {
	return path.Join(t.Interval.String(), t.Path)
}

func (t *Topic) ToDataFrame() (*data.Frame, error) {
	if t.framer == nil {
		t.framer = newFramer()
	}
	return t.framer.toFrame(t.Messages)
}

type TopicMap struct {
	sync.Map
}

func (tm *TopicMap) Load(path string) (*Topic, bool) {
	t, ok := tm.Map.Load(path)
	if !ok {
		return nil, false
	}

	topic, ok := t.(*Topic)
	return topic, ok
}

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

func (tm *TopicMap) Store(t *Topic) {
	tm.Map.Store(t.Key(), t)
}

func (tm *TopicMap) Delete(path string) {
	tm.Map.Delete(path)
}

func (tm *TopicMap) CheckForOtherSubscribers(topic *Topic) *Topic {
	var result *Topic = nil
	tm.Map.Range(func(key, t any) bool {
		otherTopic, ok := t.(*Topic)
		if !ok {
			return false
		} else if key == topic.Key() {
			return true
		} else if otherTopic.Path == topic.Path {
			result = otherTopic
			return false
		} else {
			return true
		}
	})
	return result
}
