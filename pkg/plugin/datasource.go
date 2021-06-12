package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

// NewMQTTDatasource creates a new datasource instance.
func NewMQTTInstance(s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	settings, err := getDatasourceSettings(s)
	if err != nil {
		return nil, err
	}

	client, err := mqtt.NewClient(*settings)
	if err != nil {
		return nil, err
	}

	return NewMQTTDatasource(client, s.UID), nil
}

func getDatasourceSettings(s backend.DataSourceInstanceSettings) (*mqtt.Options, error) {
	settings := &mqtt.Options{}

	if err := json.Unmarshal(s.JSONData, settings); err != nil {
		return nil, err
	}

	if password, exists := s.DecryptedSecureJSONData["password"]; exists {
		settings.Password = password
	}

	return settings, nil
}

type MQTTClient interface {
	Stream() chan mqtt.StreamMessage
	IsConnected() bool
	IsSubscribed(topic string) bool
	Messages(topic string) ([]mqtt.Message, bool)
	Subscribe(topic string)
	Unsubscribe(topic string)
}

type MQTTDatasource struct {
	Client        MQTTClient
	channelPrefix string
}

// Make sure MQTTDatasource implements required interfaces.
// This is important to do since otherwise we will only get a
// not implemented error response from plugin in runtime.
var (
	_ backend.QueryDataHandler      = (*MQTTDatasource)(nil)
	_ backend.CheckHealthHandler    = (*MQTTDatasource)(nil)
	_ backend.StreamHandler         = (*MQTTDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*MQTTDatasource)(nil)
)

// NewMQTTDatasource creates a new datasource instance.
func NewMQTTDatasource(client MQTTClient, uid string) *MQTTDatasource {
	return &MQTTDatasource{
		Client:        client,
		channelPrefix: fmt.Sprintf("ds/%s/", uid),
	}
}

// Dispose here tells plugin SDK that plugin wants to clean up resources
// when a new instance created. As soon as datasource settings change detected
// by SDK old datasource instance will be disposed and a new one will be created
// using NewMQTTDatasource factory function.
func (ds *MQTTDatasource) Dispose() {
	// Nothing to clean up yet.
}

func (ds *MQTTDatasource) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := ds.Query(q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (ds *MQTTDatasource) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if !ds.Client.IsConnected() {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "MQTT Disconnected",
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "MQTT Connected",
	}, nil
}

func (ds *MQTTDatasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	return &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusOK,
	}, nil
}

func (ds *MQTTDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	ds.Client.Subscribe(req.Path)
	defer ds.Client.Unsubscribe(req.Path)

	for {
		select {
		case <-ctx.Done():
			backend.Logger.Info("stop streaming (context canceled)")
			return nil
		case message := <-ds.Client.Stream():
			if message.Topic != req.Path {
				continue
			}
			err := ds.SendMessage(message, req, sender)
			if err != nil {
				log.DefaultLogger.Error(fmt.Sprintf("unable to send message: %s", err.Error()))
			}
		}
	}
}

func (ds *MQTTDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}

type queryModel struct {
	Topic string `json:"queryText"`
}

func (ds *MQTTDatasource) Query(query backend.DataQuery) backend.DataResponse {
	var qm queryModel

	response := backend.DataResponse{}
	response.Error = json.Unmarshal(query.JSON, &qm)

	if response.Error != nil {
		return response
	}

	// ensure the client is subscribed to the topic.
	ds.Client.Subscribe(qm.Topic)

	messages, ok := ds.Client.Messages(qm.Topic)
	if !ok {
		return response
	}

	frame := ToFrame(qm.Topic, messages)

	if qm.Topic != "" {
		frame.SetMeta(&data.FrameMeta{
			Channel: ds.channelPrefix + qm.Topic,
		})
	}

	response.Frames = append(response.Frames, frame)
	return response
}

func (ds *MQTTDatasource) SendMessage(msg mqtt.StreamMessage, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	if !ds.Client.IsSubscribed(req.Path) {
		return nil
	}

	message := mqtt.Message{
		Timestamp: time.Now(),
		Value:     msg.Value,
	}

	frame := ToFrame(msg.Topic, []mqtt.Message{message})

	log.DefaultLogger.Debug(fmt.Sprintf("Sending message to client for topic %s", msg.Topic))
	return sender.SendFrame(frame, data.IncludeAll)
}
