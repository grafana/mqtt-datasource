package plugin

import "sync"

type RefIDs struct {
	sync.Mutex
	m map[string]string
}

func (m *RefIDs) Get(topic string) (string, bool) {
	m.Lock()
	defer m.Unlock()

	value, exists := m.m[topic]

	return value, exists
}

func (m *RefIDs) Set(topic, refID string) {
	m.Lock()
	defer m.Unlock()

	if m.m == nil {
		m.m = make(map[string]string)
	}

	m.m[topic] = refID
}
