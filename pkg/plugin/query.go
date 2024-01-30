package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

func (ds *MQTTDatasource) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := ds.query(q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (ds *MQTTDatasource) query(query backend.DataQuery) backend.DataResponse {
	var (
		t        mqtt.Topic
		response backend.DataResponse
	)

	if err := json.Unmarshal(query.JSON, &t); err != nil {
		response.Error = err
		return response
	}

	if t.Path == "" {
		response.Error = fmt.Errorf("topic path is required")
		return response
	}

	// Subscribe
	if len(t.Payload) == 0 {
		t.Interval = query.Interval

		frame := data.NewFrame("")
		frame.SetMeta(&data.FrameMeta{
			Channel: path.Join(ds.channelPrefix, t.Key()),
		})

		response.Frames = append(response.Frames, frame)
		return response
	}

	// Publish
	resp, err := ds.Client.Publish(t.Path, t.Payload, t.ResponsePath)

	field := data.NewField("Body", data.Labels{}, []json.RawMessage{resp})
	response.Frames = append(response.Frames, data.NewFrame("Response", field))
	response.Error = err
	return response
}
