package plugin

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/toddtreece/mqtt-datasource/pkg/mqtt"
)

type MQTTClient interface {
	IsConnected() bool
	IsSubscribed(topic string) bool
	Messages(topic string) ([]mqtt.Message, bool)
	Subscribe(topic string)
}

type MQTTDatasource struct {
	Client MQTTClient
}

func GetDatasourceSettings(s backend.DataSourceInstanceSettings) (*mqtt.Options, error) {
	settings := &mqtt.Options{}
	if err := json.Unmarshal(s.JSONData, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

type queryModel struct {
	Topic string `json:"queryText"`
}

func ToFrame(messages []mqtt.Message) *data.Frame {
	var timestamps []time.Time
	var values []float64

	for _, m := range messages {
		if value, err := strconv.ParseFloat(m.Value, 64); err == nil {
			timestamps = append(timestamps, m.Timestamp)
			values = append(values, value)
		}
	}

	frame := data.NewFrame("Messages")
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, timestamps),
	)

	frame.Fields = append(frame.Fields,
		data.NewField("values", nil, values),
	)

	return frame
}

func NewMQTTDatasource(s backend.DataSourceInstanceSettings) (*MQTTDatasource, error) {
	settings, err := GetDatasourceSettings(s)
	if err != nil {
		return nil, err
	}

	client, err := mqtt.NewClient(*settings)
	if err != nil {
		return nil, err
	}

	ds := MQTTDatasource{
		Client: client,
	}

	return &ds, nil
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

	frame := ToFrame(messages)

	response.Frames = append(response.Frames, frame)
	return response
}

func (m *MQTTDatasource) Dispose() {
	// Called before creating a a new instance to allow plugin authors
	// to cleanup.
}
