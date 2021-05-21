package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

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
	closeCh       chan struct{}
}

func GetDatasourceSettings(s backend.DataSourceInstanceSettings) (*mqtt.Options, error) {
	settings := &mqtt.Options{}
	if err := json.Unmarshal(s.JSONData, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// Make sure SampleDatasource implements required interfaces.
// This is important to do since otherwise we will only get a
// not implemented error response from plugin in runtime.
var (
	_ backend.QueryDataHandler      = (*MQTTDatasource)(nil)
	_ backend.CheckHealthHandler    = (*MQTTDatasource)(nil)
	_ backend.StreamHandler         = (*MQTTDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*MQTTDatasource)(nil)
)

// NewMQTTDatasource creates a new datasource instance.
func NewMQTTDatasource(client MQTTClient, id int64) *MQTTDatasource {
	return &MQTTDatasource{
		Client:        client,
		channelPrefix: fmt.Sprintf("ds/%d/", id),
		closeCh:       make(chan struct{}),
	}
}

// NewMQTTDatasource creates a new datasource instance.
func NewMQTTInstance(s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	settings, err := GetDatasourceSettings(s)
	if err != nil {
		return nil, err
	}

	client, err := mqtt.NewClient(*settings)
	if err != nil {
		return nil, err
	}

	return NewMQTTDatasource(client, s.ID), nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources
// when a new instance created. As soon as datasource settings change detected
// by SDK old datasource instance will be disposed and a new one will be created
// using NewSampleDatasource factory function.
func (ds *MQTTDatasource) Dispose() {
	close(ds.closeCh)
}

func (ds *MQTTDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := ds.Query(q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (ds *MQTTDatasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
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

func (ds *MQTTDatasource) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	ds.Client.Subscribe(req.Path)

	bytes, err := data.FrameToJSON(ToFrame(req.Path, []mqtt.Message{}), true, false) // only schema
	if err != nil {
		return nil, err
	}
	return &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusOK,
		Data:   bytes, // just the schema
	}, nil
}

func (ds *MQTTDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender backend.StreamPacketSender) error {
	defer ds.Client.Unsubscribe(req.Path)

	for {
		select {
		case <-ds.closeCh:
			log.DefaultLogger.Info("Datasource restart")
			return errors.New("datasource closed")
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

func (ds *MQTTDatasource) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied, // ?? Unsupported
	}, nil
}

type queryModel struct {
	Topic string `json:"queryText"`
}

func (m *MQTTDatasource) Query(query backend.DataQuery) backend.DataResponse {
	var qm queryModel

	response := backend.DataResponse{}
	response.Error = json.Unmarshal(query.JSON, &qm)

	if response.Error != nil {
		return response
	}

	// ensure the client is subscribed to the topic
	m.Client.Subscribe(qm.Topic)

	messages, ok := m.Client.Messages(qm.Topic)
	if !ok {
		return response
	}

	frame := ToFrame(qm.Topic, messages)

	if qm.Topic != "" {
		frame.SetMeta(&data.FrameMeta{
			Channel: m.channelPrefix + qm.Topic,
		})
	}

	response.Frames = append(response.Frames, frame)
	return response
}

func (m *MQTTDatasource) SendMessage(msg mqtt.StreamMessage, req *backend.RunStreamRequest, sender backend.StreamPacketSender) error {
	if !m.Client.IsSubscribed(req.Path) {
		return nil
	}

	message := mqtt.Message{
		Timestamp: time.Now(),
		Value:     msg.Value,
	}

	frame := ToFrame(msg.Topic, []mqtt.Message{message})
	bytes, err := data.FrameToJSON(frame, false, true)
	if err != nil {
		return err
	}

	packet := &backend.StreamPacket{
		Data: bytes,
	}

	log.DefaultLogger.Debug(fmt.Sprintf("Sending message to client for topic %s", msg.Topic))
	return sender.Send(packet)
}
