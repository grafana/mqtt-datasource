package mqtt

import (
	"sync"
	"time"
)

type Message struct {
	Timestamp time.Time
	Value     string
}

type Topic struct {
	path     string
	messages []Message
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

func (tm *TopicMap) Store(topic *Topic) {
	tm.Map.Store(topic.path, topic)
}

func (tm *TopicMap) Delete(path string) {
	tm.Map.Delete(path)
}
