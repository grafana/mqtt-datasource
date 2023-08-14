package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/ISSACS-PSG/mqtt-datasource/pkg/mqtt"
)

func (ds *MQTTDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	chunks := strings.Split(req.Path, "/")
	if len(chunks) < 2 {
		return fmt.Errorf("invalid path: %s", req.Path)
	}

	interval, err := time.ParseDuration(chunks[0])
	if err != nil {
		return err
	}

	ds.Client.Subscribe(req.Path)
	defer ds.Client.Unsubscribe(req.Path)

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("stopped streaming (context canceled)", "path", req.Path)
			ds.Client.Unsubscribe(req.Path)
			ticker.Stop()
			return nil
		case <-ticker.C:
			topic, ok := ds.Client.GetTopic(req.Path)
			if !ok {
				log.DefaultLogger.Debug("topic not found", "path", req.Path)
				break
			}
			frame, err := topic.ToDataFrame()
			if err != nil {
				log.DefaultLogger.Error("failed to convert topic to data frame", "path", req.Path, "error", err)
				break
			}
			topic.Messages = []mqtt.Message{}
			if err := sender.SendFrame(frame, data.IncludeAll); err != nil {
				log.DefaultLogger.Error("failed to send data frame", "path", req.Path, "error", err)
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
