package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

func (ds *MQTTDatasource) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	// Extract the topic key from the channel path
	// Channel path format: "ds/{uid}/{topicKey}" where topicKey includes streaming key
	// We need to remove the channelPrefix ("ds/{uid}") to get the topic key
	topicKey := strings.TrimPrefix(req.Path, ds.channelPrefix+"/")

	chunks := strings.Split(topicKey, "/")
	if len(chunks) < 2 {
		return fmt.Errorf("invalid topic key: %s", topicKey)
	}

	interval, err := time.ParseDuration(chunks[0])
	if err != nil {
		return err
	}

	ds.Client.Subscribe(topicKey)
	defer ds.Client.Unsubscribe(topicKey)

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			log.DefaultLogger.Debug("stopped streaming (context canceled)", "path", req.Path, "topicKey", topicKey)
			ticker.Stop()
			return nil
		case <-ticker.C:
			topic, ok := ds.Client.GetTopic(topicKey)
			if !ok {
				log.DefaultLogger.Debug("topic not found", "path", req.Path, "topicKey", topicKey)
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

func (ds *MQTTDatasource) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	// Extract orgId from the streaming key embedded in the channel path
	// Channel: ds/{uid}/{interval}/{topic}/{datasourceUid}/{hash}/{orgId}
	pathParts := strings.Split(req.Path, "/")
	if len(pathParts) < 6 {
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusNotFound,
		}, fmt.Errorf("invalid channel path format")
	}

	orgId, err := strconv.ParseInt(pathParts[len(pathParts)-1], 10, 64)
	if err != nil {
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusNotFound,
		}, fmt.Errorf("unable to determine orgId from request")
	}

	pluginCfg := backend.PluginConfigFromContext(ctx)
	if orgId != pluginCfg.OrgID {
		return &backend.SubscribeStreamResponse{
			Status: backend.SubscribeStreamStatusPermissionDenied,
		}, fmt.Errorf("invalid orgId supplied in request")
	}

	return &backend.SubscribeStreamResponse{
		Status: backend.SubscribeStreamStatusOK,
	}, nil
}

func (ds *MQTTDatasource) PublishStream(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}
