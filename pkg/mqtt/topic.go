package mqtt

import (
	"sync"
	"time"
)

type Message struct {
	timestamp time.Time
	value     string
}

type TopicMap struct {
	sync.Map
}

func (tm *TopicMap) Load(topic string) ([]Message, bool) {
	m, ok := tm.Map.Load(topic)
	if !ok {
		return nil, false
	}

	messages, ok := m.([]Message)
	return messages, ok
}

func (tm *TopicMap) Store(topic string, messages []Message) {
	tm.Map.Store(topic, messages)
}
