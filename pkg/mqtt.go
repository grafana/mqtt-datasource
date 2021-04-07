package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// newDatasource returns datasource.ServeOpts.
func newDatasource() datasource.ServeOpts {
	// creates a instance manager for your plugin. The function passed
	// into `NewInstanceManger` is called when the instance is created
	// for the first time or when a datasource configuration changed.
	im := datasource.NewInstanceManager(newDataSourceInstance)
	ds := &MqttDataSource{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

// MqttDataSource is an example datasource used to scaffold
// new datasource plugins with an backend.
type MqttDataSource struct {
	// The instance manager can help with lifecycle management
	// of datasource instances in plugins. It's not a requirements
	// but a best practice that we recommend that you follow.
	im instancemgmt.InstanceManager
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (td *MqttDataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData", "request", req)

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := td.query(ctx, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	Format string `json:"format"`
	Topic  string `json:"queryText"`
}

func (td *MqttDataSource) query(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	// Unmarshal the json into our queryModel
	var qm queryModel

	log.DefaultLogger.Debug(fmt.Sprintf("%+v", query))

	response := backend.DataResponse{}

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return response
	}

	// Log a warning if `Format` is empty.
	if qm.Format == "" {
		log.DefaultLogger.Warn("format is empty. defaulting to time series")
	}

	log.DefaultLogger.Debug(fmt.Sprintf("qm: %+v", qm))

	// create data frame response
	frame := data.NewFrame("response")

	// add the time dimension
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
	)

	// add values
	frame.Fields = append(frame.Fields,
		data.NewField("values", nil, []int64{10, 20}),
	)

	// add the frames to the response
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (td *MqttDataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	datasource, err := td.getDatasource(req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}

	client := *datasource.mqttClient
	if client == nil || !client.IsConnectionOpen() {
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

func (td *MqttDataSource) getDatasource(pluginCtx backend.PluginContext) (*instanceSettings, error) {
	s, err := td.im.Get(pluginCtx)
	if err != nil {
		return nil, err
	}

	datasource, ok := s.(*instanceSettings)
	if !ok {
		return nil, errors.New("invalid type assertion; is not *instanceSettings")
	}

	return datasource, nil
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.DefaultLogger.Debug(fmt.Sprintf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic()))
}

type instanceSettings struct {
	mqttClient *mqtt.Client
}

func newDataSourceInstance(s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	clientId := fmt.Sprintf("grafana_%d", rand.Int())
	settings, err := LoadSettings(s)
	if err != nil {
		return nil, fmt.Errorf("error reading settings: %s", err.Error())
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", settings.Host, settings.Port))
	opts.SetClientID(clientId)

	if settings.Username != "" {
		opts.SetUsername(settings.Username)
	}

	if settings.Password != "" {
		opts.SetPassword(settings.Password)
	}

	opts.SetDefaultPublishHandler(messagePubHandler)
	log.DefaultLogger.Info("MQTT Connecting")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting to MQTT broker: %s", err.Error())
	}
	return &instanceSettings{
		mqttClient: &client,
	}, nil
}

func (s *instanceSettings) Dispose() {
	settings := *s
	client := *settings.mqttClient
	log.DefaultLogger.Info("MQTT Disconnecting")
	client.Disconnect(250)
}
