package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

func (ds *MQTTDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	defer ds.Client.Unsubscribe(req.Path)
	topic, ok := ds.Client.GetTopic(req.Path)
	if !ok {
		return fmt.Errorf("topic not found: %s", req.Path)
	}

	interval := topic.Interval
	if interval < (time.Millisecond * 100) {
		interval = time.Millisecond * 100
	}
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			backend.Logger.Debug("stopped streaming (context canceled)", "topic", req.Path)
			ds.Client.Unsubscribe(req.Path)
			ticker.Stop()
			return nil
		case <-ticker.C:
			topic, ok := ds.Client.GetTopic(req.Path)
			if !ok {
				break
			}
			frame := topic.ToDataFrame()
			topic.Messages = []mqtt.Message{}
			if err := sender.SendFrame(frame, data.IncludeDataOnly); err != nil {
				ticker.Stop()
				ds.Client.Unsubscribe(req.Path)
				return err
			}

		}
	}
}

func (ds *MQTTDatasource) SubscribeStream(_ context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	return &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusOK,
	}, nil
}

func (ds *MQTTDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}
