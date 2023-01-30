package mqtt

import (
	"path"
	"regexp"
	"strings"
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
		if TopicMatches(topic.Path, path) {
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

func TopicMatches(topic string, intopic string) bool {
	var regex = strings.Replace(topic, "/__WILDCARD__", "/(.*)", -1)
	m1 := regexp.MustCompile(regex)
	var grp = m1.ReplaceAllString(intopic, "$1")
	if grp == "" {
		return intopic == topic
	}
	var intopic2 = strings.Replace(intopic, grp, "__WILDCARD__", -1)

	return strings.Compare(topic, intopic2) == 0
}
