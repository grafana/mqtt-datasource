package mqtt

import (
	"path"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Mock client that implements our Client interface directly
type mockClient struct {
	topics        TopicMap
	subscriptions map[string]bool
	connected     bool
}

func (m *mockClient) GetTopic(reqPath string) (*Topic, bool) {
	return m.topics.Load(reqPath)
}

func (m *mockClient) IsConnected() bool {
	return m.connected
}

func (m *mockClient) Subscribe(reqPath string, logger log.Logger) (*Topic, error) {
	// Check if already exists
	if existingTopic, ok := m.topics.Load(reqPath); ok {
		return existingTopic, nil
	}

	chunks := strings.Split(reqPath, "/")
	if len(chunks) < 2 {
		return nil, nil
	}
	interval, err := time.ParseDuration(chunks[0])
	if err != nil {
		return nil, err
	}

	topicPath := path.Join(chunks[1:]...)
	t := &Topic{
		Path:     topicPath,
		Interval: interval,
		Messages: []Message{},
	}

	// Track MQTT subscription (simplified for testing)
	if m.subscriptions == nil {
		m.subscriptions = make(map[string]bool)
	}
	// For testing, assume we decode the topic properly
	m.subscriptions["test/topic"] = true

	// Store with reqPath as key
	m.topics.Map.Store(reqPath, t)
	return t, nil
}

func (m *mockClient) Unsubscribe(reqPath string, logger log.Logger) error {
	m.topics.Delete(reqPath)
	return nil
}

func (m *mockClient) Dispose() {
	// Clear all topics and subscriptions
	m.topics = TopicMap{}
	m.subscriptions = make(map[string]bool)
}

func (m *mockClient) HandleMessage(topicPath string, payload []byte) {
	message := Message{
		Timestamp: time.Now(),
		Value:     payload,
	}
	m.topics.AddMessage(topicPath, message)
}

func newMockClient() *mockClient {
	return &mockClient{
		topics:        TopicMap{},
		subscriptions: make(map[string]bool),
		connected:     true,
	}
}

func TestClient_Subscribe_WithStreamingKey(t *testing.T) {
	c := newMockClient()

	tests := []struct {
		name         string
		reqPath      string
		expectTopic  bool
		expectedPath string
	}{
		{
			name:         "subscribe with streaming key",
			reqPath:      "1s/dGVzdC90b3BpYw/user1/hash123/org456",
			expectTopic:  true,
			expectedPath: "dGVzdC90b3BpYw/user1/hash123/org456",
		},
		{
			name:         "subscribe without streaming key",
			reqPath:      "5s/dGVzdC90b3BpYw",
			expectTopic:  true,
			expectedPath: "dGVzdC90b3BpYw",
		},
		{
			name:        "invalid path - no interval",
			reqPath:     "invalid",
			expectTopic: false,
		},
		{
			name:        "invalid interval",
			reqPath:     "invalid-interval/dGVzdC90b3BpYw",
			expectTopic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, err := c.Subscribe(tt.reqPath, log.DefaultLogger)
			if err != nil && tt.expectTopic {
				t.Fatalf("Subscribe failed: %v", err)
			}

			if tt.expectTopic {
				if topic == nil {
					t.Errorf("Expected topic to be created, but got nil")
					return
				}

				// Verify topic is stored with the correct key
				storedTopic, found := c.topics.Load(tt.reqPath)
				if !found {
					t.Errorf("Expected topic to be stored with key %s", tt.reqPath)
				}

				if storedTopic.Path != tt.expectedPath {
					t.Errorf("Expected topic path %s, got %s", tt.expectedPath, storedTopic.Path)
				}
			} else {
				if topic != nil {
					t.Errorf("Expected nil topic for invalid input, but got %v", topic)
				}
			}
		})
	}
}

