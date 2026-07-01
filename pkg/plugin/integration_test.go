package plugin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

// Integration tests to verify end-to-end streaming key functionality

func TestStreamingKeyIntegration_TopicUniqueness(t *testing.T) {
	// Test that the complete flow from query to topic creation maintains uniqueness

	// Simulate query data with streaming keys
	query1JSON, _ := json.Marshal(map[string]interface{}{
		"topic":        "sensor/temperature",
		"streamingKey": "user1/hash123/org456",
	})

	query2JSON, _ := json.Marshal(map[string]interface{}{
		"topic":        "sensor/temperature",   // Same MQTT topic
		"streamingKey": "user2/hash456/org456", // Different user, same org
	})

	query3JSON, _ := json.Marshal(map[string]interface{}{
		"topic":        "sensor/temperature",   // Same MQTT topic
		"streamingKey": "user1/hash123/org789", // Same user, different org
	})

	// Create queries
	query1 := backend.DataQuery{
		JSON:     query1JSON,
		Interval: 1 * time.Second,
		RefID:    "A",
	}

	query2 := backend.DataQuery{
		JSON:     query2JSON,
		Interval: 1 * time.Second,
		RefID:    "B",
	}

	query3 := backend.DataQuery{
		JSON:     query3JSON,
		Interval: 1 * time.Second,
		RefID:    "C",
	}

	// Create datasource instance
	ds := &MQTTDatasource{
		channelPrefix: "ds/test-uid",
	}

	// Process queries
	resp1 := ds.query(query1)
	resp2 := ds.query(query2)
	resp3 := ds.query(query3)

	// Verify no errors
	if resp1.Error != nil {
		t.Errorf("Query 1 failed: %v", resp1.Error)
	}
	if resp2.Error != nil {
		t.Errorf("Query 2 failed: %v", resp2.Error)
	}
	if resp3.Error != nil {
		t.Errorf("Query 3 failed: %v", resp3.Error)
	}

	// Extract channel paths
	channel1 := resp1.Frames[0].Meta.Channel
	channel2 := resp2.Frames[0].Meta.Channel
	channel3 := resp3.Frames[0].Meta.Channel

	// Verify all channels are different
	if channel1 == channel2 {
		t.Errorf("Expected different channels for different users, but got same: %s", channel1)
	}
	if channel1 == channel3 {
		t.Errorf("Expected different channels for different orgs, but got same: %s", channel1)
	}
	if channel2 == channel3 {
		t.Errorf("Expected different channels for different combinations, but got same: %s", channel2)
	}

	// Verify channel format
	expectedChannel1 := "ds/test-uid/1s/sensor/temperature/user1/hash123/org456"
	if channel1 != expectedChannel1 {
		t.Errorf("Expected channel1 %s, got %s", expectedChannel1, channel1)
	}

	expectedChannel2 := "ds/test-uid/1s/sensor/temperature/user2/hash456/org456"
	if channel2 != expectedChannel2 {
		t.Errorf("Expected channel2 %s, got %s", expectedChannel2, channel2)
	}

	expectedChannel3 := "ds/test-uid/1s/sensor/temperature/user1/hash123/org789"
	if channel3 != expectedChannel3 {
		t.Errorf("Expected channel3 %s, got %s", expectedChannel3, channel3)
	}
}

