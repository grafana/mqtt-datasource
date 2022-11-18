package mqtt

import (
	"encoding/json"
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
}

func (t *Topic) Key() string {
	return path.Join(t.Interval.String(), t.Path)
}

func (t *Topic) ToDataFrame() *data.Frame {
	ts := make([]time.Time, 0, len(t.Messages))
	value := make([]json.RawMessage, 0, len(t.Messages))

	for _, m := range t.Messages {
		ts = append(ts, m.Timestamp)
		value = append(value, m.Value)
	}

	return data.NewFrame("", data.NewField("Time", nil, ts), data.NewField(t.Path, nil, value))
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

func (tm *TopicMap) Store(topic *Topic) {
	tm.Map.Store(topic.Key(), topic)
}

func (tm *TopicMap) Delete(path string) {
	tm.Map.Delete(path)
}
