package mqtt

import (
	"testing"
	"time"
)

func TestTopic_Key(t *testing.T) {
	tests := []struct {
		name        string
		topic       Topic
		expectedKey string
	}{
		{
			name: "topic without streaming key",
			topic: Topic{
				Path:     "sensor/temperature",
				Interval: 1 * time.Second,
			},
			expectedKey: "1s/sensor/temperature",
		},
		{
			name: "topic with streaming key",
			topic: Topic{
				Path:         "sensor/temperature",
				Interval:     1 * time.Second,
				StreamingKey: "ds123/abc456def/789",
			},
			expectedKey: "1s/sensor/temperature/ds123/abc456def/789",
		},
		{
			name: "topic with complex path and streaming key",
			topic: Topic{
				Path:         "building/floor1/room2/sensor/temp",
				Interval:     5 * time.Second,
				StreamingKey: "datasource-uid/hash123/456",
			},
			expectedKey: "5s/building/floor1/room2/sensor/temp/datasource-uid/hash123/456",
		},
		{
			name: "topic with empty streaming key",
			topic: Topic{
				Path:         "simple/topic",
				Interval:     10 * time.Second,
				StreamingKey: "",
			},
			expectedKey: "10s/simple/topic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.topic.Key()
			if got != tt.expectedKey {
				t.Errorf("Topic.Key() = %v, want %v", got, tt.expectedKey)
			}
		})
	}
}

func TestTopic_KeyUniqueness(t *testing.T) {
	// Test that different streaming keys produce different keys
	baseTopic := Topic{
		Path:     "sensor/temp",
		Interval: 1 * time.Second,
	}

	topic1 := baseTopic
	topic1.StreamingKey = "user1/hash123/org456"

	topic2 := baseTopic
	topic2.StreamingKey = "user2/hash456/org456"

	topic3 := baseTopic
	topic3.StreamingKey = "user1/hash123/org789"

	key1 := topic1.Key()
	key2 := topic2.Key()
	key3 := topic3.Key()

	// All keys should be different
	if key1 == key2 {
		t.Errorf("Expected different keys for different users, but got same: %s", key1)
	}
	if key1 == key3 {
		t.Errorf("Expected different keys for different orgs, but got same: %s", key1)
	}
	if key2 == key3 {
		t.Errorf("Expected different keys for different combinations, but got same: %s", key2)
	}

	// Verify the actual key format
	expectedKey1 := "1s/sensor/temp/user1/hash123/org456"
	if key1 != expectedKey1 {
		t.Errorf("Topic1.Key() = %v, want %v", key1, expectedKey1)
	}
}

func TestTopicMap_Store_And_Load_WithStreamingKey(t *testing.T) {
	tm := &TopicMap{}

	topic1 := &Topic{
		Path:         "sensor/temp",
		Interval:     1 * time.Second,
		StreamingKey: "user1/hash123/org456",
	}

	topic2 := &Topic{
		Path:         "sensor/temp",          // Same path
		Interval:     1 * time.Second,        // Same interval
		StreamingKey: "user2/hash456/org456", // Different streaming key
	}

	// Store both topics
	tm.Store(topic1)
	tm.Store(topic2)

	// Load topic1
	loadedTopic1, found1 := tm.Load(topic1.Key())
	if !found1 {
		t.Errorf("Expected to find topic1 with key %s", topic1.Key())
	}
	if loadedTopic1.StreamingKey != topic1.StreamingKey {
		t.Errorf("Expected streaming key %s, got %s", topic1.StreamingKey, loadedTopic1.StreamingKey)
	}

	// Load topic2
	loadedTopic2, found2 := tm.Load(topic2.Key())
	if !found2 {
		t.Errorf("Expected to find topic2 with key %s", topic2.Key())
	}
	if loadedTopic2.StreamingKey != topic2.StreamingKey {
		t.Errorf("Expected streaming key %s, got %s", topic2.StreamingKey, loadedTopic2.StreamingKey)
	}

	// Verify they are different instances
	if loadedTopic1 == loadedTopic2 {
		t.Error("Expected different topic instances for different streaming keys")
	}
}

func TestTopicMap_AddMessage_WithStreamingKey(t *testing.T) {
	tm := &TopicMap{}

	topic1 := &Topic{
		Path:         "sensor/temp",
		Interval:     1 * time.Second,
		StreamingKey: "user1/hash123/org456",
		Messages:     []Message{},
	}

	topic2 := &Topic{
		Path:         "sensor/temp", // Same MQTT path
		Interval:     1 * time.Second,
		StreamingKey: "user2/hash456/org456", // Different streaming key
		Messages:     []Message{},
	}

	tm.Store(topic1)
	tm.Store(topic2)

	// Add message - should go to both topics since they have the same MQTT path
	message := Message{
		Timestamp: time.Now(),
		Value:     []byte("test message"),
	}

	tm.AddMessage("sensor/temp", message)

	// Check that both topics received the message
	updatedTopic1, _ := tm.Load(topic1.Key())
	updatedTopic2, _ := tm.Load(topic2.Key())

	if len(updatedTopic1.Messages) != 1 {
		t.Errorf("Expected 1 message in topic1, got %d", len(updatedTopic1.Messages))
	}
	if len(updatedTopic2.Messages) != 1 {
		t.Errorf("Expected 1 message in topic2, got %d", len(updatedTopic2.Messages))
	}
}