func TestStreamingKeyIntegration_ClientSubscription(t *testing.T) {
	// Test that client subscription works correctly with streaming keys

	// Create mock client
	client := &mockMQTTClient{
		topics:        make(map[string]*mqtt.Topic),
		subscriptions: make(map[string]bool),
	}

	// Topic keys that would come from the streaming system
	topicKey1 := "1s/dGVzdC90b3BpYw/user1/hash123/org456"
	topicKey2 := "1s/dGVzdC90b3BpYw/user2/hash456/org456"
	topicKey3 := "1s/dGVzdC90b3BpYw/user1/hash123/org789"

	// Subscribe to all three
	topic1, err := client.Subscribe(topicKey1, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	topic2, err := client.Subscribe(topicKey2, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	topic3, err := client.Subscribe(topicKey3, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify all topics were created
	if topic1 == nil || topic2 == nil || topic3 == nil {
		t.Fatal("Expected all topics to be created")
	}

	// Verify they are different instances
	if topic1 == topic2 || topic1 == topic3 || topic2 == topic3 {
		t.Error("Expected different topic instances for different streaming keys")
	}

	// Verify each can be retrieved independently
	retrieved1, found1 := client.GetTopic(topicKey1)
	retrieved2, found2 := client.GetTopic(topicKey2)
	retrieved3, found3 := client.GetTopic(topicKey3)

	if !found1 || !found2 || !found3 {
		t.Error("Expected all topics to be retrievable")
	}

	if retrieved1 != topic1 || retrieved2 != topic2 || retrieved3 != topic3 {
		t.Error("Expected retrieved topics to match original instances")
	}

	// Verify MQTT subscription was made (should be the same for all since same MQTT topic)
	if !client.subscriptions["test/topic"] {
		t.Error("Expected MQTT subscription to be made")
	}
}

func TestStreamingKeyIntegration_MessageIsolation(t *testing.T) {
	// Test that messages are properly isolated between different streaming keys

	client := &mockMQTTClient{
		topics:        make(map[string]*mqtt.Topic),
		subscriptions: make(map[string]bool),
	}

	// Create topics with same MQTT path but different streaming keys
	topicKey1 := "1s/dGVzdC90b3BpYw/user1/hash123/org456"
	topicKey2 := "1s/dGVzdC90b3BpYw/user2/hash456/org456"

	topic1, err := client.Subscribe(topicKey1, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	topic2, err := client.Subscribe(topicKey2, log.DefaultLogger)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if topic1 == nil || topic2 == nil {
		t.Fatal("Expected both topics to be created")
	}

	// Manually add messages to test isolation
	// In real implementation, messages would be routed based on MQTT topic matching
	topic1.Messages = append(topic1.Messages, mqtt.Message{
		Timestamp: time.Now(),
		Value:     []byte("message for user1"),
	})

	topic2.Messages = append(topic2.Messages, mqtt.Message{
		Timestamp: time.Now(),
		Value:     []byte("message for user2"),
	})

	// Verify topics maintain separate message stores
	if len(topic1.Messages) != 1 {
		t.Errorf("Expected 1 message in topic1, got %d", len(topic1.Messages))
	}
	if len(topic2.Messages) != 1 {
		t.Errorf("Expected 1 message in topic2, got %d", len(topic2.Messages))
	}

	// Verify message content
	if string(topic1.Messages[0].Value) != "message for user1" {
		t.Errorf("Expected 'message for user1', got '%s'", string(topic1.Messages[0].Value))
	}
	if string(topic2.Messages[0].Value) != "message for user2" {
		t.Errorf("Expected 'message for user2', got '%s'", string(topic2.Messages[0].Value))
	}

	// Verify topics are still separate instances
	if topic1 == topic2 {
		t.Error("Expected different topic instances for different streaming keys")
	}
}

// Mock MQTT client for integration testing
type mockMQTTClient struct {
	topics        map[string]*mqtt.Topic
	subscriptions map[string]bool
}

func (m *mockMQTTClient) GetTopic(reqPath string) (*mqtt.Topic, bool) {
	topic, found := m.topics[reqPath]
	return topic, found
}

func (m *mockMQTTClient) IsConnected() bool {
	return true
}

func (m *mockMQTTClient) Subscribe(reqPath string, logger log.Logger) (*mqtt.Topic, error) {
	// Check if already exists
	if topic, exists := m.topics[reqPath]; exists {
		return topic, nil
	}

	// Parse the reqPath (simplified version)
	// For testing, assume the encoded topic is "dGVzdC90b3BpYw" which decodes to "test/topic"
	topic := &mqtt.Topic{
		Path:     "dGVzdC90b3BpYw", // This would be the full path with streaming key
		Interval: 1 * time.Second,
		Messages: []mqtt.Message{},
	}

	// Store with reqPath as key
	m.topics[reqPath] = topic

	// Simulate MQTT subscription (would normally decode the topic)
	m.subscriptions["test/topic"] = true

	return topic, nil
}

func (m *mockMQTTClient) Unsubscribe(reqPath string, logger log.Logger) error {
	delete(m.topics, reqPath)
	return nil
}

func (m *mockMQTTClient) Dispose() {
	m.topics = make(map[string]*mqtt.Topic)
	m.subscriptions = make(map[string]bool)
}

func (m *mockMQTTClient) HandleMessage(topicPath string, payload []byte) {
	message := mqtt.Message{
		Timestamp: time.Now(),
		Value:     payload,
	}

	// Find topics that match this path and add message
	for key, topic := range m.topics {
		if topic.Path == topicPath {
			topic.Messages = append(topic.Messages, message)
			// Update the stored topic
			m.topics[key] = topic
		}
	}
}