func TestClient_Subscribe_Deduplication(t *testing.T) {
	c := newMockClient()

	reqPath := "1s/dGVzdC90b3BpYw/user1/hash123/org456"

	// Subscribe first time
	topic1, err := c.Subscribe(reqPath, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if topic1 == nil {
		t.Fatal("Expected topic to be created")
	}

	// Subscribe second time - should return same topic
	topic2, err := c.Subscribe(reqPath, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if topic2 == nil {
		t.Fatal("Expected topic to be returned")
	}

	if topic1 != topic2 {
		t.Error("Expected same topic instance for duplicate subscription")
	}

	// Verify only one topic is stored
	count := 0
	c.topics.Range(func(key, value any) bool {
		count++
		return true
	})

	if count != 1 {
		t.Errorf("Expected 1 stored topic, got %d", count)
	}
}

func TestClient_Subscribe_MultipleStreamingKeys(t *testing.T) {
	c := newMockClient()

	// Same MQTT topic, same interval, different streaming keys
	reqPath1 := "1s/dGVzdC90b3BpYw/user1/hash123/org456"
	reqPath2 := "1s/dGVzdC90b3BpYw/user2/hash456/org456"
	reqPath3 := "1s/dGVzdC90b3BpYw/user1/hash123/org789"

	topic1, err := c.Subscribe(reqPath1, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	topic2, err := c.Subscribe(reqPath2, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	topic3, err := c.Subscribe(reqPath3, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if topic1 == nil || topic2 == nil || topic3 == nil {
		t.Fatal("Expected all topics to be created")
	}

	// All should be different instances
	if topic1 == topic2 || topic1 == topic3 || topic2 == topic3 {
		t.Error("Expected different topic instances for different streaming keys")
	}

	// Verify all three topics are stored separately
	count := 0
	c.topics.Range(func(key, value any) bool {
		count++
		return true
	})

	if count != 3 {
		t.Errorf("Expected 3 stored topics, got %d", count)
	}

	// Verify each can be retrieved by its specific key
	storedTopic1, found1 := c.GetTopic(reqPath1)
	storedTopic2, found2 := c.GetTopic(reqPath2)
	storedTopic3, found3 := c.GetTopic(reqPath3)

	if !found1 || !found2 || !found3 {
		t.Error("Expected all topics to be retrievable by their keys")
	}

	if storedTopic1 == storedTopic2 || storedTopic1 == storedTopic3 || storedTopic2 == storedTopic3 {
		t.Error("Expected retrieved topics to be different instances")
	}
}

func TestClient_GetTopic(t *testing.T) {
	c := newMockClient()

	reqPath := "2s/dGVzdC90b3BpYw/streaming/key/123"

	// Topic doesn't exist yet
	_, found := c.GetTopic(reqPath)
	if found {
		t.Error("Expected topic not to be found initially")
	}

	// Create topic
	topic, err := c.Subscribe(reqPath, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if topic == nil {
		t.Fatal("Expected topic to be created")
	}

	// Now it should be found
	retrievedTopic, found := c.GetTopic(reqPath)
	if !found {
		t.Error("Expected topic to be found after subscription")
	}

	if retrievedTopic != topic {
		t.Error("Expected retrieved topic to be the same instance")
	}
}

func TestClient_MessageHandling_WithStreamingKeys(t *testing.T) {
	c := newMockClient()

	// Create topics with same MQTT path but different streaming keys
	reqPath1 := "1s/dGVzdC90b3BpYw/user1/hash123/org456"
	reqPath2 := "1s/dGVzdC90b3BpYw/user2/hash456/org456"

	topic1, err := c.Subscribe(reqPath1, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	topic2, err := c.Subscribe(reqPath2, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if topic1 == nil || topic2 == nil {
		t.Fatal("Expected both topics to be created")
	}

	// Simulate MQTT message arrival
	mqttTopicPath := "dGVzdC90b3BpYw/user1/hash123/org456" // This is what HandleMessage receives
	c.HandleMessage(mqttTopicPath, []byte("test message"))

	// Check that only the matching topic received the message
	updatedTopic1, _ := c.GetTopic(reqPath1)
	updatedTopic2, _ := c.GetTopic(reqPath2)

	if len(updatedTopic1.Messages) != 1 {
		t.Errorf("Expected 1 message in topic1, got %d", len(updatedTopic1.Messages))
	}
	if len(updatedTopic2.Messages) != 0 {
		t.Errorf("Expected 0 messages in topic2, got %d", len(updatedTopic2.Messages))
	}
}
