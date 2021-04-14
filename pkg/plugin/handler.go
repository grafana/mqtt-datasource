package plugin

import (
	"context"
	"errors"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/toddtreece/mqtt-datasource/pkg/mqtt"
)

func GetDatasourceOpts() datasource.ServeOpts {
	im := datasource.NewInstanceManager(NewServerInstance)
	ds := &Handler{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
		StreamHandler:      ds,
	}
}

func NewServerInstance(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return NewMQTTDatasource(settings)
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
		res := ds.Query(q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (h *Handler) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	ds, err := h.getDatasource(req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}

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

func (h *Handler) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	ds, err := h.getDatasource(req.PluginContext)
	if err != nil {
		return nil, err
	}
	ds.Client.Subscribe(req.Path)

	bytes, err := data.FrameToJSON(ToFrame([]mqtt.Message{}), true, false) // only schema
	if err != nil {
		return nil, err
	}
	return &backend.SubscribeStreamResponse{
		Status:       backend.SubscribeStreamStatusOK,
		UseRunStream: true,
		Data:         bytes, // just the schema

	}, nil
}

func (h *Handler) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender backend.StreamPacketSender) error {
	ds, err := h.getDatasource(req.PluginContext)
	if err != nil {
		return err
	}

	defer ds.Client.Unsubscribe(req.Path)

	for {
		select {
		case <-ctx.Done():
			backend.Logger.Info("stop streaming (context canceled)")
			return nil
		case message := <-ds.Client.Stream():
			go ds.SendMessage(message, req, sender)
		}
	}
}

func (h *Handler) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied, // ?? Unsupported
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
