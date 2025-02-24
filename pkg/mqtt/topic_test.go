package mqtt

import (
	"encoding/base64"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTopicMap_Load(t *testing.T) {
	t.Run("loads topic by key", func(t *testing.T) {
		topic := &Topic{
			Path:     "test",
			Interval: time.Second,
			Messages: []Message{},
		}

		tm := &TopicMap{
			Map: sync.Map{},
		}
		tm.Map.Store(topic.Key(), topic)

		actual, ok := tm.Load(topic.Key())
		require.True(t, ok)
		require.Equal(t, topic, actual)
	})
}

func TestTopicMap_Store(t *testing.T) {
	t.Run("stores new topic", func(t *testing.T) {
		topic := &Topic{
			Path:     "test",
			Interval: time.Second,
			Messages: []Message{},
		}

		tm := &TopicMap{
			Map: sync.Map{},
		}

		_, ok := tm.Map.Load(topic.Key())
		require.False(t, ok)

		tm.Store(topic)

		actual, ok := tm.Map.Load(topic.Key())
		require.True(t, ok)
		require.Equal(t, topic, actual.(*Topic))
	})
}

func TestTopicMap_AddMessage(t *testing.T) {
	t.Run("adds message to existing topics by path", func(t *testing.T) {
		topic_1s := &Topic{
			Path:     "test",
			Interval: time.Second,
			Messages: []Message{},
		}

		topic_2s := &Topic{
			Path:     "test",
			Interval: 2 * time.Second,
			Messages: []Message{},
		}

		tm := &TopicMap{
			Map: sync.Map{},
		}
		tm.Store(topic_1s)
		tm.Store(topic_2s)

		message := Message{
			Timestamp: time.Now(),
			Value:     []byte("test"),
		}

		tm.AddMessage("test", message)

		actual, ok := tm.Load(topic_1s.Key())
		require.True(t, ok)
		require.Equal(t, message, actual.Messages[0])

		actual, ok = tm.Load(topic_2s.Key())
		require.True(t, ok)
		require.Equal(t, message, actual.Messages[0])
	})
}

func TestTopicMap_Delete(t *testing.T) {
	t.Run("deletes topic by key", func(t *testing.T) {
		topic_1s := &Topic{
			Path:     "test",
			Interval: time.Second,
			Messages: []Message{},
		}

		topic_2s := &Topic{
			Path:     "test",
			Interval: 2 * time.Second,
			Messages: []Message{},
		}

		tm := &TopicMap{
			Map: sync.Map{},
		}
		tm.Store(topic_1s)
		tm.Store(topic_2s)

		_, ok := tm.Load(topic_1s.Key())
		require.True(t, ok)

		_, ok = tm.Load(topic_2s.Key())
		require.True(t, ok)

		tm.Delete(topic_1s.Key())

		_, ok = tm.Load(topic_1s.Key())
		require.False(t, ok)

		_, ok = tm.Load(topic_2s.Key())
		require.True(t, ok)
	})
}

func TestTopicMap_HasSubscription(t *testing.T) {
	t.Run("should return true if matching path exists in map", func(t *testing.T) {
		topic_1s := &Topic{
			Path:     "test",
			Interval: time.Second,
			Messages: []Message{},
		}

		topic_2s := &Topic{
			Path:     "test",
			Interval: 2 * time.Second,
			Messages: []Message{},
		}

		tm := &TopicMap{
			Map: sync.Map{},
		}
		tm.Store(topic_1s)
		tm.Store(topic_2s)

		require.True(t, tm.HasSubscription("test"))
		tm.Delete(topic_1s.Key())
		require.True(t, tm.HasSubscription("test"))
		tm.Delete(topic_2s.Key())
		require.False(t, tm.HasSubscription("test"))
	})

	t.Run("should not match", func(t *testing.T) {
		topic := &Topic{
			Path:     "test",
			Interval: time.Second,
			Messages: []Message{},
		}

		tm := &TopicMap{
			Map: sync.Map{},
		}
		tm.Store(topic)

		require.False(t, tm.HasSubscription("testing"))
	})
}

func TestDecodeTopic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expTopic string
		expError bool
	}{
		{
			name:     "Valid encoded string",
			input:    base64.RawURLEncoding.EncodeToString([]byte("$test/topic/#")),
			expTopic: "$test/topic/#",
			expError: false,
		},
		{
			name:     "Invalid encoded string",
			input:    "invalid_@_base64",
			expError: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expError: false,
		},
		{
			name:     "Valid encoded string with padding",
			input:    base64.URLEncoding.EncodeToString([]byte("test/topic")),
			expTopic: "",
			expError: true, // base64.RawURLEncoding does not accept padding
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, err := decodeTopic(tt.input)

			if tt.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expTopic, topic)
			}
		})
	}
}
