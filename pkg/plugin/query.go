package plugin

import (
	"context"
	"encoding/json"
	"path"
	"time"

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
		return backend.ErrorResponseWithErrorSource(backend.DownstreamErrorf("failed to unmarshal query: %w", err))
	}

	if t.Path == "" {
		return backend.ErrorResponseWithErrorSource(backend.DownstreamErrorf("topic path is required"))
	}

	t.Interval = streamInterval(query.Interval)

	frame := data.NewFrame("")
	frame.SetMeta(&data.FrameMeta{
		Channel: path.Join(ds.channelPrefix, t.Key()),
	})

	response.Frames = append(response.Frames, frame)
	return response
}

// maxStreamInterval caps the push interval of a stream. Grafana derives the
// query interval from the dashboard range (range / maxDataPoints), so without a
// cap zooming out would create a new channel that pushes only every few minutes.
const maxStreamInterval = time.Second

func streamInterval(interval time.Duration) time.Duration {
	if interval <= 0 || interval > maxStreamInterval {
		return maxStreamInterval
	}
	return interval
}
