package mqtt

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mockMQTTClient struct {
	publishTimeout   bool
	subscribeTimeout bool
	responseTimeout  bool
}

func (m *mockMQTTClient) IsConnected() bool {
	panic("not implemented") // TODO: Implement
}

func (m *mockMQTTClient) IsConnectionOpen() bool {
	panic("not implemented") // TODO: Implement
}

func (m *mockMQTTClient) Connect() mqtt.Token {
	panic("not implemented") // TODO: Implement
}

func (m *mockMQTTClient) Disconnect(quiesce uint) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	token := newToken()
	if !m.publishTimeout {
		token.done <- struct{}{}
	}
	return token
}

func (m *mockMQTTClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	token := newToken()
	if !m.subscribeTimeout {
		token.done <- struct{}{}
	}
	if !m.responseTimeout {
		callback(nil, mockMessage{})
	}
	return token
}

func (m *mockMQTTClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	panic("not implemented") // TODO: Implement
}

func (m *mockMQTTClient) Unsubscribe(topics ...string) mqtt.Token {
	newToken := newToken()
	newToken.done <- struct{}{}
	return newToken
}
func (m *mockMQTTClient) AddRoute(topic string, callback mqtt.MessageHandler) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMQTTClient) OptionsReader() mqtt.ClientOptionsReader {
	panic("not implemented") // TODO: Implement
}

type mockTocken struct {
	err  error
	done chan struct{}
}

func newToken() *mockTocken {
	return &mockTocken{done: make(chan struct{}, 1)}
}

func (t *mockTocken) Wait() bool {
	<-t.done
	return true
}

func (t *mockTocken) WaitTimeout(timeout time.Duration) bool {
	select {
	case <-t.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (t *mockTocken) Done() <-chan struct{} {
	return t.done
}

func (t *mockTocken) Error() error {
	return t.err
}

type mockMessage struct{}

func (m mockMessage) Duplicate() bool {
	panic("not implemented") // TODO: Implement
}

func (m mockMessage) Qos() byte {
	panic("not implemented") // TODO: Implement
}

func (m mockMessage) Retained() bool {
	panic("not implemented") // TODO: Implement
}

func (m mockMessage) Topic() string {
	panic("not implemented") // TODO: Implement
}

func (m mockMessage) MessageID() uint16 {
	panic("not implemented") // TODO: Implement
}

func (m mockMessage) Payload() []byte {
	return []byte(`{"response": "ok"}`)
}

func (m mockMessage) Ack() {
	panic("not implemented") // TODO: Implement
}
