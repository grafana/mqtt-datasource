package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/toddtreece/mqtt-datasource/pkg/mqtt"
)

func GetDatasourceOpts() datasource.ServeOpts {
	im := datasource.NewInstanceManager(newDatasourceInstance)
	ds := &Handler{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

type Handler struct {
	im instancemgmt.InstanceManager
}

func (h *Handler) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	ds, err := h.getDatasource(req.PluginContext)

	if err != nil {
		return nil, err
	}

	for _, q := range req.Queries {
		res := h.query(ds.Client, q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	Topic string `json:"queryText"`
}

func (h *Handler) query(client MQTTClient, query backend.DataQuery) backend.DataResponse {
	var qm queryModel

	response := backend.DataResponse{}
	response.Error = json.Unmarshal(query.JSON, &qm)

	if response.Error != nil {
		return response
	}

	// ensure the client is subscribed to the topic
	client.Subscribe(qm.Topic)

	var timestamps []time.Time
	var values []float64

	messages, ok := client.GetMessages(qm.Topic)
	if !ok {
		return response
	}

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

	response.Frames = append(response.Frames, frame)
	return response
}

func (h *Handler) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	ds, err := h.getDatasource(req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}

	if ds.Client.IsConnected() {
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

func (h *Handler) getDatasource(pluginCtx backend.PluginContext) (*MQTTDatasource, error) {
	s, err := h.im.Get(pluginCtx)
	if err != nil {
		return nil, err
	}

	mqttDatasource, ok := s.(*MQTTDatasource)
	if !ok {
		return nil, errors.New("invalid type assertion; is not *MQTTDatasource")
	}

	return mqttDatasource, nil
}

func newDatasourceInstance(s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	options, err := LoadOptions(s)
	if err != nil {
		return nil, err
	}

	client, err := mqtt.NewClient(options)
	if err != nil {
		return nil, err
	}

	return &MQTTDatasource{
		Client: client,
	}, nil
}
